<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../api'
import { toast, XButton, XChip, XField, XModal, XPopConfirm, XStateBar, XTextarea, type Tone } from '../../kit'
import { formatBytes } from '../../i18n/format'
import { useInbounds } from '../../stores/inbounds'
import { cause } from '../settings/form'
import ClientFields from './ClientFields.vue'
import { canFlow, finalClient, fromDb, newClient } from './inbound'
import { clientKey, usedOf, type Client, type ClientStat, type DbInbound } from './model'

/**
 * Window 12 of the mockup: a new client of an inbound, or one of its clients
 * to change, with the old panel's client form. A client being changed also
 * shows whether it is over, its traffic with a reset, and its IP log when the
 * panel keeps one (MTProto, or an IP limit with the log on).
 */
const props = defineProps<{ inbound: DbInbound | null; client: Client | null }>()
const open = defineModel<boolean>('open', { default: false })
const { t } = useI18n()
const store = useInbounds()

const GB = 1024 ** 3
type Json = Record<string, any>

const form = computed(() => (props.inbound ? fromDb(props.inbound) : null))
const editing = computed(() => !!props.client)
const c = ref<Json>({})
const stat = ref<ClientStat | null>(null)
const saving = ref(false)

watch(open, (on) => {
  if (!on || !props.inbound) return
  const s = form.value!.settings
  c.value = props.client ? JSON.parse(JSON.stringify(props.client)) : newClient(props.inbound.protocol, s.method)
  stat.value = props.client ? { ...(props.inbound.clientStats ?? []).find((x) => x.email === props.client!.email)! } : null
  if (!stat.value?.email) stat.value = null
  ips.value = ''
  if (ipLog.value) void loadIps()
})

// --- State of a client being changed ------------------------------------------------

const expired = computed(() => !!stat.value && stat.value.expiryTime > 0 && stat.value.expiryTime <= Date.now())
const exhausted = computed(() => !!stat.value && stat.value.total > 0 && usedOf(stat.value) >= stat.value.total)

/** The usage chip's colour, as the old panel's clientUsageColor. */
const usageTone = computed<Tone>(() => {
  const s = stat.value
  if (!s || !s.total) return 'unl'
  const used = usedOf(s)
  if (used < s.total - (store.defaults?.trafficDiff ?? 0) * GB) return 'ok'
  return used < s.total ? 'warn' : 'err'
})
const usage = computed(() => (stat.value ? `${formatBytes(stat.value.up)} / ${formatBytes(stat.value.down)} (${formatBytes(usedOf(stat.value))})` : ''))

async function resetTraffic() {
  if (!props.inbound || !stat.value) return
  try {
    await api.post(`panel/api/inbounds/${props.inbound.id}/resetClientTraffic/${encodeURIComponent(stat.value.email)}`)
    stat.value = { ...stat.value, up: 0, down: 0 }
    void store.load().catch(() => {})
  } catch (e) {
    toast('err', cause(e))
  }
}

// --- IP log -------------------------------------------------------------------------

const mtproto = computed(() => props.inbound?.protocol === 'mtproto')
const ipLog = computed(() => editing.value && !!props.client?.email && (mtproto.value || (!!store.defaults?.ipLimitEnable && (props.client?.limitIp ?? 0) > 0)))
const ips = ref('')
const ipsLoading = ref(false)
const clearingIps = ref(false)
const ipLines = computed(() => Math.min(10, Math.max(3, ips.value.split('\n').length)))

async function loadIps() {
  ipsLoading.value = true
  try {
    const list = await api.post<string[] | string>(`panel/api/inbounds/clientIps/${encodeURIComponent(props.client!.email)}`)
    ips.value = Array.isArray(list) ? list.join('\n') : String(list ?? '')
  } catch (e) {
    ips.value = cause(e)
  } finally {
    ipsLoading.value = false
  }
}

async function clearIps() {
  clearingIps.value = true
  try {
    await api.post(`panel/api/inbounds/clearClientIps/${encodeURIComponent(props.client!.email)}`)
    ips.value = ''
  } catch (e) {
    toast('err', cause(e))
  } finally {
    clearingIps.value = false
  }
}

// --- Saving -------------------------------------------------------------------------

async function save() {
  if (!props.inbound || !form.value || saving.value) return
  saving.value = true
  const id = props.inbound.id
  const settings = JSON.stringify({ clients: [finalClient(form.value, c.value)] })
  try {
    if (props.client) await api.post(`panel/api/inbounds/updateClient/${encodeURIComponent(clientKey(props.inbound.protocol, props.client))}`, { id, settings })
    else await api.post('panel/api/inbounds/addClient', { id, settings })
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
  <XModal v-model:open="open" :title="editing ? t('inbounds.editClient') : t('inbounds.addClient')" size="md" cancel scroll-body stack :dismissible="!saving">
    <div v-if="inbound && form" class="client-window">
      <XStateBar v-if="editing && (expired || exhausted)">Account is {{ expired ? 'Expired' : 'Traffic Ended' }} And Disabled</XStateBar>
      <ClientFields :client="c" :protocol="inbound.protocol" :method="form.settings.method" :flow="canFlow(form)">
        <template v-if="ipLog" #ips>
          <XField
            v-slot="a"
            :label="t('inbounds.ipLog')"
            :help="mtproto ? t('inbounds.ipLogMtprotoDesc') : t('inbounds.ipLogDesc')"
            :action="mtproto ? undefined : t('inbounds.ipLogClear')"
            action-icon="delete"
            :action-loading="clearingIps"
            @action="clearIps"
          >
            <XTextarea v-bind="a" :model-value="ips" :rows="ipLines" :loading="ipsLoading" mono readonly />
          </XField>
        </template>
        <template v-if="editing && stat" #usage>
          <XField :label="t('inbounds.usage')">
            <div class="client-window__usage">
              <XChip :tone="usageTone" mono>{{ usage }}</XChip>
              <XPopConfirm v-slot="{ props: trigger }" :text="t('inbounds.resetConfirm')" :cta="t('inbounds.reset')" :run="resetTraffic">
                <XButton v-bind="trigger" icon="reset" :label="t('inbounds.resetTraffic')" />
              </XPopConfirm>
            </div>
          </XField>
        </template>
        <template v-if="editing && expired" #expiry>
          <XChip tone="err">Expired</XChip>
        </template>
      </ClientFields>
    </div>

    <template #footer>
      <XButton variant="primary" :loading="saving" @click="save">{{ editing ? t('inbounds.submitEdit') : t('inbounds.submitAdd') }}</XButton>
    </template>
  </XModal>
</template>

<style scoped>
.client-window {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.client-window__usage {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: var(--ctl-h);
}
</style>
