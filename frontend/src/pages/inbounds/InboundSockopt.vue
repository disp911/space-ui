<script setup lang="ts">
import { computed } from 'vue'
import { XField, XFieldGrid, XInput, XNest, XNumber, XSelect, XSwitch, XTagInput, type Option } from '../../kit'
import { domainStrategies, defaultSockopt, tcpCongestions, type InboundForm } from './inbound'
import CustomSockopts from '../xray/CustomSockopts.vue'

/**
 * Sockopt in window 11, as the old panel's form had it: a switch that adds
 * the defaults, then the numbers, the switches and the choices. It lives in
 * streamSettings, also for a tunnel, which has no transport of its own.
 */
const props = defineProps<{ f: InboundForm }>()

const on = computed({
  get: () => !!props.f.streamSettings?.sockopt,
  set: (v: boolean) => {
    const st = (props.f.streamSettings ??= {})
    if (v) st.sockopt = defaultSockopt()
    else delete st.sockopt
  },
})
const so = computed(() => props.f.streamSettings?.sockopt ?? {})

const list = (values: readonly string[]): Option[] => values.map((v) => ({ value: v, label: v }))
const tproxyOptions: Option[] = [
  { value: 'off', label: 'Off' },
  { value: 'redirect', label: 'Redirect' },
  { value: 'tproxy', label: 'TProxy' },
]
const forwarded = [{ options: ['CF-Connecting-IP', 'X-Real-IP', 'True-Client-IP', 'X-Client-IP'].map((v) => ({ value: v, label: v })) }]
</script>

<template>
  <XSwitch v-model="on" boxed label="Sockopt" />
  <XNest v-if="on" bare>
    <XFieldGrid :cols="1">
      <XFieldGrid :cols="3" :phone="2">
        <XField v-slot="a" label="Route Mark">
          <XNumber v-bind="a" v-model="so.mark" block :min="0" />
        </XField>
        <XField v-slot="a" label="TCP Keep Alive Interval">
          <XNumber v-bind="a" v-model="so.tcpKeepAliveInterval" block :min="0" />
        </XField>
        <XField v-slot="a" label="TCP Keep Alive Idle">
          <XNumber v-bind="a" v-model="so.tcpKeepAliveIdle" block :min="0" />
        </XField>
        <XField v-slot="a" label="TCP Max Seg">
          <XNumber v-bind="a" v-model="so.tcpMaxSeg" block :min="0" />
        </XField>
        <XField v-slot="a" label="TCP User Timeout">
          <XNumber v-bind="a" v-model="so.tcpUserTimeout" block :min="0" />
        </XField>
        <XField v-slot="a" label="TCP Window Clamp">
          <XNumber v-bind="a" v-model="so.tcpWindowClamp" block :min="0" />
        </XField>
      </XFieldGrid>
      <XFieldGrid :cols="3">
        <XSwitch v-model="so.acceptProxyProtocol" boxed label="Proxy Protocol" />
        <XSwitch v-model="so.tcpFastOpen" boxed label="TCP Fast Open" />
        <XSwitch v-model="so.tcpMptcp" boxed label="Multipath TCP" />
        <XSwitch v-model="so.penetrate" boxed label="Penetrate" />
        <XSwitch v-model="so.V6Only" boxed label="V6 Only" />
      </XFieldGrid>
      <XFieldGrid :cols="3">
        <XField v-slot="a" label="Domain Strategy">
          <XSelect v-bind="a" v-model="so.domainStrategy" :options="list(domainStrategies)" mono />
        </XField>
        <XField v-slot="a" label="TCP Congestion">
          <XSelect v-bind="a" v-model="so.tcpcongestion" :options="list(tcpCongestions)" mono />
        </XField>
        <XField v-slot="a" label="TProxy">
          <XSelect v-bind="a" v-model="so.tproxy" :options="tproxyOptions" />
        </XField>
        <XField v-slot="a" label="Dialer Proxy">
          <XInput v-bind="a" v-model="so.dialerProxy" mono />
        </XField>
        <XField v-slot="a" label="Interface Name">
          <XInput v-bind="a" v-model="so.interface" mono />
        </XField>
        <XField v-slot="a" label="Trusted X-Forwarded-For" class="x-wide">
          <XTagInput v-bind="a" :model-value="so.trustedXForwardedFor ?? []" :groups="forwarded" mono @update:model-value="so.trustedXForwardedFor = $event" />
        </XField>
      </XFieldGrid>
      <CustomSockopts :so="so" />
    </XFieldGrid>
  </XNest>
</template>
