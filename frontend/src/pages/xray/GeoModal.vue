<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../api'
import { formatTimestamp } from '../../i18n/format'
import {
  toast, usePhone, XButton, XChip, XConfirm, XFieldNote, XInlineButton, XModal, XPlainTable, XSection, XSpinner, type PlainColumn,
} from '../../kit'
import { cause } from '../settings/form'
import GeoSourceModal from './GeoSourceModal.vue'
import type { GeoSource } from './model'

/**
 * Geo files: the standard ones Xray ships with, updated one by one or all at
 * once, and the user's own sources. Everything here is done on the server at
 * once, apart from the template and its Save.
 */
const open = defineModel<boolean>('open', { default: false })
const { t } = useI18n()
const phone = usePhone()

const standard = ['geosite.dat', 'geoip.dat', 'geosite_RU.dat', 'geoip_RU.dat'].map((file) => ({ file }))
const fileColumns: PlainColumn[] = [{ key: 'file' }, { key: 'update', fit: true }]
const openStandard = ref(true)
const openCustom = ref(true)

// --- Standard files -------------------------------------------------------------

/** The file waiting for a yes, '' for all of them. */
const asking = ref<string | null>(null)
/** The files being updated, '' for all of them; each row goes on by itself, as in the mockup. */
const updating = reactive(new Set<string>())
const updatingFile = (file: string) => updating.has(file) || updating.has('')

async function update() {
  const file = asking.value
  asking.value = null
  if (file === null) return
  updating.add(file)
  try {
    await api.post(file ? `panel/api/server/updateGeofile/${file}` : 'panel/api/server/updateGeofile')
    toast('ok', t('xray.geoUpdated'))
  } catch (e) {
    toast('err', cause(e))
  } finally {
    updating.delete(file)
  }
}

// --- The user's sources ---------------------------------------------------------

const sources = ref<GeoSource[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    sources.value = (await api.get<GeoSource[] | null>('panel/api/custom-geo/list')) ?? []
  } catch (e) {
    toast('err', cause(e))
  } finally {
    loading.value = false
  }
}
watch(open, (on) => on && load())

const columns = computed<PlainColumn[]>(() => [
  { key: 'ext', label: t('xray.geoExt') },
  { key: 'updated', label: t('xray.geoLastUpdated') },
  { key: 'actions', label: t('xray.geoActions'), fit: true },
])
const ext = (s: GeoSource) => `ext:${s.type}_${s.alias}.dat:tag`

const editing = ref<GeoSource | null>(null)
const formOpen = ref(false)
function edit(s: GeoSource | null) {
  editing.value = s
  formOpen.value = true
}

/** The sources being downloaded again, 'all' for all of them. */
const pulling = reactive(new Set<number | 'all'>())
const pullingSource = (s: GeoSource) => pulling.has(s.id) || pulling.has('all')

async function pull(id: number | 'all') {
  pulling.add(id)
  try {
    await api.post(id === 'all' ? 'panel/api/custom-geo/update-all' : `panel/api/custom-geo/download/${id}`)
    if (id === 'all') toast('ok', t('xray.geoUpdatedAll'))
  } catch (e) {
    toast('err', cause(e))
  } finally {
    pulling.delete(id)
    load()
  }
}

const deleting = ref<GeoSource | null>(null)
const deleteOpen = computed({ get: () => !!deleting.value, set: (on) => !on && (deleting.value = null) })

async function remove() {
  const s = deleting.value
  deleting.value = null
  if (!s) return
  try {
    await api.post(`panel/api/custom-geo/delete/${s.id}`)
  } catch (e) {
    toast('err', cause(e))
  }
  load()
}
</script>

