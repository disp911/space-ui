<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { XButton, XField, XFieldGrid, XInput, XModal, XMultiSelect, XPairList, XSelect, XTagInput, type Option, type PresetGroup } from '../../kit'
import { customGeo, domainPresets, ipPresets } from './catalog'
import { useXrayForm } from './form'
import { listOf, type Rule } from './model'

/**
 * A routing rule to create, or the one at index to edit. Its fields stand in
 * one column, as in the inbound window, and its body scrolls.
 */
const props = defineProps<{ rule: Rule | null; index: number }>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ save: [rule: Rule] }>()
const { t } = useI18n()
const f = useXrayForm()

// --- The form -------------------------------------------------------------------

const blank = () => ({
  sourceIP: [] as string[],
  sourcePort: '',
  vlessRoute: '',
  network: '',
  protocol: [] as string[],
  attrs: [] as [string, string][],
  ip: [] as string[],
  domain: [] as string[],
  user: '',
  port: '',
  inboundTag: [] as string[],
  outboundTag: '',
  balancerTag: '',
})
const form = reactive(blank())
type FormKey = keyof ReturnType<typeof blank>

const text = (v: unknown) => (v == null ? '' : String(v))

watch(open, (on) => {
  if (!on) return
  const r = props.rule ?? {}
  Object.assign(form, blank(), {
    sourceIP: listOf(r.sourceIP ?? r.source),
    sourcePort: text(r.sourcePort),
    vlessRoute: text(r.vlessRoute),
    network: text(r.network),
    protocol: listOf(r.protocol),
    attrs: r.attrs && typeof r.attrs === 'object' ? Object.entries(r.attrs).map(([k, v]) => [k, text(v)]) : [],
    ip: listOf(r.ip),
    domain: listOf(r.domain),
    user: listOf(r.user).join(','),
    port: text(r.port),
    inboundTag: listOf(r.inboundTag),
    outboundTag: text(r.outboundTag),
    balancerTag: text(r.balancerTag),
  })
})

/** The rule as Xray takes it: filled fields only; fields the form does not know stay as they were. */
const result = computed<Rule>(() => {
  const { source: _s, ...kept } = (props.rule ?? {}) as Rule
  const list = (v: string[]) => v.map((s) => s.trim()).filter(Boolean)
  const known: Record<FormKey, unknown> = {
    sourceIP: list(form.sourceIP),
    sourcePort: form.sourcePort.trim(),
    vlessRoute: form.vlessRoute.trim(),
    network: form.network,
    protocol: form.protocol,
    attrs: Object.fromEntries(form.attrs.filter(([k]) => k.trim()).map(([k, v]) => [k.trim(), v])),
    ip: list(form.ip),
    domain: list(form.domain),
    user: list(form.user.split(',')),
    port: form.port.trim(),
    inboundTag: form.inboundTag,
    outboundTag: form.outboundTag,
    balancerTag: form.balancerTag,
  }
  const out: Rule = { ...kept, type: 'field' }
  for (const [key, value] of Object.entries(known)) {
    delete out[key]
    const empty = value === '' || (Array.isArray(value) && !value.length) || (typeof value === 'object' && value && !Object.keys(value).length)
    if (!empty) out[key] = value
  }
  return out
})

/** Something to match or somewhere to send: a rule of nothing but its type is no rule. */
const filled = computed(() => Object.keys(result.value).some((k) => k !== 'type'))

function save() {
  if (!filled.value) return
  emit('save', result.value)
  open.value = false
}

// --- Choices --------------------------------------------------------------------

const none: Option = { value: '', label: '—', plain: true }
const networks: Option[] = [none, ...['TCP', 'UDP', 'TCP,UDP'].map((v) => ({ value: v, label: v }))]
const protocols = ['http', 'tls', 'bittorrent', 'quic'].map((v) => ({ value: v, label: v }))

const config = computed(() => f.draft.config)
const tags = (list: unknown) => [...new Set((Array.isArray(list) ? list : []).map((x) => x?.tag).filter((tag): tag is string => !!tag))]

// A tag the rule already names stays on offer even when nothing in the template carries it any more.
const choices = (list: string[], current: string[]) => [...new Set([...list, ...current.filter(Boolean)])].map((v) => ({ value: v, label: v }))

