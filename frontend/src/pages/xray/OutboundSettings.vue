<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  XField, XFieldGrid, XInlineButton, XInput, XInputList, XListHead, XNest, XNestCard, XNumber, XSection, XSelect, XSwitch, XTagInput, type Option,
} from '../../kit'
import { wireguardKeys } from '../../wireguard'
import { options, type Outbound } from './model'
import {
  canFlow, canVisionSeed, defaultDnsRule, defaultFinalRule, defaultPeer, domainStrategies, flows, hasUser, server, ssMethods, user,
  blackholeWrong, isIp, items, vmessSecurities,
} from './outbound'

/** The part of the outbound window that belongs to its protocol, in a card titled with it. */
const props = defineProps<{ o: Outbound; tags: string[] }>()
const { t } = useI18n()
const open = ref(true)

const s = computed(() => {
  props.o.settings ??= {}
  // Read back through the reactive object: ??= hands back the plain new value.
  return props.o.settings
})
const srv = computed(() => server(props.o))
const usr = computed(() => user(props.o))

const list = (values: readonly string[], empty = false): Option[] => [
  ...(empty ? [{ value: '', label: 'Empty', plain: true }] : []),
  ...values.map((v) => ({ value: v, label: v })),
]

/** A comma list kept as an array, e.g. WireGuard addresses. */
const commaList = (key: string, numbers = false) =>
  computed({
    get: () => (Array.isArray(s.value[key]) ? s.value[key].join(',') : String(s.value[key] ?? '')),
    set: (v: string) => {
      const parts = v.split(',').map((x) => x.trim()).filter(Boolean)
      s.value[key] = numbers ? parts.map(Number) : parts
    },
  })

// --- VLESS -------------------------------------------------------------------------

const reverseTag = computed({
  get: () => s.value.reverse?.tag ?? '',
  set: (v: string) => (s.value.reverse = { ...(s.value.reverse ?? {}), tag: v }),
})
const seeds = computed(() => {
  usr.value.testseed ??= [900, 500, 900, 256]
  // Read back through the reactive object: ??= hands back the plain new value.
  return usr.value.testseed as number[]
})

// --- Shadowsocks, SOCKS, HTTP ------------------------------------------------------

/** SOCKS and HTTP keep one optional user: without a name there is none. */
const account = (key: 'user' | 'pass') =>
  computed({
    get: () => srv.value.users?.[0]?.[key] ?? '',
    set: (v: string) => {
      const u = { ...(srv.value.users?.[0] ?? {}), [key]: v }
      srv.value.users = u.user ? [u] : []
    },
  })
const socksUser = account('user')
const socksPass = account('pass')

// --- WireGuard ---------------------------------------------------------------------

const wgAddress = commaList('address')
const wgReserved = commaList('reserved', true)
const wgDns = computed({
  get: () => (Array.isArray(s.value.remoteDNS) ? s.value.remoteDNS : []),
  set: (v: string[]) => (s.value.remoteDNS = v),
})
const publicKey = computed(() => {
  try {
    return s.value.secretKey ? wireguardKeys(s.value.secretKey).publicKey : ''
  } catch {
    return ''
  }
})
const newKey = () => (s.value.secretKey = wireguardKeys().privateKey)

// --- Freedom -----------------------------------------------------------------------

/** A rule's network may come as a list; the field shows it as text. */
const networkText = (r: Record<string, any>) => (Array.isArray(r.network) ? r.network.join(',') : (r.network ?? ''))

// --- Blackhole ---------------------------------------------------------------------

const responseType = computed({
  get: () => s.value.response?.type ?? '',
  set: (v: string) => (s.value.response = v ? { type: v } : undefined),
})

const remove = (list: unknown[], i: number) => list.splice(i, 1)
</script>

