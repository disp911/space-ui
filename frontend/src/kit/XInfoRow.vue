<script setup lang="ts">
/**
 * A framed line with a label on the left and a value on the right (the slot):
 * versions, states. accent tints it for the line that matters, e.g. a new version.
 */
defineProps<{ label: string; accent?: boolean }>()
</script>

<template>
  <div :class="['x-info-row', { 'x-info-row--accent': accent }]">
    <span class="x-info-row__label">{{ label }}</span>
    <span class="x-info-row__value"><slot /></span>
  </div>
</template>

<style scoped>
.x-info-row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  padding: 12px 14px;
  border: 1px solid var(--field-line);
  border-radius: var(--r-12);
  background: var(--field);
}

.x-info-row__label {
  flex: 1;
  min-width: 0;
  color: var(--mut);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-sans);
}

.x-info-row__value {
  display: flex;
  flex: none;
  align-items: center;
  gap: 10px;
  color: var(--fg2);
  font: 500 var(--fs-13) / var(--lh-tight) var(--font-mono);
}

.x-info-row--accent {
  border-color: var(--acc-line);
  background: var(--acc-soft);
}

.x-info-row--accent .x-info-row__value {
  color: var(--acc-t);
}

@media (max-width: 767px) {
  .x-info-row {
    padding: 14px;
  }

  .x-info-row__label {
    font-size: var(--fs-13);
  }
}
</style>
