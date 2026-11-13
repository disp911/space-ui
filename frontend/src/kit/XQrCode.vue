<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { encode } from 'uqr'
import XIcon from './XIcon.vue'
import { copyText } from './clipboard'
import { toast } from './toast'

/**
 * A QR code on a white plate, black on white in both themes so cameras read
 * it. With copy, a click puts that text on the clipboard: the corner badge
 * turns from a copy sign into a check for a moment. quiet drops the badge and
 * says so in a toast instead, e.g. a row of QR codes to share.
 */
const props = withDefaults(defineProps<{ value: string; size?: number; copy?: string; copyLabel?: string; quiet?: boolean }>(), { size: 180 })
const emit = defineEmits<{ copied: [ok: boolean] }>()
const { t } = useI18n()

const qr = computed(() => encode(props.value, { ecc: 'L', border: 2 }))
const path = computed(() => {
  let d = ''
  qr.value.data.forEach((row, y) => row.forEach((on, x) => on && (d += `M${x} ${y}h1v1h-1z`)))
  return d
})

const done = ref(false)
let timer = 0
async function onCopy() {
  if (props.copy == null) return
  const ok = await copyText(props.copy)
  emit('copied', ok)
  if (props.quiet) return toast(ok ? 'ok' : 'err', t(ok ? 'kit.copied' : 'kit.copyFailed'))
  if (!ok) return
  done.value = false
  requestAnimationFrame(() => (done.value = true))
  clearTimeout(timer)
  timer = window.setTimeout(() => (done.value = false), 1800)
}
onBeforeUnmount(() => clearTimeout(timer))
</script>

<template>
  <component
    :is="copy != null ? 'button' : 'div'"
    :type="copy != null ? 'button' : undefined"
    :class="['x-qr', { 'x-qr--copy': copy != null }]"
    :style="{ width: `${size}px`, height: `${size}px` }"
    :title="copyLabel"
    :aria-label="copyLabel"
    @click="onCopy"
  >
    <svg :viewBox="`0 0 ${qr.size} ${qr.size}`" shape-rendering="crispEdges" aria-hidden="true">
      <path :d="path" fill="#000" />
    </svg>
    <span v-if="copy != null && !quiet" :class="['x-qr__badge', { 'is-done': done }]" aria-hidden="true">
      <XIcon :name="done ? 'check' : 'copy'" :size="14" :stroke="done ? 1.9 : 1.5" />
    </span>
  </component>
</template>

<style scoped>
.x-qr {
  position: relative;
  display: block;
  flex: none;
  padding: 0;
  border: 0;
  border-radius: var(--nest-r);
  background: #fff;
  box-shadow: 0 0 0 1px var(--line);
}

.x-qr svg {
  display: block;
  width: 100%;
  height: 100%;
}

.x-qr--copy {
  cursor: pointer;
  outline: none;
  transition: box-shadow var(--t-med);
}

.x-qr--copy:hover,
.x-qr--copy:focus-visible {
  box-shadow: 0 0 0 2px var(--acc), 0 0 0 6px var(--acc-ring);
}

.x-qr__badge {
  position: absolute;
  right: -9px;
  bottom: -9px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: var(--btn-r);
  background: var(--acc);
  color: var(--acc-ink);
  box-shadow: 0 0 0 3px var(--dlg);
}

.x-qr__badge.is-done {
  animation: x-qr-pop .42s cubic-bezier(.22, 1, .36, 1);
}

@keyframes x-qr-pop {
  40% {
    transform: scale(1.12);
  }
}
</style>
