<script setup lang="ts" generic="T extends OptionValue">
import { computed, nextTick, ref, useId } from 'vue'
import XIcon from './XIcon.vue'
import { useFloating } from './floating'
import type { Option, OptionValue } from './types'
import { useFieldAttrs } from './useFieldAttrs'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  options: Option<T>[]
  placeholder?: string
  mono?: boolean
  disabled?: boolean
  invalid?: boolean
  modified?: boolean
  /** Single symbols: the list is a grid of this many columns at the right edge of the field. */
  columns?: number
  /** A long list, e.g. countries: the field takes typed text and the list keeps what matches, the match marked. */
  search?: boolean
}>()

const model = defineModel<T | null>({ default: null })
const { wrapper, control } = useFieldAttrs()

const id = useId()
const trigger = ref<HTMLElement | null>(null)
const input = ref<HTMLInputElement | null>(null)
const list = ref<HTMLElement | null>(null)
const open = ref(false)
const active = ref(-1)
const { style } = useFloating(trigger, list, open, props.columns ? { placement: 'bottom-end', onDismiss: close } : { matchWidth: true, onDismiss: close })

const selected = computed(() => props.options.find((o) => o.value === model.value))

// What the list shows: every option, or with search the ones whose label holds the typed text.
const query = ref('')
const shown = computed(() => {
  const q = query.value.trim().toLowerCase()
  return q ? props.options.filter((o) => o.label.toLowerCase().includes(q)) : props.options
})

/** A label cut around the typed text, so the match can be marked. */
function parts(label: string) {
  const q = query.value.trim()
  const at = q ? label.toLowerCase().indexOf(q.toLowerCase()) : -1
  return at < 0 ? null : [label.slice(0, at), label.slice(at, at + q.length), label.slice(at + q.length)]
}

function show() {
  if (props.disabled || open.value) return
  open.value = true
  active.value = Math.max(0, shown.value.findIndex((o) => o.value === model.value))
  scrollToActive()
}

function close() {
  open.value = false
  query.value = ''
}

function choose(i: number) {
  const o = shown.value[i]
  if (!o || o.disabled) return
  model.value = o.value
  close()
  ;(input.value ?? trigger.value)?.focus()
}

function move(by: number) {
  const n = shown.value.length
  for (let i = 1; i <= n; i++) {
    const next = (active.value + by * i + n * n) % n
    if (!shown.value[next]?.disabled) {
      active.value = next
      break
    }
  }
  scrollToActive()
}

function onInput(e: Event) {
  query.value = (e.target as HTMLInputElement).value
  if (!open.value) show()
  active.value = 0
}

function scrollToActive() {
  nextTick(() => list.value?.querySelector('.is-active')?.scrollIntoView({ block: 'nearest' }))
}

let typed = ''
let typedAt = 0
function typeahead(key: string) {
  const now = Date.now()
  typed = now - typedAt > 600 ? key : typed + key
  typedAt = now
  const i = shown.value.findIndex((o) => !o.disabled && o.label.toLowerCase().startsWith(typed.toLowerCase()))
  if (i >= 0) {
    active.value = i
    scrollToActive()
  }
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    if (!open.value) show()
    else move(e.key === 'ArrowDown' ? 1 : -1)
  } else if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    if (open.value) choose(active.value)
    else show()
  } else if (e.key === 'Escape' && open.value) {
    e.preventDefault()
    e.stopPropagation()
    close()
  } else if (e.key === 'Home' || e.key === 'End') {
    if (!open.value || props.search) return
    e.preventDefault()
    active.value = e.key === 'Home' ? -1 : shown.value.length
    move(e.key === 'Home' ? 1 : -1)
  } else if (e.key === 'Tab') {
    close()
  } else if (!props.search && e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey) {
    if (!open.value) show()
    typeahead(e.key)
  }
}
</script>

