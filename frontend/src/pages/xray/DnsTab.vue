<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  XButton, XEmpty, XGridTable, XInput, XNumber, XPairList, XSelect, XSettingRow, XSettingsSection, XSwitch, XValueChip, type GridColumn, type MenuItem,
} from '../../kit'
import { useXrayForm } from './form'
import { dnsAddress, dnsList, dnsStrategies, newDns, type DnsServer, type Text } from './model'
import DnsModal from './DnsModal.vue'
import DnsPresetsModal from './DnsPresetsModal.vue'
import FakeDnsModal from './FakeDnsModal.vue'

/**
 * Xray's own DNS: switched on, its settings, its servers and the Fake DNS
 * pools. A switch turned off leaves its key out, an emptied pool list leaves
 * the template, switching DNS off drops the servers and the pools.
 */
const { t } = useI18n()
const f = useXrayForm()

const open = reactive({ general: true, hosts: true, servers: true, fake: true })
const config = computed(() => f.draft.config)
const dns = computed(() => config.value.dns as Record<string, any> | undefined)

const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b)

// --- Settings -------------------------------------------------------------------

const enabled = computed({
  get: () => dns.value != null,
  set: (on: boolean) => {
    if (on) {
      config.value.dns = newDns()
      delete config.value.fakedns
    } else {
      delete config.value.dns
      delete config.value.fakedns
    }
  },
})

/** A switch of the DNS block: on writes true, off leaves the key out. */
const flag = (key: string) =>
  computed({
    get: () => !!dns.value?.[key],
    set: (on: boolean) => {
      if (!dns.value) return
      if (on) dns.value[key] = true
      else delete dns.value[key]
    },
  })

const v = reactive({
  tag: computed({ get: () => String(dns.value?.tag ?? ''), set: (x: string) => dns.value && (dns.value.tag = x) }),
  clientIp: computed({
    get: () => String(dns.value?.clientIp ?? ''),
    set: (x: string) => {
      if (!dns.value) return
      if (x) dns.value.clientIp = x
      else delete dns.value.clientIp
    },
  }),
  queryStrategy: computed({ get: () => String(dns.value?.queryStrategy ?? ''), set: (x: string) => dns.value && (dns.value.queryStrategy = x) }),
  /** 0, Xray's own, leaves the key out. */
  serveExpiredTTL: computed({
    get: () => Number(dns.value?.serveExpiredTTL ?? 0),
    set: (x: number | null) => {
      if (!dns.value) return
      if (x) dns.value.serveExpiredTTL = x
      else delete dns.value.serveExpiredTTL
    },
  }),
})

const flags = reactive({
  disableCache: flag('disableCache'),
  disableFallback: flag('disableFallback'),
  disableFallbackIfMatch: flag('disableFallbackIfMatch'),
  enableParallelQuery: flag('enableParallelQuery'),
  useSystemHosts: flag('useSystemHosts'),
  serveStale: flag('serveStale'),
})

type Flag = keyof typeof flags
type Key = keyof typeof v | Flag
const keys: Key[] = ['tag', 'clientIp', 'queryStrategy', 'disableCache', 'disableFallback', 'disableFallbackIfMatch', 'enableParallelQuery', 'useSystemHosts', 'serveStale', 'serveExpiredTTL']

const dirtyEnabled = computed(() => (f.draft.config.dns != null) !== (f.saved.config.dns != null))
const dirty = (k: Key) => !same(f.draft.config.dns?.[k], f.saved.config.dns?.[k])
const generalChanges = computed(() => +dirtyEnabled.value + (enabled.value ? keys.filter(dirty).length : 0))

const switches: { key: Flag; title: Text; description: Text }[] = [
  { key: 'disableCache', title: 'xray.dnsDisableCache', description: 'xray.dnsDisableCacheDesc' },
  { key: 'disableFallback', title: 'xray.dnsDisableFallback', description: 'xray.dnsDisableFallbackDesc' },
  { key: 'disableFallbackIfMatch', title: 'xray.dnsNoFallbackMatch', description: 'xray.dnsNoFallbackMatchDesc' },
  { key: 'enableParallelQuery', title: 'xray.dnsParallel', description: 'xray.dnsParallelDesc' },
  { key: 'useSystemHosts', title: 'xray.dnsSystemHosts', description: 'xray.dnsSystemHostsDesc' },
  { key: 'serveStale', title: 'xray.dnsServeStale', description: 'xray.dnsServeStaleDesc' },
]

// --- Hosts ----------------------------------------------------------------------
// A domain and its answer: one address as a string, several as a list, as Xray
// reads them. The rows live here, so one still without a domain stays shown;
// only rows with a domain reach the template, an emptied list leaves it.

type Row = [string, string]
const rowsOf = (hosts: unknown): Row[] =>
  hosts && typeof hosts === 'object'
    ? Object.entries(hosts as Record<string, unknown>).map(([d, a]) => [d, Array.isArray(a) ? a.join(', ') : String(a ?? '')])
    : []
