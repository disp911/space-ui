<script setup lang="ts">
/** What a setting makes, shown as it will be: a label and the value in accent mono, in a dashed frame. */
defineProps<{ label: string; value: string }>()
</script>

<template>
  <div class="x-example">
    <span class="x-example__label">{{ label }}</span>
    <span class="x-example__value">{{ value }}</span>
  </div>
</template>

<style scoped>
.x-example {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  /* The height and corners of a field: it sits among them. */
  min-height: var(--ctl-h);
  padding: var(--sp-4) 10px;
  border: 1px dashed var(--field-line);
  border-radius: var(--ctl-r);
  background: var(--nest);
}

.x-example__label {
  flex: none;
  color: var(--dim);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-sans);
  white-space: nowrap;
}

.x-example__value {
  min-width: 0;
  overflow: hidden;
  color: var(--acc-t);
  font: 500 var(--fs-13) / var(--lh-tight) var(--font-mono);
  white-space: pre;
  text-overflow: ellipsis;
}

/* A value with no room beside the label goes under it. */
@media (max-width: 767px) {
  .x-example {
    flex-wrap: wrap;
    row-gap: var(--sp-2);
    padding-block: var(--sp-6);
  }
}
</style>
