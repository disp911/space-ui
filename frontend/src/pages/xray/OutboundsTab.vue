<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../api'
import { formatBytes } from '../../i18n/format'
import {
  toast, XButton, XButtonGroup, XCard, XChip, XEmpty, XGridTable, XPopConfirm, XTestResult, type GridColumn, type MenuItem,
} from '../../kit'
import { cause } from '../settings/form'
import { useXrayForm } from './form'
import NordModal from './NordModal.vue'
import OutboundModal from './OutboundModal.vue'
import WarpModal from './WarpModal.vue'
import { outboundAddresses, outboundChips, untestable, type Outbound, type OutboundTest, type OutboundTraffic, type Rule } from './model'

/**
 * The outbounds of the template in their order, the first being where
 * traffic goes by default: where each connects, the traffic it carried and a
 * test through it. Traffic lives on the server and is reset there at once.
 */
const { t } = useI18n()
const f = useXrayForm()

const outbounds = computed<Outbound[]>(() => f.draft.config.outbounds ?? [])

const columns = computed<GridColumn[]>(() => [
  { key: 'tag', label: t('xray.outTag'), size: '1.07fr', phone: 'head' },
  { key: 'address', label: t('xray.outAddress') },
  { key: 'traffic', label: t('xray.outTraffic'), size: '120px', phone: 'half' },
  { key: 'result', label: t('xray.outResult'), size: '176px', phone: 'half' },
  { key: 'test', label: t('xray.outTest'), size: '44px', phone: 'head' },
])

// --- Traffic --------------------------------------------------------------------

const traffic = ref<OutboundTraffic[]>([])
const refreshing = ref(false)

async function loadTraffic() {
  refreshing.value = true
  try {
    traffic.value = (await api.get<OutboundTraffic[] | null>('panel/xray/getOutboundsTraffic')) ?? []
  } catch (e) {
    toast('err', cause(e))
  } finally {
    refreshing.value = false
  }
}
onMounted(loadTraffic)

const usage = (o: Outbound) => traffic.value.find((x) => x.tag === o.tag) ?? { up: 0, down: 0 }

/** One outbound's traffic, or every one's with the server's own word for all. */
async function resetTraffic(tag: string) {
  const form = new FormData()
  form.set('tag', tag)
  try {
    await api.post('panel/xray/resetOutboundsTraffic', form)
  } catch (e) {
    toast('err', cause(e))
  }
  await loadTraffic()
}

// --- Tests ----------------------------------------------------------------------

type TestState = { state: 'testing' } | { state: 'ok'; delay: number; status?: number } | { state: 'fail'; error: string }

/** By tag, so a result stays with its outbound when rows move or the template is saved. */
const tests = reactive(new Map<string, TestState>())

async function test(o: Outbound) {
  const tag = String(o.tag ?? '')
  tests.set(tag, { state: 'testing' })
  const form = new FormData()
  form.set('outbound', JSON.stringify(o))
  // Others it may dial through (sockopt.dialerProxy) come along.
  form.set('allOutbounds', JSON.stringify(outbounds.value))
  try {
    const r = await api.post<OutboundTest>('panel/xray/testOutbound', form)
    tests.set(tag, r.success ? { state: 'ok', delay: r.delay, status: r.statusCode } : { state: 'fail', error: r.error ?? '' })
    if (r.success) toast('ok', t('xray.outTestOk', { delay: r.delay, status: r.statusCode ?? '' }))
    else toast('err', t('xray.outTestFailed', { error: r.error ?? '' }))
  } catch (e) {
    tests.set(tag, { state: 'fail', error: cause(e) })
    toast('err', t('xray.outTestError', { error: cause(e) }))
  }
}

const result = (o: Outbound): TestState | { state: 'idle' } => tests.get(String(o.tag ?? '')) ?? { state: 'idle' }

const warpOpen = ref(false)
const nordOpen = ref(false)

// The outbound window: a new outbound, or the one at editing.
const formOpen = ref(false)
const editing = ref(-1)
const otherTags = computed(() => outbounds.value.filter((_, i) => i !== editing.value).map((o) => String(o.tag ?? '')).filter(Boolean))

function create() {
  editing.value = -1
  formOpen.value = true
}

function saveOutbound(o: Outbound) {
  const list = (f.draft.config.outbounds ??= []) as Outbound[]
  if (editing.value >= 0) list[editing.value] = o
  else list.push(o)
}

// --- Rows -----------------------------------------------------------------------

const menu = (_o: Outbound, i: number): MenuItem[] => [
  ...(i > 0 ? [{ key: 'first', label: t('xray.outFirst'), icon: 'top' as const }] : []),
  { key: 'edit', label: t('xray.edit'), icon: 'edit' },
  { key: 'traffic', label: t('xray.outResetTraffic'), icon: 'reset' },
  { key: 'delete', label: t('xray.delete'), icon: 'delete', danger: true },
]

