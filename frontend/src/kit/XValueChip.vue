<script setup lang="ts">
import { computed, onBeforeUnmount, ref, useId } from 'vue'
import { useFloating } from './floating'

/**
 * A list of values in one small chip: an optional label, the first value and
 * +N for the rest. The whole list opens under it on hover or keyboard focus,
 * and on a tap where there is no hover, e.g. a phone.
 */
const props = withDefaults(defineProps<{ label?: string; values: string[]; mono?: boolean }>(), { mono: true })

const id = useId()
const anchor = ref<HTMLElement | null>(null)
const list = ref<HTMLElement | null>(null)
const open = ref(false)
const { style } = useFloating(anchor, list, open, { placement: 'bottom-start', onDismiss: () => (open.value = false) })

const head = computed(() => (props.label ? `${props.label} · ${props.values.length}` : String(props.values.length)))
const hoverable = matchMedia('(hover: hover)').matches
let timer = 0

function show() {
  clearTimeout(timer)
  open.value = true
}

// A moment to move the pointer from the chip onto the list.
function hide() {
  clearTimeout(timer)
  timer = window.setTimeout(() => (open.value = false), 160)
}

function tap() {
  if (!hoverable) open.value = !open.value
}

onBeforeUnmount(() => clearTimeout(timer))
</script>

<template>
  <button
    ref="anchor"
    type="button"
    class="x-value-chip"
    :aria-expanded="open"
    :aria-controls="`${id}-list`"
    @mouseenter="hoverable && show()"
    @mouseleave="hoverable && hide()"
    @focus="show"
    @blur="hide"
    @click="tap"
  >
    <span v-if="label" class="x-value-chip__label">{{ label }}</span>
    <span :class="['x-value-chip__value', { 'x-value-chip__value--mono': mono }]">{{ values[0] }}</span>
    <span v-if="values.length > 1" class="x-value-chip__more">+{{ values.length - 1 }}</span>
  </button>
  <Teleport to="body">
    <Transition name="x-popup">
      <div
        v-if="open"
        :id="`${id}-list`"
        ref="list"
        role="tooltip"
        class="x-value-chip__list"
        :style="style"
        @mouseenter="hoverable && show()"
        @mouseleave="hoverable && hide()"
      >
        <div class="x-caps x-value-chip__head">{{ head }}</div>
        <div class="x-value-chip__items">
          <div v-for="(v, i) in values" :key="i" :class="['x-value-chip__item', { 'x-value-chip__value--mono': mono }]">{{ v }}</div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.x-value-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  min-width: 0;
  height: var(--chip-h);
  padding: 0 var(--sp-4) 0 var(--sp-8);
  border: 1px solid var(--chip-line);
  border-radius: var(--chip-r);
  background: var(--chip);
  color: var(--fg2);
  font: 400 var(--fs-12) / 1 var(--font-sans);
  cursor: default;
  outline: none;
  transition: border-color var(--t-fast), box-shadow var(--t-fast), transform var(--t-fast);
}

.x-value-chip:active {
  transform: scale(var(--press-scale-chip));
}

.x-value-chip:hover {
  border-color: var(--field-hover);
}

.x-value-chip:focus-visible {
  border-color: var(--acc-line);
  box-shadow: 0 0 0 3px var(--acc-ring);
}

.x-value-chip__label {
  flex: none;
  color: var(--dim);
  font: 500 var(--fs-10) / 1 var(--font-sans);
}

/* Cut with an ellipsis when too long; the line is taller than the letters, so their tails are not cut too. */
.x-value-chip__value {
  min-width: 0;
  overflow: hidden;
  line-height: var(--lh-tight);
  white-space: nowrap;
  text-overflow: ellipsis;
}

.x-value-chip__value--mono {
  font-family: var(--font-mono);
  font-size: var(--fs-11);
}

.x-value-chip__more {
  flex: none;
  height: 16px;
  padding: 0 4px;
  border-radius: var(--counter-r);
  background: var(--hov);
  color: var(--mut);
  font: 500 var(--fs-10) / 16px var(--font-mono);
}

.x-value-chip__list {
  z-index: 1000;
  display: flex;
  flex-direction: column;
  width: max-content;
  min-width: 180px;
  max-width: min(340px, calc(100vw - 16px));
  padding: 8px 0 6px;
  border: 1px solid var(--line);
  border-radius: var(--pop-r);
  background: var(--dlg);
  box-shadow: var(--shadow-pop);
}

.x-value-chip__head {
  padding: 0 12px 6px;
}

.x-value-chip__items {
  max-height: 232px;
  padding: 0 12px;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.x-value-chip__item {
  padding: 2px 0;
  color: var(--fg2);
  font: 400 var(--fs-12) / 1.5 var(--font-sans);
  word-break: break-all;
}

.x-value-chip__item.x-value-chip__value--mono {
  font-size: var(--fs-11);
}
</style>
