<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../api'
import { formatDate, formatRelative } from '../../i18n/format'
import {
  toast, XButton, XConfirm, XEmpty, XField, XFieldGrid, XFieldNote, XInput, XKeyValue, XSegmented, XSettingBlock, XSettingsSection,
  XSkeletonText, XStatusText,
} from '../../kit'
import { cause, useSettingsForm } from './form'
import { addressAfterSite } from './restart'
import { loadSite, type SiteInfo, type SiteSettings } from './site'

/** The website visitors see without the secret path: where it runs, its domain and its certificate. */
const { t } = useI18n()
const f = useSettingsForm()

const open = reactive({ mode: true, cert: true })

const site = ref<SiteInfo | null>(null)
const failed = ref(false)
onMounted(async () => {
  try {
    site.value = await loadSite()
    Object.assign(form, pick(site.value))
    certSource.value = site.value.certFile ? 'files' : 'auto'
  } catch {
    failed.value = true
  }
})

// --- Mode and domain ------------------------------------------------------------

const pick = (s: SiteSettings): SiteSettings => ({ domain: s.domain, certFile: s.certFile, keyFile: s.keyFile })
const form = reactive<SiteSettings>({ domain: '', certFile: '', keyFile: '' })
const certSource = ref<'auto' | 'files'>('auto')

// The mode is set on the server only: moving the panel from here could leave the admin outside it.
const reality = computed(() => site.value?.mode === 'reality')
/** The port the website and the panel answer on: 443 behind REALITY, the panel's own otherwise. */
const sitePort = computed(() => String(reality.value ? 443 : (site.value?.port ?? '')))
const sources = computed(() => [
  { value: 'auto' as const, label: t('settings.siteCertAuto') },
  { value: 'files' as const, label: t('settings.siteCertFiles') },
])

/** What is sent: behind REALITY the files stay as they are; obtaining automatically clears them. */
const settings = computed<SiteSettings>(() => {
  if (reality.value) return { ...form, certFile: site.value?.certFile ?? '', keyFile: site.value?.keyFile ?? '' }
  return certSource.value === 'files' ? { ...form } : { ...form, certFile: '', keyFile: '' }
})
const changed = computed(() => {
  if (!site.value) return false
  const was = pick(site.value)
  const now = settings.value
  return (Object.keys(was) as (keyof SiteSettings)[]).some((k) => now[k] !== was[k])
})

const isIp = (s: string) => /^[\d.]+$/.test(s) || s.includes(':')
const errors = computed(() => ({
  domain: reality.value && (!form.domain || isIp(form.domain)) ? t('settings.siteDomainNeeded') : undefined,
  certFile: settings.value.certFile || settings.value.keyFile ? path(settings.value.certFile) : undefined,
  keyFile: settings.value.certFile || settings.value.keyFile ? path(settings.value.keyFile) : undefined,
}))
const path = (p: string) => (p.startsWith('/') ? undefined : t('settings.errPathStart'))
const valid = computed(() => !Object.values(errors.value).some(Boolean))

// Applying moves the panel: it restarts and opens where the settings put it.
const confirming = ref(false)
const applying = ref(false)
const failure = ref('')
const address = computed(() => addressAfterSite(settings.value, site.value?.mode ?? '', site.value?.port ?? 0, f.saved.webBasePath))
async function apply() {
  applying.value = true
  failure.value = ''
  try {
    await api.post('panel/setting/siteApply', settings.value)
  } catch (e) {
    failure.value = cause(e)
    return
  } finally {
    applying.value = false
    confirming.value = false
  }
  // The page saves these too: it must not put the old ones back.
  const files = { webCertFile: settings.value.certFile, webKeyFile: settings.value.keyFile }
  Object.assign(f.saved, files)
  Object.assign(f.draft, files)
  toast('ok', t('settings.siteApplied'))
  f.restart(address.value)
}

// --- Certificate ----------------------------------------------------------------

// Errors are long: they get lines of their own, as plain text that wraps.
const when = (at: string) => formatDate(Date.parse(at))
const certRows = computed(() => {
  const s = site.value
  if (!s?.cert) return []
  const c = s.cert
  const rows: { label: string; value?: string; key?: string }[] = [
    { label: t('settings.certName'), value: c.domain },
    { label: t('settings.certIssuer'), value: c.issuer || t('settings.certStopgap') },
  ]
  if (c.issuer) {
    const until = Date.parse(c.notAfter)
    rows.push({ label: t('settings.certUntil'), value: `${formatDate(until)} · ${formatRelative(until)}` })
  }
  if (!c.auto) return rows
  rows.push({ label: t('settings.certCheck'), key: 'check' })
  if (s.check && !s.check.ok) rows.push({ label: t('settings.certCheckError'), value: s.check.error })
  if (c.error) rows.push({ label: t('settings.certError'), value: `${when(c.errorAt)} · ${c.error}` })
  return rows
})

