<script setup lang="ts">
/**
 * One line of state across the top of a window, e.g. «Account is Expired
 * And Disabled» over a client: tinted by its tone, bold, centred.
 */
withDefaults(defineProps<{ tone?: 'ok' | 'warn' | 'err' }>(), { tone: 'err' })
</script>

<template>
  <p :class="['x-state-bar', `x-state-bar--${tone}`]" role="status"><slot /></p>
</template>

<style scoped>
.x-state-bar {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 32px;
  margin: 0;
  padding: 8px 12px;
  border-radius: var(--nest-r);
  font: 600 var(--fs-13) / var(--lh-tight) var(--font-sans);
  text-align: center;
}

.x-state-bar--ok {
  background: var(--ok-bg);
  color: var(--ok-t);
}

.x-state-bar--warn {
  background: var(--warn-bg);
  color: var(--warn-t);
}

.x-state-bar--err {
  background: var(--err-bg);
  color: var(--err-t);
}
</style>
