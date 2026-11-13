<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { EditorState, RangeSet, StateField, type Extension, type Range } from '@codemirror/state'
import {
  Decoration, drawSelection, EditorView, gutter, gutterLineClass, GutterMarker, highlightSpecialChars, keymap, lineNumbers, type DecorationSet, type ViewUpdate,
} from '@codemirror/view'
import {
  bracketMatching, codeFolding, foldable, foldedRanges, foldEffect, foldKeymap, HighlightStyle, indentOnInput, indentUnit, syntaxHighlighting, unfoldEffect,
} from '@codemirror/language'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import { json } from '@codemirror/lang-json'
import { tags } from '@lezer/highlight'
import XIcon from './XIcon.vue'
import { checkJson, type JsonIssue } from './jsonCheck'

/**
 * JSON in an editor: colours for keys, strings, numbers and literals, line
 * numbers, blocks that fold from the chevron beside their first line (a
 * folded block shows how many lines it hides) and the first mistake marked on
 * its line and told under the text. The model is the text as typed, wrong or
 * not; the page decides what to do with it. Its height in px, phoneHeight on
 * phones; with fit the editor is as tall as its text, up to that height.
 */
const props = withDefaults(defineProps<{ height?: number; phoneHeight?: number; label?: string; fit?: boolean }>(), { height: 560, phoneHeight: 480 })
const text = defineModel<string>({ default: '' })
const { t } = useI18n()

const host = ref<HTMLElement | null>(null)
const view = shallowRef<EditorView | null>(null)
const issue = ref<JsonIssue | null>(null)

// --- The first mistake --------------------------------------------------------

const errorLine = Decoration.line({ class: 'cm-x-error-line' })
const errorText = Decoration.mark({ class: 'cm-x-error-text' })
const errorGutter = new (class extends GutterMarker {
  elementClass = 'cm-x-error-gutter'
})()

interface Checked {
  issue: JsonIssue | null
  marks: DecorationSet
}

function check(state: EditorState): Checked {
  const found = checkJson(state.doc.toString())
  if (!found) return { issue: null, marks: Decoration.none }
  const line = state.doc.line(Math.min(found.line, state.doc.lines))
  const ranges: Range<Decoration>[] = [errorLine.range(line.from)]
  // The text of the line, without its indent.
  const start = line.from + (line.text.length - line.text.trimStart().length)
  if (start < line.to) ranges.push(errorText.range(start, line.to))
  return { issue: found, marks: Decoration.set(ranges, true) }
}

const checked = StateField.define<Checked>({
  create: check,
  update: (value, tr) => (tr.docChanged ? check(tr.state) : value),
  provide: (f) => [
    EditorView.decorations.from(f, (v) => v.marks),
    gutterLineClass.from(f, (v) => (v.issue ? RangeSet.of([errorGutter.range(v.marks.iter().from)]) : RangeSet.empty)),
  ],
})

// --- Folding: a chevron per foldable line, or the mistake's dot -------------------

class Mark extends GutterMarker {
  constructor(readonly kind: 'open' | 'folded' | 'error') {
    super()
  }

  eq(other: Mark) {
    return other.kind === this.kind
  }

  toDOM() {
    const el = document.createElement('span')
    el.className = `cm-x-mark cm-x-mark--${this.kind}`
    if (this.kind !== 'error') {
      el.innerHTML = '<svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 3.5 10.5 8 6 12.5"/></svg>'
    }
    return el
  }
}
const marks = { open: new Mark('open'), folded: new Mark('folded'), error: new Mark('error') }

/** The folded range that starts on a line. */
function foldedOn(state: EditorState, from: number, to: number) {
  let found: { from: number; to: number } | null = null
  foldedRanges(state).between(from, to, (a, b) => {
    if (!found || found.from > a) found = { from: a, to: b }
  })
  return found as { from: number; to: number } | null
}

