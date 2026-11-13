<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../api'
import { formatNumber } from '../../i18n/format'
import { XButton, XMenu, XSpinner, type MenuItem } from '../../kit'

/** CPU load history; the server averages it into 60 buckets of the chosen size. */
const { t } = useI18n()

const ranges = [
  { bucket: 5, key: 'dashboard.range5m' },
  { bucket: 30, key: 'dashboard.range30m' },
  { bucket: 60, key: 'dashboard.range1h' },
  { bucket: 300, key: 'dashboard.range5h' },
] as const

const bucket = ref<number>(5)
const points = ref<{ t: number; cpu: number }[]>([])
const current = computed(() => ranges.find((r) => r.bucket === bucket.value) ?? ranges[0])
const items = computed<MenuItem[]>(() => ranges.map((r) => ({ key: String(r.bucket), label: t(r.key), checked: r.bucket === bucket.value })))

// While a range's first picture is on its way the plot shows a spinner; later refreshes keep the last picture.
const loading = ref(true)
let timer = 0
async function load() {
  const range = bucket.value
  try {
    const next = (await api.get<{ t: number; cpu: number }[]>(`panel/api/server/cpuHistory/${range}`)) ?? []
    // An answer for a range no longer shown is dropped.
    if (range === bucket.value) points.value = next
  } catch {
    // keep the last picture
  } finally {
    if (range === bucket.value) loading.value = false
  }
}

watch(
  bucket,
  () => {
    points.value = []
    loading.value = true
    load()
    clearInterval(timer)
    // A new point appears once per bucket, every 30 s at most.
    timer = window.setInterval(load, Math.max(2, Math.min(bucket.value, 30)) * 1000)
  },
  { immediate: true },
)
onBeforeUnmount(() => clearInterval(timer))

// The line takes 512 of the 560 units; the rest holds the scale labels.
const W = 512
const H = 48
const peak = computed(() => points.value.reduce((m, p) => Math.max(m, p.cpu), 0))
const top = computed(() => Math.min(100, Math.max(25, Math.ceil((peak.value * 1.15) / 25) * 25)))
const y = (v: number) => H - 2 - (v / top.value) * (H - 6)
const ticks = computed(() => Array.from({ length: top.value / 25 + 1 }, (_, i) => i * 25).map((v) => ({ v, y: y(v) })))
const line = computed(() => {
  const n = points.value.length
  if (n < 2) return ''
  return points.value.map((p, i) => `${i ? 'L' : 'M'}${((W / (n - 1)) * i).toFixed(1)} ${y(p.cpu).toFixed(1)}`).join(' ')
})
const area = computed(() => (line.value ? `${line.value} L${W} ${H} L0 ${H} Z` : ''))
</script>

<template>
  <div class="cpu">
    <div class="cpu__head">
      <span class="x-caps cpu__title">{{ t('dashboard.cpuLoad') }}</span>
      <XMenu v-slot="{ props }" :items="items" placement="bottom-start" @select="bucket = Number($event)">
        <XButton v-bind="props" size="sm" mono caret>{{ t('dashboard.range', { range: t(current.key) }) }}</XButton>
      </XMenu>
      <span v-if="points.length" class="cpu__peak">{{ t('dashboard.peak', { value: `${formatNumber(peak, 1)}%` }) }}</span>
    </div>
    <div class="cpu__plot">
      <svg viewBox="0 0 560 48" preserveAspectRatio="none" aria-hidden="true">
        <line v-for="tick in ticks" :key="tick.v" x1="0" :x2="W" :y1="tick.y" :y2="tick.y" class="cpu__grid" vector-effect="non-scaling-stroke" />
        <path v-if="area" :d="area" class="cpu__area" />
        <path v-if="line" :d="line" class="cpu__line" vector-effect="non-scaling-stroke" />
      </svg>
      <span v-for="tick in ticks" :key="tick.v" class="cpu__tick" :style="{ top: `${(tick.y / H) * 100}%` }">{{ tick.v }}%</span>
      <div v-if="loading" class="cpu__loading" role="status"><XSpinner :size="16" /></div>
    </div>
  </div>
</template>

<style scoped>
.cpu__head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.cpu__title {
  white-space: nowrap;
}

.cpu__peak {
  margin-left: auto;
  color: var(--mut);
  font: 500 var(--fs-10) / 1 var(--font-mono);
  white-space: nowrap;
}

.cpu__plot {
  position: relative;
  overflow: hidden;
  border: 1px solid var(--field-line);
  border-radius: var(--r-12);
  background: var(--field);
}

.cpu__loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--dim);
}

.cpu__plot > svg {
  display: block;
  width: 100%;
  height: 88px;
}

.cpu__grid {
  stroke: var(--div);
  stroke-width: 1;
}

.cpu__area {
  fill: var(--acc-ring);
}

.cpu__line {
  fill: none;
  stroke: var(--acc);
  stroke-width: 1.6;
  stroke-linejoin: round;
}

.cpu__tick {
  position: absolute;
  right: 8px;
  transform: translateY(-50%);
  color: var(--faint);
  font: 500 var(--fs-10) / 1 var(--font-mono);
  pointer-events: none;
}
</style>
