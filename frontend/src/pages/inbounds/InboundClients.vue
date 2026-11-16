<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../api'
import {
  toast, XChip, XInlineButton, XInnerTable, XMark, XMenu, XPager, XPopConfirm, XPopover, XStatusDot, XSwitch, XTooltip, XUsagePill,
  type InnerColumn, type MenuItem,
} from '../../kit'
import { formatBytes, formatDate, formatRelative } from '../../i18n/format'
import { useInbounds } from '../../stores/inbounds'
import { cause } from '../settings/form'
import { ask } from './ask'
import {
  allTimeOf, clientKey, clientsOf, clientView, delayedDays, hasLink, pageSizes, renewShare, settingsOf, shownClients, statsMap,
  type Client, type ClientView, type DbInbound, type Filter, type State,
} from './model'

/**
 * The clients of an open inbound, as frames 2a–2j and 4c of the mockup show
 * them: their actions, switch, presence, state, traffic and date, and a pager
 * when there are many.
 */
const props = defineProps<{ inbound: DbInbound; query: string; filter: Filter }>()
const emit = defineEmits<{ action: [key: 'qr' | 'edit' | 'info', client: Client] }>()
const { t } = useI18n()
const store = useInbounds()

const thresholds = computed(() => store.defaults ?? { expireDiff: 0, trafficDiff: 0 })
const stats = computed(() => statsMap(props.inbound))
const total = computed(() => clientsOf(props.inbound).length)
const shown = computed(() => shownClients(props.inbound, props.query, props.filter, store.counts.get(props.inbound.id)))

// --- Pages -------------------------------------------------------------------------

const page = ref(1)
const size = ref(store.defaults?.pageSize || 25)
watch(() => [props.query, props.filter], () => (page.value = 1))
const pages = computed(() => Math.max(1, Math.ceil(shown.value.length / size.value)))
watch(pages, (n) => page.value > n && (page.value = n))
const rows = computed(() =>
  shown.value.length > size.value ? shown.value.slice((page.value - 1) * size.value, page.value * size.value) : shown.value,
)
const sizes = computed(() => pageSizes(store.defaults?.pageSize || 25, shown.value.length))

// --- Rows --------------------------------------------------------------------------

const columns = computed<InnerColumn[]>(() => [
  { key: 'menu', label: t('inbounds.colMenu'), size: '148px', quiet: true },
  { key: 'enable', label: t('inbounds.colEnable'), size: '58px', quiet: true },
  { key: 'online', label: t('inbounds.colOnline'), size: '72px' },
  { key: 'client', label: t('inbounds.colClient') },
  { key: 'traffic', label: t('inbounds.colTraffic'), size: '210px' },
  { key: 'allTime', label: t('inbounds.colAllTime'), size: '84px' },
  { key: 'expiry', label: t('inbounds.colExpiry'), size: '172px' },
])
const phoneColumns = computed<InnerColumn[]>(() => [
  // As wide as the menu's inline button, so it neither runs into the status dot nor leaves room around its dots.
  { key: 'menu', label: t('inbounds.colMenu'), size: 'var(--inline-btn)', quiet: true },
  { key: 'client', label: t('inbounds.colClient') },
  { key: 'info', label: t('inbounds.colInfo'), size: 'max-content', align: 'end', quiet: true },
])

const keyOf = (c: Client) => clientKey(props.inbound.protocol, c)
const view = (c: Client): ClientView => clientView(c, stats.value.get(c.email), store.online, thresholds.value)
const stat = (c: Client) => stats.value.get(c.email)
const toneOf = (s: State) => (s === 'unl' ? 'neutral' : s)

/** The dot: off when disabled, else the worse of traffic and date. */
const dotTone = (c: Client, v: ClientView) => (!c.enable ? 'off' : v.worst)
const dotTip = (c: Client, v: ClientView) =>
  v.depleted ? t('inbounds.stDepleted') : !c.enable ? t('inbounds.stDisabled') : v.online ? t('inbounds.stOnline') : undefined

function seen(c: Client, v: ClientView) {
  const at = v.online ? Date.now() : store.lastOnline[c.email] || stat(c)?.lastOnline || 0
  return t('inbounds.lastOnline', { date: at ? formatDate(at) : '-' })
}

/** Long comments are cut, the whole one shows in a tooltip. */
const commentLimit = 50
const cut = (text: string) => `${text.slice(0, commentLimit).trimEnd()}...`

const daysAfter = (days: number) => formatRelative(Date.now() + days * 86_400_000)

