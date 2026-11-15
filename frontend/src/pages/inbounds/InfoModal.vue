<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, saveFile } from '../../api'
import { toast, XButton, XChip, XCopyBlock, XCopyChip, XDivider, XFactRow, XFactTile, XModal, XSkeletonText, type Tone } from '../../kit'
import { formatBytes, formatDate } from '../../i18n/format'
import { useInbounds } from '../../stores/inbounds'
import { cause } from '../settings/form'
import { canFlow, isSS2022, network, security, withClients } from './inbound'
import { allLinks, linkAddress, linkForm, wireguardConfigs, wireguardLinks, wireguardPublicKey } from './links'
import { delayedDays, expiryState, hasLink, remainingOf, trafficState, usedOf, type Client, type DbInbound, type State } from './model'

/**
 * Window 15 of the mockup: what the old panel's details showed of an inbound
 * and, for one of its clients, the client: its state, traffic and dates, its
 * IP log, its subscription and share links. Inbounds without clients show
 * their own settings: WireGuard its peers with their configs and links.
 */
const props = defineProps<{ inbound: DbInbound | null; client: Client | null }>()
const open = defineModel<boolean>('open', { default: false })
const { t } = useI18n()
const store = useInbounds()

type Json = Record<string, any>

/** The inbound as its links see it: a fallback through the inbound it hangs on. */
const form = computed(() => (props.inbound ? linkForm(props.inbound, store.list) : null))
const st = computed<Json>(() => form.value?.streamSettings ?? {})
const s = computed<Json>(() => form.value?.settings ?? {})
const protocol = computed(() => props.inbound?.protocol)
const remarkModel = computed(() => store.defaults?.remarkModel || '-ieo')

// --- The inbound -------------------------------------------------------------------

const streamed = computed(() => ['vmess', 'vless', 'trojan', 'shadowsocks'].includes(protocol.value ?? ''))
const net = computed(() => (form.value ? network(form.value) : ''))
const sec = computed(() => (form.value ? security(form.value) : 'none'))
const netSettings = computed<Json>(() => st.value[`${net.value}Settings`] ?? {})

const headerOf = (headers: Json | undefined, name: string): string => {
  const hit = Object.entries(headers ?? {}).find(([k]) => k.toLowerCase() === name)?.[1]
  return String((Array.isArray(hit) ? hit[0] : hit) ?? '')
}

/** Host and path of the transports that have them; raw TCP only with its HTTP header. */
const host = computed(() => {
  const n = netSettings.value
  if (net.value === 'tcp') return n.header?.type === 'http' ? headerOf(n.header.request?.headers, 'host') : ''
  return n.host?.length > 0 ? n.host : headerOf(n.headers, 'host')
})
const path = computed(() => {
  const n = netSettings.value
  if (net.value === 'tcp') return n.header?.type === 'http' ? String(n.header.request?.path?.[0] ?? '') : ''
  return String(n.path ?? '')
})
const hostPath = computed(() => ['tcp', 'ws', 'httpupgrade', 'xhttp'].includes(net.value))

/** The name the inbound answers to: TLS's SNI, Reality's server names. */
const serverName = computed(() => {
  if (sec.value === 'tls') return String(st.value.tlsSettings?.serverName ?? '')
  if (sec.value === 'reality') return ([] as string[]).concat(st.value.realitySettings?.serverNames ?? []).join(',')
  return ''
})

// --- The client --------------------------------------------------------------------

const c = computed<Json | null>(() => props.client)
const stat = computed(() => (props.client ? (props.inbound?.clientStats ?? []).find((x) => x.email === props.client!.email) : undefined))
const thresholds = computed(() => ({ trafficDiff: store.defaults?.trafficDiff ?? 0, expireDiff: store.defaults?.expireDiff ?? 0 }))

const depleted = computed(() => {
  const x = stat.value
  if (!x || !c.value) return false
  const expiry = c.value.expiryTime ?? 0
  return (x.total > 0 && usedOf(x) >= x.total) || (expiry > 0 && Date.now() >= expiry)
})
const lastOnline = computed(() => {
  const at = (c.value && store.lastOnline[c.value.email]) || stat.value?.lastOnline || 0
  return at ? formatDate(at) : '-'
})
const dateOf = (at: unknown) => (typeof at === 'number' && at > 0 ? formatDate(at) : '-')

