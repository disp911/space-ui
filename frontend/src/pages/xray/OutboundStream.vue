<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  XDivider, XField, XFieldGrid, XInlineButton, XInput, XListHead, XNestCard, XNumber, XSection, XSelect, XSwitch, XTagInput, type Option,
} from '../../kit'
import type { Outbound } from './model'
import { canFlow, defaultNetwork, defaultQuic, items, networks, user, xhttpModes } from './outbound'
import OutboundPackets from './OutboundPackets.vue'
import QuicParams from './QuicParams.vue'
import XmuxFields from './XmuxFields.vue'
import XhttpObfsFields from './XhttpObfsFields.vue'

/**
 * The transport of the outbound window: the network first in the card, its
 * settings, and the masks over it. masksOnly is freedom's: no network, only
 * the masks Xray 26 put in place of its fragment and noises, Fragment over
 * TCP and Noise over UDP.
 */
const props = defineProps<{ o: Outbound; masksOnly?: boolean }>()
const { t } = useI18n()
const open = ref(true)

const st = computed(() => {
  props.o.streamSettings ??= props.masksOnly ? {} : { network: 'tcp', security: 'none' }
  // Read back through the reactive object: ??= hands back the plain new value.
  return props.o.streamSettings
})
const net = computed({
  get: () => st.value.network ?? 'tcp',
  set: (n: string) => {
    st.value.network = n
    st.value[`${n}Settings`] ??= defaultNetwork(n)
    // A VLESS flow lives only over raw TCP: off another network it goes, as on the old panel.
    if (props.o.protocol === 'vless' && !canFlow(props.o)) user(props.o).flow = ''
  },
})
/** The settings of the network in use, made when missing. */
const ns = computed(() => {
  st.value[`${net.value}Settings`] ??= defaultNetwork(net.value)
  // Read back through the reactive object: ??= hands back the plain new value.
  return st.value[`${net.value}Settings`]
})

const netOptions = computed<Option[]>(() => [
  ...networks,
  ...(props.o.protocol === 'hysteria' ? [{ value: 'hysteria', label: 'Hysteria2' }] : []),
])
const list = (values: readonly string[]): Option[] => values.map((v) => ({ value: v, label: v }))

// --- TCP: HTTP camouflage -------------------------------------------------------------

const httpOn = computed({
  get: () => ns.value.header?.type === 'http',
  set: (on: boolean) => (ns.value.header = on ? { type: 'http', request: { headers: { Host: [] }, path: ['/'] } } : { type: 'none' }),
})
const headerList = (get: () => string[] | undefined, put: (v: string[]) => void) =>
  computed({ get: () => (get() ?? []).join(','), set: (v: string) => put(v.split(',').map((x) => x.trim()).filter(Boolean)) })
const tcpHost = headerList(
  () => ns.value.header?.request?.headers?.Host,
  (v) => (((ns.value.header.request ??= {}).headers ??= {}).Host = v),
)
const tcpPath = headerList(
  () => ns.value.header?.request?.path,
  (v) => ((ns.value.header.request ??= {}).path = v.length ? v : ['/']),
)

// --- Masks ------------------------------------------------------------------------------

const fm = computed(() => {
  st.value.finalmask ??= {}
  // Read back through the reactive object: ??= hands back the plain new value.
  return st.value.finalmask
})
const tcpMasks = computed(() => props.masksOnly || ['tcp', 'raw', 'httpupgrade', 'ws', 'grpc', 'xhttp'].includes(net.value))
const udpMasks = computed(() => props.masksOnly || net.value === 'kcp' || props.o.protocol === 'hysteria')
const quic = computed(() => !props.masksOnly && (net.value === 'xhttp' || props.o.protocol === 'hysteria'))

const packetItem = () => ({ delay: 0, rand: 0, randRange: '0-255', type: 'array' })

function tcpMaskSettings(type: string) {
  if (type === 'fragment') return { packets: 'tlshello', length: '100-200', delay: '', maxSplit: '' }
  if (type === 'sudoku') return { password: '', ascii: '', customTable: '', customTables: [], paddingMin: 0, paddingMax: 0 }
  return { clients: [], servers: [] }
}

