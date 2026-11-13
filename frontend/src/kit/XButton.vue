<script setup lang="ts">
import { computed, useSlots } from 'vue'
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import XIcon from './XIcon.vue'
import XSpinner from './XSpinner.vue'
import type { IconName } from './icons'

const props = withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'soft' | 'danger' | 'ghost' | 'ghost-danger' | 'dashed'
    /** sm is for text buttons; a square icon button is always the regular size. */
    size?: 'md' | 'sm'
    icon?: IconName
    loading?: boolean
    disabled?: boolean
    block?: boolean
    type?: 'button' | 'submit' | 'reset'
    /** Accessible name for an icon-only button. */
    label?: string
    /** Renders a link to a page outside the app. */
    href?: string
    /** Renders a link to a route of the app. */
    to?: RouteLocationRaw
    /** Opens a menu: a caret after the text. */
    caret?: boolean
    /** Values and codes: monospace text. */
    mono?: boolean
    /** On phones only the icon is shown; the text stays the accessible name. */
    phoneIconOnly?: boolean
    /** On phones the button takes the whole row at the regular control height. */
    phoneBlock?: boolean
    /** Done: a check draws in place of the icon and the button gives a small pop. */
    success?: boolean
    /** Waits for an earlier step, e.g. Restart for Save: disabled, in a dashed outline. */
    pending?: boolean
    /** A small badge at the top right of an icon button, e.g. pause on a disabled inbound. */
    mark?: IconName
    /** A small accent dot at the top right, e.g. auto refresh is on behind a menu button. */
    dot?: boolean
  }>(),
  { variant: 'secondary', size: 'md', type: 'button' },
)

const slots = useSlots()
const iconOnly = computed(() => !slots.default && !!(props.icon || slots.icon))
// Every square icon button is one size: 34, 38 on phones.
const sizeClass = computed(() => (iconOnly.value ? 'md' : props.size))
const iconSize = computed(() => (sizeClass.value === 'sm' ? 14 : 16))
const tag = computed(() => (props.to ? RouterLink : props.href ? 'a' : 'button'))
const isLink = computed(() => !!(props.to || props.href))
// Only the attributes of the element that is rendered: RouterLink builds its own href.
const linkAttrs = computed(() => (props.to ? { to: props.to } : props.href ? { href: props.href } : { type: props.type }))
</script>

<template>
  <component
    :is="tag"
    v-bind="linkAttrs"
    :class="[
      'x-btn',
      `x-btn--${variant}`,
      `x-btn--${sizeClass}`,
      { 'x-btn--icon': iconOnly, 'x-btn--block': block, 'x-btn--mono': mono, 'x-btn--phone-icon': phoneIconOnly && !iconOnly, 'x-btn--phone-block': phoneBlock, 'is-loading': loading, 'is-success': success, 'is-pending': pending },
    ]"
    :disabled="isLink ? undefined : disabled || pending"
    :aria-disabled="loading || (isLink && disabled) || undefined"
    :aria-busy="loading || undefined"
    :aria-label="label"
    :title="iconOnly ? label : undefined"
  >
    <XSpinner v-if="loading" :size="iconSize" />
    <XIcon v-else-if="success" name="check" :size="iconSize" :stroke="2" class="x-btn__icon x-btn__check" />
    <slot v-else name="icon"><XIcon v-if="icon" :name="icon" :size="iconSize" class="x-btn__icon" /></slot>
    <span v-if="$slots.default" class="x-btn__text"><slot /></span>
    <XIcon v-if="caret" name="caret" :size="12" :stroke="1.6" class="x-btn__caret" />
    <span v-if="dot" class="x-btn__dot" aria-hidden="true" />
    <span v-if="mark && iconOnly" class="x-btn__mark" aria-hidden="true"><XIcon :name="mark" :size="12" :stroke="2.4" /></span>
  </component>
</template>

