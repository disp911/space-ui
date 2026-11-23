<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import XIcon from './XIcon.vue'
import XInlineButton from './XInlineButton.vue'
import { clockOf, momentOf } from '../i18n/zone'
import { useFieldAttrs } from './useFieldAttrs'

/**
 * A date and time, e.g. when an inbound ends: typed as YYYY-MM-DD HH:mm:ss or
 * picked in the browser's own calendar from the button. The value is
 * milliseconds since the epoch; the date and time are read and shown in the
 * panel's time zone. 0 is none, and the cross clears it to none. Text that is
 * not a date marks the field until it is one.
 */
defineOptions({ inheritAttrs: false })
const props = withDefaults(defineProps<{ placeholder?: string; disabled?: boolean; invalid?: boolean; modified?: boolean }>(), {
  placeholder: 'YYYY-MM-DD HH:mm:ss',
})
const model = defineModel<number>({ default: 0 })
const { t } = useI18n()
const { wrapper, control } = useFieldAttrs()

const p2 = (n: number) => String(n).padStart(2, '0')

function format(ms: number, sep = ' ') {
  if (!ms) return ''
  const c = clockOf(ms)
  return `${c.year}-${p2(c.month)}-${p2(c.day)}${sep}${p2(c.hour)}:${p2(c.minute)}:${p2(c.second)}`
}

/** Typed text as a date: the date alone, with hours and minutes, or with seconds. */
function parse(text: string): number | null {
  const m = /^(\d{4})-(\d{1,2})-(\d{1,2})(?:[ T](\d{1,2}):(\d{2})(?::(\d{2}))?)?$/.exec(text.trim())
  if (!m) return null
  // Hours, minutes and seconds left out are 0.
  const [y = 0, mo = 0, d = 0, h = 0, mi = 0, s = 0] = m.slice(1).map((v) => (v ? Number(v) : 0))
  if (h > 23 || mi > 59 || s > 59) return null
  // A day the month does not have, e.g. 31 April, rolls over: refused.
  const day = new Date(Date.UTC(y, mo - 1, d))
  if (day.getUTCMonth() !== mo - 1 || day.getUTCDate() !== d) return null
  return momentOf({ year: y, month: mo, day: d, hour: h, minute: mi, second: s })
}

const text = ref(format(model.value))
const wrong = ref(false)
watch(model, (ms) => {
  if (parse(text.value) !== ms) text.value = format(ms)
  wrong.value = false
})

function commit() {
  if (!text.value.trim()) {
    wrong.value = false
    model.value = 0
    return
  }
  const ms = parse(text.value)
  wrong.value = ms === null
  if (ms !== null) {
    model.value = ms
    text.value = format(ms)
  }
}

const picker = ref<HTMLInputElement | null>(null)

function openPicker() {
  const el = picker.value
  if (!el) return
  el.value = format(model.value, 'T') || format(Date.now(), 'T').slice(0, 16)
  try {
    el.showPicker()
  } catch {
    el.focus()
  }
}

function picked() {
  const ms = parse(picker.value?.value.replace('T', ' ') ?? '')
  if (ms !== null) model.value = ms
}

function clear() {
  model.value = 0
  text.value = ''
  wrong.value = false
}
</script>

<template>
  <div v-bind="wrapper" :class="['x-field-box', 'x-input', 'x-date', { 'is-disabled': disabled, 'is-invalid': invalid || wrong, 'is-modified': modified }]">
    <input
      v-bind="control"
      v-model="text"
      class="x-input__control x-date__control"
      :placeholder="placeholder"
      :disabled="disabled"
      :aria-invalid="invalid || wrong || undefined"
      inputmode="numeric"
      spellcheck="false"
      @change="commit"
      @keydown.enter.prevent="commit"
    />
    <XIcon v-if="invalid || wrong" name="warning" :size="14" class="x-field-alert" aria-hidden="true" />
    <XInlineButton v-if="model && !disabled" icon="close" :icon-size="12" :stroke="1.6" :label="t('kit.clear')" class="x-date__action" @click="clear" />
    <XInlineButton icon="calendar" :label="t('kit.pickDate')" :disabled="disabled" class="x-date__action" @click="openPicker" />
    <input ref="picker" type="datetime-local" step="1" class="x-date__native" tabindex="-1" aria-hidden="true" @change="picked" />
  </div>
</template>

<style scoped>
.x-date {
  position: relative;
  gap: 8px;
}

.x-date__control {
  flex: 1;
  min-width: 0;
  height: 100%;
  padding: 0;
  border: 0;
  outline: none;
  background: transparent;
  font: inherit;
  color: inherit;
}

.x-date__control::placeholder {
  color: var(--faint);
}

.x-date__action {
  margin-right: -6px;
}

.x-date__action + .x-date__action {
  margin-left: -6px;
}

/* The browser's calendar opens from here; the input itself stays out of sight. */
.x-date__native {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 1px;
  height: 1px;
  padding: 0;
  border: 0;
  opacity: 0;
  pointer-events: none;
}
</style>
