<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../api'
import {
  checkJson, toast, XButton, XChip, XCodeArea, XCodeEditor, XField, XFieldGrid, XFieldNote, XInlineButton, XInput, XListHead, XMenu, XModal, XSection,
  XSegmented, XSwitch, type MenuItem,
} from '../../kit'
import { cause } from './form'
import {
  checkList, listNamePattern, loneSubnetSections, maxEntries, maxPorts, maxSectionName, maxSections, readPorts, readRawSections, type ListKind, type ListNetwork, type RawSection, type RuleList,
} from './lists'

/**
 * A router list: its file's name, whether it holds domains or subnets (kept
 * once made, routers take the two apart) and its sections, e.g. one a
 * service: a name, a switch that keeps a section out of the rule-set
 * without losing it, and its entries, one a line, which can be loaded from a
 * .txt or .lst file. A subnets section can be kept to one protocol and to
 * some ports, e.g. a service's voice calls; the sections kept to the same ones
 * make one rule, which a router cannot read when it is one subnet. The sections can also be edited as JSON, as the Xray
 * template is: JSON that reads as sections goes into them as it is typed.
 * The server checks it all again and builds the rule-set from the enabled
 * sections on saving.
 */
const props = defineProps<{ list: RuleList | null; taken: string[] }>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ saved: [list: RuleList] }>()
const { t } = useI18n()

interface Section {
  key: number
  name: string
  enabled: boolean
  text: string
  open: boolean
  network: ListNetwork
  /** The ports as typed, split by commas. */
  ports: string
}

const name = ref('')
const kind = ref<ListKind>('domains')
const sections = ref<Section[]>([])
const saving = ref(false)
const mode = ref<'sections' | 'json'>('sections')
let nextKey = 0

function section(name: string, text = '', enabled = true, isOpen = true, network: ListNetwork = '', ports: string[] = []): Section {
  return { key: nextKey++, name, enabled, text, open: isOpen, network, ports: ports.join(', ') }
}

watch(open, (on) => {
  if (!on) return
  mode.value = 'sections'
  name.value = props.list?.name ?? ''
  kind.value = props.list?.kind ?? 'domains'
  const saved = props.list?.sections ?? []
  // A list with one section opens it; with more, each stays closed until asked for.
  sections.value = saved.length
    ? saved.map((s) => section(s.name, s.entries, s.enabled, saved.length === 1, s.network ?? '', s.ports ?? []))
    : [section(t('settings.listSectionDefault', { n: 1 }))]
})

const modes = computed(() => [
  { value: 'sections' as const, label: t('settings.listSections') },
  { value: 'json' as const, label: 'JSON' },
])

const kinds = computed(() => [
  { value: 'domains' as const, label: t('settings.listDomains') },
  { value: 'subnets' as const, label: t('settings.listSubnets') },
])

const networks = computed(() => [
  { value: '' as const, label: t('settings.listNetworkAny') },
  { value: 'tcp' as const, label: 'TCP' },
  { value: 'udp' as const, label: 'UDP' },
])

const nameError = computed(() => {
  const n = name.value.trim()
  if (!n) return undefined
  if (!listNamePattern.test(n)) return t('settings.listNameWrong')
  if (n !== props.list?.name && props.taken.includes(n)) return t('settings.listNameTaken')
  return undefined
})

