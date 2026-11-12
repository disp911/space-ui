<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'
import {
  toast, XButton, XCard, XChip, XConfirm, XEmpty, XExpandTable, XInlineButton, XKeyValue, XMark, XMenu, XNotice, XPager, XPopover, XRowToggle, XSpinner, XSummaryStat, XSwitch,
  XTooltip,
  type ExpandColumn, type MenuItem,
} from '../kit'
import { formatBytes, formatDate, formatRelative } from '../i18n/format'
import AppPage from '../layout/AppPage.vue'
import { useInbounds } from '../stores/inbounds'
import { insecure } from './xray/model'
import { cause } from './settings/form'
import { ask, asked, type Ask } from './inbounds/ask'
import ClientCounts from './inbounds/ClientCounts.vue'
import InboundClients from './inbounds/InboundClients.vue'
import InboundsTools from './inbounds/InboundsTools.vue'
import InboundModal from './inbounds/InboundModal.vue'
import ClientModal from './inbounds/ClientModal.vue'
import BulkClientModal from './inbounds/BulkClientModal.vue'
import CopyClientsModal from './inbounds/CopyClientsModal.vue'
import InfoModal from './inbounds/InfoModal.vue'
import QrModal from './inbounds/QrModal.vue'
import TextModal from './inbounds/TextModal.vue'
import ImportModal from './inbounds/ImportModal.vue'
import { cloneOf, toForm } from './inbounds/inbound'
import { inboundLinks, linkForm, subLinks } from './inbounds/links'
import {
  clientsOf, deepSearch, expiryState, filterEmails, hasClients, pageSizes, protocolChips, trafficState, type Client, type ClientCounts as Counts, type DbInbound, type Filter, type State,
} from './inbounds/model'

const { t } = useI18n()
const store = useInbounds()
const stop = store.watch()
onBeforeUnmount(stop)

const thresholds = computed(() => store.defaults ?? { expireDiff: 0, trafficDiff: 0 })

// --- Summary ---------------------------------------------------------------------

/** The table shows its phone columns, and the summary its phone layout with it. */
const compact = ref(false)

const totals = computed(() => {
  let up = 0
  let down = 0
  let allTime = 0
  for (const ib of store.list) {
    up += ib.up
    down += ib.down
    allTime += ib.allTime || ib.up + ib.down
  }
  return { up, down, allTime }
})

/** The clients of every inbound together, as the summary counts them. */
const allCounts = computed<Counts>(() => {
  const sum: Counts = { clients: 0, active: [], deactive: [], depleted: [], expiring: [], online: [], comments: new Map() }
  for (const c of store.counts.values()) {
    sum.clients += c.clients
    sum.active.push(...c.active)
    sum.deactive.push(...c.deactive)
    sum.depleted.push(...c.depleted)
    sum.expiring.push(...c.expiring)
    sum.online.push(...c.online)
    c.comments.forEach((v, k) => sum.comments.set(k, v))
  }
  return sum
})

// --- Search and filter ---------------------------------------------------------------

const mode = ref<'search' | 'filter'>('search')
const search = ref('')
const filter = ref<Filter>('none')

// Typed text settles before the list is searched, as on the old panel.
const query = ref('')
let typing = 0
watch(search, (q) => {
  clearTimeout(typing)
  typing = window.setTimeout(() => (query.value = q.trim()), 500)
})
onBeforeUnmount(() => clearTimeout(typing))

const shown = computed<DbInbound[]>(() => {
  const q = query.value
  if (mode.value === 'search' && q) return store.list.filter((ib) => deepSearch(ib, q))
  if (mode.value === 'filter' && filter.value !== 'none') {
    const f = filter.value
    return store.list.filter((ib) => {
      const c = store.counts.get(ib.id)
      return !!c && filterEmails(c, f).length > 0
    })
  }
  return store.list
})

// --- Pages -------------------------------------------------------------------------

const page = ref(1)
const size = ref(25)
watch(
  () => store.defaults?.pageSize,
  (s) => s && (size.value = s),
  { immediate: true },
)
watch([query, filter, mode], () => (page.value = 1))

