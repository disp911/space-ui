<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { XButton, XButtonGroup, XInput, XMenu, XPopover, XSegmented, XSelect, XSwitch, type MenuItem } from '../../kit'
import { useInbounds } from '../../stores/inbounds'
import type { Filter } from './model'

/**
 * The tools over the inbounds: create one, general actions, search or a
 * filter by client state, and refreshing by hand or on its own.
 */
const props = defineProps<{ subscription: boolean }>()
const mode = defineModel<'search' | 'filter'>('mode', { required: true })
const search = defineModel<string>('search', { required: true })
const filter = defineModel<Filter>('filter', { required: true })
const emit = defineEmits<{ create: []; action: [key: string] }>()
const { t } = useI18n()
const store = useInbounds()

const general = computed<MenuItem[]>(() => [
  { key: 'import', label: t('inbounds.importInbound'), icon: 'download' },
  { key: 'exportLinks', label: t('inbounds.exportLinks'), icon: 'link' },
  ...(props.subscription ? [{ key: 'exportSubs', label: t('inbounds.exportSubs'), icon: 'link' as const }] : []),
  { key: 'resetAllInbounds', label: t('inbounds.resetAllInbounds'), icon: 'reset', divided: true },
  { key: 'resetAllClients', label: t('inbounds.resetAllClients'), icon: 'reset' },
  { key: 'delDepleted', label: t('inbounds.delDepleted'), icon: 'userX', danger: true, divided: true },
])

const modes = computed(() => [
  { value: 'search' as const, label: t('inbounds.search'), icon: 'search' as const },
  { value: 'filter' as const, label: t('inbounds.filter'), icon: 'filter' as const },
])

const filters = computed<{ value: Filter; label: string }[]>(() => [
  { value: 'none', label: t('inbounds.stNone') },
  { value: 'active', label: t('inbounds.stActive') },
  { value: 'disabled', label: t('inbounds.stDisabled') },
  { value: 'depleted', label: t('inbounds.stDepleted') },
  { value: 'expiring', label: t('inbounds.stExpiring') },
  { value: 'online', label: t('inbounds.stOnline') },
])

const intervals = [5, 10, 30, 60].map((s) => ({ value: s * 1000, label: `${s}s` }))

// A mode switched clears the other one's choice, as on the old panel.
function setMode(m: 'search' | 'filter') {
  mode.value = m
  if (m === 'search') filter.value = 'none'
  else search.value = ''
}
</script>

<template>
  <div class="tools">
    <XButton variant="primary" icon="plus" phone-icon-only @click="emit('create')">{{ t('inbounds.create') }}</XButton>
    <XMenu v-slot="{ props: trigger }" :items="general" placement="bottom-start" @select="emit('action', $event)">
      <XButton v-bind="trigger" icon="menu" phone-icon-only>{{ t('inbounds.general') }}</XButton>
    </XMenu>

    <div class="tools__find">
      <XSegmented :model-value="mode" :options="modes" icon-only :label="t('inbounds.mode')" @update:model-value="setMode" />
      <XInput v-if="mode === 'search'" v-model="search" type="search" icon="search" clearable :placeholder="t('inbounds.search')" :aria-label="t('inbounds.search')" class="tools__search" />
      <XSegmented v-else v-model="filter" :options="filters" scroll :label="t('inbounds.filter')" class="tools__filter" />
    </div>

    <XButtonGroup :label="t('inbounds.autoRefresh')" class="tools__refresh">
      <XButton icon="refresh" :label="t('inbounds.refresh')" :loading="store.refreshing" @click="store.load().catch(() => {})" />
      <XPopover :label="t('inbounds.autoRefresh')">
        <template #default="{ props: trigger }">
          <XButton v-bind="trigger" icon="caret" :label="t('inbounds.autoRefresh')" :dot="store.autoRefresh" />
        </template>
        <template #content>
          <div class="tools__auto">
            <div class="tools__auto-head">
              <span class="tools__auto-title">{{ t('inbounds.autoRefresh') }}</span>
              <XSwitch :model-value="store.autoRefresh" :label="t('inbounds.autoRefresh')" @update:model-value="store.setAutoRefresh" />
            </div>
            <div class="tools__auto-row">
              <span :class="['tools__auto-label', { 'is-off': !store.autoRefresh }]">{{ t('inbounds.interval') }}</span>
              <XSelect
                :model-value="store.interval"
                :options="intervals"
                :disabled="!store.autoRefresh"
                mono
                class="tools__interval"
                :aria-label="t('inbounds.interval')"
                @update:model-value="store.setRefreshInterval(Number($event))"
              />
            </div>
          </div>
        </template>
      </XPopover>
    </XButtonGroup>
  </div>
</template>

<style scoped>
.tools {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px 8px;
}

.tools__find {
  display: flex;
  flex: 1 1 0;
  align-items: center;
  gap: 8px;
  min-width: 0;
  margin-left: 4px;
}

.tools__search {
  flex: 1;
  max-width: 300px;
}

.tools__filter {
  min-width: 0;
}

.tools__refresh {
  margin-left: auto;
}

.tools__auto {
  display: flex;
  flex-direction: column;
  width: 224px;
}

.tools__auto-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--div);
}

.tools__auto-title {
  color: var(--fg);
  font: 600 var(--fs-13) / var(--lh-tight) var(--font-sans);
}

.tools__auto-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.tools__auto-label {
  color: var(--fg2);
  font: 500 var(--fs-13) / var(--lh-tight) var(--font-sans);
}

.tools__auto-label.is-off {
  color: var(--faint);
}

.tools__interval {
  width: 92px;
}

@media (max-width: 767px) {
  .tools__find {
    flex: 1 1 100%;
    order: 1;
    margin-left: 0;
  }

  .tools__search {
    max-width: none;
  }

  .tools__interval {
    width: 100px;
  }
}
</style>
