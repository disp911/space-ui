<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../api'
import { formatBytes } from '../../i18n/format'
import { toast, XButton, XChip, XDivider, XField, XInput, XKeyValue, XModal, XSection, XSpinner } from '../../kit'
import { base64Bytes, wireguardKeys } from '../../wireguard'
import { cause } from '../settings/form'
import { useXrayForm } from './form'
import type { Outbound } from './model'

/**
 * Cloudflare WARP: an account registered on the server with keys made here,
 * a WARP+ license, the account's details and an outbound tagged warp made of
 * them. The account changes on the server at once; the outbound goes into
 * the template, saved with it.
 */
const open = defineModel<boolean>('open', { default: false })
const { t } = useI18n()
const f = useXrayForm()

interface WarpData {
  access_token: string
  device_id: string
  license_key: string
  private_key: string
}
interface WarpConfig {
  name: string
  model: string
  enabled: boolean
  account?: { account_type: string; role: string; premium_data: number; quota: number; usage?: number }
  config: {
    client_id: string
    peers: { public_key: string; endpoint: { host: string } }[]
    interface: { addresses: { v4?: string; v6?: string } }
  }
}

const data = ref<WarpData | null>(null)
const config = ref<WarpConfig | null>(null)
const busy = ref<'load' | 'reg' | 'del' | 'license' | 'info' | null>(null)
const license = ref('')
const licenseOpen = ref(true)

// The server answers with JSON in a string, empty when there is no account.
const parse = <T,>(text: string | null | undefined): T | null => (text ? (JSON.parse(text) as T) : null)

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

watch(open, (on) => {
  if (!on) return
  config.value = null
  license.value = ''
  run('load', async () => {
    data.value = parse<WarpData>(await api.post<string>('panel/xray/warp/data'))
  })
})

const form = (fields: Record<string, string>) => {
  const fd = new FormData()
  for (const [k, v] of Object.entries(fields)) fd.set(k, v)
  return fd
}

function register() {
  run('reg', async () => {
    const keys = wireguardKeys()
    const reply = parse<{ data: WarpData; config: WarpConfig }>(
      await api.post<string>('panel/xray/warp/reg', form({ privateKey: keys.privateKey, publicKey: keys.publicKey })),
    )
    data.value = reply?.data ?? null
    config.value = reply?.config ?? null
  })
}

function updateLicense() {
  run('license', async () => {
    data.value = parse<WarpData>(await api.post<string>('panel/xray/warp/license', form({ license: license.value })))
    config.value = null
    license.value = ''
  })
}

function loadInfo() {
  run('info', async () => {
    config.value = parse<WarpConfig>(await api.post<string>('panel/xray/warp/config'))
  })
}

const outbounds = () => (f.draft.config.outbounds ??= []) as Outbound[]
const index = computed(() => ((f.draft.config.outbounds ?? []) as Outbound[]).findIndex((o) => o.tag === 'warp'))

function remove() {
  run('del', async () => {
    await api.post('panel/xray/warp/del')
    data.value = null
    config.value = null
    if (index.value >= 0) outbounds().splice(index.value, 1)
    open.value = false
  })
}

/** The outbound the account makes, as the old panel built it. */
const outbound = computed<Outbound | null>(() => {
  const c = config.value?.config
  const peer = c?.peers?.[0]
  if (!c || !peer || !data.value) return null
  const a = c.interface.addresses
  return {
    tag: 'warp',
    protocol: 'wireguard',
    settings: {
      mtu: 1420,
      secretKey: data.value.private_key,
      address: [a.v4 && `${a.v4}/32`, a.v6 && `${a.v6}/128`].filter(Boolean),
      reserved: base64Bytes(c.client_id),
      peers: [{ publicKey: peer.public_key, endpoint: peer.endpoint.host }],
      noKernelTun: false,
    },
  }
})

function putOutbound() {
  if (!outbound.value) return
  if (index.value >= 0) outbounds()[index.value] = outbound.value
  else outbounds().push(outbound.value)
  open.value = false
}

const keys = computed(() =>
  data.value
    ? [
        { label: 'Access Token', value: data.value.access_token },
        { label: 'Device ID', value: data.value.device_id },
        { label: 'License Key', value: data.value.license_key },
        { label: 'Private Key', value: data.value.private_key },
      ]
    : [],
)

const details = computed(() => {
  const c = config.value
  if (!c) return []
  const rows = [
    { label: 'Device Name', value: c.name },
    { label: 'Device Model', value: c.model },
    { label: 'Device Enabled', value: String(c.enabled) },
  ]
  if (c.account) {
    rows.push(
      { label: 'Account Type', value: c.account.account_type },
      { label: 'Role', value: c.account.role },
      { label: 'WARP+ Data', value: formatBytes(c.account.premium_data ?? 0) },
      { label: 'Quota', value: formatBytes(c.account.quota ?? 0) },
    )
    if (c.account.usage) rows.push({ label: 'Usage', value: formatBytes(c.account.usage) })
  }
  return rows
})
</script>

<template>
  <XModal v-model:open="open" title="Cloudflare WARP" :dismissible="busy !== 'reg' && busy !== 'del'">
    <!-- While the account is looked up a spinner stands where «Create» would be. -->
    <div v-if="!data" class="warp warp--empty">
      <span v-if="busy === 'load'" class="warp__wait" role="status"><XSpinner :size="16" /></span>
      <XButton v-else variant="primary" :loading="busy === 'reg'" @click="register">{{ t('xray.create') }}</XButton>
    </div>

    <div v-else class="warp">
      <XKeyValue :rows="keys" />
      <div>
        <XButton variant="danger" size="sm" icon="delete" :loading="busy === 'del'" @click="remove">{{ t('xray.delete') }}</XButton>
      </div>

      <XDivider :label="t('xray.settings')" />
      <XSection v-model:open="licenseOpen" title="WARP/WARP+ License Key" card>
        <XField v-slot="a" label="Key">
          <div class="warp__line">
            <XInput v-bind="a" v-model="license" mono />
            <XButton variant="primary" :disabled="license.length < 26" :loading="busy === 'license'" @click="updateLicense">{{ t('xray.update') }}</XButton>
          </div>
        </XField>
      </XSection>

      <XDivider :label="t('xray.accountInfo')" />
      <div>
        <XButton size="sm" icon="refresh" :loading="busy === 'info'" @click="loadInfo">{{ t('xray.info') }}</XButton>
      </div>
      <template v-if="config">
        <XKeyValue :rows="details" />
        <XDivider :label="t('xray.outStatus')" />
        <div class="warp__status">
          <XChip :tone="index >= 0 ? 'ok' : 'err'">{{ index >= 0 ? t('xray.enabled') : t('xray.disabled') }}</XChip>
          <XButton v-if="index >= 0" size="sm" icon="reset" :disabled="!outbound" @click="putOutbound">{{ t('xray.outReset') }}</XButton>
          <XButton v-else variant="primary" size="sm" :disabled="!outbound" @click="putOutbound">{{ t('xray.outAdd') }}</XButton>
        </div>
      </template>
    </div>
  </XModal>
</template>

<style scoped>
.warp {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.warp--empty {
  align-items: center;
  padding: 16px 0;
}

.warp__wait {
  display: flex;
  align-items: center;
  height: var(--ctl-h);
  color: var(--acc-t);
}

.warp__line {
  display: flex;
  flex: 1;
  gap: 8px;
  min-width: 0;
}

.warp__line > :first-child {
  flex: 1;
  min-width: 0;
}

.warp__status {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
</style>
