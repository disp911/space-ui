<script setup lang="ts">
/** Buttons that belong together in one frame, split by a thin line, e.g. refresh and reset traffic. */
defineProps<{ label?: string }>()
</script>

<template>
  <div class="x-btn-group" role="group" :aria-label="label"><slot /></div>
</template>

<style scoped>
.x-btn-group {
  display: inline-flex;
  flex: none;
  overflow: clip;
  border: 1px solid var(--field-line);
  border-radius: var(--btn-r);
  background: var(--ctrl);
}

/* The frame is the group's: the buttons inside lose their own and meet at a line. */
/* Doubled class: above the buttons' own hover borders and backgrounds. */
.x-btn-group > :deep(.x-btn.x-btn) {
  /* Inside the frame, so the group stands as tall as any control. */
  height: calc(var(--ctl-h) - 2px);
  border-color: transparent;
  border-radius: 0;
  background: transparent;
}

.x-btn-group > :deep(.x-btn.x-btn + .x-btn) {
  border-left-color: var(--field-line);
}

.x-btn-group > :deep(.x-btn.x-btn:hover:not(:disabled)) {
  background: var(--hov);
}

.x-btn-group > :deep(.x-btn:focus-visible) {
  box-shadow: inset 0 0 0 2px var(--acc-line);
}
</style>
