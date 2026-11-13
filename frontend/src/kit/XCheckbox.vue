<script setup lang="ts">
import XIcon from './XIcon.vue'

defineProps<{
  disabled?: boolean
  /** Drawn as a control in a toolbar: the frame and height of a secondary button. */
  boxed?: boolean
  indeterminate?: boolean
  invalid?: boolean
  /** Boxed only: when checked, the frame, box and label take this colour, e.g. connection types. */
  tone?: 'ok' | 'err' | 'upload'
}>()

const model = defineModel<boolean>({ default: false })
</script>

<template>
  <label :class="['x-check', { 'x-check--boxed': boxed, 'is-disabled': disabled }, boxed && tone && `x-check--${tone}`, { 'is-checked': model }]">
    <input
      v-model="model"
      class="x-check__input sr-only"
      type="checkbox"
      :disabled="disabled"
      :aria-invalid="invalid || undefined"
      :indeterminate="indeterminate"
    />
    <span :class="['x-check__box', { 'is-on': model || indeterminate, 'is-invalid': invalid }]" aria-hidden="true">
      <span v-if="indeterminate" class="x-check__dash" />
      <XIcon v-else-if="model" name="check" :size="12" :stroke="2.2" />
    </span>
    <span v-if="$slots.default" class="x-check__label"><slot /></span>
  </label>
</template>

<style scoped>
.x-check {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: var(--check);
  color: var(--fg2);
  font-size: var(--ctl-fs);
  cursor: pointer;
  user-select: none;
}

.x-check__box {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: var(--check);
  height: var(--check);
  border: 1px solid var(--field-hover);
  border-radius: var(--check-r);
  background: var(--field);
  color: var(--acc-ink);
  transition: background-color var(--t-fast), border-color var(--t-fast), box-shadow var(--t-fast), transform var(--t-fast);
}

/* The box gives under the finger, as a switch's knob does. */
.x-check:not(.is-disabled):active .x-check__box {
  transform: scale(var(--press-scale-icon));
}

.x-check:hover .x-check__box,
.x-check.is-hover .x-check__box {
  border-color: var(--acc-line);
}

.x-check__box.is-on {
  border-color: var(--acc);
  background: var(--acc);
}

.x-check__dash {
  width: 8px;
  height: 2px;
  border-radius: var(--r-pill);
  background: currentColor;
}

.x-check__input:focus-visible + .x-check__box,
.x-check.is-focus .x-check__box {
  box-shadow: 0 0 0 2px var(--bg), 0 0 0 4px var(--acc-line);
}

.x-check__box.is-invalid {
  border-color: var(--err);
  box-shadow: 0 0 0 3px var(--err-ring);
}

.x-check--boxed {
  flex: none;
  height: var(--ctl-h);
  padding: 0 12px;
  border: 1px solid var(--field-line);
  border-radius: var(--btn-r);
  background: var(--ctrl);
  font-weight: 500;
  transition: border-color var(--t-fast), box-shadow var(--t-fast);
}

.x-check--boxed:hover,
.x-check--boxed.is-hover {
  border-color: var(--field-hover);
}

.x-check--boxed:has(:focus-visible),
.x-check--boxed.is-focus {
  border-color: var(--acc-line);
  box-shadow: 0 0 0 3px var(--acc-ring);
}

.x-check--boxed .x-check__input:focus-visible + .x-check__box {
  box-shadow: none;
}

.x-check--ok { --tone: var(--ok); --tone-t: var(--ok-t); --tone-bg: var(--ok-bg); --tone-line: var(--ok-line); }
.x-check--err { --tone: var(--err); --tone-t: var(--err-t); --tone-bg: var(--err-bg); --tone-line: var(--err-line); }
.x-check--upload { --tone: var(--upload); --tone-t: var(--upload-t); --tone-bg: var(--upload-bg); --tone-line: var(--upload-line); }

:is(.x-check--ok, .x-check--err, .x-check--upload).is-checked {
  border-color: var(--tone-line);
  background: var(--tone-bg);
  color: var(--tone-t);
}

:is(.x-check--ok, .x-check--err, .x-check--upload).is-checked .x-check__box {
  border-color: var(--tone);
  background: var(--tone);
}

.x-check.is-disabled {
  cursor: not-allowed;
  opacity: .45;
}
</style>