function menu(c: Client): MenuItem[] {
  return [
    ...(hasLink(props.inbound.protocol) ? [{ key: 'qr', label: t('inbounds.qr'), icon: 'qr' as const }] : []),
    { key: 'edit', label: t('inbounds.editClient'), icon: 'edit' },
    { key: 'info', label: t('inbounds.info'), icon: 'info' },
    { key: 'reset', label: t('inbounds.resetTraffic'), icon: 'reset' },
    ...(total.value > 1 ? [{ key: 'delete', label: t('inbounds.delete'), icon: 'delete' as const, danger: true }] : []),
    { key: 'enable', label: t('inbounds.enable'), switch: true, checked: c.enable, busy: toggling.has(keyOf(c)), divided: true },
  ]
}

function act(key: string, c: Client) {
  if (key === 'enable') setEnable(c, !c.enable)
  else if (key === 'reset')
    ask({ title: `${t('inbounds.resetTraffic')} ${c.email}`, text: t('inbounds.resetConfirm'), cta: t('inbounds.reset'), danger: true, run: () => resetRequest(c) })
  else if (key === 'delete')
    ask({ title: `${t('inbounds.deleteClient')} ${c.email}`, text: t('inbounds.deleteConfirm'), cta: t('inbounds.delete'), danger: true, run: () => removeRequest(c) })
  else if (key === 'qr' || key === 'edit' || key === 'info') emit('action', key, c)
}

// --- Changes -----------------------------------------------------------------------

/** Clients whose switch is being applied, and those whose change failed a moment ago. */
const toggling = reactive(new Set<string>())
const refused = reactive(new Set<string>())

/** The inbound with one client's switch changed, in its settings and its traffic. */
function withEnable(c: Client, on: boolean): Partial<DbInbound> {
  const settings = settingsOf(props.inbound)
  const key = keyOf(c)
  settings.clients = (settings.clients as Client[]).map((x) => (keyOf(x) === key ? { ...x, enable: on } : x))
  const clientStats = (props.inbound.clientStats ?? []).map((s) => (s.email === c.email ? { ...s, enable: on } : s))
  return { settings: JSON.stringify(settings, null, 2), clientStats }
}

/** Applied at once and rolled back if the server refuses, as the inbound's own switch. */
async function setEnable(c: Client, on: boolean) {
  const key = keyOf(c)
  if (toggling.has(key)) return
  const id = props.inbound.id
  const before = withEnable(c, c.enable)
  store.patch(id, withEnable(c, on))
  toggling.add(key)
  try {
    await api.post(`panel/api/inbounds/updateClient/${encodeURIComponent(key)}`, { id, settings: JSON.stringify({ clients: [{ ...c, enable: on }] }) })
  } catch (e) {
    store.patch(id, before)
    refused.add(key)
    window.setTimeout(() => refused.delete(key), 3400)
    toast('err', cause(e))
  } finally {
    toggling.delete(key)
  }
}

const resetRequest = (c: Client) => api.post(`panel/api/inbounds/${props.inbound.id}/resetClientTraffic/${encodeURIComponent(c.email)}`)
const removeRequest = (c: Client) => api.post(`panel/api/inbounds/${props.inbound.id}/delClient/${encodeURIComponent(keyOf(c))}`)

/** From the row's own confirmation on desktop; phones ask in the page's window. */
async function confirmed(request: Promise<unknown>) {
  try {
    await request
  } catch (e) {
    toast('err', cause(e))
  }
  await store.load().catch(() => {})
}
</script>

