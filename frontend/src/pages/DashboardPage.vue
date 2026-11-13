<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, ApiError } from '../api'
import { config } from '../config'
import { bytesParts, formatBytes, formatDuration, formatNumber, formatSpeed } from '../i18n/format'
import { toast, XButton, XIcon, XInlineButton, XNotice, XRowLink, XSkeletonText, XStatusDot } from '../kit'
import AppPage from '../layout/AppPage.vue'
import { useServerStatus } from '../stores/server'
import { loadSite, siteTrouble, type SiteInfo } from './settings/site'
import DashCard from './dashboard/DashCard.vue'
import DashStat from './dashboard/DashStat.vue'
import BackupModal from './dashboard/BackupModal.vue'
import ConfigModal from './dashboard/ConfigModal.vue'
import XrayLogModal from './dashboard/XrayLogModal.vue'
import XrayVersionModal from './dashboard/XrayVersionModal.vue'
import PanelLogModal from './dashboard/PanelLogModal.vue'
import PanelUpdateModal, { type PanelUpdateInfo } from './dashboard/PanelUpdateModal.vue'
import ResourcesCard from './dashboard/ResourcesCard.vue'
import ServiceCard from './dashboard/ServiceCard.vue'

const { t } = useI18n()
const server = useServerStatus()
server.watch()
const status = computed(() => server.status)
// Until the first status arrives the figures are grey bars; if it cannot be fetched they fall back to dashes.
const loading = computed(() => !server.status && !server.failed)
const host = location.hostname

// --- Services -------------------------------------------------------------

type Service = 'xray' | 'telegram'
const busy = ref<Record<Service, 'restart' | 'stop' | null>>({ xray: null, telegram: null })

const endpoints = {
  xray: { restart: 'restartXrayService', stop: 'stopXrayService' },
  telegram: { restart: 'restartTelemtService', stop: 'stopTelemtService' },
} as const
const done = {
  xray: { restart: 'dashboard.xrayRestarted', stop: 'dashboard.xrayStopped' },
  telegram: { restart: 'dashboard.telegramRestarted', stop: 'dashboard.telegramStopped' },
} as const

async function act(service: Service, action: 'restart' | 'stop') {
  busy.value[service] = action
  try {
    await api.post(`panel/api/server/${endpoints[service][action]}`)
    toast('ok', t(done[service][action]))
  } catch (e) {
    toast('err', t('dashboard.actionFailed', { error: e instanceof ApiError ? e.message : String(e) }))
  } finally {
    await server.refresh().catch(() => {})
    busy.value[service] = null
  }
}

// --- Notices and small state ----------------------------------------------

const localHosts = ['localhost', '127.0.0.1', '[::1]']
const insecure = location.protocol === 'http:' && !localHosts.includes(location.hostname)
const httpDismissed = ref(read('dash.httpDismissed', sessionStorage) === '1')
function dismissHttp() {
  httpDismissed.value = true
  write('dash.httpDismissed', '1', sessionStorage)
}

// Each address has its own eye, like a password field; the choice is remembered.
type IpKind = 'ipv4' | 'ipv6'
const ipShown = ref<Record<IpKind, boolean>>({
  ipv4: read('dash.ipShown.ipv4', localStorage) === '1',
  ipv6: read('dash.ipShown.ipv6', localStorage) === '1',
})
function toggleIp(kind: IpKind) {
  ipShown.value[kind] = !ipShown.value[kind]
  write(`dash.ipShown.${kind}`, ipShown.value[kind] ? '1' : '0', localStorage)
}
const ipTiles = [
  { kind: 'ipv4', name: 'IPv4' },
  { kind: 'ipv6', name: 'IPv6' },
] as const

function read(key: string, storage: Storage) {
  try {
    return storage.getItem(key)
  } catch {
    return null
  }
}
function write(key: string, value: string, storage: Storage) {
  try {
    storage.setItem(key, value)
  } catch {
    // only this visit remembers it
  }
}

