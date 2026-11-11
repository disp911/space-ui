<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast, XButton, XCodeEditor, XField, XFieldGrid, XInput, XModal, XNumber, XSegmented, XSelect } from '../../kit'
import type { Outbound } from './model'
import { canStream, canTls, defaultSettings, finalize, fromLink, hasAddressPort, newOutbound, protocols, server, setProtocolStream } from './outbound'
import OutboundSettings from './OutboundSettings.vue'
import OutboundStream from './OutboundStream.vue'
import OutboundSecurity from './OutboundSecurity.vue'
import OutboundExtra from './OutboundExtra.vue'

/**
 * An outbound to create, or the one at index to edit: a form of the parts
 * that go with its protocol, or its JSON with a share link to read. A tag
 * must be there and be its own. Its fields stand in one column, as in the
 * inbound window, and its body scrolls.
 */
const props = defineProps<{ outbound: Outbound | null; index: number; tags: string[] }>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ save: [outbound: Outbound] }>()
const { t } = useI18n()

const o = ref<Outbound>(newOutbound())
const mode = ref<'form' | 'json'>('form')

/** The outbound as the form first showed it, with the defaults the form fills in on sight. */
let shown = ''

watch(open, async (on) => {
  if (!on) return
  o.value = props.outbound ? JSON.parse(JSON.stringify(props.outbound)) : newOutbound()
  mode.value = 'form'
  link.value = ''
  await nextTick()
  shown = JSON.stringify(o.value)
})

const protocol = computed({
  get: () => o.value.protocol as string,
  set: (p: string) => {
    o.value.protocol = p
    o.value.settings = defaultSettings(p)
    setProtocolStream(o.value)
  },
})

const tagTaken = computed(() => props.tags.includes(String(o.value.tag ?? '').trim()))
const valid = computed(() => !!String(o.value.tag ?? '').trim() && !tagTaken.value)

const address = computed({ get: () => server(o.value).address ?? '', set: (v: string) => (server(o.value).address = v) })
const port = computed({ get: () => server(o.value).port ?? null, set: (v: number | null) => (server(o.value).port = v) })

function save() {
  if (!valid.value) return
  // Looked at but not changed: the outbound stays as it was, defaults and all.
  if (!props.outbound || JSON.stringify(o.value) !== shown) emit('save', finalize(o.value))
  open.value = false
}

// --- JSON ---------------------------------------------------------------------------

const json = ref('')
watch(mode, (m) => m === 'json' && (json.value = JSON.stringify(finalize(o.value), null, 2)))

function onJson(text: string) {
  json.value = text
  try {
    const parsed = JSON.parse(text)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) o.value = parsed
  } catch {
    // The editor marks what is wrong; the form keeps the last good outbound.
  }
}

/** The link stays in its field after an import; one that cannot be read is marked until it is edited. */
const link = ref('')
const linkWrong = ref(false)
watch(link, () => (linkWrong.value = false))
function importLink() {
  const read = fromLink(link.value.trim())
  if (!read) {
    linkWrong.value = true
    toast('err', t('xray.linkWrong'))
    return
  }
  o.value = read
  json.value = JSON.stringify(finalize(read), null, 2)
  toast('ok', t('xray.linkImported'))
}

const modes = computed(() => [
  { value: 'form' as const, label: 'Form' },
  { value: 'json' as const, label: 'JSON' },
])
const protocolOptions = protocols.map((p) => ({ value: p, label: p }))
const title = computed(() => (props.outbound ? t('xray.outEdit', { n: props.index + 1 }) : t('xray.outAdd')))
const linkHint = 'vmess:// vless:// trojan:// ss:// hysteria2://'
</script>

<template>
  <XModal v-model:open="open" :title="title" size="md" cancel scroll-body stack>
    <div class="outbound">
      <div class="outbound__mode">
        <XSegmented v-model="mode" :options="modes" block :label="title" />
      </div>

      <template v-if="mode === 'form'">
        <XFieldGrid :cols="3">
          <XField v-slot="a" :label="t('xray.protocol')">
            <XSelect v-bind="a" v-model="protocol" :options="protocolOptions" mono />
          </XField>
          <XField v-slot="a" :label="t('xray.outTag')">
            <XInput v-bind="a" v-model="o.tag" mono :invalid="tagTaken" />
          </XField>
          <XField v-slot="a" :label="t('xray.sendThrough')">
            <XInput v-bind="a" v-model="o.sendThrough" mono />
          </XField>
        </XFieldGrid>
        <XFieldGrid v-if="hasAddressPort(o)" columns="minmax(0, 1fr) 150px">
          <XField v-slot="a" :label="t('xray.outAddress')">
            <XInput v-bind="a" v-model="address" mono />
          </XField>
          <XField v-slot="a" :label="t('xray.port')">
            <XNumber v-bind="a" block v-model="port" :min="1" :max="65535" />
          </XField>
        </XFieldGrid>

        <OutboundSettings :o="o" :tags="tags" />
        <OutboundStream v-if="canStream(o)" :o="o" />
        <OutboundSecurity v-if="canTls(o)" :o="o" />
        <OutboundExtra :o="o" :tags="tags" />
      </template>

      <template v-else>
        <XField v-slot="a" :label="t('xray.link')">
          <div class="outbound__link">
            <XInput v-bind="a" v-model="link" mono :invalid="linkWrong" :placeholder="linkHint" @keydown.enter="importLink" />
            <XButton variant="primary" icon="download" :label="t('xray.linkImport')" :disabled="!link.trim()" @click="importLink" />
          </div>
        </XField>
        <XCodeEditor :model-value="json" :height="420" :phone-height="360" label="JSON" @update:model-value="onJson" />
      </template>
    </div>

    <template #footer>
      <XButton variant="primary" :disabled="!valid" @click="save">{{ outbound ? t('xray.save') : t('xray.outAdd') }}</XButton>
    </template>
  </XModal>
</template>

<style scoped>
.outbound {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.outbound__mode {
  width: 220px;
  max-width: 100%;
}

.outbound__link {
  display: flex;
  flex: 1;
  gap: 8px;
  min-width: 0;
}

.outbound__link > :first-child {
  flex: 1;
  min-width: 0;
}

@media (max-width: 767px) {
  .outbound__mode {
    width: 100%;
  }
}
</style>
