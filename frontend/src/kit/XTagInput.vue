<script setup lang="ts">
import { computed, nextTick, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import XIcon from './XIcon.vue'
import { useFloating } from './floating'
import { usePhone } from './media'
import { useFieldAttrs } from './useFieldAttrs'

defineOptions({ inheritAttrs: false })

type Preset = string | { value: string; label: string }

/** A titled group of presets for the catalog. */
export interface PresetGroup {
  title?: string
  options: { value: string; label: string }[]
}

/**
 * Free values as chips, typed in and taken with Enter or a comma. Presets open
 * under the field on focus, filtered by the typed text, and a chip shows a
 * preset's label.
 *
 * As a list (presets): choosing a preset toggles it and a chosen one is
 * ticked; typed text that is no preset is offered at the bottom as «Add …».
 *
 * As a catalog (groups): titled groups in two columns on desktop, each preset
 * with a box to tick and, when it has a name of its own, its value beside the
 * name; typed text is offered at the top. Chosen named presets stand out as
 * accent chips among typed values, and a field with many values scrolls
 * instead of growing.
 */
const props = defineProps<{
  presets?: Preset[]
  /** A caps title over the presets. */
  presetsTitle?: string
  groups?: PresetGroup[]
  placeholder?: string
  /** The placeholder once something is chosen. */
  morePlaceholder?: string
  mono?: boolean
  disabled?: boolean
  invalid?: boolean
  modified?: boolean
}>()

const model = defineModel<string[]>({ default: () => [] })
const { t } = useI18n()
const { wrapper, control } = useFieldAttrs()
const phone = usePhone()

const id = useId()
const box = ref<HTMLElement | null>(null)
const input = ref<HTMLInputElement | null>(null)
const list = ref<HTMLElement | null>(null)
const text = ref('')
const focused = ref(false)
const active = ref(0)

const catalog = computed(() => !!props.groups)
const all = computed(() =>
  props.groups
    ? props.groups.flatMap((g) => g.options)
    : (props.presets ?? []).map((p) => (typeof p === 'string' ? { value: p, label: p } : p)),
)
const preset = (v: string) => all.value.find((p) => p.value === v)
const label = (v: string) => preset(v)?.label ?? v
/** A preset named otherwise than its value, e.g. Russia for geoip:ru; one named by its value reads as typed. */
const named = (p?: { value: string; label: string }) => !!p && p.label !== p.value

const query = computed(() => text.value.trim())
const matches = (p: { value: string; label: string }) => {
  const q = query.value.toLowerCase()
  return !q || p.label.toLowerCase().includes(q) || p.value.toLowerCase().includes(q)
}
const shown = computed(() => all.value.filter(matches))
const shownGroups = computed(() => (props.groups ?? []).map((g) => ({ ...g, options: g.options.filter(matches) })).filter((g) => g.options.length))

/** Typed text that is neither a preset nor chosen already. */
const custom = computed(() => {
  const q = query.value
  return q && !model.value.includes(q) && !all.value.some((p) => p.value === q || p.label === q) ? q : ''
})

// The rows in keyboard order: the catalog offers the typed value first, the list last.
const rows = computed(() => {
  const options = shown.value.map((p) => p.value)
  if (!custom.value) return options.map((value) => ({ value, custom: false }))
  const add = { value: custom.value, custom: true }
  return catalog.value ? [add, ...options.map((value) => ({ value, custom: false }))] : [...options.map((value) => ({ value, custom: false })), add]
})
const rowIndex = (value: string) => rows.value.findIndex((r) => !r.custom && r.value === value)
const addIndex = computed(() => rows.value.findIndex((r) => r.custom))

const open = computed(() => focused.value && !props.disabled && rows.value.length > 0)
const { style } = useFloating(box, list, open, { matchWidth: true, onDismiss: () => (focused.value = false) })
watch(query, () => (active.value = 0))

function add(value: string) {
  const v = value.trim()
  if (v && !model.value.includes(v)) model.value = [...model.value, v]
  text.value = ''
}

function toggle(value: string) {
  model.value = model.value.includes(value) ? model.value.filter((x) => x !== value) : [...model.value, value]
  text.value = ''
}

function remove(i: number) {
  model.value = model.value.filter((_, j) => j !== i)
  input.value?.focus()
}

function pick(i: number) {
  const row = rows.value[i]
  if (!row) return
  if (row.custom) add(row.value)
  else toggle(row.value)
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && open.value) {
    e.preventDefault()
    pick(active.value)
  } else if ((e.key === 'Enter' || e.key === ',') && query.value) {
    e.preventDefault()
    add(query.value)
  } else if (e.key === 'Backspace' && !text.value && model.value.length) {
    remove(model.value.length - 1)
  } else if ((e.key === 'ArrowDown' || e.key === 'ArrowUp') && rows.value.length) {
    e.preventDefault()
    if (!focused.value) focused.value = true
    active.value = (active.value + (e.key === 'ArrowDown' ? 1 : -1) + rows.value.length) % rows.value.length
    nextTick(() => list.value?.querySelector('.is-active')?.scrollIntoView({ block: 'nearest' }))
  } else if (e.key === 'Escape' || e.key === 'Tab') {
    focused.value = false
  }
}

