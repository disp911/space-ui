import { onBeforeUnmount, watch, type Ref } from 'vue'

/** How often an open log window reads its lines again. */
export const logRefreshEvery = 5000

/**
 * Calls refresh every logRefreshEvery ms while open is on and the page is in
 * view, never while a call is still running, and not while wait says so
 * (e.g. the reader scrolled down from the newest lines).
 */
export function useAutoRefresh(open: Ref<boolean>, refresh: () => Promise<unknown>, wait: () => boolean = () => false) {
  let timer: ReturnType<typeof setInterval> | undefined
  let busy = false
  async function tick() {
    if (!open.value || document.hidden || busy || wait()) return
    busy = true
    try {
      await refresh()
    } finally {
      busy = false
    }
  }
  watch(
    open,
    (on) => {
      clearInterval(timer)
      if (on) timer = setInterval(tick, logRefreshEvery)
    },
    { immediate: true },
  )
  onBeforeUnmount(() => clearInterval(timer))
}

/** The box shows its top, the newest lines: it is not scrolled down, or not there. */
export const atTop = (el: Element | null | undefined) => !el || el.scrollTop < 8
