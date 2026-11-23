<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import XIcon from './XIcon.vue'
import XInput from './XInput.vue'
import XListHead from './XListHead.vue'

/**
 * Name and value pairs, e.g. the attributes of a routing rule: a title with a
 * dashed + at its right, then numbered rows of two fields and a − to drop the
 * row, under a line of column names. headless leaves the title out, when a
 * switch above the list names it and adds its rows, e.g. accounts under
 * «Password»; add() is exposed for that.
 */
defineProps<{ title: string; nameLabel: string; valueLabel: string; mono?: boolean; headless?: boolean }>()
const pairs = defineModel<[string, string][]>({ default: () => [] })
const { t } = useI18n()

function add() {
  pairs.value = [...pairs.value, ['', '']]
}

defineExpose({ add })

function drop(i: number) {
  pairs.value = pairs.value.filter((_, j) => j !== i)
}

function set(i: number, k: 0 | 1, v: string) {
  pairs.value = pairs.value.map((p, j) => (j === i ? (k ? [p[0], v] : [v, p[1]]) : p))
}
</script>

<template>
  <div class="x-pairs">
    <XListHead v-if="!headless" :title="title" add @add="add" />
    <template v-if="pairs.length">
      <div class="x-pairs__names" aria-hidden="true">
        <span>{{ nameLabel }}</span>
        <span>{{ valueLabel }}</span>
      </div>
      <div v-for="(p, i) in pairs" :key="i" class="x-pairs__row">
        <span class="x-pairs__n">{{ i + 1 }}</span>
        <XInput :model-value="p[0]" :mono="mono" :aria-label="`${nameLabel} ${i + 1}`" @update:model-value="set(i, 0, $event)" />
        <XInput :model-value="p[1]" :mono="mono" :aria-label="`${valueLabel} ${i + 1}`" @update:model-value="set(i, 1, $event)" />
        <button type="button" class="x-pairs__drop" :aria-label="t('kit.removeRow', { n: i + 1 })" @click="drop(i)">
          <XIcon name="minus" :size="14" :stroke="1.6" />
        </button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.x-pairs {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.x-pairs__names {
  display: flex;
  gap: 8px;
  padding: 0 calc(var(--btn-sq) + 8px) 0 26px;
  color: var(--dim);
  font: 400 var(--fs-11) / var(--lh-tight) var(--font-sans);
}

.x-pairs__names > span {
  flex: 1;
  min-width: 0;
}

.x-pairs__row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.x-pairs__row > :deep(.x-field-box) {
  flex: 1;
  min-width: 0;
}

.x-pairs__n {
  flex: none;
  width: 18px;
  color: var(--dim);
  font: 400 var(--fs-11) / 1 var(--font-mono);
  text-align: center;
}

.x-pairs__drop {
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

.x-pairs__drop:active {
  transform: scale(var(--press-scale-icon));
}

@media (hover: hover) {
  .x-pairs__drop:hover {
    border-color: var(--field-hover);
    color: var(--fg);
  }
}

.x-pairs__drop:focus-visible {
  box-shadow: 0 0 0 3px var(--acc-ring);
}
</style>