// The website: no valid certificate, no answer from outside or a failed renewal is told here.
const site = ref<SiteInfo | null>(null)
onMounted(async () => {
  try {
    site.value = await loadSite()
  } catch {
    // nothing to tell
  }
})
const trouble = computed(() => siteTrouble(site.value))
const troubleTitles = { cert: 'dashboard.certTitle', check: 'dashboard.siteTitle', renew: 'dashboard.renewTitle' } as const

const ip = (v?: string) => (!v || v === 'N/A' ? '—' : v)

// --- Windows ----------------------------------------------------------------

const panelLogOpen = ref(false)
const backupOpen = ref(false)
const configOpen = ref(false)
const xrayVersionOpen = ref(false)
const xrayLogOpen = ref(false)

// --- Panel version ----------------------------------------------------------

const update = ref<PanelUpdateInfo | null>(null)
const updateOpen = ref(false)
onMounted(async () => {
  try {
    update.value = await api.get('panel/api/server/getPanelUpdateInfo')
  } catch {
    // no answer from GitHub: nothing to offer
  }
})

const sent = computed(() => bytesParts(status.value?.netTraffic.sent ?? 0))
const received = computed(() => bytesParts(status.value?.netTraffic.recv ?? 0))
const dash = '—'
</script>

<template>
  <AppPage :title="t('nav.dashboard')">
    <template #subtitle>
      <i18n-t scope="global" keypath="common.server" tag="span">
        <template #host><span class="mono">{{ host }}</span></template>
      </i18n-t>
    </template>
    <template v-if="(insecure && !httpDismissed) || trouble" #aside>
      <XNotice v-if="trouble" :tone="trouble.tone" :title="t(troubleTitles[trouble.kind])" compact>
        {{ trouble.text || t('dashboard.certExpired') }}
        <template #actions>
          <XButton variant="ghost" icon="arrowRight" :label="t('dashboard.siteMore')" :to="{ name: 'settings', query: { tab: 'site' } }" />
        </template>
      </XNotice>
      <XNotice v-else tone="warn" :title="t('dashboard.httpTitle')" compact>
        {{ t('dashboard.httpText') }}
        <template #actions>
          <XButton variant="ghost" icon="close" :label="t('kit.close')" @click="dismissHttp" />
        </template>
      </XNotice>
    </template>

    <div class="dash">
      <div class="dash__col">
        <ServiceCard
          title="Xray"
          version-prefix="v"
          class="area-xray"
          :service="status?.xray"
          :loading="loading"
          :error-title="t('dashboard.xrayError')"
          :busy="busy.xray"
          @restart="act('xray', 'restart')"
          @stop="act('xray', 'stop')"
          @log="panelLogOpen = true"
        >
          <XRowLink icon="doc" :label="t('dashboard.xrayLog')" @click="xrayLogOpen = true" />
          <XRowLink icon="download" :label="t('dashboard.xrayVersion')" @click="xrayVersionOpen = true" />
        </ServiceCard>

        <ServiceCard
          v-if="status?.telemt"
          :title="t('dashboard.telegram')"
          class="area-telegram"
          :service="status.telemt"
          :error-title="t('dashboard.telegramError')"
          :busy="busy.telegram"
          @restart="act('telegram', 'restart')"
          @stop="act('telegram', 'stop')"
          @log="panelLogOpen = true"
        />

        <DashCard :title="t('dashboard.manage')" class="grow area-manage">
          <div class="links">
            <XRowLink icon="doc" :label="t('dashboard.panelLog')" @click="panelLogOpen = true" />
            <XRowLink icon="code" :label="t('dashboard.config')" @click="configOpen = true" />
            <XRowLink icon="database" :label="t('dashboard.backup')" @click="backupOpen = true" />
          </div>
          <div class="version">
            <span class="version__label">{{ t('dashboard.panelVersion') }}</span>
            <span class="mono version__value">v{{ config.version }}</span>
            <XButton v-if="update?.updateAvailable" variant="soft" size="sm" phone-block class="version__update" @click="updateOpen = true">
              <template #icon><XStatusDot size="sm" blink /></template>
              {{ t('dashboard.update', { version: update.latestVersion }) }}
            </XButton>
          </div>
        </DashCard>

        <DashCard :title="t('dashboard.ip')" class="area-ip">
          <div class="tiles tiles--ip">
            <div v-for="tile in ipTiles" :key="tile.kind" class="tile tile--ip">
              <div class="x-caps tile__label">{{ tile.name }}</div>
              <div :class="['tile__ip', { 'is-hidden': !ipShown[tile.kind] }]">
                <XSkeletonText v-if="loading" :width="104" />
                <template v-else>{{ ip(status?.publicIP[tile.kind]) }}</template>
              </div>
              <XInlineButton
                :icon="ipShown[tile.kind] ? 'eyeOff' : 'eye'"
                :label="t(ipShown[tile.kind] ? 'dashboard.hideAddress' : 'dashboard.showAddress', { name: tile.name })"
                :aria-pressed="ipShown[tile.kind]"
                class="tile__eye"
                @click="toggleIp(tile.kind)"
              />
            </div>
          </div>
        </DashCard>
      </div>

      <div class="dash__col">
        <ResourcesCard :status="status" :loading="loading" class="grow area-resources" />

        <DashCard :title="t('dashboard.network')" class="area-network">
          <div class="tiles">
            <div class="tile tile--total">
              <div class="x-caps tile__label">
                <XIcon name="arrowUp" :size="12" class="up" />
                {{ t('dashboard.sent') }}
              </div>
              <div class="tile__total up">
                <XSkeletonText v-if="loading" :width="88" />
                <template v-else>{{ status ? sent.n : dash }} <small v-if="status">{{ sent.unit }}</small></template>
              </div>
            </div>
            <div class="tile tile--total">
              <div class="x-caps tile__label">
                <XIcon name="arrowDown" :size="12" class="down" />
                {{ t('dashboard.received') }}
              </div>
              <div class="tile__total down">
                <XSkeletonText v-if="loading" :width="88" />
                <template v-else>{{ status ? received.n : dash }} <small v-if="status">{{ received.unit }}</small></template>
              </div>
            </div>
          </div>
          <div class="net-figures">
            <div class="net-figures__pair">
              <DashStat :label="t('dashboard.upload')" :value="status ? formatSpeed(status.netIO.up) : dash" :loading="loading" small tone="up" />
              <DashStat :label="t('dashboard.download')" :value="status ? formatSpeed(status.netIO.down) : dash" :loading="loading" small tone="down" />
            </div>
            <div class="net-figures__pair">
              <DashStat label="TCP" :value="status ? formatNumber(status.tcpCount) : dash" :loading="loading" small />
              <DashStat label="UDP" :value="status ? formatNumber(status.udpCount) : dash" :loading="loading" small />
            </div>
          </div>
        </DashCard>

        <DashCard :title="t('dashboard.system')" class="area-system">
          <div class="system">
            <DashStat :label="t('dashboard.osUptime')" :value="status ? formatDuration(status.uptime) : dash" :loading="loading" class="system__uptime" />
            <div class="system__uptime">
              <DashStat :label="t('dashboard.panelUptime')" :value="status ? formatDuration(status.appStats.uptime) : dash" :loading="loading" />
              <p v-if="status" class="system__process">
                {{ t('dashboard.panelProcess', { mem: formatBytes(status.appStats.mem), n: formatNumber(status.appStats.threads) }, status.appStats.threads) }}
              </p>
            </div>
            <DashStat :label="t('dashboard.load')" value="" :loading="loading" class="system__load">
              <template v-if="status?.loads?.length">
                <template v-for="(l, i) in status.loads" :key="i">
                  <span v-if="i" class="system__sep"> | </span>{{ formatNumber(l, 2) }}
                </template>
              </template>
              <template v-else>{{ dash }}</template>
            </DashStat>
          </div>
        </DashCard>
      </div>
    </div>

    <PanelLogModal v-model:open="panelLogOpen" />
    <BackupModal v-model:open="backupOpen" />
    <ConfigModal v-model:open="configOpen" />
    <XrayVersionModal v-model:open="xrayVersionOpen" />
    <XrayLogModal v-model:open="xrayLogOpen" />
    <PanelUpdateModal v-if="update" v-model:open="updateOpen" v-model:info="update" />
  </AppPage>
