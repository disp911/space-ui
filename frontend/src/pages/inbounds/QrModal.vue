<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { XButton, XModal, XPopover, XQrCard, XSwitch } from '../../kit'
import { useInbounds } from '../../stores/inbounds'
import { useServerStatus } from '../../stores/server'
import { allLinks, linkForm, wireguardConfigs, withIPv4 } from './links'
import type { Client, DbInbound } from './model'

/**
 * Window 16 of the mockup: a client's subscriptions and share links as QR
 * codes, or an inbound's own (WireGuard's peer configs, Shadowsocks with one
 * key), as the old panel's QR window. A click on a code copies it; the gear
 * switches each link to the server's public IPv4 instead of its address.
 */
const props = defineProps<{ inbound: DbInbound | null; client: Client | null }>()
const open = defineModel<boolean>('open', { default: false })
const { t } = useI18n()
const store = useInbounds()
const server = useServerStatus()
const uid = useId()

interface Code {
  label: string
  link: string
}

const form = computed(() => (props.inbound ? linkForm(props.inbound, store.list) : null))
const remarkModel = computed(() => store.defaults?.remarkModel || '-ieo')

/** Subscriptions first, as the old window had them; MTProto has none. */
const subs = computed<Code[]>(() => {
  const c = props.client
  const d = store.defaults
  if (!c?.subId || !d?.subEnable || props.inbound?.protocol === 'mtproto') return []
  const out = [{ label: t('inbounds.qrSub'), link: (d.subURI ?? '') + c.subId }]
  if (d.subJsonEnable) out.push({ label: t('inbounds.qrSubJson'), link: (d.subJsonURI ?? '') + c.subId })
  return out
})

const links = computed<Code[]>(() => {
  const f = form.value
  if (!f) return []
  if (f.protocol === 'wireguard') return wireguardConfigs(f, remarkModel.value).map((link, i) => ({ label: `Peer ${i + 1}`, link }))
  return allLinks(f, remarkModel.value, props.client ?? undefined).map((l) => ({ label: l.remark, link: l.link }))
})

/** Which links go to the server's IPv4, by their place. */
const useIPv4 = ref<boolean[]>([])
const ipv4 = computed(() => server.status?.publicIP?.ipv4 ?? '')

watch(open, (on) => {
  if (!on) return
  useIPv4.value = []
  server.refresh().catch(() => {})
})

const shown = computed(() => links.value.map((l, i) => ({ ...l, link: useIPv4.value[i] ? withIPv4(l.link, ipv4.value) : l.link })))
const count = computed(() => subs.value.length + shown.value.length)

/** MTProto's WEB proxy keeps its domain, which its TLS needs. */
const switchable = (l: Code) => !l.link.startsWith('tg://webproxy')

function setIPv4(i: number, on: boolean) {
  const next = [...useIPv4.value]
  next[i] = on
  useIPv4.value = next
}
</script>

<template>
  <XModal v-model:open="open" :title="t('inbounds.qr')" size="fit">
    <template #title-end>
      <XPopover v-if="links.some(switchable)" :label="t('inbounds.qrOptions')">
        <template #default="{ props: trigger, open: gearOpen }">
          <XButton v-bind="trigger" icon="gear" :variant="gearOpen ? 'soft' : 'ghost'" :label="t('inbounds.qrOptions')" />
        </template>
        <template #content>
          <div class="qr-options">
            <template v-for="(l, i) in links" :key="i">
              <div v-if="switchable(l)" class="qr-options__item">
                <span class="qr-options__name x-ellipsis">{{ l.label }}</span>
                <span class="qr-options__line">
                  <XSwitch :id="`${uid}-${i}`" :model-value="!!useIPv4[i]" :label="t('inbounds.useIPv4ForHost')" @update:model-value="setIPv4(i, $event)" />
                  <label :for="`${uid}-${i}`" class="qr-options__text">{{ t('inbounds.useIPv4ForHost') }}</label>
                </span>
              </div>
            </template>
          </div>
        </template>
      </XPopover>
    </template>

    <div class="qr" :style="{ '--qr-cols': Math.min(3, count) }">
      <XQrCard v-for="s in subs" :key="s.label" :label="s.label" :value="s.link" tone="unl" />
      <XQrCard v-for="(l, i) in shown" :key="i" :label="l.label" :value="l.link" />
    </div>
  </XModal>
</template>

<style scoped>
.qr {
  display: grid;
  grid-template-columns: repeat(var(--qr-cols), auto);
  gap: var(--sp-12);
}

.qr-options {
  display: flex;
  flex-direction: column;
  width: 276px;
}

.qr-options__item {
  display: flex;
  flex-direction: column;
  gap: var(--sp-6);
  padding-block: var(--sp-10);
}

.qr-options__item:first-child {
  padding-top: 0;
}

.qr-options__item:last-child {
  padding-bottom: 0;
}

/* Hairlines between the links, as between the items of every popup. */
.qr-options__item + .qr-options__item {
  border-top: 1px solid var(--div);
}

.qr-options__name {
  color: var(--fg);
  font: 600 var(--fs-13) / var(--lh-tight) var(--font-sans);
}

.qr-options__line {
  display: flex;
  align-items: center;
  gap: var(--sp-8);
}

.qr-options__text {
  color: var(--fg2);
  font: 400 var(--fs-12) / var(--lh-text) var(--font-sans);
  cursor: pointer;
}

@media (max-width: 767px) {
  .qr {
    grid-template-columns: 1fr;
  }

  .qr-options {
    width: auto;
  }
}
</style>
