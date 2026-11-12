<script setup lang="ts" generic="T extends OptionValue">
import { nextTick, onMounted, ref, watch } from 'vue'
import XIcon from './XIcon.vue'
import { useEdgeFade } from './edgeFade'
import { useSlidingThumb } from './slidingThumb'
import type { IconName } from './icons'
import type { OptionValue } from './types'

const props = defineProps<{
  options: { value: T; label: string; icon?: IconName; disabled?: boolean }[]
  /** Show only icons; labels become tooltips and accessible names. */
  iconOnly?: boolean
  /** Takes the width it is given; the options share it equally. */
  block?: boolean
  disabled?: boolean
  label?: string
  /** Wider than its place, e.g. on a phone: scrolls sideways instead of squeezing the options, fading out at an edge with more beyond it. */
  scroll?: boolean
}>()

const model = defineModel<T>({ required: true })
const root = ref<HTMLElement | null>(null)
// The options sit in a track inside the frame: it is what scrolls and fades, so the frame stays whole.
const track = ref<HTMLElement | null>(null)
const { classes: fade, update: updateFade } = useEdgeFade(track)

// A strip that scrolls keeps the chosen option in sight, without moving the page.
function reveal() {
  const box = track.value
  const el = box?.querySelector<HTMLElement>('.is-selected')
  if (!props.scroll || !box || !el) return
  // Clear of the edge fade, which is 32px (kit.css).
  const room = 32
  if (el.offsetLeft - room < box.scrollLeft) box.scrollLeft = el.offsetLeft - room
  else if (el.offsetLeft + el.offsetWidth + room > box.scrollLeft + box.clientWidth) box.scrollLeft = el.offsetLeft + el.offsetWidth - box.clientWidth + room
}
onMounted(reveal)
watch(model, () => nextTick(reveal))
watch(() => props.options, () => nextTick(updateFade), { deep: true })

// The chosen option's background is one thumb that travels between options.
const thumb = useSlidingThumb(track, '.is-selected', [model, () => props.options])

function onKeydown(e: KeyboardEvent) {
  const dir = e.key === 'ArrowRight' || e.key === 'ArrowDown' ? 1 : e.key === 'ArrowLeft' || e.key === 'ArrowUp' ? -1 : 0
  if (!dir) return
  e.preventDefault()
  const n = props.options.length
  let i = props.options.findIndex((o) => o.value === model.value)
  for (let k = 0; k < n; k++) {
    i = (i + dir + n) % n
    if (!props.options[i]?.disabled) break
  }
  const next = props.options[i]
  if (!next) return
  model.value = next.value
  root.value?.querySelectorAll<HTMLElement>('[role="radio"]')[i]?.focus()
}
</script>

<template>
  <div ref="root" role="radiogroup" :aria-label="label" :class="['x-seg', { 'x-seg--block': block, 'x-seg--scroll': scroll, 'is-disabled': disabled }]" @keydown="onKeydown">
    <div ref="track" :class="['x-seg__track', scroll && fade]">
      <span
        v-if="thumb"
        :class="['x-seg__thumb', thumb.ahead && `x-slide--${thumb.ahead}`]"
        :style="{ left: `${thumb.left}px`, right: `${thumb.right}px` }"
        aria-hidden="true"
      />
      <button
        v-for="o in options"
        :key="String(o.value)"
        type="button"
        role="radio"
        :aria-checked="o.value === model"
        :aria-label="iconOnly ? o.label : undefined"
        :title="iconOnly ? o.label : undefined"
        :tabindex="o.value === model ? 0 : -1"
        :disabled="disabled || o.disabled"
        :class="['x-seg__item', { 'is-selected': o.value === model, 'x-seg__item--icon': iconOnly }]"
        @click="model = o.value"
      >
        <XIcon v-if="o.icon" :name="o.icon" :size="14" />
        <!-- The bold copy keeps the width the same when the option is chosen. -->
        <span v-if="!iconOnly" class="x-seg__label" :data-label="o.label"><span>{{ o.label }}</span></span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.x-seg {
  position: relative;
  display: inline-flex;
  align-items: stretch;
  height: var(--ctl-h);
  padding: 2px;
  border: 1px solid var(--field-line);
  border-radius: var(--ctl-r);
  background: var(--field);
}

.x-seg--block {
  display: flex;
  width: 100%;
}

.x-seg__track {
  position: relative;
  display: flex;
  flex: 1;
  align-items: stretch;
  gap: 2px;
  min-width: 0;
}

.x-seg--scroll {
  max-width: 100%;
}

.x-seg--scroll .x-seg__track {
  overflow-x: auto;
  border-radius: var(--ctl-sm-r);
  scrollbar-width: none;
}

.x-seg--scroll .x-seg__track::-webkit-scrollbar {
  display: none;
}

.x-seg--scroll .x-seg__item {
  flex: none;
}

.x-seg--block .x-seg__item {
  flex: 1;
  min-width: 0;
}

/* Under the options; set to the chosen one's box. */
.x-seg__thumb {
  position: absolute;
  top: 0;
  bottom: 0;
  border-radius: var(--ctl-sm-r);
  background: var(--seg);
  box-shadow: var(--shadow-seg);
  pointer-events: none;
}

.x-seg__item {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 0 12px;
  border: 0;
  border-radius: var(--ctl-sm-r);
  background: transparent;
  color: var(--mut);
  font: 500 var(--fs-12) / 1 var(--font-sans);
  white-space: nowrap;
  outline: none;
  transition: background-color var(--t-fast), color var(--t-fast), box-shadow var(--t-fast);
}

.x-seg__label {
  display: inline-grid;
}

.x-seg__label > span,
.x-seg__label::after {
  grid-area: 1 / 1;
}

.x-seg__label::after {
  content: attr(data-label);
  height: 0;
  overflow: hidden;
  font-weight: 600;
  visibility: hidden;
}

.x-seg__item--icon {
  width: calc(var(--ctl-h) - 4px);
  padding: 0;
  color: var(--dim);
}

.x-seg__item:hover:not(:disabled),
.x-seg__item.is-hover {
  color: var(--fg);
}

.x-seg__item:focus-visible,
.x-seg__item.is-focus {
  box-shadow: inset 0 0 0 2px var(--acc-line);
  color: var(--fg);
}

.x-seg__item:active:not(:disabled),
.x-seg__item.is-active {
  opacity: .75;
}

.x-seg__item.is-selected {
  color: var(--fg);
  font-weight: 600;
}

.x-seg__item--icon.is-selected {
  color: var(--acc-t);
}

.x-seg__item:disabled {
  color: var(--faint);
  cursor: not-allowed;
}

.x-seg.is-disabled {
  opacity: .45;
}

@media (max-width: 767px) {
  .x-seg__item {
    font-size: var(--fs-13);
  }
}
</style>
