<script setup lang="ts" generic="T extends string">
import { nextTick, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import XIcon from './XIcon.vue'
import { useFieldAttrs } from './useFieldAttrs'
import { useFloating } from './floating'

defineOptions({ inheritAttrs: false })

/**
 * Several values from a list: the chosen ones are chips in the field, each
 * with a cross; the list ticks them and stays open while choosing. With
 * nothing chosen the placeholder shows, e.g. «all».
 */
const props = defineProps<{
  options: { value: T; label: string }[]
  placeholder?: string
  /** Shown in the list when there is nothing to choose from. */
  empty?: string
  mono?: boolean
  disabled?: boolean
  invalid?: boolean
  modified?: boolean
}>()

const model = defineModel<T[]>({ default: () => [] })
const { wrapper, control } = useFieldAttrs()
const { t } = useI18n()

const id = useId()
const box = ref<HTMLElement | null>(null)
const list = ref<HTMLElement | null>(null)
const open = ref(false)
const active = ref(0)
const { style } = useFloating(box, list, open, { matchWidth: true, onDismiss: () => (open.value = false) })

const label = (v: T) => props.options.find((o) => o.value === v)?.label ?? v

function toggle(v: T) {
  model.value = model.value.includes(v) ? model.value.filter((x) => x !== v) : [...model.value, v]
}

function show() {
  if (props.disabled) return
  open.value = !open.value
  active.value = 0
}

function onKeydown(e: KeyboardEvent) {
  const n = props.options.length
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    if (!open.value) open.value = true
    else if (n) active.value = (active.value + (e.key === 'ArrowDown' ? 1 : -1) + n) % n
    nextTick(() => list.value?.querySelector('.is-active')?.scrollIntoView({ block: 'nearest' }))
  } else if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    const o = props.options[active.value]
    if (open.value && o) toggle(o.value)
    else open.value = true
  } else if (e.key === 'Backspace' && model.value.length) {
    model.value = model.value.slice(0, -1)
  } else if (e.key === 'Tab') {
    open.value = false
  }
}
</script>

<template>
  <div
    ref="box"
    v-bind="{ ...wrapper, ...control }"
    role="combobox"
    aria-haspopup="listbox"
    :aria-expanded="open"
    :aria-controls="`${id}-list`"
    :aria-invalid="invalid || undefined"
    :tabindex="disabled ? -1 : 0"
    :class="['x-field-box', 'x-multi', { 'x-field-box--mono': mono, 'is-open': open, 'is-disabled': disabled, 'is-invalid': invalid, 'is-modified': modified }]"
    @click="show"
    @keydown="onKeydown"
  >
    <span v-for="v in model" :key="v" class="x-tag-chip">
      <span class="x-tag-chip__text">{{ label(v) }}</span>
      <button v-if="!disabled" type="button" class="x-tag-chip__remove" :aria-label="t('kit.remove', { value: label(v) })" @click.stop="toggle(v)">
        <XIcon name="close" :size="12" :stroke="1.8" />
      </button>
    </span>
    <span v-if="!model.length" class="x-multi__placeholder">{{ placeholder }}</span>
    <XIcon name="caret" :size="12" :stroke="1.7" :class="['x-multi__caret', { 'is-open': open }]" />
  </div>
  <Teleport to="body">
    <Transition name="x-popup">
      <ul v-if="open" :id="`${id}-list`" ref="list" role="listbox" aria-multiselectable="true" class="x-popup x-multi__list" :style="style">
        <li
          v-for="(o, i) in options"
          :key="o.value"
          role="option"
          :aria-selected="model.includes(o.value)"
          :class="['x-popup-item', { 'is-active': i === active, 'is-selected': model.includes(o.value), 'x-multi__option--mono': mono }]"
          @pointermove="active = i"
          @pointerdown.prevent
          @click="toggle(o.value)"
        >
          <XIcon name="check" :size="16" :stroke="1.8" :class="['x-popup-check', { 'is-on': model.includes(o.value) }]" />
          <span class="x-multi__label">{{ o.label }}</span>
        </li>
        <li v-if="!options.length" class="x-popup-empty">{{ empty }}</li>
      </ul>
    </Transition>
  </Teleport>
</template>

<style scoped>
.x-multi {
  flex-wrap: wrap;
  gap: 4px;
  height: auto;
  min-height: var(--ctl-h);
  padding: 4px 10px 4px 4px;
  cursor: pointer;
  outline: none;
}

.x-multi__placeholder {
  flex: 1;
  padding: 0 calc(var(--input-px) - 4px);
  color: var(--faint);
}

.x-multi__caret {
  flex: none;
  margin-left: auto;
  color: var(--dim);
  transition: transform var(--t-fast);
}

.x-multi__caret.is-open {
  transform: rotate(180deg);
}

.x-multi__list {
  max-height: 264px;
  margin: 0;
  list-style: none;
}

.x-multi__label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.x-multi__option--mono {
  font-family: var(--font-mono);
}
</style>
