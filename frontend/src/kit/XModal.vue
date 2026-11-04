<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import XButton from './XButton.vue'
import { lockScroll } from './scrollLock'
import { startSheetDrag } from './sheetDrag'

/**
 * Dialog on desktop, bottom sheet on phones. The sheet has a handle instead
 * of the close button, so its footer always starts with Cancel, before the
 * window's own buttons. On desktop Cancel shows only with the cancel prop.
 */
const props = withDefaults(
  defineProps<{
    title: string
    /** wide: up to 1368, for tables such as the connection log; fit: as wide as what it holds, e.g. QR codes side by side. */
    size?: 'xs' | 'sm' | 'md' | 'form' | 'lg' | 'xl' | 'table' | 'wide' | 'fit'
    /** Close on Escape and on a click outside. Off while a form is saving. */
    dismissible?: boolean
    /** The body draws its own sections edge to edge: no padding, no gap. */
    flush?: boolean
    /** On phones fills the screen instead of rising as a sheet: logs, editors. */
    full?: boolean
    /** No title bar: the body carries the title, as XConfirm does. */
    headless?: boolean
    /** Cancel in the footer on desktop too, before the window's buttons. */
    cancel?: boolean
    /**
     * A long form that keeps to the screen: centred, its title and buttons in
     * view and only the body scrolling, e.g. the inbound window. As phones do.
     */
    scrollBody?: boolean
  }>(),
  { size: 'md', dismissible: true },
)

const open = defineModel<boolean>('open', { default: false })
const { t } = useI18n()
const id = useId()
const dialog = ref<HTMLElement | null>(null)
let returnFocus: HTMLElement | null = null

const FOCUSABLE = 'a[href], button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])'

function close() {
  if (props.dismissible) open.value = false
}

// Phones: the sheet comes down when pulled by its handle or title bar; the bar's buttons stay buttons.
function onPointerDown(e: PointerEvent) {
  const target = e.target as HTMLElement
  if (!props.dismissible || !dialog.value || !matchMedia('(max-width: 767px)').matches) return
  if (!target.closest('.x-modal__grip, .x-modal__head') || target.closest('button, a, input, textarea, select')) return
  startSheetDrag(e, dialog.value, close)
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.stopPropagation()
    close()
  } else if (e.key === 'Tab' && dialog.value) {
    const list = [...dialog.value.querySelectorAll<HTMLElement>(FOCUSABLE)]
    const first = list[0]
    const last = list[list.length - 1]
    if (!first || !last) return
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault()
      last.focus()
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault()
      first.focus()
    }
  }
}

watch(open, async (on, was) => {
  if (on !== !!was) lockScroll(on)
  if (on) {
    returnFocus = document.activeElement as HTMLElement | null
    await nextTick()
    const target = dialog.value?.querySelector<HTMLElement>('[autofocus]') ?? dialog.value
    // Without scrolling: the window is still below the screen's edge on its way in, and the page must not move under it.
    target?.focus({ preventScroll: true })
  } else {
    returnFocus?.focus({ preventScroll: true })
    returnFocus = null
  }
}, { immediate: true })