</template>

<style scoped>
.mono {
  color: var(--fg2);
  font-family: var(--font-mono);
}

.dash {
  display: grid;
  flex: 1 0 auto;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  align-items: stretch;
}

.dash__col {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}

.grow {
  flex: 1 0 auto;
}

/* Management */
.links {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-bottom: 12px;
}

.version {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-height: 28px;
  margin-top: auto;
  padding-top: 12px;
  border-top: 1px solid var(--div);
}

.version__label {
  color: var(--dim);
  font-size: var(--fs-12);
}

.version__value {
  font-size: var(--fs-12);
}

.version__update {
  margin-left: auto;
}

/* Tiles: IP addresses and traffic totals */

.tiles {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.tile {
  min-width: 0;
  padding: 10px 14px;
  border: 1px solid var(--field-line);
  border-radius: var(--nest-r);
  background: var(--field);
}

.tile--total {
  padding: 12px 14px;
}

/* The eye sits at the right, across the label and the address, like in a password field. */
.tile--ip {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  column-gap: 8px;
  align-items: center;
}

.tile__eye {
  grid-column: 2;
  grid-row: 1 / span 2;
}

.tile__label {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  white-space: nowrap;
}

.tile__ip {
  overflow: hidden;
  color: var(--fg);
  font: 500 var(--fs-14) / var(--lh-tight) var(--font-mono);
  white-space: nowrap;
  text-overflow: ellipsis;
  transition: filter var(--t-med);
}

.tile__ip.is-hidden {
  filter: blur(5px);
  user-select: none;
}

.tile__total {
  font: 600 var(--fs-20) / var(--lh-tight) var(--font-mono);
  letter-spacing: -.5px;
  white-space: nowrap;
}

.tile__total small {
  color: var(--mut);
  font-size: var(--fs-13);
  font-weight: 500;
  letter-spacing: 0;
}

.up {
  color: var(--ok);
}

.down {
  color: var(--on-t);
}

/* Network figures: two pairs under the two totals. */
.net-figures,
.net-figures__pair {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.net-figures {
  margin-top: 16px;
}

.net-figures__pair {
  padding: 0 14px;
}

/* System: the OS, the panel's own process and the load side by side; where
   the load does not fit, it goes under the uptimes. */
.system {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 16px 35px;
}

.system__uptime {
  flex: none;
}

.system__load {
  flex: 1 0 auto;
}

.system__sep {
  color: var(--faint);
}

.system__process {
  margin: 4px 0 0;
  color: var(--dim);
  font-size: var(--fs-11);
  white-space: nowrap;
}

@media (max-width: 1100px) {
  .dash {
    grid-template-columns: minmax(0, 1fr);
  }

  .grow {
    flex: none;
  }
}

/* Phones: one column in the order of the mockup. */
@media (max-width: 767px) {
  /* IP addresses need the full width next to their eye. */
  .tiles--ip {
    grid-template-columns: minmax(0, 1fr);
  }

  .dash {
    display: flex;
    flex-direction: column;
  }

  .dash__col {
    display: contents;
  }

  .area-xray { order: 1; }
  .area-telegram { order: 2; }
  .area-resources { order: 3; }
  .area-network { order: 4; }
  .area-ip { order: 5; }
  .area-manage { order: 6; }
  .area-system { order: 7; }

  /* Two rows of two, each figure under its total: four in a row leave no room for a speed. */
  .net-figures {
    grid-template-columns: minmax(0, 1fr);
  }

  .system {
    column-gap: 25px;
  }

  .tile__total {
    font-size: var(--fs-18);
  }

  /* On phones the update offer is a full-width button under the version. */
  .version {
    margin-top: 0;
    padding-top: 12px;
  }

  .version__update {
    margin: 2px 0 0;
  }
}
</style>
