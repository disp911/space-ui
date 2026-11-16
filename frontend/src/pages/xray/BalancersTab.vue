<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { checkJson, XButton, XCard, XChip, XCodeEditor, XEmpty, XGridTable, XSegmented, type GridColumn, type MenuItem } from '../../kit'
import { useXrayForm } from './form'
import { listOf, strategies, strategyOf, syncObservatories, type Balancer, type Rule } from './model'
import BalancerModal from './BalancerModal.vue'

/**
 * The balancers of the routing and, under them, the observers that test their
 * outbounds: Observatory for Least Ping, Burst Observatory for the others.
 */
const { t } = useI18n()
const f = useXrayForm()

const balancers = computed<Balancer[]>(() => f.draft.config.routing?.balancers ?? [])

const columns = computed<GridColumn[]>(() => [
  { key: 'tag', label: t('xray.outTag'), phone: 'head' },
  { key: 'strategy', label: t('xray.balStrategy'), size: '150px', phone: 'head' },
  { key: 'selector', label: t('xray.balSelectors'), size: '2.4fr' },
])

const strategyLabel = (b: Balancer) => strategies.find((s) => s.value === strategyOf(b))?.label ?? strategyOf(b)

const menu = (): MenuItem[] => [
  { key: 'edit', label: t('xray.edit'), icon: 'edit' },
  { key: 'delete', label: t('xray.delete'), icon: 'delete', danger: true },
]

const rules = () => (f.draft.config.routing?.rules ?? []) as Rule[]

// The balancer window: a new balancer, or the one at editing.
const formOpen = ref(false)
const editing = ref(-1)

function create() {
  editing.value = -1
  formOpen.value = true
}

function select(key: string, row: Balancer, index: number) {
  if (key === 'edit') {
    editing.value = index
    formOpen.value = true
  } else if (key === 'delete') {
    const routing = f.draft.config.routing
    routing.balancers.splice(index, 1)
    if (!routing.balancers.length) delete routing.balancers
    // Rules that sent traffic to it lose their target, as on the old panel.
    for (const r of rules()) if (r.balancerTag === row.tag) delete r.balancerTag
    syncObservatories(f.draft.config)
  }
}

function saveBalancer(b: Balancer) {
  const routing = (f.draft.config.routing ??= {})
  const list = (routing.balancers ??= []) as Balancer[]
  if (editing.value >= 0) {
    const before = list[editing.value]?.tag
    list[editing.value] = b
    // A renamed balancer takes its rules along.
    if (before && before !== b.tag) for (const r of rules()) if (r.balancerTag === before) r.balancerTag = b.tag
  } else list.push(b)
  syncObservatories(f.draft.config)
}

// --- Observers ------------------------------------------------------------------

type Observer = 'observatory' | 'burstObservatory'
const observerLabels: Record<Observer, string> = { observatory: 'Observatory', burstObservatory: 'Burst Observatory' }

const observers = computed(() => (['observatory', 'burstObservatory'] as Observer[]).filter((k) => f.draft.config[k] !== undefined))
const picked = ref<Observer>('observatory')
const observer = computed<Observer | undefined>(() => (observers.value.includes(picked.value) ? picked.value : observers.value[0]))
const observerOptions = computed(() => observers.value.map((k) => ({ value: k, label: observerLabels[k] })))

/** The chosen observer as JSON under its own key; text that parses goes into the template. */
const shown = computed(() => (observer.value ? { [observer.value]: f.draft.config[observer.value] } : null))
const text = ref('')
const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b)

watch(text, (value) => {
  const key = observer.value
  if (!key || checkJson(value)) return
  const parsed = JSON.parse(value)
  const next = parsed && typeof parsed === 'object' ? parsed[key] : undefined
  if (next && typeof next === 'object' && !same(next, f.draft.config[key])) f.draft.config[key] = next
})

// The observer changed elsewhere, or the other one is chosen: the editor shows it,
// unless the text already says the same in its own layout.
watch(
  [shown, observer],
  ([value], [, before]) => {
    if (!value) return
    if (observer.value === before && !checkJson(text.value) && same(JSON.parse(text.value), value)) return
    text.value = JSON.stringify(value, null, 2)
  },
  { deep: true, immediate: true },
)
</script>

<template>
  <div class="xray-tab">
    <template v-if="balancers.length">
      <div class="xray-tab__tools">
        <XButton variant="primary" icon="plus" class="xray-tab__main" @click="create">{{ t('xray.balAdd') }}</XButton>
      </div>

      <XGridTable
        :columns="columns"
        :rows="balancers"
        :row-key="(_, i) => i"
        :menu="menu"
        :row-label="(i) => t('xray.balN', { n: i + 1 })"
        numbered
        @select="select"
      >
        <template #cell-tag="{ row }">
          <span class="balancers__tag x-ellipsis" :title="row.tag">{{ row.tag }}</span>
        </template>
        <template #cell-strategy="{ row }">
          <XChip tone="ok">{{ strategyLabel(row) }}</XChip>
        </template>
        <template #cell-selector="{ row }">
          <span class="balancers__chips">
            <XChip v-for="s in listOf(row.selector)" :key="s" mono>{{ s }}</XChip>
          </span>
        </template>
      </XGridTable>
    </template>
    <XCard v-else>
      <XEmpty icon="balancer" :title="t('xray.balEmpty')" tab>
        <XButton variant="primary" icon="plus" @click="create">{{ t('xray.balAdd') }}</XButton>
      </XEmpty>
    </XCard>

    <XCard v-if="observer" :title="observers.length > 1 ? undefined : observerLabels[observer]" seamless class="balancers__observer">
      <template v-if="observers.length > 1" #heading>
        <XSegmented v-model="picked" :options="observerOptions" :label="t('xray.balObservers')" />
      </template>
      <XCodeEditor v-model="text" :height="280" :phone-height="280" fit :label="observerLabels[observer]" />
    </XCard>

    <BalancerModal v-model:open="formOpen" :balancer="editing >= 0 ? balancers[editing] ?? null : null" :index="editing" @save="saveBalancer" />
  </div>
</template>

<style scoped>
.xray-tab__tools {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

/* On phones the tag fills the card's head and the strategy keeps to its right. */
.balancers__tag {
  flex: 1;
  color: var(--fg);
  font: 500 var(--fs-13) / var(--lh-tight) var(--font-mono);
}

.balancers__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  min-width: 0;
}

.balancers__observer {
  margin-top: 16px;
}

@media (max-width: 767px) {
  .xray-tab__main {
    flex: 1;
  }
}
</style>
