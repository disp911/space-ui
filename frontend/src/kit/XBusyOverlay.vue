<script setup lang="ts">
import { watch } from 'vue'
import XSpinner from './XSpinner.vue'
import { lockScroll } from './scrollLock'

/**
 * Covers the page while something it cannot be used during runs, e.g. a panel
 * restart: a spinner, the title, a line of what happens now and a 4px bar.
 * progress in percent fills the bar; without it the bar runs on its own.
 */
const props = defineProps<{ open: boolean; title: string; text?: string; progress?: number }>()

watch(() => props.open, (on, was) => on !== !!was && lockScroll(on), { immediate: true })
</script>

<template>
  <Teleport to="body">
    <Transition name="x-busy">
      <div v-if="open" class="x-busy">
        <div class="x-busy__card" role="status" aria-live="polite">
          <div class="x-busy__head">
            <XSpinner :size="16" />
            <div class="x-busy__text">
              <div class="x-busy__title">{{ title }}</div>
              <div v-if="text" class="x-busy__line">{{ text }}</div>
            </div>
          </div>
          <div class="x-busy__track">
            <div :class="['x-busy__bar', { 'is-running': progress === undefined }]" :style="progress !== undefined ? { width: `${progress}%` } : undefined" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.x-busy {
  position: fixed;
  inset: 0;
  z-index: 1100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: var(--scrim);
  /* Blurred on every screen, unlike the windows: nothing can be done under it. */
  backdrop-filter: var(--scrim-blur);
}

.x-busy__card {
  width: 100%;
  max-width: 420px;
  padding: 20px;
  border: 1px solid var(--line);
  border-radius: var(--win-r);
  background: var(--dlg);
  box-shadow: var(--shadow-modal);
}

.x-busy__head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  color: var(--acc-t);
}

.x-busy__text {
  flex: 1;
  min-width: 0;
}

.x-busy__title {
  color: var(--fg);
  font: 600 var(--fs-16) / var(--lh-tight) var(--font-sans);
  letter-spacing: -.2px;
}

.x-busy__line {
  margin-top: 2px;
  color: var(--mut);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-sans);
}

.x-busy__track {
  height: 4px;
  overflow: hidden;
  border-radius: var(--r-pill);
  background: var(--track);
}

.x-busy__bar {
  height: 100%;
  border-radius: var(--r-pill);
  background: var(--acc);
  transition: width .9s linear;
}

.x-busy__bar.is-running {
  width: 30%;
  animation: x-busy-run 1.4s ease-in-out infinite;
}

@keyframes x-busy-run {
  from { transform: translateX(-100%); }
  to { transform: translateX(340%); }
}

.x-busy-enter-active,
.x-busy-leave-active {
  transition: opacity var(--t-med);
}

.x-busy-enter-from,
.x-busy-leave-to {
  opacity: 0;
}
</style>
