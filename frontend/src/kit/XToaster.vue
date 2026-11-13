<script setup lang="ts">
import XIcon from './XIcon.vue'
import { dismissToast, toasts } from './toast'

/** Mount once in the app; toast() from ./toast adds messages. */
</script>

<template>
  <div class="x-toaster" role="status" aria-live="polite">
    <TransitionGroup name="x-toast">
      <div v-for="item in toasts" :key="item.id" class="x-toast" @click="dismissToast(item.id)">
        <span :class="['x-toast__icon', `x-toast__icon--${item.kind}`]">
          <XIcon :name="item.kind === 'ok' ? 'check' : item.kind === 'warn' ? 'warning' : 'alert'" :size="12" :stroke="item.kind === 'ok' ? 2 : 1.6" />
        </span>
        <span class="x-toast__text">{{ item.text }}</span>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.x-toaster {
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 1200;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
  max-width: min(420px, calc(100vw - 28px));
  pointer-events: none;
}

.x-toast {
  display: flex;
  align-items: center;
  gap: 10px;
  max-width: 100%;
  padding: 10px 14px 10px 10px;
  border: 1px solid var(--line);
  border-radius: var(--nest-r);
  background: var(--dlg);
  box-shadow: var(--shadow-pop);
  pointer-events: auto;
  cursor: default;
}

.x-toast__icon {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: var(--r-6);
}

.x-toast__icon--ok {
  background: var(--ok-bg);
  color: var(--ok-t);
}

.x-toast__icon--warn {
  background: var(--warn-bg);
  color: var(--warn);
}

.x-toast__icon--err {
  background: var(--err-bg);
  color: var(--err-t);
}

.x-toast__text {
  color: var(--fg);
  font: 500 var(--fs-13) / var(--lh-text) var(--font-sans);
  text-wrap: pretty;
}

.x-toast-enter-active,
.x-toast-leave-active {
  transition: opacity var(--t-med), transform var(--t-med) var(--ease);
}

.x-toast-enter-from,
.x-toast-leave-to {
  opacity: 0;
  transform: translateY(14px);
}

@media (max-width: 767px) {
  .x-toaster {
    right: 14px;
    /* Above a page's action bar while one is shown. */
    bottom: var(--toast-bottom, calc(14px + env(safe-area-inset-bottom)));
    left: 14px;
    max-width: none;
  }
}
</style>
