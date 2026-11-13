<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  toast, XAddButton, XButton, XButtonRow, XField, XFieldGrid, XFieldNote, XInlineButton, XInput, XInputList, XListHead, XNest, XNestCard, XNumber, XPairList, XPartHead, XSelect,
  XSwitch, XTagInput, type Option,
} from '../../kit'
import { wireguardKeys } from '../../wireguard'
import { cause } from '../settings/form'
import InboundSockopt from './InboundSockopt.vue'
import { headerRows } from './rows'
import {
  canVisionSeed, isSS2022, network, newFallback, newPeer, newVlessEnc, randomVisionSeed, setSSMethod, ssMethods, ssPassword, visionSeed, type InboundForm,
  tunMissing,
  wfpLeakWrong,
} from './inbound'

/**
 * The protocol's own settings in window 11, as the old panel's protocol forms
 * had them: VLESS authentication, fallbacks and Vision seed; Trojan fallbacks;
 * Shadowsocks method and key; WireGuard keys and peers; mixed and HTTP
 * accounts; tunnel port mapping; TUN; MTProto's masking domain and WEB proxy.
 */
const props = defineProps<{ f: InboundForm }>()
const { t } = useI18n()

const s = computed(() => props.f.settings)
const p = computed(() => props.f.protocol)
const list = (values: string[]): Option[] => values.map((v) => ({ value: v, label: v }))
/** What Xray's TUN blocks from leaking past it on Windows: DNS, its client, a broken TUN, or a port such as 53. */
const wfpLeaks = [{ options: list(['dns', 'windnsclient', 'misconfigtun', 'udp/53', 'tcp/853']) }]
const networkOptions: Option[] = [
  { value: 'tcp,udp', label: 'TCP,UDP' },
  { value: 'tcp', label: 'TCP' },
  { value: 'udp', label: 'UDP' },
]

/** Whether the part has anything to show: VMess has nothing beyond its clients. */
const shown = computed(() => p.value !== 'vmess')

// --- VLESS ----------------------------------------------------------------------------

const authOptions: Option[] = [
  { value: '', label: 'None' },
  { value: 'X25519, not Post-Quantum', label: 'X25519 (not Post-Quantum)' },
  { value: 'ML-KEM-768, Post-Quantum', label: 'ML-KEM-768 (Post-Quantum)' },
]
const fetching = ref(false)

/** Keys for the chosen authentication from the server, or none without one. */
async function fetchVlessEnc() {
  const label = s.value.selectedAuth
  if (!label) return clearVlessEnc()
  fetching.value = true
  try {
    const block = (await newVlessEnc()).auths?.find((a) => a.label === label)
    if (block) {
      s.value.decryption = block.decryption
      s.value.encryption = block.encryption
    }
  } catch (e) {
    toast('err', cause(e))
  } finally {
    fetching.value = false
  }
}

function clearVlessEnc() {
  s.value.decryption = 'none'
  s.value.encryption = 'none'
  s.value.selectedAuth = undefined
}

const selectedAuth = computed({
  get: () => String(s.value.selectedAuth ?? ''),
  set: (v: string) => {
    s.value.selectedAuth = v || undefined
    void fetchVlessEnc()
  },
})

/** Fallbacks go with raw TCP, and for VLESS only without authentication. */
const fallbacks = computed(() => network(props.f) === 'tcp' && (p.value === 'trojan' || (p.value === 'vless' && !s.value.selectedAuth)))

const fallbackList = computed<Record<string, any>[]>(() => (Array.isArray(s.value.fallbacks) ? s.value.fallbacks : []))
const peers = computed<Record<string, any>[]>(() => (Array.isArray(s.value.peers) ? s.value.peers : []))

const seed = (i: number) =>
  computed({
    get: () => (Array.isArray(s.value.testseed) ? (s.value.testseed[i] ?? null) : null),
    set: (v: number | null) => {
      if (!Array.isArray(s.value.testseed)) s.value.testseed = visionSeed()
      s.value.testseed[i] = v ?? 0
    },
  })
