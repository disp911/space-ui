<script setup lang="ts">
/**
 * Page frame: the title with an optional subtitle on desktop (the top bar
 * shows the title on phones), actions on the right and an aside — a notice
 * that sits next to the title on desktop and above the content on phones.
 *
 * Pages that edit settings add a banner above the title, tabs under it and a
 * bar of buttons that phones pin to the bottom of the screen in place of the
 * actions. With fixed, the banner, the title and the tabs stay on screen on
 * desktop while the content scrolls; on phones the tabs stay under the top bar.
 * While busy, e.g. a restart, a line runs under the tabs and the content dims.
 */
import { XProgressLine } from '../kit'

defineProps<{ title: string; fixed?: boolean; busy?: boolean }>()
</script>

<template>
  <div :class="['page', { 'page--fixed': fixed, 'page--bar': $slots.bar, 'page--busy': busy }]">
    <div class="page__top">
      <div v-if="$slots.banner" class="page__banner"><slot name="banner" /></div>
      <header class="page__head">
        <div class="page__heading">
          <h1 class="page__title">{{ title }}</h1>
          <p v-if="$slots.subtitle" class="page__subtitle"><slot name="subtitle" /></p>
        </div>
        <div v-if="$slots.actions" class="page__actions"><slot name="actions" /></div>
        <div v-if="$slots.aside" class="page__aside"><slot name="aside" /></div>
      </header>
      <div v-if="$slots.tabs" class="page__tabs">
        <slot name="tabs" />
        <XProgressLine v-if="busy" class="page__busy" />
      </div>
    </div>
    <slot />
    <div v-if="$slots.bar" class="page__bar"><slot name="bar" /></div>
  </div>
</template>

<style scoped>
.page {
  display: flex;
  flex: 1 0 auto;
  flex-direction: column;
  gap: 16px;
}

.page__top {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* Over the content while it scrolls: reaches the edges of the content area. */
.page--fixed .page__top {
  position: sticky;
  top: 0;
  z-index: 10;
  margin: -20px -24px 0;
  padding: 20px 24px 0;
  background: var(--bg);
}

.page__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px 20px;
  min-height: var(--ctl-h);
}

.page__heading {
  /* Keeps its text whole: the actions move to a line of their own instead. */
  flex: 1 0 auto;
  max-width: 100%;
}

.page__title {
  margin: 0;
  font: 600 var(--fs-20) / var(--lh-tight) var(--font-sans);
  letter-spacing: -.5px;
}

.page__subtitle {
  margin: 4px 0 0;
  overflow: hidden;
  color: var(--dim);
  font-size: var(--fs-14);
  white-space: nowrap;
  text-overflow: ellipsis;
}

.page__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px;
}

.page__aside {
  flex: 0 1 calc(50% - 6px);
  min-width: 0;
}

/* The tabs sit 14px under the title: the column gap less the 6px they borrow back. */
.page__tabs {
  position: relative;
  margin-top: -6px;
}

/* Over the line under the tabs, across the whole top of a fixed page. */
.page__busy {
  position: absolute;
  right: 0;
  bottom: -1px;
  left: 0;
}

.page--fixed .page__busy {
  right: -24px;
  left: -24px;
}

.page--busy > :deep(:not(.page__top, .page__bar)) {
  opacity: .5;
  filter: saturate(.7);
  pointer-events: none;
  transition: opacity var(--t-med), filter var(--t-med);
}

.page__bar {
  display: none;
}

@media (max-width: 767px) {
  .page--fixed {
    gap: 12px;
  }

  .page__head {
    min-height: 0;
  }

  /* The top bar already shows the title and the server. */
  .page__heading {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    white-space: nowrap;
  }

  .page__head:not(:has(.page__actions)) {
    display: contents;
  }

  .page__aside {
    flex-basis: 100%;
  }

  /* The parts of the top become items of the page, so the banner can follow the tabs. */
  .page__top,
  .page--fixed .page__top {
    display: contents;
  }

  /* Phones: the bar carries the actions. */
  .page--bar .page__actions {
    display: none;
  }

  .page--bar .page__head:has(.page__actions):not(:has(.page__aside)) {
    display: contents;
  }

  /* A strip under the top bar, from edge to edge; the tabs draw their own line. */
  .page__tabs {
    order: -1;
    margin: -12px -14px 0;
    background: var(--bg);
  }

  .page--fixed .page__tabs {
    position: sticky;
    top: var(--topbar-h);
    z-index: 10;
  }

  /* The strip already reaches the edges. */
  .page--fixed .page__busy {
    right: 0;
    left: 0;
  }

  /* Pinned to the bottom of the screen, from edge to edge, over the content's bottom padding. */
  .page__bar {
    position: sticky;
    bottom: 0;
    z-index: 10;
    display: block;
    margin: auto -14px -24px;
  }
}
</style>
