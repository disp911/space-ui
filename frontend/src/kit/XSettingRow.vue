<script setup lang="ts">
import XIcon from './XIcon.vue'

/**
 * One setting: the title and a description of up to two lines on the left, the
 * control (the slot) in a 400px column on the right; on phones one column, or
 * the title beside a narrow control. The row looks at its own width, not the
 * screen's: in a narrow window it stands in one column as on phones. A changed setting gets a blue dot, a tint
 * and a bar at the left; an error a red bar and its text under the control.
 * The note slot adds lines under the control, e.g. the address after a restart;
 * the nested slot spans the whole row below, e.g. an XNest a switch opens.
 */
withDefaults(
  defineProps<{
    title: string
    description?: string
    modified?: boolean
    error?: string
    /** A switch: kept at the right, beside the title on phones too. */
    inline?: boolean
    /** A short value, e.g. a port: 140px at the right; on phones the full width under the title. */
    narrow?: boolean
    /** A short choice, e.g. the language: 220px at the right; on phones the full width under the title. */
    compact?: boolean
    /** A tall control: the title stays at its top. */
    tall?: boolean
  }>(),
  {},
)
</script>

<template>
  <div :class="['x-setting', { 'is-modified': modified, 'is-invalid': !!error, 'x-setting--inline': inline, 'x-setting--narrow': narrow, 'x-setting--compact': compact, 'x-setting--tall': tall }]">
    <div class="x-setting__grid">
      <div class="x-setting__text">
        <div class="x-setting__head">
          <span class="x-setting__title">{{ title }}</span>
          <span v-if="modified" class="x-setting__dot" aria-hidden="true" />
        </div>
        <p v-if="description" class="x-setting__description">{{ description }}</p>
      </div>
      <div class="x-setting__control">
        <slot />
        <p v-if="error" class="x-setting__error" role="alert"><XIcon name="alert" :size="12" :stroke="1.6" />{{ error }}</p>
        <slot name="note" />
      </div>
      <div v-if="$slots.nested" class="x-setting__nested"><slot name="nested" /></div>
    </div>
  </div>
</template>

<style scoped>
.x-setting {
  /* The columns below follow the row's width. */
  container-type: inline-size;
  padding: 14px 20px;
  border-top: 1px solid var(--div);
  transition: background-color var(--t-med);
}

/* The row right after a group label has no line of its own. */
:where(.x-setting-divider) + .x-setting {
  border-top: 0;
}

.x-setting__grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 400px;
  align-items: center;
  column-gap: 40px;
}

.x-setting--tall .x-setting__grid {
  align-items: start;
}

.x-setting.is-modified {
  background: var(--mod-bg);
  box-shadow: inset 3px 0 0 var(--mod);
}

.x-setting.is-invalid {
  box-shadow: inset 3px 0 0 var(--err);
}

.x-setting__text {
  min-width: 0;
}

.x-setting__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 8px;
}

.x-setting__title {
  color: var(--fg);
  font: 500 var(--fs-14) / var(--lh-tight) var(--font-sans);
  text-wrap: pretty;
}

.x-setting__dot {
  flex: none;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--mod);
}

.x-setting__description {
  display: -webkit-box;
  margin: 4px 0 0;
  overflow: hidden;
  color: var(--mut);
  font: 400 var(--fs-12) / var(--lh-text) var(--font-sans);
  text-wrap: pretty;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.x-setting__control {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 6px;
  min-width: 0;
}

.x-setting--inline .x-setting__control,
.x-setting--narrow .x-setting__control,
.x-setting--compact .x-setting__control {
  align-items: flex-end;
}

.x-setting--narrow .x-setting__control > :first-child {
  width: 140px;
}

.x-setting--compact .x-setting__control > :first-child {
  width: 220px;
}

.x-setting__nested {
  grid-column: 1 / -1;
  min-width: 0;
  margin-top: 14px;
}

.x-setting__error {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin: 0;
  color: var(--err-t);
  font: 400 var(--fs-12) / var(--lh-text) var(--font-sans);
  text-wrap: pretty;
}

.x-setting__error :deep(svg) {
  flex: none;
  margin-top: 3px;
}

/* Too narrow for a 200px title beside the 400px control: one column, as on phones. */
@container (max-width: 639px) {
  .x-setting__grid {
    grid-template-columns: minmax(0, 1fr);
    row-gap: 10px;
  }

  .x-setting--inline .x-setting__grid {
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: center;
    column-gap: 14px;
  }

  .x-setting--narrow .x-setting__control,
  .x-setting--compact .x-setting__control {
    align-items: stretch;
  }

  .x-setting--narrow .x-setting__control > :first-child,
  .x-setting--compact .x-setting__control > :first-child {
    width: auto;
  }

  .x-setting__nested {
    margin-top: 4px;
  }
}

@media (max-width: 767px) {
  .x-setting {
    padding: 14px 16px;
  }
}
</style>
