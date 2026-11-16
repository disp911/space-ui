<script setup lang="ts">
import XIcon from './XIcon.vue'
import type { IconName } from './icons'
import type { Tone } from './types'

/** Value chip coloured by status: traffic, expiry, client counts. */
const props = withDefaults(
  defineProps<{
    tone?: Tone
    icon?: IconName
    /** Small status dot before the text (online). */
    dot?: boolean
    mono?: boolean
    /** The system monospace, as code is: an IP log's addresses. */
    code?: boolean
    /** Renders a button: opens a popover or a tooltip. */
    interactive?: boolean
    /** Only the frame and the text, e.g. a transport beside a filled protocol. */
    outline?: boolean
    /** A long value, e.g. a key: at most 300px and its place, cut with an ellipsis. */
    truncate?: boolean
  }>(),
  { tone: 'neutral' },
)
</script>

<template>
  <component
    :is="props.interactive ? 'button' : 'span'"
    :type="props.interactive ? 'button' : undefined"
    :class="['x-chip', `x-chip--${tone}`, { 'x-chip--mono': mono, 'x-chip--code': code, 'x-chip--interactive': interactive, 'x-chip--outline': outline, 'x-chip--truncate': truncate }]"
  >
    <span v-if="dot" class="x-chip__dot" />
    <XIcon v-if="icon" :name="icon" :size="12" :stroke="1.6" />
    <span v-if="truncate" class="x-chip__text"><slot /></span>
    <slot v-else />
  </component>
</template>

<style scoped>
.x-chip {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 4px;
  height: var(--chip-h);
  padding: 0 8px;
  border: 1px solid var(--tone-line);
  border-radius: var(--chip-r);
  background: var(--tone-bg);
  color: var(--tone-t);
  font: 500 var(--fs-11) / 1 var(--font-sans);
  white-space: nowrap;
  outline: none;
}

.x-chip--outline {
  background: transparent;
}

.x-chip--mono {
  font-family: var(--font-mono);
}

.x-chip--code {
  font-family: var(--font-code);
}

.x-chip--truncate {
  flex: 0 1 auto;
  min-width: 0;
  max-width: min(300px, 100%);
}

/* A line of its own height inside the chip's tight one, so the ellipsis' clip leaves the tails (g, y) whole. */
.x-chip__text {
  overflow: hidden;
  line-height: var(--lh-text);
  text-overflow: ellipsis;
}

.x-chip__dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--tone);
}

.x-chip--interactive {
  cursor: pointer;
  transition: filter var(--t-fast), transform var(--t-fast), box-shadow var(--t-fast);
}

.x-chip--interactive:hover,
.x-chip--interactive.is-hover {
  filter: brightness(1.15);
}

.x-chip--interactive:focus-visible,
.x-chip--interactive.is-focus {
  box-shadow: 0 0 0 2px var(--card), 0 0 0 4px var(--acc-line);
}

.x-chip--interactive:active,
.x-chip--interactive.is-active {
  transform: scale(var(--press-scale-chip));
}

.x-chip--neutral {
  --tone: var(--bar-unl);
  --tone-t: var(--fg2);
  --tone-bg: var(--chip);
  --tone-line: var(--chip-line);
}
.x-chip--ok {
  --tone: var(--ok);
  --tone-t: var(--ok-t);
  --tone-bg: var(--ok-bg);
  --tone-line: var(--ok-line);
}
.x-chip--warn {
  --tone: var(--warn);
  --tone-t: var(--warn-t);
  --tone-bg: var(--warn-bg);
  --tone-line: var(--warn-line);
}
.x-chip--err {
  --tone: var(--err);
  --tone-t: var(--err-t);
  --tone-bg: var(--err-bg);
  --tone-line: var(--err-line);
}
.x-chip--unl {
  --tone: var(--unl);
  --tone-t: var(--unl-t);
  --tone-bg: var(--unl-bg);
  --tone-line: var(--unl-line);
}
.x-chip--on {
  --tone: var(--on);
  --tone-t: var(--on-t);
  --tone-bg: var(--on-bg);
  --tone-line: var(--on-line);
}
.x-chip--off {
  --tone: var(--off);
  --tone-t: var(--off-t);
  --tone-bg: var(--off-bg);
  --tone-line: var(--off-line);
}
.x-chip--neutral.x-chip--outline {
  --tone-t: var(--mut);
  --tone-line: var(--field-line);
}

.x-chip--ok.x-chip--outline {
  --tone-t: var(--acc);
  --tone-line: var(--acc-line);
}
</style>
