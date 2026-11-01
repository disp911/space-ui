<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePhone, XAddButton, XButton, XField, XFieldGrid, XInput, XMultiSelect, XNest, XNestCard, XNumber, XSelect, XSwitch, type Option } from '../../kit'
import { alpns, defaultExternalProxy, fingerprints, type InboundForm } from './inbound'

/**
 * External Proxy in window 11: the addresses links point at instead of the
 * panel's own, e.g. a CDN, each with its TLS, port and remark. Turning it on
 * starts with the panel's address and the inbound's port, as the old panel did.
 * With Force TLS a row also says the SNI, fingerprint and ALPN of that TLS,
 * as a CDN reached by its address needs.
 */
const props = defineProps<{ f: InboundForm }>()
const { t } = useI18n()
const phone = usePhone()

type Json = Record<string, any>
const rows = computed<Json[]>(() => (Array.isArray(props.f.streamSettings?.externalProxy) ? props.f.streamSettings!.externalProxy : []))

const on = computed({
  get: () => rows.value.length > 0,
  set: (v: boolean) => ((props.f.streamSettings ??= {}).externalProxy = v ? [defaultExternalProxy(props.f.port)] : []),
})

const add = () => rows.value.push({ forceTls: 'same', dest: '', port: 443, remark: '' })

const tlsOptions = computed<Option[]>(() => [
  { value: 'same', label: t('inbounds.same') },
  { value: 'none', label: t('inbounds.none') },
  { value: 'tls', label: 'TLS' },
])
const fingerprintOptions: Option[] = [{ value: '', label: 'None' }, ...fingerprints.map((v) => ({ value: v, label: v }))]
const alpnOptions: Option[] = alpns.map((v) => ({ value: v, label: v }))
</script>

<template>
  <XSwitch v-model="on" boxed label="External Proxy">
    <template v-if="on" #actions>
      <XAddButton :label="t('kit.addRow', { list: 'External Proxy' })" @click="add" />
    </template>
  </XSwitch>
  <XNest v-if="on" bare>
    <XFieldGrid :cols="1" tight>
      <template v-if="phone">
        <XNestCard v-for="(r, i) in rows" :key="i" :title="r.remark || String(i + 1)">
          <XFieldGrid :cols="1">
            <XField v-slot="a" label="Force TLS">
              <XFieldGrid :cols="2" :phone="2">
                <XSelect v-bind="a" v-model="r.forceTls" :options="tlsOptions" />
                <XNumber v-model="r.port" block :min="1" :max="65535" :aria-label="t('inbounds.colPort')" />
              </XFieldGrid>
            </XField>
            <XInput v-model.trim="r.dest" mono :placeholder="t('inbounds.host')" :aria-label="t('inbounds.host')" />
            <template v-if="r.forceTls === 'tls'">
              <XInput v-model.trim="r.sni" mono placeholder="SNI" aria-label="SNI" />
              <XSelect :model-value="r.fingerprint" :options="fingerprintOptions" placeholder="Fingerprint" aria-label="Fingerprint" @update:model-value="r.fingerprint = $event" />
              <XMultiSelect :model-value="r.alpn ?? []" :options="alpnOptions" mono placeholder="ALPN" aria-label="ALPN" @update:model-value="r.alpn = $event" />
            </template>
            <XFieldGrid columns="minmax(0, 1fr) auto" phone-columns="minmax(0, 1fr) auto">
              <XInput v-model.trim="r.remark" :placeholder="t('inbounds.colRemark')" :aria-label="t('inbounds.colRemark')" />
              <XButton icon="minus" :label="t('kit.removeRow', { n: i + 1 })" @click="rows.splice(i, 1)" />
            </XFieldGrid>
          </XFieldGrid>
        </XNestCard>
      </template>
      <template v-for="(r, i) in rows" v-else :key="i">
        <XFieldGrid columns="110px minmax(0, 1.6fr) 76px minmax(0, 1fr) auto" tight>
          <XSelect v-model="r.forceTls" :options="tlsOptions" aria-label="Force TLS" />
          <XInput v-model.trim="r.dest" mono :placeholder="t('inbounds.host')" :aria-label="t('inbounds.host')" />
          <XNumber v-model="r.port" block :min="1" :max="65535" :aria-label="t('inbounds.colPort')" />
          <XInput v-model.trim="r.remark" :placeholder="t('inbounds.colRemark')" :aria-label="t('inbounds.colRemark')" />
          <XButton icon="minus" :label="t('kit.removeRow', { n: i + 1 })" @click="rows.splice(i, 1)" />
        </XFieldGrid>
        <!-- The TLS of the row under it, from the host's column to the remove button's. -->
        <XFieldGrid v-if="r.forceTls === 'tls'" columns="110px repeat(3, minmax(0, 1fr)) var(--btn-sq)" tight>
          <span />
          <XInput v-model.trim="r.sni" mono placeholder="SNI" aria-label="SNI" />
          <XSelect :model-value="r.fingerprint" :options="fingerprintOptions" placeholder="Fingerprint" aria-label="Fingerprint" @update:model-value="r.fingerprint = $event" />
          <XMultiSelect :model-value="r.alpn ?? []" :options="alpnOptions" mono placeholder="ALPN" aria-label="ALPN" @update:model-value="r.alpn = $event" />
          <span />
        </XFieldGrid>
      </template>
    </XFieldGrid>
  </XNest>
</template>