onBeforeUnmount(() => {
  if (open.value) lockScroll(false)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="x-modal">
      <div v-if="open" :class="['x-modal', { 'x-modal--full': full }]" @keydown="onKeydown" @click.self="close">
        <div
          ref="dialog"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="headless ? undefined : `${id}-title`"
          :aria-label="headless ? title : undefined"
          tabindex="-1"
          :class="['x-modal__dialog', `x-modal__dialog--${size}`, { 'x-modal__dialog--full': full, 'x-modal__dialog--headless': headless, 'x-modal__dialog--scroll': scrollBody }]"
          @pointerdown="onPointerDown"
        >
          <div class="x-modal__grip" aria-hidden="true" />
          <header v-if="!headless" :class="['x-modal__head', { 'x-modal__head--sub': $slots.subtitle }]">
            <div class="x-modal__heading">
              <div class="x-modal__title-row">
                <h2 :id="`${id}-title`" class="x-modal__title">{{ title }}</h2>
                <!-- Beside the title, e.g. a gear with the window's options. -->
                <slot name="title-end" />
              </div>
              <p v-if="$slots.subtitle" class="x-modal__subtitle"><slot name="subtitle" /></p>
            </div>
            <slot name="head" />
            <XButton icon="close" :label="t('kit.close')" :disabled="!dismissible" class="x-modal__close" @click="close" />
          </header>
          <div :class="['x-modal__body', { 'x-modal__body--flush': flush }]"><slot /></div>
          <footer v-if="$slots.footer || cancel || !full" :class="['x-modal__foot', { 'x-modal__foot--phone': !$slots.footer && !cancel }]">
            <XButton v-if="cancel || !full" :disabled="!dismissible" :class="{ 'x-modal__cancel--phone': !cancel }" @click="close">
              {{ t('kit.cancel') }}
            </XButton>
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.x-modal {
  position: fixed;
  inset: 0;
  z-index: 900;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 48px 16px;
  overflow-y: auto;
  /* Room for a scrollbar on both sides: a window taller than the screen does not move when it appears. */
  scrollbar-gutter: stable both-edges;
  background: var(--scrim);
  overscroll-behavior: contain;
}

.x-modal__dialog {
  display: flex;
  flex: none;
  flex-direction: column;
  width: 100%;
  /* In the middle of the screen; a window taller than it starts at the top and scrolls. */
  margin: auto 0;
  border: 1px solid var(--line);
  border-radius: var(--r-16);
  background: var(--card);
  box-shadow: var(--shadow-modal);
  outline: none;
  /* Bars and flush bodies with their own background follow the rounded corners; popups are teleported, so nothing is cut. */
  overflow: clip;
}

.x-modal__dialog--xs { max-width: 400px; }
.x-modal__dialog--sm { max-width: 440px; }
.x-modal__dialog--md { max-width: 560px; }
/* A record's fields in two columns, e.g. a client. */
.x-modal__dialog--form { max-width: 600px; }
.x-modal__dialog--lg { max-width: 720px; }
.x-modal__dialog--xl { max-width: 800px; }
/* A wide table to pick rows from, e.g. clients to copy. */
.x-modal__dialog--table { max-width: 900px; }
.x-modal__dialog--wide { max-width: 1368px; }

@media (min-width: 768px) {
  .x-modal__dialog--fit {
    width: auto;
    max-width: 100%;
  }
}

.x-modal__head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 12px 12px 16px;
  border-bottom: 1px solid var(--div);
}

.x-modal__head--sub {
  padding-block: 14px;
}

.x-modal__heading {
  flex: 1;
  min-width: 0;
}

.x-modal__title-row {
  display: flex;
  align-items: center;
  gap: var(--sp-6);
  min-width: 0;
}

.x-modal__title {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  color: var(--fg);
  font: 600 var(--fs-16) / var(--lh-tight) var(--font-sans);
  letter-spacing: -.3px;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.x-modal__subtitle {
  margin: 2px 0 0;
  overflow: hidden;
  color: var(--dim);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-sans);
  white-space: nowrap;
  text-overflow: ellipsis;
}

.x-modal__body {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 16px;
}

.x-modal__body--flush {
  gap: 0;
  padding: 0;
}

/* The window within the screen's padding; the body scrolls, with room kept for its scrollbar. */
.x-modal__dialog--scroll {
  max-height: calc(100dvh - 96px);
}

.x-modal__dialog--scroll .x-modal__body {
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
}

.x-modal__foot {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--div);
  background: var(--foot);
}

.x-modal__foot--phone,
.x-modal__cancel--phone {
  display: none;
}

/* The handle of a phone sheet; desktop windows have none. */
.x-modal__grip {
  display: none;
}

/* Desktop: the scrim fades in and the window rises into place. */
.x-modal-enter-active {
  transition: opacity var(--t-enter) var(--ease-out);
}

.x-modal-leave-active {
  transition: opacity var(--t-leave) ease-in;
}

.x-modal-enter-active .x-modal__dialog {
  transition: transform var(--t-enter) var(--ease-out);
}

.x-modal-leave-active .x-modal__dialog {
  transition: transform var(--t-leave) ease-in;
}

.x-modal-enter-from,
.x-modal-leave-to {
  opacity: 0;
}

.x-modal-enter-from .x-modal__dialog,
.x-modal-leave-to .x-modal__dialog {
  transform: translateY(16px) scale(.96);
}

