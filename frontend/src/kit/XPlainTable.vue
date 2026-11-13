<script setup lang="ts" generic="T">
import { computed } from 'vue'
import XIcon from './XIcon.vue'
import { usePhone } from './media'

export interface PlainColumn {
  key: string
  /** The header; when no column has one the table is a list without a header. */
  label?: string
  /** As wide as its content, e.g. a row's buttons. */
  fit?: boolean
}

/**
 * A framed table of a few rows, e.g. inside a window: a header in plain words
 * and rows split by thin lines; without labels a list of the same look. A row
 * at work, e.g. downloading, is marked by active. A cell shows the
 * `cell-<key>` slot and a header the `head-<key>` slot, e.g. a checkbox for
 * every row; the cells of fitted columns line up their buttons. With
 * no rows the empty slot shows, an empty tray by default.
 *
 * On phones a table turns into a card per row: the first column on top, the
 * others under it with their labels and the fitted ones at the end of that
 * line. A list stays a list with taller rows. With a height the table scrolls
 * under its header, e.g. hundreds of clients, and stays a table on phones.
 */
const props = defineProps<{
  columns: PlainColumn[]
  rows: T[]
  rowKey: (row: T, index: number) => string | number
  active?: (row: T) => boolean
  /** The most it grows to before it scrolls, in pixels; phoneHeight on phones. */
  height?: number
  phoneHeight?: number
}>()

const phone = usePhone()
const head = computed(() => props.columns.some((c) => c.label))
const cards = computed(() => phone.value && head.value && !props.height)
const maxHeight = computed(() => {
  const h = phone.value ? (props.phoneHeight ?? props.height) : props.height
  return h ? `${h}px` : undefined
})
const first = computed(() => props.columns[0]!)
const labelled = computed(() => props.columns.slice(1).filter((c) => !c.fit))
const fitted = computed(() => props.columns.slice(1).filter((c) => c.fit))
</script>

<template>
  <div v-if="cards" class="x-ptable-cards">
    <div v-if="!rows.length" class="x-ptable-cards__empty">
      <slot name="empty"><XIcon name="inbox" :size="16" :stroke="1.3" /></slot>
    </div>
    <div v-for="(row, i) in rows" :key="rowKey(row, i)" :class="['x-ptable-card', { 'is-active': active?.(row) }]">
      <div class="x-ptable-card__title"><slot :name="`cell-${first.key}`" :row="row" /></div>
      <div class="x-ptable-card__line">
        <span v-for="c in labelled" :key="c.key" class="x-ptable-card__field">
          <span class="x-ptable-card__label">{{ c.label }}</span>
          <slot :name="`cell-${c.key}`" :row="row" />
        </span>
        <span class="x-ptable__tools x-ptable-card__end">
          <template v-for="c in fitted" :key="c.key"><slot :name="`cell-${c.key}`" :row="row" /></template>
        </span>
      </div>
    </div>
  </div>

  <div v-else :class="['x-ptable', { 'x-ptable--list': !head, 'x-ptable--phone': phone, 'x-ptable--scroll': !!maxHeight }]" :style="{ maxHeight }">
    <table>
      <thead v-if="head">
        <tr>
          <th v-for="c in columns" :key="c.key" :class="{ 'x-ptable__fit': c.fit }"><slot :name="`head-${c.key}`">{{ c.label }}</slot></th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="!rows.length">
          <td :colspan="columns.length" class="x-ptable__empty">
            <slot name="empty"><XIcon name="inbox" :size="16" :stroke="1.3" /></slot>
          </td>
        </tr>
        <tr v-for="(row, i) in rows" :key="rowKey(row, i)" :class="{ 'is-active': active?.(row) }">
          <td v-for="c in columns" :key="c.key" :class="{ 'x-ptable__fit': c.fit }">
            <div v-if="c.fit" class="x-ptable__tools"><slot :name="`cell-${c.key}`" :row="row" /></div>
            <slot v-else :name="`cell-${c.key}`" :row="row" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.x-ptable {
  min-width: 0;
  overflow: clip;
  border: 1px solid var(--div);
  border-radius: var(--nest-r);
  background: var(--card);
}

