<script setup lang="ts">
/** Tiny number next to a navigation item or a tab. */
withDefaults(defineProps<{ accent?: boolean }>(), {})
</script>

<template>
  <span :class="['x-counter', { 'x-counter--accent': accent }]"><slot /></span>
</template>

<style scoped>
.x-counter {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  min-width: var(--counter-h);
  height: var(--counter-h);
  padding: 0 4px;
  border-radius: var(--counter-r);
  background: var(--chip);
  color: var(--mut);
  font: 600 var(--fs-10) / 1 var(--font-mono);
}

.x-counter--accent {
  background: var(--acc-soft);
  color: var(--acc-t);
}
</style>
