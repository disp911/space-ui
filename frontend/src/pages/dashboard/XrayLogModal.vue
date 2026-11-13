<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, ApiError, saveFile } from '../../api'
import { formatNumber } from '../../i18n/format'
import {
  toast, usePhone, XButton, XCheckbox, XDataTable, XEmpty, XInput, XMenu, XModal, XPager, XRecordCard, XStatusDot,
  type MenuItem, type RowTone, type TableColumn,
} from '../../kit'

/** Connections Xray wrote to its access log, a day at a time: filters, search, pages of 200. */
const open = defineModel<boolean>('open', { default: false })
const { t, locale } = useI18n()

interface Entry {
  DateTime: string
  FromAddress: string
  ToAddress: string
  Inbound: string
  Outbound: string
  Email: string
  /** 0 direct, 1 blocked, 2 proxied. */
  Event: number
}

interface LogPage {
  entries: Entry[] | null
  total: number
  page: number
  pageSize: number
  date: string
  dates: string[] | null
  clients: string[] | null
}

type Kind = 'direct' | 'blocked' | 'proxy'
const kinds = [
  { key: 'direct', label: 'Direct', tone: 'ok', dot: 'var(--ok)' },
  { key: 'blocked', label: 'Blocked', tone: 'err', dot: 'var(--err)' },
  { key: 'proxy', label: 'Proxy', tone: 'upload', dot: 'var(--upload)' },
] as const
const byEvent: Kind[] = ['direct', 'blocked', 'proxy']
const tones: Record<Kind, RowTone | undefined> = { direct: undefined, blocked: 'err', proxy: 'upload' }
const dots: Record<Kind, string> = { direct: 'var(--faint)', blocked: 'var(--err)', proxy: 'var(--upload)' }

// Filters; an empty date asks for the latest day.
const date = ref('')
const client = ref('')
const query = ref('')
const shown = reactive<Record<Kind, boolean>>({ direct: true, blocked: true, proxy: true })
const page = ref(1)

const data = ref<LogPage | null>(null)
const loading = ref(false)
const failed = ref('')

function form(withPage: boolean) {
  const body = new FormData()
  body.set('date', date.value)
  body.set('email', client.value)
  body.set('filter', query.value.trim())
  body.set('showDirect', String(shown.direct))
  body.set('showBlocked', String(shown.blocked))
  body.set('showProxy', String(shown.proxy))
  if (withPage) body.set('page', String(page.value))
  return body
}

let request = 0
async function load() {
  const id = ++request
  loading.value = true
  failed.value = ''
  try {
    const reply = await api.post<LogPage>('panel/api/server/xraylogs', form(true))
    if (id === request) data.value = reply
  } catch (e) {
    if (id !== request) return
    data.value = null
    failed.value = e instanceof ApiError ? e.message : String(e)
  } finally {
    if (id === request) loading.value = false
  }
}

/** A changed filter starts again from the first page. */
function reload() {
  page.value = 1
  load()
}

watch(open, (on) => on && reload(), { immediate: true })
watch([date, client, shown], reload)

// Search waits for a pause in typing.
let typing: ReturnType<typeof setTimeout> | undefined
watch(query, () => {
  clearTimeout(typing)
  typing = setTimeout(reload, 300)
})
onBeforeUnmount(() => clearTimeout(typing))

// The phone cards scroll on their own; another page starts at their top.
const cards = ref<HTMLElement | null>(null)

function goTo(to: number) {
  page.value = to
  cards.value?.scrollTo({ top: 0 })
  load()
}

// --- Labels -----------------------------------------------------------------

const formatDay = (day: string) =>
  day ? new Intl.DateTimeFormat(locale.value, { day: '2-digit', month: '2-digit', year: 'numeric' }).format(new Date(`${day}T00:00:00`)) : '—'

const total = computed(() => data.value?.total ?? 0)
const pageSize = computed(() => data.value?.pageSize || 200)
const pages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const day = computed(() => formatDay(data.value?.date ?? ''))

const range = computed(() => {
  if (!total.value) return t('dashboard.xlogRangeEmpty')
  const from = (page.value - 1) * pageSize.value + 1
  const to = Math.min(page.value * pageSize.value, total.value)
  return t('dashboard.xlogRange', {
    from: formatNumber(from), to: formatNumber(to), total: formatNumber(total.value), page: page.value, pages: pages.value,
  })
})

const dayItems = computed<MenuItem[]>(() =>
  (data.value?.dates ?? []).map((d) => ({ key: d, label: formatDay(d), checked: d === data.value?.date })),
)
const clientItems = computed<MenuItem[]>(() => [
  { key: '', label: t('dashboard.xlogAllClients'), checked: !client.value },
  ...(data.value?.clients ?? []).map((c) => ({ key: c, label: c, checked: c === client.value })),
])
const kindItems = computed<MenuItem[]>(() => kinds.map((k) => ({ key: k.key, label: k.label, dot: k.dot, checked: shown[k.key] })))
// Phones show the types filter as an icon square (a dot when it holds back rows) and page at the end of the cards.
const phone = usePhone()
const allKinds = computed(() => kinds.every((k) => shown[k.key]))
const kindsLabel = computed(() => {
  const on = kinds.filter((k) => shown[k.key])
  if (on.length === kinds.length) return t('dashboard.xlogAllTypes')
  if (!on.length) return t('dashboard.xlogNoTypes')
  return on.length === 1 ? on[0]!.label : t('dashboard.xlogSomeTypes', { n: on.length })
})

