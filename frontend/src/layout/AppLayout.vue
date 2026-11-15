<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { XButton } from '../kit'
import AppLogo from './AppLogo.vue'
import AppNav from './AppNav.vue'

const { t } = useI18n()
const route = useRoute()
const host = location.hostname

const drawerOpen = ref(false)
const drawer = ref<HTMLElement | null>(null)
const menuButton = ref<InstanceType<typeof XButton> | null>(null)

function openDrawer() {
  drawerOpen.value = true
  nextTick(() => drawer.value?.querySelector<HTMLElement>('a, button')?.focus())
}

function closeDrawer() {
  if (!drawerOpen.value) return
  drawerOpen.value = false
  nextTick(() => (menuButton.value?.$el as HTMLElement | undefined)?.focus())
}

watch(() => route.fullPath, () => (drawerOpen.value = false))
</script>

<template>
  <div class="layout">
    <aside class="sidebar">
      <AppLogo class="sidebar__logo" />
      <AppNav />
    </aside>

    <header class="topbar">
      <XButton ref="menuButton" icon="menu" :label="t('nav.openMenu')" :aria-expanded="drawerOpen" @click="openDrawer" />
      <div class="topbar__text">
        <div class="topbar__title">{{ route.meta.titleKey ? t(route.meta.titleKey) : '' }}</div>
        <div class="topbar__host">{{ host }}</div>
      </div>
    </header>

    <Transition name="drawer">
      <div v-if="drawerOpen" class="drawer" @keydown.esc="closeDrawer">
        <!-- A tap beside the menu, Escape or a page chosen closes it. -->
        <div class="drawer__scrim" aria-hidden="true" @click="closeDrawer" />
        <div ref="drawer" class="drawer__panel" role="dialog" aria-modal="true" :aria-label="t('nav.main')">
          <div class="drawer__head">
            <AppLogo />
          </div>
          <AppNav @navigate="closeDrawer" />
        </div>
      </div>
    </Transition>

    <main class="content">
      <RouterView />
    </main>
  </div>
</template>

<style scoped>
.layout {
  display: grid;
  grid-template-columns: 236px minmax(0, 1fr);
  min-height: 100vh;
  min-height: 100dvh;
}

.sidebar {
  position: sticky;
  top: 0;
  height: 100vh;
  height: 100dvh;
  padding: 20px 14px 16px;
  border-right: 1px solid var(--line);
  background: var(--side);
}

/*
 * Logo, menu and logout in three rows; the outer rows are equal, so the
 * menu sits in the middle of the whole height, as in the mockup.
 */
.sidebar,
.drawer__panel {
  display: grid;
  grid-template-rows: 1fr auto 1fr;
}

.sidebar > :first-child,
.drawer__panel > :first-child {
  align-self: start;
}

.sidebar__logo {
  padding: 0 4px;
}

.topbar,
.drawer {
  display: none;
}

.content {
  /* Pages may stretch to the bottom of the window, as the dashboard does. */
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 20px 24px 24px;
}

@media (max-width: 767px) {
  .layout {
    display: flex;
    flex-direction: column;
  }

  .sidebar {
    display: none;
  }

  .topbar {
    position: sticky;
    top: 0;
    z-index: 10;
    display: flex;
    align-items: center;
    gap: 12px;
    height: var(--topbar-h);
    padding: 0 16px;
    border-bottom: 1px solid var(--div);
    background: var(--side);
  }

  .topbar__text {
    min-width: 0;
  }

  .topbar__title {
    font: 600 var(--fs-18) / var(--lh-tight) var(--font-sans);
    letter-spacing: -.4px;
  }

  .topbar__host {
    margin-top: 4px;
    overflow: hidden;
    color: var(--dim);
    font: 400 var(--fs-12) / var(--lh-text) var(--font-mono);
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  .drawer {
    position: fixed;
    inset: 0;
    z-index: 800;
    display: block;
  }

  .drawer__scrim {
    position: absolute;
    inset: 0;
    background: var(--scrim);
    backdrop-filter: var(--scrim-blur);
  }

  .drawer__panel {
    position: relative;
    width: min(282px, 85vw);
    height: 100%;
    padding: 16px 14px calc(16px + env(safe-area-inset-bottom));
    border-right: 1px solid var(--div);
    background: var(--side);
    box-shadow: var(--shadow-pop);
  }

  .drawer__head {
    display: flex;
    align-items: center;
    min-height: var(--ctl-h);
    padding-left: 4px;
  }

  /*
   * The menu slides in from the left while the scrim darkens and blurs the
   * page, as a window's sheet rises: the scrim is a layer of its own that only
   * fades, so the blur comes and goes smoothly, and both run on the compositor.
   */
  .drawer-enter-active,
  .drawer-leave-active {
    /* Keeps the drawer in the page while its panel and scrim move. */
    transition: opacity var(--t-sheet);
  }

  .drawer-enter-active .drawer__scrim,
  .drawer-leave-active .drawer__scrim {
    transition: opacity var(--t-sheet) var(--ease-sheet);
    will-change: opacity;
  }

  .drawer-enter-active .drawer__panel,
  .drawer-leave-active .drawer__panel {
    transition: transform var(--t-sheet) var(--ease-sheet);
    will-change: transform;
  }

  .drawer-enter-from .drawer__scrim,
  .drawer-leave-to .drawer__scrim {
    opacity: 0;
  }

  .drawer-enter-from .drawer__panel,
  .drawer-leave-to .drawer__panel {
    transform: translate3d(-100%, 0, 0);
  }

  @media (prefers-reduced-motion: reduce) {
    .drawer-enter-from .drawer__panel,
    .drawer-leave-to .drawer__panel {
      transform: none;
    }
  }

  /* Down to the bottom of the screen, where a page's action bar sits. */
  .content {
    flex: 1;
    padding: 12px 14px 24px;
  }
}
</style>