const markGutter = gutter({
  class: 'cm-x-marks',
  lineMarker(v, line) {
    const found = v.state.field(checked).issue
    if (found && v.state.doc.lineAt(line.from).number === found.line) return marks.error
    if (foldedOn(v.state, line.from, line.to)) return marks.folded
    return foldable(v.state, line.from, line.to) ? marks.open : null
  },
  lineMarkerChange: (u) =>
    u.docChanged || u.viewportChanged || u.transactions.some((tr) => tr.effects.some((e) => e.is(foldEffect) || e.is(unfoldEffect))),
  initialSpacer: () => marks.open,
  domEventHandlers: {
    click(v, line) {
      const folded = foldedOn(v.state, line.from, line.to)
      if (folded) {
        v.dispatch({ effects: unfoldEffect.of(folded) })
        return true
      }
      const range = foldable(v.state, line.from, line.to)
      if (range) v.dispatch({ effects: foldEffect.of(range) })
      return !!range
    },
  },
})

function placeholder(_view: EditorView, onclick: (e: Event) => void, hidden: number) {
  const el = document.createElement('button')
  el.type = 'button'
  el.className = 'cm-x-folded'
  el.textContent = `⋯ ${hidden}`
  el.title = t('kit.unfold')
  el.onclick = onclick
  return el
}

// --- Colours ----------------------------------------------------------------

const colours = HighlightStyle.define([
  { tag: tags.propertyName, color: 'var(--syn-key)' },
  { tag: tags.string, color: 'var(--syn-str)' },
  { tag: tags.number, color: 'var(--syn-num)' },
  { tag: [tags.bool, tags.null], color: 'var(--syn-lit)' },
  { tag: [tags.separator, tags.brace, tags.squareBracket, tags.punctuation], color: 'var(--dim)' },
])

// --- The editor -------------------------------------------------------------

function extensions(): Extension[] {
  return [
    lineNumbers(),
    markGutter,
    highlightSpecialChars(),
    history(),
    drawSelection(),
    indentOnInput(),
    bracketMatching(),
    json(),
    syntaxHighlighting(colours),
    codeFolding({
      preparePlaceholder: (state, range) => state.doc.lineAt(range.to).number - state.doc.lineAt(range.from).number,
      placeholderDOM: placeholder,
    }),
    checked,
    EditorState.tabSize.of(2),
    indentUnit.of('  '),
    keymap.of([...defaultKeymap, ...historyKeymap, ...foldKeymap, indentWithTab]),
    EditorView.contentAttributes.of({ 'aria-label': props.label ?? '', spellcheck: 'false' }),
    EditorView.updateListener.of((u: ViewUpdate) => {
      if (!u.docChanged) return
      issue.value = u.state.field(checked).issue
      const value = u.state.doc.toString()
      if (value !== text.value) text.value = value
    }),
  ]
}

onMounted(() => {
  const state = EditorState.create({ doc: text.value, extensions: extensions() })
  issue.value = state.field(checked).issue
  view.value = new EditorView({ state, parent: host.value! })
})
onBeforeUnmount(() => view.value?.destroy())

// A new text from outside, e.g. another part of the template, replaces the document.
watch(text, (value) => {
  const v = view.value
  if (!v || value === v.state.doc.toString()) return
  v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: value } })
})

const reasons = computed<Record<JsonIssue['kind'], string>>(() => ({
  commaOrBrace: t('kit.jsonCommaOrBrace'),
  commaOrBracket: t('kit.jsonCommaOrBracket'),
  key: t('kit.jsonKey'),
  colon: t('kit.jsonColon'),
  value: t('kit.jsonValue'),
  end: t('kit.jsonEnd'),
  trailing: t('kit.jsonTrailing'),
  string: t('kit.jsonString'),
  number: t('kit.jsonNumber'),
}))
const reason = computed(() => (issue.value ? reasons.value[issue.value.kind] : ''))
</script>

<template>
  <div :class="['x-code-editor', { 'x-code-editor--fit': fit }]" :style="{ '--editor-h': `${height}px`, '--editor-phone-h': `${phoneHeight}px` }">
    <div ref="host" class="x-code-editor__host" />
    <div v-if="issue" class="x-code-editor__issue" role="status">
      <XIcon name="alert" :size="14" :stroke="1.5" />
      <span class="x-code-editor__where">{{ t('kit.jsonLine', { line: issue.line, reason }) }}</span>
      <span class="x-code-editor__kept">{{ t('kit.jsonNotApplied') }}</span>
    </div>
  </div>
</template>

<style scoped>
.x-code-editor {
  overflow: hidden;
  border: 1px solid var(--field-line);
  border-radius: var(--nest-r);
  background: var(--field);
  transition: border-color var(--t-fast);
}