<template>
  <XModal v-model:open="open" :title="t('xray.geoFiles')" size="xl">
    <div class="geo">
      <XSection v-model:open="openStandard" :title="t('xray.geoStandard')" card>
        <div class="geo__block">
          <XPlainTable :columns="fileColumns" :rows="standard" :row-key="(r) => r.file" :active="(r) => updatingFile(r.file)">
            <template #cell-file="{ row }">
              <XChip :tone="row.file.startsWith('geoip') ? 'ok' : 'unl'" mono>{{ row.file }}</XChip>
            </template>
            <template #cell-update="{ row }">
              <XInlineButton icon="refresh" :label="t('xray.geoDownload')" :loading="updatingFile(row.file)" @click="asking = row.file" />
            </template>
          </XPlainTable>
          <div class="geo__end">
            <XButton size="sm" icon="refresh" :loading="updating.has('')" @click="asking = ''">
              {{ t('xray.geoUpdateAll') }}
            </XButton>
          </div>
        </div>
      </XSection>

      <XSection v-model:open="openCustom" :title="t('xray.geoCustom')" card>
        <div class="geo__block">
          <XFieldNote tone="info">{{ t('xray.geoHint') }}</XFieldNote>
          <div class="geo__tools">
            <XButton variant="primary" size="sm" icon="plus" @click="edit(null)">{{ t('xray.geoAdd') }}</XButton>
            <XButton size="sm" icon="refresh" :loading="pulling.has('all')" :disabled="!sources.length" @click="pull('all')">
              {{ t('xray.geoUpdateAll') }}
            </XButton>
          </div>
          <XPlainTable :columns="columns" :rows="sources" :row-key="(s) => s.id" :active="pullingSource">
            <template #cell-ext="{ row }">
              <span v-if="phone" class="geo__name">{{ ext(row) }}</span>
              <code v-else class="geo__ext">{{ ext(row) }}</code>
            </template>
            <template #cell-updated="{ row }">
              <span class="geo__time">{{ row.lastUpdatedAt ? formatTimestamp(row.lastUpdatedAt) : '—' }}</span>
            </template>
            <template #cell-actions="{ row }">
              <XInlineButton icon="edit" :label="t('xray.geoEdit')" @click="edit(row)" />
              <XInlineButton icon="refresh" :label="t('xray.geoDownload')" :loading="pullingSource(row)" @click="pull(row.id)" />
              <XInlineButton icon="delete" :label="t('xray.delete')" danger @click="deleting = row" />
            </template>
            <template v-if="loading" #empty><XSpinner :size="16" /></template>
          </XPlainTable>
        </div>
      </XSection>
    </div>
  </XModal>

  <GeoSourceModal v-model:open="formOpen" :source="editing" @saved="load" />

  <XConfirm :open="asking !== null" icon="refresh" :title="t('xray.geoUpdateTitle')" :cta="t('xray.confirm')" @update:open="!$event && (asking = null)" @confirm="update">
    {{ asking ? t('xray.geoUpdateFile', { file: asking }) : t('xray.geoUpdateAllText') }}
  </XConfirm>
  <XConfirm v-model:open="deleteOpen" icon="delete" :title="t('xray.delete')" :cta="t('xray.confirm')" @confirm="remove">
    {{ t('xray.geoDeleteText') }}
  </XConfirm>
</template>

<style scoped>
.geo {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.geo__block {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}

.geo__end {
  display: flex;
  justify-content: flex-end;
}

.geo__tools {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.geo__ext {
  padding: 2px 6px;
  border-radius: var(--r-6);
  background: var(--hov);
  color: var(--fg2);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-mono);
  white-space: nowrap;
}

/* Phones: the name heads its card, whole and plain. */
.geo__name {
  color: var(--fg);
  font: 400 var(--fs-12) / var(--lh-text) var(--font-mono);
  word-break: break-all;
}

.geo__time {
  color: var(--mut);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-mono);
  white-space: nowrap;
}

/* Phones: the time goes under the date rather than under the buttons. */
@media (max-width: 767px) {
  .geo__time {
    white-space: normal;
  }
}
</style>