<template>
  <XInnerTable :columns="columns" :phone-columns="phoneColumns" :rows="rows" :row-key="keyOf">
    <template #cell-menu="{ row }">
      <div class="clients__acts">
        <XInlineButton v-if="hasLink(inbound.protocol)" icon="qr" :label="t('inbounds.qr')" tip @click="emit('action', 'qr', row)" />
        <XInlineButton icon="edit" :label="t('inbounds.editClient')" tip @click="emit('action', 'edit', row)" />
        <XInlineButton icon="info" :label="t('inbounds.info')" tip @click="emit('action', 'info', row)" />
        <XPopConfirm v-slot="{ props: trigger }" :text="t('inbounds.resetConfirm')" :cta="t('inbounds.reset')" :run="() => confirmed(resetRequest(row))">
          <XInlineButton v-bind="trigger" icon="reset" :label="t('inbounds.resetTraffic')" tip />
        </XPopConfirm>
        <XPopConfirm v-if="total > 1" v-slot="{ props: trigger }" :text="t('inbounds.deleteConfirm')" :cta="t('inbounds.delete')" :run="() => confirmed(removeRequest(row))">
          <XInlineButton v-bind="trigger" icon="delete" :label="t('inbounds.delete')" danger tip />
        </XPopConfirm>
      </div>
    </template>
    <template #cell-enable="{ row }">
      <XSwitch
        :model-value="row.enable"
        :loading="toggling.has(keyOf(row))"
        :invalid="refused.has(keyOf(row))"
        :label="t('inbounds.enable')"
        @update:model-value="setEnable(row, $event)"
      />
    </template>
    <template #cell-online="{ row }">
      <XTooltip :text="seen(row, view(row))" :delay="0">
        <XChip :tone="view(row).online ? 'ok' : 'neutral'" interactive>{{ view(row).online ? t('inbounds.online') : t('inbounds.offline') }}</XChip>
      </XTooltip>
    </template>
    <template #cell-client="{ row }">
      <div class="clients__who">
        <span class="clients__dot"><XStatusDot :tone="dotTone(row, view(row))" :pulse="view(row).online" :label="dotTip(row, view(row))" tip /></span>
        <div class="clients__name">
          <div :class="['clients__email', 'x-ellipsis', { 'is-off': !row.enable }]" :title="row.email"><XMark :text="row.email" :query="query" /></div>
          <div v-if="row.comment && row.comment.length > commentLimit" class="clients__comment x-ellipsis">
            <XTooltip :text="row.comment" :delay="0" block><span class="x-ellipsis">{{ cut(row.comment) }}</span></XTooltip>
          </div>
          <div v-else-if="row.comment" class="clients__comment x-ellipsis"><XMark :text="row.comment" :query="query" /></div>
        </div>
      </div>
    </template>
    <template #cell-traffic="{ row }">
      <XTooltip :delay="0" block>
        <XUsagePill
          :used="formatBytes(view(row).used)"
          :limit="view(row).total ? formatBytes(view(row).total) : '∞'"
          :ratio="view(row).total ? view(row).used / view(row).total : null"
          :tone="toneOf(view(row).traffic)"
          :exhausted="row.enable && view(row).traffic === 'err'"
          :disabled="!row.enable"
          block
          interactive
        />
        <template #content>
          <div class="clients__updown">↑{{ formatBytes(stat(row)?.up ?? 0) }} ↓{{ formatBytes(stat(row)?.down ?? 0) }}</div>
          <div v-if="view(row).total && view(row).used < view(row).total" class="clients__remain">
            <span>{{ t('inbounds.remain') }}</span><span>{{ formatBytes(view(row).total - view(row).used) }}</span>
          </div>
        </template>
      </XTooltip>
    </template>
    <template #cell-allTime="{ row }">
      <XChip mono>{{ formatBytes(allTimeOf(stat(row))) }}</XChip>
    </template>
    <template #cell-expiry="{ row }">
      <XChip v-if="!view(row).at" icon="infinity" :aria-label="t('inbounds.noExpiry')" />
      <XTooltip v-else-if="view(row).at < 0" :text="t('inbounds.delayedStart')" :delay="0">
        <XChip :tone="row.enable ? 'ok' : 'off'" interactive>{{ daysAfter(delayedDays(view(row).at)!) }}</XChip>
      </XTooltip>
      <XTooltip v-else-if="view(row).renew" :text="formatDate(view(row).at)" :delay="0" block>
        <XUsagePill
          :used="formatRelative(view(row).at)"
          :limit="t('inbounds.periodDays', { n: view(row).renew })"
          :ratio="renewShare(view(row).at, view(row).renew)"
          :tone="toneOf(view(row).expiry)"
          :disabled="!row.enable"
          textual
          block
          interactive
        />
      </XTooltip>
      <XTooltip v-else :text="formatDate(view(row).at)" :delay="0">
        <XChip :tone="row.enable ? toneOf(view(row).expiry) : 'off'" interactive>{{ formatRelative(view(row).at) }}</XChip>
      </XTooltip>
    </template>

    <template #phone-menu="{ row }">
      <XMenu v-slot="{ props: trigger }" :items="menu(row)" placement="bottom-start" @select="act($event, row)">
        <XInlineButton v-bind="trigger" icon="dots" :icon-size="16" :label="t('inbounds.colMenu')" />
      </XMenu>
    </template>
    <template #phone-client="{ row }">
      <div class="clients__who clients__who--phone">
        <span class="clients__dot"><XStatusDot :tone="dotTone(row, view(row))" :pulse="view(row).online" /></span>
        <div class="clients__name">
          <div :class="['clients__email', 'x-ellipsis', { 'is-off': !row.enable }]"><XMark :text="row.email" :query="query" /></div>
          <div v-if="row.comment" class="clients__comment x-ellipsis">
            <template v-if="row.comment.length > commentLimit">{{ cut(row.comment) }}</template>
            <XMark v-else :text="row.comment" :query="query" />
          </div>
        </div>
      </div>
    </template>
    <template #phone-info="{ row }">
      <div class="clients__tail">
        <!-- The traffic used, in the colour of its state; the bar does not fit a phone row, the summary has it. -->
        <XChip :tone="row.enable ? toneOf(view(row).traffic) : 'off'" mono>{{ formatBytes(view(row).used) }}</XChip>
        <XPopover :label="t('inbounds.colInfo')">
          <template #default="{ props: trigger }">
            <!-- A badge with the info mark rather than a round button, as the inbounds have; paused and grey while the client is off. -->
            <XChip v-bind="trigger" interactive :icon="row.enable ? 'info' : 'pause'" :tone="row.enable ? 'neutral' : 'off'" :aria-label="t('inbounds.colInfo')" />
          </template>
          <template #content>
            <div class="clients__info">
              <div class="clients__label">{{ t('inbounds.colTraffic') }}</div>
              <XUsagePill
                :used="formatBytes(view(row).used)"
                :limit="view(row).total ? formatBytes(view(row).total) : '∞'"
                :ratio="view(row).total ? view(row).used / view(row).total : null"
                :tone="toneOf(view(row).traffic)"
                :exhausted="row.enable && view(row).traffic === 'err'"
                :disabled="!row.enable"
                block
                large
              />
              <div class="clients__line">
                <span>↑{{ formatBytes(stat(row)?.up ?? 0) }} ↓{{ formatBytes(stat(row)?.down ?? 0) }}</span>
                <span v-if="view(row).total && view(row).used < view(row).total" class="clients__end">
                  {{ t('inbounds.remain') }} {{ formatBytes(view(row).total - view(row).used) }}
                </span>
              </div>
              <div class="clients__split" />
              <div class="clients__label">{{ t('inbounds.colExpiry') }}</div>
              <XChip v-if="!view(row).at" icon="infinity" :aria-label="t('inbounds.noExpiry')" />
              <template v-else-if="view(row).renew">
                <XUsagePill
                  :used="formatRelative(view(row).at)"
                  :limit="t('inbounds.periodDays', { n: view(row).renew })"
                  :ratio="renewShare(view(row).at, view(row).renew)"
                  :tone="toneOf(view(row).expiry)"
                  :disabled="!row.enable"
                  textual
                  block
                  large
                />
                <div class="clients__line">{{ formatDate(view(row).at) }}</div>
              </template>
              <div v-else class="clients__when">
                <XChip v-if="view(row).at < 0" :tone="row.enable ? 'ok' : 'off'">{{ daysAfter(delayedDays(view(row).at)!) }}</XChip>
                <XChip v-else :tone="row.enable ? toneOf(view(row).expiry) : 'off'">{{ formatRelative(view(row).at) }}</XChip>
                <span class="clients__date">{{ view(row).at < 0 ? t('inbounds.delayedStart') : formatDate(view(row).at) }}</span>
              </div>
            </div>
          </template>
        </XPopover>
      </div>
    </template>

    <template v-if="shown.length > (store.defaults?.pageSize || 25)" #foot>
      <XPager v-model="page" v-model:size="size" :pages="pages" :sizes="sizes" class="clients__pager" />
    </template>
  </XInnerTable>
