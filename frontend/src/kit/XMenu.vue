<script setup lang="ts">
import { nextTick, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import XButton from './XButton.vue'
import XIcon from './XIcon.vue'
import XSpinner from './XSpinner.vue'
import { useFloating, type Placement } from './floating'
import { startSheetDrag } from './sheetDrag'
import type { IconName } from './icons'

export interface MenuItem {
  key: string
  label: string
  icon?: IconName
  danger?: boolean
  disabled?: boolean
  /** Thin line above the item. */
  divided?: boolean
  /** Makes the item a choice: a check marks the chosen one. */
  checked?: boolean
  /** Makes the item a switch, on by checked, e.g. «Enable» in a phone row's menu; choosing it keeps the menu open. */
  switch?: boolean
  /** The switch's change is being applied: a spinner in its knob. */
  busy?: boolean
  /** Colour of a small dot before the label, e.g. a log level. */
  dot?: string
}

/**
 * Action menu. The trigger is any button that binds the slot props:
 *   <XMenu :items="items" @select="run" v-slot="{ props }">
 *     <XButton v-bind="props" variant="ghost" icon="dots" label="More" />
 *   </XMenu>
 */
const props = withDefaults(
  defineProps<{
    items: MenuItem[]
    placement?: Placement
    title?: string
    /** On phones opens as a bottom sheet over a scrim: filters in windows. */
    sheet?: boolean
    /** Several items can be checked: choosing one toggles it and the menu stays open. */
    multiple?: boolean
    /** At least as wide as its button, e.g. the app buttons of the subscription page. */
    matchWidth?: boolean
  }>(),
  { placement: 'bottom-end' },
)
const emit = defineEmits<{ select: [key: string] }>()

const { t } = useI18n()
const id = useId()
const anchor = ref<HTMLElement | null>(null)
const menu = ref<HTMLElement | null>(null)
const open = ref(false)
const asSheet = ref(false)
const { style } = useFloating(anchor, menu, open, { placement: props.placement, matchWidth: props.matchWidth, onDismiss: () => close(!!menu.value?.contains(document.activeElement)) })

function enabledItems() {
  return [...(menu.value?.querySelectorAll<HTMLElement>('[role^="menuitem"]:not([aria-disabled="true"])') ?? [])]
}

function focusItem(index: number) {
  const list = enabledItems()
  // Without scrolling: a phone sheet is still below the screen's edge on its way in.
  list[(index + list.length) % list.length]?.focus({ preventScroll: true })
}

function show(first: 'first' | 'last' | 'none') {
  asSheet.value = props.sheet && matchMedia('(max-width: 767px)').matches
  open.value = true
  if (first !== 'none') nextTick(() => focusItem(first === 'first' ? 0 : -1))
}

function close(refocus: boolean) {
  open.value = false
  if (refocus) anchor.value?.focus({ preventScroll: true })
}

// A phone sheet comes down when pulled by its handle or title.
function onPointerDown(e: PointerEvent) {
  const target = e.target as HTMLElement
  if (!asSheet.value || !menu.value || !target.closest('.x-menu__grip, .x-menu__title')) return
  startSheetDrag(e, menu.value, () => close(false))
}

function onTriggerClick(e: MouseEvent) {
  anchor.value = e.currentTarget as HTMLElement
  if (open.value) close(false)
  else show(e.detail === 0 ? 'first' : 'none')
}

function onTriggerKeydown(e: KeyboardEvent) {
  anchor.value = e.currentTarget as HTMLElement
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    show(e.key === 'ArrowDown' ? 'first' : 'last')
  }
}

function onMenuKeydown(e: KeyboardEvent) {
  const list = enabledItems()
  const i = list.indexOf(document.activeElement as HTMLElement)
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    focusItem(i + (e.key === 'ArrowDown' ? 1 : -1))
  } else if (e.key === 'Home' || e.key === 'End') {
    e.preventDefault()
    focusItem(e.key === 'Home' ? 0 : -1)
  } else if (e.key === 'Tab') {
    close(false)
  }
}

function choose(item: MenuItem) {
  if (item.disabled || item.busy) return
  if (!props.multiple && !item.switch) close(true)
  emit('select', item.key)
}
</script>

