<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import { scrollbarCursor } from './scrollbarCursor'
import { useFieldAttrs } from './useFieldAttrs'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    placeholder?: string
    rows?: number
    mono?: boolean
    /** The system monospace, as code is: an IP log, a config. */
    code?: boolean
    disabled?: boolean
    readonly?: boolean
    invalid?: boolean
    modified?: boolean
    /** A long one-line value: starts as a field of control height, grows with the text up to this many lines, Enter adds nothing. */
    grow?: number
    /** The value is on its way: grey bars over the empty field, which keeps its size. */
    loading?: boolean
  }>(),
  { rows: 3 },
)

const model = defineModel<string>({ default: '' })
const { wrapper, control } = useFieldAttrs()

const field = ref<HTMLTextAreaElement | null>(null)

function fit() {
  const el = field.value
  if (!el || !props.grow) return
  el.style.height = 'auto'
  const cs = getComputedStyle(el)
  const most = parseFloat(cs.lineHeight) * props.grow + parseFloat(cs.paddingTop) + parseFloat(cs.paddingBottom)
  el.style.height = `${Math.min(el.scrollHeight, most)}px`
}

onMounted(fit)
watch(model, () => nextTick(fit))

function onKeydown(e: KeyboardEvent) {
  if (props.grow && e.key === 'Enter') e.preventDefault()
}
</script>

<template>
  <div
    v-bind="wrapper"
    :class="['x-field-box', 'x-textarea', { 'x-textarea--grow': grow, 'x-field-box--mono': mono, 'x-field-box--code': code, 'is-disabled': disabled, 'is-invalid': invalid, 'is-modified': modified }]"
    :aria-busy="loading || undefined"
  >
    <div v-if="loading" class="x-textarea__skeleton x-skeleton" aria-hidden="true">
      <span v-for="w in [62, 44, 54].slice(0, grow ? 1 : rows)" :key="w" class="x-textarea__line"><span class="x-skeleton__bar" :style="{ width: `${w}%` }" /></span>
    </div>
    <textarea
      ref="field"
      v-model="model"
      v-bind="control"
      class="x-textarea__control"
      :rows="grow ? 1 : rows"
      :placeholder="placeholder"
      :disabled="disabled"
      :readonly="readonly"
      :aria-invalid="invalid || undefined"
      :spellcheck="mono || code ? false : undefined"
      @keydown="onKeydown"
      @pointermove="scrollbarCursor"
      @pointerleave="scrollbarCursor"
    />
  </div>
</template>

<style scoped>
.x-textarea {
  position: relative;
  height: auto;
  min-height: 76px;
  padding: 8px 10px 8px var(--input-px);
  align-items: stretch;
}

.x-textarea__control {
  flex: 1;
  min-width: 0;
  min-height: 58px;
  padding: 0;
  border: 0;
  outline: none;
  background: transparent;
  font: inherit;
  line-height: var(--lh-text);
  color: inherit;
  resize: vertical;
}

/* A long one-line value: control height at one line, characters wrap, no resize. */
.x-textarea--grow {
  min-height: var(--ctl-h);
  padding-block: 0;
}

.x-textarea--grow .x-textarea__control {
  min-height: 0;
  padding-block: calc((var(--ctl-h) - 2px - 1lh) / 2);
  overflow-y: auto;
  resize: none;
  word-break: break-all;
}

/* One bar per text line, over the field while its value loads. */
.x-textarea__skeleton {
  position: absolute;
  inset: 8px 10px auto var(--input-px);
  line-height: var(--lh-text);
  pointer-events: none;
}

.x-textarea__line {
  display: flex;
  align-items: center;
  height: 1lh;
}

.x-textarea__control::placeholder {
  color: var(--faint);
}
</style>