.x-code-editor:focus-within {
  border-color: var(--acc-line);
}

.x-code-editor__host :deep(.cm-editor) {
  /* The height is for the text; 8px of room above and below it come on top. */
  height: calc(var(--editor-h) + 16px);
  outline: none;
}

.x-code-editor--fit .x-code-editor__host :deep(.cm-editor) {
  height: auto;
  max-height: calc(var(--editor-h) + 16px);
}

.x-code-editor__host :deep(.cm-scroller) {
  font: 400 14px / 20px var(--font-code);
}

.x-code-editor__host :deep(.cm-content) {
  padding: 8px 16px 8px 10px;
  color: var(--fg);
  caret-color: var(--fg);
}

.x-code-editor__host :deep(.cm-cursor) {
  border-left-color: var(--fg);
}

.x-code-editor__host :deep(.cm-selectionBackground),
.x-code-editor__host :deep(.cm-focused .cm-selectionBackground) {
  background: var(--acc-soft);
}

.x-code-editor__host :deep(.cm-matchingBracket) {
  background: var(--acc-soft);
  color: inherit;
}

.x-code-editor__host :deep(.cm-gutters) {
  border: 0;
  background: var(--gut);
  color: var(--faint);
}

.x-code-editor__host :deep(.cm-lineNumbers .cm-gutterElement) {
  min-width: 44px;
  padding: 0 6px 0 0;
  text-align: right;
}

.x-code-editor__host :deep(.cm-lineNumbers .cm-x-error-gutter) {
  color: var(--err-t);
  font-weight: 600;
}

.x-code-editor__host :deep(.cm-x-marks .cm-gutterElement) {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
}

.x-code-editor__host :deep(.cm-x-mark) {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: var(--r-4);
  color: var(--faint);
  cursor: pointer;
  transition: background-color var(--t-fast), color var(--t-fast);
}

.x-code-editor__host :deep(.cm-x-mark:hover) {
  background: var(--hov);
  color: var(--fg);
}

.x-code-editor__host :deep(.cm-x-mark--open svg) {
  transform: rotate(90deg);
}

.x-code-editor__host :deep(.cm-x-mark svg) {
  transition: transform var(--t-fast);
}

.x-code-editor__host :deep(.cm-x-mark--error) {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--err);
  box-shadow: 0 0 0 3px var(--err-ring);
  cursor: default;
}

.x-code-editor__host :deep(.cm-x-error-line) {
  background: var(--err-bg);
}

.x-code-editor__host :deep(.cm-x-error-text) {
  text-decoration: underline wavy var(--err);
  text-decoration-skip-ink: none;
  text-underline-offset: 4px;
}

.x-code-editor__host :deep(.cm-x-folded) {
  display: inline-flex;
  align-items: center;
  height: 16px;
  margin: 0 4px;
  padding: 0 6px;
  border: 1px solid var(--field-line);
  border-radius: var(--r-4);
  background: var(--chip);
  color: var(--mut);
  font: 500 var(--fs-10) / 1 var(--font-mono);
  vertical-align: 1px;
  cursor: pointer;
}

.x-code-editor__host :deep(.cm-x-folded:hover) {
  border-color: var(--field-hover);
  color: var(--fg);
}

.x-code-editor__host :deep(.cm-foldPlaceholder) {
  border: 0;
  background: none;
}

.x-code-editor__issue {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  padding: var(--sp-8) var(--sp-12);
  border-top: 1px solid var(--err-line);
  background: var(--err-bg);
  color: var(--err-t);
  font: 500 var(--fs-12) / var(--lh-tight) var(--font-sans);
}

.x-code-editor__where {
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-mono);
}

.x-code-editor__kept {
  margin-left: auto;
  color: var(--fg2);
  font-weight: 400;
}

@media (max-width: 767px) {
  .x-code-editor__host :deep(.cm-editor) {
    height: calc(var(--editor-phone-h) + 16px);
  }

  .x-code-editor--fit .x-code-editor__host :deep(.cm-editor) {
    height: auto;
    max-height: calc(var(--editor-phone-h) + 16px);
  }

  .x-code-editor__host :deep(.cm-lineNumbers .cm-gutterElement) {
    min-width: 34px;
  }
}
</style>
