<script setup lang="ts">
import { useId } from 'vue'
import XCollapse from './XCollapse.vue'
import XIcon from './XIcon.vue'

/**
 * Collapsible group of settings. Extra attributes go to the header button.
 * card: a framed card on a faint wash, its body under a line, e.g. the parts
 * of a window.
 */
defineOptions({ inheritAttrs: false })
defineProps<{
  title: string
  description?: string
  card?: boolean
  /** The title is a value, e.g. a protocol: monospace. */
  mono?: boolean
}>()

const open = defineModel<boolean>('open', { default: true })
const id = useId()
</script>

<template>
  <div :class="['x-section', { 'x-section--card': card, 'is-open': open }]">
    <div class="x-section__bar">
      <button
        v-bind="$attrs"
        type="button"
        class="x-section__head"
        :aria-expanded="open"
        :aria-controls="`${id}-body`"
        @click="open = !open"
      >
        <XIcon name="chevron" :size="card ? 14 : 12" :stroke="card ? 1.5 : 1.7" class="x-section__chevron" />
        <span :class="['x-section__title', { 'x-section__title--mono': mono }]">{{ title }}</span>
      </button>
      <div v-if="$slots.meta" class="x-section__meta"><slot name="meta" /></div>
    </div>
    <p v-if="description && open" class="x-section__description">{{ description }}</p>
    <XCollapse v-if="$slots.default" :open="open">
      <div :id="`${id}-body`" class="x-section__body">
        <slot />
      </div>
    </XCollapse>
  </div>
</template>

<style scoped>
.x-section {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.x-section__bar {
  display: flex;
  align-items: center;
  gap: 8px;
}

.x-section__head {
  display: flex;
  flex: 1;
  min-width: 0;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 40px;
  padding: 0 10px;
  border: 0;
  border-radius: var(--r-10);
  background: transparent;
  color: var(--fg);
  font: 600 var(--fs-13) / 1 var(--font-sans);
  text-align: left;
  outline: none;
  transition: background-color var(--t-fast);
}

.x-section__head:hover,
.x-section__head.is-hover {
  background: var(--hov);
}

.x-section__head:active,
.x-section__head.is-active {
  background: var(--press);
}

/* Keyboard focus marks the chevron, not the whole header. */
.x-section__head:focus-visible .x-section__chevron,
.x-section__head.is-focus .x-section__chevron {
  background: var(--acc-soft);
  color: var(--acc-t);
  box-shadow: 0 0 0 4px var(--acc-soft), 0 0 0 5px var(--acc-line);
}

.x-section__chevron {
  border-radius: 2px;
  color: var(--dim);
  /* The focus box is always full size and only fades in, so it never grows out of a square. */
  box-shadow: 0 0 0 4px transparent, 0 0 0 5px transparent;
  transition: transform var(--t-fast) var(--ease), color var(--t-fast), background-color var(--t-fast), box-shadow var(--t-fast);
}

.is-open > .x-section__bar .x-section__chevron {
  transform: rotate(90deg);
}

.x-section__title {
  flex: 1;
  min-width: 0;
}

.x-section__title--mono {
  font-family: var(--font-mono);
}

.x-section__meta {
  display: flex;
  flex: none;
  align-items: center;
  gap: 8px;
  padding-right: 10px;
}

.x-section__description {
  /* Text lines up with the title: header padding, chevron, gap. */
  margin: 0 var(--sp-10) var(--sp-4) calc(var(--sp-10) + var(--icon-s) + var(--sp-8));
  color: var(--dim);
  font: 400 var(--fs-12) / var(--lh-text) var(--font-sans);
}

.x-section__body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 8px 10px 12px calc(var(--sp-10) + var(--icon-s) + var(--sp-8));
}

/* Card: the header fills the top of the frame, the body sits under a line. */
.x-section--card {
  overflow: clip;
  border: 1px solid var(--div);
  border-radius: var(--r-12);
  background: var(--tint);
}

.x-section--card .x-section__head {
  height: 44px;
  padding: 0 var(--sp-6) 0 var(--sp-12);
  border-radius: 0;
}

.x-section--card .x-section__body {
  padding: var(--sp-12);
  border-top: 1px solid var(--div);
}

@media (max-width: 767px) {
  /* A title and its control too wide for one line: the control moves under, to the right. */
  .x-section__bar {
    flex-wrap: wrap;
  }

  .x-section__head {
    flex: 1 0 auto;
    width: auto;
    height: var(--ctl-h);
    font-size: var(--fs-14);
  }

  .x-section__meta {
    margin-left: auto;
  }

  .x-section__body {
    padding-left: 10px;
  }

  .x-section--card .x-section__head {
    height: 50px;
    padding: 0 var(--sp-8) 0 var(--sp-14);
  }

  .x-section--card .x-section__body {
    gap: var(--sp-14);
    padding: var(--sp-14);
  }
}
</style>
