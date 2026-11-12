<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../api'
import { toast, XButton, XChip, XDivider, XField, XInput, XKeyValue, XModal, XSegmented, XSelect, XSpinner, type Option } from '../../kit'
import { cause } from '../settings/form'
import { useXrayForm } from './form'
import type { Outbound } from './model'

/**
 * NordVPN NordLynx: log in with an access token or a private key, pick a
 * country, a city and a server, and make an outbound tagged nord-<host> of
 * them. The login lives on the server; the outbound goes into the template.
 */
const open = defineModel<boolean>('open', { default: false })
const { t } = useI18n()
const f = useXrayForm()

interface NordData {
  token?: string
  private_key: string
}
interface Country {
  id: number
  name: string
  code: string
}
interface City {
  id: number
  name: string
}
interface Server {
  id: number
  name: string
  hostname: string
  station: string
  load: number
  cityId: number | null
  cityName: string
  technologies: { id: number; metadata: { name: string; value: string }[] }[]
}

const data = ref<NordData | null>(null)
const busy = ref<'load' | 'login' | 'logout' | 'servers' | null>(null)
const way = ref<'token' | 'key'>('token')
const secret = ref('')

const countries = ref<Country[]>([])
const countryId = ref<number | null>(null)
const cities = ref<City[]>([])
/** 0 is every city of the country. */
const cityId = ref(0)
const servers = ref<Server[]>([])
const serverId = ref<number | null>(null)

const parse = <T,>(text: string | null | undefined): T | null => (text ? (JSON.parse(text) as T) : null)
const form = (fields: Record<string, string>) => {
  const fd = new FormData()
  for (const [k, v] of Object.entries(fields)) fd.set(k, v)
  return fd
}

async function run(what: NonNullable<typeof busy.value>, job: () => Promise<void>) {
  busy.value = what
  try {
    await job()
  } catch (e) {
    toast('err', cause(e))
  } finally {
    busy.value = null
  }
}

async function loadCountries() {
  countries.value = parse<Country[]>(await api.post<string>('panel/xray/nord/countries')) ?? []
}

watch(open, (on) => {
  if (!on) return
  run('load', async () => {
    data.value = parse<NordData>(await api.post<string>('panel/xray/nord/data'))
    if (data.value) await loadCountries()
  })
})

function login() {
  run('login', async () => {
    const reply = way.value === 'token'
      ? await api.post<string>('panel/xray/nord/reg', form({ token: secret.value.trim() }))
      : await api.post<string>('panel/xray/nord/setKey', form({ key: secret.value.trim() }))
    data.value = parse<NordData>(reply)
    secret.value = ''
    if (data.value) await loadCountries()
  })
}

const outbounds = () => (f.draft.config.outbounds ??= []) as Outbound[]
const index = computed(() => ((f.draft.config.outbounds ?? []) as Outbound[]).findIndex((o) => String(o.tag ?? '').startsWith('nord-')))

function logout() {
  run('logout', async () => {
    await api.post('panel/xray/nord/del')
    if (index.value >= 0) outbounds().splice(index.value, 1)
    // Rules that sent traffic to NordVPN go with it, as on the old panel.
    const routing = f.draft.config.routing
    if (Array.isArray(routing?.rules)) routing.rules = routing.rules.filter((r: Outbound) => !String(r.outboundTag ?? '').startsWith('nord-'))
    data.value = null
    countries.value = []
    countryId.value = null
    cities.value = []
    cityId.value = 0
    servers.value = []
    serverId.value = null
  })
}

// --- Country, city, server -----------------------------------------------------

watch(countryId, (id) => {
  servers.value = []
  cities.value = []
  serverId.value = null
  cityId.value = 0
  if (id == null) return
  run('servers', async () => {
    type Reply = { servers?: (Server & { location_ids?: number[] })[]; locations?: { id: number; country?: { city?: City } }[] }
    const reply = parse<Reply>(await api.post<string>('panel/xray/nord/servers', form({ countryId: String(id) })))
    const byLocation = new Map<number, City>()
    const found = new Map<number, City>()
    for (const loc of reply?.locations ?? []) {
      const city = loc.country?.city
      if (!city) continue
      found.set(city.id, city)
      byLocation.set(loc.id, city)
    }
    cities.value = [...found.values()].sort((a, b) => a.name.localeCompare(b.name))
    servers.value = (reply?.servers ?? [])
      .map((s) => {
        const city = byLocation.get(s.location_ids?.[0] ?? -1)
        return { ...s, cityId: city?.id ?? null, cityName: city?.name ?? 'Unknown' }
      })
      .sort((a, b) => a.load - b.load)
    serverId.value = servers.value[0]?.id ?? null
    if (!servers.value.length) toast('err', t('xray.nordNoServers'))
  })
})

const shownServers = computed(() => (!cityId.value ? servers.value : servers.value.filter((s) => s.cityId === cityId.value)))
watch(cityId, () => (serverId.value = shownServers.value[0]?.id ?? null))