const pages = computed(() => Math.max(1, Math.ceil(shown.value.length / size.value)))
watch(pages, (n) => page.value > n && (page.value = n))
const rows = computed(() =>
  shown.value.length > size.value ? shown.value.slice((page.value - 1) * size.value, page.value * size.value) : shown.value,
)
const sizes = computed(() => pageSizes(store.defaults?.pageSize || 25, shown.value.length))

// --- Rows --------------------------------------------------------------------------

const columns = computed<ExpandColumn[]>(() => [
  { key: 'id', label: 'ID', size: '24px', align: 'center' },
  { key: 'menu', label: t('inbounds.colMenu'), size: '28px', muted: 'none', quiet: true },
  { key: 'enable', label: t('inbounds.colEnable'), size: '52px', muted: 'none', quiet: true },
  { key: 'remark', label: t('inbounds.colRemark'), size: 'minmax(0, 1.6fr)', muted: 'soft' },
  { key: 'port', label: t('inbounds.colPort'), size: '56px' },
  { key: 'protocol', label: t('inbounds.colProtocol'), size: 'minmax(0, 1.15fr)' },
  { key: 'clients', label: t('inbounds.clients'), size: 'minmax(0, 1.3fr)' },
  { key: 'traffic', label: t('inbounds.colTraffic'), size: '150px' },
  { key: 'allTime', label: t('inbounds.colAllTime'), size: '86px' },
  { key: 'expiry', label: t('inbounds.colExpiry'), size: '128px' },
])
const phoneColumns = computed<ExpandColumn[]>(() => [
  { key: 'menu', label: t('inbounds.colMenu'), size: '44px', muted: 'none', quiet: true },
  { key: 'remark', label: t('inbounds.colRemark'), muted: 'none' },
  { key: 'info', label: t('inbounds.colInfo'), size: 'var(--btn-sq)', align: 'end', muted: 'none', quiet: true },
])

const toneOf = (s: State) => (s === 'unl' ? 'neutral' : s)
const used = (ib: DbInbound) => ib.up + ib.down
const traffic = (ib: DbInbound) => trafficState(used(ib), ib.total, thresholds.value)
const expiry = (ib: DbInbound) => expiryState(ib.expiryTime, thresholds.value)
const security = (s: string) => (s === 'tls' ? 'TLS' : s === 'reality' ? 'Reality' : s)
const countsOf = (ib: DbInbound) => store.counts.get(ib.id)

/** The menu of an inbound, as the mockup orders it; on phones it ends with a switch. */
function menu(ib: DbInbound, phone = false): MenuItem[] {
  const items: MenuItem[] = [{ key: 'edit', label: t('inbounds.edit'), icon: 'edit' }]
  const settings = JSON.parse(ib.settings || '{}')
  if ((ib.protocol === 'shadowsocks' && !hasClients(ib, settings)) || ib.protocol === 'wireguard') items.push({ key: 'qr', label: t('inbounds.qr'), icon: 'qr' })
  if (hasClients(ib, settings)) {
    items.push(
      { key: 'addClient', label: t('inbounds.addClient'), icon: 'userPlus', divided: true },
      { key: 'addBulk', label: t('inbounds.addBulk'), icon: 'users' },
      { key: 'copyClients', label: t('inbounds.copyClients'), icon: 'copy' },
      { key: 'resetClients', label: t('inbounds.resetClients'), icon: 'reset' },
      { key: 'exportLinks', label: t('inbounds.exportLinks'), icon: 'link' },
      ...(store.defaults?.subEnable ? [{ key: 'exportSubs', label: t('inbounds.exportSubs'), icon: 'link' as const }] : []),
      { key: 'delDepleted', label: t('inbounds.delDepleted'), icon: 'userX', danger: true },
    )
  } else items.push({ key: 'info', label: t('inbounds.info'), icon: 'info' })
  items.push(
    { key: 'exportInbound', label: t('inbounds.exportInbound'), icon: 'upload', divided: true },
    { key: 'resetTraffic', label: t('inbounds.resetTraffic'), icon: 'reset' },
    ...(ib.protocol !== 'mtproto' ? [{ key: 'clone', label: t('inbounds.clone'), icon: 'clone' as const }] : []),
    { key: 'delete', label: t('inbounds.delete'), icon: 'delete', danger: true },
  )
  if (phone) items.push({ key: 'enable', label: t('inbounds.enable'), switch: true, checked: ib.enable, busy: toggling.has(ib.id), divided: true })
  return items
}

