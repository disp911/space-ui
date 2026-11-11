<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../api'
import {
  toast, XButton, XDateTime, XDivider, XField, XFieldGrid, XInput, XModal, XNumber, XPlainTable, XSection, XSelect, XSwitch, type Option, type PlainColumn,
} from '../../kit'
import { formatDate } from '../../i18n/format'
import { useInbounds } from '../../stores/inbounds'
import { cause } from '../settings/form'
import { canFlow, canStream, clientsOf, fromDb, newInbound, setProtocol, toForm, trafficResets, tunWrong, withClients, type InboundForm } from './inbound'
import ClientFields from './ClientFields.vue'
import InboundProtocol from './InboundProtocol.vue'
import InboundStream from './InboundStream.vue'
import InboundExternal from './InboundExternal.vue'
import InboundSecurity from './InboundSecurity.vue'
import InboundSniffing from './InboundSniffing.vue'
import { protocols, type DbInbound, type Protocol } from './model'
import { sessionIdWrong } from '../xray/xhttp'
import { customSockoptWrong } from '../xray/sockopt'

/**
 * Window 11 of the mockup: a new inbound, or the stored one to change. It
 * opens centred and its body scrolls, as every window. Its fields stand in
 * one column, one under another, as the old panel's long window had them. The protocol of a
 * stored inbound stays; the panel serves one MTProto inbound only.
 */
const props = defineProps<{ inbound: DbInbound | null }>()
const open = defineModel<boolean>('open', { default: false })
const { t } = useI18n()
const store = useInbounds()

const GB = 1024 ** 3
const f = ref<InboundForm>(newInbound())
const saving = ref(false)

watch(open, (on) => {
  if (!on) return
  f.value = props.inbound ? fromDb(props.inbound) : newInbound()
  clientOpen.value = !props.inbound
  sniffingOpen.value = false
})

const editing = computed(() => !!props.inbound)
const mtprotoTaken = computed(() => !editing.value && store.list.some((ib) => ib.protocol === 'mtproto'))

const protocolOptions = computed<Option[]>(() =>
  protocols.map((p) => (p === 'mtproto' && mtprotoTaken.value ? { value: p, label: p, disabled: true, note: t('inbounds.mtprotoSingle') } : { value: p, label: p })),
)
const protocol = computed({
  get: () => f.value.protocol as string,
  set: (p: string) => setProtocol(f.value, p as Protocol),
})

const resetOptions = computed<Option[]>(() => {
  const labels: Record<string, string> = {
    never: t('inbounds.resetNever'),
    hourly: t('inbounds.resetHourly'),
    daily: t('inbounds.resetDaily'),
    weekly: t('inbounds.resetWeekly'),
    monthly: t('inbounds.resetMonthly'),
  }
  return trafficResets.map((r) => ({ value: r, label: labels[r] ?? r }))
})
const resetHelp = computed(() =>
  f.value.lastTrafficResetTime > 0
    ? `${t('inbounds.trafficResetHelp')}. ${t('inbounds.lastReset')}: ${formatDate(f.value.lastTrafficResetTime)}`
    : t('inbounds.trafficResetHelp'),
)

/** The limit in gigabytes, as the old panel showed it, to two places. */
const totalGb = computed({
  get: () => Math.round((f.value.total / GB) * 100) / 100,
  set: (gb: number | null) => (f.value.total = Math.round((gb ?? 0) * GB)),
})

// --- Clients -------------------------------------------------------------------------
// A new inbound gets its first client's fields, open; a stored one lists its
// clients, closed, as the old panel did. Clients are changed in their own windows.

const clientOpen = ref(true)
const sniffingOpen = ref(false)
const hasClients = computed(() => withClients(f.value.protocol, f.value.settings))
const clients = computed(() => clientsOf(f.value))

/** The column a client is known by, besides its email; VMess shows its security too. */
const clientColumns = computed<PlainColumn[]>(() => {
  const key: Record<string, string> = { vmess: 'ID', vless: 'ID', mtproto: 'Secret', trojan: 'Password', shadowsocks: 'Password', hysteria: 'Auth' }
  return [
    { key: 'email', label: t('inbounds.email') },
    { key: 'key', label: key[f.value.protocol] ?? 'ID' },
    ...(f.value.protocol === 'vmess' ? [{ key: 'security', label: t('inbounds.security') }] : []),
  ]
})
const keyOf = (c: Record<string, any>) => String(c.id ?? c.password ?? c.auth ?? '')