.x-ptable table {
  width: 100%;
  border-collapse: collapse;
}

.x-ptable th {
  padding: var(--sp-8) var(--sp-12);
  border-bottom: 1px solid var(--div);
  background: var(--ctrl);
  color: var(--mut);
  font: 600 var(--fs-12) / var(--lh-tight) var(--font-sans);
  text-align: left;
  white-space: nowrap;
}

.x-ptable td {
  padding: var(--sp-6) var(--sp-12);
  color: var(--fg2);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-sans);
  vertical-align: middle;
  /* A long value, e.g. an email, breaks rather than pushing the table wider than a phone. */
  overflow-wrap: anywhere;
}

/* A checkbox in a cell sits in its middle, not on the text line, which would add to the row. */
.x-ptable :deep(.x-check) {
  vertical-align: middle;
}

.x-ptable tbody tr + tr td {
  border-top: 1px solid var(--div);
}

.x-ptable tr.is-active,
.x-ptable-card.is-active {
  background: var(--tint);
}

/* A fitted column holds inline buttons: its label stands over the first icon. */
.x-ptable th.x-ptable__fit {
  padding-left: calc(var(--sp-8) + var(--inline-inset));
}

.x-ptable td.x-ptable__fit {
  width: 1%;
  padding: var(--sp-4) var(--sp-8);
  white-space: nowrap;
}

.x-ptable__tools {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}

.x-ptable td.x-ptable__empty {
  height: 96px;
  color: var(--dim);
  text-align: center;
}

.x-ptable__empty > :deep(*) {
  display: inline-block;
  vertical-align: middle;
}

/* A long table scrolls under its header; room for the scrollbar stays, so nothing moves when it shows. */
.x-ptable--scroll {
  overflow: auto;
  scrollbar-gutter: stable;
}

.x-ptable--scroll th {
  position: sticky;
  top: 0;
  z-index: 1;
}

/* A list: rows of a set height, the buttons close to the edge. */
.x-ptable--list tr {
  height: 40px;
}

.x-ptable--list td {
  padding: 0 var(--sp-12);
}

.x-ptable--list td.x-ptable__fit {
  padding: 0 var(--sp-6);
}

/* Phones: a list with taller rows and buttons of the regular square size. */
.x-ptable--phone {
  border-radius: var(--nest-r);
}

.x-ptable--phone.x-ptable--list tr {
  height: 50px;
}

/* A checkbox in a row gets a finger's room: the control height. */
.x-ptable--phone td :deep(.x-check) {
  min-height: var(--ctl-h);
}

.x-ptable--phone td:has(.x-check) {
  padding-block: var(--sp-4);
}

.x-ptable--phone :deep(.x-inline-btn),
.x-ptable-card :deep(.x-inline-btn) {
  width: var(--btn-sq);
  height: var(--btn-sq);
}

/* Phones: a card per row. */
.x-ptable-cards {
  display: flex;
  flex-direction: column;
  gap: var(--sp-8);
  min-width: 0;
}

.x-ptable-cards__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 96px;
  border: 1px dashed var(--field-line);
  border-radius: var(--nest-r);
  color: var(--dim);
}

.x-ptable-card {
  display: flex;
  flex-direction: column;
  gap: var(--sp-6);
  min-width: 0;
  padding: var(--sp-12) var(--sp-6) var(--sp-6) var(--sp-12);
  border: 1px solid var(--div);
  border-radius: var(--nest-r);
  background: var(--card);
}

.x-ptable-card__title {
  min-width: 0;
  padding-right: var(--sp-6);
}

.x-ptable-card__line {
  display: flex;
  align-items: center;
  gap: var(--sp-8);
  min-width: 0;
}

.x-ptable-card__field {
  display: flex;
  align-items: center;
  gap: var(--sp-8);
  min-width: 0;
}

.x-ptable-card__label {
  flex: none;
  color: var(--dim);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-sans);
}

.x-ptable-card__end {
  margin-left: auto;
}
</style>
