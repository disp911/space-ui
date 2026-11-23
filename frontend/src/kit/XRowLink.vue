<script setup lang="ts">
import XIcon from './XIcon.vue'
import type { IconName } from './icons'

/**
 * A row in a card that opens a window: icon, label and a chevron. Its icon
 * lines up with the card's text; the hover area reaches 12px past it.
 */
defineProps<{ icon: IconName; label: string }>()
</script>

<template>
  <button type="button" class="x-row-link">
    <XIcon :name="icon" :size="16" :stroke="1.4" />
    <span class="x-row-link__label">{{ label }}</span>
    <XIcon name="chevron" :size="12" :stroke="1.6" class="x-row-link__chevron" />
  </button>
</template>

<style scoped>
.x-row-link {
  display: flex;
  align-items: center;
  gap: 12px;
  width: calc(100% + 24px);
  height: var(--ctl-h);
  margin: 0 -12px;
  padding: 0 12px;
  border: 0;
  border-radius: var(--ctl-r);
  background: transparent;
  color: var(--fg2);
  font: 400 var(--ctl-fs) / 1 var(--font-sans);
  text-align: left;
  outline: none;
  transition: background-color var(--t-fast), color var(--t-fast);
}

.x-row-link.is-hover {
  background: var(--hov);
  color: var(--fg);
}

@media (hover: hover) {
  .x-row-link:hover {
    background: var(--hov);
    color: var(--fg);
  }
}

.x-row-link:active {
  background: var(--press);
}

.x-row-link:focus-visible,
.x-row-link.is-focus {
  box-shadow: inset 0 0 0 2px var(--acc-line);
}

.x-row-link__label {
  flex: 1;
  min-width: 0;
}

.x-row-link__chevron {
  color: var(--faint);
}
</style>