const checks = computed(() => sections.value.map((s) => checkList(kind.value, s.text)))
/** Each section's name error, as the server checks names. */
const sectionNameErrors = computed(() => {
  const seen = new Set<string>()
  return sections.value.map((s) => {
    const n = s.name.trim()
    if (!n) return t('settings.listSectionNameEmpty')
    if ([...n].length > maxSectionName) return t('settings.listSectionNameLong', { n: maxSectionName })
    const key = n.toLowerCase()
    if (seen.has(key)) return t('settings.listSectionNameTaken')
    seen.add(key)
    return undefined
  })
})
/** Each section's ports, and what is wrong in them; only a subnets list keeps them. */
const portChecks = computed(() => sections.value.map((s) => readPorts(kind.value === 'subnets' ? s.ports : '')))
const portErrors = computed(() =>
  portChecks.value.map((p) =>
    p.bad !== undefined ? t('settings.listPortsBad', { text: p.bad }) : p.tooMany ? t('settings.listPortsTooMany', { n: maxPorts }) : undefined,
  ),
)
/** Each section's error when its rule would be one subnet, which a router cannot read, so loading none of the list. */
const loneErrors = computed(() => {
  if (kind.value !== 'subnets') return sections.value.map(() => undefined)
  const lone = loneSubnetSections(
    sections.value.map((s, i) => ({ enabled: s.enabled, network: s.network, ports: portChecks.value[i]!.ports, entries: lines(s.text) })),
  )
  return lone.map((on, i) => {
    if (!on) return undefined
    return t(sections.value[i]!.network || portChecks.value[i]!.ports.length ? 'settings.listLoneSubnetFiltered' : 'settings.listLoneSubnet')
  })
})
/** What a section is kept to, on its chip: «UDP · 2 ports»; nothing when it takes every connection. */
const filterChip = (i: number) => {
  if (kind.value !== 'subnets') return undefined
  const n = portChecks.value[i]!.ports.length
  const parts = [sections.value[i]!.network.toUpperCase(), n ? t('settings.listFilterPorts', { n }, n) : ''].filter(Boolean)
  return parts.length ? parts.join(' · ') : undefined
}
const tooMany = computed(() => checks.value.reduce((n, c) => n + c.count, 0) > maxEntries)
const tooManySections = computed(() => sections.value.length > maxSections)
const canSave = computed(
  () =>
    !!name.value.trim() &&
    !nameError.value &&
    sectionNameErrors.value.every((e) => !e) &&
    portErrors.value.every((e) => !e) &&
    loneErrors.value.every((e) => !e) &&
    checks.value.every((c) => !c.bad) &&
    !tooMany.value &&
    !tooManySections.value &&
    !(mode.value === 'json' && (jsonShape.value || !!checkJson(json.value))) &&
    !saving.value,
)

// --- Sections ---------------------------------------------------------------------

function add() {
  const names = new Set(sections.value.map((s) => s.name.trim().toLowerCase()))
  let n = sections.value.length + 1
  while (names.has(t('settings.listSectionDefault', { n }).toLowerCase())) n++
  sections.value.push(section(t('settings.listSectionDefault', { n })))
}

const sectionMenu = (i: number): MenuItem[] => [
  { key: 'up', label: t('settings.listSectionUp'), icon: 'arrowUp', disabled: i === 0 },
  { key: 'down', label: t('settings.listSectionDown'), icon: 'arrowDown', disabled: i === sections.value.length - 1 },
  { key: 'delete', label: t('settings.listSectionDelete'), icon: 'delete', danger: true, divided: true },
]

function act(key: string, i: number) {
  const list = sections.value
  if (key === 'delete') list.splice(i, 1)
  else if (key === 'up' && i > 0) list.splice(i - 1, 2, list[i]!, list[i - 1]!)
  else if (key === 'down' && i < list.length - 1) list.splice(i, 2, list[i + 1]!, list[i]!)
}

// --- The sections as JSON ------------------------------------------------------------

const lines = (text: string) => text.split('\n').map((l) => l.trim()).filter(Boolean)
/** A section as JSON shows it: a subnets section's protocol and ports only when it is kept to them. */
const raw = (): RawSection[] =>
  sections.value.map((s, i) => {
    const ports = portChecks.value[i]!.bad === undefined ? portChecks.value[i]!.ports : s.ports.split(',').map((p) => p.trim()).filter(Boolean)
    return {
      name: s.name,
      enabled: s.enabled,
      ...(kind.value === 'subnets' && s.network ? { network: s.network } : {}),
      ...(kind.value === 'subnets' && ports.length ? { ports } : {}),
      entries: lines(s.text),
    }
  })
