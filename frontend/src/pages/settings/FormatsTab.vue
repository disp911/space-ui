<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  XButton, XField, XInput, XNest, XNestCard, XNumber, XSelect, XSettingRow, XSettingsSection, XSwitch, XTagInput, XTextarea,
} from '../../kit'
import { useSettingsForm } from './form'
import { checkValue, muxCheck, type AllSetting, type SettingKey } from './model'

/**
 * What JSON and Clash subscriptions add to a client's config. Fragmentation,
 * noises, mux and direct rules are kept as JSON in their settings, an empty
 * one meaning off; the defaults and the shape follow the old settings page,
 * except that direct connection starts with Russia instead of Iran.
 */
const { t } = useI18n()
const f = useSettingsForm()

const open = reactive({ general: true, fragment: true, noises: true, mux: true, direct: true })

const general = computed(() => [
  ...(f.draft.subJsonEnable ? (['subJsonPath', 'subJsonURI'] as const) : []),
  ...(f.draft.subClashEnable ? (['subClashPath', 'subClashURI'] as const) : []),
] as SettingKey[])

/** A setting that holds JSON, as a value: null when empty. */
function json<T>(key: keyof AllSetting & `subJson${string}`) {
  return computed<T | null>({
    get: () => {
      const s = f.draft[key] as string
      if (!s) return null
      try {
        return JSON.parse(s) as T
      } catch {
        return null
      }
    },
    set: (v) => ((f.draft[key] as string) = v == null ? '' : JSON.stringify(v)),
  })
}
const savedJson = <T,>(key: keyof AllSetting) => {
  try {
    return f.saved[key] ? (JSON.parse(f.saved[key] as string) as T) : null
  } catch {
    return null
  }
}

// --- Fragmentation ------------------------------------------------------------

interface Fragment { packets: string; length: string; interval: string; maxSplit: string }
const fragment = json<Fragment>('subJsonFragment')
const fragmentFields = [
  { key: 'packets', label: 'Packets', placeholder: '1-1 | 1-3 | tlshello | ...' },
  { key: 'length', label: 'Length', placeholder: '100-200' },
  { key: 'interval', label: 'Interval', placeholder: '10-20' },
  { key: 'maxSplit', label: 'MaxSplit', placeholder: '300-400' },
] as const
function setFragment(on: boolean) {
  fragment.value = on ? { packets: 'tlshello', length: '100-200', interval: '10-20', maxSplit: '300-400' } : null
}

// --- Noises -------------------------------------------------------------------

// The subscription sends them as finalmask's noise, which has no applyTo:
// one saved before loses it when it is changed.
interface Noise { type: string; packet: string; delay: string; applyTo?: string }
const noises = json<Noise[]>('subJsonNoises')
const newNoise = (): Noise => ({ type: 'rand', packet: '10-20', delay: '10-16' })
const noiseTypes = ['rand', 'base64', 'str', 'hex'].map((v) => ({ value: v, label: v }))
function setNoise(i: number, patch: Partial<Noise>) {
  noises.value = (noises.value ?? []).map((n, j) => {
    if (j !== i) return n
    const { applyTo: _, ...rest } = { ...n, ...patch }
    return rest
  })
}
const noiseChanged = (i: number) => JSON.stringify(noises.value?.[i]) !== JSON.stringify(savedJson<Noise[]>('subJsonNoises')?.[i])

// --- Mux ----------------------------------------------------------------------

interface Mux { enabled: boolean; concurrency: number | null; xudpConcurrency: number | null; xudpProxyUDP443: string }
const mux = json<Mux>('subJsonMux')
const udp443 = ['reject', 'allow', 'skip'].map((v) => ({ value: v, label: v }))
function setMux(patch: Partial<Mux>) {
  if (mux.value) mux.value = { ...mux.value, ...patch }
}
function muxError(v: unknown) {
  const fail = checkValue(muxCheck, v)
  return fail ? t(fail[0], fail[1] ?? {}) : undefined
}

// --- Direct connection ----------------------------------------------------------

