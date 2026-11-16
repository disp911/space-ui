<script setup lang="ts">
import XIcon from './XIcon.vue'

/**
 * A row's name that opens what the row holds, on phones in XExpandTable: a
 * chevron and the name, e.g. an inbound's remark opening its clients. A row
 * that holds nothing shows the name in the same place, without the chevron.
 */
defineProps<{ open?: boolean; expandable?: boolean }>()
defineEmits<{ toggle: [] }>()
</script>

<template>
  <button v-if="expandable" type="button" class="x-row-toggle" :aria-expanded="open" @click="$emit('toggle')">
    <XIcon name="chevron" :size="12" :stroke="1.8" :class="['x-row-toggle__chevron', { 'is-open': open }]" />
    <span class="x-row-toggle__name"><slot /></span>
  </button>
  <div v-else class="x-row-toggle x-row-toggle--still">
    <span class="x-row-toggle__name"><slot /></span>
  </div>
</template>

<style scoped>
.x-row-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  /* The whole row is the target: as tall as the table's row, 48px on phones. */
  min-height: max(var(--menu-item-h), var(--row-h, 0px));
  padding: 6px 8px 6px 6px;
  border: 0;
  border-radius: var(--ctl-r);
  background: transparent;
  color: inherit;
  text-align: left;
  outline: none;
}

.x-row-toggle:focus-visible {
  box-shadow: inset 0 0 0 2px var(--acc-line);
}

.x-row-toggle:active:not(.x-row-toggle--still) {
  background: var(--press);
}

/* The name stands where it would after a chevron. */
.x-row-toggle--still {
  padding-left: 26px;
}

.x-row-toggle__chevron {
  flex: none;
  margin-top: 2px;
  color: var(--dim);
  transition: transform var(--t-med) var(--ease);
}

.x-row-toggle__chevron.is-open {
  transform: rotate(90deg);
}

.x-row-toggle__name {
  min-width: 0;
}
</style>