const seeds = [seed(0), seed(1), seed(2), seed(3)]

// --- Shadowsocks ----------------------------------------------------------------------

const method = computed({ get: () => String(s.value.method ?? ''), set: (m: string) => setSSMethod(s.value, m) })

// --- WireGuard ------------------------------------------------------------------------

const wgPublic = computed(() => (s.value.secretKey ? wireguardKeys(s.value.secretKey).publicKey : ''))
function newPeerKeys(peer: Record<string, any>) {
  const keys = wireguardKeys()
  peer.privateKey = keys.privateKey
  peer.publicKey = keys.publicKey
}
const newPsk = () => btoa(String.fromCharCode(...crypto.getRandomValues(new Uint8Array(32))))

// --- Accounts and port mapping ---------------------------------------------------------

type Pair = [string, string]

const accounts = computed<Pair[]>({
  get: () => (s.value.accounts ?? []).map((a: Record<string, string>) => [a.user ?? '', a.pass ?? ''] as Pair),
  set: (pairs) => (s.value.accounts = pairs.map(([user, pass]) => ({ user, pass }))),
})
const mixedPassword = computed({
  get: () => s.value.auth === 'password',
  set: (on: boolean) => {
    s.value.auth = on ? 'password' : 'noauth'
    if (on && !Array.isArray(s.value.accounts)) s.value.accounts = []
  },
})
const accountList = ref<{ add: () => void } | null>(null)

/** Port mapping is edited as rows, as the old panel did, so empty or repeated ports stay rows while typed. */
const portMap = headerRows(() => (p.value === 'tunnel' ? props.f.settings : undefined), 'portMap')

// --- MTProto ------------------------------------------------------------------------------

const web = computed(() => {
  s.value.web ??= { enable: false, host: '', path: '', publicIp: '' }
  // Read back through the reactive object: ??= hands back the plain new value.
  return s.value.web
})

/** Turning the WEB proxy on suggests the domain the panel is opened on, as the old panel did. */
const webOn = computed({
  get: () => !!web.value.enable,
  set: (on: boolean) => {
    web.value.enable = on
    const host = location.hostname
    if (on && !web.value.host && host.includes('.') && !/^[0-9.]+$/.test(host) && !host.includes(':')) web.value.host = host
  },
})

/** The panel itself is the WEB proxy's HTTPS front on port 443. */
const webFrontOk = location.protocol === 'https:' && (location.port === '' || location.port === '443')
</script>

