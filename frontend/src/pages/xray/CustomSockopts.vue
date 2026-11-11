<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { XField, XFieldGrid, XInlineButton, XInput, XListHead, XNestCard, XSelect, type Option } from '../../kit'
import { optWrong } from './sockopt'

/**
 * Socket options Xray has no field of its own for, set by their number, e.g.
 * level 6 (TCP) opt 13 (TCP_LINGER2). The inbound's and the outbound's
 * sockopt have them; an emptied list leaves the template.
 */
const props = defineProps<{ so: Record<string, any> }>()
const { t } = useI18n()

const items = computed<Record<string, any>[]>(() => (Array.isArray(props.so.customSockopt) ? props.so.customSockopt : []))

function add() {
  props.so.customSockopt = [...items.value, { system: '', network: '', level: '', opt: '', value: '', type: 'int' }]
}
function remove(i: number) {
  const left = items.value.filter((_, j) => j !== i)
  if (left.length) props.so.customSockopt = left
  else delete props.so.customSockopt
}

const any: Option = { value: '', label: 'Any', plain: true }
const systems: Option[] = [any, ...['linux', 'windows', 'darwin'].map((v) => ({ value: v, label: v }))]
const networks: Option[] = [any, ...['tcp', 'udp'].map((v) => ({ value: v, label: v }))]
const types: Option[] = ['int', 'str'].map((v) => ({ value: v, label: v }))
</script>

<template>
  <XListHead title="Custom Sockopt" add @add="add">
    <XNestCard v-for="(c, i) in items" :key="i" :title="String(i + 1)">
      <template #actions>
        <XInlineButton icon="delete" danger :label="t('xray.removeItem', { n: i + 1 })" @click="remove(i)" />
      </template>
      <XFieldGrid :cols="3">
        <XField v-slot="a" label="System"><XSelect v-bind="a" :model-value="c.system ?? ''" :options="systems" mono @update:model-value="c.system = String($event ?? '')" /></XField>
        <XField v-slot="a" label="Network"><XSelect v-bind="a" :model-value="c.network ?? ''" :options="networks" mono @update:model-value="c.network = String($event ?? '')" /></XField>
        <XField v-slot="a" label="Type"><XSelect v-bind="a" :model-value="c.type ?? 'int'" :options="types" mono @update:model-value="c.type = String($event ?? 'int')" /></XField>
        <XField v-slot="a" label="Level"><XInput v-bind="a" v-model.trim="c.level" mono placeholder="6" :invalid="!!c.level && !/^\d+$/.test(c.level)" /></XField>
        <XField v-slot="a" label="Opt"><XInput v-bind="a" v-model.trim="c.opt" mono placeholder="13" :invalid="optWrong(c)" /></XField>
        <XField v-slot="a" label="Value"><XInput v-bind="a" v-model="c.value" mono :invalid="c.type !== 'str' && !/^-?\d+$/.test(String(c.value ?? '').trim())" /></XField>
      </XFieldGrid>
    </XNestCard>
  </XListHead>
</template>