// --- Rows -------------------------------------------------------------------

const columns = computed<TableColumn[]>(() => [
  { key: 'time', label: t('dashboard.xlogTime'), width: 110, dim: true },
  { key: 'from', label: t('dashboard.xlogFrom'), width: 160 },
  { key: 'to', label: t('dashboard.xlogTo') },
  { key: 'inbound', label: t('dashboard.xlogInbound'), width: 150 },
  { key: 'outbound', label: t('dashboard.xlogOutbound'), width: 132 },
  { key: 'email', label: t('dashboard.xlogEmail'), width: 150, strong: true },
])

const rows = computed(() =>
  (data.value?.entries ?? []).map((e, i) => {
    // The server's own clock, as the day was cut by it: no conversion to this browser's zone.
    const [, , month = '', dayOfMonth = '', clock = ''] = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}:\d{2}:\d{2})/.exec(e.DateTime) ?? []
    const kind = byEvent[e.Event] ?? 'direct'
    return {
      id: i,
      time: `${dayOfMonth}.${month} ${clock}`,
      clock,
      from: e.FromAddress,
      to: e.ToAddress,
      inbound: e.Inbound,
      outbound: e.Outbound,
      email: e.Email,
      kind,
    }
  }),
)
type Row = (typeof rows.value)[number]

// --- Download ---------------------------------------------------------------

const downloading = ref(false)

async function download() {
  downloading.value = true
  try {
    const { data: file, name } = await api.file('panel/api/server/xraylogs/download', form(false))
    saveFile(file, name ?? 'xray.log')
  } catch (e) {
    toast('err', t('dashboard.actionFailed', { error: e instanceof ApiError ? e.message : String(e) }))
  } finally {
    downloading.value = false
  }
}
</script>

<template>
  <XModal v-model:open="open" :title="t('dashboard.xlogTitle')" size="wide" flush full>
    <template #subtitle>
      <template v-if="data?.date">{{ t('dashboard.xlogSubtitle', { n: formatNumber(total), date: day }, total) }}</template>
      <template v-else>{{ t('dashboard.logEmpty') }}</template>
    </template>

    <div class="xlog-bar">
      <XButton icon="refresh" :label="t('dashboard.logRefresh')" :loading="loading" class="xlog-refresh" @click="load" />

      <XMenu v-slot="{ props }" :items="dayItems" :title="t('dashboard.xlogDate')" placement="bottom-start" sheet @select="date = $event">
        <XButton v-bind="props" :label="`${t('dashboard.xlogDate')}: ${day}`" caret class="xlog-filter">
          <span class="x-caps xlog-filter__label">{{ t('dashboard.xlogDate') }}</span> <span class="mono">{{ day }}</span>
        </XButton>
      </XMenu>

      <XMenu v-slot="{ props }" :items="clientItems" :title="t('dashboard.xlogClient')" placement="bottom-start" sheet @select="client = $event">
        <XButton v-bind="props" :label="`${t('dashboard.xlogClient')}: ${client || t('dashboard.xlogAllClients')}`" caret class="xlog-filter">
          <span class="x-caps xlog-filter__label">{{ t('dashboard.xlogClient') }}</span> <span class="mono">{{ client || t('dashboard.xlogAllClients') }}</span>
        </XButton>
      </XMenu>

      <XInput v-model="query" type="search" icon="search" clearable mono :placeholder="t('dashboard.xlogSearch')" class="xlog-search" />

      <XMenu v-slot="{ props }" :items="kindItems" :title="t('dashboard.xlogTypes')" placement="bottom-start" sheet multiple @select="shown[$event as Kind] = !shown[$event as Kind]">
        <XButton
          v-bind="props"
          :label="`${t('dashboard.xlogTypes')}: ${kindsLabel}`"
          :icon="phone ? 'filter' : undefined"
          :caret="!phone"
          :dot="phone && !allKinds"
          phone-icon-only
          class="xlog-filter xlog-types xlog-icon"
        >
          {{ kindsLabel }}
        </XButton>
      </XMenu>

      <XCheckbox v-for="k in kinds" :key="k.key" v-model="shown[k.key]" boxed :tone="k.tone" class="xlog-check">{{ k.label }}</XCheckbox>

      <XButton icon="download" :label="t('dashboard.logDownload')" :loading="downloading" phone-icon-only class="xlog-download" @click="download">
        {{ t('dashboard.logDownload') }}
      </XButton>
    </div>

    <div class="xlog-area">
      <XDataTable
        class="xlog-wide"
        :columns="columns"
        :rows="rows"
        :row-key="(r: Row) => r.id"
        :row-tone="(r: Row) => tones[r.kind]"
        :loading="loading"
        :height="396"
      >
        <template #cell-outbound="{ row }">
          <XStatusDot size="sm" :color="dots[(row as Row).kind]" class="xlog-dot" /> {{ (row as Row).outbound }}
        </template>
        <template #empty>
          <XEmpty v-if="failed" :title="t('dashboard.logFailed', { error: failed })">
            <XButton size="sm" icon="refresh" @click="load">{{ t('dashboard.retry') }}</XButton>
          </XEmpty>
          <XEmpty v-else-if="!data?.dates?.length" :title="t('dashboard.logEmpty')" />
          <XEmpty v-else :title="t('dashboard.xlogNothing')" :hint="t('dashboard.xlogNothingHint')" />
        </template>
      </XDataTable>

      <div ref="cards" class="xlog-cards xlog-narrow">
        <template v-if="loading">
          <XRecordCard v-for="n in 6" :key="n" loading />
        </template>
        <template v-else-if="rows.length">
          <XRecordCard
            v-for="r in rows"
            :key="r.id"
            :title="r.to"
            :time="r.clock"
            :name="r.email"
            :detail="r.inbound"
            :badge="r.outbound"
            :tone="tones[r.kind]"
          />
        </template>
        <XEmpty v-else-if="failed" :title="t('dashboard.logFailed', { error: failed })">
          <XButton size="sm" icon="refresh" @click="load">{{ t('dashboard.retry') }}</XButton>
        </XEmpty>
        <XEmpty v-else-if="!data?.dates?.length" :title="t('dashboard.logEmpty')" />
        <XEmpty v-else :title="t('dashboard.xlogNothing')" :hint="t('dashboard.xlogNothingHint')" />
        <!-- Phones page at the end of the cards, not in a bar under them. -->
        <div v-if="!loading && rows.length && pages > 1" class="xlog-pages">
          <XPager :model-value="page" :pages="pages" @update:model-value="goTo" />
        </div>
      </div>
    </div>

    <template v-if="!phone" #footer>
      <div class="xlog-foot">
        <span class="xlog-range xlog-wide">{{ range }}</span>
        <XPager :model-value="page" :pages="pages" @update:model-value="goTo" />
      </div>
    </template>
  </XModal>
