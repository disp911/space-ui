import { computed, nextTick, onMounted, ref, watch, type Ref } from 'vue'

/** A table column whose size may be 'fit': as wide as its widest cell, heading included. */
interface Sized {
  key: string
  size?: string
}

/**
 * The 'fit' columns of a table whose rows are grids of their own, so a
 * max-content track would differ from row to row: each 'fit' cell, marked with
 * data-col, is let out to its own width for a moment and read; all are set,
 * read and put back in one go, before the browser paints. Measured again when
 * the rows or the columns change or the pause ends; kept while `paused`, e.g.
 * while the table shows its phone columns.
 */
export function useFitColumns(root: Ref<HTMLElement | null>, columns: Ref<Sized[]>, rows: () => unknown, paused: Ref<boolean>) {
  const fit = ref<Record<string, number>>({})
  const fitTotal = computed(() => Object.values(fit.value).reduce((sum, w) => sum + w, 0))

  /** The grid track of a column. */
  const track = (c: Sized) =>
    c.size === 'fit' ? (fit.value[c.key] ? `${fit.value[c.key]}px` : 'max-content') : (c.size ?? 'minmax(0, 1fr)')

  /** Marks a 'fit' column's cells, for `data-col`. */
  const mark = (c: Sized) => (c.size === 'fit' ? c.key : undefined)

  function measure() {
    if (paused.value || !root.value || !columns.value.some((c) => c.size === 'fit')) return
    const next: Record<string, number> = {}
    for (const c of columns.value) {
      if (c.size !== 'fit') continue
      const cells = [...root.value.querySelectorAll<HTMLElement>(`[data-col="${c.key}"]`)]
      cells.forEach((el) => (el.style.width = 'max-content'))
      next[c.key] = Math.ceil(Math.max(0, ...cells.map((el) => el.getBoundingClientRect().width)))
      cells.forEach((el) => (el.style.width = ''))
    }
    fit.value = next
  }
  onMounted(() => nextTick(measure))
  watch([rows, columns, paused], () => nextTick(measure), { flush: 'post' })

  return { fitTotal, track, mark }
}
