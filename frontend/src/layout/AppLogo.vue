<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { globeColors, globeFrame, globeSvg, restAngle } from './globe'

/** large is the login page's mark: it turns, unless motion is reduced. */
const props = defineProps<{ large?: boolean }>()

const mark = globeSvg()
const canvas = ref<HTMLCanvasElement>()
let frame = 0

function paint(el: HTMLCanvasElement, angle: number) {
  const size = el.clientWidth
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
  if (!props.large || !el) return
  if (matchMedia('(prefers-reduced-motion: reduce)').matches) {
    paint(el, restAngle)
    return
  }
  const start = performance.now()
  const tick = (now: number) => {
    const t = (now - start) / 1000
    paint(el, restAngle + t * 0.3)
    frame = requestAnimationFrame(tick)
  }
  frame = requestAnimationFrame(tick)
})
onBeforeUnmount(() => cancelAnimationFrame(frame))
</script>

<template>
  <div :class="['logo', { 'logo--large': large }]">
    <canvas v-if="large" ref="canvas" class="logo__mark" aria-hidden="true" />
    <span v-else class="logo__mark" aria-hidden="true" v-html="mark" />
    <span class="logo__name">Space <span class="logo__panel">panel</span></span>
  </div>
</template>

<style scoped>
.logo {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.logo__mark {
  display: block;
  flex: none;
  width: 26px;
  height: 26px;
}

.logo__mark :deep(svg) {
  display: block;
  width: 100%;
  height: 100%;
}

.logo--large {
  gap: 12px;
}

.logo--large .logo__mark {
  width: 32px;
  height: 32px;
}

.logo--large .logo__name {
  font-size: var(--fs-16);
}

@media (max-width: 767px) {
  .logo--large .logo__mark {
    width: 28px;
    height: 28px;
  }

  .logo--large .logo__name {
    font-size: var(--fs-14);
  }
}

.logo__name {
  font: 600 var(--fs-14) / 1 var(--font-sans);
  letter-spacing: -.3px;
  white-space: nowrap;
}

.logo__panel {
  color: var(--dim);
  font-weight: 400;
}
</style>
