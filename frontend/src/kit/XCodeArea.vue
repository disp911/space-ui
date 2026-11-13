<script setup lang="ts">
import { computed, ref } from 'vue'
import { scrollbarCursor } from './scrollbarCursor'

/**
 * Monospace text with line numbers, e.g. config.json: a plain text field, no
 * highlighting. The numbers scroll with the text. A height in px, or none to
 * fill the column it sits in; phoneFill drops the height on phones, where a
 * full-screen window gives the text all the room left. While loading the
 * lines are grey bars. plain drops the numbers, e.g. a list of links;
 * autofocus has a window put the cursor in it, e.g. text to paste.
 */
defineProps<{ height?: number; phoneFill?: boolean; loading?: boolean; readonly?: boolean; label?: string; plain?: boolean; autofocus?: boolean }>()
const text = defineModel<string>({ default: '' })

const lines = computed(() => text.value.split('\n').length)
const gutter = ref<HTMLElement | null>(null)

function sync(e: Event) {
  if (gutter.value) gutter.value.scrollTop = (e.target as HTMLElement).scrollTop
}

// Bar widths vary like real lines.
const skeleton = [14, 30, 46, 38, 24, 52, 44, 30, 20, 48, 36, 26, 42, 18]
</script>

<template>
  <div :class="['x-code', { 'x-code--phone-fill': phoneFill }]" :style="{ '--code-h': height ? `${height}px` : undefined }">
    <div v-if="loading" class="x-code__skeleton x-skeleton" aria-hidden="true">
      <span v-for="(w, i) in skeleton" :key="i" class="x-skeleton__bar" :style="{ width: `${w}%` }" />
    </div>
    <template v-else>
      <div v-if="!plain" ref="gutter" class="x-code__gutter" aria-hidden="true">
        <div v-for="n in lines" :key="n">{{ n }}</div>
      </div>
      <textarea
        v-model="text"
        class="x-code__text"
        spellcheck="false"
        wrap="off"
        :readonly="readonly"
        :autofocus="autofocus || undefined"
        :aria-label="label"
        @scroll="sync"
        @pointermove="scrollbarCursor"
        @pointerleave="scrollbarCursor"
      />
    </template>
  </div>
</template>

<style scoped>
.x-code {
  display: flex;
  flex: 1 1 auto;
  min-width: 0;
  height: var(--code-h, auto);
  min-height: 0;
  padding: 4px 2px 4px 0;
  border: 1px solid var(--field-line);
  border-radius: var(--nest-r);
  background: var(--bg);
  clip-path: inset(0 round var(--nest-r));
  font: 400 14px / 20px var(--font-code);
  transition: border-color var(--t-fast);
}

.x-code:focus-within {
  border-color: var(--acc-line);
}

.x-code__gutter {
  flex: none;
  min-width: 30px;
  height: 100%;
  padding: 6px 6px 24px;
  overflow: hidden;
  border-right: 1px solid var(--div);
  color: var(--faint);
  text-align: right;
  font-variant-numeric: tabular-nums;
  user-select: none;
}

.x-code__text {
  display: block;
  flex: 1;
  min-width: 0;
  height: 100%;
  margin: 0;
  padding: 6px 12px;
  overflow: auto;
  border: 0;
  outline: none;
  background: transparent;
  color: var(--fg2);
  font: inherit;
  white-space: pre;
  resize: none;
  tab-size: 2;
}

@media (max-width: 767px) {
  .x-code--phone-fill {
    height: auto;
  }
}

.x-code__skeleton {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 12px;
  padding: 12px 16px;
}
</style>