const json = ref('')
/** What keeps the typed JSON from being sections, when it parses. */
const jsonShape = ref(false)
watch(mode, (m) => {
  if (m !== 'json') return
  json.value = JSON.stringify(raw(), null, 2)
  jsonShape.value = false
})

function onJson(text: string) {
  json.value = text
  // The editor marks a mistake; the sections keep the last good JSON.
  if (checkJson(text)) return
  const read = readRawSections(JSON.parse(text))
  jsonShape.value = !read
  if (!read || JSON.stringify(read) === JSON.stringify(raw())) return
  // A section keeps its key and whether it is open while its name stays.
  const before = new Map(sections.value.map((s) => [s.name.trim().toLowerCase(), s]))
  sections.value = read.map((r) => {
    const was = before.get(r.name.trim().toLowerCase())
    return { ...section(r.name, r.entries.join('\n'), r.enabled, was?.open ?? false, r.network ?? '', r.ports ?? []), key: was?.key ?? nextKey++ }
  })
}

/** In JSON the first thing wrong in a section, named by its number and name. */
const jsonProblem = computed(() => {
  for (let i = 0; i < sections.value.length; i++) {
    const s = sections.value[i]!
    const what =
      sectionNameErrors.value[i] ??
      (checks.value[i]!.bad && t(kind.value === 'domains' ? 'settings.listJsonBadDomain' : 'settings.listJsonBadSubnet', { text: checks.value[i]!.bad!.text })) ??
      portErrors.value[i] ??
      loneErrors.value[i]
    if (what) return t('settings.listJsonSection', { n: i + 1, name: s.name.trim() || t('settings.listSectionUnnamed'), error: what })
  }
  return undefined
})

/** A section's IPv6 entries: how many, the first, and that a router may not load them. */
const v6Note = (i: number) => {
  const v6 = checks.value[i]!.v6!
  return t('settings.listIpv6', { n: v6.count, line: v6.line, text: v6.text })
}
/** In JSON the first section with IPv6 entries. */
const jsonV6 = computed(() => {
  const i = checks.value.findIndex((c) => c.v6)
  if (i < 0) return undefined
  return t('settings.listJsonSection', { n: i + 1, name: sections.value[i]!.name.trim() || t('settings.listSectionUnnamed'), error: v6Note(i) })
})

// --- Loading a file into a section -----------------------------------------------

