<script setup lang="ts">
import { XSkeletonText } from '../../kit'

/** A labelled figure. Small figures are secondary, coloured by direction. A grey bar stands in while loading. */
defineProps<{ label: string; value: string; small?: boolean; tone?: 'up' | 'down'; loading?: boolean }>()
</script>

<template>
  <div :class="['stat', { 'stat--small': small }, tone && `stat--${tone}`]">
    <div class="x-caps stat__label">{{ label }}</div>
    <div class="stat__value">
      <XSkeletonText v-if="loading" :width="small ? 48 : 64" />
      <slot v-else>{{ value }}</slot>
    </div>
  </div>
</template>

<style scoped>
.stat {
  min-width: 0;
}

.stat__label {
  margin-bottom: 4px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.stat--small .stat__value {
  color: var(--fg2);
  font-size: var(--fs-14);
}

/* The server's traffic: sent in the success green, received in the kit's blue. */
.stat--up .stat__value { color: var(--ok-t); }
.stat--down .stat__value { color: var(--on-t); }

.stat__value {
  overflow: hidden;
  color: var(--fg);
  font: 500 var(--fs-16) / var(--lh-tight) var(--font-mono);
  white-space: nowrap;
  text-overflow: ellipsis;
}
</style>
