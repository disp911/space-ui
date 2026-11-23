<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { locales, setLocale, type Locale } from '../../i18n'
import { XNumber, XSegmented, XSelect, XSettingRow, XSettingsSection } from '../../kit'
import { useTheme, type Theme } from '../../stores/theme'
import { useSettingsForm } from './form'
import type { SettingKey } from './model'
import { timeZoneOptions } from './timezones'

/**
 * How the panel looks and counts. The language and the theme belong to this
 * browser: they apply at once and need no saving.
 */
const { t, locale } = useI18n()
const theme = useTheme()
const f = useSettingsForm()

const open = reactive({ general: true, time: true, expiring: true })
const sections = {
  general: ['pageSize'],
  time: ['timeLocation'],
  expiring: ['expireDiff', 'trafficDiff'],
} satisfies Record<string, SettingKey[]>

// Every zone once, the saved one among them even when the browser does not list it.
const zones = computed(() => timeZoneOptions(t('settings.timeLocationServer'), f.saved.timeLocation))

const themes = computed(() => [
  { value: 'light' as Theme, label: t('settings.themeLight'), icon: 'sun' as const },
  { value: 'dark' as Theme, label: t('settings.themeDark'), icon: 'moon' as const },
])
</script>

<template>
  <div class="settings-tab">
    <XSettingsSection v-model:open="open.general" :title="t('settings.secGeneral')" :count="3" :modified="f.count(sections.general).dirty" :errors="f.count(sections.general).errors">
      <XSettingRow :title="t('settings.language')" compact>
        <XSelect :model-value="locale" :options="locales" @update:model-value="setLocale($event as Locale)" />
      </XSettingRow>
      <XSettingRow :title="t('settings.theme')" compact>
        <XSegmented :model-value="theme.theme" :options="themes" block :label="t('settings.theme')" @update:model-value="theme.set" />
      </XSettingRow>
      <XSettingRow :title="t('settings.pageSize')" :description="t('settings.pageSizeDesc')" narrow :modified="f.isDirty('pageSize')" :error="f.error('pageSize')">
        <XNumber v-model="f.draft.pageSize" :step="5" :modified="f.isDirty('pageSize')" :invalid="!!f.error('pageSize')" />
      </XSettingRow>
    </XSettingsSection>

    <XSettingsSection v-model:open="open.time" :title="t('settings.secDateTime')" :count="1" :modified="f.count(sections.time).dirty" :errors="f.count(sections.time).errors">
      <XSettingRow :title="t('settings.timeLocation')" :description="t('settings.timeLocationDesc')" :modified="f.isDirty('timeLocation')">
        <XSelect v-model="f.draft.timeLocation" :options="zones" search :modified="f.isDirty('timeLocation')" />
      </XSettingRow>
    </XSettingsSection>

    <XSettingsSection v-model:open="open.expiring" :title="t('settings.secExpiring')" :count="2" :modified="f.count(sections.expiring).dirty" :errors="f.count(sections.expiring).errors">
      <XSettingRow :title="t('settings.expireDiff')" :description="t('settings.expireDiffDesc')" narrow :modified="f.isDirty('expireDiff')" :error="f.error('expireDiff')">
        <XNumber v-model="f.draft.expireDiff" :unit="t('settings.unitDays')" :modified="f.isDirty('expireDiff')" :invalid="!!f.error('expireDiff')" />
      </XSettingRow>
      <XSettingRow :title="t('settings.trafficDiff')" :description="t('settings.trafficDiffDesc')" narrow :modified="f.isDirty('trafficDiff')" :error="f.error('trafficDiff')">
        <XNumber v-model="f.draft.trafficDiff" :unit="t('settings.unitGb')" :modified="f.isDirty('trafficDiff')" :invalid="!!f.error('trafficDiff')" />
      </XSettingRow>
    </XSettingsSection>
  </div>
</template>

<style scoped>
.settings-tab {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
