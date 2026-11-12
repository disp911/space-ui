<script setup lang="ts">
import XIcon from './XIcon.vue'

/** Where traffic goes: an arrow before an outbound's tag, an accent fork before a balancer's. */
defineProps<{ tag: string; balancer?: boolean }>()
</script>

<template>
  <span :class="['x-route-target', { 'x-route-target--balancer': balancer }]">
    <XIcon :name="balancer ? 'balancer' : 'arrowRight'" :size="14" :stroke="1.5" class="x-route-target__icon" />
    <span class="x-route-target__tag">{{ tag }}</span>
  </span>
</template>

<style scoped>
.x-route-target {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-6);
  min-width: 0;
  min-height: 22px;
}

.x-route-target__icon {
  flex: none;
  color: var(--dim);
}

.x-route-target--balancer .x-route-target__icon {
  color: var(--acc-t);
}

.x-route-target__tag {
  min-width: 0;
  overflow: hidden;
  color: var(--fg);
  font: 500 var(--fs-12) / var(--lh-tight) var(--font-mono);
  white-space: nowrap;
  text-overflow: ellipsis;
}
</style>
