<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  checkJson, XCodeEditor, XDivider, XField, XFieldGrid, XInput, XInputList, XNest, XNumber, XPairList, XPartHead, XSelect, XSwitch, type Option,
} from '../../kit'
import { canTls, defaultMasquerade, defaultTcpRequest, defaultTcpResponse, network, networks, setNetwork, xhttpModes, type InboundForm } from './inbound'
import XmuxFields from '../xray/XmuxFields.vue'
import XhttpObfsFields from '../xray/XhttpObfsFields.vue'
import InboundMasks from './InboundMasks.vue'
import InboundSockopt from './InboundSockopt.vue'
import { headerRows } from './rows'

/**
 * The transport part of window 11, as the old panel's stream forms had it:
 * the transport and its settings (TCP with HTTP masking, mKCP, WebSocket,
 * gRPC, HTTPUpgrade, XHTTP, Hysteria's own), then Sockopt and the masks.
 */
const props = defineProps<{ f: InboundForm }>()
const { t } = useI18n()

type Json = Record<string, any>
const st = computed<Json>(() => props.f.streamSettings ?? {})
const net = computed(() => network(props.f))
/** The settings of the transport in use, e.g. xhttpSettings. */
const ns = computed<Json>(() => st.value[`${net.value}Settings`] ?? {})
const hysteria = computed(() => props.f.protocol === 'hysteria')

const list = (values: readonly string[]): Option[] => values.map((v) => ({ value: v, label: v }))
const netModel = computed({ get: () => net.value, set: (v: string) => setNetwork(props.f, v) })

// --- TCP ---------------------------------------------------------------------------------

const httpMask = computed({
  get: () => ns.value.header?.type === 'http',
  set: (on: boolean) => {
    const h = (ns.value.header ??= {})
    h.type = on ? 'http' : 'none'
    if (on) {
      h.request = { ...defaultTcpRequest(), ...h.request }
      h.response = { ...defaultTcpResponse(), ...h.response }
    }
  },
})
const req = computed<Json>(() => ns.value.header?.request ?? {})
const res = computed<Json>(() => ns.value.header?.response ?? {})
const reqHeaders = headerRows(() => (httpMask.value ? req.value : undefined), 'headers', true)
const resHeaders = headerRows(() => (httpMask.value ? res.value : undefined), 'headers', true)

// --- WebSocket, HTTPUpgrade, XHTTP ---------------------------------------------------------

const headers = headerRows(() => (['ws', 'httpupgrade', 'xhttp'].includes(net.value) ? ns.value : undefined), 'headers')


/** HTTP/2 Params as JSON text; text that parses to an object goes into the settings. */
const h2On = computed({
  get: () => ns.value.h2Params !== undefined,
  set: (on: boolean) => {
    if (on) ns.value.h2Params = {}
    else delete ns.value.h2Params
  },
})
const h2Text = ref('')
watch(
  () => ns.value.h2Params,
  (v) => {
    if (v === undefined) return
    const same = !checkJson(h2Text.value) && JSON.stringify(JSON.parse(h2Text.value)) === JSON.stringify(v)
    if (!same) h2Text.value = JSON.stringify(v, null, 2)
  },
  { immediate: true },
)
watch(h2Text, (text) => {
  if (ns.value.h2Params === undefined || checkJson(text)) return
  const v = JSON.parse(text)
  if (v && typeof v === 'object' && !Array.isArray(v)) ns.value.h2Params = v
})

// --- Hysteria ------------------------------------------------------------------------------

const masqueradeOn = computed({
  get: () => !!ns.value.masquerade,
  set: (on: boolean) => {
    if (on) ns.value.masquerade = defaultMasquerade()
    else delete ns.value.masquerade
  },
})
const mq = computed<Json>(() => ns.value.masquerade ?? {})
const mqHeaders = headerRows(() => ns.value.masquerade, 'headers')
const masqueradeTypes: Option[] = [
  { value: 'file', label: 'File' },
  { value: 'proxy', label: 'Proxy' },
  { value: 'string', label: 'String' },
]
</script>

