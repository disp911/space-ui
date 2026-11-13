<script setup lang="ts">
import { ref, useId, watch } from 'vue'
import { useFloating } from './floating'

/**
 * Short hint shown on hover and keyboard focus; error is what a failure said:
 * a red frame and mono text. block lets what it hangs on fill its place, e.g.
 * a usage pill across a cell. silent keeps it away, e.g. while the button it
 * names has a popover open.
 */
const props = withDefaults(defineProps<{ text?: string; delay?: number; error?: boolean; block?: boolean; silent?: boolean }>(), { delay: 300 })

const id = useId()
const anchor = ref<HTMLElement | null>(null)
const tip = ref<HTMLElement | null>(null)
const open = ref(false)
const { style } = useFloating(anchor, tip, open, { placement: 'top', onDismiss: hide })
let timer = 0

function show() {
  clearTimeout(timer)
  if (props.silent) return
  timer = window.setTimeout(() => (open.value = true), props.delay)
}

function hide() {
  clearTimeout(timer)
  open.value = false
}

watch(
  () => props.silent,
  (h) => h && hide(),
)
</script>

<template>
  <span
    ref="anchor"
    :class="['x-tooltip-anchor', { 'x-tooltip-anchor--block': block }]"
    :aria-describedby="open ? id : undefined"
    @mouseenter="show"
    @mouseleave="hide"
    @focusin="show"
    @focusout="hide"
  >
    <slot />
  </span>
  <Teleport to="body">
    <Transition name="x-tooltip">
      <div v-if="open" :id="id" ref="tip" role="tooltip" :class="['x-tooltip', { 'x-tooltip--error': error }]" :style="style">
        <slot name="content">{{ text }}</slot>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.x-tooltip-anchor {
  display: inline-flex;
  min-width: 0;
}

.x-tooltip-anchor--block {
  display: flex;
  width: 100%;
}

.x-tooltip {
  z-index: 1100;
  max-width: 280px;
  padding: 6px 10px;
  border: 1px solid var(--line);
  border-radius: var(--r-12);
  background: var(--dlg);
  color: var(--fg2);
  box-shadow: var(--shadow-pop);
  font: 400 var(--fs-12) / var(--lh-text) var(--font-sans);
  pointer-events: none;
}

.x-tooltip--error {
  max-width: 300px;
  padding: var(--sp-10) var(--sp-12);
  border-color: var(--err-line);
  font-family: var(--font-mono);
  word-break: break-word;
}

.x-tooltip-enter-active,
.x-tooltip-leave-active {
  transition: opacity var(--t-fast);
}

.x-tooltip-enter-from,
.x-tooltip-leave-to {
  opacity: 0;
}
</style>
