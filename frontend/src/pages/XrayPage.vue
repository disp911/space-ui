<script setup lang="ts">
import { computed, nextTick, onMounted, provide, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'
import {
  toast, XActionBar, XButton, XCard, XEmpty, XNotice, XProcessError, XSettingsSkeleton, XStatusText, XTabs, type TabItem,
} from '../kit'
import AppPage from '../layout/AppPage.vue'
import { cause } from './settings/form'
import { formKey } from './xray/form'
import {
  changes, defaultTestUrl, fieldValue, insecure, ruleTarget, tabs, type FieldKey, type Rule, type TabKey, type XrayConfig, type XrayReply, type XraySetting,
} from './xray/model'
import GeneralTab from './xray/GeneralTab.vue'
import AdvancedTab from './xray/AdvancedTab.vue'
import RoutingTab from './xray/RoutingTab.vue'
import OutboundsTab from './xray/OutboundsTab.vue'
import BalancersTab from './xray/BalancersTab.vue'
import DnsTab from './xray/DnsTab.vue'

const { t } = useI18n()

// --- Loading ----------------------------------------------------------------

/** As the server has it, and as edited on the page. */
const saved = ref<XraySetting | null>(null)
const draft = ref<XraySetting | null>(null)
/** Tags of the inbounds and of the clients' reverse proxies, for the routing rules. */
const inboundTags = ref<string[]>([])
const reverseTags = ref<string[]>([])
const loading = ref(true)
const failed = ref(false)

// Through JSON: a reactive proxy cannot be cloned structurally.
const copy = (s: XraySetting): XraySetting => ({ config: JSON.parse(JSON.stringify(s.config)), testUrl: s.testUrl })

/** quiet keeps the page as it is, e.g. after a save. */
async function load(quiet = false) {
  if (!quiet) {
    loading.value = true
    failed.value = false
  }
  try {
    const reply = JSON.parse(await api.post<string>('panel/xray/')) as XrayReply
    saved.value = { config: reply.xraySetting, testUrl: reply.outboundTestUrl || defaultTestUrl }
    draft.value = copy(saved.value)
    inboundTags.value = reply.inboundTags ?? []
    reverseTags.value = reply.clientReverseTags ?? []
  } catch {
    if (!quiet) failed.value = true
  } finally {
    loading.value = false
  }
}

/** What Xray printed when it last failed to start; empty while it runs. */
const result = ref('')
const resultOpen = ref(false)
async function loadResult() {
  try {
    result.value = await api.get<string>('panel/xray/getXrayResult')
  } catch {
    // the button stays as it was
  }
}

onMounted(() => {
  load()
  loadResult()
})

const ready = computed(() => !loading.value && !failed.value && !!draft.value)

// --- Changes ----------------------------------------------------------------

const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b)
const isDirty = (k: FieldKey) => !!draft.value && !!saved.value && !same(fieldValue(draft.value, k), fieldValue(saved.value, k))
const tabChanges = (tab: TabKey) => (draft.value && saved.value ? changes(tab, draft.value, saved.value) : 0)
const dirty = computed(() => tabs.reduce((n, tab) => n + tabChanges(tab.key), 0))

provide(formKey, {
  get draft() {
    return draft.value as XraySetting
  },
  get saved() {
    return saved.value as XraySetting
  },
  get inboundTags() {
    return inboundTags.value
  },
  get reverseTags() {
    return reverseTags.value
  },
  isDirty,
  count: (keys) => keys.filter(isDirty).length,
  async resetToDefault() {
    try {
      const config = await api.get<XrayConfig>('panel/xray/getDefaultJsonConfig')
      if (draft.value) draft.value.config = config
    } catch (e) {
      toast('err', cause(e))
    }
  },
})

const status = computed(() => (ready.value && dirty.value ? t('settings.unsaved', { n: dirty.value }) : null))

// --- Tabs -------------------------------------------------------------------

const tab = ref<TabKey>('general')
const tabItems = computed<TabItem<TabKey>[]>(() =>
  tabs.map((item) => ({ key: item.key, label: t(item.label), dot: tabChanges(item.key) ? 'mod' : undefined })),
)
// A tab starts from its top.
watch(tab, () => nextTick(() => window.scrollTo({ top: 0 })))

// --- Security banner --------------------------------------------------------

const bannerClosed = ref(false)
const showBanner = computed(() => ready.value && !bannerClosed.value && insecure())

// --- Save -------------------------------------------------------------------