<template>
  <XPartHead :title="t('inbounds.transport')" />
  <XFieldGrid v-if="!hysteria">
    <XField v-slot="a" :label="t('common.transportType')"><XSelect v-bind="a" v-model="netModel" :options="networks" /></XField>
  </XFieldGrid>

  <!-- TCP (RAW) -->
  <template v-if="net === 'tcp'">
    <XFieldGrid>
      <XSwitch v-if="canTls(f)" v-model="ns.acceptProxyProtocol" boxed label="Proxy Protocol" />
      <XSwitch v-model="httpMask" boxed :label="`HTTP ${t('inbounds.camouflage')}`" />
    </XFieldGrid>
    <XNest v-if="httpMask" bare>
      <XFieldGrid :cols="1">
        <XDivider :label="t('inbounds.request')" />
        <XFieldGrid>
          <XField v-slot="a" :label="t('inbounds.version')"><XInput v-bind="a" v-model.trim="req.version" mono /></XField>
          <XField v-slot="a" :label="t('inbounds.method')"><XInput v-bind="a" v-model.trim="req.method" mono /></XField>
        </XFieldGrid>
        <XInputList v-model="req.path" :title="t('inbounds.path')" mono />
        <XPairList v-model="reqHeaders" :title="t('inbounds.requestHeader')" :name-label="t('inbounds.headerName')" :value-label="t('inbounds.headerValue')" mono />
        <XDivider :label="t('inbounds.response')" />
        <XFieldGrid :cols="3">
          <XField v-slot="a" :label="t('inbounds.version')"><XInput v-bind="a" v-model.trim="res.version" mono /></XField>
          <XField v-slot="a" :label="t('inbounds.status')"><XInput v-bind="a" v-model.trim="res.status" mono /></XField>
          <XField v-slot="a" :label="t('inbounds.statusDescription')"><XInput v-bind="a" v-model.trim="res.reason" /></XField>
        </XFieldGrid>
        <XPairList v-model="resHeaders" :title="t('inbounds.responseHeader')" :name-label="t('inbounds.headerName')" :value-label="t('inbounds.headerValue')" mono />
      </XFieldGrid>
    </XNest>
  </template>

  <!-- mKCP -->
  <XFieldGrid v-if="net === 'kcp'" :cols="3" :phone="2">
    <XField v-slot="a" label="MTU"><XNumber v-bind="a" v-model="ns.mtu" block :min="576" :max="1460" /></XField>
    <XField v-slot="a" label="TTI (ms)"><XNumber v-bind="a" v-model="ns.tti" block :min="10" :max="100" /></XField>
    <XField v-slot="a" label="Uplink (MB/s)"><XNumber v-bind="a" v-model="ns.uplinkCapacity" block :min="0" /></XField>
    <XField v-slot="a" label="Downlink (MB/s)"><XNumber v-bind="a" v-model="ns.downlinkCapacity" block :min="0" /></XField>
    <XField v-slot="a" label="CWND Multiplier"><XNumber v-bind="a" v-model="ns.cwndMultiplier" block :min="1" /></XField>
    <XField v-slot="a" label="Max Sending Window"><XNumber v-bind="a" v-model="ns.maxSendingWindow" block :min="0" /></XField>
  </XFieldGrid>

  <!-- WebSocket and HTTPUpgrade -->
  <template v-if="net === 'ws' || net === 'httpupgrade'">
    <XSwitch v-model="ns.acceptProxyProtocol" boxed label="Proxy Protocol" />
    <XFieldGrid>
      <XField v-slot="a" :label="t('inbounds.host')"><XInput v-bind="a" v-model.trim="ns.host" mono /></XField>
      <XField v-slot="a" :label="t('inbounds.path')"><XInput v-bind="a" v-model.trim="ns.path" mono /></XField>
      <XField v-if="net === 'ws'" v-slot="a" label="Heartbeat Period"><XNumber v-bind="a" v-model="ns.heartbeatPeriod" block :min="0" /></XField>
    </XFieldGrid>
    <XPairList v-model="headers" :title="t('inbounds.requestHeader')" :name-label="t('inbounds.headerName')" :value-label="t('inbounds.headerValue')" mono />
  </template>

  <!-- gRPC -->
  <template v-if="net === 'grpc'">
    <XFieldGrid>
      <XField v-slot="a" label="Service Name"><XInput v-bind="a" v-model.trim="ns.serviceName" mono /></XField>
      <XField v-slot="a" label="Authority"><XInput v-bind="a" v-model.trim="ns.authority" mono /></XField>
    </XFieldGrid>
    <XSwitch v-model="ns.multiMode" boxed label="Multi Mode" />
    <!-- The server's keepalive pings: 0 leaves them to gRPC. -->
    <XFieldGrid>
      <XField v-slot="a" label="Idle Timeout (s)"><XNumber v-bind="a" block v-model="ns.idle_timeout" :min="0" /></XField>
      <XField v-slot="a" label="Health Check Timeout (s)"><XNumber v-bind="a" block v-model="ns.health_check_timeout" :min="0" /></XField>
    </XFieldGrid>
  </template>

  <!-- XHTTP -->
  <template v-if="net === 'xhttp'">
    <XFieldGrid>
      <XField v-slot="a" :label="t('inbounds.host')"><XInput v-bind="a" v-model.trim="ns.host" mono /></XField>
      <XField v-slot="a" :label="t('inbounds.path')"><XInput v-bind="a" v-model.trim="ns.path" mono /></XField>
    </XFieldGrid>
    <XPairList v-model="headers" :title="t('inbounds.requestHeader')" :name-label="t('inbounds.headerName')" :value-label="t('inbounds.headerValue')" mono />
    <XFieldGrid>
      <XField v-slot="a" label="Mode"><XSelect v-bind="a" v-model="ns.mode" :options="list(xhttpModes)" mono /></XField>
    </XFieldGrid>
    <XFieldGrid v-if="ns.mode === 'packet-up' || ns.mode === 'stream-up'">
      <template v-if="ns.mode === 'packet-up'">
        <XField v-slot="a" label="Max Buffered Upload"><XNumber v-bind="a" v-model="ns.scMaxBufferedPosts" block /></XField>
        <XField v-slot="a" label="Max Upload Size (Byte)"><XInput v-bind="a" v-model.trim="ns.scMaxEachPostBytes" mono /></XField>
      </template>
      <XField v-else v-slot="a" label="Stream-Up Server"><XInput v-bind="a" v-model.trim="ns.scStreamUpServerSecs" mono /></XField>
    </XFieldGrid>
    <XhttpObfsFields :ns="ns">
      <template #method>
        <XField v-slot="a" label="Server Max Header Bytes"><XNumber v-bind="a" v-model="ns.serverMaxHeaderBytes" block :min="0" placeholder="0 (default)" /></XField>
      </template>
    </XhttpObfsFields>
    <XSwitch v-model="ns.noSSEHeader" boxed label="No SSE Header" />
    <XSwitch v-model="h2On" boxed label="HTTP/2 Params" />
    <XNest v-if="h2On" bare>
      <XCodeEditor v-model="h2Text" :height="160" :phone-height="160" fit label="h2Params" />
    </XNest>
    <XDivider :label="t('inbounds.xhttpClients')" />
    <XFieldGrid v-if="ns.mode === 'packet-up' || ns.mode === 'stream-up' || ns.mode === 'stream-one'">
      <XField v-if="ns.mode === 'packet-up'" v-slot="a" label="Min Upload Interval (Ms)"><XInput v-bind="a" v-model="ns.scMinPostsIntervalMs" mono placeholder="30" /></XField>
      <XSwitch v-else v-model="ns.noGRPCHeader" boxed label="No gRPC Header" />
    </XFieldGrid>
    <XmuxFields :ns="ns" />
  </template>

  <!-- Hysteria -->
  <template v-if="net === 'hysteria'">
    <XFieldGrid>
      <XField v-slot="a" label="UDP Idle Timeout"><XNumber v-bind="a" v-model="ns.udpIdleTimeout" block :min="0" /></XField>
    </XFieldGrid>
    <XSwitch v-model="masqueradeOn" boxed label="Masquerade" />
    <XNest v-if="masqueradeOn" bare>
      <XFieldGrid :cols="1">
        <XFieldGrid>
          <XField v-slot="a" label="Type"><XSelect v-bind="a" v-model="mq.type" :options="masqueradeTypes" /></XField>
          <XField v-if="mq.type === 'file'" v-slot="a" label="Dir"><XInput v-bind="a" v-model.trim="mq.dir" mono /></XField>
          <XField v-if="mq.type === 'proxy'" v-slot="a" label="URL"><XInput v-bind="a" v-model.trim="mq.url" mono /></XField>
          <template v-if="mq.type === 'string'">
            <XField v-slot="a" label="Content"><XInput v-bind="a" v-model.trim="mq.content" /></XField>
            <XField v-slot="a" label="Status Code"><XNumber v-bind="a" v-model="mq.statusCode" block /></XField>
          </template>
        </XFieldGrid>
        <XFieldGrid v-if="mq.type === 'proxy'">
          <XSwitch v-model="mq.rewriteHost" boxed label="Rewrite Host" />
          <XSwitch v-model="mq.insecure" boxed label="Insecure" />
        </XFieldGrid>
        <XPairList v-if="mq.type === 'string'" v-model="mqHeaders" :title="t('inbounds.requestHeader')" :name-label="t('inbounds.headerName')" :value-label="t('inbounds.headerValue')" mono />
      </XFieldGrid>
    </XNest>
  </template>

  <InboundSockopt :f="f" />
  <InboundMasks :f="f" />
</template>
