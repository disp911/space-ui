<script setup lang="ts" generic="T">
export interface TableColumn {
  key: string
  label: string
  /** Fixed width in px; a column without one takes the rest. */
  width?: number
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

const cellStyle = (c: TableColumn) => (c.width ? { width: `${c.width}px`, flex: 'none' } : { flex: '1', minWidth: '0' })
const value = (row: T, key: string) => (row as Record<string, unknown>)[key] as string

// Bar widths vary like real values.
const skeleton = [72, 58, 80, 64, 50, 76, 60, 68, 54, 74, 62, 70, 56, 66]
</script>

<template>
  <div class="x-table" role="table">
    <div class="x-table__head x-caps" role="row">
      <span v-for="c in columns" :key="c.key" class="x-table__cell" role="columnheader" :style="cellStyle(c)">{{ c.label }}</span>
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
            :class="['x-table__cell', { 'x-table__cell--dim': c.dim, 'x-table__cell--strong': c.strong }]"
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
  gap: 14px;
  padding: 0 14px;
  white-space: nowrap;
}

/* Body and head keep room for the scrollbar, so rows do not move when it appears and stay under their column names. */
.x-table__head {
  flex: none;
  height: 34px;
  overflow: hidden;
  scrollbar-gutter: stable;
  border-bottom: 1px solid var(--line);
  background: var(--field);
}

.x-table__body {
  overflow: auto;
  scrollbar-gutter: stable;
  outline: none;
}

.x-table__row {
  height: 34px;
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

.x-table__cell {
  overflow: hidden;
  text-overflow: ellipsis;
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
