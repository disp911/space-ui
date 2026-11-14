<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { XDateTime, XField, XFieldGrid, XFieldNote, XInput, XNumber, XSelect, XSwitch, XTagInput, type Option } from '../../kit'
import { useInbounds } from '../../stores/inbounds'
import { uuid } from '../xray/outbound'
import { flows, mtprotoSecret, randomLowerNum, randomSeq, sourceDenyWrong, ssPassword, vmessSecurities } from './inbound'
import type { Protocol } from './model'
import { useClientTerms } from './terms'

/**
 * A client's fields as the old panel's client form had them, laid out as
 * window 11 of the mockup groups them: who it is, its limits, its options,
 * its traffic and when it ends. The client is changed in place. The client
 * window (12) adds its own parts through slots: the IP log under the limits
 * (an MTProto client's denied addresses follow it),
 * the usage beside the traffic limit and a badge after the end date.
 */
const props = defineProps<{
  client: Record<string, any>
  protocol: Protocol
  /** The inbound's Shadowsocks method, for a new key of the right length. */
  method?: string
  /** VLESS over raw TCP with TLS or Reality: the client may take a Vision flow. */
  flow?: boolean
}>()
const { t } = useI18n()
const store = useInbounds()

const c = computed(() => props.client)

const subscription = computed(() => !!c.value.email && !!store.defaults?.subEnable && props.protocol !== 'mtproto')
const ipLimit = computed(() => !!store.defaults?.ipLimitEnable || props.protocol === 'mtproto')

/** The key a client is known by, its label and how a new one is made. */
const key = computed(() => {
  switch (props.protocol) {
    case 'vmess':
    case 'vless':
      return { field: 'id', label: 'ID', make: () => uuid() }
    case 'mtproto':
      return { field: 'id', label: 'Secret', make: () => mtprotoSecret() }
    case 'trojan':
      return { field: 'password', label: t('inbounds.password'), make: () => randomSeq(10) }
    case 'shadowsocks':
      return { field: 'password', label: t('inbounds.password'), make: () => ssPassword(props.method) }
    case 'hysteria':
      return { field: 'auth', label: 'Auth Password', make: () => randomSeq(10) }
    default:
      return null
  }
})

const securityOptions: Option[] = vmessSecurities.map((s) => ({ value: s, label: s }))
const flowOptions = computed<Option[]>(() => [{ value: '', label: t('inbounds.none') }, ...flows.map((f) => ({ value: f, label: f }))])

const reverseTag = computed({
  get: () => String(c.value.reverse?.tag ?? ''),
  set: (tag: string) => (c.value.reverse = { tag }),
})

const { totalGb, delayed, setDelayed, days, expiry } = useClientTerms(() => props.client)
</script>

<template>
  <div class="client">
    <XSwitch v-model="client.enable" boxed :label="t('inbounds.enable')" />

    <XFieldGrid>
      <XField v-slot="a" :label="t('inbounds.email')" :help="t('inbounds.emailDesc')" :action="t('inbounds.reset')" @action="client.email = randomLowerNum(9)">
        <XInput v-bind="a" v-model.trim="client.email" />
      </XField>
      <XField v-if="key" v-slot="a" :label="key.label" :action="t('inbounds.reset')" @action="client[key.field] = key.make()">
        <XInput v-bind="a" v-model.trim="client[key.field]" mono />
      </XField>
      <XField v-if="protocol === 'vmess'" v-slot="a" :label="t('inbounds.security')">
        <XSelect v-bind="a" v-model="client.security" :options="securityOptions" mono />
      </XField>
      <XField v-if="subscription" v-slot="a" label="Subscription" :help="t('inbounds.subscriptionDesc')" :action="t('inbounds.reset')" @action="client.subId = randomLowerNum(16)">
        <XInput v-bind="a" v-model.trim="client.subId" mono />
      </XField>
      <XField v-if="client.email" v-slot="a" :label="t('inbounds.commentLabel')">
        <XInput v-bind="a" v-model.trim="client.comment" />
      </XField>
    </XFieldGrid>

    <XFieldGrid v-if="ipLimit" :phone="2">
      <XField v-slot="a" :label="t('inbounds.ipLimit')" :help="t('inbounds.ipLimitDesc')">
        <XNumber v-bind="a" v-model="client.limitIp" block :min="0" />
      </XField>
      <template v-if="protocol === 'mtproto'">
        <XField v-slot="a" :label="t('inbounds.rateDown')" :help="t('inbounds.rateDownDesc')">
          <XNumber v-bind="a" v-model="client.rateDown" block :min="0" :max="100000" />
        </XField>
        <XField v-slot="a" :label="t('inbounds.rateUp')" :help="t('inbounds.rateUpDesc')">
          <XNumber v-bind="a" v-model="client.rateUp" block :min="0" :max="100000" />
        </XField>
        <XField v-slot="a" :label="t('inbounds.maxConns')" :help="t('inbounds.maxConnsDesc')">
          <XNumber v-bind="a" v-model="client.maxConns" block :min="0" />
        </XField>
      </template>
    </XFieldGrid>

    <slot name="ips" />

    <template v-if="protocol === 'mtproto'">
      <XField v-slot="a" :label="t('inbounds.sourceDeny')" :help="t('inbounds.sourceDenyDesc')">
        <XTagInput
          v-bind="a"
          :model-value="client.sourceDeny ?? []"
          mono
          :placeholder="t('inbounds.sourceDenyPlaceholder')"
          :invalid="sourceDenyWrong(client.sourceDeny)"
          @update:model-value="client.sourceDeny = $event"
        />
      </XField>
      <XFieldNote tone="info" attached>{{ t('inbounds.sourceDenyRestart') }}</XFieldNote>
    </template>

    <XFieldGrid v-if="flow || protocol === 'vless'">
      <XField v-if="flow" v-slot="a" label="Flow">
        <XSelect v-bind="a" v-model="client.flow" :options="flowOptions" mono />
      </XField>
      <XField v-if="protocol === 'vless'" v-slot="a" :label="t('inbounds.reverseTag')" :help="t('inbounds.reverseTagDesc')">
        <XInput v-bind="a" v-model.trim="reverseTag" :placeholder="t('inbounds.reverseTagPlaceholder')" />
      </XField>
    </XFieldGrid>

    <XFieldGrid>
      <XField v-slot="a" :label="t('inbounds.totalFlow')" :help="t('inbounds.totalHelp')">
        <XNumber v-bind="a" v-model="totalGb" block decimal :min="0" :unit="t('inbounds.gb')" />
      </XField>
      <slot name="usage" />
    </XFieldGrid>

    <XSwitch :model-value="delayed" boxed :label="t('inbounds.delayedStart')" @update:model-value="setDelayed" />
    <XFieldGrid>
      <XField v-if="delayed" v-slot="a" :label="t('inbounds.expireDays')">
        <XNumber v-bind="a" v-model="days" block :min="0" />
      </XField>
      <XField v-else :label="t('inbounds.colExpiry')" :help="t('inbounds.expiryHelp')">
        <template v-if="$slots.expiry" #badge><slot name="expiry" /></template>
        <template #default="a"><XDateTime v-bind="a" v-model="expiry" /></template>
      </XField>
      <XField v-if="client.expiryTime" v-slot="a" :label="t('inbounds.renew')" :help="t('inbounds.renewDesc')">
        <XNumber v-bind="a" v-model="client.reset" block :min="0" />
      </XField>
    </XFieldGrid>
  </div>
</template>

<style scoped>
.client {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
