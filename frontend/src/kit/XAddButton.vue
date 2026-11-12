<script setup lang="ts">
import XIcon from './XIcon.vue'

/** A small dashed + that adds an item to a list, e.g. beside a list's title or a switch. label names it for screen readers. */
defineProps<{ label: string }>()
</script>

<template>
  <button type="button" class="x-add" :aria-label="label">
    <XIcon name="plus" :size="12" :stroke="1.8" />
  </button>
</template>

<style scoped>
.x-add {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: 1px dashed var(--acc);
  border-radius: var(--chip-r);
  background: transparent;
  color: var(--acc-t);
  outline: none;
  transition: background-color var(--t-fast), transform var(--t-fast);
}

.x-add:active {
  transform: scale(var(--press-scale));
}

.x-add:hover {
  background: var(--acc-soft);
}

.x-add:focus-visible {
  box-shadow: 0 0 0 3px var(--acc-ring);
}

/* A finger's size on phones. */
@media (max-width: 767px) {
  .x-add {
    width: 28px;
    height: 28px;
  }
}
</style>
