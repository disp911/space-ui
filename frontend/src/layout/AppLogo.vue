<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { globeColors, globeFrame, restAngle } from './globe'

/**
 * The panel's mark, turning in the menu and on the login page alike unless
 * motion is reduced; large is the login page's size. It draws some 30 frames
 * a second, enough for its slow turn, and nothing while it is hidden.
 */
defineProps<{ large?: boolean }>()

const canvas = ref<HTMLCanvasElement>()
let frame = 0
let resize: ResizeObserver | undefined

function paint(el: HTMLCanvasElement, angle: number) {
  const size = el.clientWidth
  if (!size) return
  const dpr = window.devicePixelRatio || 1
  if (el.width !== Math.round(size * dpr)) {
    el.width = Math.round(size * dpr)
    el.height = Math.round(size * dpr)
  }
  const g = el.getContext('2d')
  if (!g) return
  g.setTransform(dpr, 0, 0, dpr, 0, 0)
  g.clearRect(0, 0, size, size)
  g.fillStyle = globeColors.tile
  g.beginPath()
  g.roundRect(0, 0, size, size, size * 0.28)
  g.fill()
  for (const d of globeFrame(size, angle).dots) {
    g.fillStyle = d.color
    g.beginPath()
    g.arc(d.x, d.y, d.r, 0, Math.PI * 2)
    g.fill()
  }
}

onMounted(() => {
  const el = canvas.value
  if (!el) return
  if (matchMedia('(prefers-reduced-motion: reduce)').matches) {
    // Still, drawn again whenever it changes size, such as a hidden menu shown.
    resize = new ResizeObserver(() => paint(el, restAngle))
    resize.observe(el)
    return
  }
  const start = performance.now()
  let last = -Infinity
  const tick = (now: number) => {
    if (now - last >= 32) {
      last = now
      paint(el, restAngle + ((now - start) / 1000) * 0.3)
    }
    frame = requestAnimationFrame(tick)
  }
  frame = requestAnimationFrame(tick)
})
onBeforeUnmount(() => {
  cancelAnimationFrame(frame)
  resize?.disconnect()
})
</script>

<template>
  <div :class="['logo', { 'logo--large': large }]">
    <canvas ref="canvas" class="logo__mark" aria-hidden="true" />
    <span class="logo__name">Space <span class="logo__panel">panel</span></span>
  </div>
</template>

<style scoped>
.logo {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

/* One size everywhere: the height of the kit's controls on desktop. */
.logo__mark {
  display: block;
  flex: none;
  width: 34px;
  height: 34px;
}

.logo__name {
  font: 600 var(--fs-16) / 1 var(--font-sans);
  letter-spacing: -.3px;
  white-space: nowrap;
}

.logo--large {
  gap: 14px;
}

.logo--large .logo__name {
  font-size: var(--fs-18);
}

/* Phones scale the type up (--fs-16 is 18px there), so the login page keeps 18px. */
@media (max-width: 767px) {
  .logo--large .logo__name {
    font-size: var(--fs-16);
  }
}

.logo__panel {
  color: var(--dim);
  font-weight: 400;
}
</style>