function select(key: string, o: Outbound, i: number) {
  const list = f.draft.config.outbounds as Outbound[]
  if (key === 'first') list.unshift(list.splice(i, 1)[0]!)
  else if (key === 'delete') {
    list.splice(i, 1)
    // Rules that sent traffic to it lose their target, as when a balancer goes.
    for (const r of (f.draft.config.routing?.rules ?? []) as Rule[]) if (o.tag && r.outboundTag === o.tag) delete r.outboundTag
  } else if (key === 'traffic') resetTraffic(String(o.tag ?? ''))
  else if (key === 'edit') {
    editing.value = i
    formOpen.value = true
  }
}
</script>

<template>
  <div class="xray-tab">
    <div class="outbounds__tools">
      <XButton variant="primary" icon="plus" phone-icon-only @click="create">{{ t('xray.outAdd') }}</XButton>
      <XButton @click="warpOpen = true">WARP</XButton>
      <XButton @click="nordOpen = true">NordVPN</XButton>
      <XButtonGroup :label="t('xray.outTrafficTools')" class="outbounds__end">
        <XButton icon="refresh" :label="t('xray.outRefresh')" :loading="refreshing" @click="loadTraffic" />
        <XPopConfirm v-slot="{ props }" :text="t('xray.outResetAllText')" :cta="t('xray.outReset')" :run="() => resetTraffic('-alltags-')">
          <XButton v-bind="props" icon="reset" :label="t('xray.outResetTraffic')" />
        </XPopConfirm>
      </XButtonGroup>
    </div>

    <XGridTable
      v-if="outbounds.length"
      :columns="columns"
      :rows="outbounds"
      :row-key="(_, i) => i"
      :menu="menu"
      :row-label="(i) => t('xray.outN', { n: i + 1 })"
      numbered
      center
      @select="select"
    >
      <template #cell-tag="{ row }">
        <div class="outbounds__tag">
          <span class="outbounds__name" :title="row.tag">{{ row.tag }}</span>
          <span class="outbounds__chips">
            <XChip mono>{{ outboundChips(row).protocol }}</XChip>
            <XChip v-if="outboundChips(row).network" mono outline>{{ outboundChips(row).network }}</XChip>
            <XChip v-if="outboundChips(row).security" tone="ok" mono outline>{{ outboundChips(row).security }}</XChip>
          </span>
        </div>
      </template>
      <template #cell-address="{ row }">
        <div class="outbounds__addresses">
          <span v-for="a in outboundAddresses(row)" :key="a" class="outbounds__address">{{ a }}</span>
          <span v-if="!outboundAddresses(row).length" class="outbounds__none">—</span>
        </div>
      </template>
      <template #cell-traffic="{ row }">
        <div class="outbounds__traffic">
          <span :title="t('xray.outUp')">↑ {{ formatBytes(usage(row).up) }}</span>
          <span :title="t('xray.outDown')">↓ {{ formatBytes(usage(row).down) }}</span>
        </div>
      </template>
      <template #cell-result="{ row }">
        <XTestResult v-bind="result(row)" />
      </template>
      <template #cell-test="{ row }">
        <XButton
          icon="bolt"
          :label="t('xray.outTest')"
          :loading="result(row).state === 'testing'"
          :disabled="untestable(row)"
          @click="test(row)"
        />
      </template>
    </XGridTable>
    <XCard v-else>
      <XEmpty icon="arrowRight" tab />
    </XCard>

    <WarpModal v-model:open="warpOpen" />
    <NordModal v-model:open="nordOpen" />
    <OutboundModal v-model:open="formOpen" :outbound="editing >= 0 ? outbounds[editing] ?? null : null" :index="editing" :tags="otherTags" @save="saveOutbound" />
  </div>
</template>

<style scoped>
.outbounds__tools {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.outbounds__end {
  margin-left: auto;
}

.outbounds__tag {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.outbounds__name {
  overflow: hidden;
  color: var(--fg);
  font: 500 var(--fs-13) / var(--lh-tight) var(--font-mono);
  white-space: nowrap;
  text-overflow: ellipsis;
  /* Room for what hangs below the line (g, y) or above it (Ё), which the ellipsis' clip cut; the box keeps its size. */
  padding-block: .15em;
  margin-block: -.15em;
}

.outbounds__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.outbounds__addresses {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.outbounds__address {
  color: var(--fg2);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-code);
  word-break: break-all;
}

.outbounds__none {
  color: var(--faint);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-mono);
}

.outbounds__traffic {
  display: flex;
  flex-direction: column;
  gap: 4px;
  color: var(--fg2);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-mono);
  white-space: nowrap;
}

/* Phones: the two directions on one line. */
@media (max-width: 767px) {
  .outbounds__traffic {
    flex-direction: row;
    gap: 8px;
  }
}
</style>
