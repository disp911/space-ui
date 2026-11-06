<script setup lang="ts">
import { computed } from 'vue'
import { useStack } from './media'

/**
 * Fields of a form in columns, 12px apart and lined up at their bottom, e.g.
 * in a window: cols equal columns on desktop, phone of them on phones (one,
 * or two for four columns). columns and phoneColumns give other tracks, e.g.
 * an address beside a narrow port. A child with the class x-wide takes the
 * whole row, one with x-wide-phone only on phones. One column is a stack.
 * top lines them up at their top instead, e.g. a field beside a taller list.
 * tight sets them 8px apart, e.g. the rows of a list of fields. In a window
 * that stacks its fields there is one column, on desktop too; phoneColumns
 * stay, as they hold one control made of parts, e.g. a field and its button.
 */
const props = withDefaults(
  defineProps<{ cols?: 1 | 2 | 3 | 4; phone?: 1 | 2 | 4; columns?: string; phoneColumns?: string; top?: boolean; tight?: boolean }>(),
  { cols: 2 },
)

const stack = useStack()
const even = (n: number) => `repeat(${n}, minmax(0, 1fr))`
const phoneTrack = computed(() => props.phoneColumns ?? even(stack ? 1 : (props.phone ?? (props.cols === 4 ? 2 : 1))))
const desktop = computed(() => (stack ? phoneTrack.value : (props.columns ?? even(props.cols))))
</script>

<template>
  <div :class="['x-field-grid', { 'x-field-grid--top': top, 'x-field-grid--tight': tight }]"><slot /></div>
</template>

<style scoped>
.x-field-grid {
  display: grid;
  grid-template-columns: v-bind(desktop);
  align-items: end;
  gap: 12px;
  min-width: 0;
}

.x-field-grid--top {
  align-items: start;
}

.x-field-grid--tight {
  gap: var(--sp-8);
}

.x-field-grid > :slotted(.x-wide) {
  grid-column: 1 / -1;
}

@media (max-width: 767px) {
  .x-field-grid {
    grid-template-columns: v-bind(phoneTrack);
  }

  .x-field-grid > :slotted(.x-wide-phone) {
    grid-column: 1 / -1;
  }
}
</style>
