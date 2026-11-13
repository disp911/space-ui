<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import XChip from './XChip.vue'
import XQrCode from './XQrCode.vue'
import { usePhone } from './media'
import type { Tone } from './types'

/**
 * A QR code to scan under its name in a small frame, e.g. a share link or a
 * subscription. A click on the code copies what it holds, said in a toast.
 * On phones the frame takes the width and the code grows.
 */
withDefaults(defineProps<{ label: string; value: string; tone?: Tone }>(), { tone: 'ok' })
const { t } = useI18n()
const phone = usePhone()
</script>

<template>
  <div class="x-qr-card">
    <XChip :tone="tone" truncate :title="label">{{ label }}</XChip>
    <XQrCode :value="value" :copy="value" :copy-label="t('kit.copy')" :size="phone ? 240 : 190" quiet />
  </div>
</template>

<style scoped>
.x-qr-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--sp-10);
  min-width: 0;
  padding: var(--sp-14);
  border: 1px solid var(--div);
  border-radius: var(--nest-r);
  background: var(--tint);
}
</style>