interface Rule { type: string; outboundTag: string; domain?: string[]; ip?: string[] }
const rules = json<Rule[]>('subJsonRules')
const defaultRules = (): Rule[] => [
  { type: 'field', outboundTag: 'direct', domain: ['geosite:category-ru'] },
  { type: 'field', outboundTag: 'direct', ip: ['geoip:private', 'geoip:ru'] },
]
const ipPresets = [
  { value: 'geoip:private', label: 'Private IP' },
  { value: 'geoip:ir', label: '🇮🇷 Iran' },
  { value: 'geoip:cn', label: '🇨🇳 China' },
  { value: 'geoip:ru', label: '🇷🇺 Russia' },
  { value: 'geoip:vn', label: '🇻🇳 Vietnam' },
  { value: 'geoip:es', label: '🇪🇸 Spain' },
  { value: 'geoip:id', label: '🇮🇩 Indonesia' },
  { value: 'geoip:ua', label: '🇺🇦 Ukraine' },
  { value: 'geoip:tr', label: '🇹🇷 Türkiye' },
  { value: 'geoip:br', label: '🇧🇷 Brazil' },
]
const domainPresets = [
  { value: 'geosite:private', label: 'Private DNS' },
  { value: 'geosite:category-ir', label: '🇮🇷 Iran' },
  { value: 'geosite:cn', label: '🇨🇳 China' },
  { value: 'geosite:category-ru', label: '🇷🇺 Russia' },
  { value: 'geosite:apple', label: 'Apple' },
  { value: 'geosite:meta', label: 'Meta' },
  { value: 'geosite:google', label: 'Google' },
]

/** The values of the first rule with this field; setting them adds the rule or drops it when empty. */
function directList(field: 'ip' | 'domain') {
  return computed<string[]>({
    get: () => (Array.isArray(rules.value) ? (rules.value.find((r) => r[field])?.[field] ?? []) : []),
    set: (list) => {
      if (!Array.isArray(rules.value)) return
      let next = rules.value.map((r) => ({ ...r }))
      if (!list.length) next = next.filter((r) => !r[field])
      else {
        const i = next.findIndex((r) => r[field])
        if (i >= 0) next[i] = { ...next[i]!, [field]: list }
        else next.push({ type: 'field', outboundTag: 'direct', [field]: list })
      }
      rules.value = next
    },
  })
}
const directIps = directList('ip')
const directDomains = directList('domain')
</script>

