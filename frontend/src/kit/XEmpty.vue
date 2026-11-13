<script setup lang="ts">
import XIcon from './XIcon.vue'
import type { IconName } from './icons'

/**
 * Nothing to show yet: an icon in a dashed frame, a title and a hint. Fills and
 * centres in its box. tone err is a failed load: a red icon and a plain title.
 * framed is an empty list among other content: the dashed frame goes round the
 * whole, without an icon, as tall as what it holds. tab is a whole empty tab
 * in an XCard, e.g. no balancers: the icon in a solid frame on the nested
 * surface, roomier and as tall as what it holds.
 */
withDefaults(defineProps<{ title?: string; hint?: string; icon?: IconName; tone?: 'err'; framed?: boolean; tab?: boolean }>(), { icon: 'doc' })
</script>

<template>
  <div :class="['x-empty', { 'x-empty--err': tone === 'err', 'x-empty--framed': framed, 'x-empty--tab': tab }]" :role="tone === 'err' ? 'alert' : undefined">
    <span v-if="!framed" class="x-empty__icon"><XIcon :name="icon" :size="16" :stroke="tone === 'err' ? 1.4 : 1.3" /></span>
    <span v-if="title" class="x-empty__title">{{ title }}</span>
    <span v-if="hint" class="x-empty__hint">{{ hint }}</span>
    <div v-if="$slots.default" class="x-empty__actions"><slot /></div>
  </div>
</template>

<style scoped>
.x-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 100%;
  min-height: 200px;
  padding: 0 16px;
  text-align: center;
}

.x-empty--framed {
  gap: 12px;
  height: auto;
  min-height: 0;
  padding: 28px 16px;
  border: 1px dashed var(--field-line);
  border-radius: var(--nest-r);
}

/* With the card's 16px around it: 44px above and below, 20px at the sides. */
.x-empty--tab {
  gap: 14px;
  height: auto;
  min-height: 0;
  padding: 28px 4px;
}

.x-empty--tab .x-empty__icon {
  border: 1px solid var(--div);
  background: var(--nest);
  color: var(--dim);
}

/* Buttons under the title, side by side. */
.x-empty__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
}

.x-empty__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: 1px dashed var(--track);
  border-radius: var(--r-12);
  color: var(--faint);
}

.x-empty__title {
  color: var(--mut);
  font: 500 var(--fs-13) / var(--lh-tight) var(--font-sans);
}

.x-empty__hint {
  color: var(--dim);
  font: 400 var(--fs-12) / var(--lh-text) var(--font-sans);
  text-wrap: pretty;
}

.x-empty--err .x-empty__icon {
  border: 1px solid var(--err-line);
  background: var(--err-bg);
  color: var(--err-t);
}

.x-empty--err .x-empty__title {
  color: var(--fg);
  font-size: var(--fs-14);
  font-weight: 600;
}
</style>
