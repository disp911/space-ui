<script setup lang="ts">
import { computed, watchEffect } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatBytes, formatDate } from '../i18n/format'
import { locales, setLocale, type Locale } from '../i18n'
import {
  copyText, toast, XButton, XChip, XCopyFrame, XIcon, XKeyValue, XMenu, XPopover, XQrCard, XSelect, XSwitch, XToaster,
  type MenuItem, type Tone,
} from '../kit'
import { useTheme } from '../stores/theme'
import type { SubData } from './data'
import { apps, delayedDays, linkName, statusOf, type App } from './model'

/**
 * The subscription page of mockup «xpanel Subscription»: what a client sees
 * opening its subscription link in a browser. Codes and links copy on a
 * click; the app buttons open an app with the subscription or copy it for one.
 */
const props = defineProps<{ data: SubData | null }>()
const { t, locale } = useI18n()
const theme = useTheme()

watchEffect(() => (document.title = t('sub.title')))

const codes = computed(() => {
  const d = props.data
  if (!d) return []
  return [
    { label: t('sub.sub'), value: d.subUrl },
    { label: t('sub.subJson'), value: d.subJsonUrl },
    { label: 'Clash / Mihomo', value: d.subClashUrl },
  ].filter((c) => c.value)
})

const status = computed(() => (props.data ? statusOf(props.data) : 'inactive'))
const statusTone = computed<Tone>(() => ({ active: 'ok', unlimited: 'unl', inactive: 'err' } as const)[status.value])
const statusText = computed(() => ({ active: t('sub.active'), unlimited: t('sub.unlimited'), inactive: t('sub.inactive') })[status.value])

const rows = computed(() => {
  const d = props.data
  if (!d) return []
  const used = d.up + d.down
  const days = delayedDays(d.expiry)
  return [
    { label: t('sub.subId'), value: d.subId },
    { label: t('sub.status'), key: 'status' },
    { label: t('sub.downloaded'), value: formatBytes(d.down) },
    { label: t('sub.uploaded'), value: formatBytes(d.up) },
    { label: t('sub.usage'), value: formatBytes(used) },
    { label: t('sub.totalQuota'), value: d.total ? formatBytes(d.total) : '∞' },
    ...(d.total ? [{ label: t('sub.remained'), value: formatBytes(Math.max(d.total - used, 0)) }] : []),
    { label: t('sub.lastOnline'), value: d.lastOnline ? formatDate(d.lastOnline) : '-' },
    { label: t('sub.expiry'), value: days != null ? t('sub.afterFirstUse', { n: days }) : d.expiry ? formatDate(d.expiry) : t('sub.noExpiry') },
  ]
})

const links = computed(() => (props.data?.links ?? []).map((link, i) => ({ link, name: linkName(link) || t('sub.link', { n: i + 1 }) })))

const platforms = computed(() => {
  if (!props.data) return []
  const { android, ios } = apps(props.data)
  return [
    { name: 'Android', apps: android },
    { name: 'iOS', apps: ios },
  ]
})

const menuItems = (list: App[]): MenuItem[] => list.map((a) => ({ key: a.name, label: a.name }))

async function useApp(list: App[], name: string) {
  const app = list.find((a) => a.name === name)
  if (!app || !props.data) return
  if (app.open) {
    location.href = app.open
    return
  }
  const ok = await copyText(props.data.subUrl)
  toast(ok ? 'ok' : 'err', t(ok ? 'kit.copied' : 'kit.copyFailed'))
}

const languages = locales.map((l) => ({ value: l.value, label: `${l.flag}  ${l.label}` }))
const language = computed({
  get: () => locale.value as Locale,
  set: (l: Locale) => void setLocale(l),
})
const dark = computed({
  get: () => theme.theme === 'dark',
  set: (on: boolean) => theme.set(on ? 'dark' : 'light'),
})
</script>

