<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../api'
import { toast, XButton, XCodeArea, XField, XFieldGrid, XFieldNote, XInput, XModal, XSegmented } from '../../kit'
import { cause } from './form'
import { checkList, listNamePattern, maxEntries, type ListKind, type RuleList } from './lists'

/**
 * A podkop list: its file's name, whether it holds domains or subnets (kept
 * once made, podkop takes the two apart) and its entries, one a line. The
 * entries can be loaded from a .txt or .lst file. The server checks it all
 * again and builds the rule-set on saving.
 */
const props = defineProps<{ list: RuleList | null; taken: string[] }>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ saved: [list: RuleList] }>()
const { t } = useI18n()

const name = ref('')
const kind = ref<ListKind>('domains')
const text = ref('')
const saving = ref(false)

watch(open, (on) => {
  if (!on) return
  name.value = props.list?.name ?? ''
  kind.value = props.list?.kind ?? 'domains'
  text.value = props.list?.entries ?? ''
})

const kinds = computed(() => [
  { value: 'domains' as const, label: t('settings.listDomains') },
  { value: 'subnets' as const, label: t('settings.listSubnets') },
])

const nameError = computed(() => {
  const n = name.value.trim()
  if (!n) return undefined
  if (!listNamePattern.test(n)) return t('settings.listNameWrong')
  if (n !== props.list?.name && props.taken.includes(n)) return t('settings.listNameTaken')
  return undefined
})
const check = computed(() => checkList(kind.value, text.value))
const tooMany = computed(() => check.value.count > maxEntries)
const canSave = computed(() => !!name.value.trim() && !nameError.value && !check.value.bad && !tooMany.value && !saving.value)

// --- Loading a file -------------------------------------------------------------

const picker = ref<HTMLInputElement | null>(null)
async function pick(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  try {
    text.value = (await file.text()).replace(/\r\n?/g, '\n')
  } catch (err) {
    toast('err', cause(err))
  }
}

async function save() {
  if (!canSave.value) return
  saving.value = true
  try {
    const list = await api.post<RuleList>('panel/api/lists/save', {
      id: props.list?.id ?? 0,
      name: name.value.trim(),
      kind: kind.value,
      entries: text.value,
    })
    emit('saved', list)
    open.value = false
  } catch (e) {
    toast('err', cause(e))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <XModal v-model:open="open" :title="list ? t('settings.listEditTitle', { name: list.name }) : t('settings.listNew')" size="lg" full cancel :dismissible="!saving">
    <div class="list-window">
      <XFieldGrid :cols="2" top>
        <XField v-slot="a" :label="t('settings.listFileName')" :hint="nameError ? undefined : t('settings.listFileNameHint', { name: name.trim() || 'name' })" :error="nameError">
          <XInput v-bind="a" v-model="name" mono placeholder="youtube" :invalid="!!nameError" />
        </XField>
        <XField :label="t('settings.listKind')">
          <XSegmented v-model="kind" :options="kinds" :disabled="!!list" />
        </XField>
      </XFieldGrid>
      <XField
        :label="t('settings.listEntries')"
        :hint="t(kind === 'domains' ? 'settings.listEntriesDomains' : 'settings.listEntriesSubnets', { n: check.count })"
        :action="t('settings.listFromFile')"
        action-icon="upload"
        fill
        @action="picker?.click()"
      >
        <XCodeArea v-model="text" :height="420" phone-fill :label="t('settings.listEntries')" />
      </XField>
      <XFieldNote v-if="check.bad" tone="err" attached>
        {{ t(kind === 'domains' ? 'settings.listBadDomain' : 'settings.listBadSubnet', { line: check.bad.line, text: check.bad.text }) }}
      </XFieldNote>
      <XFieldNote v-else-if="tooMany" tone="err" attached>{{ t('settings.listTooMany', { n: maxEntries }) }}</XFieldNote>
      <input ref="picker" type="file" accept=".txt,.lst,text/plain" hidden @change="pick" />
    </div>

    <template #footer>
      <XButton variant="primary" :loading="saving" :disabled="!canSave" @click="save">{{ list ? t('settings.listSave') : t('settings.listCreate') }}</XButton>
    </template>
  </XModal>
</template>

<style scoped>
.list-window {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* On phones the window fills the screen and the entries take the room left. */
@media (max-width: 767px) {
  .list-window {
    flex: 1 1 auto;
    min-height: 0;
  }
}
</style>
