<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import XButton from './XButton.vue'
import XIcon from './XIcon.vue'
import XInlineButton from './XInlineButton.vue'
import { useFloating } from './floating'

/**
 * A red square button that opens what a program printed when it failed to
 * start: a title over the output in mono, kept as it was printed. The panel
 * opens under the button, or above it at the bottom of a phone screen.
 */
const props = defineProps<{ title: string; text: string }>()
const open = defineModel<boolean>('open', { default: false })
const { t } = useI18n()

const id = useId()
const anchor = ref<HTMLElement | null>(null)
const panel = ref<HTMLElement | null>(null)
// A page may hold the button twice, for desktop and for phones: only the one on screen opens.
const shown = computed(() => open.value && !!anchor.value?.offsetParent)
const { style } = useFloating(anchor, panel, shown, { placement: 'bottom-end', onDismiss: () => (open.value = false) })
watch(() => props.text, (text) => !text && (open.value = false))
</script>

<template>
  <span ref="anchor" class="x-process-error">
    <XButton variant="danger" icon="alert" :label="title" :aria-expanded="shown" :aria-controls="`${id}-panel`" @click="open = !open" />
  </span>
  <Teleport to="body">
    <Transition name="x-popup">
      <div v-if="shown" :id="`${id}-panel`" ref="panel" role="dialog" :aria-label="title" class="x-process-error__panel" :style="style">
        <div class="x-process-error__head">
          <span class="x-process-error__mark"><XIcon name="exclaim" :size="12" :stroke="1.8" /></span>
          <span class="x-process-error__title">{{ title }}</span>
          <XInlineButton icon="close" :label="t('kit.close')" :icon-size="12" :stroke="1.7" @click="open = false" />
        </div>
        <pre class="x-process-error__text">{{ text }}</pre>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.x-process-error {
  display: inline-flex;
  flex: none;
}

.x-process-error__panel {
  z-index: 1000;
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: min(560px, calc(100vw - 16px));
  padding: 12px;
  border: 1px solid var(--line);
  border-radius: var(--nest-r);
  background: var(--dlg);
  box-shadow: var(--shadow-pop);
}

.x-process-error__head {
  display: flex;
  align-items: center;
  gap: var(--sp-8);
  margin-left: 2px;
}

.x-process-error__mark {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: var(--r-6);
  background: var(--err-bg);
  color: var(--err-t);
}

.x-process-error__title {
  flex: 1;
  min-width: 0;
  font: 600 var(--fs-14) / var(--lh-tight) var(--font-sans);
}

.x-process-error__text {
  min-height: 0;
  max-height: 300px;
  margin: 0;
  padding: 10px 12px;
  overflow: auto;
  border: 1px solid var(--div);
  border-radius: var(--nest-r);
  background: var(--nest);
  color: var(--fg2);
  font: 400 var(--fs-11) / var(--lh-code) var(--font-mono);
  white-space: pre-wrap;
  word-break: break-word;
}

@media (max-width: 767px) {
  .x-process-error__panel {
    width: calc(100vw - 28px);
  }
}
</style>