function hostsOf(rows: Row[]) {
  const out: Record<string, string | string[]> = {}
  for (const [d, a] of rows) {
    const addrs = a.split(',').map((x) => x.trim()).filter(Boolean)
    if (d.trim() && addrs.length) out[d.trim()] = addrs.length === 1 ? addrs[0]! : addrs
  }
  return Object.keys(out).length ? out : undefined
}
const hostRows = ref<Row[]>(rowsOf(dns.value?.hosts))
let written = JSON.stringify(dns.value?.hosts ?? null)
watch(
  () => JSON.stringify(dns.value?.hosts ?? null),
  (now) => {
    // Loaded, saved over or set to the defaults elsewhere: the rows follow.
    if (now !== written) {
      written = now
      hostRows.value = rowsOf(dns.value?.hosts)
    }
  },
)
watch(
  hostRows,
  (rows) => {
    if (!dns.value) return
    const hosts = hostsOf(rows)
    if (hosts) dns.value.hosts = hosts
    else delete dns.value.hosts
    written = JSON.stringify(hosts ?? null)
  },
  { deep: true },
)
const hostsDirty = computed(() => !same(f.draft.config.dns?.hosts, f.saved.config.dns?.hosts))

const strategies = dnsStrategies.map((x) => ({ value: x, label: x }))

// --- Servers --------------------------------------------------------------------

const servers = computed<DnsServer[]>(() => (dns.value?.servers ?? []) as DnsServer[])
const serverColumns = computed<GridColumn[]>(() => [
  { key: 'address', label: t('xray.outAddress'), size: '1.3fr', phone: 'head' },
  { key: 'domains', label: t('xray.dnsDomains'), size: '1.2fr', phone: 'half' },
  { key: 'expectIPs', label: t('xray.dnsExpectIPs'), phone: 'half' },
])
const serverBlank = (s: DnsServer, key: string) => key !== 'address' && !dnsList(s, key as 'domains' | 'expectIPs').length

const menu = (): MenuItem[] => [
  { key: 'edit', label: t('xray.edit'), icon: 'edit' },
  { key: 'delete', label: t('xray.delete'), icon: 'delete', danger: true },
]

const serverOpen = ref(false)
const presetsOpen = ref(false)
const serverAt = ref(-1)

function addServer() {
  serverAt.value = -1
  serverOpen.value = true
}

function selectServer(key: string, _row: DnsServer, i: number) {
  if (!dns.value) return
  if (key === 'delete') dns.value.servers.splice(i, 1)
  else if (key === 'edit') {
    serverAt.value = i
    serverOpen.value = true
  }
}

function saveServer(s: DnsServer) {
  if (!dns.value) return
  const list = (dns.value.servers ??= []) as DnsServer[]
  if (serverAt.value >= 0) list[serverAt.value] = s
  else list.push(s)
}

function installPreset(list: string[]) {
  if (dns.value) dns.value.servers = list
}

// --- Fake DNS -------------------------------------------------------------------

type Pool = Record<string, any>
const pools = computed<Pool[]>(() => (Array.isArray(config.value.fakedns) ? config.value.fakedns : []))
const poolColumns = computed<GridColumn[]>(() => [
  { key: 'ipPool', label: t('xray.fakePool'), phone: 'head' },
  { key: 'poolSize', label: t('xray.fakePoolSize') },
])

const poolOpen = ref(false)
const poolAt = ref(-1)

function addPool() {
  poolAt.value = -1
  poolOpen.value = true
}

/** An emptied list leaves the template. */
function writePools(list: Pool[]) {
  if (list.length) config.value.fakedns = list
  else delete config.value.fakedns
}

function selectPool(key: string, _row: Pool, i: number) {
  if (key === 'delete') writePools(pools.value.filter((_, j) => j !== i))
  else if (key === 'edit') {
    poolAt.value = i
    poolOpen.value = true
  }
}

function savePool(p: Pool) {
  const list = [...pools.value]
  if (poolAt.value >= 0) list[poolAt.value] = p
  else list.push(p)
  writePools(list)
}
</script>

