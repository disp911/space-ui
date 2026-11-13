<script setup lang="ts">
import XChip from './XChip.vue'
import XCopyChip from './XCopyChip.vue'
import type { Tone } from './types'

/**
 * Text to take away, e.g. a share link or a WireGuard config: its name in a
 * chip with a copy chip and any further actions (the actions slot), then the
 * text whole in a frame, mono, kept as written and broken anywhere.
 */
withDefaults(defineProps<{ label: string; text: string; tone?: Tone }>(), { tone: 'ok' })
</script>

<template>
  <div class="x-copy-block">
    <div class="x-copy-block__head">
      <XChip :tone="tone">{{ label }}</XChip>
      <XCopyChip :text="text" />
      <slot name="actions" />
    </div>
    <div class="x-copy-block__text">{{ text }}</div>
  </div>
</template>

<style scoped>
.x-copy-block {
  display: flex;
  flex-direction: column;
  gap: var(--sp-6);
  min-width: 0;
}

.x-copy-block__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--sp-6);
}

.x-copy-block__text {
  padding: var(--sp-8) var(--sp-10);
  border: 1px solid var(--div);
  border-radius: var(--nest-r);
  background: var(--tint);
  color: var(--fg2);
  font: 400 var(--fs-11) / var(--lh-code) var(--font-mono);
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