<style scoped>
.x-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: var(--ctl-h);
  padding: 0 var(--btn-px);
  border: 1px solid transparent;
  border-radius: var(--btn-r);
  font: 600 var(--ctl-fs) / 1 var(--font-sans);
  white-space: nowrap;
  text-decoration: none;
  user-select: none;
  outline: none;
  transition: background-color var(--t-fast), border-color var(--t-fast), color var(--t-fast), box-shadow var(--t-fast), transform var(--t-fast);
}

/* Every button gives a little under the finger; each variant below darkens as well. */
.x-btn:active:not(:disabled):not([aria-disabled='true']),
.x-btn.is-active {
  transform: scale(var(--press-scale));
}

.x-btn[aria-disabled='true'] {
  pointer-events: none;
  opacity: .5;
}

.x-btn--sm {
  height: var(--ctl-sm-h);
  padding: 0 10px;
  gap: 6px;
  border-radius: var(--btn-r);
  font-size: var(--ctl-sm-fs);
}

/* A square never gives way in a crowded row. */
.x-btn--icon {
  flex-shrink: 0;
  width: var(--btn-sq);
  height: var(--btn-sq);
  padding: 0;
  border-radius: var(--btn-sq-r);
}

/* 16px, 18px on phones: the regular button grows there, its icon too. */
.x-btn--md .x-btn__icon {
  width: var(--btn-icon);
  height: var(--btn-icon);
}

.x-btn--block {
  display: flex;
  width: 100%;
}

/* Cut with an ellipsis when too long; the line is taller than the letters, so their tails (у, р, д) are not cut too. */
.x-btn__text {
  min-width: 0;
  overflow: hidden;
  line-height: var(--lh-tight);
  text-overflow: ellipsis;
}

.x-btn--mono {
  font-family: var(--font-mono);
  font-weight: 500;
}

/* Pushed to the far edge when the button is stretched. */
.x-btn__caret {
  flex: none;
  margin-left: auto;
  color: var(--dim);
}

/* A primary button opening a menu of actions, e.g. the apps of the subscription page, keeps it by its text, in its colour. */
.x-btn--primary .x-btn__caret {
  margin-left: 0;
  color: inherit;
}

@media (max-width: 767px) {
  .x-btn--phone-icon {
    position: relative;
    flex-shrink: 0;
    width: var(--btn-sq);
    height: var(--btn-sq);
    padding: 0;
    border-radius: var(--btn-sq-r);
  }

  .x-btn--phone-block {
    display: flex;
    width: 100%;
    height: var(--ctl-h);
    padding: 0 var(--btn-px);
    border-radius: var(--btn-r);
    font-size: var(--ctl-fs);
  }

  .x-btn--phone-icon .x-btn__text {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
    white-space: nowrap;
  }
}

.x-btn.is-success {
  animation: x-btn-pop .34s ease-out;
}

.x-btn__check {
  stroke-dasharray: 16;
  stroke-dashoffset: 16;
  animation: x-btn-check .34s .05s ease-out forwards;
}

@keyframes x-btn-pop {
  40% {
    transform: scale(.975);
  }
}

@keyframes x-btn-check {
  to {
    stroke-dashoffset: 0;
  }
}

.x-btn.is-loading {
  pointer-events: none;
  opacity: .8;
}

.x-btn:has(.x-btn__dot, .x-btn__mark) {
  position: relative;
}

.x-btn__dot {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 6px;
  height: 6px;
  border-radius: var(--r-pill);
  background: var(--acc);
}

.x-btn__mark {
  position: absolute;
  top: -3px;
  right: -3px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border: 2px solid var(--card);
  border-radius: var(--r-pill);
  background: var(--fg2);
  color: var(--card);
}


.x-btn:focus-visible,
.x-btn.is-focus {
  box-shadow: 0 0 0 2px var(--bg), 0 0 0 4px var(--acc);
}

/* primary */
.x-btn--primary {
  background: var(--acc);
  color: var(--acc-ink);
}
.x-btn--primary:hover:not(:disabled),
.x-btn--primary.is-hover {
  background: var(--acc-hover);
  box-shadow: 0 0 0 3px var(--acc-ring), 0 0 14px var(--acc-glow);
}
.x-btn--primary:active:not(:disabled),
.x-btn--primary.is-active {
  background: var(--acc-press);
  box-shadow: none;
}