const countryOptions = computed<Option<number>[]>(() => countries.value.map((c) => ({ value: c.id, label: `${c.name} (${c.code})` })))
const cityOptions = computed<Option<number>[]>(() => [
  { value: 0, label: t('xray.nordAllCities') },
  ...cities.value.map((c) => ({ value: c.id, label: c.name })),
])
const serverOptions = computed<Option<number>[]>(() =>
  shownServers.value.map((s) => ({ value: s.id, label: t('xray.nordLoad', { name: `${s.cityName} - ${s.name}`, load: s.load }) })),
)

/** The chosen server as a WireGuard outbound, as the old panel built it. */
const outbound = computed<Outbound | null>(() => {
  const s = servers.value.find((x) => x.id === serverId.value)
  const key = s?.technologies.find((x) => x.id === 35)?.metadata.find((m) => m.name === 'public_key')?.value
  if (!s || !key || !data.value) return null
  return {
    tag: `nord-${s.hostname}`,
    protocol: 'wireguard',
    settings: {
      secretKey: data.value.private_key,
      address: ['10.5.0.2/32'],
      peers: [{ publicKey: key, endpoint: `${s.station}:51820` }],
      noKernelTun: false,
    },
  }
})

function putOutbound() {
  const o = outbound.value
  if (!o) return
  if (index.value >= 0) {
    const old = outbounds()[index.value]!.tag
    outbounds()[index.value] = o
    // Rules follow the outbound to its new tag.
    for (const r of (f.draft.config.routing?.rules ?? []) as Outbound[]) if (r.outboundTag === old) r.outboundTag = o.tag
    toast('ok', t('xray.nordUpdated'))
  } else {
    outbounds().push(o)
    toast('ok', t('xray.nordAdded'))
  }
  open.value = false
}

const ways = computed(() => [
  { value: 'token' as const, label: t('xray.nordToken') },
  { value: 'key' as const, label: t('xray.nordKey') },
])

const keys = computed(() =>
  data.value
    ? [
        ...(data.value.token ? [{ label: t('xray.nordToken'), value: data.value.token }] : []),
        { label: t('xray.nordKey'), value: data.value.private_key },
      ]
    : [],
)
</script>

<template>
  <XModal v-model:open="open" title="NordVPN NordLynx" :dismissible="busy !== 'login' && busy !== 'logout'">
    <!-- While the account is looked up a spinner stands where the login form would be. -->
    <div v-if="!data && busy === 'load'" class="nord nord--wait" role="status"><XSpinner :size="16" /></div>

    <div v-else-if="!data" class="nord">
      <div class="nord__way">
        <XSegmented v-model="way" :options="ways" :label="t('xray.loginBy')" block />
      </div>
      <div class="nord__line">
        <XInput v-model="secret" mono :aria-label="way === 'token' ? t('xray.nordToken') : t('xray.nordKey')" />
        <XButton variant="primary" :loading="busy === 'login'" :disabled="!secret.trim()" @click="login">
          {{ way === 'token' ? t('xray.nordLogin') : t('xray.save') }}
        </XButton>
      </div>
    </div>

    <div v-else class="nord">
      <XKeyValue :rows="keys" />
      <div>
        <XButton variant="danger" size="sm" :loading="busy === 'logout'" @click="logout">{{ t('xray.nordLogout') }}</XButton>
      </div>

      <XDivider :label="t('xray.settings')" />
      <XField v-slot="a" :label="t('xray.nordCountry')">
        <XSelect v-bind="a" v-model="countryId" :options="countryOptions" search />
      </XField>
      <!-- A country's cities and servers load after it is picked: a spinner after their labels meanwhile. -->
      <XField :label="t('xray.nordCity')">
        <template #default="a"><XSelect v-bind="a" v-model="cityId" :options="cityOptions" :disabled="!cities.length" /></template>
        <template v-if="busy === 'servers'" #badge><XSpinner :size="12" class="nord__spin" /></template>
      </XField>
      <XField :label="t('xray.nordServer')">
        <template #default="a"><XSelect v-bind="a" v-model="serverId" :options="serverOptions" :disabled="!shownServers.length" /></template>
        <template v-if="busy === 'servers'" #badge><XSpinner :size="12" class="nord__spin" /></template>
      </XField>

      <XDivider :label="t('xray.outStatus')" />
      <div class="nord__status">
        <XChip :tone="index >= 0 ? 'ok' : 'err'">{{ index >= 0 ? t('xray.enabled') : t('xray.disabled') }}</XChip>
        <XButton v-if="index >= 0" size="sm" icon="reset" :disabled="!outbound" @click="putOutbound">{{ t('xray.outReset') }}</XButton>
        <XButton v-else variant="primary" size="sm" :disabled="!outbound" @click="putOutbound">{{ t('xray.outAdd') }}</XButton>
      </div>
    </div>
  </XModal>
</template>

<style scoped>
.nord {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* As tall as WARP's «Create» row, 16px above and below a control. */
.nord--wait {
  align-items: center;
  justify-content: center;
  min-height: calc(var(--ctl-h) + 32px);
  color: var(--acc-t);
}

.nord__spin {
  color: var(--acc-t);
}

.nord__way {
  width: 300px;
  max-width: 100%;
}

.nord__line {
  display: flex;
  gap: 8px;
}

.nord__line > :first-child {
  flex: 1;
  min-width: 0;
}

.nord__status {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
</style>