const portValid = computed(() => !!f.value.port && f.value.port >= 1 && f.value.port <= 65535)
const valid = computed(
  () => portValid.value && !(f.value.protocol === 'mtproto' && mtprotoTaken.value) && !sessionIdWrong(f.value.streamSettings?.xhttpSettings) && !customSockoptWrong(f.value.streamSettings?.sockopt) && !(f.value.protocol === 'tun' && tunWrong(f.value.settings)),
)

async function save() {
  if (!valid.value || saving.value) return
  saving.value = true
  try {
    await api.post(editing.value ? `panel/api/inbounds/update/${props.inbound!.id}` : 'panel/api/inbounds/add', toForm(f.value))
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
  <XModal v-model:open="open" :title="editing ? t('inbounds.editTitle') : t('inbounds.addTitle')" size="md" cancel scroll-body stack :dismissible="!saving">
    <div class="inbound">
      <XSwitch v-model="f.enable" boxed :label="t('inbounds.enable')" />
      <XFieldGrid>
        <XField v-slot="a" :label="t('inbounds.colRemark')">
          <XInput v-bind="a" v-model.trim="f.remark" />
        </XField>
        <XField v-slot="a" :label="t('inbounds.colProtocol')">
          <XSelect v-bind="a" v-model="protocol" :options="protocolOptions" :disabled="editing" mono />
        </XField>
      </XFieldGrid>
      <XFieldGrid>
        <XField v-slot="a" :label="t('inbounds.listen')" :help="t('inbounds.listenHelp')">
          <XInput v-bind="a" v-model.trim="f.listen" mono />
        </XField>
        <XField v-slot="a" :label="t('inbounds.colPort')">
          <XNumber v-bind="a" v-model="f.port" block :min="1" :max="65535" :invalid="!portValid" />
        </XField>
      </XFieldGrid>
      <XFieldGrid :cols="3">
        <XField v-slot="a" :label="t('inbounds.totalFlow')" :help="t('inbounds.totalHelp')">
          <XNumber v-bind="a" v-model="totalGb" block decimal :min="0" :unit="t('inbounds.gb')" />
        </XField>
        <XField v-slot="a" :label="t('inbounds.trafficReset')" :help="resetHelp">
          <XSelect v-bind="a" v-model="f.trafficReset" :options="resetOptions" />
        </XField>
        <XField v-slot="a" :label="t('inbounds.colExpiry')" :help="t('inbounds.expiryHelp')">
          <XDateTime v-bind="a" v-model="f.expiryTime" />
        </XField>
      </XFieldGrid>

      <template v-if="hasClients">
        <XSection v-if="!editing && clients[0]" v-model:open="clientOpen" :title="t('inbounds.client')" card>
          <ClientFields :client="clients[0]" :protocol="f.protocol" :method="f.settings.method" :flow="canFlow(f)" />
        </XSection>
        <XSection v-else-if="editing" v-model:open="clientOpen" :title="`${t('inbounds.clientCount')} : ${clients.length}`" card>
          <XPlainTable :columns="clientColumns" :rows="clients" :row-key="(c, i) => `${i}-${c.email}`" :height="280" :phone-height="320">
            <template #cell-email="{ row }">{{ row.email }}</template>
            <template #cell-key="{ row }"><span class="inbound__key">{{ keyOf(row) }}</span></template>
            <template #cell-security="{ row }"><span class="inbound__key">{{ row.security }}</span></template>
          </XPlainTable>
        </XSection>
      </template>

      <InboundProtocol :f="f" />
      <InboundStream v-if="canStream(f.protocol)" :f="f" />
      <template v-if="canStream(f.protocol) || f.protocol === 'mtproto'">
        <XDivider spaced />
        <InboundExternal :f="f" />
      </template>
      <InboundSecurity v-if="canStream(f.protocol)" :f="f" />
      <template v-if="f.protocol !== 'mtproto'">
        <XDivider spaced />
        <InboundSniffing v-model:open="sniffingOpen" :f="f" />
      </template>
    </div>

    <template #footer>
      <XButton variant="primary" :loading="saving" :disabled="!valid" @click="save">{{ editing ? t('inbounds.update') : t('inbounds.createShort') }}</XButton>
    </template>
  </XModal>
</template>

<style scoped>
.inbound {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.inbound__key {
  display: block;
  max-width: 420px;
  overflow: hidden;
  color: var(--mut);
  font: 400 var(--fs-11) / var(--lh-tight) var(--font-mono);
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 767px) {
  .inbound__key {
    max-width: 170px;
  }
}
</style>
