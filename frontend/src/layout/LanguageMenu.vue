<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { locales, setLocale, type Locale } from '../i18n'
import { XButton, XMenu, type MenuItem } from '../kit'

/** Language picker: a kit menu dropping from its button, the current language checked, on phones too. */
const { t, locale } = useI18n()

const code = (l: string) => l.slice(0, 2).toUpperCase()
const items = computed<MenuItem[]>(() =>
  locales.map((l) => ({ key: l.value, label: l.label, checked: l.value === locale.value })),
)
</script>

<template>
  <XMenu v-slot="{ props }" :items="items" @select="setLocale($event as Locale)">
    <XButton v-bind="props" icon="globe" :label="t('common.language')" caret>{{ code(locale) }}</XButton>
  </XMenu>
</template>

