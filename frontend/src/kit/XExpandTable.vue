<script setup lang="ts" generic="T">
import { computed, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import XCollapse from './XCollapse.vue'
import XIcon from './XIcon.vue'
import { compactKey, usePhone } from './media'

export interface ExpandColumn {
  key: string
  label: string
  /** Grid track, e.g. '56px' or 'minmax(0, 1.6fr)'; minmax(0, 1fr) when left out. */
  size?: string
  /** How a muted row shows the cell: dimmed (the default), a little dimmed, or as it is, e.g. a menu or a switch. */
  muted?: 'soft' | 'none'
  /** Label and content at the end of the cell, e.g. a button at the row's right; or in its middle, e.g. a short number. */
  align?: 'end' | 'center'
  /** What the column holds is plain to see, e.g. a menu or a switch: the label is read out but not shown. */
  quiet?: boolean
}

/**
 * A card with a list whose rows open onto what they hold, e.g. inbounds and
 * their clients: tools on top, a caps header, rows of one line with a chevron
 * that opens the `expanded` slot under the row, and room for a pager below.
 * Phones get their own columns; there the page places the toggle itself, from
 * the slot props, as the row's name usually opens it. A muted row, e.g. a
 * disabled inbound, dims its cells. A cell shows the `cell-<key>` slot on
 * desktop and `phone-<key>` on phones. With compactBelow it takes its phone
 * columns whenever it is narrower than that, e.g. a laptop screen too small
 * for every column; the lists its rows open onto follow it, and `compact`
 * tells the page.
 */
const props = defineProps<{
  columns: ExpandColumn[]
  phoneColumns: ExpandColumn[]
  rows: T[]
  rowKey: (row: T, index: number) => string | number
  /** Rows that open; the others get no chevron. */
  expandable?: (row: T) => boolean
  muted?: (row: T) => boolean
  /** The accessible name of a row's chevron, e.g. «Clients». */
  toggleLabel?: string
  /** The narrowest width, in pixels, the desktop columns fit in; below it the phone columns show. */
  compactBelow?: number
}>()
const open = defineModel<(string | number)[]>('open', { default: () => [] })
const emit = defineEmits<{ compact: [compact: boolean] }>()
const { t } = useI18n()
const screenPhone = usePhone()
const root = ref<HTMLElement | null>(null)
const narrow = ref(false)
let observer: ResizeObserver | null = null
onMounted(() => {
  if (!props.compactBelow || !root.value) return
  // Measured at once too, so the first paint already has the right columns.
  narrow.value = root.value.getBoundingClientRect().width < props.compactBelow
  observer = new ResizeObserver(([entry]) => (narrow.value = !!entry && entry.contentRect.width < props.compactBelow!))
  observer.observe(root.value)
})
onBeforeUnmount(() => observer?.disconnect())

/** Phone columns on a phone, or wherever the desktop ones do not fit. */
const phone = computed(() => screenPhone.value || narrow.value)
provide(compactKey, phone)
watch(phone, (v) => emit('compact', v), { immediate: true })

const columns = computed(() => (phone.value ? props.phoneColumns : props.columns))
const template = computed(() =>
  [...(phone.value ? [] : ['24px']), ...columns.value.map((c) => c.size ?? 'minmax(0, 1fr)')].join(' '),
)

const isOpen = (key: string | number) => open.value.includes(key)
function toggle(key: string | number) {
  open.value = isOpen(key) ? open.value.filter((k) => k !== key) : [...open.value, key]
}
const canOpen = (row: T) => props.expandable?.(row) ?? false
</script>

<template>
  <section ref="root" :class="['x-expand', { 'x-expand--phone': phone }]">
    <div v-if="$slots.tools" class="x-expand__tools"><slot name="tools" /></div>

    <div class="x-expand__head x-caps" role="row">
      <span v-if="!phone" aria-hidden="true" />
      <span
        v-for="c in columns"
        :key="c.key"
        :class="{ 'is-end': c.align === 'end', 'is-center': c.align === 'center' }"
        role="columnheader"
      ><span :class="{ 'sr-only': c.quiet }">{{ c.label }}</span></span>
    </div>

    <div v-for="(row, i) in rows" :key="rowKey(row, i)" class="x-expand__item">
      <div :class="['x-expand__row', { 'is-open': isOpen(rowKey(row, i)) }]" role="row">
        <div v-if="!phone" class="x-expand__toggle-cell">
          <button
            v-if="canOpen(row)"
            type="button"
            class="x-expand__toggle"
            :aria-expanded="isOpen(rowKey(row, i))"
            :aria-label="toggleLabel ?? t('kit.expand')"
            @click="toggle(rowKey(row, i))"
          >
            <XIcon name="chevron" :size="12" :stroke="1.8" :class="['x-expand__chevron', { 'is-open': isOpen(rowKey(row, i)) }]" />
          </button>
        </div>
        <div
          v-for="c in columns"
          :key="c.key"
          :class="['x-expand__cell', { 'is-muted': muted?.(row) && c.muted !== 'none', 'is-soft': c.muted === 'soft', 'is-end': c.align === 'end', 'is-center': c.align === 'center' }]"
          role="cell"
        >
          <slot
            :name="`${phone ? 'phone' : 'cell'}-${c.key}`"
            :row="row"
            :index="i"
            :open="isOpen(rowKey(row, i))"
            :expandable="canOpen(row)"
            :toggle="() => toggle(rowKey(row, i))"
          />
        </div>
      </div>
      <XCollapse :open="isOpen(rowKey(row, i)) && canOpen(row)" lazy>
        <div class="x-expand__inner">
          <slot name="expanded" :row="row" :index="i" />
        </div>
      </XCollapse>
    </div>

    <div v-if="!rows.length && $slots.empty" class="x-expand__empty"><slot name="empty" /></div>
    <div v-if="$slots.foot" class="x-expand__foot"><slot name="foot" /></div>
  </section>
</template>

<style scoped>
.x-expand {
  min-width: 0;
  border: 1px solid var(--line);
  border-radius: var(--card-r);
  background: var(--card);
}

.x-expand__tools {
  padding: 14px 14px 12px;
}

.x-expand__head,
.x-expand__row {
  display: grid;
  grid-template-columns: v-bind(template);
  gap: 10px;
  align-items: center;
}

.x-expand__head {
  padding: 10px 14px;
  border-top: 1px solid var(--div);
  background: var(--field);
  white-space: nowrap;
}

.x-expand__head:first-child {
  border-top: 0;
  border-radius: calc(var(--card-r) - 1px) calc(var(--card-r) - 1px) 0 0;
}

.x-expand__item {
  border-top: 1px solid var(--div);
}

.x-expand__row {
  min-height: 46px;
  padding: 9px 14px;
  transition: background-color var(--t-fast);
}

.x-expand__row.is-open {
  background: var(--hov);
}

.x-expand__toggle-cell {
  display: flex;
  justify-content: center;
}

.x-expand__toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 0;
  border-radius: var(--r-6);
  background: transparent;
  color: var(--dim);
  outline: none;
  transition: background-color var(--t-fast), color var(--t-fast), transform var(--t-fast);
}

