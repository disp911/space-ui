<script setup lang="ts">
/**
 * Chips in a frame that scrolls past a height, e.g. the emails clients will
 * get once copied: a quiet list to look over, not to edit.
 */
withDefaults(defineProps<{ maxHeight?: number; label?: string }>(), { maxHeight: 120 })
</script>

<template>
  <div class="x-chip-box" role="list" :aria-label="label" :style="{ maxHeight: `${maxHeight}px` }"><slot /></div>
</template>

<style scoped>
.x-chip-box {
  display: flex;
  flex-wrap: wrap;
  align-content: flex-start;
  gap: 6px;
  padding: 8px;
  border: 1px solid var(--div);
  border-radius: var(--ctl-r);
  background: var(--field);
  overflow-y: auto;
  scrollbar-gutter: stable;
}
</style>
