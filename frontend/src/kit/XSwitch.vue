<script setup lang="ts">
import { computed, useAttrs } from 'vue'
import XSpinner from './XSpinner.vue'
import XTooltip from './XTooltip.vue'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  disabled?: boolean
  /** The change is being applied; the knob shows a spinner. */
  loading?: boolean
  /** Applying failed and the switch was rolled back. */
  invalid?: boolean
  modified?: boolean
  label?: string
  /**
   * A word of state to the left, e.g. «On» for a setting that applies at once:
   * accent when on, dim when off, amber while the change is applied.
   */
  text?: string
  /**
   * The label shown beside the switch in a frame of a field's height, e.g.
   * Mux among the fields of a window; the actions slot sits before the switch.
   * Only the switch reacts to the pointer; the label and frame are plain.
   */
  boxed?: boolean
  /** A round ? after a boxed label that tells more on hover or focus. */
  help?: string
}>()

const model = defineModel<boolean>({ default: false })
const attrs = useAttrs()

function toggle() {
  model.value = !model.value
}

const button = computed(() => ({
  ...attrs,
  type: 'button' as const,
  role: 'switch',
  'aria-checked': model.value,
  'aria-label': props.label,
  'aria-busy': props.loading || undefined,
  'aria-invalid': props.invalid || undefined,
  disabled: props.disabled,
  class: ['x-switch', attrs.class, { 'is-on': model.value, 'is-loading': props.loading, 'is-invalid': props.invalid, 'is-modified': props.modified }],
}))
</script>

<template>
  <div v-if="boxed" :class="['x-switch-box', { 'is-disabled': disabled }]">
    <span class="x-switch-box__label" aria-hidden="true">{{ label }}</span>
    <XTooltip v-if="help" :text="help">
      <span class="x-switch-box__help" tabindex="0" :aria-label="help">?</span>
    </XTooltip>
    <span v-if="help" class="x-switch-box__gap" />
    <slot name="actions" />
    <button v-bind="button" @click="loading || toggle()">
      <span class="x-switch__knob"><XSpinner v-if="loading" :size="12" /></span>
    </button>
  </div>
  <span v-else-if="text" class="x-switch-line">
    <span :class="['x-switch-line__text', { 'is-on': model, 'is-loading': loading }]" aria-hidden="true">{{ text }}</span>
    <button v-bind="button" @click="loading || toggle()">
      <span class="x-switch__knob"><XSpinner v-if="loading" :size="12" /></span>
    </button>
  </span>
  <button v-else v-bind="button" @click="loading || toggle()">
    <span class="x-switch__knob"><XSpinner v-if="loading" :size="12" /></span>
  </button>
</template>

<style scoped>
.x-switch-box {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  height: var(--ctl-h);
  padding: 0 10px 0 12px;
  border: 1px solid var(--field-line);
  border-radius: var(--ctl-r);
  transition: border-color var(--t-fast);
}

.x-switch-box__help {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  margin-left: -4px;
  border: 1.2px solid;
  border-radius: 50%;
  color: var(--dim);
  font: 600 var(--fs-10) / 1 var(--font-sans);
  cursor: help;
  outline: none;
}

.x-switch-box__help:hover,
.x-switch-box__help:focus-visible {
  color: var(--fg);
}

/* The label and its ? sit together at the left; the switch keeps to the right. */
.x-switch-box:has(.x-switch-box__help) .x-switch-box__label {
  flex: 0 1 auto;
}

.x-switch-box__gap {
  flex: 1;
  align-self: stretch;
}

.x-switch-box.is-disabled {
  cursor: default;
  opacity: .5;
}

.x-switch-box__label {
  flex: 1;
  align-self: stretch;
  display: flex;
  align-items: center;
  min-width: 0;
  overflow: hidden;
  color: var(--fg2);
  font: 400 var(--fs-13) / var(--lh-tight) var(--font-sans);
  white-space: nowrap;
  text-overflow: ellipsis;
}

.x-switch-line {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}

.x-switch-line__text {
  color: var(--dim);
  font: 500 var(--fs-12) / var(--lh-tight) var(--font-sans);
  white-space: nowrap;
  transition: color var(--t-fast);
}

.x-switch-line__text.is-on {
  color: var(--acc-t);
}

.x-switch-line__text.is-loading {
  color: var(--warn-t);
}

.x-switch {
  --knob-size: calc(var(--switch-h) - 6px);
  /* The knob sits evenly inside the track. */
  --inset: calc((var(--switch-h) - var(--knob-size)) / 2);
  position: relative;
  flex: none;
  width: var(--switch-w);
  height: var(--switch-h);
  padding: 0;
  border: 0;
  border-radius: var(--r-pill);
  background: var(--track);
  outline: none;
  transition: background-color var(--t-med), box-shadow var(--t-fast), filter var(--t-fast);
}

.x-switch__knob {
  position: absolute;
  top: var(--inset);
  left: var(--inset);
  display: flex;
  align-items: center;
  justify-content: center;
  width: var(--knob-size);
  height: var(--knob-size);
  border-radius: var(--r-pill);
  background: var(--knob);
  color: var(--dim);
  box-shadow: var(--shadow-knob);
  transition: left var(--t-med) var(--ease), width var(--t-fast) var(--ease), background-color var(--t-med);
}

.x-switch.is-on {
  background: var(--acc);
}

.x-switch.is-on .x-switch__knob {
  left: calc(100% - var(--knob-size) - var(--inset));
  background: var(--knob-on);
  color: var(--acc);
}

.x-switch:hover:not(:disabled),
.x-switch.is-hover {
  filter: brightness(1.12);
}

.x-switch:active:not(:disabled) .x-switch__knob,
.x-switch.is-active .x-switch__knob {
  width: calc(var(--knob-size) + var(--inset));
}

.x-switch.is-on:active:not(:disabled) .x-switch__knob,
.x-switch.is-on.is-active .x-switch__knob {
  left: calc(100% - var(--knob-size) - 2 * var(--inset));
}

.x-switch:focus-visible,
.x-switch.is-focus {
  box-shadow: 0 0 0 2px var(--bg), 0 0 0 4px var(--acc-line);
}

.x-switch.is-modified {
  box-shadow: 0 0 0 2px var(--bg), 0 0 0 4px var(--mod-line);
}

.x-switch.is-invalid {
  box-shadow: 0 0 0 1px var(--err), 0 0 0 4px var(--err-ring);
}

.x-switch.is-loading {
  cursor: progress;
  opacity: .8;
}

.x-switch:disabled {
  cursor: not-allowed;
  opacity: .4;
  filter: grayscale(.6);
}
</style>