function act(key: string, ib: DbInbound) {
  const base = 'panel/api/inbounds'
  switch (key) {
    case 'enable':
      return setEnable(ib, !ib.enable)
    case 'edit':
      return openInbound(ib)
    case 'addClient':
      return openClient(ib, null)
    case 'addBulk':
      bulkInbound.value = ib
      bulkOpen.value = true
      return
    case 'copyClients':
      copyInbound.value = ib
      copyOpen.value = true
      return
    case 'info':
      return openInfo(ib, null)
    case 'qr':
      return openQr(ib, null)
    case 'exportLinks':
      return showText(t('inbounds.exportLinks'), linksOf(ib).join('\r\n'), ib.remark)
    case 'exportSubs':
      return showText(t('inbounds.exportSubs'), subsOf([ib]).join('\n'), `${ib.remark}-Subs`)
    case 'exportInbound':
      return showText(t('inbounds.inboundData'), JSON.stringify(ib, null, 2))
    case 'clone':
      return ask({
        title: `${t('inbounds.cloneTitle')} "${ib.remark}"`,
        text: t('inbounds.cloneText'),
        cta: t('inbounds.cloneOk'),
        icon: 'info',
        run: () => api.post(`${base}/add`, toForm(cloneOf(ib))),
      })
    case 'resetTraffic':
      return ask({ title: `${t('inbounds.resetTraffic')} #${ib.id}`, text: t('inbounds.resetConfirm'), cta: t('inbounds.reset'), danger: true, run: () => resetTraffic(ib) })
    case 'delete':
      return ask({ title: `${t('inbounds.deleteInbound')} #${ib.id}`, text: t('inbounds.deleteInboundText'), cta: t('inbounds.delete'), danger: true, run: () => api.post(`${base}/del/${ib.id}`) })
    case 'resetClients':
      return ask({ title: t('inbounds.resetClientsTitle'), text: t('inbounds.resetClientsText'), cta: t('inbounds.reset'), danger: true, run: () => api.post(`${base}/resetAllClientTraffics/${ib.id}`) })
    case 'delDepleted':
      return ask({ title: t('inbounds.delDepletedTitle'), text: t('inbounds.delDepletedText'), cta: t('inbounds.delete'), danger: true, run: () => api.post(`${base}/delDepletedClients/${ib.id}`) })
  }
}

/** The general actions over every inbound. */
function actAll(key: string) {
  const base = 'panel/api/inbounds'
  switch (key) {
    case 'import':
      importOpen.value = true
      return
    case 'exportLinks':
      return showText(t('inbounds.exportLinks'), store.list.flatMap(linksOf).join('\r\n'), 'All-Inbounds')
    case 'exportSubs':
      return showText(t('inbounds.exportSubs'), subsOf(store.list).join('\r\n'), 'All-Inbounds-Subs')
    case 'resetAllInbounds':
      return ask({ title: t('inbounds.resetAllInbounds'), text: t('inbounds.resetAllInboundsText'), cta: t('inbounds.reset'), danger: true, run: () => api.post(`${base}/resetAllTraffics`) })
    case 'resetAllClients':
      return ask({ title: t('inbounds.resetAllClients'), text: t('inbounds.resetAllClientsText'), cta: t('inbounds.reset'), danger: true, run: () => api.post(`${base}/resetAllClientTraffics/-1`) })
    case 'delDepleted':
      return ask({ title: t('inbounds.delDepletedTitle'), text: t('inbounds.delDepletedText'), cta: t('inbounds.delete'), danger: true, run: () => api.post(`${base}/delDepletedClients/-1`) })
  }
}

/** The inbound saved again with its counters at zero, as the old panel reset one inbound's traffic. */
function resetTraffic(ib: DbInbound) {
  const form = new FormData()
  const fields = {
    up: 0, down: 0, total: ib.total, remark: ib.remark, enable: ib.enable, expiryTime: ib.expiryTime, trafficReset: ib.trafficReset,
    lastTrafficResetTime: ib.lastTrafficResetTime, listen: ib.listen, port: ib.port, protocol: ib.protocol, settings: ib.settings,
    streamSettings: ib.streamSettings, sniffing: ib.sniffing,
  }
  for (const [k, v] of Object.entries(fields)) form.set(k, String(v))
  return api.post(`panel/api/inbounds/update/${ib.id}`, form)
}