<template>
  <div class="settings-tab">
    <XSettingsSection v-model:open="open.general" :title="t('settings.secGeneral')" :count="general.length" :modified="f.count(general).dirty" :errors="f.count(general).errors">
      <template v-if="f.draft.subJsonEnable">
        <XSettingRow :title="t('settings.subJsonPath')" :modified="f.isDirty('subJsonPath')" :error="f.error('subJsonPath')">
          <XInput v-model="f.draft.subJsonPath" mono placeholder="/json/" :modified="f.isDirty('subJsonPath')" :invalid="!!f.error('subJsonPath')" />
        </XSettingRow>
        <XSettingRow :title="t('settings.subJsonUri')" tall :modified="f.isDirty('subJsonURI')">
          <XTextarea v-model="f.draft.subJsonURI" mono :grow="3" placeholder="(http|https)://domain[:port]/path/" :modified="f.isDirty('subJsonURI')" />
        </XSettingRow>
      </template>
      <template v-if="f.draft.subClashEnable">
        <XSettingRow :title="t('settings.subClashPath')" :modified="f.isDirty('subClashPath')" :error="f.error('subClashPath')">
          <XInput v-model="f.draft.subClashPath" mono placeholder="/clash/" :modified="f.isDirty('subClashPath')" :invalid="!!f.error('subClashPath')" />
        </XSettingRow>
        <XSettingRow :title="t('settings.subClashUri')" tall :modified="f.isDirty('subClashURI')">
          <XTextarea v-model="f.draft.subClashURI" mono :grow="3" placeholder="(http|https)://domain[:port]/path/" :modified="f.isDirty('subClashURI')" />
        </XSettingRow>
      </template>
    </XSettingsSection>

    <XSettingsSection v-model:open="open.fragment" :title="t('settings.fragment')" :count="1" :modified="f.count(['subJsonFragment']).dirty">
      <XSettingRow :title="t('settings.fragment')" :description="t('settings.fragmentDesc')" inline :modified="f.isDirty('subJsonFragment')">
        <XSwitch :model-value="!!fragment" :label="t('settings.fragment')" :modified="f.isDirty('subJsonFragment')" @update:model-value="setFragment" />
        <template v-if="fragment" #nested>
          <XNest :title="t('settings.fragmentNest')" :columns="140">
            <XField v-for="field in fragmentFields" :key="field.key" v-slot="a" :label="field.label">
              <XInput
                v-bind="a"
                :model-value="fragment[field.key]"
                mono
                :placeholder="field.placeholder"
                @update:model-value="fragment = { ...fragment!, [field.key]: $event }"
              />
            </XField>
          </XNest>
        </template>
      </XSettingRow>
    </XSettingsSection>

    <XSettingsSection v-model:open="open.noises" :title="t('settings.noises')" :count="1" :modified="f.count(['subJsonNoises']).dirty">
      <XSettingRow :title="t('settings.noises')" :description="t('settings.noisesDesc')" inline :modified="f.isDirty('subJsonNoises')">
        <XSwitch :model-value="!!noises" :label="t('settings.noises')" :modified="f.isDirty('subJsonNoises')" @update:model-value="noises = $event ? [newNoise()] : null" />
        <template v-if="noises" #nested>
          <XNest :title="t('settings.noises')">
            <div class="noises">
              <XNestCard v-for="(n, i) in noises" :key="i" :title="t('settings.noise', { n: i + 1 })" :modified="noiseChanged(i)" :columns="130">
                <template v-if="noises.length > 1" #actions>
                  <XButton variant="ghost-danger" size="sm" icon="delete" @click="noises = noises!.filter((_, j) => j !== i)">{{ t('settings.remove') }}</XButton>
                </template>
                <XField v-slot="a" label="Type">
                  <XSelect v-bind="a" :model-value="n.type" :options="noiseTypes" mono @update:model-value="setNoise(i, { type: $event ?? 'rand' })" />
                </XField>
                <XField v-slot="a" label="Packet">
                  <XInput v-bind="a" :model-value="n.packet" mono @update:model-value="setNoise(i, { packet: $event })" />
                </XField>
                <XField v-slot="a" label="Delay (ms)">
                  <XInput v-bind="a" :model-value="n.delay" mono @update:model-value="setNoise(i, { delay: $event })" />
                </XField>
              </XNestCard>
              <div>
                <XButton variant="dashed" icon="plus" @click="noises = [...noises!, newNoise()]">{{ t('settings.addNoise') }}</XButton>
              </div>
            </div>
          </XNest>
        </template>
      </XSettingRow>
    </XSettingsSection>

    <XSettingsSection v-model:open="open.mux" :title="t('settings.mux')" :count="1" :modified="f.count(['subJsonMux']).dirty" :errors="f.count(['subJsonMux']).errors">
      <XSettingRow :title="t('settings.mux')" :description="t('settings.muxDesc')" inline :modified="f.isDirty('subJsonMux')">
        <XSwitch
          :model-value="!!mux"
          :label="t('settings.mux')"
          :modified="f.isDirty('subJsonMux')"
          @update:model-value="mux = $event ? { enabled: true, concurrency: 8, xudpConcurrency: 16, xudpProxyUDP443: 'reject' } : null"
        />
        <template v-if="mux" #nested>
          <XNest :title="t('settings.muxNest')" :columns="140">
            <XField v-slot="a" label="Concurrency" :error="muxError(mux.concurrency)">
              <XNumber v-bind="a" :model-value="mux.concurrency" :invalid="!!muxError(mux.concurrency)" @update:model-value="setMux({ concurrency: $event })" />
            </XField>
            <XField v-slot="a" label="xudp Concurrency" :error="muxError(mux.xudpConcurrency)">
              <XNumber v-bind="a" :model-value="mux.xudpConcurrency" :invalid="!!muxError(mux.xudpConcurrency)" @update:model-value="setMux({ xudpConcurrency: $event })" />
            </XField>
            <XField v-slot="a" label="xudp UDP 443">
              <XSelect v-bind="a" :model-value="mux.xudpProxyUDP443" :options="udp443" mono @update:model-value="setMux({ xudpProxyUDP443: $event ?? 'reject' })" />
            </XField>
          </XNest>
        </template>
      </XSettingRow>
    </XSettingsSection>

    <XSettingsSection v-model:open="open.direct" :title="t('settings.direct')" :count="1" :modified="f.count(['subJsonRules']).dirty">
      <XSettingRow :title="t('settings.direct')" :description="t('settings.directDesc')" inline :modified="f.isDirty('subJsonRules')">
        <XSwitch :model-value="!!rules" :label="t('settings.direct')" :modified="f.isDirty('subJsonRules')" @update:model-value="rules = $event ? defaultRules() : null" />
        <template v-if="rules" #nested>
          <XNest :title="t('settings.direct')" :columns="280">
            <XField v-slot="a" :label="t('settings.directIps')">
              <XTagInput
                v-bind="a"
                v-model="directIps"
                :presets="ipPresets"
                :presets-title="t('settings.presets')"
                :placeholder="t('settings.presetPick')"
                :more-placeholder="t('settings.presetMore')"
              />
            </XField>
            <XField v-slot="a" :label="t('settings.directDomains')">
              <XTagInput
                v-bind="a"
                v-model="directDomains"
                :presets="domainPresets"
                :presets-title="t('settings.presets')"
                :placeholder="t('settings.presetPick')"
                :more-placeholder="t('settings.presetMore')"
              />
            </XField>
          </XNest>
        </template>
      </XSettingRow>
    </XSettingsSection>
  </div>
</template>

<style scoped>
.settings-tab {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.noises {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
