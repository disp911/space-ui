<script setup lang="ts">
import { useStack } from './media'

/**
 * A block nested under a setting that a switch opens: a caps title and the
 * fields, one step below the card. With columns, fields fill a grid of at
 * least that many px each; one column in a window that stacks its fields.
 */
const stack = useStack()
defineProps<{
  title?: string
  columns?: number
  /** Only a step in behind a thin line, no frame: fields that a switch opens among other fields, e.g. XMUX. */
  bare?: boolean
}>()
</script>

<template>
  <div :class="['x-nest', { 'x-nest--bare': bare }]">
    <div v-if="title" class="x-caps x-caps--lg x-nest__title">{{ title }}</div>
    <div :class="{ 'x-nest__grid': columns }" :style="columns ? { '--nest-col': stack ? '100%' : `${columns}px` } : undefined"><slot /></div>
  </div>
</template>

<style scoped>
.x-nest {
  min-width: 0;
  padding: 12px 14px 14px;
  border: 1px solid var(--div);
  border-radius: var(--nest-r);
  background: var(--nest);
}

.x-nest--bare {
  margin-left: var(--sp-6);
  padding: 0 0 0 var(--sp-14);
  border: 0;
  border-left: 1px solid var(--div);
  border-radius: 0;
  background: none;
}

.x-nest__title {
  margin-bottom: 10px;
}

.x-nest__grid {
  display: grid;
  /* Columns of at least --nest-col, but never wider than the block on a narrow screen. */
  grid-template-columns: repeat(auto-fit, minmax(min(var(--nest-col), 100%), 1fr));
  gap: 12px;
}

@media (max-width: 767px) {
  .x-nest--bare {
    padding-left: var(--sp-12);
  }
}
</style>