<template>
  <XSection v-model:open="open" :title="o.protocol" card mono>
    <!-- VMess, VLESS -->
    <template v-if="o.protocol === 'vmess' || o.protocol === 'vless'">
      <XFieldGrid :cols="2">
        <XField v-slot="a" label="ID">
          <XInput v-bind="a" v-model="usr.id" mono />
        </XField>
        <XField v-if="o.protocol === 'vmess'" v-slot="a" label="Security">
          <XSelect v-bind="a" v-model="usr.security" :options="list(vmessSecurities)" mono />
        </XField>
        <XField v-else v-slot="a" label="encryption">
          <XInput v-bind="a" v-model="usr.encryption" mono />
        </XField>
      </XFieldGrid>
      <XFieldGrid :cols="2" v-if="o.protocol === 'vless'">
        <XField v-slot="a" :label="t('xray.reverseTag')" :help="t('xray.reverseTagDesc')">
          <XInput v-bind="a" v-model="reverseTag" mono />
        </XField>
        <XField v-if="canFlow(o)" v-slot="a" label="Flow">
          <XSelect v-bind="a" :model-value="usr.flow ?? ''" :options="list(flows, true)" @update:model-value="usr.flow = String($event)" mono />
        </XField>
      </XFieldGrid>
      <XNest v-if="canVisionSeed(o)" bare>
        <XFieldGrid :cols="1">
          <XFieldGrid :cols="3">
            <XField v-slot="a" label="Vision Pre-Connect">
              <XNumber v-bind="a" block v-model="usr.testpre" :min="0" :max="10" placeholder="0" />
            </XField>
          </XFieldGrid>
          <XListHead title="Vision Seed" />
          <XFieldGrid :cols="4" :phone="4">
            <XField v-for="(_, i) in 4" :key="i" v-slot="a" :label="`[${i}]`">
              <XNumber v-bind="a" block v-model="seeds[i]" :min="0" :max="9999" />
            </XField>
          </XFieldGrid>
        </XFieldGrid>
      </XNest>
    </template>

    <!-- Trojan, Shadowsocks -->
    <XFieldGrid :cols="2" v-if="o.protocol === 'trojan' || o.protocol === 'shadowsocks'">
      <XField v-slot="a" :label="t('xray.password')">
        <XInput v-bind="a" v-model="srv.password" mono />
      </XField>
      <XField v-if="o.protocol === 'shadowsocks'" v-slot="a" :label="t('xray.ssMethod')">
        <XSelect v-bind="a" v-model="srv.method" :options="list(ssMethods)" mono />
      </XField>
    </XFieldGrid>

    <!-- SOCKS, HTTP -->
    <XFieldGrid :cols="2" v-if="hasUser(o)">
      <XField v-slot="a" :label="t('xray.username')">
        <XInput v-bind="a" v-model="socksUser" mono />
      </XField>
      <XField v-slot="a" :label="t('xray.password')">
        <XInput v-bind="a" v-model="socksPass" mono />
      </XField>
    </XFieldGrid>

    <!-- Hysteria -->
    <XFieldGrid :cols="3" v-if="o.protocol === 'hysteria'">
      <XField v-slot="a" label="Version">
        <XNumber v-bind="a" block v-model="s.version" disabled />
      </XField>
    </XFieldGrid>

    <!-- DNS -->
    <template v-if="o.protocol === 'dns'">
      <XFieldGrid :cols="2">
        <XField v-slot="a" :label="t('xray.ruleNetwork')">
          <XSelect v-bind="a" v-model="s.network" :options="list(['udp', 'tcp'])" mono />
        </XField>
      </XFieldGrid>
      <XListHead title="Rules" add @add="(s.rules ??= []).push(defaultDnsRule())">
        <XNestCard v-for="(r, i) in items(s.rules)" :key="i" :title="String(i + 1)">
          <template #actions>
            <XInlineButton icon="delete" danger :label="t('xray.removeItem', { n: i + 1 })" @click="remove(s.rules, i)" />
          </template>
          <XFieldGrid :cols="3">
            <XField v-slot="a" label="Action">
              <XSelect v-bind="a" v-model="r.action" :options="list(['direct', 'drop', 'return', 'hijack'])" mono />
            </XField>
            <XField v-slot="a" label="QType" :help="t('xray.hintQtype')">
              <XInput v-bind="a" v-model="r.qType" mono placeholder="1,3,23-24" />
            </XField>
            <XField v-slot="a" label="Domain" :help="t('xray.hintDomainRules')">
              <XInput v-bind="a" v-model="r.domain" mono placeholder="domain:example.com" />
            </XField>
          </XFieldGrid>
        </XNestCard>
      </XListHead>
    </template>

    <!-- Blackhole -->
    <XFieldGrid :cols="2" v-if="o.protocol === 'blackhole'">
      <XField v-slot="a" label="Response Type">
        <XSelect v-bind="a" v-model="responseType" :options="list(['none', 'http', 'custom'], true)" mono />
      </XField>
      <XField v-if="responseType === 'custom'" v-slot="a" label="Response Data" :help="t('xray.blackholeDataHelp')" class="x-wide">
        <XInput v-bind="a" v-model.trim="s.response.customResponseData" mono :invalid="blackholeWrong(o)" />
      </XField>
    </XFieldGrid>

    <!-- Freedom -->
    <template v-if="o.protocol === 'freedom'">
      <XFieldGrid :cols="2">
        <XField v-slot="a" label="Strategy">
          <XSelect v-bind="a" :model-value="s.domainStrategy ?? ''" :options="list(domainStrategies, true)" @update:model-value="s.domainStrategy = String($event)" mono />
        </XField>
        <XField v-slot="a" label="Redirect">
          <XInput v-bind="a" v-model="s.redirect" mono />
        </XField>
      </XFieldGrid>
      <XListHead title="Final Rules" add @add="(s.finalRules ??= []).push(defaultFinalRule())">
        <XNestCard v-for="(r, i) in items(s.finalRules)" :key="i" :title="String(i + 1)">
          <template #actions>
            <XInlineButton icon="delete" danger :label="t('xray.removeItem', { n: i + 1 })" @click="remove(s.finalRules, i)" />
          </template>
          <XFieldGrid :cols="3" :phone="2">
            <XField v-slot="a" label="Action">
              <XSelect v-bind="a" v-model="r.action" :options="list(['allow', 'block'])" mono />
            </XField>
            <XField v-slot="a" label="Network" :help="t('xray.hintNetwork')">
              <XInput v-bind="a" :model-value="networkText(r)" mono placeholder="tcp,udp" @update:model-value="r.network = $event" />
            </XField>
            <XField v-slot="a" label="Port" :help="t('xray.hintPortRule')">
              <XInput v-bind="a" v-model="r.port" mono placeholder="53,443" />
            </XField>
            <XField v-slot="a" label="IP" :help="t('xray.hintIpRule')">
              <XTagInput v-bind="a" v-model="r.ip" mono placeholder="geoip:cn,10.0.0.0/8" />
            </XField>
            <template v-if="r.action === 'block'">
              <XField v-slot="a" label="Delay">
                <XInput v-bind="a" v-model="r.delay" mono placeholder="30-90" />
              </XField>
              <XField v-slot="a" label="Log Level">
                <XSelect v-bind="a" v-model="r.logLevel" :options="list(options.logLevel, true)" mono />
              </XField>
            </template>
          </XFieldGrid>
        </XNestCard>
      </XListHead>
    </template>

    <!-- WireGuard -->
    <template v-if="o.protocol === 'wireguard'">
      <XFieldGrid :cols="2">
        <XField v-slot="a" :label="t('xray.privateKey')">
          <XInput v-bind="a" v-model="s.secretKey" mono>
            <template #actions><XInlineButton icon="refresh" :label="t('xray.newKey')" @click="newKey" /></template>
          </XInput>
        </XField>
        <XField v-slot="a" :label="t('xray.publicKey')">
          <XInput v-bind="a" :model-value="publicKey" mono readonly />
        </XField>
      </XFieldGrid>
      <XFieldGrid :cols="2">
        <XField v-slot="a" label="MTU">
          <XNumber v-bind="a" block v-model="s.mtu" :min="0" />
        </XField>
      </XFieldGrid>
      <XFieldGrid :cols="2">
        <XField v-slot="a" :label="t('xray.outAddress')" :help="t('xray.useComma')">
          <XInput v-bind="a" v-model="wgAddress" mono />
        </XField>
        <XField v-slot="a" label="Reserved" :help="t('xray.useComma')">
          <XInput v-bind="a" v-model="wgReserved" mono />
        </XField>
        <XSwitch v-model="s.noKernelTun" boxed label="No Kernel Tun" />
      </XFieldGrid>
      <!-- DNS inside the tunnel; none means Xray's own, Cloudflare's 1.1.1.1 and its kin. -->
      <XInputList v-model="wgDns" title="Remote DNS" mono placeholder="1.1.1.1" :invalid="(v) => !isIp(v)" />
      <XListHead title="Peers" add @add="(s.peers ??= []).push(defaultPeer())">
        <XNestCard v-for="(p, i) in items(s.peers)" :key="i" :title="t('xray.peer', { n: i + 1 })">
          <template #actions>
            <XInlineButton v-if="s.peers.length > 1" icon="delete" danger :label="t('xray.removeItem', { n: i + 1 })" @click="remove(s.peers, i)" />
          </template>
          <XFieldGrid :cols="1">
            <XField v-slot="a" :label="t('xray.endpoint')">
              <XInput v-bind="a" v-model="p.endpoint" mono />
            </XField>
            <XFieldGrid :cols="2">
              <XField v-slot="a" :label="t('xray.publicKey')">
                <XInput v-bind="a" v-model="p.publicKey" mono />
              </XField>
              <XField v-slot="a" :label="t('xray.psk')">
                <XInput v-bind="a" v-model="p.preSharedKey" mono />
              </XField>
            </XFieldGrid>
            <XInputList v-model="p.allowedIPs" title="Allowed IPs" mono />
            <XFieldGrid :cols="3">
              <XField v-slot="a" label="Keep Alive">
                <XNumber v-bind="a" block v-model="p.keepAlive" :min="0" />
              </XField>
            </XFieldGrid>
          </XFieldGrid>
        </XNestCard>
      </XListHead>
    </template>
  </XSection>
</template>