const renewing = ref(false)
async function renew() {
  renewing.value = true
  try {
    site.value = await api.post<SiteInfo>('panel/setting/siteRenew')
    toast('ok', t('settings.certRenewed'))
  } catch (e) {
    toast('err', t('settings.certRenewFailed', { error: cause(e) }))
  } finally {
    renewing.value = false
  }
}
</script>

<template>
  <div class="settings-tab">
    <XSettingsSection v-model:open="open.mode" :title="t('settings.secSite')" :count="reality ? 2 : 3">
      <XSettingBlock>
        <XSkeletonText v-if="!site && !failed" width="60%" />
        <XEmpty v-else-if="failed" tone="err" icon="alert" :title="t('settings.siteLoadFailed')" />
        <XFieldGrid v-else top>
          <XField v-slot="a" :label="t('settings.siteMode')" class="x-wide">
            <XInput v-bind="a" :model-value="sitePort" mono readonly />
          </XField>
          <XFieldNote tone="info" class="x-wide">
            <i18n-t scope="global" keypath="settings.siteModeWhere"><template #menu><code>space-ui → 19</code></template></i18n-t>
          </XFieldNote>
          <XField v-slot="a" :label="t('settings.siteDomain')" :hint="reality ? t('settings.siteDomainReality') : t('settings.siteDomainPort')" :error="errors.domain">
            <XInput v-bind="a" v-model.trim="form.domain" mono placeholder="example.com" :invalid="!!errors.domain" />
          </XField>
          <template v-if="!reality">
            <XField v-slot="a" :label="t('settings.siteCert')" class="x-wide">
              <XSegmented v-bind="a" v-model="certSource" :options="sources" block />
            </XField>
            <template v-if="certSource === 'files'">
              <XField v-slot="a" :label="t('settings.siteCertFile')" :error="errors.certFile">
                <XInput v-bind="a" v-model.trim="form.certFile" mono placeholder="/root/cert/fullchain.pem" :invalid="!!errors.certFile" />
              </XField>
              <XField v-slot="a" :label="t('settings.siteKeyFile')" :error="errors.keyFile">
                <XInput v-bind="a" v-model.trim="form.keyFile" mono placeholder="/root/cert/privkey.pem" :invalid="!!errors.keyFile" />
              </XField>
            </template>
          </template>
        </XFieldGrid>
        <template v-if="site" #note>
          <XFieldNote v-if="failure" tone="err">{{ failure }}</XFieldNote>
          <span>{{ t('settings.siteApplyHint') }}</span>
        </template>
        <template v-if="site" #actions>
          <XButton variant="primary" icon="check" :disabled="!changed || !valid" @click="confirming = true">{{ t('settings.siteApply') }}</XButton>
        </template>
      </XSettingBlock>
    </XSettingsSection>

    <XSettingsSection v-if="site" v-model:open="open.cert" :title="t('settings.secSiteCert')">
      <XSettingBlock>
        <XKeyValue v-if="site.cert" :rows="certRows">
          <template #value-check>
            <XStatusText v-if="site.check" :tone="site.check.ok ? 'ok' : 'err'">
              {{ t(site.check.ok ? 'settings.certCheckOk' : 'settings.certCheckFail', { time: when(site.check.at) }) }}
            </XStatusText>
            <template v-else>{{ t('settings.certCheckNone') }}</template>
          </template>
        </XKeyValue>
        <XFieldNote v-else>{{ t('settings.certNone') }}</XFieldNote>
        <template v-if="site.cert?.auto" #note>
          <span>{{ t('settings.certAutoHint') }}</span>
        </template>
        <template v-if="site.cert?.auto" #actions>
          <XButton variant="secondary" icon="refresh" :loading="renewing" @click="renew">{{ t(site.cert.issuer ? 'settings.certRenew' : 'settings.certObtain') }}</XButton>
        </template>
      </XSettingBlock>
    </XSettingsSection>

    <XConfirm v-model:open="confirming" icon="refresh" :title="t('settings.siteApplyTitle')" :cta="t('settings.siteApply')" :busy="applying" @confirm="apply">
      <i18n-t scope="global" keypath="settings.siteApplyText"><template #address><code>{{ address.panel }}</code></template></i18n-t>
    </XConfirm>
  </div>
</template>

<style scoped>
.settings-tab {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