<template>
  <main class="sub">
    <section v-if="data" class="sub__card">
      <header class="sub__head">
        <div class="sub__title">
          <h1 class="sub__name">{{ t('sub.title') }}</h1>
          <XChip mono truncate :title="data.subId">{{ data.subId }}</XChip>
        </div>
        <XPopover :label="t('sub.settings')" placement="bottom-end">
          <template #default="{ props: trigger }">
            <XButton v-bind="trigger" icon="gear" :label="t('sub.settings')" />
          </template>
          <template #content>
            <div class="sub-settings">
              <p class="sub-settings__title">{{ t('sub.settings') }}</p>
              <div class="sub-settings__theme">
                <XIcon name="sun" :size="14" />
                <XSwitch v-model="dark" :label="t('sub.darkTheme')" />
                <XIcon name="moon" :size="14" />
              </div>
              <label class="sub-settings__label" for="sub-language">{{ t('common.language') }}</label>
              <XSelect id="sub-language" v-model="language" :options="languages" />
            </div>
          </template>
        </XPopover>
      </header>

      <div class="sub__body">
        <div class="sub__codes">
          <XQrCard v-for="c in codes" :key="c.label" :label="c.label" :value="c.value" tone="unl" />
        </div>

        <XKeyValue :rows="rows" columns>
          <template #value-status>
            <XChip :tone="statusTone">{{ statusText }}</XChip>
          </template>
        </XKeyValue>

        <div v-if="links.length" class="sub__links">
          <XCopyFrame v-for="(l, i) in links" :key="i" :label="l.name" :text="l.link" />
        </div>

        <div class="sub__apps x-ctl-lg">
          <XMenu v-for="p in platforms" :key="p.name" v-slot="{ props: trigger }" :items="menuItems(p.apps)" placement="top" match-width @select="useApp(p.apps, $event)">
            <XButton v-bind="trigger" variant="primary" caret block>{{ p.name }}</XButton>
          </XMenu>
        </div>
      </div>
    </section>
  </main>
  <XToaster />
</template>

<style scoped>
.sub {
  display: flex;
  justify-content: center;
  align-items: flex-start;
  min-height: 100dvh;
  padding: var(--sp-48) var(--sp-16) var(--sp-56);
  background: var(--bg);
}

.sub__card {
  width: 760px;
  max-width: 100%;
  border: 1px solid var(--line);
  border-radius: var(--card-r);
  background: var(--card);
}

.sub__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-12);
  padding: var(--sp-14) var(--sp-16) var(--sp-14) var(--sp-20);
  border-bottom: 1px solid var(--div);
}

.sub__title {
  display: flex;
  align-items: center;
  gap: var(--sp-8);
  min-width: 0;
}

.sub__name {
  margin: 0;
  color: var(--fg);
  font: 600 var(--fs-16) / var(--lh-tight) var(--font-sans);
  white-space: nowrap;
}

.sub__body {
  display: flex;
  flex-direction: column;
  gap: var(--sp-20);
  padding: var(--sp-20) var(--sp-24) var(--sp-24);
}

.sub__codes {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: var(--sp-12);
}

.sub__codes > * {
  width: 220px;
}

.sub__links {
  display: flex;
  flex-direction: column;
  gap: var(--sp-16);
}

.sub__apps {
  display: grid;
  grid-template-columns: repeat(2, 240px);
  justify-content: center;
  gap: var(--sp-12);
}

.sub-settings {
  display: flex;
  flex-direction: column;
  gap: var(--sp-10);
  width: 216px;
}

.sub-settings__title {
  margin: 0;
  color: var(--fg);
  font: 600 var(--fs-13) / var(--lh-tight) var(--font-sans);
}

.sub-settings__theme {
  display: flex;
  align-items: center;
  gap: var(--sp-8);
  color: var(--mut);
}

.sub-settings__label {
  color: var(--mut);
  font: 500 var(--fs-12) / var(--lh-tight) var(--font-sans);
}

@media (max-width: 767px) {
  .sub {
    padding: var(--sp-8) var(--sp-8) var(--sp-24);
  }

  .sub__head {
    padding: var(--sp-12) var(--sp-12) var(--sp-12) var(--sp-16);
  }

  /* The client's id goes under the title when both do not fit. */
  .sub__title {
    flex-wrap: wrap;
    row-gap: var(--sp-4);
  }

  .sub__body {
    padding: var(--sp-16) var(--sp-14) var(--sp-20);
  }

  .sub__codes {
    flex-direction: column;
  }

  .sub__codes > * {
    width: auto;
  }

  .sub__links {
    gap: var(--sp-14);
  }

  .sub__apps {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--sp-10);
  }

  .sub-settings {
    width: 232px;
  }
}
</style>
