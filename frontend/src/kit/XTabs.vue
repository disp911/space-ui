<script setup lang="ts" generic="K extends string">
import { nextTick, onMounted, ref, watch } from 'vue'
import { useEdgeFade } from './edgeFade'
import { useSlidingThumb } from './slidingThumb'

export interface TabItem<K extends string = string> {
  key: K
  label: string
  /** A dot after the label: unsaved changes (mod) or errors (err) inside the tab. */
  dot?: 'mod' | 'err'
}

/**
 * Page tabs. On wide screens labels sit on a line with an accent ink under the
 * current one; on phones they are pills in a strip that scrolls sideways.
 */
const props = defineProps<{
  items: TabItem<K>[]
  label?: string
  /** While the page loads: the tabs show but do not switch. */
  disabled?: boolean
}>()
const model = defineModel<K>({ required: true })

const strip = ref<HTMLElement | null>(null)
// The accent ink travels to the current tab like liquid.
const ink = useSlidingThumb(strip, '.is-current', [model])
// A strip that scrolls fades out at an edge with more tabs beyond it.
const { classes: fade, update: updateFade } = useEdgeFade(strip)
watch(() => props.items, () => nextTick(updateFade), { deep: true })

// In a strip that scrolls the current tab stays in sight.
function reveal() {
  strip.value?.querySelector<HTMLElement>('.is-current')?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
}
onMounted(reveal)
watch(model, () => nextTick(reveal))

function onKeydown(e: KeyboardEvent, items: TabItem<K>[]) {
  const i = items.findIndex((t) => t.key === model.value)
  const by = e.key === 'ArrowRight' ? 1 : e.key === 'ArrowLeft' ? -1 : 0
  if (!by) return
  e.preventDefault()
  const next = items[(i + by + items.length) % items.length]
  if (next) model.value = next.key
  nextTick(() => strip.value?.querySelector<HTMLElement>('.is-current')?.focus())
}
</script>

<template>
  <div class="x-tabs">
    <div ref="strip" :class="['x-tabs__strip', fade]" role="tablist" :aria-label="label" @keydown="onKeydown($event, items)">
      <button
        v-for="tab in items"
        :key="tab.key"
        type="button"
        role="tab"
        :aria-selected="tab.key === model"
        :tabindex="tab.key === model ? 0 : -1"
        :disabled="disabled"
        :class="['x-tabs__tab', { 'is-current': tab.key === model }]"
        @click="model = tab.key"
      >
        <!-- The bold copy keeps the width the same when the tab becomes current. -->
        <span class="x-tabs__label" :data-label="tab.label"><span>{{ tab.label }}</span></span>
        <span v-if="tab.dot" :class="['x-tabs__dot', `x-tabs__dot--${tab.dot}`]" />
      </button>
      <span v-if="ink" :class="['x-tabs__ink', ink.ahead && `x-slide--${ink.ahead}`]" :style="{ left: `${ink.left}px`, right: `${ink.right}px` }" />
    </div>
  </div>
</template>

<style scoped>
.x-tabs {
  position: relative;
  width: 100%;
  min-width: 0;
  /* Takes the width it is given: a long strip scrolls instead of widening the page. */
  contain: inline-size;
}

.x-tabs__strip {
  position: relative;
  display: flex;
  gap: 24px;
  border-bottom: 1px solid var(--div);
}

.x-tabs__tab {
  position: relative;
  isolation: isolate;
  display: flex;
  flex: none;
  align-items: center;
  gap: 8px;
  height: 40px;
  margin-bottom: -1px;
  padding: 0 1px;
  border: 0;
  background: none;
  color: var(--mut);
  font: 500 var(--fs-13) / 1 var(--font-sans);
  white-space: nowrap;
  cursor: pointer;
  outline: none;
  transition: color var(--t-fast), opacity var(--t-fast);
}

/* A tab has no box to darken: it dims under the finger, as a segment does. */
.x-tabs__tab:active:not(:disabled) {
  opacity: .75;
}

.x-tabs__tab:hover:not(:disabled),
.x-tabs__tab.is-current {
  color: var(--fg);
}

.x-tabs__tab:disabled {
  opacity: .5;
  cursor: default;
}

.x-tabs__tab.is-current {
  font-weight: 600;
}

/* Keyboard focus: the label turns accent on a soft tint, no ring. */
.x-tabs__tab::before {
  content: '';
  position: absolute;
  inset: 6px -8px;
  z-index: -1;
  border-radius: var(--r-8);
  background: var(--acc-soft);
  opacity: 0;
  pointer-events: none;
  transition: opacity var(--t-fast);
}

.x-tabs__tab:focus-visible {
  color: var(--acc-t);
}

.x-tabs__tab:focus-visible::before {
  opacity: 1;
}

.x-tabs__label {
  display: inline-grid;
}

.x-tabs__label > span,
.x-tabs__label::after {
  grid-area: 1 / 1;
}

.x-tabs__label::after {
  content: attr(data-label);
  height: 0;
  overflow: hidden;
  font-weight: 600;
  visibility: hidden;
}

.x-tabs__dot {
  flex: none;
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.x-tabs__dot--mod {
  background: var(--mod);
}

.x-tabs__dot--err {
  background: var(--err);
}

.x-tabs__ink {
  position: absolute;
  bottom: -1px;
  height: 2px;
  border-radius: var(--r-pill);
  background: var(--acc);
  pointer-events: none;
}

/*
 * Phones: the same labels and ink in a strip that scrolls sideways, fading out
 * at an edge with more tabs beyond it. The fade is a mask on the strip, so the
 * line is drawn under it by the frame and stays whole; the ink sits on it.
 */
@media (max-width: 767px) {
  .x-tabs {
    box-shadow: inset 0 -1px var(--div);
  }

  .x-tabs__strip {
    padding: 0 14px;
    overflow-x: auto;
    /* The current tab scrolls into view clear of the fade. */
    scroll-padding-inline: 32px;
    border-bottom: 0;
  }

  .x-tabs__tab {
    margin-bottom: 0;
  }

  .x-tabs__ink {
    bottom: 0;
  }
}
</style>
