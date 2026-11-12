<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import XIcon from './XIcon.vue'
import XTooltip from './XTooltip.vue'
import { copyText } from './clipboard'
import { toast } from './toast'

/**
 * A small accent square beside a value that puts text on the clipboard: an ID,
 * a key, a link. As tall as a chip on desktop, larger for a finger on phones;
 * the copy sign turns into a check for a moment once copied.
 */
const props = defineProps<{ text: string; label?: string }>()
const { t } = useI18n()

const done = ref(false)
let timer = 0

async function copy() {
  if (!(await copyText(props.text))) return toast('err', t('kit.copyFailed'))
  done.value = true
  clearTimeout(timer)
  timer = window.setTimeout(() => (done.value = false), 1800)
}
onBeforeUnmount(() => clearTimeout(timer))
</script>

<template>
  <XTooltip :text="done ? t('kit.copied') : (label ?? t('kit.copy'))" :delay="0">
    <button type="button" class="x-copy-chip" :aria-label="label ?? t('kit.copy')" @click="copy">
      <XIcon :name="done ? 'check' : 'copy'" :size="12" :stroke="1.6" />
    </button>
  </XTooltip>
</template>

<style scoped>
.x-copy-chip {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: var(--chip-h);
  height: var(--chip-h);
  padding: 0;
  border: 0;
  border-radius: var(--chip-r);
  background: var(--ok-bg);
  color: var(--ok-t);
  cursor: pointer;
  outline: none;
  transition: filter var(--t-fast), transform var(--t-fast), box-shadow var(--t-fast);
}

.x-copy-chip:hover {
  filter: brightness(1.15);
}

.x-copy-chip:focus-visible {
  box-shadow: 0 0 0 2px var(--card), 0 0 0 4px var(--acc-line);
}

.x-copy-chip:active {
  transform: scale(var(--press-scale-chip));
}

@media (max-width: 767px) {
  .x-copy-chip {
    width: 32px;
    height: 32px;
  }
}
</style>