<template>
  <template v-if="shown">
    <XPartHead :title="p" mono />

    <!-- VLESS authentication -->
    <template v-if="p === 'vless'">
      <XFieldGrid>
        <XField v-slot="a" label="Authentication">
          <XSelect v-bind="a" v-model="selectedAuth" :options="authOptions" />
        </XField>
      </XFieldGrid>
      <XFieldGrid>
        <XField v-slot="a" label="decryption">
          <XInput v-bind="a" v-model.trim="s.decryption" mono />
        </XField>
        <XField v-slot="a" label="encryption">
          <XInput v-bind="a" v-model.trim="s.encryption" mono />
        </XField>
      </XFieldGrid>
      <XButtonRow>
        <XButton size="sm" variant="primary" icon="download" :loading="fetching" @click="fetchVlessEnc">Get New keys</XButton>
        <XButton size="sm" variant="danger" @click="clearVlessEnc">Clear</XButton>
      </XButtonRow>
    </template>

    <!-- Fallbacks -->
    <template v-if="fallbacks">
      <XListHead title="Fallbacks" add @add="(s.fallbacks ??= []).push(newFallback())">
        <XNestCard v-for="(fb, i) in fallbackList" :key="i" :title="`Fallback ${i + 1}`">
          <template #actions>
            <XInlineButton icon="delete" danger :label="t('xray.removeItem', { n: i + 1 })" @click="s.fallbacks.splice(i, 1)" />
          </template>
          <XFieldGrid :cols="3" :phone="2">
            <XField v-slot="a" label="SNI">
              <XInput v-bind="a" v-model="fb.name" mono />
            </XField>
            <XField v-slot="a" label="ALPN">
              <XInput v-bind="a" v-model="fb.alpn" mono />
            </XField>
            <XField v-slot="a" label="Path">
              <XInput v-bind="a" v-model="fb.path" mono />
            </XField>
            <XField v-slot="a" label="Dest">
              <XInput v-bind="a" v-model="fb.dest" mono />
            </XField>
            <XField v-slot="a" label="xVer">
              <XNumber v-bind="a" v-model="fb.xver" block :min="0" :max="2" />
            </XField>
          </XFieldGrid>
        </XNestCard>
      </XListHead>
    </template>

    <!-- Vision seed -->
    <template v-if="p === 'vless' && canVisionSeed(f)">
      <XListHead title="Vision Seed" />
      <XFieldGrid :cols="4">
        <XField v-for="(v, i) in seeds" :key="i" v-slot="a" :label="`[${i}]`">
          <XNumber v-bind="a" v-model="v.value" block :min="0" :max="9999" />
        </XField>
      </XFieldGrid>
      <XButtonRow>
        <XButton size="sm" variant="primary" @click="s.testseed = randomVisionSeed()">Rand</XButton>
        <XButton size="sm" @click="s.testseed = visionSeed()">Reset</XButton>
      </XButtonRow>
    </template>

    <!-- Shadowsocks -->
    <template v-if="p === 'shadowsocks'">
      <XFieldGrid>
        <XField v-slot="a" :label="t('inbounds.encryption')">
          <XSelect v-bind="a" v-model="method" :options="list(ssMethods)" mono />
        </XField>
        <XField v-if="isSS2022(s.method)" v-slot="a" label="Password" :action="t('inbounds.reset')" @action="s.password = ssPassword(s.method)">
          <XInput v-bind="a" v-model.trim="s.password" mono />
        </XField>
      </XFieldGrid>
      <XFieldGrid>
        <XField v-slot="a" :label="t('inbounds.network')">
          <XSelect v-bind="a" v-model="s.network" :options="networkOptions" mono />
        </XField>
      </XFieldGrid>
    </template>

    <!-- Hysteria -->
    <XFieldGrid v-if="p === 'hysteria'" :cols="3">
      <XField v-slot="a" :label="t('inbounds.version')">
        <XNumber v-bind="a" :model-value="s.version" block disabled />
      </XField>
    </XFieldGrid>

    <!-- WireGuard -->
    <template v-if="p === 'wireguard'">
      <XFieldGrid>
        <XField v-slot="a" :label="t('inbounds.wgSecretKey')" :action="t('inbounds.reset')" @action="s.secretKey = wireguardKeys().privateKey">
          <XInput v-bind="a" v-model.trim="s.secretKey" mono />
        </XField>
        <XField v-slot="a" :label="t('inbounds.publicKey')">
          <XInput v-bind="a" :model-value="wgPublic" mono readonly />
        </XField>
      </XFieldGrid>
      <XFieldGrid>
        <XField v-slot="a" label="MTU">
          <XNumber v-bind="a" v-model="s.mtu" block :min="0" />
        </XField>
        <XSwitch v-model="s.noKernelTun" boxed label="No Kernel Tun" />
      </XFieldGrid>
      <XListHead title="Peers" add @add="s.peers.push(newPeer(s.peers.length))">
        <XNestCard v-for="(peer, i) in peers" :key="i" :title="`Peer ${i + 1}`">
          <template #actions>
            <XInlineButton v-if="s.peers.length > 1" icon="delete" danger :label="t('xray.removeItem', { n: i + 1 })" @click="s.peers.splice(i, 1)" />
          </template>
          <XFieldGrid :cols="1">
            <XFieldGrid>
              <XField v-slot="a" :label="t('inbounds.wgSecretKey')" :action="t('inbounds.reset')" @action="newPeerKeys(peer)">
                <XInput v-bind="a" v-model.trim="peer.privateKey" mono />
              </XField>
              <XField v-slot="a" :label="t('inbounds.publicKey')">
                <XInput v-bind="a" v-model.trim="peer.publicKey" mono />
              </XField>
              <XField v-slot="a" :label="t('inbounds.wgPsk')" :action="t('inbounds.reset')" class="x-wide" @action="peer.preSharedKey = newPsk()">
                <XInput v-bind="a" v-model.trim="peer.preSharedKey" mono />
              </XField>
              <XInputList v-model="peer.allowedIPs" :title="t('inbounds.wgAllowedIPs')" mono class="x-wide" />
            </XFieldGrid>
            <XFieldGrid :cols="3">
              <XField v-slot="a" label="Keep Alive">
                <XNumber v-bind="a" v-model="peer.keepAlive" block :min="0" />
              </XField>
            </XFieldGrid>
          </XFieldGrid>
        </XNestCard>
      </XListHead>
    </template>

    <!-- Mixed -->
    <template v-if="p === 'mixed'">
      <XSwitch v-model="s.udp" boxed :label="`${t('inbounds.enable')} UDP`" />
      <XFieldGrid v-if="s.udp">
        <XField v-slot="a" label="IP">
          <XInput v-bind="a" v-model.trim="s.ip" mono />
        </XField>
      </XFieldGrid>
      <XSwitch v-model="mixedPassword" boxed :label="t('inbounds.password')">
        <template v-if="mixedPassword" #actions>
          <XAddButton :label="t('kit.addRow', { list: t('inbounds.password') })" @click="accountList?.add()" />
        </template>
      </XSwitch>
      <XPairList
        v-if="mixedPassword"
        ref="accountList"
        v-model="accounts"
        headless
        :title="t('inbounds.password')"
        :name-label="t('inbounds.username')"
        :value-label="t('inbounds.password')"
        mono
      />
    </template>

    <!-- HTTP -->
    <template v-if="p === 'http'">
      <XPairList v-model="accounts" :title="t('inbounds.password')" :name-label="t('inbounds.username')" :value-label="t('inbounds.password')" mono />
      <XSwitch v-model="s.allowTransparent" boxed label="Allow Transparent" />
    </template>

    <!-- Tunnel -->
    <template v-if="p === 'tunnel'">
      <XFieldGrid columns="minmax(0, 1fr) 120px" phone-columns="minmax(0, 1fr) 120px">
        <XField v-slot="a" :label="t('inbounds.targetAddress')">
          <XInput v-bind="a" v-model.trim="s.address" mono />
        </XField>
        <XField v-slot="a" :label="t('inbounds.destinationPort')">
          <XNumber v-bind="a" v-model="s.port" block :min="1" :max="65535" />
        </XField>
      </XFieldGrid>
      <XPairList v-model="portMap" :title="t('inbounds.portMap')" :name-label="t('inbounds.colPort')" :value-label="t('inbounds.targetAddress')" mono />
      <XFieldGrid>
        <XField v-slot="a" :label="t('inbounds.network')">
          <XSelect v-bind="a" v-model="s.network" :options="networkOptions" mono />
        </XField>
      </XFieldGrid>
      <XSwitch v-model="s.followRedirect" boxed label="Follow Redirect" />
      <InboundSockopt :f="f" />
    </template>

    <!-- TUN -->
    <template v-if="p === 'tun'">
      <XFieldGrid>
        <XField v-slot="a" label="Interface Name" :help="t('inbounds.tunNameDesc')">
          <XInput v-bind="a" v-model.trim="s.name" mono placeholder="xray0" />
        </XField>
        <XField v-slot="a" :label="t('inbounds.tunUserLevel')" :help="t('inbounds.tunUserLevelDesc')">
          <XNumber v-bind="a" v-model="s.userLevel" block :min="0" placeholder="0" />
        </XField>
        <XField v-slot="a" label="MTU" :help="t('inbounds.tunMtuDesc')">
          <XNumber v-bind="a" v-model="s.mtu" block :min="1" :max="9000" placeholder="1500" />
        </XField>
        <XField v-slot="a" label="Gateway">
          <XTagInput v-bind="a" v-model="s.gateway" mono placeholder="IPv4/IPv6 gateway" :invalid="tunMissing(s).gateway" />
        </XField>
        <XField v-slot="a" label="DNS">
          <XTagInput v-bind="a" v-model="s.dns" mono placeholder="DNS servers" :invalid="tunMissing(s).dns" />
        </XField>
        <XField v-slot="a" label="Auto Routing Table">
          <XTagInput v-bind="a" v-model="s.autoSystemRoutingTable" mono placeholder="e.g. vpn, proxy" :invalid="tunMissing(s).routingTable" />
        </XField>
        <XField v-slot="a" label="Auto Outbounds">
          <XInput v-bind="a" v-model.trim="s.autoOutboundsInterface" mono placeholder="auto" />
        </XField>
        <XField v-slot="a" label="Description" :help="t('inbounds.tunWindowsOnly')">
          <XInput v-bind="a" v-model.trim="s.desc" mono />
        </XField>
        <XField v-slot="a" label="WFP Block Leak" :help="t('inbounds.tunWfpDesc')">
          <XTagInput v-bind="a" :model-value="s.autoSystemWfpBlockLeak ?? []" :groups="wfpLeaks" mono :invalid="wfpLeakWrong(s.autoSystemWfpBlockLeak)" @update:model-value="s.autoSystemWfpBlockLeak = $event" />
        </XField>
      </XFieldGrid>
      <XSwitch v-model="s.autoSystemDnsToGateway" boxed label="DNS To Gateway" :help="t('inbounds.tunDnsGatewayDesc')" />
    </template>

    <!-- MTProto -->
    <template v-if="p === 'mtproto'">
      <XField v-slot="a" :label="t('inbounds.mtprotoDomain')" :help="t('inbounds.mtprotoDomainDesc')">
        <XInput v-bind="a" v-model.trim="s.tlsDomain" mono placeholder="www.example.com" />
      </XField>
      <XSwitch v-model="webOn" boxed :label="t('inbounds.mtprotoWeb')" :help="t('inbounds.mtprotoWebDesc')" />
      <XFieldNote v-if="webOn && !webFrontOk" attached>{{ t('inbounds.mtprotoWebNeedsHttps') }}</XFieldNote>
      <XNest v-if="webOn" bare>
        <XFieldGrid :cols="1">
          <XFieldGrid>
            <XField v-slot="a" :label="t('inbounds.mtprotoWebHost')" :help="t('inbounds.mtprotoWebHostDesc')">
              <XInput v-bind="a" v-model.trim="web.host" mono placeholder="proxy.example.com" />
            </XField>
            <XField v-slot="a" :label="t('inbounds.mtprotoWebIp')" :help="t('inbounds.mtprotoWebIpDesc')">
              <XInput v-bind="a" v-model.trim="web.publicIp" mono :placeholder="t('inbounds.mtprotoWebIpAuto')" />
            </XField>
          </XFieldGrid>
          <XFieldGrid v-if="web.path">
            <XField v-slot="a" :label="t('inbounds.mtprotoWebPath')" :action="t('inbounds.mtprotoWebPathDesc')" @action="web.path = ''">
              <XInput v-bind="a" :model-value="web.path" mono readonly />
            </XField>
          </XFieldGrid>
        </XFieldGrid>
      </XNest>
    </template>
  </template>
</template>