<template>
  <slot
    :open="open"
    :props="{
      'aria-haspopup': 'menu' as const,
      'aria-expanded': open,
      'aria-controls': `${id}-menu`,
      onClick: onTriggerClick,
      onKeydown: onTriggerKeydown,
    }"
  />
  <Teleport to="body">
    <Transition name="x-menu-scrim">
      <div v-if="open && asSheet" class="x-menu-scrim" aria-hidden="true" />
    </Transition>
    <Transition :name="asSheet ? 'x-menu-sheet' : 'x-popup'">
      <div
        v-if="open"
        :id="`${id}-menu`"
        ref="menu"
        role="menu"
        :aria-label="title"
        :class="['x-popup', 'x-menu', { 'x-menu--sheet': asSheet }]"
        :style="asSheet ? undefined : style"
        @keydown="onMenuKeydown"
        @pointerdown="onPointerDown"
      >
        <div v-if="asSheet" class="x-menu__grip" aria-hidden="true" />
        <div v-if="title" class="x-caps x-menu__title" aria-hidden="true">{{ title }}</div>
        <template v-for="item in items" :key="item.key">
          <div v-if="item.divided" class="x-menu__divider" role="separator" />
          <button
            type="button"
            :role="item.switch || (item.checked !== undefined && multiple) ? 'menuitemcheckbox' : item.checked === undefined ? 'menuitem' : 'menuitemradio'"
            tabindex="-1"
            :aria-checked="item.checked"
            :aria-disabled="item.disabled || undefined"
            :class="['x-popup-item', { 'is-danger': item.danger, 'is-selected': item.checked }]"
            @click="choose(item)"
          >
            <span v-if="item.switch" :class="['x-menu__switch', { 'is-on': item.checked, 'is-busy': item.busy }]" aria-hidden="true">
              <span class="x-menu__knob"><XSpinner v-if="item.busy" :size="12" /></span>
            </span>
            <XIcon v-else-if="item.icon" :name="item.icon" :size="14" :stroke="1.4" />
            <span v-if="item.dot" class="x-menu__dot" :style="{ background: item.dot }" />
            <span class="x-menu__label">{{ item.label }}</span>
            <XIcon v-if="item.checked !== undefined && !item.switch" name="check" :size="12" :stroke="1.8" :class="['x-menu__check', { 'is-on': item.checked }]" />
          </button>
        </template>
        <!-- Each choice applies at once; the button closes the list. -->
        <div v-if="multiple" class="x-menu__foot"><XButton variant="primary" block @click="close(true)">{{ t('kit.apply') }}</XButton></div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
/* A switch in a menu item: smaller than XSwitch, drawn, as the whole item is the control. */
.x-menu__switch {
  position: relative;
  flex: none;
  width: 30px;
  height: 18px;
  border-radius: var(--r-pill);
  background: var(--track);
  transition: background-color var(--t-med);
}

.x-menu__switch.is-on {
  background: var(--acc);
}

.x-menu__switch.is-busy {
  opacity: .8;
}

.x-menu__knob {
  position: absolute;
  top: 3px;
  left: 3px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 12px;
  height: 12px;
  border-radius: var(--r-pill);
  background: var(--knob);
  color: var(--dim);
  box-shadow: var(--shadow-knob);
  transition: left var(--t-med) var(--ease);
}

.x-menu__switch.is-on .x-menu__knob {
  left: 15px;
  background: var(--knob-on);
  color: var(--acc);
}

.x-menu--sheet {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  max-height: calc(100% - 48px);
  padding: 10px 12px calc(16px + env(safe-area-inset-bottom));
  border-width: 1px 0 0;
  border-radius: var(--r-16) var(--r-16) 0 0;
  box-shadow: var(--shadow-sheet);
}

/* The handle, with room around it to take hold of; the bar sits where it did. */
.x-menu__grip {
  flex: none;
  align-self: stretch;
  height: 20px;
  margin: -10px 0 0;
  touch-action: none;
}

.x-menu__grip::before {
  content: '';
  display: block;
  width: 38px;
  height: 4px;
  margin: 10px auto 0;
  border-radius: var(--r-pill);
  background: var(--track);
}

.x-menu--sheet .x-menu__title {
  margin-bottom: 6px;
  padding: 0 6px;
  touch-action: none;
}

.x-menu--sheet .x-popup-item {
  --item-px: 12px;
  border-radius: var(--pop-item-r);
}

.x-menu--sheet .x-menu__check {
  width: 14px;
  height: 14px;
}

.x-menu-scrim {
  position: fixed;
  inset: 0;
  z-index: 999;
  background: var(--scrim);
  backdrop-filter: var(--scrim-blur);
}

.x-menu-scrim-enter-active,
.x-menu-scrim-leave-active {
  transition: opacity var(--t-sheet) var(--ease-sheet);
  will-change: opacity;
}

.x-menu-scrim-enter-from,
.x-menu-scrim-leave-to {
  opacity: 0;
}

/* As phone windows: up from the bottom unhurried, back down quicker. */
.x-menu-sheet-enter-active,
.x-menu-sheet-leave-active {
  transition: transform var(--t-sheet) var(--ease-sheet);
  will-change: transform;
}

.x-menu-sheet-enter-from,
.x-menu-sheet-leave-to {
  transform: translate3d(0, 100%, 0);
}

/* Less motion: the sheet fades in place. */
@media (prefers-reduced-motion: reduce) {
  .x-menu-sheet-enter-active,
  .x-menu-sheet-leave-active,
  .x-menu-scrim-enter-active,
  .x-menu-scrim-leave-active {
    transition: opacity var(--t-enter) ease !important;
  }

  .x-menu-sheet-enter-from,
  .x-menu-sheet-leave-to {
    opacity: 0;
    transform: none;
  }
}

.x-menu__title {
  padding: 8px 10px 6px;
}

.x-menu__dot {
  flex: none;
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.x-menu__label {
  flex: 1;
  min-width: 0;
}

.x-menu__check {
  visibility: hidden;
}

.x-menu__check.is-on {
  visibility: visible;
}

.x-menu__foot {
  display: flex;
  flex-direction: column;
  margin-top: 10px;
}

.x-menu__divider {
  height: 1px;
  margin: 4px 6px;
  background: var(--div);
}
</style>
