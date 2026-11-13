<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../api'
import { toast, XButton, XDateTime, XField, XFieldGrid, XInput, XModal, XNumber, XSelect, XSwitch, type Option } from '../../kit'
import { useInbounds } from '../../stores/inbounds'
import { cause } from '../settings/form'
import { canFlow, finalClient, flows, fromDb, newClient, randomLowerNum, vmessSecurities } from './inbound'
import type { DbInbound } from './model'
import { useClientTerms } from './terms'

/**
 * Window 13 of the mockup: many clients of an inbound at once, as the old
 * panel's bulk form made them. Their emails come from the method chosen:
 * random, random with a prefix, or numbered between a first and a last with
 * a prefix and a postfix. An empty subscription gives each its own.
 */
const props = defineProps<{ inbound: DbInbound | null }>()
const open = defineModel<boolean>('open', { default: false })
const { t } = useI18n()
const store = useInbounds()

type Json = Record<string, any>

const form = computed(() => (props.inbound ? fromDb(props.inbound) : null))

/** What every new client gets, besides its email and key. */
const base = ref<Json>({})
const method = ref(0)
const quantity = ref<number | null>(1)
const first = ref<number | null>(1)
const last = ref<number | null>(1)
const prefix = ref('')
const postfix = ref('')
const saving = ref(false)

watch(open, (on) => {
  if (!on) return
  base.value = { subId: '', security: 'auto', flow: '', limitIp: 0, totalGB: 0, expiryTime: 0, reset: 0 }
  method.value = 0
  quantity.value = 1
  first.value = 1
  last.value = 1
  prefix.value = ''
  postfix.value = ''
})

const { totalGb, delayed, setDelayed, days, expiry } = useClientTerms(() => base.value)

const methodOptions: Option<number>[] = [
  { value: 0, label: 'Random' },
  { value: 1, label: 'Random+Prefix' },
  { value: 2, label: 'Random+Prefix+Num' },
  { value: 3, label: 'Random+Prefix+Num+Postfix' },
  { value: 4, label: 'Prefix+Num+Postfix' },
]
const securityOptions: Option[] = vmessSecurities.map((s) => ({ value: s, label: s }))
const flowOptions = computed<Option[]>(() => [{ value: '', label: t('inbounds.none') }, ...flows.map((f) => ({ value: f, label: f }))])

const flowOk = computed(() => !!form.value && canFlow(form.value))
const subscription = computed(() => !!store.defaults?.subEnable && props.inbound?.protocol !== 'mtproto')
const ipLimit = computed(() => !!store.defaults?.ipLimitEnable || props.inbound?.protocol === 'mtproto')

/** The emails the method gives, as the old panel built them. */
function emails(): string[] {
  const m = method.value
  const [start, end] = m > 1 ? [first.value ?? 1, (last.value ?? 1) + 1] : [0, quantity.value ?? 1]
  const pre = m > 0 ? prefix.value : ''
  const post = m > 2 ? postfix.value : ''
  const out: string[] = []
  for (let i = start; i < end; i++) out.push((m === 4 ? '' : randomLowerNum(8)) + (m > 1 ? `${pre}${i}${post}` : `${pre}${post}`))
  return out
}

const valid = computed(() => (method.value > 1 ? (first.value ?? 0) >= 1 && (last.value ?? 0) >= (first.value ?? 0) : (quantity.value ?? 0) >= 1))