/** Tiles colour a figure only when it calls for attention. */
const tileTone = (state: State): Tone => (state === 'warn' || state === 'err' ? state : 'neutral')
const trafficTone = computed(() => tileTone(stat.value ? trafficState(usedOf(stat.value), stat.value.total, thresholds.value) : 'unl'))

const mtproto = computed(() => protocol.value === 'mtproto')
const ipLimit = computed(() => !!store.defaults?.ipLimitEnable || mtproto.value)
const ipLog = computed(() => !!c.value?.email && (mtproto.value || (!!store.defaults?.ipLimitEnable && (c.value.limitIp ?? 0) > 0)))
const ips = ref<string[]>([])
const ipsLoading = ref(false)
const clearingIps = ref(false)

async function loadIps() {
  ipsLoading.value = true
  try {
    const list = await api.post<string[] | string>(`panel/api/inbounds/clientIps/${encodeURIComponent(c.value!.email)}`)
    ips.value = Array.isArray(list) ? list.map(String) : []
  } catch (e) {
    toast('err', cause(e))
  } finally {
    ipsLoading.value = false
  }
}

async function clearIps() {
  clearingIps.value = true
  try {
    await api.post(`panel/api/inbounds/clearClientIps/${encodeURIComponent(c.value!.email)}`)
    ips.value = []
  } catch (e) {
    toast('err', cause(e))
  } finally {
    clearingIps.value = false
  }
}

watch(open, (on) => {
  if (!on) return
  ips.value = []
  if (ipLog.value) void loadIps()
})

// --- Links -------------------------------------------------------------------------

const subscription = computed(() => !!store.defaults?.subEnable && !!c.value?.subId && !mtproto.value)
const links = computed(() => {
  const f = form.value
  if (!f || !hasLink(f.protocol)) return []
  if (c.value) return allLinks(f, remarkModel.value, c.value)
  // Shadowsocks with one key for everyone has the inbound's own link.
  return f.protocol === 'shadowsocks' && !withClients(f.protocol, f.settings) ? allLinks(f, remarkModel.value) : []
})

const peers = computed<Json[]>(() => (protocol.value === 'wireguard' && Array.isArray(s.value.peers) ? s.value.peers : []))
const peerConfigs = computed(() => (form.value && peers.value.length ? wireguardConfigs(form.value, remarkModel.value) : []))
const peerLinks = computed(() => (form.value && peers.value.length ? wireguardLinks(form.value, remarkModel.value) : []))
const wgPublicKey = computed(() => (protocol.value === 'wireguard' ? wireguardPublicKey(s.value.secretKey) : ''))

const accounts = computed<Json[]>(() => (Array.isArray(s.value.accounts) && (protocol.value === 'http' || s.value.auth === 'password') ? s.value.accounts : []))
</script>

