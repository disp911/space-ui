<script setup lang="ts">
import { computed, useId } from 'vue'
import XIcon from './XIcon.vue'
import XSpinner from './XSpinner.vue'
import XTooltip from './XTooltip.vue'
import type { IconName } from './icons'

/**
 * Label, control and its hint or error. The control receives id and
 * aria-describedby through the default slot:
 *   <XField label="Port" v-slot="f"><XNumber v-bind="f" v-model="port" /></XField>
 */
const props = withDefaults(
  defineProps<{
    label?: string
    /** A round ? after the label that tells more on hover or focus. */
    help?: string
    hint?: string
    error?: string
    required?: boolean
    /** row: label on the left, control on the right (switches, short values). */
    layout?: 'stack' | 'row'
    /** A small button after the label that does something to the value, e.g. «Reset» to a new random one; emits action. */
    action?: string
    actionIcon?: IconName
    /** The action is running: a spinner in place of its icon, clicks wait. */
    actionLoading?: boolean
  }>(),
  { layout: 'stack', actionIcon: 'refresh' },
)
const emit = defineEmits<{ action: [] }>()

const id = useId()
const describedBy = computed(() => (props.error || props.hint ? `${id}-note` : undefined))
</script>

<template>
  <div :class="['x-field', `x-field--${layout}`, { 'is-invalid': !!error }]">
    <div v-if="label || $slots.label || (layout === 'row' && (hint || error))" class="x-field__head">
      <label v-if="label || $slots.label" class="x-field__label" :for="id">
        <slot name="label">{{ label }}</slot>
        <span v-if="required" class="x-field__required" aria-hidden="true">*</span>
      </label>
      <XTooltip v-if="help" :text="help">
        <span class="x-field__help" tabindex="0" :aria-label="help">?</span>
      </XTooltip>
      <XTooltip v-if="action" :text="action" :delay="0">
        <button type="button" class="x-field__action" :aria-label="action" :aria-busy="actionLoading || undefined" @click="actionLoading || emit('action')">
          <XSpinner v-if="actionLoading" :size="12" />
          <XIcon v-else :name="actionIcon" :size="12" :stroke="1.6" />
        </button>
      </XTooltip>
      <!-- A state of the value after the label, e.g. an «Expired» chip. -->
      <span v-if="$slots.badge" class="x-field__badge"><slot name="badge" /></span>
      <p v-if="layout === 'row' && (error || hint)" :id="`${id}-note`" :class="['x-field__note', { 'is-error': !!error }]">
        {{ error || hint }}
      </p>
    </div>
    <div class="x-field__control">
      <slot :id="id" :aria-describedby="describedBy" :aria-required="required || undefined" />
    </div>
    <p v-if="layout === 'stack' && (error || hint)" :id="`${id}-note`" :class="['x-field__note', { 'is-error': !!error }]">
      {{ error || hint }}
    </p>
  </div>
</template>

<style scoped>
.x-field {
  display: flex;
  min-width: 0;
}

.x-field--stack {
  flex-direction: column;
  gap: 6px;
}

.x-field--row {
  align-items: center;
  gap: 16px;
  min-height: var(--ctl-h);
}

.x-field__head {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.x-field--row .x-field__head {
  flex: 1;
}

.x-field__label {
  display: flex;
  align-items: center;
  gap: 4px;
  min-height: 18px;
  color: var(--mut);
  font: 500 var(--fs-12) / var(--lh-tight) var(--font-sans);
}

.x-field--row .x-field__label {
  color: var(--fg2);
  font-size: var(--ctl-fs);
}

.x-field--stack .x-field__head:has(.x-field__help, .x-field__action, .x-field__badge) {
  flex-direction: row;
  align-items: center;
  gap: var(--sp-6);
}

.x-field__help {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  border: 1.2px solid;
  border-radius: 50%;
  color: var(--dim);
  font: 600 var(--fs-10) / 1 var(--font-sans);
  cursor: help;
  outline: none;
}

.x-field__help:hover,
.x-field__help:focus-visible {
  color: var(--fg);
}

.x-field__action {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  margin: -1px 0;
  padding: 0;
  border: 0;
  border-radius: var(--r-6);
  background: transparent;
  color: var(--acc-t);
  cursor: pointer;
  outline: none;
  transition: background-color var(--t-fast), transform var(--t-fast);
}

.x-field__action:active {
  transform: scale(var(--press-scale-icon));
}

.x-field__action[aria-busy='true'] {
  cursor: progress;
}

.x-field__action:hover {
  background: var(--acc-soft);
}

.x-field__action:focus-visible {
  box-shadow: 0 0 0 2px var(--acc-line);
}

.x-field__badge {
  display: inline-flex;
  margin: -2px 0;
}

.x-field__required {
  color: var(--err-t);
}

.x-field__control {
  display: flex;
  min-width: 0;
}

.x-field--row .x-field__control {
  flex: none;
  justify-content: flex-end;
}

.x-field__note {
  margin: 0;
  color: var(--dim);
  font: 400 var(--fs-12) / var(--lh-text) var(--font-sans);
}

.x-field__note.is-error {
  color: var(--err-t);
}
</style>