@media (max-width: 767px) {
  .x-modal {
    align-items: flex-end;
    padding: 24px 0 0;
    overflow: hidden;
  }

  .x-modal__dialog {
    max-width: none;
    max-height: 100%;
    margin: 0;
    border-bottom: 0;
    border-radius: var(--r-16) var(--r-16) 0 0;
    box-shadow: var(--shadow-sheet);
  }

  .x-modal__body {
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  /* Sheet: a handle on top, the title under it with no line, no close button. */
  .x-modal__dialog:not(.x-modal__dialog--full) {
    padding-top: 10px;
  }

  /* Sheet: the handle, with room around it to take hold of. */
  .x-modal__dialog:not(.x-modal__dialog--full) .x-modal__grip {
    display: block;
    flex: none;
    align-self: stretch;
    height: 24px;
    /* Over the sheet's top padding: the bar sits where it did, the title 28px down. */
    margin: -10px 0 4px;
    touch-action: none;
  }

  .x-modal__dialog:not(.x-modal__dialog--full) .x-modal__grip::before {
    content: '';
    display: block;
    width: 38px;
    height: 4px;
    margin: 10px auto 0;
    border-radius: var(--r-pill);
    background: var(--track);
  }

  /* The title bar pulls the window down too. */
  .x-modal__head {
    touch-action: none;
  }

  .x-modal__dialog:not(.x-modal__dialog--full) .x-modal__head {
    padding: 0 16px;
    border-bottom: 0;
  }

  .x-modal__dialog:not(.x-modal__dialog--full) .x-modal__close {
    display: none;
  }

  .x-modal__dialog:not(.x-modal__dialog--full) .x-modal__body:not(.x-modal__body--flush) {
    padding: 14px 16px 16px;
  }

  /* Without a title bar the body sits right under the handle. */
  .x-modal__dialog--headless:not(.x-modal__dialog--full) .x-modal__body:not(.x-modal__body--flush) {
    padding-top: 0;
  }

  .x-modal__foot--phone,
  .x-modal__cancel--phone {
    display: flex;
  }

  .x-modal--full {
    padding: 0;
  }

  .x-modal__dialog--full {
    height: 100%;
    border: 0;
    border-radius: 0;
    box-shadow: none;
  }

  .x-modal__dialog--full .x-modal__head {
    padding-top: calc(12px + env(safe-area-inset-top));
  }

  .x-modal__dialog--full .x-modal__head--sub {
    padding-top: calc(14px + env(safe-area-inset-top));
  }

  .x-modal__dialog--full .x-modal__body {
    flex: 1;
    min-height: 0;
  }

  /* Buttons too long to share a line each take a line of their own. */
  .x-modal__foot {
    flex-wrap: wrap;
    padding-bottom: calc(12px + env(safe-area-inset-bottom));
  }

  .x-modal__foot > :deep(*) {
    flex: 1;
  }

  /*
   * Phones: every window slides up from the bottom while the scrim darkens, and
   * slides back down to close. The scrim is a layer of its own that only fades,
   * and both run on the compositor while they move, so nothing repaints the page
   * and nothing jolts.
   */
  .x-modal {
    background: none;
  }

  .x-modal::before {
    content: '';
    position: fixed;
    inset: 0;
    background: var(--scrim);
    backdrop-filter: var(--scrim-blur);
    transition: opacity var(--t-sheet) var(--ease-sheet);
  }

  .x-modal__dialog {
    position: relative;
  }

  .x-modal-enter-from,
  .x-modal-leave-to {
    opacity: 1;
  }

  .x-modal-enter-from::before,
  .x-modal-leave-to::before {
    opacity: 0;
  }

  /* The window itself does not change; this keeps it in the page while its sheet and scrim move. */
  .x-modal-enter-active,
  .x-modal-leave-active {
    transition: opacity var(--t-sheet);
  }

  .x-modal-enter-active::before,
  .x-modal-leave-active::before,
  .x-modal-enter-active .x-modal__dialog,
  .x-modal-leave-active .x-modal__dialog {
    will-change: opacity, transform;
  }

  .x-modal-enter-active .x-modal__dialog,
  .x-modal-leave-active .x-modal__dialog {
    transition: transform var(--t-sheet) var(--ease-sheet);
  }

  .x-modal-enter-from .x-modal__dialog,
  .x-modal-leave-to .x-modal__dialog {
    transform: translate3d(0, 100%, 0);
  }
}

/* Less motion: windows only fade, nothing moves. */
@media (prefers-reduced-motion: reduce) {
  .x-modal-enter-active,
  .x-modal-leave-active,
  .x-modal-enter-active .x-modal__dialog,
  .x-modal-leave-active .x-modal__dialog {
    transition: opacity var(--t-enter) ease !important;
  }

  .x-modal-enter-from,
  .x-modal-leave-to {
    opacity: 0;
  }

  .x-modal-enter-from .x-modal__dialog,
  .x-modal-leave-to .x-modal__dialog {
    transform: none;
  }
}
</style>