</template>

<style scoped>
/*
 * One line of tools. The bar measures itself: when the line no longer fits
 * (1020px of content, in Russian, the longer language), the type boxes fold
 * into the Types menu and the filters drop their labels, as on phones, so the
 * window keeps its height. Phones arrange the tools on two lines below.
 */
.xlog-bar {
  container: xlog-bar / inline-size;
  display: flex;
  flex: none;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--div);
  background: var(--foot);
}

.xlog-filter__label {
  margin-right: 2px;
}

/* A long client email keeps to its button: the line's width does not depend on it. */
.xlog-filter {
  max-width: 240px;
}

.xlog-types {
  display: none;
}

.xlog-search {
  flex: 1;
  min-width: 160px;
}

@container xlog-bar (max-width: 1020px) {
  .xlog-check,
  .xlog-filter__label {
    display: none;
  }

  .xlog-types {
    display: inline-flex;
  }

  .xlog-filter {
    max-width: 160px;
  }

  .xlog-search {
    min-width: 100px;
  }
}

/* Sits in the cell's text line. */
.xlog-dot {
  margin-right: 2px;
  vertical-align: middle;
}

.xlog-area {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  padding: 12px;
  background: var(--bg);
}

.xlog-foot {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  min-width: 0;
}

.xlog-range {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: var(--mut);
  font: 400 var(--fs-12) / var(--lh-tight) var(--font-sans);
  white-space: nowrap;
  text-overflow: ellipsis;
}

/* The table and its tools on wide screens; cards and a types menu on phones. */
@media (min-width: 768px) {
  .xlog-narrow {
    display: none;
  }
}

@media (max-width: 767px) {
  .xlog-wide {
    display: none;
  }

  /* Refresh, search and the download square on the first line, then date, client and the types square. */
  .xlog-bar {
    padding: 10px 14px;
  }

  .xlog-refresh {
    order: -3;
  }

  .xlog-search {
    flex-basis: calc(100% - 2 * var(--btn-sq) - 16px);
    order: -2;
  }

  .xlog-download {
    order: -1;
  }

  /* Date and client share the line, the types square keeps to its right; a long value ends in an ellipsis. */
  .xlog-filter {
    flex: 1;
    min-width: 0;
    max-width: none;
  }

  .xlog-icon {
    flex: none;
  }

  .xlog-area {
    padding: 0;
  }

  .xlog-cards {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 8px;
    min-height: 0;
    /* The last card and the pages clear the home indicator, as the window's footer did. */
    padding: 12px 14px calc(14px + env(safe-area-inset-bottom));
    overflow-y: auto;
  }

  .xlog-pages {
    display: flex;
    justify-content: center;
    padding-top: 4px;
  }
}
</style>
