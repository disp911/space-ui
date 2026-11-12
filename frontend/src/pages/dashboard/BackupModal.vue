<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, ApiError, saveFile } from '../../api'
import { toast, XActionList, XActionRow, XConfirm, XModal } from '../../kit'

/** Export and import of the panel database. */
const open = defineModel<boolean>('open', { default: false })
const { t } = useI18n()

const failed = (e: unknown) => toast('err', t('dashboard.actionFailed', { error: e instanceof ApiError ? e.message : String(e) }))

const exporting = ref(false)

async function exportDb() {
  exporting.value = true
  try {
    const { data, name } = await api.file('panel/api/server/getDb')
    saveFile(data, name ?? 'space-ui.db')
  } catch (e) {
    failed(e)
  } finally {
    exporting.value = false
  }
}

// Import: pick a file, confirm, upload, then restart the panel and reload as it comes back.
const picker = ref<HTMLInputElement | null>(null)
const file = ref<File | null>(null)
const confirming = ref(false)
const restoring = ref(false)

function pick(e: Event) {
  const input = e.target as HTMLInputElement
  file.value = input.files?.[0] ?? null
  input.value = ''
  if (file.value) confirming.value = true
}

async function restore() {
  if (!file.value) return
  restoring.value = true
  try {
    const body = new FormData()
    body.set('db', file.value)
    await api.post('panel/api/server/importDB', body)
    await api.post('panel/setting/restartPanel')
    setTimeout(() => location.reload(), 5000)
  } catch (e) {
    failed(e)
    restoring.value = false
    confirming.value = false
  }
}
</script>

<template>
  <XModal v-model:open="open" :title="t('dashboard.backup')" size="sm">
    <XActionList>
      <XActionRow :title="t('dashboard.backupExport')" icon="download" :loading="exporting" @click="exportDb">
        <i18n-t scope="global" keypath="dashboard.backupExportText" tag="span"><template #ext><code>.db</code></template></i18n-t>
      </XActionRow>
      <XActionRow :title="t('dashboard.backupImport')" icon="upload" :loading="restoring" @click="picker?.click()">
        <i18n-t scope="global" keypath="dashboard.backupImportText" tag="span"><template #ext><code>.db</code></template></i18n-t>
      </XActionRow>
    </XActionList>
    <input ref="picker" type="file" accept=".db" hidden @change="pick" />
  </XModal>

  <XConfirm
    v-model:open="confirming"
    :title="t('dashboard.backupRestoreTitle')"
    :cta="t('dashboard.backupRestore')"
    :busy="restoring"
    :busy-label="t('dashboard.backupRestoring')"
    @confirm="restore"
  >
    {{ t('dashboard.backupRestoreText', { name: file?.name ?? '' }) }}
  </XConfirm>
</template>
