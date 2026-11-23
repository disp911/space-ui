<script setup lang="ts">
import { useId } from 'vue'
import XIcon from './XIcon.vue'
import XSpinner from './XSpinner.vue'
import type { IconName } from './icons'

/**
 * A row of XActionList: a title, a description (the slot) and a square
 * accent button. Only the button reacts to the pointer and clicks; the
 * text beside it is plain.
 */
defineProps<{ title: string; icon: IconName; loading?: boolean; disabled?: boolean }>()
const emit = defineEmits<{ click: [MouseEvent] }>()
const textId = useId()
</script>

<template>
  <div class="x-action-row">
    <div class="x-action-row__body">
      <span class="x-action-row__title">{{ title }}</span>
      <span v-if="$slots.default" :id="textId" class="x-action-row__text"><slot /></span>
    </div>
    <button
      type="button"
      class="x-action-row__icon"
      :aria-label="title"
      :aria-describedby="$slots.default ? textId : undefined"
      :disabled="disabled"
      :aria-disabled="loading || undefined"
      :aria-busy="loading || undefined"
      @click="emit('click', $event)"
    >
      <XSpinner v-if="loading" :size="14" />
      <XIcon v-else :name="icon" :size="16" :stroke="1.6" class="x-action-row__glyph" />
    </button>
  </div>
</template>

<style scoped>
.x-action-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  color: var(--fg);
}

.x-action-row__body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.x-action-row__title {
  font: 600 var(--fs-13) / var(--lh-tight) var(--font-sans);
  letter-spacing: -.15px;
}

.x-action-row__text {
  color: var(--mut);
  font: 400 var(--fs-12) / var(--lh-text) var(--font-sans);
  text-wrap: pretty;
}

/* Inline code in the description, e.g. the .db extension. */
.x-action-row__text :deep(code) {
  color: var(--fg2);
  font-family: var(--font-mono);
}

.x-action-row__glyph {
  width: var(--btn-icon);
  height: var(--btn-icon);
}

.x-action-row__icon {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: var(--btn-sq);
  height: var(--btn-sq);
  padding: 0;
  border: 0;
  border-radius: var(--btn-sq-r);
  background: var(--acc);
  color: var(--acc-ink);
  cursor: pointer;
  outline: none;
  transition: background-color var(--t-fast), box-shadow var(--t-fast), transform var(--t-fast);
}

.x-action-row__icon:active:not(:disabled) {
  transform: scale(var(--press-scale-icon));
}

.x-action-row.is-hover .x-action-row__icon {
  background: var(--acc-hover);
  box-shadow: 0 0 0 4px var(--acc-ring), 0 0 18px var(--acc-glow);
}

@media (hover: hover) {
  .x-action-row__icon:hover:not(:disabled) {
    background: var(--acc-hover);
    box-shadow: 0 0 0 4px var(--acc-ring), 0 0 18px var(--acc-glow);
  }
}

.x-action-row__icon:focus-visible,
.x-action-row.is-focus .x-action-row__icon {
  box-shadow: 0 0 0 3px var(--acc-ring);
}

.x-action-row__icon[aria-disabled='true'] {
  pointer-events: none;
}

.x-action-row__icon:disabled {
  cursor: not-allowed;
  opacity: .5;
}

@media (max-width: 767px) {
  .x-action-row {
    padding: 16px 14px;
  }

  .x-action-row__title {
    font-size: var(--fs-14);
  }
}
</style>