.x-expand__toggle:hover,
.x-expand__toggle.is-hover {
  background: var(--press);
  color: var(--fg);
}

.x-expand__toggle:focus-visible,
.x-expand__toggle.is-focus {
  box-shadow: 0 0 0 2px var(--acc-line);
  color: var(--fg);
}

.x-expand__toggle:active,
.x-expand__toggle.is-active {
  transform: scale(var(--press-scale-icon));
}

.x-expand__chevron {
  transition: transform var(--t-med) var(--ease);
}

.x-expand__chevron.is-open {
  transform: rotate(90deg);
}

.x-expand__head .is-end {
  text-align: end;
}

.x-expand__cell.is-end {
  display: flex;
  justify-content: flex-end;
}

.x-expand__head .is-center {
  text-align: center;
}

.x-expand__cell.is-center {
  display: flex;
  justify-content: center;
}

.x-expand__cell {
  min-width: 0;
  transition: opacity var(--t-med);
}

.x-expand__cell.is-muted {
  opacity: .42;
  filter: grayscale(.7);
}

.x-expand__cell.is-muted.is-soft {
  opacity: .5;
  filter: none;
}

/* As far from the open row, which is tinted, as from the next one. */
.x-expand__inner {
  margin: 12px 14px;
  /* A hovered last row keeps to the rounded corners; popups are teleported, so nothing else is cut. */
  overflow: clip;
  border: 1px solid var(--div);
  border-radius: var(--nest-r);
  background: var(--nest);
}

.x-expand__empty,
.x-expand__foot {
  border-top: 1px solid var(--div);
}

/* As a card's body around an empty state, e.g. XEmpty tab. */
.x-expand__empty {
  padding: 16px;
}

.x-expand__foot {
  padding: 10px 14px;
}

.x-expand--phone .x-expand__tools {
  padding: 12px;
}

.x-expand--phone .x-expand__head {
  gap: 6px;
  padding: 9px 14px 9px 12px;
}

/*
 * The menu button at the left is borderless, its dots 10px inside it; the info
 * button at the right has a frame. Its frame keeps as far from the edge as the
 * dots do, so the row looks even.
 */
.x-expand--phone .x-expand__row {
  gap: 6px;
  min-height: 56px;
  padding: 2px 14px 2px 4px;
}

.x-expand--phone .x-expand__inner {
  margin: 10px 8px;
}
</style>