function onBlur() {
  if (query.value) add(query.value)
}
</script>

<template>
  <div
    ref="box"
    v-bind="wrapper"
    :class="['x-field-box', 'x-tags', { 'x-tags--catalog': catalog, 'x-field-box--mono': mono, 'is-open': open, 'is-disabled': disabled, 'is-invalid': invalid, 'is-modified': modified }]"
    @click="input?.focus()"
  >
    <span v-for="(tag, i) in model" :key="tag" :class="['x-tag-chip', { 'x-tag-chip--preset': catalog && named(preset(tag)) }]">
      <span class="x-tag-chip__text">{{ label(tag) }}</span>
      <button v-if="!disabled" type="button" class="x-tag-chip__remove" :aria-label="t('kit.remove', { value: label(tag) })" @click.stop="remove(i)">
        <XIcon name="close" :size="12" :stroke="1.8" />
      </button>
    </span>
    <input
      ref="input"
      v-model="text"
      v-bind="control"
      class="x-tags__control"
      role="combobox"
      aria-autocomplete="list"
      :aria-expanded="open"
      :aria-controls="`${id}-list`"
      :aria-activedescendant="open ? `${id}-${active}` : undefined"
      :placeholder="model.length ? morePlaceholder : placeholder"
      :disabled="disabled"
      autocomplete="off"
      spellcheck="false"
      @focus="focused = true"
      @blur="onBlur"
      @keydown="onKeydown"
    />
  </div>
  <Teleport to="body">
    <Transition name="x-popup">
      <ul v-if="open && !catalog" :id="`${id}-list`" ref="list" role="listbox" aria-multiselectable="true" class="x-popup x-tags__list" :style="style">
        <li v-if="presetsTitle && shown.length" class="x-caps x-tags__title" role="presentation">{{ presetsTitle }}</li>
        <li
          v-for="p in shown"
          :id="`${id}-${rowIndex(p.value)}`"
          :key="p.value"
          role="option"
          :aria-selected="model.includes(p.value)"
          :class="['x-popup-item', { 'is-active': rowIndex(p.value) === active, 'is-selected': model.includes(p.value), 'x-tags__option--mono': mono && p.label === p.value }]"
          @pointermove="active = rowIndex(p.value)"
          @pointerdown.prevent
          @click="toggle(p.value)"
        >
          <span class="x-tags__label">{{ p.label }}</span>
          <XIcon v-if="model.includes(p.value)" name="check" :size="14" :stroke="1.8" />
        </li>
        <li
          v-if="custom"
          :id="`${id}-${addIndex}`"
          role="option"
          :aria-selected="false"
          :class="['x-popup-item', 'x-tags__add', { 'is-active': active === addIndex }]"
          @pointermove="active = addIndex"
          @pointerdown.prevent
          @click="add(custom)"
        >
          <XIcon name="plus" :size="12" :stroke="1.7" />
          <span class="x-tags__label">{{ t('kit.addValue', { value: custom }) }}</span>
          <span class="x-tags__key">Enter</span>
        </li>
      </ul>

      <div v-else-if="open" :id="`${id}-list`" ref="list" role="listbox" aria-multiselectable="true" class="x-popup x-tags__catalog" :style="style">
        <div
          v-if="custom"
          :id="`${id}-${addIndex}`"
          role="option"
          :aria-selected="false"
          :class="['x-tags__entry', 'x-tags__add', { 'is-active': active === addIndex }]"
          @pointermove="active = addIndex"
          @pointerdown.prevent
          @click="add(custom)"
        >
          <XIcon name="plus" :size="12" :stroke="1.7" />
          <span class="x-tags__label mono">{{ custom }}</span>
          <span class="x-tags__key">Enter</span>
        </div>
        <template v-for="g in shownGroups" :key="g.title ?? ''">
          <div v-if="g.title" class="x-tags__group" role="presentation">{{ g.title }}</div>
          <div :class="['x-tags__grid', { 'x-tags__grid--one': phone }]" role="group" :aria-label="g.title">
            <div
              v-for="p in g.options"
              :id="`${id}-${rowIndex(p.value)}`"
              :key="p.value"
              role="option"
              :aria-selected="model.includes(p.value)"
              :class="['x-tags__entry', { 'is-active': rowIndex(p.value) === active, 'is-selected': model.includes(p.value) }]"
              @pointermove="active = rowIndex(p.value)"
              @pointerdown.prevent
              @click="toggle(p.value)"
            >
              <span class="x-tags__tick"><XIcon v-if="model.includes(p.value)" name="check" :size="12" :stroke="2" /></span>
              <span class="x-tags__name">{{ p.label }}</span>
              <span v-if="named(p)" class="x-tags__value">{{ p.value }}</span>
            </div>
          </div>
        </template>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.x-tags {
  flex-wrap: wrap;
  gap: 4px;
  height: auto;
  min-height: var(--ctl-h);
  padding: 4px;
  cursor: text;
}