// --- The inbound window ----------------------------------------------------------

const inboundOpen = ref(false)
const editing = shallowRef<DbInbound | null>(null)

function openInbound(ib: DbInbound | null) {
  editing.value = ib
  inboundOpen.value = true
}

// --- The client window -------------------------------------------------------------

const clientOpen = ref(false)
const clientInbound = shallowRef<DbInbound | null>(null)
const clientEditing = shallowRef<Client | null>(null)

function openClient(ib: DbInbound, c: Client | null) {
  clientInbound.value = ib
  clientEditing.value = c
  clientOpen.value = true
}

const bulkOpen = ref(false)
const bulkInbound = shallowRef<DbInbound | null>(null)
const copyOpen = ref(false)
const copyInbound = shallowRef<DbInbound | null>(null)

const infoOpen = ref(false)
const infoInbound = shallowRef<DbInbound | null>(null)
const infoClient = shallowRef<Client | null>(null)

function openInfo(ib: DbInbound, c: Client | null) {
  infoInbound.value = ib
  infoClient.value = c
  infoOpen.value = true
}

// --- Exports and import ------------------------------------------------------------

const textOpen = ref(false)
const text = shallowRef({ title: '', text: '', fileName: '' })
const importOpen = ref(false)

function showText(title: string, body: string, fileName = '') {
  text.value = { title, text: body, fileName }
  textOpen.value = true
}

/** An inbound's share links, a fallback's through the inbound it hangs on. */
const linksOf = (ib: DbInbound) => inboundLinks(linkForm(ib, store.list), store.defaults?.remarkModel || '-ieo')
const subsOf = (list: DbInbound[]) => subLinks(list.flatMap((ib) => clientsOf(ib)), store.defaults?.subURI ?? '')

const qrOpen = ref(false)
const qrInbound = shallowRef<DbInbound | null>(null)
const qrClient = shallowRef<Client | null>(null)

function openQr(ib: DbInbound, c: Client | null) {
  qrInbound.value = ib
  qrClient.value = c
  qrOpen.value = true
}

function clientAction(key: 'qr' | 'edit' | 'info', ib: DbInbound, c: Client) {
  if (key === 'edit') openClient(ib, c)
  else if (key === 'info') openInfo(ib, c)
  else openQr(ib, c)
}

// --- Asking first ------------------------------------------------------------------

const answering = ref(false)
/** The last question, kept while the window fades out. */
const question = shallowRef<Ask | null>(null)
watch(asked, (q) => q && (question.value = q))

onBeforeUnmount(() => (asked.value = null))

function dismiss() {
  if (!answering.value) asked.value = null
}

/** The action asked about: done while the window waits, then the list is fetched again. */
async function answer() {
  const q = asked.value
  if (!q || answering.value) return
  answering.value = true
  try {
    await q.run()
  } catch (e) {
    toast('err', cause(e))
  } finally {
    answering.value = false
    asked.value = null
  }
  await store.load().catch(() => {})
}

// --- Enabling ----------------------------------------------------------------------

/** Inbounds whose switch is being applied, and those whose change failed a moment ago. */
const toggling = reactive(new Set<number>())
const refused = reactive(new Set<number>())

/** Applied at once and rolled back if the server refuses, as on the old panel. */
async function setEnable(ib: DbInbound, on: boolean) {
  if (toggling.has(ib.id)) return
  const before = ib.enable
  store.patch(ib.id, { enable: on })
  toggling.add(ib.id)
  const form = new FormData()
  form.set('enable', String(on))
  try {
    await api.post(`panel/api/inbounds/setEnable/${ib.id}`, form)
  } catch {
    store.patch(ib.id, { enable: before })
    refused.add(ib.id)
    window.setTimeout(() => refused.delete(ib.id), 3400)
    toast('err', t('inbounds.toggleFailed'))
  } finally {
    toggling.delete(ib.id)
  }
}

