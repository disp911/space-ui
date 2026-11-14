<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { XButton, XCard, XChip, XEmpty, XGridTable, XRouteTarget, XValueChip, type GridColumn, type MenuItem } from '../../kit'
import { useXrayForm } from './form'
import { ruleGroups, ruleTarget, type Rule } from './model'
import RuleModal from './RuleModal.vue'
import GeoModal from './GeoModal.vue'

/** The routing rules in their order: Xray takes the first that matches. */
const { t } = useI18n()
const f = useXrayForm()

const rules = computed<Rule[]>(() => f.draft.config.routing?.rules ?? [])

const columns = computed<GridColumn[]>(() => [
  ...ruleGroups.map((g) => ({ key: g.key, label: t(g.label) })),
  { key: 'target', label: t('xray.ruleOutbound'), size: '.85fr', phone: 'head' as const },
])

/** The parts of a group a rule fills in, each with its values. */
const chips = (r: Rule, key: string) =>
  (ruleGroups.find((g) => g.key === key)?.parts ?? []).map(([label, read]) => ({ label, values: read(r) })).filter((c) => c.values.length)

// A rule with no target is not blank but wrong: Xray will not start with it.
const blank = (r: Rule, key: string) => key !== 'target' && !chips(r, key).length

const menu = (): MenuItem[] => [
  { key: 'edit', label: t('xray.edit'), icon: 'edit' },
  { key: 'delete', label: t('xray.delete'), icon: 'delete', danger: true },
]

function move(from: number, to: number) {
  const list = f.draft.config.routing.rules as Rule[]
  list.splice(to, 0, list.splice(from, 1)[0]!)
}

// The rule window: a new rule, or the one at editing.
const ruleOpen = ref(false)
const geoOpen = ref(false)
const editing = ref(-1)

function create() {
  editing.value = -1
  ruleOpen.value = true
}

function select(key: string, _row: Rule, index: number) {
  if (key === 'delete') (f.draft.config.routing.rules as Rule[]).splice(index, 1)
  else if (key === 'edit') {
    editing.value = index
    ruleOpen.value = true
  }
}

function saveRule(rule: Rule) {
  const routing = (f.draft.config.routing ??= {})
  const list = (routing.rules ??= []) as Rule[]
  if (editing.value >= 0) list[editing.value] = rule
  else list.push(rule)
}
</script>

<template>
  <div class="xray-tab">
    <div class="xray-tab__tools">
      <XButton variant="primary" icon="plus" class="xray-tab__main" @click="create">{{ t('xray.ruleAdd') }}</XButton>
      <XButton icon="globe" phone-icon-only @click="geoOpen = true">{{ t('xray.geoFiles') }}</XButton>
    </div>

    <XGridTable
      v-if="rules.length"
      :columns="columns"
      :rows="rules"
      :row-key="(_, i) => i"
      :blank="blank"
      :menu="menu"
      :row-label="(i) => t('xray.ruleN', { n: i + 1 })"
      sortable
      numbered
      @move="move"
      @select="select"
    >
      <template v-for="g in ruleGroups" :key="g.key" #[`cell-${g.key}`]="{ row }">
        <XValueChip v-for="c in chips(row, g.key)" :key="c.label" :label="c.label" :values="c.values" />
      </template>
      <template #cell-target="{ row }">
        <XRouteTarget v-if="ruleTarget(row)" :tag="ruleTarget(row)!.tag" :balancer="ruleTarget(row)!.kind === 'balancer'" />
        <XChip v-else tone="err">{{ t('xray.ruleNoTarget') }}</XChip>
      </template>
    </XGridTable>
    <XCard v-else>
      <XEmpty icon="rules" :title="t('xray.ruleEmpty')" tab />
    </XCard>

    <GeoModal v-model:open="geoOpen" />
    <RuleModal v-model:open="ruleOpen" :rule="editing >= 0 ? rules[editing] ?? null : null" :index="editing" @save="saveRule" />
  </div>
</template>

<style scoped>
.xray-tab__tools {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

@media (max-width: 767px) {
  .xray-tab__main {
    flex: 1;
  }
}
</style>
