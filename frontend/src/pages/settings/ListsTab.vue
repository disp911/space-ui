<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../api'
import {
  toast, XButton, XCard, XChip, XConfirm, XCopyChip, XEmpty, XGridTable, XInlineButton, XInput, XSettingRow, XSettingsSection, XSpinner,
  type GridColumn, type MenuItem,
} from '../../kit'
import { cause, useSettingsForm } from './form'
import { listUrl, type ListsReply, type RuleList } from './lists'
import ListModal from './ListModal.vue'

/**
 * Lists of domains and subnets for a router's podkop: the panel keeps them
 * and serves each as a sing-box rule-set at <path><secret>/<name>.srs, the
 * address podkop takes in its remote domain and subnet lists. The path and
 * the secret are settings, saved with the page; a list is saved by its window.
 */
const { t } = useI18n()
const f = useSettingsForm()

const open = ref(true)
const keys = ['listsPath', 'listsSecret'] as const

/** A new secret of 24 letters and digits. */
function newSecret() {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  const bytes = crypto.getRandomValues(new Uint8Array(24))
  f.draft.listsSecret = Array.from(bytes, (b) => chars[b % chars.length]).join('')
}

// --- The lists ----------------------------------------------------------------

const base = ref('')
const lists = ref<RuleList[]>([])
const loading = ref(true)

async function load() {
  try {
    const reply = await api.get<ListsReply>('panel/api/lists')
    base.value = reply.base
    lists.value = reply.lists ?? []
  } catch (e) {
    toast('err', cause(e))
  } finally {
    loading.value = false
  }
}
onMounted(load)

/** The address a list is fetched at, with the path and secret as saved: the ones the server answers with. */
const url = (l: RuleList) => listUrl(base.value, f.saved.listsPath, f.saved.listsSecret, l.name)

const columns = computed<GridColumn[]>(() => [
  { key: 'name', label: t('settings.listName'), phone: 'head' },
  { key: 'kind', label: t('settings.listKind'), size: 'fit' },
  { key: 'count', label: t('settings.listCount'), size: 'fit' },
  { key: 'url', label: t('settings.listUrl'), size: '2fr' },
])

const menu = (): MenuItem[] => [
  { key: 'edit', label: t('settings.listEdit'), icon: 'edit' },
  { key: 'delete', label: t('settings.listDelete'), icon: 'delete', danger: true },
]

const editing = ref<RuleList | null>(null)
const windowOpen = ref(false)
const deleting = ref<RuleList | null>(null)
const deleteOpen = ref(false)
const deleteBusy = ref(false)

function create() {
  editing.value = null
  windowOpen.value = true
}

function select(key: string, row: RuleList) {
  if (key === 'edit') {
    editing.value = row
    windowOpen.value = true
  } else if (key === 'delete') {
    deleting.value = row
    deleteOpen.value = true
  }
}

function saved(list: RuleList) {
  const i = lists.value.findIndex((l) => l.id === list.id)
  if (i >= 0) lists.value[i] = list
  else lists.value.push(list)
}

async function remove() {
  if (!deleting.value) return
  deleteBusy.value = true
  try {
    await api.post(`panel/api/lists/delete/${deleting.value.id}`)
    lists.value = lists.value.filter((l) => l.id !== deleting.value!.id)
    deleteOpen.value = false
  } catch (e) {
    toast('err', cause(e))
  } finally {
    deleteBusy.value = false
  }
}
</script>

<template>
  <div class="settings-tab">
    <XSettingsSection v-model:open="open" :title="t('settings.secLists')" :count="keys.length" :modified="f.count([...keys]).dirty" :errors="f.count([...keys]).errors">
      <XSettingRow :title="t('settings.listsPath')" :description="t('settings.listsPathDesc')" :modified="f.isDirty('listsPath')" :error="f.error('listsPath')">
        <XInput v-model="f.draft.listsPath" mono placeholder="/lists/" :modified="f.isDirty('listsPath')" :invalid="!!f.error('listsPath')" />
      </XSettingRow>
      <XSettingRow :title="t('settings.listsSecret')" :description="t('settings.listsSecretDesc')" :modified="f.isDirty('listsSecret')" :error="f.error('listsSecret')">
        <XInput v-model.trim="f.draft.listsSecret" mono :modified="f.isDirty('listsSecret')" :invalid="!!f.error('listsSecret')">
          <template #actions><XInlineButton icon="refresh" :label="t('settings.listsNewSecret')" @click="newSecret" /></template>
        </XInput>
      </XSettingRow>
    </XSettingsSection>

    <div class="lists__tools">
      <XButton variant="primary" icon="plus" @click="create">{{ t('settings.listAdd') }}</XButton>
    </div>

    <XCard v-if="loading">
      <XEmpty :title="t('settings.listsLoading')"><XSpinner :size="16" /></XEmpty>
    </XCard>
    <XGridTable
      v-else-if="lists.length"
      :columns="columns"
      :rows="lists"
      :row-key="(r) => r.id"
      :menu="menu"
      :row-label="(i) => lists[i]?.name ?? ''"
      numbered
      center
      @select="select"
    >
      <template #cell-name="{ row }"><span class="lists__mono">{{ row.name }}</span></template>
      <template #cell-kind="{ row }">
        <XChip :tone="row.kind === 'domains' ? 'on' : 'neutral'">{{ row.kind === 'domains' ? t('settings.listDomains') : t('settings.listSubnets') }}</XChip>
      </template>
      <template #cell-count="{ row }">{{ row.count }}</template>
      <template #cell-url="{ row }">
        <span class="lists__url">
          <span class="lists__mono lists__link" :title="url(row)">{{ url(row) }}</span>
          <XCopyChip :text="url(row)" :label="t('settings.listCopy')" />
        </span>
      </template>
    </XGridTable>
    <XCard v-else>
      <XEmpty icon="rules" :title="t('settings.listsEmpty')" :hint="t('settings.listsEmptyHint')" tab />
    </XCard>

    <ListModal v-model:open="windowOpen" :list="editing" :taken="lists.map((l) => l.name)" @saved="saved" />
    <XConfirm
      v-model:open="deleteOpen"
      :title="t('settings.listDeleteTitle', { name: deleting?.name ?? '' })"
      :cta="t('settings.listDelete')"
      :busy="deleteBusy"
      icon="delete"
      danger
      @confirm="remove"
    >
      {{ t('settings.listDeleteText') }}
    </XConfirm>
  </div>
</template>

<style scoped>
.lists__tools {
  display: flex;
  gap: 8px;
  margin: 12px 0;
}

.lists__mono {
  font-family: var(--font-mono);
}

.lists__url {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  min-width: 0;
}

.lists__link {
  flex: 0 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--mut);
  font-size: var(--fs-12);
}
</style>