const inbounds = computed(() => {
  const all = [...tags(config.value.inbounds), ...f.inboundTags]
  if (config.value.dns?.tag) all.push(config.value.dns.tag)
  return choices(all, form.inboundTag)
})
const outbounds = computed<Option[]>(() => [none, ...choices([...tags(config.value.outbounds), ...f.reverseTags], [form.outboundTag])])
const balancers = computed<Option[]>(() => [none, ...choices(tags(config.value.routing?.balancers), [form.balancerTag])])

const custom = ref<{ ip: { value: string; label: string }[]; domain: { value: string; label: string }[] }>({ ip: [], domain: [] })
// The user's geo files, fetched each time the window opens: they may have changed in the Geo files window.
watch(open, async (on) => on && (custom.value = await customGeo(t('xray.customSuffix'))))

const ipGroups = computed<PresetGroup[]>(() => [{ options: [...ipPresets, ...custom.value.ip] }])
const domainGroups = computed<PresetGroup[]>(() => [
  ...domainPresets.map((g) => ({ title: t(g.title), options: g.options })),
  ...(custom.value.domain.length ? [{ title: t('xray.catalogCustom'), options: custom.value.domain }] : []),
])

const title = computed(() => (props.rule ? t('xray.ruleEdit', { n: props.index + 1 }) : t('xray.ruleAdd')))
</script>

<template>
  <XModal v-model:open="open" :title="title" size="md" cancel scroll-body stack>
    <div class="rule">
      <XField v-slot="a" label="Source IPs" :help="t('xray.useComma')">
        <XTagInput v-bind="a" v-model="form.sourceIP" :groups="ipGroups" mono placeholder="e.g. 0.0.0.0/8, fc00::/7, geoip:ir" />
      </XField>
      <XFieldGrid>
        <XField v-slot="a" label="Source Port" :help="t('xray.useComma')">
          <XInput v-bind="a" v-model="form.sourcePort" mono placeholder="53,443,1000-2000" />
        </XField>
        <XField v-slot="a" label="VLESS Route" :help="t('xray.useComma')">
          <XInput v-bind="a" v-model="form.vlessRoute" mono placeholder="53,443,1000-2000" />
        </XField>
      </XFieldGrid>

      <hr class="rule__line" />

      <XFieldGrid>
        <XField v-slot="a" label="Network">
          <XSelect v-bind="a" v-model="form.network" :options="networks" mono />
        </XField>
        <XField v-slot="a" label="Protocol">
          <XMultiSelect v-bind="a" v-model="form.protocol" :options="protocols" mono />
        </XField>
      </XFieldGrid>
      <XPairList v-model="form.attrs" title="Attributes" :name-label="t('xray.attrName')" :value-label="t('xray.attrValue')" mono />

      <hr class="rule__line" />

      <XField v-slot="a" label="IP" :help="t('xray.useComma')">
        <XTagInput v-bind="a" v-model="form.ip" :groups="ipGroups" mono placeholder="e.g. 0.0.0.0/8, fc00::/7, geoip:ir" />
      </XField>
      <XField v-slot="a" label="Domain" :help="t('xray.useComma')">
        <XTagInput v-bind="a" v-model="form.domain" :groups="domainGroups" mono placeholder="e.g. google.com, geosite:cn" />
      </XField>
      <XFieldGrid>
        <XField v-slot="a" label="User" :help="t('xray.useComma')">
          <XInput v-bind="a" v-model="form.user" placeholder="email address" />
        </XField>
        <XField v-slot="a" label="Port" :help="t('xray.useComma')">
          <XInput v-bind="a" v-model="form.port" mono placeholder="53,443,1000-2000" />
        </XField>
      </XFieldGrid>

      <hr class="rule__line" />

      <XField v-slot="a" label="Inbound Tags">
        <XMultiSelect v-bind="a" v-model="form.inboundTag" :options="inbounds" mono />
      </XField>
      <XFieldGrid>
        <XField v-slot="a" label="Outbound Tag">
          <XSelect v-bind="a" v-model="form.outboundTag" :options="outbounds" mono />
        </XField>
        <XField v-slot="a" label="Balancer Tag" :help="t('xray.balancerDesc')">
          <XSelect v-bind="a" v-model="form.balancerTag" :options="balancers" mono />
        </XField>
      </XFieldGrid>
    </div>

    <template #footer>
      <XButton variant="primary" :disabled="!filled" @click="save">{{ t('xray.save') }}</XButton>
    </template>
  </XModal>
</template>

<style scoped>
.rule {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}

.rule__line {
  margin: 4px 0;
  border: 0;
  border-top: 1px solid var(--div);
}
</style>