<template>
  <XModal v-model:open="open" :title="t('inbounds.details')" size="form" scroll-body>
    <div v-if="inbound && form" class="info">
      <XFactRow :label="t('inbounds.colProtocol')"><XChip tone="unl" mono>{{ inbound.protocol }}</XChip></XFactRow>
      <XFactRow :label="t('inbounds.address')"><XChip mono truncate :title="linkAddress(form)">{{ linkAddress(form) }}</XChip></XFactRow>
      <XFactRow :label="t('inbounds.colPort')"><XChip mono>{{ form.port }}</XChip></XFactRow>

      <template v-if="streamed">
        <XFactRow :label="t('inbounds.transport')"><XChip mono>{{ net }}</XChip></XFactRow>
        <template v-if="hostPath">
          <XFactRow :label="t('inbounds.host')">
            <XChip v-if="host" mono truncate :title="host">{{ host }}</XChip>
            <XChip v-else tone="warn">{{ t('inbounds.none') }}</XChip>
          </XFactRow>
          <XFactRow :label="t('inbounds.path')">
            <XChip v-if="path" mono truncate :title="path">{{ path }}</XChip>
            <XChip v-else tone="warn">{{ t('inbounds.none') }}</XChip>
          </XFactRow>
        </template>
        <XFactRow v-if="net === 'xhttp'" :label="t('inbounds.mode')"><XChip mono>{{ netSettings.mode }}</XChip></XFactRow>
        <template v-if="net === 'grpc'">
          <XFactRow label="grpc serviceName"><XChip mono truncate :title="netSettings.serviceName">{{ netSettings.serviceName }}</XChip></XFactRow>
          <XFactRow label="grpc multiMode"><XChip mono>{{ !!netSettings.multiMode }}</XChip></XFactRow>
        </template>
      </template>

      <XFactRow v-if="mtproto" :label="t('inbounds.mtprotoDomain')"><XChip mono truncate>{{ s.tlsDomain }}</XChip></XFactRow>

      <template v-if="hasLink(inbound.protocol) && !mtproto">
        <XFactRow :label="t('inbounds.security')">
          <XChip :tone="sec === 'none' ? 'err' : 'ok'" mono>{{ sec }}</XChip>
        </XFactRow>
        <template v-if="inbound.protocol === 'vless'">
          <XFactRow :label="t('inbounds.authentication')">
            <XChip mono truncate>{{ s.selectedAuth || 'none' }}</XChip>
            <XCopyChip :text="s.selectedAuth || 'none'" />
          </XFactRow>
          <XFactRow :label="t('inbounds.encryption')">
            <XChip mono truncate :title="s.encryption">{{ s.encryption || 'none' }}</XChip>
            <XCopyChip :text="s.encryption || 'none'" />
          </XFactRow>
        </template>
        <XFactRow v-if="sec !== 'none'" :label="t('inbounds.domain')">
          <XChip v-if="serverName" mono truncate :title="serverName">{{ serverName }}</XChip>
          <XChip v-else tone="warn">{{ t('inbounds.none') }}</XChip>
        </XFactRow>
      </template>

      <template v-if="inbound.protocol === 'shadowsocks'">
        <XFactRow :label="t('inbounds.encryption')"><XChip mono>{{ s.method }}</XChip></XFactRow>
        <XFactRow v-if="isSS2022(s.method)" :label="t('inbounds.password')">
          <XChip mono truncate>{{ s.password }}</XChip>
          <XCopyChip :text="s.password" />
        </XFactRow>
        <XFactRow :label="t('inbounds.network')"><XChip mono>{{ s.network }}</XChip></XFactRow>
      </template>

      <!-- A client -->
      <template v-if="c">
        <XDivider :label="t('inbounds.client')" spaced />
        <XFactRow label="Email">
          <XChip v-if="c.email" truncate :title="c.email">{{ c.email }}</XChip>
          <XChip v-else tone="err">{{ t('inbounds.none') }}</XChip>
        </XFactRow>
        <XFactRow v-if="c.id && !mtproto" label="ID">
          <XChip mono truncate>{{ c.id }}</XChip>
          <XCopyChip :text="c.id" />
        </XFactRow>
        <XFactRow v-if="inbound.protocol === 'vmess'" :label="t('inbounds.security')"><XChip mono>{{ c.security }}</XChip></XFactRow>
        <XFactRow v-if="canFlow(form)" label="Flow">
          <XChip v-if="c.flow" mono>{{ c.flow }}</XChip>
          <XChip v-else tone="warn">{{ t('inbounds.none') }}</XChip>
        </XFactRow>
        <XFactRow v-if="c.password" :label="t('inbounds.password')">
          <XChip mono truncate>{{ c.password }}</XChip>
          <XCopyChip :text="c.password" />
        </XFactRow>
        <XFactRow :label="t('inbounds.status')">
          <XChip v-if="depleted" tone="err">{{ t('inbounds.stDepleted') }}</XChip>
          <XChip v-else-if="c.enable" tone="ok">{{ t('inbounds.enabled') }}</XChip>
          <XChip v-else>{{ t('inbounds.stDisabled') }}</XChip>
        </XFactRow>
        <XFactRow v-if="stat" :label="t('inbounds.usage')">
          <XChip mono>{{ formatBytes(usedOf(stat)) }}</XChip>
          <XChip mono>↑ {{ formatBytes(stat.up) }}</XChip>
          <XChip mono>↓ {{ formatBytes(stat.down) }}</XChip>
        </XFactRow>
        <XFactRow :label="t('inbounds.createdAt')"><XChip mono>{{ dateOf(c.created_at) }}</XChip></XFactRow>
        <XFactRow :label="t('inbounds.updatedAt')"><XChip mono>{{ dateOf(c.updated_at) }}</XChip></XFactRow>
        <XFactRow :label="t('inbounds.lastOnlineAt')"><XChip mono>{{ lastOnline }}</XChip></XFactRow>
        <XFactRow v-if="c.comment" :label="t('inbounds.commentLabel')"><XChip truncate :title="c.comment">{{ c.comment }}</XChip></XFactRow>
        <XFactRow v-if="ipLimit" :label="t('inbounds.ipLimit')"><XChip mono>{{ c.limitIp ?? 0 }}</XChip></XFactRow>
        <XFactRow v-if="ipLog" :label="t('inbounds.ipLog')">
          <XSkeletonText v-if="ipsLoading" :width="104" style="height: var(--chip-h)" />
          <template v-else>
            <XChip v-for="ip in ips" :key="ip" code>{{ ip }}</XChip>
            <XChip v-if="!ips.length" tone="warn">{{ t('inbounds.none') }}</XChip>
            <XButton v-if="!mtproto && ips.length" size="sm" variant="danger" :loading="clearingIps" @click="clearIps">{{ t('inbounds.ipLogClear') }}</XButton>
          </template>
        </XFactRow>

        <div class="info__tiles">
          <XFactTile :label="t('inbounds.remain')">
            <XChip v-if="stat && stat.total > 0" :tone="trafficTone" mono>{{ remainingOf(stat) > 0 ? formatBytes(remainingOf(stat)) : '-' }}</XChip>
            <XChip v-else icon="infinity" :aria-label="t('inbounds.unlimited')" />
          </XFactTile>
          <XFactTile :label="t('inbounds.totalFlow')">
            <XChip v-if="(c.totalGB ?? 0) > 0" :tone="trafficTone" mono>{{ formatBytes(c.totalGB) }}</XChip>
            <XChip v-else icon="infinity" :aria-label="t('inbounds.unlimited')" />
          </XFactTile>
          <XFactTile :label="t('inbounds.colExpiry')">
            <XChip v-if="(c.expiryTime ?? 0) > 0" :tone="tileTone(expiryState(c.expiryTime, thresholds))" mono>{{ formatDate(c.expiryTime) }}</XChip>
            <XChip v-else-if="(c.expiryTime ?? 0) < 0" mono>{{ t('inbounds.periodDays', { n: delayedDays(c.expiryTime) }) }}</XChip>
            <XChip v-else icon="infinity" :aria-label="t('inbounds.noExpiry')" />
          </XFactTile>
        </div>

        <template v-if="subscription">
          <XDivider :label="t('inbounds.subscriptionUrl')" spaced />
          <XCopyBlock label="Subscription Link" tone="unl" :text="(store.defaults?.subURI ?? '') + c.subId" />
          <XCopyBlock v-if="store.defaults?.subJsonEnable" label="Json Link" tone="unl" :text="(store.defaults?.subJsonURI ?? '') + c.subId" />
        </template>
      </template>

      <template v-if="links.length">
        <XDivider label="URL" spaced />
        <XCopyBlock v-for="l in links" :key="l.link" :label="l.remark" :text="l.link" />
      </template>

      <!-- Inbounds without clients -->
      <template v-if="inbound.protocol === 'tunnel'">
        <XFactRow :label="t('inbounds.targetAddress')"><XChip mono truncate>{{ s.address }}</XChip></XFactRow>
        <XFactRow :label="t('inbounds.destinationPort')"><XChip mono>{{ s.port }}</XChip></XFactRow>
        <XFactRow :label="t('inbounds.network')"><XChip mono>{{ s.network }}</XChip></XFactRow>
        <XFactRow label="FollowRedirect"><XChip mono>{{ !!s.followRedirect }}</XChip></XFactRow>
      </template>
      <template v-if="inbound.protocol === 'mixed'">
        <XFactRow :label="t('inbounds.auth')"><XChip mono>{{ s.auth }}</XChip></XFactRow>
        <XFactRow label="UDP"><XChip mono>{{ !!s.udp }}</XChip></XFactRow>
        <XFactRow label="IP"><XChip mono>{{ s.ip }}</XChip></XFactRow>
      </template>
      <XFactRow v-for="(a, i) in accounts" :key="i" :label="t('inbounds.accountN', { n: i + 1 })">
        <XChip mono truncate>{{ a.user }}</XChip>
        <XChip mono truncate>{{ a.pass }}</XChip>
        <XCopyChip :text="a.pass" />
      </XFactRow>

      <template v-if="inbound.protocol === 'wireguard'">
        <XFactRow :label="t('inbounds.wgSecretKey')">
          <XChip mono truncate>{{ s.secretKey || '-' }}</XChip>
          <XCopyChip v-if="s.secretKey" :text="s.secretKey" />
        </XFactRow>
        <XFactRow :label="t('inbounds.publicKey')">
          <XChip mono truncate>{{ wgPublicKey || '-' }}</XChip>
          <XCopyChip v-if="wgPublicKey" :text="wgPublicKey" />
        </XFactRow>
        <XFactRow label="MTU"><XChip mono>{{ s.mtu ?? 1420 }}</XChip></XFactRow>
        <XFactRow label="No Kernel Tun"><XChip mono>{{ !!s.noKernelTun }}</XChip></XFactRow>
        <template v-for="(p, i) in peers" :key="i">
          <XDivider :label="t('inbounds.peerN', { n: i + 1 })" spaced />
          <XFactRow :label="t('inbounds.wgSecretKey')">
            <XChip mono truncate>{{ p.privateKey || '-' }}</XChip>
            <XCopyChip v-if="p.privateKey" :text="p.privateKey" />
          </XFactRow>
          <XFactRow :label="t('inbounds.publicKey')">
            <XChip mono truncate>{{ p.publicKey || '-' }}</XChip>
            <XCopyChip v-if="p.publicKey" :text="p.publicKey" />
          </XFactRow>
          <XFactRow :label="t('inbounds.wgPsk')">
            <template v-if="p.preSharedKey">
              <XChip mono truncate>{{ p.preSharedKey }}</XChip>
              <XCopyChip :text="p.preSharedKey" />
            </template>
            <XChip v-else tone="warn">{{ t('inbounds.none') }}</XChip>
          </XFactRow>
          <XFactRow :label="t('inbounds.wgAllowedIPs')">
            <XChip v-for="a in p.allowedIPs ?? []" :key="a" mono>{{ a }}</XChip>
          </XFactRow>
          <XFactRow label="Keep Alive"><XChip mono>{{ p.keepAlive ?? 0 }}</XChip></XFactRow>
          <XCopyBlock label="Config" :text="peerConfigs[i] ?? ''">
            <template #actions>
              <XButton size="sm" icon="download" @click="saveFile(peerConfigs[i] ?? '', `peer-${i + 1}.conf`)">{{ t('inbounds.download') }}</XButton>
            </template>
          </XCopyBlock>
          <XCopyBlock label="Link" :text="peerLinks[i] ?? ''" />
        </template>
      </template>
    </div>
  </XModal>
</template>

<style scoped>
.info {
  display: flex;
  flex-direction: column;
  gap: var(--sp-12);
}

.info__tiles {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--sp-8);
}

@media (max-width: 767px) {
  .info {
    gap: var(--sp-14);
  }

  /* Two a line, the expiry across: a date and time do not fit a third of a phone. */
  .info__tiles {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .info__tiles > :last-child {
    grid-column: 1 / -1;
  }
}
</style>