/* secondary */
.x-btn--secondary {
  background: var(--ctrl);
  border-color: var(--field-line);
  color: var(--fg2);
  font-weight: 500;
}
.x-btn--secondary:hover:not(:disabled),
.x-btn--secondary.is-hover {
  border-color: var(--field-hover);
  color: var(--fg);
}
.x-btn--secondary:focus-visible,
.x-btn--secondary.is-focus {
  border-color: var(--acc-line);
  box-shadow: 0 0 0 3px var(--acc-ring);
}
.x-btn--secondary:active:not(:disabled),
.x-btn--secondary.is-active {
  background: var(--press);
  border-color: var(--field-hover);
  color: var(--fg);
}

/* soft */
.x-btn--soft {
  background: var(--acc-soft);
  border-color: var(--acc-line);
  color: var(--acc-t);
}
.x-btn--soft:hover:not(:disabled),
.x-btn--soft.is-hover {
  border-color: var(--acc);
}

/* danger */
.x-btn--danger {
  background: var(--err-bg);
  border-color: var(--err-line);
  color: var(--err-t);
}
.x-btn--danger:hover:not(:disabled),
.x-btn--danger.is-hover {
  box-shadow: 0 0 0 3px var(--err-ring);
}
.x-btn--danger:focus-visible,
.x-btn--danger.is-focus {
  box-shadow: 0 0 0 2px var(--bg), 0 0 0 4px var(--err);
}
.x-btn--danger:active:not(:disabled),
.x-btn--danger.is-active {
  background: var(--err-ring);
  border-color: var(--err);
}

/* dashed: adds one more of something to a list */
.x-btn--dashed {
  border: 1px dashed var(--field-hover);
  background: transparent;
  color: var(--fg2);
  font-weight: 500;
}
.x-btn--dashed:hover:not(:disabled),
.x-btn--dashed.is-hover {
  border-color: var(--acc-line);
  background: var(--acc-soft);
  color: var(--acc-t);
}
.x-btn--dashed:focus-visible,
.x-btn--dashed.is-focus {
  border-color: var(--acc-line);
  box-shadow: 0 0 0 3px var(--acc-ring);
}

/* ghost: row actions, "more" buttons */
.x-btn--ghost {
  background: transparent;
  color: var(--dim);
  font-weight: 500;
}
.x-btn--ghost:hover:not(:disabled),
.x-btn--ghost.is-hover {
  background: var(--hov);
  color: var(--fg);
}
.x-btn--ghost:focus-visible,
.x-btn--ghost.is-focus {
  box-shadow: 0 0 0 2px var(--acc-line);
  color: var(--fg);
}
.x-btn--ghost:active:not(:disabled),
.x-btn--ghost.is-active {
  background: var(--press);
  color: var(--fg);
}

/* ghost-danger: delete action in a row */
.x-btn--ghost-danger {
  background: transparent;
  color: var(--err-t);
  font-weight: 500;
}
.x-btn--ghost-danger:hover:not(:disabled),
.x-btn--ghost-danger.is-hover {
  background: var(--err-bg);
}
.x-btn--ghost-danger:focus-visible,
.x-btn--ghost-danger.is-focus {
  box-shadow: 0 0 0 2px var(--err-line);
}
.x-btn--ghost-danger:active:not(:disabled),
.x-btn--ghost-danger.is-active {
  background: var(--err-ring);
}

.x-btn:disabled {
  cursor: not-allowed;
  background: var(--ctrl);
  border-color: var(--line);
  color: var(--faint);
  box-shadow: none;
}
.x-btn--ghost:disabled,
.x-btn--ghost-danger:disabled {
  background: transparent;
  border-color: transparent;
}
.x-btn.is-pending {
  background: transparent;
  border: 1px dashed var(--field-hover);
}
</style>
