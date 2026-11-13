<script setup lang="ts">
/** A figure in a small frame, its name over its value (the slot, a chip), e.g. what is left of a client's traffic. */
defineProps<{ label: string }>()
</script>

<template>
  <div class="x-fact-tile">
    <span class="x-fact-tile__label">{{ label }}</span>
    <slot />
  </div>
</template>

<style scoped>
.x-fact-tile {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--sp-6);
  min-width: 0;
  padding: var(--sp-10) var(--sp-6);
  border: 1px solid var(--div);
  border-radius: var(--nest-r);
  background: var(--tint);
  text-align: center;
}

.x-fact-tile__label {
  color: var(--mut);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-sans);
}
</style>
