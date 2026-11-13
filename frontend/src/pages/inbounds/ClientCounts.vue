<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { XChip, XCountList } from '../../kit'
import type { Tone } from '../../kit/types'
import type { ClientCounts } from './model'

/**
 * How many clients there are and how many of them are disabled, depleted,
 * nearly depleted or online, each state a chip that lists its emails.
 * labelled names the states, as the summary does.
 */
const props = defineProps<{ counts: ClientCounts; labelled?: boolean }>()
const { t } = useI18n()

const states = computed(() => {
  const c = props.counts
  const list = (emails: string[]) => emails.map((email) => ({ email, comment: c.comments.get(email) }))
  return [
    { key: 'off', tone: 'off' as Tone, label: t('inbounds.stDisabled'), items: list(c.deactive) },
    { key: 'err', tone: 'err' as Tone, label: t('inbounds.stDepleted'), items: list(c.depleted) },
    { key: 'warn', tone: 'warn' as Tone, label: t('inbounds.stExpiring'), items: list(c.expiring) },
    { key: 'on', tone: 'ok' as Tone, label: t('inbounds.stOnline'), items: list(c.online), dot: true },
  ].filter((s) => s.items.length)
})
</script>

<template>
  <div class="counts">
    <XChip tone="ok" icon="user" mono :title="t('inbounds.clients')">{{ counts.clients }}</XChip>
    <XCountList v-for="s in states" :key="s.key" :tone="s.tone" :label="s.label" :items="s.items" :dot="s.dot" :labelled="labelled" />
  </div>
</template>

<style scoped>
.counts {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  min-width: 0;
}
</style>
