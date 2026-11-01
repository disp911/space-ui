<script setup lang="ts">
import { computed } from 'vue'
import { XField, XFieldGrid, XInput, XNest, XNumber, XSwitch } from '../../kit'
import { defaultXmux } from './outbound'

/**
 * XMUX of an XHTTP client: how it spreads requests over connections. The
 * outbound window has it, and so has the inbound window for the clients its
 * links and subscriptions set up.
 */
const props = defineProps<{ ns: Record<string, any> }>()

const on = computed({
  get: () => !!props.ns.xmux,
  set: (v: boolean) => (props.ns.xmux = v ? defaultXmux() : undefined),
})
</script>

<template>
  <XSwitch v-model="on" boxed label="XMUX" />
  <XNest v-if="on" bare>
    <XFieldGrid :cols="3" :phone="2">
      <XField v-if="!ns.xmux.maxConnections" v-slot="a" label="Max Concurrency"><XInput v-bind="a" v-model="ns.xmux.maxConcurrency" mono /></XField>
      <XField v-if="!ns.xmux.maxConcurrency" v-slot="a" label="Max Connections"><XInput v-bind="a" v-model="ns.xmux.maxConnections" mono /></XField>
      <XField v-slot="a" label="Max Reuse Times"><XInput v-bind="a" v-model="ns.xmux.cMaxReuseTimes" mono /></XField>
      <XField v-slot="a" label="Max Request Times"><XInput v-bind="a" v-model="ns.xmux.hMaxRequestTimes" mono /></XField>
      <XField v-slot="a" label="Max Reusable Secs"><XInput v-bind="a" v-model="ns.xmux.hMaxReusableSecs" mono /></XField>
      <XField v-slot="a" label="Keep Alive Period"><XNumber v-bind="a" block v-model="ns.xmux.hKeepAlivePeriod" /></XField>
    </XFieldGrid>
  </XNest>
</template>
