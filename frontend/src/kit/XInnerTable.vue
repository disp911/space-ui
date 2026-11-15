<script setup lang="ts" generic="T">
import { computed } from 'vue'
import { useCompact } from './media'

export interface InnerColumn {
  key: string
  label: string
  /** Grid track, e.g. '72px'; minmax(0, 1fr) when left out. */
  size?: string
  /** Label and content at the end of the cell, e.g. a button at the row's right. */
  align?: 'end'
  /** What the column holds is plain to see, e.g. a menu or a switch: the label is read out but not shown. */
  quiet?: boolean
}

/**
 * The list a row of XExpandTable opens onto, e.g. an inbound's clients: a caps
 * header and one-line rows split by thin lines, with room for a pager below.
 * Phones get their own columns, and so does a narrow XExpandTable around it. A
 * cell shows the `cell-<key>` slot on desktop and `phone-<key>` on phones.
 */
const props = defineProps<{
  columns: InnerColumn[]
  phoneColumns: InnerColumn[]
  rows: T[]
  rowKey: (row: T, index: number) => string | number
}>()
const phone = useCompact()

const columns = computed(() => (phone.value ? props.phoneColumns : props.columns))
const template = computed(() => columns.value.map((c) => c.size ?? 'minmax(0, 1fr)').join(' '))
</script>

<template>
  <div :class="['x-inner', { 'x-inner--phone': phone }]" role="table">
    <div class="x-inner__head x-caps" role="row">
      <span v-for="c in columns" :key="c.key" :class="{ 'is-end': c.align === 'end' }" role="columnheader"><span :class="{ 'sr-only': c.quiet }">{{ c.label }}</span></span>
    </div>
    <div v-for="(row, i) in rows" :key="rowKey(row, i)" class="x-inner__row" role="row">
      <div v-for="c in columns" :key="c.key" :class="['x-inner__cell', { 'is-end': c.align === 'end' }]" role="cell">
        <slot :name="`${phone ? 'phone' : 'cell'}-${c.key}`" :row="row" :index="i" />
      </div>
    </div>
    <div v-if="$slots.foot" class="x-inner__foot"><slot name="foot" /></div>
  </div>
</template>

<style scoped>
.x-inner {
  min-width: 0;
}

.x-inner__head,
.x-inner__row {
  display: grid;
  grid-template-columns: v-bind(template);
  gap: 12px;
  align-items: center;
}

.x-inner__head {
  padding: 9px 12px;
  white-space: nowrap;
}

.x-inner__head .is-end {
  text-align: end;
}

.x-inner__row {
  padding: 7px 12px;
  border-top: 1px solid var(--div);
  transition: background-color var(--t-fast);
}

.x-inner__row:hover {
  background: var(--hov);
}

.x-inner__cell {
  min-width: 0;
}

.x-inner__cell.is-end {
  display: flex;
  justify-content: flex-end;
}

.x-inner__foot {
  padding: 9px 12px;
  border-top: 1px solid var(--div);
}

.x-inner--phone .x-inner__head {
  /* Room between the menu button and a status dot with its pulse ring. */
  gap: 8px;
  padding: 8px 8px 8px 10px;
}

.x-inner--phone .x-inner__row {
  box-sizing: content-box;
  gap: 8px;
  min-height: 54px;
  padding: 2px 8px 2px 4px;
}

.x-inner--phone .x-inner__row:hover {
  background: transparent;
}
</style>