// --- Page ----------------------------------------------------------------------------

const bannerClosed = ref(false)
const showBanner = computed(() => store.loaded && !bannerClosed.value && insecure())
const subscription = computed(() => !!store.defaults?.subEnable)

// --- Open rows -----------------------------------------------------------------------
// While searching or filtering, the inbounds found stand open on their clients
// and the closed ones are remembered instead, as in the mockup.

const narrowed = computed(() => (mode.value === 'search' ? !!query.value : filter.value !== 'none'))
const opened = ref<number[]>([])
const shut = ref<number[]>([])
watch([query, filter, mode], () => (shut.value = []))

const open = computed<(string | number)[]>({
  get: () => (narrowed.value ? rows.value.filter((ib) => hasClients(ib) && !shut.value.includes(ib.id)).map((ib) => ib.id) : opened.value),
  set: (keys) => {
    if (narrowed.value) shut.value = rows.value.filter((ib) => hasClients(ib) && !keys.includes(ib.id)).map((ib) => ib.id)
    else opened.value = keys as number[]
  },
})
const clientQuery = computed(() => (mode.value === 'search' ? query.value : ''))
const clientFilter = computed<Filter>(() => (mode.value === 'filter' ? filter.value : 'none'))
</script>

<template>
  <AppPage :title="t('inbounds.title')">
    <template v-if="showBanner" #banner>
      <XNotice tone="err" :title="t('settings.securityTitle')" closable @close="bannerClosed = true">{{ t('xray.securityText') }}</XNotice>
    </template>

    <div v-if="!store.loaded && !store.failed" class="inbounds__loading" role="status"><XSpinner :size="16" /></div>
    <XCard v-else-if="store.failed">
      <XEmpty tone="err" icon="alert" :title="t('settings.loadFailed')" />
    </XCard>

    <template v-else>
      <XCard class="inbounds__summary">
        <div :class="['inbounds__stats', { 'inbounds__stats--compact': compact }]">
          <XSummaryStat icon="updown" :compact="compact" :label="t('inbounds.statSentReceived')" :value="`${formatBytes(totals.up)} / ${formatBytes(totals.down)}`" />
          <XSummaryStat icon="database" :compact="compact" :label="t('inbounds.statTraffic')" :value="formatBytes(totals.up + totals.down)" />
          <XSummaryStat icon="clock" :compact="compact" :label="t('inbounds.statAllTime')" :value="formatBytes(totals.allTime)" />
          <XSummaryStat icon="swap" :compact="compact" :label="t('inbounds.statInbounds')" :value="String(store.list.length)" />
          <XSummaryStat icon="user" :compact="compact" :label="t('inbounds.clients')" class="inbounds__clients">
            <ClientCounts :counts="allCounts" labelled />
          </XSummaryStat>
        </div>
      </XCard>

      <XExpandTable
        v-model:open="open"
        :columns="columns"
        :phone-columns="phoneColumns"
        :rows="rows"
        :row-key="(ib) => ib.id"
        :expandable="(ib) => hasClients(ib)"
        :muted="(ib) => !ib.enable"
        :toggle-label="t('inbounds.clients')"
        :compact-below="1000"
        @compact="compact = $event"
      >
        <template #tools>
          <InboundsTools v-model:mode="mode" v-model:search="search" v-model:filter="filter" :subscription="subscription" @action="actAll" @create="openInbound(null)" />
        </template>

        <template #cell-id="{ row }">
          <span class="inbounds__id">{{ row.id }}</span>
        </template>
        <template #cell-menu="{ row }">
          <XMenu v-slot="{ props: trigger }" :items="menu(row)" placement="bottom-start" @select="act($event, row)">
            <XInlineButton v-bind="trigger" icon="dots" :label="t('inbounds.colMenu')" />
          </XMenu>
        </template>
        <template #cell-enable="{ row }">
          <XSwitch
            :model-value="row.enable"
            :loading="toggling.has(row.id)"
            :invalid="refused.has(row.id)"
            :label="t('inbounds.enable')"
            @update:model-value="setEnable(row, $event)"
          />
        </template>
        <template #cell-remark="{ row }">
          <div class="inbounds__remark" :title="row.remark"><XMark :text="row.remark" :query="clientQuery" /></div>
        </template>
        <template #cell-port="{ row }">
          <span class="inbounds__port"><XMark :text="row.port" :query="clientQuery" /></span>
        </template>
        <template #cell-protocol="{ row }">
          <div class="inbounds__chips">
            <XChip mono>{{ protocolChips(row).protocol }}</XChip>
            <XChip v-if="protocolChips(row).network" mono outline>{{ protocolChips(row).network }}</XChip>
            <XChip v-if="protocolChips(row).security" tone="ok" mono outline>{{ security(protocolChips(row).security!) }}</XChip>
          </div>
        </template>
        <template #cell-clients="{ row }">
          <ClientCounts v-if="hasClients(row) && countsOf(row)" :counts="countsOf(row)!" />
        </template>
        <template #cell-traffic="{ row }">
          <XTooltip :delay="0">
            <XChip :tone="toneOf(traffic(row))" mono interactive>{{ formatBytes(used(row)) }} / {{ row.total ? formatBytes(row.total) : '∞' }}</XChip>
            <template #content>
              <div class="inbounds__updown">↑{{ formatBytes(row.up) }} ↓{{ formatBytes(row.down) }}</div>
              <div v-if="row.total && used(row) < row.total" class="inbounds__remain">
                <span>{{ t('inbounds.remain') }}</span><span>{{ formatBytes(row.total - used(row)) }}</span>
              </div>
            </template>
          </XTooltip>
        </template>
        <template #cell-allTime="{ row }">
          <XChip mono>{{ formatBytes(row.allTime || used(row)) }}</XChip>
        </template>
        <template #cell-expiry="{ row }">
          <XChip v-if="!row.expiryTime" icon="infinity" :aria-label="t('inbounds.noExpiry')" />
          <XTooltip v-else :text="formatDate(row.expiryTime)" :delay="0">
            <XChip :tone="toneOf(expiry(row))" interactive>{{ formatRelative(row.expiryTime) }}</XChip>
          </XTooltip>
        </template>

        <template #phone-menu="{ row }">
          <XMenu v-slot="{ props: trigger }" :items="menu(row, true)" placement="bottom-start" @select="act($event, row)">
            <XButton v-bind="trigger" variant="ghost" icon="dots" :label="t('inbounds.colMenu')" />
          </XMenu>
        </template>
        <template #phone-remark="{ row, open: isOpen, expandable, toggle }">
          <XRowToggle :open="isOpen" :expandable="expandable" @toggle="toggle">
            <span :class="['inbounds__remark', 'inbounds__remark--phone', { 'is-off': !row.enable }]"><XMark :text="row.remark" :query="clientQuery" /></span>
          </XRowToggle>
        </template>
        <template #phone-info="{ row }">
          <XPopover :label="t('inbounds.colInfo')">
            <template #default="{ props: trigger }">
              <XButton v-bind="trigger" icon="info" :mark="row.enable ? undefined : 'pause'" :label="t('inbounds.colInfo')" />
            </template>
            <template #content>
              <XKeyValue
                plain
                class="inbounds__facts"
                :rows="[
                  { key: 'protocol', label: t('inbounds.colProtocol') },
                  { label: t('inbounds.colPort'), value: String(row.port) },
                  ...(hasClients(row) ? [{ key: 'clients', label: t('inbounds.clients') }] : []),
                  { key: 'traffic', label: t('inbounds.colTraffic') },
                  { key: 'expiry', label: t('inbounds.colExpiry') },
                  { label: t('inbounds.trafficReset'), value: row.trafficReset },
                ]"
              >
                <template #value-protocol>
                  <XChip mono>{{ protocolChips(row).protocol }}</XChip>
                  <XChip v-if="protocolChips(row).network" mono outline>{{ protocolChips(row).network }}</XChip>
                  <XChip v-if="protocolChips(row).security" tone="ok" mono outline>{{ security(protocolChips(row).security!) }}</XChip>
                </template>
                <template #value-clients>
                  <ClientCounts v-if="countsOf(row)" :counts="countsOf(row)!" />
                </template>
                <template #value-traffic>
                  <XTooltip :delay="0">
                    <XChip :tone="toneOf(traffic(row))" mono interactive>{{ formatBytes(used(row)) }} / {{ row.total ? formatBytes(row.total) : '∞' }}</XChip>
                    <template #content>
                      <div class="inbounds__updown">↑{{ formatBytes(row.up) }} ↓{{ formatBytes(row.down) }}</div>
                      <div v-if="row.total && used(row) < row.total" class="inbounds__remain">
                        <span>{{ t('inbounds.remain') }}</span><span>{{ formatBytes(row.total - used(row)) }}</span>
                      </div>
                    </template>
                  </XTooltip>
                </template>
                <template #value-expiry>
                  <XChip v-if="!row.expiryTime" icon="infinity" :aria-label="t('inbounds.noExpiry')" />
                  <XChip v-else :tone="expiry(row) === 'err' ? 'err' : 'on'" mono>{{ formatDate(row.expiryTime) }}</XChip>
                </template>
              </XKeyValue>
            </template>
          </XPopover>
        </template>

        <template #expanded="{ row }">
          <InboundClients :inbound="row" :query="clientQuery" :filter="clientFilter" @action="(key, c) => clientAction(key, row, c)" />
        </template>

        <template #empty>
          <XEmpty icon="inbox" :title="t('inbounds.noData')" tab />
        </template>
        <template v-if="shown.length > 25" #foot>
          <XPager v-model="page" v-model:size="size" :pages="pages" :sizes="sizes" class="inbounds__pager" />
        </template>
      </XExpandTable>
    </template>
  </AppPage>

  <InboundModal v-model:open="inboundOpen" :inbound="editing" />
  <ClientModal v-model:open="clientOpen" :inbound="clientInbound" :client="clientEditing" />
  <BulkClientModal v-model:open="bulkOpen" :inbound="bulkInbound" />
  <CopyClientsModal v-model:open="copyOpen" :inbound="copyInbound" />
  <InfoModal v-model:open="infoOpen" :inbound="infoInbound" :client="infoClient" />
  <QrModal v-model:open="qrOpen" :inbound="qrInbound" :client="qrClient" />
  <TextModal v-model:open="textOpen" :title="text.title" :text="text.text" :file-name="text.fileName" />
  <ImportModal v-model:open="importOpen" />

  <XConfirm
    :open="!!asked"
    :title="question?.title ?? ''"
    :cta="question?.cta ?? ''"
    :danger="question?.danger"
    :icon="question?.icon"
    :busy="answering"
    @update:open="!$event && dismiss()"
    @confirm="answer"
  >
    {{ question?.text }}
  </XConfirm>