/* Many values scroll inside the field instead of pushing the form down. */
.x-tags--catalog {
  max-height: 124px;
  overflow-y: auto;
}

.x-tags__control {
  flex: 1;
  min-width: 110px;
  height: calc(var(--ctl-h) - 10px);
  padding: 0 6px;
  border: 0;
  outline: none;
  background: transparent;
  font: inherit;
  color: inherit;
}

.x-tags__control::placeholder {
  color: var(--faint);
}

/* A chosen preset among typed values. */
.x-tag-chip--preset {
  border-color: transparent;
  background: var(--acc-soft);
  color: var(--acc-t);
  font-family: var(--font-sans);
}

.x-tag-chip--preset .x-tag-chip__remove {
  color: inherit;
  opacity: .75;
}

.x-tags__list {
  max-height: 264px;
  margin: 0;
  list-style: none;
}

.x-tags__title {
  padding: 6px 10px 5px;
}

.x-tags__label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.x-tags__option--mono {
  font-family: var(--font-mono);
}

.x-tags__add {
  color: var(--acc-t);
}

.x-tags__key {
  flex: none;
  color: var(--dim);
  font: 400 var(--fs-10) / 1 var(--font-mono);
}

/* The catalog. */
.x-tags__catalog {
  max-height: 540px;
  padding: var(--sp-4);
}

.x-tags__group {
  padding: var(--sp-8) var(--sp-8) var(--sp-4);
  color: var(--dim);
  font: 600 var(--fs-10) / var(--lh-tight) var(--font-sans);
  letter-spacing: .06em;
  text-transform: uppercase;
}

.x-tags__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1px 6px;
}

.x-tags__grid--one {
  grid-template-columns: minmax(0, 1fr);
}

.x-tags__entry {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  height: 30px;
  padding: 0 8px;
  border-radius: var(--r-8);
  color: var(--fg);
  font: 400 var(--fs-12) / 1 var(--font-sans);
  cursor: pointer;
}

.x-tags__entry.is-active {
  background: var(--hov);
}

.x-tags__tick {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 15px;
  height: 15px;
  border: 1.5px solid var(--field-hover);
  border-radius: var(--r-4);
  color: var(--acc-ink);
}

.x-tags__entry.is-selected .x-tags__tick {
  border-color: var(--acc);
  background: var(--acc);
}

.x-tags__name {
  flex: none;
  white-space: nowrap;
}

.x-tags__value {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: var(--dim);
  font: 400 var(--fs-10) / 1 var(--font-mono);
  text-align: right;
  white-space: nowrap;
  text-overflow: ellipsis;
}
</style>
