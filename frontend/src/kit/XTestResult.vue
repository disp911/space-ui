<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import XIcon from './XIcon.vue'
import XSpinner from './XSpinner.vue'
import XTooltip from './XTooltip.vue'

/**
 * What a connection test gave: a dash before any, a spinner while it runs,
 * the delay and the HTTP status in accent, or a failure in red that shows
 * the error on hover, focus or tap.
 */
defineProps<{ state: 'idle' | 'testing' | 'ok' | 'fail'; delay?: number; status?: number; error?: string }>()
const { t } = useI18n()
</script>

<template>
  <span v-if="state === 'testing'" class="x-test-result x-test-result--testing" role="status" :aria-label="t('kit.testing')">
    <XSpinner :size="16" />
  </span>
  <span v-else-if="state === 'ok'" class="x-test-result x-test-result--ok">
    <XIcon name="check" :size="12" :stroke="1.8" />{{ t('kit.delayMs', { n: delay ?? 0 }) }}<template v-if="status"> · {{ status }}</template>
  </span>
  <XTooltip v-else-if="state === 'fail'" :text="error" error :delay="0">
    <button type="button" class="x-test-result x-test-result--fail">
      <XIcon name="close" :size="12" :stroke="1.8" />{{ t('kit.testFailed') }}
    </button>
  </XTooltip>
  <span v-else class="x-test-result x-test-result--idle">—</span>
</template>

<style scoped>
.x-test-result {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-4);
  min-width: 0;
  font: 500 var(--fs-12) / var(--lh-tight) var(--font-mono);
  white-space: nowrap;
  transition: transform var(--t-fast);
}

button.x-test-result:active {
  transform: scale(var(--press-scale-chip));
}

.x-test-result--testing {
  color: var(--acc-t);
}
.x-test-result--ok {
  color: var(--ok-t);
}

.x-test-result--idle {
  color: var(--faint);
  font-weight: 400;
}

.x-test-result--fail {
  padding: 2px 4px;
  margin: -2px -4px;
  border: 0;
  border-radius: var(--r-4);
  background: transparent;
  color: var(--err-t);
  font-family: var(--font-sans);
  cursor: help;
  outline: none;
  transition: background-color var(--t-fast);
}

.x-test-result--fail:hover,
.x-test-result--fail:focus-visible {
  background: var(--err-bg);
}
</style>
