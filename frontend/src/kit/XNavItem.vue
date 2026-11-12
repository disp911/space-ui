<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import XIcon from './XIcon.vue'
import type { IconName } from './icons'

/**
 * Item of the side menu: a link to a section, marked while it is open, or a
 * button (no `to`) such as logging out; danger turns its hover red.
 */
defineProps<{ icon: IconName; label: string; to?: RouteLocationRaw; danger?: boolean }>()
</script>

<template>
  <RouterLink v-if="to" :to="to" class="x-nav-item" exact-active-class="is-current">
    <XIcon :name="icon" :size="16" :stroke="1.4" />
    <span>{{ label }}</span>
  </RouterLink>
  <button v-else type="button" :class="['x-nav-item', { 'x-nav-item--danger': danger }]">
    <XIcon :name="icon" :size="16" :stroke="1.4" />
    <span>{{ label }}</span>
  </button>
</template>

<style scoped>
/* Over XNav's thumb, which carries the current item's tint. */
.x-nav-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  height: var(--sidebar-item-h);
  padding: 0 12px;
  border: 0;
  border-radius: var(--ctl-r);
  background: transparent;
  color: var(--mut);
  font: 400 var(--fs-13) / 1 var(--font-sans);
  text-align: left;
  text-decoration: none;
  white-space: nowrap;
  cursor: pointer;
  outline: none;
  transition: color var(--t-fast), opacity var(--t-fast);
}

/* Only XNav's thumb has a box: hover lights the text and icon, a press dims them, as tabs and segments do. */
.x-nav-item:hover,
.x-nav-item.is-hover {
  color: var(--fg);
}

.x-nav-item:active {
  opacity: .75;
}

.x-nav-item:focus-visible,
.x-nav-item.is-focus {
  box-shadow: inset 0 0 0 2px var(--acc-line);
  color: var(--fg);
}

.x-nav-item.is-current {
  color: var(--acc-t);
  font-weight: 500;
}

.x-nav-item--danger {
  color: var(--dim);
}

.x-nav-item--danger:hover,
.x-nav-item--danger.is-hover {
  color: var(--err-t);
}

@media (max-width: 767px) {
  .x-nav-item {
    font-size: var(--fs-14);
  }
}
</style>
