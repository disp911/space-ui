<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import XIcon from './XIcon.vue'
import XInput from './XInput.vue'
import XListHead from './XListHead.vue'

/**
 * A list of single values, each in a field of its own, e.g. a peer's allowed
 * IPs: a title with a dashed + at its right, then a field and a − per value.
 * The last value stays, so the list is never left without a field. With
 * invalid, a value it refuses is marked, e.g. a DNS server that is no IP.
 */
const props = defineProps<{ title: string; mono?: boolean; placeholder?: string; invalid?: (value: string) => boolean }>()
const values = defineModel<string[]>({ default: () => [] })
const { t } = useI18n()

const set = (i: number, v: string) => (values.value = values.value.map((x, j) => (j === i ? v : x)))
</script>

<template>
  <div class="x-inputs">
    <XListHead :title="title" add @add="values = [...values, '']" />
    <div v-for="(v, i) in values" :key="i" class="x-inputs__row">
      <XInput :model-value="v" :mono="mono" :placeholder="placeholder" :invalid="!!v.trim() && !!invalid?.(v.trim())" :aria-label="`${props.title} ${i + 1}`" @update:model-value="set(i, $event)" />
      <button
        type="button"
        class="x-inputs__drop"
        :disabled="values.length < 2"
        :aria-label="t('kit.removeRow', { n: i + 1 })"
        @click="values = values.filter((_, j) => j !== i)"
      >
        <XIcon name="minus" :size="14" :stroke="1.6" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.x-inputs {
  display: flex;
  flex-direction: column;
  gap: var(--sp-6);
  min-width: 0;
}

.x-inputs__row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.x-inputs__row > :deep(.x-field-box) {
  flex: 1;
  min-width: 0;
}

.x-inputs__drop {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: var(--btn-sq);
  height: var(--btn-sq);
  padding: 0;
  border: 1px solid var(--field-line);
  border-radius: var(--btn-sq-r);
  background: transparent;
  color: var(--mut);
  outline: none;
  transition: border-color var(--t-fast), color var(--t-fast), transform var(--t-fast);
}

.x-inputs__drop:active:not(:disabled) {
  transform: scale(var(--press-scale-icon));
}

.x-inputs__drop:hover:not(:disabled) {
  border-color: var(--field-hover);
  color: var(--fg);
}

.x-inputs__drop:focus-visible {
  box-shadow: 0 0 0 3px var(--acc-ring);
}

.x-inputs__drop:disabled {
  cursor: default;
  opacity: .4;
}
</style>
