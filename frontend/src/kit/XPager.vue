<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import XIcon from './XIcon.vue'
import XMenu, { type MenuItem } from './XMenu.vue'
import { useEdgeFade } from './edgeFade'
import { useSlidingThumb } from './slidingThumb'

/**
 * Pages of a list: one line of every page number with the current one lit,
 * the whole width on phones. A tap or click picks a page; a finger or the
 * mouse drawn along the line picks the one it is let go on; a line too long
 * for its place scrolls sideways with the current page in its middle, its
 * numbers fading out at an edge that has more beyond it. With sizes, a choice
 * of rows per page follows it.
 */
const props = defineProps<{ pages: number; sizes?: number[] }>()
const page = defineModel<number>({ default: 1 })
const size = defineModel<number>('size', { default: 0 })
const { t } = useI18n()

const sizeItems = computed<MenuItem[]>(() => (props.sizes ?? []).map((n) => ({ key: String(n), label: t('kit.perPage', { n }), checked: n === size.value })))

function go(to: number) {
  page.value = Math.min(Math.max(to, 1), props.pages)
}

const line = ref<HTMLElement | null>(null)
/** The page lit: the current one, or the one under a moving finger. */
const lit = ref(page.value)
watch(page, (p) => (lit.value = p))
// The lit page's accent travels between the numbers like liquid.
const thumb = useSlidingThumb(line, '.x-pager__cell.is-current', [lit, () => props.pages])

// A line that fits is drawn along to pick; one too long scrolls instead.
const fits = ref(true)
let sizer: ResizeObserver | null = null
const { classes: fade, update: updateFade } = useEdgeFade(line)
const measure = () => {
  fits.value = !line.value || line.value.scrollWidth <= line.value.clientWidth + 1
  updateFade()
}
onMounted(() => {
  if (!line.value) return
  // Measured once laid out, and again on a resize: the current page comes back to the middle.
  sizer = new ResizeObserver(() => (measure(), center(false)))
  sizer.observe(line.value)
})
onBeforeUnmount(() => sizer?.disconnect())

function center(smooth = true) {
  const box = line.value
  const el = box?.querySelector<HTMLElement>('.x-pager__cell.is-current')
  if (!box || !el || fits.value) return
  box.scrollTo({ left: el.offsetLeft - (box.clientWidth - el.offsetWidth) / 2, behavior: smooth ? 'smooth' : 'auto' })
}
watch([page, () => props.pages], () => nextTick(() => (measure(), center())))

let drawing = false
let moved = false
let startX = 0
function drawStart(e: PointerEvent) {
  drawing = true
  moved = false
  startX = e.clientX
}
function drawMove(e: PointerEvent) {
  if (!drawing || !fits.value) return
  if (Math.abs(e.clientX - startX) > 6) moved = true
  if (!moved) return
  const cell = document.elementFromPoint(e.clientX, e.clientY)?.closest<HTMLElement>('.x-pager__cell')
  if (cell?.dataset.page) lit.value = Number(cell.dataset.page)
}
function drawEnd() {
  if (!drawing) return
  drawing = false
  if (moved) go(lit.value)
}
</script>

<template>
  <nav class="x-pager" :aria-label="t('kit.pages')">
    <div class="x-pager__line">
      <div
        ref="line"
        :class="['x-pager__track', fade, { 'is-fit': fits }]"
        @pointerdown="drawStart"
        @pointermove="drawMove"
        @pointerup="drawEnd"
        @pointercancel="drawing = false"
      >
        <span
          v-if="thumb"
          :class="['x-pager__thumb', thumb.ahead && `x-slide--${thumb.ahead}`]"
          :style="{ left: `${thumb.left}px`, right: `${thumb.right}px` }"
          aria-hidden="true"
        />
        <button
          v-for="n in pages"
          :key="n"
          type="button"
          :data-page="n"
          :class="['x-pager__cell', { 'is-current': n === lit }]"
          :aria-current="n === page ? 'page' : undefined"
          @click="go(n)"
        >
          {{ n }}
        </button>
      </div>
    </div>
    <XMenu v-if="sizes?.length" v-slot="{ props: trigger }" :items="sizeItems" placement="bottom-end" @select="(size = Number($event)), (page = 1)">
      <button type="button" class="x-pager__size" v-bind="trigger">
        {{ t('kit.perPage', { n: size }) }}
        <XIcon name="caret" :size="12" :stroke="1.7" />
      </button>
    </XMenu>
  </nav>
</template>

<style scoped>
.x-pager {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

/* The frame; the numbers scroll inside it, so the fade never dims the frame. */
.x-pager__line {
  display: flex;
  flex: 0 1 auto;
  min-width: 0;
  height: var(--ctl-h);
  padding: 2px;
  border: 1px solid var(--field-line);
  border-radius: var(--btn-r);
  background: var(--field);
}

.x-pager__track {
  position: relative;
  display: flex;
  flex: 1;
  gap: 2px;
  min-width: 0;
  overflow-x: auto;
  border-radius: var(--btn-r);
  scrollbar-width: none;
  touch-action: pan-x;
  user-select: none;
}

.x-pager__track::-webkit-scrollbar {
  display: none;
}

.x-pager__track.is-fit {
  touch-action: none;
}

/* Under the numbers, which are lifted above it. */
.x-pager__thumb {
  position: absolute;
  top: 0;
  bottom: 0;
  z-index: 1;
  border-radius: var(--btn-r);
  background: var(--acc);
  pointer-events: none;
}

/* A number is at least a square of the line's inner height. */
.x-pager__cell {
  position: relative;
  z-index: 2;
  flex: 1 0 auto;
  min-width: calc(var(--ctl-h) - 6px);
  padding: 0 8px;
  border: 0;
  border-radius: var(--btn-r);
  background: transparent;
  color: var(--fg2);
  font: 500 var(--fs-13) / 1 var(--font-mono);
  outline: none;
  transition: color var(--t-med);
}

.x-pager__cell:hover:not(.is-current) {
  color: var(--fg);
}

.x-pager__cell:focus-visible {
  box-shadow: inset 0 0 0 2px var(--acc-line);
}

.x-pager__cell.is-current {
  color: var(--acc-ink);
  font-weight: 600;
}

.x-pager__size {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 6px;
  height: var(--ctl-h);
  padding: 0 10px 0 12px;
  border: 1px solid var(--field-line);
  border-radius: var(--btn-r);
  background: var(--ctrl);
  color: var(--fg2);
  font: 500 var(--fs-12) / 1 var(--font-mono);
  white-space: nowrap;
  outline: none;
  transition: border-color var(--t-fast), background-color var(--t-fast), box-shadow var(--t-fast), transform var(--t-fast);
}

.x-pager__size:hover {
  border-color: var(--field-hover);
}

.x-pager__size:focus-visible {
  border-color: var(--acc-line);
  box-shadow: 0 0 0 3px var(--acc-ring);
}

.x-pager__size:active {
  background: var(--press);
  transform: scale(var(--press-scale));
}

.x-pager__size :deep(.x-icon) {
  color: var(--dim);
}

/* On phones the line takes what the row leaves, its numbers sharing it. */
@media (max-width: 767px) {
  .x-pager {
    width: 100%;
  }

  .x-pager__line {
    flex: 1;
  }
}
</style>
