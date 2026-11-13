<script setup lang="ts">
import XStatusDot from './XStatusDot.vue'

export interface PickItem {
  key: string
  label: string
  /** The entry in use: tinted and marked with a dot; picking it does nothing. */
  current?: boolean
}

/**
 * A framed list that scrolls inside, to pick one entry from, e.g. a version.
 * Rows look like menu items. While loading the rows are grey bars; with no
 * entries the slot is shown instead (an XEmpty).
 */
defineProps<{ items: PickItem[]; loading?: boolean; currentLabel?: string }>()
const emit = defineEmits<{ pick: [key: string] }>()

function pick(item: PickItem) {
  if (!item.current) emit('pick', item.key)
}

// Ten rows, the height of ten entries, so the window does not change size when they arrive; bar widths vary like real entries.
const skeleton = [34, 30, 38, 28, 32, 36, 30, 26, 34, 30]
</script>

<template>
  <div class="x-pick">
    <div class="x-pick__scroll" :aria-busy="loading || undefined">
      <div v-if="loading" class="x-skeleton" aria-hidden="true">
        <div v-for="(w, i) in skeleton" :key="i" class="x-pick__item x-pick__item--skeleton">
          <span class="x-skeleton__bar" :style="{ width: `${w}%` }" />
        </div>
      </div>
      <div v-else-if="!items.length" class="x-pick__empty"><slot /></div>
      <div v-else class="x-pick__list">
        <button
          v-for="item in items"
          :key="item.key"
          type="button"
          :class="['x-popup-item', 'x-pick__item', { 'is-selected': item.current }]"
          :aria-current="item.current || undefined"
          @click="pick(item)"
        >
          <span class="x-pick__label">{{ item.label }}</span>
          <XStatusDot v-if="item.current" size="sm" blink :label="currentLabel" />
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Shrinks to the room left in a window, scrolling inside instead of the window. */
.x-pick {
  display: flex;
  min-width: 0;
  min-height: 0;
  padding: 4px 2px 4px 0;
  border: 1px solid var(--field-line);
  border-radius: var(--nest-r);
  background: var(--field);
  clip-path: inset(0 round var(--nest-r));
}

.x-pick__scroll {
  flex: 1;
  min-width: 0;
  max-height: 360px;
  padding: 0 2px 0 4px;
  overflow-y: auto;
  /* Longer lists scroll here; the room for the scrollbar is kept, so rows do not move when it appears. */
  scrollbar-gutter: stable;
}

.x-pick__list {
  display: flex;
  flex-direction: column;
}

.x-pick__item {
  font: 500 var(--fs-13) / var(--lh-tight) var(--font-mono);
}

.x-pick__item.is-selected {
  cursor: default;
}

.x-pick__item--skeleton {
  display: flex;
  align-items: center;
  min-height: var(--menu-item-h);
  padding: 0 10px;
}

.x-pick__label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.x-pick__empty {
  display: flex;
  flex-direction: column;
}

@media (max-width: 767px) {
  .x-pick__scroll {
    max-height: 430px;
  }

  /* As phone menu items. */
  .x-pick__item {
    border-radius: var(--r-10);
  }
}
</style>
