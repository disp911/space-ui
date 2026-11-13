<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { XButton, XField, XFieldGrid, XInput, XInputList, XModal, XNumber, XSelect, XSwitch } from '../../kit'
import { defaultDnsServer, dnsStrategies, listOf, type DnsServer } from './model'

/**
 * A DNS server to create, or the one at index to edit: where it is, how it
 * resolves and for which domains, and the IPs its answers must or must not
 * hold. Only what is changed is written: confirmed as shown, the server stays
 * as it was; a bare address stays one while only the address changes. A new
 * server gets the old panel's defaults.
 */
const props = defineProps<{ server: DnsServer | null; index: number }>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ save: [server: DnsServer] }>()
const { t } = useI18n()

const form = reactive({
  address: '',
  port: 53 as number | null,
  queryStrategy: 'UseIP',
  domains: [] as string[],
  expectIPs: [] as string[],
  unexpectedIPs: [] as string[],
  skipFallback: true,
  disableCache: false,
  finalQuery: false,
  clientIp: '',
  tag: '',
  timeoutMs: 0 as number | null,
  /** '' leaves it to the DNS block's own; on or off overrides it for this server. */
  serveStale: '' as '' | 'on' | 'off',
  serveExpiredTTL: null as number | null,
})

/** The server as an object: a bare address gets the defaults around it. */
const base = () => {
  const s = props.server
  if (s == null) return defaultDnsServer()
  return typeof s === 'string' ? { ...defaultDnsServer(), address: s } : s
}

type Form = typeof form
/** The form as it opened, to tell what was changed. */
let shown: Form | null = null

watch(open, (on) => {
  if (!on) return
  const s = base()
  Object.assign(form, {
    address: String(s.address ?? ''),
    port: typeof s.port === 'number' ? s.port : null,
    queryStrategy: String(s.queryStrategy ?? 'UseIP'),
    domains: listOf(s.domains),
    expectIPs: listOf(s.expectIPs),
    unexpectedIPs: listOf(s.unexpectedIPs),
    skipFallback: !!s.skipFallback,
    disableCache: !!s.disableCache,
    finalQuery: !!s.finalQuery,
    clientIp: String(s.clientIp ?? ''),
    tag: String(s.tag ?? ''),
    timeoutMs: typeof s.timeoutMs === 'number' ? s.timeoutMs : 0,
    serveStale: s.serveStale === true ? 'on' : s.serveStale === false ? 'off' : '',
    serveExpiredTTL: typeof s.serveExpiredTTL === 'number' ? s.serveExpiredTTL : null,
  })
  shown = JSON.parse(JSON.stringify(form))
})

const list = (v: string[]) => v.map((x) => x.trim()).filter(Boolean)

function save() {
  const now: Form = { ...form, address: form.address.trim(), domains: list(form.domains), expectIPs: list(form.expectIPs), unexpectedIPs: list(form.unexpectedIPs) }
  const changed = (Object.keys(now) as (keyof Form)[]).filter((k) => JSON.stringify(now[k]) !== JSON.stringify(shown?.[k]))
  open.value = false
  if (props.server != null && !changed.length) return
  const s = props.server
  if (typeof s === 'string' && changed.every((k) => k === 'address')) return emit('save', now.address)
  // A new server or a bare address with more settings: the whole form, as the old panel wrote it.
  const out: Record<string, unknown> = s && typeof s === 'object' ? { ...s } : { ...base() }
  for (const k of s && typeof s === 'object' ? changed : (Object.keys(now) as (keyof Form)[])) out[k] = now[k]
  if (out.port == null) delete out.port
  // Left as Xray's own, they leave the server; serveStale is a switch of three.
  for (const k of ['clientIp', 'tag'] as const) if (!String(out[k] ?? '').trim()) delete out[k]
  if (!out.timeoutMs) delete out.timeoutMs
  if (out.serveExpiredTTL == null) delete out.serveExpiredTTL
  if (out.serveStale === 'on') out.serveStale = true
  else if (out.serveStale === 'off') out.serveStale = false
  else delete out.serveStale
  emit('save', out)
}

const strategies = dnsStrategies.map((v) => ({ value: v, label: v }))
const staleOptions = [
  { value: '', label: 'Default' },
  { value: 'on', label: 'On' },
  { value: 'off', label: 'Off' },
]
const title = computed(() => (props.server != null ? t('xray.dnsEditN', { n: props.index + 1 }) : t('xray.dnsAdd')))
</script>

<template>
  <XModal v-model:open="open" :title="title" size="md" cancel scroll-body stack>
    <div class="dns">
      <!-- Address and port share a line, in a window that stacks its fields too. -->
      <XFieldGrid columns="minmax(0, 1fr) 120px" phone-columns="minmax(0, 1fr) 120px">
        <XField v-slot="a" :label="t('xray.outAddress')">
          <XInput v-bind="a" v-model="form.address" mono />
        </XField>
        <XField v-slot="a" :label="t('xray.port')">
          <XNumber v-bind="a" block v-model="form.port" :min="1" :max="65535" />
        </XField>
      </XFieldGrid>
      <XField v-slot="a" :label="t('xray.dnsStrategy')">
        <XSelect v-bind="a" v-model="form.queryStrategy" :options="strategies" mono />
      </XField>

      <XInputList v-model="form.domains" :title="t('xray.dnsDomains')" mono />
      <XInputList v-model="form.expectIPs" :title="t('xray.dnsExpectIPs')" mono />
      <XInputList v-model="form.unexpectedIPs" :title="t('xray.dnsUnexpectIPs')" mono />

      <XFieldGrid :cols="3">
        <XSwitch v-model="form.skipFallback" boxed label="Skip Fallback" />
        <XSwitch v-model="form.disableCache" boxed label="Disable Cache" />
        <XSwitch v-model="form.finalQuery" boxed label="Final Query" />
      </XFieldGrid>

      <XFieldGrid>
        <XField v-slot="a" label="Client IP"><XInput v-bind="a" v-model.trim="form.clientIp" mono placeholder="203.0.113.1" /></XField>
        <XField v-slot="a" label="Tag"><XInput v-bind="a" v-model.trim="form.tag" mono /></XField>
        <XField v-slot="a" label="Timeout (ms)"><XNumber v-bind="a" block v-model="form.timeoutMs" :min="0" placeholder="4000" /></XField>
        <XField v-slot="a" label="Serve Stale"><XSelect v-bind="a" v-model="form.serveStale" :options="staleOptions" /></XField>
        <XField v-slot="a" label="Serve Expired TTL"><XNumber v-bind="a" block v-model="form.serveExpiredTTL" :min="0" /></XField>
      </XFieldGrid>
    </div>

    <template #footer>
      <XButton variant="primary" @click="save">{{ t('xray.confirm') }}</XButton>
    </template>
  </XModal>
</template>

<style scoped>
.dns {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}
</style>
