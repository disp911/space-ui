<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  XButton, XField, XFieldGrid, XInlineButton, XInput, XListHead, XModal, XNest, XNestCard, XNumber, XSelect, XSwitch, XTagInput, type Option, type PresetGroup,
} from '../../kit'
import { useXrayForm } from './form'
import { listOf, strategies, strategyOf, type Balancer } from './model'

/**
 * A balancer to create, or the one at index to edit: its tag, how it picks an
 * outbound, the outbounds it picks from and where traffic goes when none is up.
 * An empty or taken tag and an empty selector list are marked from the start.
 * leastLoad has its settings under it: how many outbounds to keep, the
 * delays it measures them by and what each one costs.
 */
const props = defineProps<{ balancer: Balancer | null; index: number }>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ save: [balancer: Balancer] }>()
const { t } = useI18n()
const f = useXrayForm()

const form = reactive({ tag: '', strategy: 'random', selector: [] as string[], fallbackTag: '' })

type Cost = { regexp: boolean; match: string; value: number | null }
/** leastLoad's settings as Xray reads them; empty ones are left out. */
const ll = reactive({ expected: null as number | null, maxRTT: '', tolerance: null as number | null, baselines: [] as string[], costs: [] as Cost[] })

/** A duration as Go reads one, e.g. 1s, 300ms or 1m30s. */
const isDuration = (v: string) => /^(\d+(\.\d+)?(ns|us|µs|ms|s|m|h))+$/.test(v.trim())
const rttWrong = computed(() => !!ll.maxRTT.trim() && !isDuration(ll.maxRTT))
const baselinesWrong = computed(() => ll.baselines.some((b) => !isDuration(b)))
const llWrong = computed(() => form.strategy === 'leastLoad' && (rttWrong.value || baselinesWrong.value))

watch(open, (on) => {
  if (!on) return
  const b = props.balancer ?? {}
  Object.assign(form, {
    tag: String(b.tag ?? ''),
    strategy: strategyOf(b),
    selector: listOf(b.selector),
    fallbackTag: String(b.fallbackTag ?? ''),
  })
  const s = (b.strategy?.type === 'leastLoad' ? b.strategy.settings : null) ?? {}
  Object.assign(ll, {
    expected: typeof s.expected === 'number' ? s.expected : null,
    maxRTT: String(s.maxRTT ?? ''),
    tolerance: typeof s.tolerance === 'number' ? s.tolerance : null,
    baselines: listOf(s.baselines).map(String),
    costs: (Array.isArray(s.costs) ? s.costs : []).map((c: Record<string, any>) => ({ regexp: !!c.regexp, match: String(c.match ?? ''), value: typeof c.value === 'number' ? c.value : null })),
  })
})

/** leastLoad's settings as they go into the template, beside the ones the form does not know. */
function leastLoadSettings(kept: Record<string, any>) {
  const out: Record<string, any> = { ...kept }
  for (const k of ['expected', 'maxRTT', 'tolerance', 'baselines', 'costs']) delete out[k]
  if (ll.expected) out.expected = ll.expected
  if (ll.maxRTT.trim()) out.maxRTT = ll.maxRTT.trim()
  if (ll.tolerance) out.tolerance = ll.tolerance
  const baselines = ll.baselines.map((b) => b.trim()).filter(Boolean)
  if (baselines.length) out.baselines = baselines
  const costs = ll.costs.filter((c) => c.match.trim()).map((c) => ({ ...(c.regexp ? { regexp: true } : {}), match: c.match.trim(), value: c.value ?? 0 }))
  if (costs.length) out.costs = costs
  return out
}

const tags = (list: unknown) => [...new Set((Array.isArray(list) ? list : []).map((x) => x?.tag).filter((tag): tag is string => !!tag))]

const taken = computed(() => {
  const others = ((f.draft.config.routing?.balancers ?? []) as Balancer[]).filter((_, i) => i !== props.index)
  return tags(others).includes(form.tag.trim())
})
const tagWrong = computed(() => !form.tag.trim() || taken.value)
const selectorEmpty = computed(() => !form.selector.length)