<template>
  <!-- With search the field is a text input in the box; Space types instead of opening. -->
  <div
    v-if="search"
    ref="trigger"
    v-bind="wrapper"
    :class="['x-field-box', 'x-select', 'x-select--search', { 'x-field-box--mono': mono, 'is-open': open, 'is-disabled': disabled, 'is-invalid': invalid, 'is-modified': modified }]"
    @click="input?.focus(); show()"
  >
    <input
      ref="input"
      v-bind="control"
      class="x-select__input"
      role="combobox"
      aria-autocomplete="list"
      :aria-expanded="open"
      :aria-controls="`${id}-list`"
      :aria-activedescendant="open && active >= 0 ? `${id}-${active}` : undefined"
      :aria-invalid="invalid || undefined"
      :disabled="disabled"
      :value="open ? query : (selected?.label ?? '')"
      :placeholder="open ? (selected?.label ?? placeholder) : placeholder"
      autocomplete="off"
      spellcheck="false"
      @input="onInput"
      @keydown="onKeydown"
    />
    <XIcon name="caret" :size="12" :stroke="1.7" :class="['x-select__caret', { 'is-open': open }]" />
  </div>
  <button
    v-else
    ref="trigger"
    v-bind="{ ...wrapper, ...control }"
    type="button"
    role="combobox"
    aria-haspopup="listbox"
    :aria-expanded="open"
    :aria-controls="`${id}-list`"
    :aria-activedescendant="open && active >= 0 ? `${id}-${active}` : undefined"
    :aria-invalid="invalid || undefined"
    :disabled="disabled"
    :class="['x-field-box', 'x-select', { 'x-field-box--mono': mono, 'is-open': open, 'is-disabled': disabled, 'is-invalid': invalid, 'is-modified': modified }]"
    @click="open ? close() : show()"
    @keydown="onKeydown"
  >
    <span :class="['x-select__value', { 'is-placeholder': !selected }]">{{ selected?.label ?? placeholder }}</span>
    <XIcon name="caret" :size="12" :stroke="1.7" :class="['x-select__caret', { 'is-open': open }]" />
  </button>
  <Teleport to="body">
    <Transition name="x-popup">
      <ul
        v-if="open"
        :id="`${id}-list`"
        ref="list"
        role="listbox"
        :class="['x-popup', 'x-select__list', { 'x-select__list--grid': columns }]"
        :style="[style, columns ? { gridTemplateColumns: `repeat(${columns}, 34px)` } : {}]"
      >
        <li
          v-for="(o, i) in shown"
          :id="`${id}-${i}`"
          :key="String(o.value)"
          role="option"
          :aria-selected="o.value === model"
          :aria-disabled="o.disabled || undefined"
          :class="['x-popup-item', { 'is-active': i === active, 'is-selected': o.value === model, 'x-select__option--mono': mono && !o.plain, 'x-select__option--plain': o.plain, 'x-select__option--noted': o.note }]"
          @pointermove="!o.disabled && (active = i)"
          :title="columns ? o.label : undefined"
          @click="choose(i)"
        >
          <span v-if="columns" class="x-select__glyph">{{ o.glyph ?? o.label }}</span>
          <template v-else>
            <span v-if="parts(o.label)" class="x-select__label">{{ parts(o.label)![0] }}<mark class="x-select__match">{{ parts(o.label)![1] }}</mark>{{ parts(o.label)![2] }}</span>
            <span v-else class="x-select__label">{{ o.label }}</span>
            <XIcon v-if="o.value === model" name="check" :size="14" :stroke="1.8" />
            <XIcon v-else-if="o.disabled && o.note" name="lock" :size="12" :stroke="1.5" />
            <span v-if="o.note" class="x-select__note">{{ o.note }}</span>
          </template>
        </li>
      </ul>
    </Transition>
  </Teleport>
</template>

<style scoped>
.x-select {
  gap: 8px;
  text-align: left;
  cursor: pointer;
  outline: none;
}

.x-select__value {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.x-select__value.is-placeholder {
  color: var(--faint);
}

.x-select__caret {
  color: var(--dim);
  transition: transform var(--t-fast);
}

.x-select__caret.is-open {
  transform: rotate(180deg);
}

.x-select__list {
  margin: 0;
  list-style: none;
}

.x-select--search {
  cursor: text;
}

.x-select__input {
  flex: 1;
  min-width: 0;
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  outline: none;
}

.x-select__input::placeholder {
  color: var(--faint);
}

/* While typing, the chosen value shows as the hint it was. */
.x-select--search.is-open .x-select__input::placeholder {
  color: var(--dim);
}

.x-select__match {
  background: none;
  color: var(--acc-t);
  font-weight: 600;
}

.x-select__label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

/* An option with a line under it: the label and its mark on top, the note below. */
.x-select__option--noted {
  flex: none;
  flex-wrap: wrap;
  row-gap: 2px;
  padding-top: 6px;
  padding-bottom: 6px;
}

.x-select__note {
  flex: 1 0 100%;
  color: var(--mut);
  font: 400 var(--fs-12) / 1.35 var(--font-sans);
  text-wrap: pretty;
}

/* A grid of symbols: square cells, no lines between them. */
.x-select__list--grid {
  display: grid;
  gap: 4px;
  min-width: 0;
  padding: 6px;
}

.x-select__list--grid .x-popup-item {
  justify-content: center;
  min-height: 32px;
  padding: 0;
  border-top: 0;
}

.x-select__list--grid .x-popup-item::before {
  display: none;
}

.x-select__glyph {
  font: 500 var(--fs-14) / 1 var(--font-mono);
}

.x-select__option--mono {
  font-family: var(--font-mono);
  font-weight: 400;
}

.x-select__option--plain {
  font-style: italic;
}
</style>