const picker = ref<HTMLInputElement | null>(null)
let target = -1
function load(i: number) {
  target = i
  picker.value?.click()
}
async function pick(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  const sec = sections.value[target]
  if (!file || !sec) return
  try {
    sec.text = (await file.text()).replace(/\r\n?/g, '\n')
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
      sections: sections.value.map((s, i) => ({
        name: s.name.trim(),
        enabled: s.enabled,
        entries: s.text,
        ...(kind.value === 'subnets' ? { network: s.network, ports: portChecks.value[i]!.ports } : {}),
      })),
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
      <div class="list-window__mode">
        <XSegmented v-model="mode" :options="modes" block :label="t('settings.listSections')" />
      </div>
      <XFieldGrid :cols="2" top>
        <XField v-slot="a" :label="t('settings.listFileName')" :hint="nameError ? undefined : t('settings.listFileNameHint')" :error="nameError">
          <XInput v-bind="a" v-model="name" mono :invalid="!!nameError" />
        </XField>
        <XField :label="t('settings.listKind')">
          <XSegmented v-model="kind" :options="kinds" :disabled="!!list" />
        </XField>
      </XFieldGrid>

      <template v-if="mode === 'json'">
        <XCodeEditor :model-value="json" :height="420" :phone-height="360" label="JSON" @update:model-value="onJson" />
        <XFieldNote v-if="jsonShape" tone="err" attached>{{ t('settings.listJsonShape') }}</XFieldNote>
        <XFieldNote v-else-if="jsonProblem" tone="err" attached>{{ jsonProblem }}</XFieldNote>
        <XFieldNote v-else-if="jsonV6" tone="warn" attached>{{ jsonV6 }}</XFieldNote>
      </template>
      <XListHead v-else :title="t('settings.listSections')" :add="!tooManySections" @add="add">
        <div class="list-window__sections">
          <XSection v-for="(sec, i) in sections" :key="sec.key" v-model:open="sec.open" :title="sec.name.trim() || t('settings.listSectionUnnamed')" card>
            <template #meta>
              <XChip v-if="filterChip(i)" :tone="sec.enabled ? 'on' : 'off'">{{ filterChip(i) }}</XChip>
              <XChip mono :tone="sec.enabled ? 'neutral' : 'off'" :title="t('settings.listSectionEntries', { n: checks[i]!.count })">{{ checks[i]!.count }}</XChip>
              <XSwitch v-model="sec.enabled" :label="t('settings.listSectionOn')" />
              <XMenu v-slot="{ props: trigger }" :items="sectionMenu(i)" placement="bottom-end" @select="act($event, i)">
                <XInlineButton v-bind="trigger" icon="dots" :icon-size="16" :label="t('settings.listSectionMenu')" />
              </XMenu>
            </template>
            <div class="list-window__section">
              <XField v-slot="a" :label="t('settings.listSectionName')" :error="sectionNameErrors[i]">
                <XInput v-bind="a" v-model="sec.name" :invalid="!!sectionNameErrors[i]" />
              </XField>
              <XFieldGrid v-if="kind === 'subnets'" :cols="2" top>
                <XField :label="t('settings.listSectionNetwork')">
                  <XSegmented v-model="sec.network" :options="networks" block />
                </XField>
                <XField v-slot="a" :label="t('settings.listSectionPorts')" :error="portErrors[i]">
                  <XInput v-bind="a" v-model="sec.ports" mono :invalid="!!portErrors[i]" />
                </XField>
              </XFieldGrid>
              <XField
                :label="t('settings.listEntries')"
                :hint="t(kind === 'domains' ? 'settings.listEntriesDomains' : 'settings.listEntriesSubnets', { n: checks[i]!.count })"
                :action="t('settings.listFromFile')"
                action-icon="upload"
                @action="load(i)"
              >
                <XCodeArea v-model="sec.text" :height="240" :label="t('settings.listEntries')" />
              </XField>
              <XFieldNote v-if="checks[i]!.bad" tone="err" attached>
                {{ t(kind === 'domains' ? 'settings.listBadDomain' : 'settings.listBadSubnet', { line: checks[i]!.bad!.line, text: checks[i]!.bad!.text }) }}
              </XFieldNote>
              <XFieldNote v-else-if="loneErrors[i]" tone="err" attached>{{ loneErrors[i] }}</XFieldNote>
              <XFieldNote v-else-if="checks[i]!.v6" tone="warn" attached>{{ v6Note(i) }}</XFieldNote>
            </div>
          </XSection>
        </div>
      </XListHead>
      <XFieldNote v-if="tooMany" tone="err">{{ t('settings.listTooMany', { n: maxEntries }) }}</XFieldNote>
      <XFieldNote v-else-if="tooManySections" tone="err">{{ t('settings.listSectionsTooMany', { n: maxSections }) }}</XFieldNote>
      <input ref="picker" type="file" accept=".txt,.lst,text/plain" hidden @change="pick" />
    </div>

    <template #footer>
      <XButton variant="primary" :loading="saving" :disabled="!canSave" @click="save">{{ list ? t('settings.listSave') : t('settings.listCreate') }}</XButton>
    </template>
  </XModal>
</template>

<style scoped>
.list-window,
.list-window__sections,
.list-window__section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.list-window__sections {
  gap: 8px;
}

.list-window__mode {
  width: 220px;
  max-width: 100%;
}

@media (max-width: 767px) {
  .list-window__mode {
    width: 100%;
  }
}
</style>
