<script setup lang="ts" generic="T">
import { computed, onBeforeUnmount, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import XIcon from './XIcon.vue'
import XMenu, { type MenuItem } from './XMenu.vue'
import { usePhone } from './media'

export interface GridColumn {
  key: string
  label: string
  /** Grid track, e.g. '.85fr' or '120px'; 1fr when left out. */
  size?: string
  /**
   * On phones: in the card's head beside the number, or under it with its
   * label (body, the default), or so in half the width beside another half.
   */
  phone?: 'head' | 'body' | 'half'
}

/**
 * A table of records in a card: a caps header and rows of several lines, e.g.
 * routing rules. A row may lead with a handle to drag it to another place
 * (live reorder; the keyboard takes Space to lift, arrows to move, Space or
 * Escape to drop), its number and a menu of actions. On phones every row
 * becomes a card: the handle, the number, the head columns and the menu on
 * top, the other columns with their labels under it, empty ones left out.
 * A cell shows the `cell-<key>` slot; a blank one shows a dash.
 */
const props = defineProps<{
  columns: GridColumn[]
  rows: T[]
  rowKey: (row: T, index: number) => string | number
  /** A cell with nothing in it. */
  blank?: (row: T, key: string) => boolean
  sortable?: boolean
  numbered?: boolean
  /** The actions of a row; a menu button shows when there are any. */
  menu?: (row: T, index: number) => MenuItem[]
  /** The accessible name of a row's handle and menu, e.g. «Rule 3». */
  rowLabel?: (index: number) => string
  /** Cells sit in the middle of the row rather than at its top, e.g. one-line values beside a two-line tag. */
  center?: boolean
  /** Inside a card, e.g. a settings section: a smaller radius, no surface of its own; on phones darker cards. */
  nested?: boolean
}>()
const emit = defineEmits<{ move: [from: number, to: number]; select: [key: string, row: T, index: number] }>()
const { t } = useI18n()
const phone = usePhone()
const hintId = useId()

const lead = computed(() => props.sortable || props.numbered || !!props.menu)
// The lead holds the handle, the number and the menu; without a handle it is narrower.
// The number column is as wide as the longest number, so «#» and the digits share one centre.
const digits = computed(() => String(Math.max(props.rows.length, 1)).length)
const template = computed(() => [lead.value ? (props.sortable ? '108px' : '76px') : '', ...props.columns.map((c) => `minmax(0, ${c.size ?? '1fr'})`)].filter(Boolean).join(' '))
const headColumns = computed(() => props.columns.filter((c) => c.phone === 'head'))
const bodyColumns = computed(() => props.columns.filter((c) => c.phone !== 'head'))
const isBlank = (row: T, key: string) => props.blank?.(row, key) ?? false
const name = (i: number) => props.rowLabel?.(i) ?? String(i + 1)

// --- Reordering ---------------------------------------------------------------
// The rows keep their place in the page while one is lifted: the lifted row
// follows the pointer and the rows it passes step aside, all by transforms.

const lifted = ref(-1)
/** Where the lifted row would land. */
const target = ref(-1)
const offset = ref(0)
const byKeyboard = ref(false)

const rowEls = new Map<number, HTMLElement>()
const setRow = (i: number) => (el: unknown) => (el ? rowEls.set(i, el as HTMLElement) : rowEls.delete(i))

/** Tops and heights in page coordinates when the row was lifted, and the gap between rows. */
let tops: number[] = []
let heights: number[] = []
let gap = 0
let grab = 0
let pointerY = 0
let frame = 0

function start(i: number) {
  tops = props.rows.map((_, k) => (rowEls.get(k)?.getBoundingClientRect().top ?? 0) + window.scrollY)
  heights = props.rows.map((_, k) => rowEls.get(k)?.offsetHeight ?? 0)
  gap = props.rows.length > 1 ? tops[1]! - tops[0]! - heights[0]! : 0
  lifted.value = i
  target.value = i
  offset.value = 0
}

function finish(keep: boolean) {
  const from = lifted.value
  const to = target.value
  cancelAnimationFrame(frame)
  lifted.value = -1
  target.value = -1
  offset.value = 0
  byKeyboard.value = false
  if (keep && from >= 0 && to >= 0 && from !== to) emit('move', from, to)
}

/** How far the lifted row travels to reach its target slot. */
function slotOffset() {
  const from = lifted.value
  const to = target.value
  let by = 0
  for (let k = Math.min(from, to); k <= Math.max(from, to); k++) if (k !== from) by += heights[k]! + gap
  return to > from ? by : -by
}

/** A row the lifted one has passed steps aside by the lifted row's height. */
function shift(k: number) {
  const from = lifted.value
  const to = target.value
  if (from < 0 || k === from) return 0
  const pitch = heights[from]! + gap
  if (from < to && k > from && k <= to) return -pitch
  if (to < from && k >= to && k < from) return pitch
  return 0
}

// The lifted row's middle against the other rows' middles gives its slot.
function follow() {
  const from = lifted.value
  if (from < 0) return
  const top = pointerY + window.scrollY - grab
  const middle = top + heights[from]! / 2
  target.value = tops.filter((t, k) => k !== from && t + heights[k]! / 2 < middle).length
  offset.value = top - tops[from]!
}

// Near the top or the bottom of the screen the page scrolls on by itself.
function scrollEdges() {
  const h = window.innerHeight
  const by = pointerY < h * 0.25 ? -(h * 0.25 - pointerY) : pointerY > h * 0.85 ? pointerY - h * 0.85 : 0
  if (by) {
    window.scrollBy(0, Math.max(-18, Math.min(18, by / 4)))
    follow()
  }
  frame = requestAnimationFrame(scrollEdges)
}

function onPointerDown(e: PointerEvent, i: number) {
  if (e.button !== 0 || byKeyboard.value) return
  e.preventDefault()
  ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  start(i)
  pointerY = e.clientY
  grab = e.clientY + window.scrollY - tops[i]!
  frame = requestAnimationFrame(scrollEdges)
}

function onPointerMove(e: PointerEvent) {
  if (lifted.value < 0 || byKeyboard.value) return
  pointerY = e.clientY
  follow()
}

function onKeydown(e: KeyboardEvent, i: number) {
  if (e.key === ' ' || e.key === 'Enter') {
    e.preventDefault()
    if (lifted.value < 0) {
      start(i)
      byKeyboard.value = true
    } else finish(true)
  } else if (e.key === 'Escape' && lifted.value >= 0) {
    e.preventDefault()
    finish(false)
  } else if ((e.key === 'ArrowUp' || e.key === 'ArrowDown') && byKeyboard.value) {
    e.preventDefault()
    target.value = Math.max(0, Math.min(props.rows.length - 1, target.value + (e.key === 'ArrowUp' ? -1 : 1)))
    offset.value = slotOffset()
  }
}

function onBlur() {
  if (byKeyboard.value) finish(false)
}

onBeforeUnmount(() => cancelAnimationFrame(frame))

function rowStyle(k: number) {
  if (lifted.value < 0) return undefined
  if (k === lifted.value) return { transform: `translateY(${offset.value}px) scale(1.006)` }
  const by = shift(k)
  return by ? { transform: `translateY(${by}px)` } : undefined
}
</script>

<template>
  <div :class="['x-grid', { 'x-grid--phone': phone, 'x-grid--center': center, 'x-grid--nested': nested, 'is-dragging': lifted >= 0 }]" role="table">
    <span :id="hintId" hidden>{{ t('kit.moveHint') }}</span>

    <div v-if="!phone" class="x-grid__head x-caps" role="row">
      <span v-if="lead" class="x-grid__lead-head" role="columnheader">
        <span v-if="sortable" class="x-grid__grip-room" />
        <span v-if="numbered" class="x-grid__number x-grid__number--head">#</span>
      </span>
      <span v-for="c in columns" :key="c.key" role="columnheader">{{ c.label }}</span>
    </div>

    <div
      v-for="(row, i) in rows"
      :key="rowKey(row, i)"
      :ref="setRow(i)"
      :class="['x-grid__row', { 'is-lifted': i === lifted, 'is-following': i === lifted && !byKeyboard }]"
      :style="rowStyle(i)"
      role="row"
    >
      <div v-if="lead || (phone && headColumns.length)" :class="phone ? 'x-grid__card-head' : 'x-grid__lead'" :role="phone ? undefined : 'cell'">
        <button
          v-if="sortable"
          type="button"
          class="x-grid__grip"
          :aria-label="t('kit.moveRow', { row: name(i) })"
          :aria-pressed="i === lifted"
          :aria-describedby="hintId"
          :title="t('kit.moveHint')"
          @pointerdown="onPointerDown($event, i)"
          @pointermove="onPointerMove"
          @pointerup="finish(true)"
          @pointercancel="finish(false)"
          @keydown="onKeydown($event, i)"
          @blur="onBlur"
        >
          <XIcon name="grip" :size="phone ? 16 : 14" />
        </button>
        <span v-if="numbered" class="x-grid__number">{{ i + 1 }}</span>
        <div v-if="phone" class="x-grid__card-title">
          <template v-for="c in headColumns" :key="c.key">
            <span v-if="isBlank(row, c.key)" class="x-grid__blank">—</span>
            <slot v-else :name="`cell-${c.key}`" :row="row" :index="i" />
          </template>
        </div>
        <XMenu v-if="menu && menu(row, i).length" v-slot="{ props: trigger }" :items="menu(row, i)" @select="emit('select', $event, row, i)">
          <button type="button" class="x-grid__more" v-bind="trigger" :aria-label="t('kit.rowActions', { row: name(i) })">
            <XIcon name="dots" :size="phone ? 16 : 14" />
          </button>
        </XMenu>
      </div>

      <template v-if="!phone">
        <div v-for="c in columns" :key="c.key" class="x-grid__cell" role="cell">
          <span v-if="isBlank(row, c.key)" class="x-grid__blank">—</span>
          <slot v-else :name="`cell-${c.key}`" :row="row" :index="i" />
        </div>
      </template>
      <div v-else-if="bodyColumns.some((c) => !isBlank(row, c.key))" class="x-grid__card-body">
        <div v-for="c in bodyColumns.filter((c) => !isBlank(row, c.key))" :key="c.key" :class="['x-grid__field', { 'x-grid__field--half': c.phone === 'half' }]">
          <div class="x-grid__field-label">{{ c.label }}</div>
          <div class="x-grid__field-value"><slot :name="`cell-${c.key}`" :row="row" :index="i" /></div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Desktop: one card with a header and rows. */
.x-grid {
  min-width: 0;
  border: 1px solid var(--line);
  border-radius: var(--r-16);
  background: var(--card);
}

.x-grid.is-dragging {
  user-select: none;
}

.x-grid__head,
.x-grid:not(.x-grid--phone) .x-grid__row {
  display: grid;
  grid-template-columns: v-bind(template);
  gap: 14px;
  padding: 10px 14px;
}

.x-grid__head {
  border-radius: calc(var(--r-16) - 1px) calc(var(--r-16) - 1px) 0 0;
  background: var(--field);
}

/* The head repeats the lead's layout: room for the handle, then «#» over the numbers. */
.x-grid__lead-head {
  display: flex;
  align-items: center;
  gap: 4px;
}

.x-grid__grip-room {
  flex: none;
  width: 22px;
}

.x-grid:not(.x-grid--phone) .x-grid__row {
  position: relative;
  align-items: start;
  border-top: 1px solid var(--div);
  transition: background-color var(--t-fast);
}

.x-grid:not(.x-grid--phone) .x-grid__row:last-child {
  border-radius: 0 0 calc(var(--r-16) - 1px) calc(var(--r-16) - 1px);
}

.x-grid:not(.is-dragging) .x-grid__row:hover {
  background: var(--hov);
}

.x-grid--center:not(.x-grid--phone) .x-grid__row {
  align-items: center;
}

.x-grid__lead {
  display: flex;
  align-items: center;
  gap: 4px;
  min-height: 24px;
  margin: -2px 0;
}

.x-grid__cell {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  min-width: 0;
}

.x-grid__blank {
  color: var(--faint);
  font: 400 var(--fs-12) / 22px var(--font-mono);
}

.x-grid__grip,
.x-grid__more {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 0;
  background: transparent;
  outline: none;
  transition: background-color var(--t-fast), color var(--t-fast), transform var(--t-fast);
}

/* A held handle drags its row, so it keeps its size then. */
.x-grid__grip:active:not([aria-pressed='true']),
.x-grid__more:active {
  transform: scale(var(--press-scale-icon));
}

.x-grid__grip {
  width: 22px;
  height: var(--inline-btn);
  border-radius: var(--btn-r);
  color: var(--faint);
  cursor: grab;
  touch-action: none;
}

.x-grid__more {
  width: var(--inline-btn);
  height: var(--inline-btn);
  border-radius: var(--btn-r);
  color: var(--dim);
}

.x-grid__grip:hover,
.x-grid__more:hover,
.x-grid__more[aria-expanded='true'] {
  background: var(--hov);
  color: var(--fg);
}

.x-grid__grip:focus-visible,
.x-grid__more:focus-visible {
  box-shadow: 0 0 0 2px var(--acc-line);
  color: var(--fg);
}

.x-grid__grip[aria-pressed='true'] {
  background: var(--acc-soft);
  color: var(--acc-t);
  cursor: grabbing;
}

.x-grid__number {
  flex: none;
  width: calc(v-bind(digits) * 1ch);
  /* The handle and the dots carry 4px and 7px of room inside their boxes: 8px of space either side of the number. */
  margin-right: -3px;
  color: var(--dim);
  font: 500 var(--fs-12) / 1 var(--font-mono);
  text-align: center;
}

.x-grid__number--head {
  color: inherit;
}

/* While a row is lifted the others step aside smoothly; the lifted one follows the pointer at once. */
.x-grid.is-dragging .x-grid__row {
  transition: transform var(--t-fast) var(--ease);
}

.x-grid.is-dragging .x-grid__row.is-following {
  transition: none;
}

/* The lifted row rides over the others. */
.x-grid__row.is-lifted {
  z-index: 2;
  background: var(--dlg);
  box-shadow: var(--lift), 0 0 0 1px var(--acc-line);
}

/* Inside a card. */
.x-grid--nested {
  border-radius: var(--r-12);
  background: none;
}

.x-grid--nested .x-grid__head {
  border-radius: calc(var(--r-12) - 1px) calc(var(--r-12) - 1px) 0 0;
}

.x-grid--nested:not(.x-grid--phone) .x-grid__row:last-child {
  border-radius: 0 0 calc(var(--r-12) - 1px) calc(var(--r-12) - 1px);
}

/* Phones: a card per row. */
.x-grid--phone {
  display: flex;
  flex-direction: column;
  gap: 8px;
  border: 0;
  background: none;
}

.x-grid--phone .x-grid__row {
  position: relative;
  border: 1px solid var(--line);
  border-radius: var(--r-16);
  background: var(--card);
}

.x-grid--phone.x-grid--nested .x-grid__row {
  border-radius: var(--r-12);
  background: var(--nest);
}

.x-grid__card-head {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 2px;
}

.x-grid__card-head:not(:last-child) {
  border-bottom: 1px solid var(--div);
}

.x-grid--phone .x-grid__grip,
.x-grid--phone .x-grid__more {
  width: var(--btn-sq);
  height: var(--btn-sq);
  border-radius: var(--btn-sq-r);
}

.x-grid--phone .x-grid__number {
  width: auto;
  margin-right: 0;
}

/* Without a handle the number opens the card: kept off its edge. */
.x-grid--phone .x-grid__number:first-child {
  margin-left: var(--sp-10);
}

.x-grid__card-title {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
  min-width: 0;
  /* A title of two lines keeps off the card's top and bottom; one line stays within the buttons' height. */
  padding: var(--sp-8) 0;
}

.x-grid__card-body {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px 12px;
  padding: 10px 14px 12px;
}

.x-grid__field {
  grid-column: 1 / -1;
  min-width: 0;
}

.x-grid__field--half {
  grid-column: auto;
}

.x-grid__field-label {
  margin-bottom: 4px;
  color: var(--dim);
  font: 500 var(--fs-11) / var(--lh-tight) var(--font-sans);
}

.x-grid__field-value {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  min-width: 0;
}
</style>
