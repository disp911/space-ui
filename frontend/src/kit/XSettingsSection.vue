<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import XChip from './XChip.vue'
import XCollapse from './XCollapse.vue'
import XIcon from './XIcon.vue'

/**
 * A card of settings that folds: a header with a chevron, the title and the
 * number of rows, and on the right a note (desktop only) and the counts of
 * changed fields and errors inside. The rows (XSettingRow, XSettingDivider) are the slot. When
 * the header is too narrow the badges move to a line of their own.
 */
defineProps<{
  title: string
  /** Rows inside, shown after the title. */
  count?: number
  /** A short remark at the right of the header. */
  note?: string
  modified?: number
  errors?: number
  /** Free content in place of rows, e.g. a single button, kept off the card's edges. */
  padded?: boolean
}>()
const open = defineModel<boolean>('open', { default: true })
const { t } = useI18n()
</script>

<template>
  <section :class="['x-settings', { 'is-open': open }]">
    <button type="button" class="x-settings__head" :aria-expanded="open" @click="open = !open">
      <XIcon name="chevron" :size="14" :stroke="1.6" class="x-settings__chevron" />
      <span class="x-settings__title">{{ title }}</span>
      <span v-if="count" class="x-settings__count">{{ count }}</span>
      <span v-if="note || modified || errors" class="x-settings__badges">
        <XChip v-if="note" class="x-settings__note">{{ note }}</XChip>
        <XChip v-if="modified" tone="on" dot mono>{{ t('kit.modified', { n: modified }) }}</XChip>
        <XChip v-if="errors" tone="err" mono>{{ t('kit.errors', { n: errors }) }}</XChip>
      </span>
    </button>
    <XCollapse :open="open">
      <div :class="['x-settings__body', { 'x-settings__body--padded': padded }]"><slot /></div>
    </XCollapse>
  </section>
</template>

<style scoped>
.x-settings {
  min-width: 0;
  border: 1px solid var(--line);
  border-radius: var(--r-16);
  background: var(--card);
  /* Tinted rows follow the rounded corners; popups are teleported, so nothing is cut. */
  overflow: clip;
}

.x-settings__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  width: 100%;
  min-height: 45px;
  padding: 6px 20px;
  border: 0;
  background: none;
  color: var(--fg);
  text-align: left;
  cursor: pointer;
  user-select: none;
  outline: none;
  transition: background-color var(--t-fast);
}

.x-settings__head:hover {
  background: var(--hov);
}

.x-settings__head:active {
  background: var(--press);
}

/* Keyboard focus marks the chevron, not the whole header. */
.x-settings__head:focus-visible .x-settings__chevron {
  background: var(--acc-soft);
  color: var(--acc-t);
  box-shadow: 0 0 0 4px var(--acc-soft), 0 0 0 5px var(--acc-line);
}

.x-settings__body--padded {
  padding: 4px 20px 18px;
}

.x-settings__chevron {
  flex: none;
  border-radius: 2px;
  color: var(--dim);
  /* The focus box is always full size and only fades in, so it never grows out of a square. */
  box-shadow: 0 0 0 4px transparent, 0 0 0 5px transparent;
  transition: transform var(--t-fast), color var(--t-fast), background-color var(--t-fast), box-shadow var(--t-fast);
}

.is-open .x-settings__chevron {
  transform: rotate(90deg);
}

.x-settings__title {
  flex: 0 1 auto;
  min-width: 0;
  overflow: hidden;
  font: 600 var(--fs-14) / var(--lh-tight) var(--font-sans);
  letter-spacing: -.2px;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.x-settings__count {
  flex: none;
  color: var(--faint);
  font: 500 var(--fs-11) / 1 var(--font-mono);
}

.x-settings__badges {
  display: flex;
  flex: 0 1 auto;
  flex-wrap: wrap;
  justify-content: flex-end;
  align-items: center;
  gap: 6px;
  min-width: 0;
  margin-left: auto;
}

@media (max-width: 767px) {
  .x-settings__head {
    min-height: var(--menu-item-h);
    padding: 6px 16px;
  }

  /* A long title takes a second line rather than losing its end. */
  .x-settings__title {
    white-space: normal;
    text-wrap: balance;
  }

  .x-settings__body--padded {
    padding: 4px 16px 18px;
  }

  /* Phones keep the header to the title and the counts. */
  .x-settings__note {
    display: none;
  }
}
</style>
