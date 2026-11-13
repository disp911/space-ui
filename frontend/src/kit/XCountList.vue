<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import XIcon from './XIcon.vue'
import XTooltip from './XTooltip.vue'
import { useFloating } from './floating'
import { usePhone } from './media'
import type { Tone } from './types'

/**
 * A count of people in a state, e.g. depleted clients: a chip in the state's
 * tone that opens the list of their emails, a comment behind the speech
 * mark of those that have one. labelled shows the state's name in the chip,
 * as in a summary; dot marks those online.
 */
const props = defineProps<{ tone: Tone; label: string; items: { email: string; comment?: string }[]; labelled?: boolean; dot?: boolean }>()
const { t } = useI18n()
const phone = usePhone()

const id = useId()
const anchor = ref<HTMLElement | null>(null)
const list = ref<HTMLElement | null>(null)
const open = ref(false)
const { style } = useFloating(anchor, list, open, { onDismiss: () => (open.value = false) })

const panelStyle = computed(() => (phone.value && anchor.value ? { ...style.value, width: `${Math.min(window.innerWidth - 16, 360)}px` } : style.value))
</script>

<template>
  <button
    ref="anchor"
    type="button"
    :class="['x-count', `x-count--${tone}`]"
    :aria-label="`${label}: ${items.length}`"
    :aria-expanded="open"
    :aria-controls="`${id}-list`"
    :title="labelled ? undefined : label"
    @click="open = !open"
  >
    <span v-if="dot" class="x-count__dot" />
    <span v-if="labelled" class="x-count__label">{{ label }}</span>
    {{ items.length }}
  </button>
  <Teleport to="body">
    <Transition name="x-popup">
      <div v-if="open" :id="`${id}-list`" ref="list" role="dialog" :aria-label="label" class="x-count__list" :style="panelStyle">
        <div class="x-count__head">
          <span :class="['x-count__mark', `x-count--${tone}`]" />
          <span class="x-count__title">{{ label }}</span>
          <span class="x-count__n">{{ items.length }}</span>
        </div>
        <div class="x-count__items">
          <div v-for="item in items" :key="item.email" class="x-count__item">
            <span class="x-count__email" :title="item.email">{{ item.email }}</span>
            <XTooltip v-if="item.comment" :text="item.comment" :delay="0">
              <button type="button" class="x-count__comment" :aria-label="t('kit.comment', { text: item.comment })">
                <XIcon name="comment" :size="12" />
              </button>
            </XTooltip>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.x-count {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 4px;
  height: var(--chip-h);
  padding: 0 8px;
  border: 1px solid var(--tone-line);
  border-radius: var(--r-6);
  background: var(--tone-bg);
  color: var(--tone-t);
  font: 500 var(--fs-12) / 1 var(--font-mono);
  white-space: nowrap;
  outline: none;
  transition: filter var(--t-fast), transform var(--t-fast), box-shadow var(--t-fast);
}

.x-count:hover,
.x-count.is-hover {
  filter: brightness(1.15);
}

.x-count:focus-visible,
.x-count.is-focus {
  box-shadow: 0 0 0 2px var(--card), 0 0 0 4px var(--acc-line);
}

.x-count:active,
.x-count.is-active {
  transform: scale(var(--press-scale-chip));
}

.x-count__dot {
  width: 6px;
  height: 6px;
  border-radius: var(--r-pill);
  background: var(--tone);
}

.x-count__label {
  font-family: var(--font-sans);
}

.x-count--ok { --tone: var(--ok); --tone-t: var(--ok-t); --tone-bg: var(--ok-bg); --tone-line: var(--ok-line); }
.x-count--warn { --tone: var(--warn); --tone-t: var(--warn-t); --tone-bg: var(--warn-bg); --tone-line: var(--warn-line); }
.x-count--err { --tone: var(--err); --tone-t: var(--err-t); --tone-bg: var(--err-bg); --tone-line: var(--err-line); }
.x-count--unl { --tone: var(--unl); --tone-t: var(--unl-t); --tone-bg: var(--unl-bg); --tone-line: var(--unl-line); }
.x-count--on { --tone: var(--on); --tone-t: var(--on-t); --tone-bg: var(--on-bg); --tone-line: var(--on-line); }
.x-count--off { --tone: var(--off); --tone-t: var(--off-t); --tone-bg: var(--off-bg); --tone-line: var(--off-line); }
.x-count--neutral { --tone: var(--bar-unl); --tone-t: var(--fg2); --tone-bg: var(--chip); --tone-line: var(--chip-line); }

.x-count__list {
  z-index: 1000;
  display: flex;
  flex-direction: column;
  width: 280px;
  max-width: calc(100vw - 16px);
  padding: 6px;
  border: 1px solid var(--line);
  border-radius: var(--nest-r);
  background: var(--dlg);
  box-shadow: var(--shadow-pop);
}

.x-count__head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px 8px;
  border-bottom: 1px solid var(--div);
}

.x-count__mark {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: var(--r-pill);
  background: var(--tone);
}

.x-count__title {
  flex: 1;
  color: var(--fg);
  font: 600 var(--fs-13) / var(--lh-tight) var(--font-sans);
}

.x-count__n {
  color: var(--dim);
  font: 500 var(--fs-12) / 1 var(--font-mono);
}

.x-count__items {
  max-height: 232px;
  padding: 4px 0 2px;
  overflow: hidden auto;
  scrollbar-gutter: stable;
}

.x-count__item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 28px;
  padding: 0 8px;
  border-radius: var(--pop-item-r);
}

.x-count__item:hover {
  background: var(--hov);
}

.x-count__email {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: var(--fg2);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-mono);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.x-count__comment {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: 0;
  border-radius: var(--btn-r);
  background: transparent;
  color: var(--dim);
  cursor: help;
  outline: none;
}

.x-count__comment:hover {
  background: var(--hov);
  color: var(--fg);
}

.x-count__comment:focus-visible {
  box-shadow: 0 0 0 2px var(--acc-line);
}

@media (max-width: 767px) {
  .x-count__item {
    min-height: 36px;
    padding-right: 4px;
  }

  .x-count__email {
    font-size: var(--fs-13);
  }

  .x-count__comment {
    width: var(--ctl-h);
    height: var(--ctl-h);
  }
}
</style>
