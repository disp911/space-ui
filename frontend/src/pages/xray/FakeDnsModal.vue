<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { XButton, XField, XFieldGrid, XInput, XModal, XNumber } from '../../kit'
import { defaultFakeDns } from './model'

/** A Fake DNS pool to create, or the one at index to edit: the subnet it hands addresses out of and how many. */
const props = defineProps<{ pool: Record<string, any> | null; index: number }>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ save: [pool: Record<string, any>] }>()
const { t } = useI18n()

const form = reactive({ ipPool: '', poolSize: 65535 as number | null })
/** The form as it opened: confirmed unchanged, the pool stays as it was. */
let shown = ''

watch(open, (on) => {
  if (!on) return
  const p = props.pool ?? defaultFakeDns()
  Object.assign(form, { ipPool: String(p.ipPool ?? ''), poolSize: typeof p.poolSize === 'number' ? p.poolSize : null })
  shown = JSON.stringify(form)
})

function save() {
  const same = JSON.stringify({ ...form, ipPool: form.ipPool.trim() }) === shown
  open.value = false
  if (props.pool && same) return
  emit('save', { ...(props.pool ?? {}), ipPool: form.ipPool.trim(), poolSize: form.poolSize })
}

const title = computed(() => (props.pool ? t('xray.fakeEditN', { n: props.index + 1 }) : t('xray.fakeAdd')))
</script>

<template>
  <XModal v-model:open="open" :title="title" size="md" cancel scroll-body stack>
    <XFieldGrid>
      <XField v-slot="a" :label="t('xray.fakePool')">
        <XInput v-bind="a" v-model="form.ipPool" mono />
      </XField>
      <XField v-slot="a" :label="t('xray.fakePoolSize')">
        <XNumber v-bind="a" block v-model="form.poolSize" :min="1" />
      </XField>
    </XFieldGrid>

    <template #footer>
      <XButton variant="primary" @click="save">{{ t('xray.confirm') }}</XButton>
    </template>
  </XModal>
</template>