function udpMaskSettings(type: string) {
  switch (type) {
    case 'salamander':
      return { password: '' }
    case 'mkcp-legacy':
      return { header: '', value: '' }
    case 'xdns':
      return { domains: [{ names: [] }], resolvers: [{ addrs: [] }] }
    case 'xicmp':
      return { dgram: false, ips: [] }
    case 'header-custom':
      return { client: [], server: [] }
    case 'noise':
      return { reset: 0, noise: [] }
    case 'udphop':
      return { mode: 'intervalRemote', remotePorts: '', interval: '5-10' }
    default:
      return {}
  }
}

/** The core takes a UDP hop only as the last UDP mask, the one at the socket. */
function keepHopLast() {
  const udp: Record<string, any>[] = fm.value.udp ?? []
  const hops = udp.filter((m) => m.type === 'udphop')
  if (hops.length) fm.value.udp = [...udp.filter((m) => m.type !== 'udphop'), ...hops]
}

function setTcpType(m: Record<string, any>, type: string) {
  m.type = type
  m.settings = tcpMaskSettings(type)
}

function setUdpType(m: Record<string, any>, type: string) {
  m.type = type
  m.settings = udpMaskSettings(type)
  // xDNS rides in small packets: mKCP's MTU follows it, as on the old panel.
  if (net.value === 'kcp') ns.value.mtu = type === 'xdns' ? 900 : 1350
  keepHopLast()
}

function addUdpMask() {
  const type = props.masksOnly ? 'noise' : props.o.protocol === 'hysteria' ? 'salamander' : 'mkcp-legacy'
  ;(fm.value.udp ??= []).push({ type, settings: udpMaskSettings(type) })
  keepHopLast()
}

const tcpTypes = computed<Option[]>(() => [
  { value: 'fragment', label: 'Fragment' },
  ...(props.masksOnly
    ? []
    : [
        { value: 'header-custom', label: 'Header Custom' },
        { value: 'sudoku', label: 'Sudoku' },
      ]),
])
const udpTypes = computed<Option[]>(() =>
  props.masksOnly
    ? [{ value: 'noise', label: 'Noise' }]
    : props.o.protocol === 'hysteria'
    ? [
        { value: 'salamander', label: 'Salamander (Hysteria2)' },
        { value: 'udphop', label: 'UDP Hop' },
      ]
    : [
        { value: 'mkcp-legacy', label: 'mKCP' },
        { value: 'xdns', label: 'xDNS' },
        { value: 'xicmp', label: 'xICMP' },
        { value: 'header-custom', label: 'Header Custom' },
        { value: 'noise', label: 'Noise' },
      ],
)
const kcpHeaders: Option[] = [
  { value: '', label: 'None' },
  { value: 'dns', label: 'DNS' },
  { value: 'dtls', label: 'DTLS 1.2' },
  { value: 'srtp', label: 'SRTP' },
  { value: 'utp', label: 'uTP' },
  { value: 'wechat', label: 'WeChat Video' },
  { value: 'wireguard', label: 'WireGuard' },
]

// QUIC parameters: present or not, the card's switch says.
const quicOn = computed({
  get: () => !!fm.value.quicParams,
  set: (on: boolean) => (fm.value.quicParams = on ? defaultQuic() : undefined),
})
const qp = computed(() => fm.value.quicParams ?? {})
const quicOpen = ref(true)

/** Sudoku's custom tables are one text in the field, a list in the JSON. */
const tablesText = (m: Record<string, any>) => (Array.isArray(m.settings.customTables) ? m.settings.customTables.join(',') : '')
const setTables = (m: Record<string, any>, v: string) => (m.settings.customTables = v.split(',').map((x) => x.trim()).filter(Boolean))

const remove = (list: unknown[], i: number) => list.splice(i, 1)

// In the script: vue-tsc misreads the // of a URL inside the template.
const resolverHint = computed(() => t('xray.hintExample', { v: '8.8.8.8:53' }))
</script>

