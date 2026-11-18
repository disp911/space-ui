<script setup lang="ts" generic="T">
import { computed } from 'vue'

export interface TableColumn {
  key: string
  label: string
  /** Width in px; a column without one takes the rest. */
  width?: number
  /**
   * The least width in px. A column with a width gives up room down to it when
   * the table is narrow, so the column without one keeps its own least width,
   * e.g. the most telling value; without it a column with a width keeps it.
   * Narrower still, every column shrinks by the same share of these widths and
   * cuts its values with an ellipsis: the table never scrolls sideways.
   */
  min?: number
  /** Secondary values, e.g. time: dimmed. */
  dim?: boolean
  /** Key values, e.g. a client: heavier. */
  strong?: boolean
}

export type RowTone = 'err' | 'upload'

/**
 * A monospace log table: a caps header and rows of one line that scroll inside
 * a box of fixed height. Rows can be tinted by tone. A cell shows row[key], or
 * the `cell-<key>` slot. While loading the rows are grey bars; with no rows the
 * empty slot fills the box.
 */
const props = defineProps<{
  columns: TableColumn[]
  rows: T[]
  rowKey: (row: T, index: number) => string | number
  rowTone?: (row: T) => RowTone | undefined
  loading?: boolean
  /** Height of the scrolling body, px. */
  height?: number
}>()

/** The gap between cells, as .x-table__row has it. */
/** Room between the columns: --table-gap. */
const GAP = 10
const least = (c: TableColumn) => c.min ?? c.width ?? 0
/** What all the columns need at their least, without the gaps. */
const leastTotal = computed(() => props.columns.reduce((sum, c) => sum + least(c), 0) || 1)
/*
 * A column's least width, or its share of the row when the row has less than
 * they all need, so the row never runs past its box.
 */
function minWidth(c: TableColumn) {
  const gaps = GAP * (props.columns.length - 1)
  return `min(${least(c)}px, (100% - ${gaps}px) * ${(least(c) / leastTotal.value).toFixed(4)})`
}
const cellStyle = (c: TableColumn) =>
  c.width ? { flex: `0 1 ${c.width}px`, minWidth: minWidth(c) } : { flex: '1 1 0', minWidth: minWidth(c) }
const value = (row: T, key: string) => (row as Record<string, unknown>)[key] as string

// Bar widths vary like real values.
const skeleton = [72, 58, 80, 64, 50, 76, 60, 68, 54, 74, 62, 70, 56, 66]
</script>

<template>
  <div class="x-table" role="table">
    <div class="x-table__head x-table-head" role="row">
      <span v-for="c in columns" :key="c.key" class="x-table__cell x-ellipsis" role="columnheader" :style="cellStyle(c)">{{ c.label }}</span>
    </div>
    <div class="x-table__body" :style="{ height: height ? `${height}px` : undefined }" tabindex="0" :aria-busy="loading || undefined">
      <div v-if="loading" class="x-skeleton" aria-hidden="true">
        <div v-for="(w, i) in skeleton" :key="i" class="x-table__row">
          <span v-for="c in columns" :key="c.key" class="x-table__cell" :style="cellStyle(c)">
            <span class="x-skeleton__bar" :style="{ width: `${c.width ? 70 : w}%` }" />
          </span>
        </div>
      </div>
      <div v-else-if="!rows.length" class="x-table__empty"><slot name="empty" /></div>
      <template v-else>
        <div
          v-for="(row, i) in rows"
          :key="rowKey(row, i)"
          :class="['x-table__row', rowTone?.(row) && `x-table__row--${rowTone(row)}`]"
          role="row"
        >
          <span
            v-for="c in columns"
            :key="c.key"
            :class="['x-table__cell', 'x-ellipsis', { 'x-table__cell--dim': c.dim, 'x-table__cell--strong': c.strong }]"
            role="cell"
            :style="cellStyle(c)"
          >
            <slot :name="`cell-${c.key}`" :row="row">{{ value(row, c.key) }}</slot>
          </span>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.x-table {
  display: flex;
  flex-direction: column;
  min-width: 0;
  border: 1px solid var(--line);
  border-radius: var(--nest-r);
  background: var(--bg);
  clip-path: inset(0 round var(--nest-r));
}

.x-table__head,
.x-table__row {
  display: flex;
  align-items: center;
  gap: var(--table-gap);
  padding: 0 var(--table-px);
  white-space: nowrap;
}

/* Body and head keep room for the scrollbar, so rows do not move when it appears and stay under their column names. */
.x-table__head {
  flex: none;
  height: var(--table-head-h);
  overflow: hidden;
  scrollbar-gutter: stable;
  border-bottom: 1px solid var(--div);
  background: var(--table-head-bg);
}

.x-table__body {
  overflow: hidden auto;
  scrollbar-gutter: stable;
  outline: none;
}

/* A dense row, as a log's. */
.x-table__row {
  height: var(--table-row-dense-h);
  border-bottom: 1px solid var(--div);
  color: var(--fg2);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-mono);
  transition: background-color var(--t-fast);
}

.x-table__row:hover {
  background-color: var(--hov);
}

.x-table__row--err {
  background-color: color-mix(in srgb, var(--err) 6%, transparent);
  color: var(--err-t);
}

.x-table__row--upload {
  background-color: color-mix(in srgb, var(--upload) 6%, transparent);
  color: var(--upload-t);
}

.x-table__cell--dim {
  opacity: .7;
}

.x-table__cell--strong {
  font-weight: 500;
}

.x-table__empty {
  display: flex;
  flex-direction: column;
  height: 100%;
}
</style>