</template>

<style scoped>
.clients__acts {
  display: flex;
  gap: 2px;
}

/* One line: the dot in its middle, the comment after the email, cut first when the line runs out. */
.clients__who {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.clients__dot {
  display: flex;
  flex: none;
}

/* The dot's hover spot is wider than the dot: the dot itself stands under the column label. */
.clients__who:not(.clients__who--phone) .clients__dot {
  margin-left: -4px;
}

.clients__name {
  display: flex;
  flex: 1;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
}

.clients__email {
  flex: 0 1 auto;
  color: var(--fg);
  font: 500 var(--fs-13) / 1.4 var(--font-mono);
}

.clients__email.is-off {
  color: var(--mut);
}

.clients__comment {
  flex: 1 1 0;
  min-width: 3em;
  color: var(--dim);
  font: 400 var(--fs-12) / 1.4 var(--font-sans);
}

.clients__tail {
  display: flex;
  align-items: center;
  gap: 6px;
}

.clients__updown {
  color: var(--fg);
  font: 500 var(--fs-12) / var(--lh-tight) var(--font-mono);
  white-space: nowrap;
}

.clients__remain {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px solid var(--div);
  color: var(--dim);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-sans);
}

.clients__remain span:last-child {
  color: var(--fg);
  font: 500 var(--fs-12) / var(--lh-tight) var(--font-mono);
}

.clients__info {
  /* A phone's popover: wide enough for phone type, short of the screen's edges. */
  width: min(320px, calc(100vw - 56px));
}

.clients__label {
  margin-bottom: 8px;
  color: var(--dim);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-sans);
}

.clients__line {
  display: flex;
  gap: 12px;
  margin-top: 8px;
  color: var(--mut);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-mono);
  white-space: nowrap;
}

.clients__end {
  margin-left: auto;
}

.clients__split {
  height: 1px;
  margin: 12px 0;
  background: var(--div);
}

.clients__when {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.clients__date {
  color: var(--mut);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-mono);
}

.clients__pager {
  justify-content: flex-end;
}
</style>
