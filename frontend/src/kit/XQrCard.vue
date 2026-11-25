<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { encode } from 'uqr'
import XChip from './XChip.vue'
import XInlineButton from './XInlineButton.vue'
import XModal from './XModal.vue'
import XQrCode from './XQrCode.vue'
import { usePhone } from './media'
import type { Tone } from './types'

/**
 * A QR code to scan under its name in a small frame, e.g. a share link or a
 * subscription. A click on the code copies what it holds, said in a toast.
 * On phones the frame takes the width and the code grows. A code too dense
 * for its frame, under three screen pixels a dot (a long link, e.g. one with
 * a post-quantum key), gets a button by its name that shows it large in a
 * window of its own: as large as the screen allows, the width on phones.
 */
const props = withDefaults(defineProps<{ label: string; value: string; tone?: Tone }>(), { tone: 'ok' })
const { t } = useI18n()
const phone = usePhone()

const size = computed(() => (phone.value ? 240 : 190))
const dense = computed(() => encode(props.value, { ecc: 'L', border: 2 }).size * 3 > size.value)

const big = ref(false)
</script>

<template>
  <div class="x-qr-card">
    <div :class="['x-qr-card__head', { 'x-qr-card__head--dense': dense }]">
      <XChip :tone="tone" truncate :title="label">{{ label }}</XChip>
      <span v-if="dense" class="x-qr-card__enlarge">
        <XInlineButton icon="expand" :label="t('kit.enlarge')" tip @click="big = true" />
      </span>
    </div>
    <XQrCode :value="value" :copy="value" :copy-label="t('kit.copy')" :size="size" quiet />
    <XModal v-if="dense" v-model:open="big" :title="label" size="fit">
      <div class="x-qr-card__big">
        <XQrCode :value="value" :copy="value" :copy-label="t('kit.copy')" fill quiet />
      </div>
    </XModal>
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

/* The name stays centred over the code; the button sits at the frame's right. */
.x-qr-card__head {
  position: relative;
  display: flex;
  justify-content: center;
  align-self: stretch;
  min-width: 0;
}

.x-qr-card__head--dense {
  padding: 0 calc(var(--inline-btn) + var(--sp-6));
}

/* Large: as much of the screen as the window leaves, up to 720; the sheet's width on phones. */
.x-qr-card__big {
  width: min(720px, calc(100dvh - 200px), calc(100vw - 96px));
}

@media (max-width: 767px) {
  .x-qr-card__big {
    width: 100%;
  }
}

.x-qr-card__enlarge {
  position: absolute;
  display: flex;
  top: 50%;
  right: 0;
  transform: translateY(-50%);
}
</style>
