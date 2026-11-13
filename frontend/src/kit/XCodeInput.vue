<script setup lang="ts">
import { useFieldAttrs } from './useFieldAttrs'

defineOptions({ inheritAttrs: false })

/**
 * A one-time code from an authenticator app: digits only, spaced wide in the
 * middle, with a count of the digits typed at the right. Enter emits submit.
 */
const props = withDefaults(defineProps<{ length?: number; invalid?: boolean; label?: string }>(), { length: 6 })
const emit = defineEmits<{ submit: [] }>()
const model = defineModel<string>({ default: '' })
const { wrapper, control } = useFieldAttrs()

function onInput(e: Event) {
  const input = e.target as HTMLInputElement
  model.value = input.value.replace(/\D/g, '').slice(0, props.length)
  input.value = model.value
}
</script>

<template>
  <div v-bind="wrapper" :class="['x-code', { 'is-invalid': invalid, 'is-full': model.length === length }]">
    <input
      v-bind="control"
      class="x-code__control"
      :value="model"
      inputmode="numeric"
      autocomplete="one-time-code"
      spellcheck="false"
      :maxlength="length"
      :placeholder="'·'.repeat(length)"
      :aria-label="label"
      :aria-invalid="invalid || undefined"
      @input="onInput"
      @keydown.enter.prevent="emit('submit')"
    />
    <span class="x-code__count" aria-hidden="true">{{ model.length }} / {{ length }}</span>
  </div>
</template>

<style scoped>
.x-code {
  position: relative;
}

.x-code__control {
  display: block;
  width: 100%;
  height: 44px;
  margin: 0;
  padding: 0 56px;
  border: 1px solid var(--field-line);
  border-radius: var(--ctl-r);
  outline: none;
  background: var(--field);
  color: var(--fg);
  caret-color: var(--acc);
  font: 600 var(--fs-18) / 1 var(--font-mono);
  letter-spacing: .32em;
  text-align: center;
  /* The spacing follows each digit; the indent balances the last one. */
  text-indent: .32em;
  transition: border-color var(--t-fast), box-shadow var(--t-fast), background-color var(--t-fast);
}

.x-code__control::placeholder {
  color: var(--faint);
}

.x-code__control:hover {
  border-color: var(--field-hover);
}

.x-code__control:focus {
  border-color: var(--acc-line);
  box-shadow: 0 0 0 3px var(--acc-ring);
}

.x-code.is-invalid .x-code__control {
  border-color: var(--err-line);
  background: var(--err-bg);
  box-shadow: 0 0 0 3px var(--err-ring);
}

.x-code__count {
  position: absolute;
  top: 50%;
  right: 14px;
  color: var(--dim);
  font: 500 var(--fs-11) / 1 var(--font-mono);
  pointer-events: none;
  transform: translateY(-50%);
}

.x-code.is-full .x-code__count {
  color: var(--acc-t);
}

.x-code.is-invalid .x-code__count {
  color: var(--err-t);
}

/* Like the other large fields: the regular control height on phones. */
@media (max-width: 767px) {
  .x-code__control {
    height: var(--ctl-h);
  }
}
</style>