async function save() {
  if (!props.inbound || !form.value || !valid.value || saving.value) return
  saving.value = true
  const b = base.value
  const clients = emails().map((email) => {
    const c = newClient(props.inbound!.protocol, form.value!.settings.method)
    Object.assign(c, { email, limitIp: b.limitIp ?? 0, totalGB: b.totalGB, expiryTime: b.expiryTime, reset: b.reset ?? 0 })
    if (b.subId) c.subId = b.subId
    if (props.inbound!.protocol === 'vmess') c.security = b.security
    if (flowOk.value) c.flow = b.flow
    return finalClient(form.value!, c)
  })
  try {
    await api.post('panel/api/inbounds/addClient', { id: props.inbound.id, settings: JSON.stringify({ clients }) })
    open.value = false
    await store.load().catch(() => {})
  } catch (e) {
    toast('err', cause(e))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <XModal v-model:open="open" :title="`${t('inbounds.addBulk')} ${inbound?.remark ?? ''}`" size="md" cancel scroll-body stack :dismissible="!saving">
    <div v-if="inbound" class="bulk">
      <XField v-slot="a" :label="t('inbounds.method')">
        <XSelect v-bind="a" v-model="method" :options="methodOptions" />
      </XField>

      <XFieldGrid v-if="method < 2">
        <XField v-slot="a" :label="t('inbounds.clientCount')">
          <XNumber v-bind="a" v-model="quantity" block :min="1" :max="500" />
        </XField>
      </XFieldGrid>
      <XFieldGrid v-else :phone="2">
        <XField v-slot="a" :label="t('inbounds.first')">
          <XNumber v-bind="a" v-model="first" block :min="1" />
        </XField>
        <XField v-slot="a" :label="t('inbounds.last')">
          <XNumber v-bind="a" v-model="last" block :min="first ?? 1" />
        </XField>
      </XFieldGrid>
      <XFieldGrid v-if="method > 0" :phone="2">
        <XField v-slot="a" :label="t('inbounds.prefix')">
          <XInput v-bind="a" v-model.trim="prefix" />
        </XField>
        <XField v-if="method > 2" v-slot="a" :label="t('inbounds.postfix')">
          <XInput v-bind="a" v-model.trim="postfix" />
        </XField>
      </XFieldGrid>

      <XFieldGrid v-if="inbound.protocol === 'vmess' || flowOk || subscription">
        <XField v-if="inbound.protocol === 'vmess'" v-slot="a" :label="t('inbounds.security')">
          <XSelect v-bind="a" v-model="base.security" :options="securityOptions" mono />
        </XField>
        <XField v-if="flowOk" v-slot="a" label="Flow">
          <XSelect v-bind="a" v-model="base.flow" :options="flowOptions" mono />
        </XField>
        <XField v-if="subscription" v-slot="a" label="Subscription" :help="t('inbounds.subscriptionDesc')" :action="t('inbounds.reset')" @action="base.subId = randomLowerNum(16)">
          <XInput v-bind="a" v-model.trim="base.subId" mono />
        </XField>
      </XFieldGrid>

      <XFieldGrid>
        <XField v-if="ipLimit" v-slot="a" :label="t('inbounds.ipLimit')" :help="t('inbounds.ipLimitDesc')">
          <XNumber v-bind="a" v-model="base.limitIp" block :min="0" />
        </XField>
        <XField v-slot="a" :label="t('inbounds.totalFlow')" :help="t('inbounds.totalHelp')">
          <XNumber v-bind="a" v-model="totalGb" block decimal :min="0" :unit="t('inbounds.gb')" />
        </XField>
      </XFieldGrid>

      <XSwitch :model-value="delayed" boxed :label="t('inbounds.delayedStart')" @update:model-value="setDelayed" />
      <XFieldGrid>
        <XField v-if="delayed" v-slot="a" :label="t('inbounds.expireDays')">
          <XNumber v-bind="a" v-model="days" block :min="0" />
        </XField>
        <XField v-else v-slot="a" :label="t('inbounds.colExpiry')" :help="t('inbounds.expiryHelp')">
          <XDateTime v-bind="a" v-model="expiry" />
        </XField>
        <XField v-if="base.expiryTime" v-slot="a" :label="t('inbounds.renew')" :help="t('inbounds.renewDesc')">
          <XNumber v-bind="a" v-model="base.reset" block :min="0" />
        </XField>
      </XFieldGrid>
    </div>

    <template #footer>
      <XButton variant="primary" :loading="saving" :disabled="!valid" @click="save">{{ t('inbounds.addBulk') }}</XButton>
    </template>
  </XModal>
</template>

<style scoped>
.bulk {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