/** The balancer as Xray takes it; a strategy kept as it was keeps its settings, fields the form does not know stay. */
const result = computed<Balancer>(() => {
  const { tag: _t, strategy, selector: _s, fallbackTag: _f, ...kept } = (props.balancer ?? {}) as Balancer
  const out: Balancer = { ...kept, tag: form.tag.trim(), selector: form.selector }
  if (form.fallbackTag) out.fallbackTag = form.fallbackTag
  if (form.strategy === 'leastLoad') {
    const kept = strategy?.type === 'leastLoad' && strategy.settings && typeof strategy.settings === 'object' ? strategy.settings : {}
    const settings = leastLoadSettings(kept)
    out.strategy = Object.keys(settings).length ? { type: 'leastLoad', settings } : { type: 'leastLoad' }
  } else if (strategy && strategyOf({ strategy }) === form.strategy) out.strategy = strategy
  else if (form.strategy !== 'random') out.strategy = { type: form.strategy }
  return out
})

function save() {
  if (tagWrong.value || selectorEmpty.value || llWrong.value) return
  emit('save', result.value)
  open.value = false
}

const outboundTags = computed(() => tags(f.draft.config.outbounds))
const selectorGroups = computed<PresetGroup[]>(() => [{ options: outboundTags.value.map((v) => ({ value: v, label: v })) }])
// A fallback the balancer already names stays on offer even when no outbound carries it any more.
const fallbacks = computed<Option[]>(() => [
  { value: '', label: '—', plain: true },
  ...[...new Set([...outboundTags.value, ...(form.fallbackTag ? [form.fallbackTag] : [])])].map((v) => ({ value: v, label: v })),
])

const title = computed(() => (props.balancer ? t('xray.balEdit') : t('xray.balAdd')))
</script>

<template>
  <XModal v-model:open="open" :title="title" cancel scroll-body>
    <div class="balancer">
      <XField v-slot="a" :label="t('xray.outTag')">
        <XInput v-bind="a" v-model="form.tag" mono :invalid="tagWrong" :placeholder="t('xray.balTagHint')" />
      </XField>
      <XField v-slot="a" :label="t('xray.balStrategy')">
        <XSelect v-bind="a" v-model="form.strategy" :options="strategies" />
      </XField>
      <XNest v-if="form.strategy === 'leastLoad'" bare>
        <XFieldGrid :cols="1">
          <XField v-slot="a" label="Expected" :help="t('xray.balExpectedHelp')"><XNumber v-bind="a" block v-model="ll.expected" :min="0" /></XField>
          <XField v-slot="a" label="Max RTT" :help="t('xray.balMaxRttHelp')"><XInput v-bind="a" v-model="ll.maxRTT" mono placeholder="1s" :invalid="rttWrong" /></XField>
          <XField v-slot="a" label="Tolerance" :help="t('xray.balToleranceHelp')"><XNumber v-bind="a" block decimal v-model="ll.tolerance" :min="0" :max="1" placeholder="0.01" /></XField>
          <XField v-slot="a" label="Baselines" :help="t('xray.balBaselinesHelp')"><XTagInput v-bind="a" v-model="ll.baselines" mono placeholder="300ms" :invalid="baselinesWrong" /></XField>
          <XListHead title="Costs" add @add="ll.costs.push({ regexp: false, match: '', value: 1 })">
            <XNestCard v-for="(c, i) in ll.costs" :key="i" :title="String(i + 1)">
              <template #actions>
                <XInlineButton icon="delete" danger :label="t('xray.removeItem', { n: i + 1 })" @click="ll.costs.splice(i, 1)" />
              </template>
              <XFieldGrid :cols="1">
                <XField v-slot="a" label="Match"><XInput v-bind="a" v-model="c.match" mono /></XField>
                <XField v-slot="a" label="Value"><XNumber v-bind="a" block decimal v-model="c.value" :min="0" /></XField>
                <XSwitch v-model="c.regexp" boxed label="Regexp" />
              </XFieldGrid>
            </XNestCard>
          </XListHead>
        </XFieldGrid>
      </XNest>
      <XField v-slot="a" :label="t('xray.balSelectors')">
        <XTagInput v-bind="a" v-model="form.selector" :groups="selectorGroups" mono :invalid="selectorEmpty" />
      </XField>
      <XField v-slot="a" label="Fallback">
        <XSelect v-bind="a" v-model="form.fallbackTag" :options="fallbacks" mono />
      </XField>
    </div>

    <template #footer>
      <XButton variant="primary" :disabled="tagWrong || selectorEmpty || llWrong" @click="save">{{ balancer ? t('xray.save') : t('xray.balAdd') }}</XButton>
    </template>
  </XModal>
</template>

<style scoped>
.balancer {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}
</style>