<template>
  <XSection v-model:open="open" :title="masksOnly ? 'Final Mask' : t('xray.transport')" card>
    <template v-if="!masksOnly">
    <XFieldGrid :cols="2">
      <XField v-slot="a" :label="t('common.transportType')"><XSelect v-bind="a" v-model="net" :options="netOptions" mono /></XField>
    </XFieldGrid>

    <!-- TCP -->
    <template v-if="net === 'tcp'">
      <XFieldGrid :cols="2">
        <XSwitch v-model="httpOn" boxed :label="t('xray.httpCamouflage')" />
      </XFieldGrid>
      <XFieldGrid :cols="2" v-if="httpOn">
        <XField v-slot="a" :label="t('xray.host')">
          <XInput v-bind="a" v-model="tcpHost" mono />
        </XField>
        <XField v-slot="a" :label="t('xray.path')">
          <XInput v-bind="a" v-model="tcpPath" mono />
        </XField>
      </XFieldGrid>
    </template>

    <!-- mKCP -->
    <XFieldGrid :cols="3" v-if="net === 'kcp'">
      <XField v-slot="a" label="MTU"><XNumber v-bind="a" block v-model="ns.mtu" :min="0" /></XField>
      <XField v-slot="a" label="TTI (ms)"><XNumber v-bind="a" block v-model="ns.tti" :min="0" /></XField>
      <XField v-slot="a" label="Uplink (MB/s)"><XNumber v-bind="a" block v-model="ns.uplinkCapacity" :min="0" /></XField>
      <XField v-slot="a" label="Downlink (MB/s)"><XNumber v-bind="a" block v-model="ns.downlinkCapacity" :min="0" /></XField>
      <XField v-slot="a" label="CWND Multiplier"><XNumber v-bind="a" block v-model="ns.cwndMultiplier" :min="1" /></XField>
      <XField v-slot="a" label="Max Sending Window"><XNumber v-bind="a" block v-model="ns.maxSendingWindow" :min="0" /></XField>
    </XFieldGrid>

    <!-- WebSocket, HTTPUpgrade, XHTTP -->
    <XFieldGrid :cols="2" v-if="net === 'ws' || net === 'httpupgrade' || net === 'xhttp'">
      <XField v-slot="a" :label="t('xray.host')">
        <XInput v-bind="a" v-model="ns.host" mono />
      </XField>
      <XField v-slot="a" :label="t('xray.path')">
        <XInput v-bind="a" v-model="ns.path" mono />
      </XField>
    </XFieldGrid>
    <XFieldGrid :cols="3" v-if="net === 'ws'">
      <XField v-slot="a" label="Heartbeat Period"><XNumber v-bind="a" block v-model="ns.heartbeatPeriod" :min="0" /></XField>
    </XFieldGrid>

    <!-- gRPC -->
    <template v-if="net === 'grpc'">
      <XFieldGrid :cols="2">
        <XField v-slot="a" label="Service Name"><XInput v-bind="a" v-model="ns.serviceName" mono /></XField>
        <XField v-slot="a" label="Authority"><XInput v-bind="a" v-model="ns.authority" mono /></XField>
        <XSwitch v-model="ns.multiMode" boxed label="Multi Mode" />
        <!-- Keepalive pings and the HTTP/2 client's own: 0 or empty leaves them to gRPC. -->
        <XField v-slot="a" label="Idle Timeout (s)"><XNumber v-bind="a" block v-model="ns.idle_timeout" :min="0" /></XField>
        <XField v-slot="a" label="Health Check Timeout (s)"><XNumber v-bind="a" block v-model="ns.health_check_timeout" :min="0" /></XField>
        <XSwitch v-model="ns.permit_without_stream" boxed label="Permit Without Stream" />
        <XField v-slot="a" label="Initial Windows Size"><XNumber v-bind="a" block v-model="ns.initial_windows_size" :min="0" /></XField>
        <XField v-slot="a" label="User Agent"><XInput v-bind="a" v-model.trim="ns.user_agent" mono /></XField>
      </XFieldGrid>
    </template>

    <!-- XHTTP -->
    <template v-if="net === 'xhttp'">
      <XFieldGrid :cols="2">
        <XField v-slot="a" label="Mode">
          <XSelect v-bind="a" v-model="ns.mode" :options="list(xhttpModes)" mono />
        </XField>
        <XSwitch v-if="ns.mode === 'stream-up' || ns.mode === 'stream-one'" v-model="ns.noGRPCHeader" boxed label="No gRPC Header" />
        <XField v-if="ns.mode === 'packet-up'" v-slot="a" label="Min Upload Interval (Ms)">
          <XInput v-bind="a" v-model="ns.scMinPostsIntervalMs" mono />
        </XField>
        <XField v-if="ns.mode === 'packet-up'" v-slot="a" label="Max Upload Size (Byte)">
          <XInput v-bind="a" v-model.trim="ns.scMaxEachPostBytes" mono />
        </XField>
      </XFieldGrid>
      <!-- As the server has them: a server that hides its padding or ids takes no other client. -->
      <XhttpObfsFields :ns="ns" />
      <XmuxFields :ns="ns" />
    </template>

    <!-- Hysteria -->
    <template v-if="net === 'hysteria'">
      <XFieldGrid :cols="2">
        <XField v-slot="a" label="Auth Password"><XInput v-bind="a" v-model="ns.auth" mono /></XField>
      </XFieldGrid>
    </template>

    <XDivider v-if="tcpMasks || udpMasks || quic" />
    </template>

    <!-- TCP masks -->
    <template v-if="tcpMasks">
      <XListHead title="TCP Masks" add @add="(fm.tcp ??= []).push({ type: 'fragment', settings: tcpMaskSettings('fragment') })">
        <XNestCard v-for="(m, i) in items(fm.tcp)" :key="i" :title="String(i + 1)">
          <template #actions>
            <XInlineButton icon="delete" danger :label="t('xray.removeItem', { n: i + 1 })" @click="remove(fm.tcp, i)" />
          </template>
          <XFieldGrid :cols="1">
            <XFieldGrid :cols="2">
              <XField v-slot="a" label="Type">
                <XSelect v-bind="a" :model-value="m.type" :options="tcpTypes" @update:model-value="setTcpType(m, String($event))" />
              </XField>
            </XFieldGrid>
            <XFieldGrid :cols="4" v-if="m.type === 'fragment'">
              <XField v-slot="a" label="Packets"><XSelect v-bind="a" v-model="m.settings.packets" :options="list(['tlshello', '1-3', '1-5'])" mono /></XField>
              <XField v-slot="a" label="Length"><XInput v-bind="a" v-model="m.settings.length" mono :placeholder="t('xray.hintExample', { v: '100-200' })" /></XField>
              <XField v-slot="a" label="Delay"><XInput v-bind="a" v-model="m.settings.delay" mono :placeholder="t('xray.hintExample', { v: '10-20' })" /></XField>
              <XField v-slot="a" label="Max Split"><XInput v-bind="a" v-model="m.settings.maxSplit" mono :placeholder="t('xray.hintExample', { v: '3-6' })" /></XField>
            </XFieldGrid>
            <XFieldGrid :cols="3" v-if="m.type === 'sudoku'">
              <XField v-slot="a" label="Password"><XInput v-bind="a" v-model="m.settings.password" mono :placeholder="t('xray.hintObfs')" /></XField>
              <XField v-slot="a" label="ASCII"><XInput v-bind="a" v-model="m.settings.ascii" mono /></XField>
              <XField v-slot="a" label="Custom Table"><XInput v-bind="a" v-model="m.settings.customTable" mono /></XField>
              <XField v-slot="a" label="Custom Tables">
                <XInput v-bind="a" :model-value="tablesText(m)" mono @update:model-value="setTables(m, $event)" />
              </XField>
              <XField v-slot="a" label="Padding Min"><XNumber v-bind="a" block v-model="m.settings.paddingMin" :min="0" /></XField>
              <XField v-slot="a" label="Padding Max"><XNumber v-bind="a" block v-model="m.settings.paddingMax" :min="0" /></XField>
            </XFieldGrid>
            <template v-if="m.type === 'header-custom'">
              <template v-for="side in (['clients', 'servers'] as const)" :key="side">
                <XListHead :title="side === 'clients' ? 'Clients' : 'Servers'" add @add="(m.settings[side] ??= []).push([packetItem()])">
                  <XNestCard v-for="(group, gi) in items(m.settings[side])" :key="gi" :title="`${side === 'clients' ? 'Clients' : 'Servers'} Group ${gi + 1}`">
                    <template #actions>
                      <XInlineButton icon="delete" danger :label="t('xray.removeItem', { n: gi + 1 })" @click="remove(m.settings[side], gi)" />
                    </template>
                    <OutboundPackets :items="items(group)" with-delay />
                  </XNestCard>
                </XListHead>
              </template>
            </template>
          </XFieldGrid>
        </XNestCard>
      </XListHead>
    </template>

    <!-- UDP masks -->
    <template v-if="udpMasks">
      <XListHead title="UDP Masks" add @add="addUdpMask">
        <XNestCard v-for="(m, i) in items(fm.udp)" :key="i" :title="String(i + 1)">
          <template #actions>
            <XInlineButton icon="delete" danger :label="t('xray.removeItem', { n: i + 1 })" @click="remove(fm.udp, i)" />
          </template>
          <XFieldGrid :cols="1">
            <XFieldGrid :cols="2">
              <XField v-slot="a" label="Type">
                <XSelect v-bind="a" :model-value="m.type" :options="udpTypes" @update:model-value="setUdpType(m, String($event))" />
              </XField>
              <XField v-if="m.type === 'salamander'" v-slot="a" label="Password">
                <XInput v-bind="a" v-model="m.settings.password" mono :placeholder="t('xray.hintObfs')" />
              </XField>
              <template v-if="m.type === 'mkcp-legacy'">
                <XField v-slot="a" label="Header"><XSelect v-bind="a" v-model="m.settings.header" :options="kcpHeaders" /></XField>
                <XField v-if="!m.settings.header" v-slot="a" label="Password">
                  <XInput v-bind="a" v-model="m.settings.value" mono :placeholder="t('xray.hintMkcp')" />
                </XField>
                <XField v-if="m.settings.header === 'dns'" v-slot="a" label="Domain">
                  <XInput v-bind="a" v-model="m.settings.value" mono :placeholder="t('xray.hintExample', { v: 'www.example.com' })" />
                </XField>
              </template>
              <template v-if="m.type === 'xdns'">
                <XField v-slot="a" label="Domains" class="x-wide">
                  <XTagInput v-bind="a" v-model="m.settings.domains[0].names" mono :placeholder="t('xray.hintExample', { v: 'www.example.com' })" />
                </XField>
                <XField v-slot="a" label="Resolvers" class="x-wide">
                  <XTagInput v-bind="a" v-model="m.settings.resolvers[0].addrs" mono :placeholder="resolverHint" />
                </XField>
              </template>
              <template v-if="m.type === 'xicmp'">
                <XField v-slot="a" label="IPs"><XTagInput v-bind="a" v-model="m.settings.ips" mono :placeholder="t('xray.hintExample', { v: '1.2.3.4' })" /></XField>
                <XSwitch v-model="m.settings.dgram" boxed label="DGRAM" />
              </template>
              <XField v-if="m.type === 'noise'" v-slot="a" label="Reset"><XNumber v-bind="a" block v-model="m.settings.reset" :min="0" /></XField>
              <template v-if="m.type === 'udphop'">
                <XField v-slot="a" label="Ports"><XInput v-bind="a" v-model="m.settings.remotePorts" mono :placeholder="t('xray.hintPortHop')" /></XField>
                <XField v-slot="a" label="Interval (s)"><XInput v-bind="a" v-model="m.settings.interval" mono /></XField>
              </template>
            </XFieldGrid>
            <template v-if="m.type === 'noise'">
              <XListHead title="Noise" add @add="m.settings.noise.push({ ...packetItem(), delay: '' })">
                <OutboundPackets :items="m.settings.noise" with-delay noise removable />
              </XListHead>
            </template>
            <template v-if="m.type === 'header-custom'">
              <XListHead title="Client" add @add="m.settings.client.push(packetItem())">
                <OutboundPackets :items="m.settings.client" removable />
              </XListHead>
              <XListHead title="Server" add @add="m.settings.server.push(packetItem())">
                <OutboundPackets :items="m.settings.server" removable />
              </XListHead>
            </template>
          </XFieldGrid>
        </XNestCard>
      </XListHead>
    </template>

    <!-- QUIC parameters -->
    <XSection v-if="quic" v-model:open="quicOpen" title="QUIC Params" card mono>
      <XFieldGrid :cols="2">
        <XSwitch v-model="quicOn" boxed :label="t('inbounds.enable')" />
      </XFieldGrid>
      <QuicParams v-if="quicOn" :qp="qp" />
    </XSection>
  </XSection>
</template>