</template>

<style scoped>
.inbounds__loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 640px;
  color: var(--acc-t);
}

.inbounds__stats {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(0, .95fr) minmax(0, 1.15fr) minmax(0, 1fr) minmax(0, 1.6fr);
  gap: 18px;
}

.inbounds__remark {
  display: -webkit-box;
  overflow: hidden;
  color: var(--fg);
  font: 500 var(--fs-13) / 1.4 var(--font-sans);
  word-break: break-word;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.inbounds__remark--phone {
  font-size: var(--fs-14);
}

.inbounds__remark--phone.is-off {
  color: var(--mut);
}

.inbounds__id,
.inbounds__port {
  color: var(--fg2);
  font: 400 var(--fs-13) / var(--lh-tight) var(--font-mono);
}

.inbounds__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.inbounds__updown {
  color: var(--fg);
  font: 500 var(--fs-12) / var(--lh-tight) var(--font-mono);
  white-space: nowrap;
}

.inbounds__remain {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px solid var(--div);
  color: var(--dim);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-sans);
}

.inbounds__remain span:last-child {
  color: var(--fg);
  font: 500 var(--fs-12) / var(--lh-tight) var(--font-mono);
}

.inbounds__facts {
  /* A phone's popover: wide enough for phone type, short of the screen's edges. */
  width: min(320px, calc(100vw - 56px));
}

.inbounds__pager {
  justify-content: flex-end;
}

/* One a row with the table's phone columns, as on a phone. */
.inbounds__stats--compact {
  grid-template-columns: minmax(0, 1fr);
  gap: 14px;
}
</style>
