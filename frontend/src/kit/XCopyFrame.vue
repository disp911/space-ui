<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { copyText } from './clipboard'
import { toast } from './toast'
import XChip from './XChip.vue'
import type { Tone } from './types'

/**
 * Text to take away in a frame with its name on the frame's top edge; a
 * click anywhere on it copies the text, said in a toast, e.g. the links of
 * the subscription page. The text is mono, kept as written, broken anywhere.
 */
const props = withDefaults(defineProps<{ label: string; text: string; tone?: Tone }>(), { tone: 'unl' })
const { t } = useI18n()

async function copy() {
  const ok = await copyText(props.text)
  toast(ok ? 'ok' : 'err', t(ok ? 'kit.copied' : 'kit.copyFailed'))
}
</script>

<template>
  <div class="x-copy-frame">
    <span class="x-copy-frame__label"><XChip :tone="tone" truncate :title="label">{{ label }}</XChip></span>
    <button type="button" class="x-copy-frame__text" :aria-label="`${t('kit.copy')}: ${label}`" @click="copy">{{ text }}</button>
  </div>
</template>

<style scoped>
.x-copy-frame {
  position: relative;
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding-top: calc(var(--chip-h) / 2);
}

/* The name sits on the edge; the card shows under it so the line stops at the chip. */
.x-copy-frame__label {
  position: absolute;
  top: 0;
  left: 50%;
  z-index: 1;
  max-width: calc(100% - 2 * var(--sp-16));
  border-radius: var(--r-6);
  background: var(--card);
  transform: translateX(-50%);
}

.x-copy-frame__text {
  padding: var(--sp-20) var(--sp-16) var(--sp-14);
  border: 1px solid var(--div);
  border-radius: var(--nest-r);
  background: var(--tint);
  color: var(--fg2);
  font: 400 var(--fs-12) / var(--lh-code) var(--font-mono);
  text-align: left;
  white-space: pre-wrap;
  word-break: break-all;
  cursor: pointer;
  outline: none;
  transition: border-color var(--t-fast), background-color var(--t-fast);
}

.x-copy-frame__text:active {
  background-color: var(--press);
}

.x-copy-frame__text:hover {
  border-color: var(--field-hover);
}

.x-copy-frame__text:focus-visible {
  box-shadow: 0 0 0 3px var(--acc-ring);
  border-color: var(--acc-line);
}

@media (max-width: 767px) {
  .x-copy-frame__text {
    padding-inline: var(--sp-14);
  }
}
</style>
