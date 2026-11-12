<script setup lang="ts" generic="T extends string">
/**
 * Parts to put together in order, e.g. what a remark is made of: each part is
 * a button, and a chosen one shows its place in the order. Choosing appends a
 * part, choosing it again takes it out.
 */
defineProps<{ options: { value: T; label: string }[]; label?: string; mono?: boolean }>()
const model = defineModel<T[]>({ default: () => [] })

function toggle(v: T) {
  model.value = model.value.includes(v) ? model.value.filter((x) => x !== v) : [...model.value, v]
}
</script>

<template>
  <div class="x-order" role="group" :aria-label="label">
    <button
      v-for="o in options"
      :key="o.value"
      type="button"
      :aria-pressed="model.includes(o.value)"
      :class="['x-order__item', { 'is-on': model.includes(o.value), 'x-order__item--mono': mono }]"
      @click="toggle(o.value)"
    >
      <span v-if="model.includes(o.value)" class="x-order__place">{{ model.indexOf(o.value) + 1 }}</span>
      {{ o.label }}
    </button>
  </div>
</template>

<style scoped>
.x-order {
  display: flex;
  flex: 1;
  gap: 6px;
  min-width: 0;
}

.x-order__item {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 0;
  height: var(--ctl-h);
  padding: 0 6px;
  border: 1px solid var(--field-line);
  border-radius: var(--ctl-r);
  background: var(--field);
  color: var(--mut);
  font: 500 var(--ctl-fs) / 1 var(--font-sans);
  white-space: nowrap;
  cursor: pointer;
  outline: none;
  transition: border-color var(--t-fast), background-color var(--t-fast), color var(--t-fast), transform var(--t-fast);
}

.x-order__item:active {
  transform: scale(var(--press-scale-chip));
}

.x-order__item--mono {
  font-family: var(--font-mono);
}

.x-order__item:hover {
  border-color: var(--field-hover);
  color: var(--fg);
}

.x-order__item:focus-visible {
  border-color: var(--acc-line);
  box-shadow: 0 0 0 3px var(--acc-ring);
}

.x-order__item.is-on {
  border-color: var(--acc-line);
  background: var(--acc-soft);
  color: var(--acc-t);
}

.x-order__place {
  flex: none;
  width: 16px;
  height: 16px;
  border-radius: var(--r-4);
  background: var(--acc);
  color: var(--acc-ink);
  font: 600 var(--fs-10) / 16px var(--font-mono);
  text-align: center;
}
</style>