const saving = ref(false)
/** The first rule with nowhere to send its traffic, counted from 1: Xray will not start with it. */
const noTargetRule = computed(() => {
  const rules = (draft.value?.config.routing?.rules ?? []) as Rule[]
  return rules.findIndex((r) => !ruleTarget(r)) + 1
})
const canSave = computed(() => ready.value && dirty.value > 0 && !noTargetRule.value && !saving.value && !restarting.value)
const saveTitle = computed(() => {
  if (canSave.value || saving.value) return undefined
  return noTargetRule.value ? t('xray.ruleNoTargetHelp', { n: noTargetRule.value }) : t('settings.nothingToSave')
})

async function save() {
  if (!canSave.value || !draft.value) return
  saving.value = true
  const form = new FormData()
  form.set('xraySetting', JSON.stringify(draft.value.config, null, 2))
  form.set('outboundTestUrl', draft.value.testUrl || defaultTestUrl)
  try {
    await api.post('panel/xray/update', form)
    saved.value = copy(draft.value)
    toast('ok', t('settings.saved'))
    // The server may tidy the template up.
    await load(true)
  } catch (e) {
    toast('err', t('settings.saveFailed', { error: cause(e) }))
  } finally {
    saving.value = false
  }
}

// --- Restart ----------------------------------------------------------------

const restarting = ref(false)
const canRestart = computed(() => ready.value && !dirty.value && !saving.value)

async function restart() {
  if (!canRestart.value || restarting.value) return
  restarting.value = true
  resultOpen.value = false
  try {
    await api.post('panel/api/server/restartXrayService')
    // Xray reports a failed start a moment after the restart returns.
    await new Promise((done) => setTimeout(done, 500))
    await loadResult()
    if (result.value) resultOpen.value = true
  } catch (e) {
    toast('err', t('xray.restartFailed', { error: cause(e) }))
  } finally {
    restarting.value = false
  }
}
</script>

<template>
  <AppPage :title="t('nav.xray')" fixed :busy="restarting">
    <template v-if="showBanner" #banner>
      <XNotice tone="err" :title="t('settings.securityTitle')" closable @close="bannerClosed = true">
        {{ t('xray.securityText') }}
      </XNotice>
    </template>
    <template #subtitle>{{ t('xray.subtitle') }}</template>
    <template #actions>
      <XStatusText v-if="ready && noTargetRule" tone="err">{{ t('xray.ruleNoTargetN', { n: noTargetRule }) }}</XStatusText>
      <XStatusText v-else-if="status" tone="mod">{{ status }}</XStatusText>
      <XProcessError v-if="result" v-model:open="resultOpen" :title="t('xray.startError')" :text="result" />
      <XButton variant="primary" icon="check" :disabled="!canSave && !saving" :loading="saving" :title="saveTitle" @click="save">
        {{ saving ? t('settings.saving') : t('settings.save') }}
      </XButton>
      <XButton variant="primary" icon="refresh" :disabled="!canRestart" :loading="restarting" :title="canRestart ? undefined : t('settings.saveFirst')" @click="restart">
        {{ t('xray.restart') }}
      </XButton>
    </template>
    <template #tabs>
      <XTabs v-model="tab" :items="tabItems" :label="t('xray.sections')" :disabled="loading" />
    </template>

    <XSettingsSkeleton v-if="loading" :rows="[3, 2]" />
    <XCard v-else-if="failed">
      <XEmpty tone="err" icon="alert" :title="t('settings.loadFailed')" />
    </XCard>
    <div v-else :key="tab" class="x-tab-panel">
      <GeneralTab v-if="tab === 'general'" />
      <RoutingTab v-else-if="tab === 'routing'" />
      <OutboundsTab v-else-if="tab === 'outbounds'" />
      <BalancersTab v-else-if="tab === 'balancer'" />
      <DnsTab v-else-if="tab === 'dns'" />
      <AdvancedTab v-else-if="tab === 'advanced'" />
      <XNotice v-else tone="info">{{ t('common.screenDraft') }}</XNotice>
    </div>

    <template #bar>
      <XActionBar>
        <template v-if="(ready && noTargetRule) || status" #state>
          <XStatusText v-if="ready && noTargetRule" tone="err">{{ t('xray.ruleNoTargetN', { n: noTargetRule }) }}</XStatusText>
          <XStatusText v-else tone="mod">{{ status }}</XStatusText>
        </template>
        <XButton variant="primary" :disabled="!canSave && !saving" :loading="saving" :title="saveTitle" @click="save">
          {{ saving ? t('settings.saving') : t('settings.save') }}
        </XButton>
        <XButton variant="primary" icon="refresh" :disabled="!canRestart" :pending="ready && !!dirty" :loading="restarting" @click="restart">
          {{ t('settings.restartShort') }}
        </XButton>
        <XProcessError v-if="result" v-model:open="resultOpen" :title="t('xray.startError')" :text="result" />
      </XActionBar>
    </template>
  </AppPage>

</template>
