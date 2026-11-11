<script setup lang="ts">
import { computed } from 'vue'
import { XField, XFieldGrid, XInput, XNest, XNumber, XSelect, XSwitch, XTagInput, type Option } from '../../kit'
import type { Outbound } from './model'
import CustomSockopts from './CustomSockopts.vue'
import { addressPortStrategies, canMux, defaultMux, defaultSockopt, domainStrategies, forwardedForHeaders, tcpCongestions } from './outbound'

/** Socket options and Mux at the foot of the outbound window: a switch each, their fields under it when on. */
const props = defineProps<{ o: Outbound; tags: string[] }>()

const sockoptOn = computed({
  get: () => !!props.o.streamSettings?.sockopt,
  set: (on: boolean) => {
    if (on) (props.o.streamSettings ??= {}).sockopt = defaultSockopt()
    else if (props.o.streamSettings) delete props.o.streamSettings.sockopt
  },
})
const so = computed(() => props.o.streamSettings?.sockopt ?? {})

/** Happy Eyeballs: on, it races the server's IPv4 and IPv6 addresses; off leaves it out. */
const eyeballsOn = computed({
  get: () => !!so.value.happyEyeballs,
  set: (on: boolean) => {
    if (on) so.value.happyEyeballs = { prioritizeIPv6: false, tryDelayMs: 250, interleave: 1, maxConcurrentTry: 4 }
    else delete so.value.happyEyeballs
  },
})

const muxOn = computed({
  get: () => !!props.o.mux?.enabled,
  set: (on: boolean) => (props.o.mux = on ? { ...defaultMux(), ...(props.o.mux ?? {}), enabled: true } : undefined),
})

const list = (values: readonly string[]): Option[] => values.map((v) => ({ value: v, label: v }))
const dialers = computed<Option[]>(() => [{ value: '', label: 'Empty', plain: true }, ...list(props.tags)])
const headers = [{ options: forwardedForHeaders.map((h) => ({ value: h, label: h })) }]
/** None leaves the system's own, as Xray does. */
const congestions: Option[] = [{ value: '', label: 'Empty', plain: true }, ...list(tcpCongestions)]
</script>

<template>
  <XFieldGrid :cols="2">
    <XSwitch v-model="sockoptOn" boxed label="Sockopts" />
    <XSwitch v-if="canMux(o)" v-model="muxOn" boxed label="Mux" />
  </XFieldGrid>

  <XNest v-if="sockoptOn" bare>
    <XFieldGrid :cols="1">
      <XFieldGrid :cols="3">
        <XField v-slot="a" label="Dialer Proxy"><XSelect v-bind="a" :model-value="so.dialerProxy ?? ''" @update:model-value="so.dialerProxy = String($event)" :options="dialers" mono /></XField>
        <!-- How the server's domain is resolved, e.g. WireGuard's endpoint to IPv4 or IPv6. -->
        <XField v-slot="a" label="Domain Strategy"><XSelect v-bind="a" :model-value="so.domainStrategy ?? 'AsIs'" @update:model-value="so.domainStrategy = String($event)" :options="list(domainStrategies)" mono /></XField>
        <XField v-slot="a" label="Address Port Strategy"><XSelect v-bind="a" v-model="so.addressPortStrategy" :options="list(addressPortStrategies)" mono /></XField>
        <XField v-slot="a" label="Keep Alive Interval"><XNumber v-bind="a" block v-model="so.tcpKeepAliveInterval" :min="0" /></XField>
        <XField v-slot="a" label="Keep Alive Idle"><XNumber v-bind="a" block v-model="so.tcpKeepAliveIdle" :min="0" /></XField>
        <XField v-slot="a" label="Route Mark"><XNumber v-bind="a" block v-model="so.mark" :min="0" /></XField>
        <XField v-slot="a" label="TCP Max Seg"><XNumber v-bind="a" block v-model="so.tcpMaxSeg" :min="0" /></XField>
        <XField v-slot="a" label="TCP User Timeout"><XNumber v-bind="a" block v-model="so.tcpUserTimeout" :min="0" /></XField>
        <XField v-slot="a" label="TCP Window Clamp"><XNumber v-bind="a" block v-model="so.tcpWindowClamp" :min="0" /></XField>
        <XField v-slot="a" label="TCP Congestion"><XSelect v-bind="a" :model-value="so.tcpCongestion ?? ''" @update:model-value="so.tcpCongestion = String($event)" :options="congestions" mono /></XField>
        <!-- The network interface the outbound goes out by, e.g. wg0 or eth1. -->
        <XField v-slot="a" label="Interface Name"><XInput v-bind="a" v-model.trim="so.interface" mono /></XField>
        <XSwitch v-model="so.tcpFastOpen" boxed label="TCP Fast Open" />
        <XSwitch v-model="so.tcpMptcp" boxed label="Multipath TCP" />
        <XSwitch v-model="so.penetrate" boxed label="Penetrate" />
      </XFieldGrid>
      <XField v-slot="a" label="Trusted X-Forwarded-For">
        <XTagInput v-bind="a" v-model="so.trustedXForwardedFor" :groups="headers" mono />
      </XField>
      <XSwitch v-model="eyeballsOn" boxed label="Happy Eyeballs" />
      <XNest v-if="eyeballsOn" bare>
        <XFieldGrid :cols="3">
          <XSwitch v-model="so.happyEyeballs.prioritizeIPv6" boxed label="Prioritize IPv6" />
          <XField v-slot="a" label="Try Delay (ms)"><XNumber v-bind="a" block v-model="so.happyEyeballs.tryDelayMs" :min="0" /></XField>
          <XField v-slot="a" label="Interleave"><XNumber v-bind="a" block v-model="so.happyEyeballs.interleave" :min="1" /></XField>
          <XField v-slot="a" label="Max Concurrent Try"><XNumber v-bind="a" block v-model="so.happyEyeballs.maxConcurrentTry" :min="1" /></XField>
        </XFieldGrid>
      </XNest>
      <CustomSockopts :so="so" />
    </XFieldGrid>
  </XNest>

  <XNest v-if="muxOn && canMux(o)" bare>
    <XFieldGrid :cols="3">
      <XField v-slot="a" label="Concurrency"><XNumber v-bind="a" block v-model="o.mux.concurrency" :min="-1" :max="1024" /></XField>
      <XField v-slot="a" label="xudp Concurrency"><XNumber v-bind="a" block v-model="o.mux.xudpConcurrency" :min="-1" :max="1024" /></XField>
      <XField v-slot="a" label="xudp UDP 443"><XSelect v-bind="a" v-model="o.mux.xudpProxyUDP443" :options="list(['reject', 'allow', 'skip'])" mono /></XField>
    </XFieldGrid>
  </XNest>
</template>