<template>
  <div class="xray-tab">
    <XSettingsSection v-model:open="open.general" :title="t('xray.secGeneral')" :count="enabled ? 11 : 1" :modified="generalChanges">
      <XSettingRow :title="t('xray.dnsEnable')" :description="t('xray.dnsEnableDesc')" inline :modified="dirtyEnabled">
        <XSwitch v-model="enabled" :label="t('xray.dnsEnable')" :modified="dirtyEnabled" />
      </XSettingRow>
      <template v-if="enabled">
        <XSettingRow :title="t('xray.dnsTag')" :description="t('xray.dnsTagDesc')" :modified="dirty('tag')">
          <XInput v-model="v.tag" mono :modified="dirty('tag')" />
        </XSettingRow>
        <XSettingRow :title="t('xray.dnsClientIp')" :description="t('xray.dnsClientIpDesc')" :modified="dirty('clientIp')">
          <XInput v-model="v.clientIp" mono placeholder="203.0.113.1" :modified="dirty('clientIp')" />
        </XSettingRow>
        <XSettingRow :title="t('xray.dnsStrategy')" :description="t('xray.dnsStrategyDesc')" :modified="dirty('queryStrategy')">
          <XSelect v-model="v.queryStrategy" :options="strategies" mono :modified="dirty('queryStrategy')" />
        </XSettingRow>
        <XSettingRow v-for="s in switches" :key="s.key" :title="t(s.title)" :description="t(s.description)" inline :modified="dirty(s.key)">
          <XSwitch v-model="flags[s.key]" :label="t(s.title)" :modified="dirty(s.key)" />
        </XSettingRow>
        <XSettingRow :title="t('xray.dnsExpiredTtl')" :description="t('xray.dnsExpiredTtlDesc')" :modified="dirty('serveExpiredTTL')">
          <XNumber v-model="v.serveExpiredTTL" block :min="0" :unit="t('settings.unitSeconds')" :modified="dirty('serveExpiredTTL')" />
        </XSettingRow>
      </template>
    </XSettingsSection>

    <XSettingsSection v-if="enabled" v-model:open="open.hosts" :title="t('xray.dnsHosts')" :count="hostRows.length" :modified="+hostsDirty" padded>
      <p class="dns-list__note">{{ t('xray.dnsHostsDesc') }}</p>
      <XPairList v-model="hostRows" :title="t('xray.dnsHostsList')" :name-label="t('xray.dnsHostsDomain')" :value-label="t('xray.dnsHostsAddress')" mono />
    </XSettingsSection>

    <template v-if="enabled">
      <XSettingsSection v-model:open="open.servers" title="DNS" :count="servers.length" padded>
        <div v-if="servers.length" class="dns-list">
          <div class="dns-list__tools">
            <XButton variant="primary" icon="plus" @click="addServer">{{ t('xray.dnsAdd') }}</XButton>
          </div>
          <XGridTable
            :columns="serverColumns"
            :rows="servers"
            :row-key="(_, i) => i"
            :blank="serverBlank"
            :menu="menu"
            :row-label="(i) => t('xray.dnsN', { n: i + 1 })"
            numbered
            nested
            @select="selectServer"
          >
            <template #cell-address="{ row }">
              <span class="dns-list__value">{{ dnsAddress(row) }}</span>
            </template>
            <template #cell-domains="{ row }">
              <XValueChip :values="dnsList(row, 'domains')" />
            </template>
            <template #cell-expectIPs="{ row }">
              <XValueChip :values="dnsList(row, 'expectIPs')" />
            </template>
          </XGridTable>
        </div>
        <XEmpty v-else framed :title="t('xray.dnsEmpty')">
          <XButton variant="primary" icon="plus" @click="addServer">{{ t('xray.dnsAdd') }}</XButton>
          <XButton icon="templates" :label="t('xray.dnsPresets')" @click="presetsOpen = true" />
        </XEmpty>
      </XSettingsSection>

      <XSettingsSection v-model:open="open.fake" title="Fake DNS" :count="pools.length" padded>
        <div v-if="pools.length" class="dns-list">
          <div class="dns-list__tools">
            <XButton variant="primary" icon="plus" @click="addPool">{{ t('xray.fakeAdd') }}</XButton>
          </div>
          <XGridTable
            :columns="poolColumns"
            :rows="pools"
            :row-key="(_, i) => i"
            :menu="menu"
            :row-label="(i) => t('xray.fakeN', { n: i + 1 })"
            numbered
            nested
            @select="selectPool"
          >
            <template #cell-ipPool="{ row }">
              <span class="dns-list__value">{{ row.ipPool }}</span>
            </template>
            <template #cell-poolSize="{ row }">
              <span class="dns-list__value dns-list__value--quiet">{{ row.poolSize }}</span>
            </template>
          </XGridTable>
        </div>
        <XEmpty v-else framed :title="t('xray.fakeEmpty')">
          <XButton variant="primary" icon="plus" @click="addPool">{{ t('xray.fakeAdd') }}</XButton>
        </XEmpty>
      </XSettingsSection>
    </template>

    <DnsModal v-model:open="serverOpen" :server="serverAt >= 0 ? servers[serverAt] ?? null : null" :index="serverAt" @save="saveServer" />
    <DnsPresetsModal v-model:open="presetsOpen" @install="installPreset" />
    <FakeDnsModal v-model:open="poolOpen" :pool="poolAt >= 0 ? pools[poolAt] ?? null : null" :index="poolAt" @save="savePool" />
  </div>
</template>

<style scoped>
.xray-tab {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dns-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dns-list__tools {
  display: flex;
}

.dns-list__note {
  margin: 0 0 12px;
  color: var(--fg2);
  font: 400 var(--fs-12) / var(--lh-normal) var(--font-sans);
}

/* On phones the address fills the card's head. */
.dns-list__value {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--fg);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-mono);
}

.dns-list__value--quiet {
  color: var(--fg2);
}
</style>
