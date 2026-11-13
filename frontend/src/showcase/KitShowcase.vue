<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useTheme } from '../stores/theme'
import {
  toast, XActionBar, XActionList, XActionRow, XAddButton, XBadge, XCountList, XExpandTable, XSummaryStat, XBusyOverlay, XButton, XButtonGroup, XCard, XCheckbox, XChip, XCodeArea, XCodeEditor, XCodeInput, XConfirm, XCounter, XDataTable, XDivider, XEmpty, XErrorBox, XExample, XField, XFieldGrid, XFieldNote, XGridTable, XInfoRow, XInlineButton, XInput, XInputList, XKeyValue, XListHead, XMenu, XModal, XMultiSelect, XNav, XNavItem, XNest, XNestCard, XNotice, XPlainTable,
  XNumber, XOrderToggles, XPager, XPairList, XPickList, XPopConfirm, XProcessError, XProgressLine, XQrCode, XRecordCard, XRouteTarget, XRowLink, XSection, XSegmented, XSelect, XSettingBlock, XSettingDivider, XSettingRow, XSettingsSection, XSettingsSkeleton, XStatusDot, XStatusText, XSwitch, XTabs, XTagInput, XTestResult, XTextarea,
  XTooltip, XUsagePill, XValueChip, XInnerTable, XMark, XRowToggle, XDateTime, XPartHead, XButtonRow, XCheckGroup, XStateBar, XChipBox,
  XFactRow, XFactTile, XCopyChip, XCopyBlock, XCopyFrame, XQrCard, XPopover,
  type MenuItem,
} from '../kit'

/*
 * Every kit component in every state, in both themes. The state table forces
 * hover, focus and pressed with the is-hover / is-focus / is-active classes
 * that the components accept next to the real pseudo-classes.
 */

const store = useTheme()
const theme = computed({ get: () => store.theme, set: (v) => store.set(v) })

const states = [
  { key: 'default', label: 'Обычное', cls: '' },
  { key: 'hover', label: 'Наведение', cls: 'is-hover' },
  { key: 'focus', label: 'Фокус', cls: 'is-focus' },
  { key: 'active', label: 'Нажатие', cls: 'is-active' },
  { key: 'disabled', label: 'Недоступно', cls: '' },
  { key: 'invalid', label: 'Ошибка', cls: '' },
  { key: 'modified', label: 'Изменено', cls: '' },
] as const
type State = (typeof states)[number]['key']
const has = (s: State, ...list: State[]) => list.includes(s)

const languages = [
  { value: 'ru', label: '🇷🇺 Русский' },
  { value: 'en', label: '🇬🇧 English' },
  { value: 'zh', label: '🇨🇳 中文', disabled: true },
]
const strategies = ['AsIs', 'IPIfNonMatch', 'IPOnDemand'].map((v) => ({ value: v, label: v }))
const badRouting = 'happ://routing/…'
const geo = ['geoip:private', 'geoip:ru', 'geosite:category-ads-all', 'geosite:google', 'geosite:telegram']

const form = ref({
  enabled: true,
  remark: 'Germany-VLESS',
  port: 443 as number | null,
  lang: 'ru' as string | null,
  strategy: 'IPIfNonMatch' as string | null,
  password: 'correct-horse',
  tags: ['geoip:private', 'geoip:ru'],
  note: 'Текст объявления для подписки',
  sniffing: true,
  routeOnly: false,
  agree: true,
})
const view = ref<'all' | 'online' | 'depleted'>('all')
const mode = ref<'search' | 'filter'>('search')
const page = ref(5)
const modalOpen = ref(false)
const sheetSize = ref<'sm' | 'md' | 'lg' | 'xl'>('md')
const boxedCheck = ref(true)
const successDemo = ref(false)
const logOpen = ref(false)
const confirmOpen = ref(false)
const confirmBusy = ref(false)
function runConfirm() {
  confirmBusy.value = true
  setTimeout(() => {
    confirmBusy.value = false
    confirmOpen.value = false
  }, 1500)
}
const level = ref('info')
const levelItems = computed<MenuItem[]>(() =>
  [
    { key: 'debug', label: 'Debug', dot: 'var(--upload)' },
    { key: 'info', label: 'Info', dot: 'var(--acc)' },
    { key: 'warning', label: 'Warning', dot: 'var(--warn)' },
    { key: 'err', label: 'Error', dot: 'var(--err)' },
  ].map((i) => ({ ...i, checked: i.key === level.value })),
)
const lastAction = ref('—')
const codeText = ref(JSON.stringify({ log: { loglevel: 'warning' }, dns: { servers: ['https://1.1.1.1/dns-query'] }, routing: { domainStrategy: 'AsIs' } }, null, 2))
const setTab = ref('panel')
const setTabs = [
  { key: 'panel', label: 'Панель', dot: 'err' as const },
  { key: 'ui', label: 'Интерфейс' },
  { key: 'acct', label: 'Учетная запись' },
  { key: 'ldap', label: 'LDAP', dot: 'mod' as const },
  { key: 'sub', label: 'Подписка' },
  { key: 'fmt', label: 'Formats' },
]
const setOpen = ref(true)
const xrayOpen = ref(true)
const editorPart = ref('all')
const demoAttrs = ref<[string, string][]>([[':method', 'GET'], [':path', '/api/*']])
const demoDomains = ref(['geosite:netflix', 'geosite:disney'])
const demoCatalog = [
  { title: 'Сервисы', options: [{ label: 'Google', value: 'geosite:google' }, { label: 'Netflix', value: 'geosite:netflix' }, { label: 'OpenAI', value: 'geosite:openai' }] },
  { title: 'Реклама и угрозы', options: [{ label: 'Ads All', value: 'geosite:category-ads-all' }] },
]
const demoRules = ref([
  { dest: ['geosite:category-ads-all', 'geosite:category-ads', 'domain:doubleclick.net'], to: 'blocked', bal: false },
  { dest: ['geoip:private'], to: 'direct', bal: false },
  { dest: [] as string[], to: 'eu-pool', bal: true },
])
const demoColumns = [
  { key: 'dest', label: 'Пункт назначения' },
  { key: 'to', label: 'Исходящее подключение', size: '.85fr', phone: 'head' as const },
]
const demoCountry = ref(2)
const demoProto = ref('vless')
const demoEmail = ref('k3x9mf2q')
const demoNet = ref('xhttp')
const demoWeb = ref(true)
const demoSec = ref('reality')
const demoSniff = ref(['http', 'tls', 'quic'])
const demoShortIds = ref('6ba85179e30d4fc2\n2f\n4b0c9e71\na1b2c3d4e5')
const demoManyClients = Array.from({ length: 30 }, (_, i) => ({ email: `user-${String(i + 1).padStart(3, '0')}`, id: `0740d687-743b-4269-ad61-${String(i).padStart(12, '0')}` }))
const demoDate = ref(0)
const plainColumns = [{ key: 'ext', label: 'Маршрутизация (ext:…)' }, { key: 'at', label: 'Обновлено' }, { key: 'act', label: 'Действия', fit: true }]
const plainRows = [
  { ext: 'ext:geosite_ads.dat:tag', at: '2026-10-09 14:32:05', busy: false },
  { ext: 'ext:geosite_corp.dat:tag', at: '—', busy: true },
]
const noPlainRows: typeof plainRows = []
function moveDemo(from: number, to: number) {
  demoRules.value.splice(to, 0, demoRules.value.splice(from, 1)[0]!)
}
const editorText = ref(`{
  "log": {
    "access": "none",
    "dnsLog": false
    "loglevel": "warning"
  },
  "routing": {
    "domainStrategy": "AsIs",
    "rules": [{ "type": "field", "outboundTag": "blocked", "ip": ["geoip:private"] }]
  }
}`)
const xrayLog = ref('')
const xrayBusy = ref(false)
const xrayErrorOpen = ref(false)
const xrayStart = `Failed to start: main: failed to load config files: [bin/config.json] > infra/conf: failed to build outbound config with tag nordvpn-de
exit status 23`
const setVals = ref({ port: 2054, session: 30, json: true, path: '/panel-x7/', rules: 'geosite:category-ads-all,geosite:ru,geoip:private,geoip:ru', tags: ['inbound-443'], frag: true })
const setInbounds = ['inbound-443', 'inbound-8443', 'vless-reality', 'trojan-ws'].map((v) => ({ value: v, label: v }))
const busyOpen = ref(false)
const restartAsk = ref(false)
const deleteAsk = ref(false)
const setTheme = ref('dark')
const demoCode = ref('')
const demoTfa = ref(false)
const demoParts = ref(['Inbound', 'Email'])
const demoSep = ref('-')
const demoTags = ref(['geoip:private'])
watch(busyOpen, (on) => on && setTimeout(() => (busyOpen.value = false), 3000))
const typeChecks = ref({ direct: true, blocked: true, proxy: false })
const tableColumns = [
  { key: 'time', label: 'Дата', width: 110, dim: true },
  { key: 'to', label: 'Куда' },
  { key: 'out', label: 'Исходящий', width: 120 },
  { key: 'email', label: 'Email', width: 100, strong: true },
]
const tableRows = [
  { time: '20.09 23:58:58', to: 'wa.me:443', out: 'direct', email: 'alice', tone: undefined },
  { time: '20.09 23:58:56', to: 'cdn.jsdelivr.net:443', out: 'proxy-de', email: 'bob', tone: 'upload' as const },
  { time: '20.09 23:58:50', to: 'ads.doubleclick.net:443', out: 'blocked', email: 'carol', tone: 'err' as const },
]
const pickItems = ['v26.4.25', 'v26.3.27', 'v25.12.8', 'v25.11.11'].map((v, i) => ({ key: v, label: v, current: i === 0 }))
const sectionOpen = ref(true)
// The parts the Outbounds, Balancer and DNS tabs added.
const demoIps = ref(['0.0.0.0/0', '::/0'])
const demoDoh = 'https://1.1.1.1/dns-query'
// The parts the Inbounds list added.
const demoInbPage = ref(1)
const demoInbSize = ref(25)
const demoInbOpen = ref<(string | number)[]>([1])
const demoInbOn = ref(true)
const demoInbRows = [
  { id: 1, remark: 'Нидерланды · Reality', port: 443, on: true },
  { id: 2, remark: 'vmess-grpc-legacy', port: 2053, on: false },
]
const demoInbColumns = [
  { key: 'remark', label: 'Примечание', muted: 'soft' as const },
  { key: 'port', label: 'Порт', size: '56px' },
]
const demoInbPhone = [
  { key: 'remark', label: 'Примечание' },
  { key: 'port', label: 'Порт', size: '76px', align: 'end' as const },
]
const demoEmails = [{ email: 'alice@family', comment: 'Мама, iPhone' }, { email: 'boris' }, { email: 'dasha.m', comment: 'Дача' }]
const demoInnerColumns = [
  { key: 'acts', label: 'Меню', size: '64px' },
  { key: 'email', label: 'Клиент' },
]
const demoInnerRows = [{ email: 'alice@family' }, { email: 'boris' }]
const demoInbMenu = computed<MenuItem[]>(() => [
  { key: 'edit', label: 'Изменить', icon: 'edit' },
  { key: 'enable', label: 'Включить', switch: true, checked: demoInbOn.value, divided: true },
])
const demoFragment = ref(true)
const demoMux = ref(false)
const demoMtu = ref<number | null>(1420)
const demoObserver = ref<'observatory' | 'burstObservatory'>('observatory')
const demoObserverText = ref(JSON.stringify({ observatory: { subjectSelector: ['vless-reality-fi', 'xhttp-nl'], probeUrl: 'https://www.google.com/generate_204', probeInterval: '10s' } }, null, 2))
const demoSelectors = ref(['warp', 'hy2-*'])
const demoOutboundTags = [{ options: ['direct', 'warp', 'nordvpn-de', 'xhttp-nl'].map((v) => ({ value: v, label: v })) }]
const demoDnsOpen = ref(true)
const demoDnsEmptyOpen = ref(true)
const demoServers = [
  { address: '1.1.1.1', domains: [] as string[] },
  { address: 'https://dns.google/dns-query', domains: ['geosite:google', 'geosite:youtube', 'geosite:openai'] },
]
const demoServerColumns = [
  { key: 'address', label: 'Адрес', size: '1.3fr', phone: 'head' as const },
  { key: 'domains', label: 'Домены', phone: 'half' as const },
]
const sw = ref({ a: false, b: true, loading: false })

const menuItems: MenuItem[] = [
  { key: 'edit', label: 'Изменить', icon: 'edit' },
  { key: 'refresh', label: 'Сбросить трафик', icon: 'refresh' },
  { key: 'export', label: 'Экспорт ссылок', icon: 'arrowRight', disabled: true },
  { key: 'delete', label: 'Удалить', icon: 'delete', danger: true, divided: true },
]

function applySwitch() {
  sw.value.loading = true
  setTimeout(() => (sw.value.loading = false), 1200)
}

const swatches = {
  'Поверхности': ['bg', 'side', 'card', 'nest', 'dlg', 'field', 'ctrl', 'chip', 'seg', 'track'],
  'Текст': ['fg', 'fg2', 'mut', 'dim', 'faint'],
  'Акцент': ['acc', 'acc-t', 'acc-hover', 'acc-press', 'acc-soft', 'acc-ink'],
  'Статусы': ['ok', 'warn', 'err', 'unl', 'on', 'off', 'mod'],
}
const radii = [4, 6, 8, 10, 12, 16]
const fontSizes = [10, 11, 12, 13, 14, 16, 18, 20, 24]
const spacing = [2, 4, 6, 8, 10, 12, 14, 16, 20, 24, 32, 40, 48, 56]
</script>

<template>
  <div class="kit">
    <header class="kit__top">
      <div>
        <h1 class="kit__title">space-ui · kit</h1>
        <p class="kit__sub">Компоненты v3 во всех состояниях. Ширина окна меньше 768 px показывает мобильные размеры.</p>
      </div>
      <XSegmented
        v-model="theme"
        label="Тема"
        :options="[{ value: 'dark', label: 'Тёмная', icon: 'moon' }, { value: 'light', label: 'Светлая', icon: 'sun' }]"
      />
    </header>

    <!-- Tokens -->
    <XCard title="Токены">
      <div class="kit__tokens">
        <div v-for="(names, group) in swatches" :key="group" class="kit__swatch-group">
          <h3 class="kit__h3">{{ group }}</h3>
          <div class="kit__swatches">
            <div v-for="n in names" :key="n" class="kit__swatch">
              <span class="kit__swatch-color" :style="{ background: `var(--${n})` }" />
              <code>--{{ n }}</code>
            </div>
          </div>
        </div>
        <div class="kit__swatch-group">
          <h3 class="kit__h3">Радиусы</h3>
          <div class="kit__swatches">
            <div v-for="r in radii" :key="r" class="kit__radius" :style="{ borderRadius: `var(--r-${r})` }">{{ r }}</div>
          </div>
        </div>
        <div class="kit__swatch-group">
          <h3 class="kit__h3">Размеры шрифта</h3>
          <div class="kit__fonts">
            <span v-for="f in fontSizes" :key="f" :style="{ fontSize: `var(--fs-${f})` }">{{ f }} Подключения</span>
            <code class="kit__mono">IBM Plex Mono · 203.0.113.7:443 · 48,2 МБ</code>
          </div>
        </div>
        <div class="kit__swatch-group">
          <h3 class="kit__h3">Отступы</h3>
          <div class="kit__spacing">
            <div v-for="s in spacing" :key="s" class="kit__space">
              <span :style="{ width: `var(--sp-${s})` }" />
              <code>{{ s }}</code>
            </div>
          </div>
        </div>
      </div>
    </XCard>

    <!-- States -->
    <XCard title="Состояния" flush>
      <div class="kit__table-wrap">
        <table class="kit__table">
          <thead>
            <tr>
              <th>Компонент</th>
              <th v-for="s in states" :key="s.key">{{ s.label }}</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <th><b>Button</b><code>primary</code></th>
              <td v-for="s in states" :key="s.key">
                <XButton v-if="!has(s.key, 'invalid', 'modified')" variant="primary" icon="plus" :class="s.cls" :disabled="s.key === 'disabled'">Создать</XButton>
                <XButton v-else-if="s.key === 'invalid'" variant="primary" loading>Сохранение…</XButton>
              </td>
            </tr>
            <tr>
              <th><b>Button</b><code>secondary</code></th>
              <td v-for="s in states" :key="s.key">
                <XButton v-if="!has(s.key, 'invalid', 'modified')" icon="globe" :class="s.cls" :disabled="s.key === 'disabled'">Geo-файлы</XButton>
              </td>
            </tr>
            <tr>
              <th><b>Button</b><code>soft</code></th>
              <td v-for="s in states" :key="s.key">
                <XButton v-if="!has(s.key, 'invalid', 'modified')" variant="soft" :class="s.cls" :disabled="s.key === 'disabled'">Подтвердить</XButton>
              </td>
            </tr>
            <tr>
              <th><b>Button</b><code>ok</code></th>
              <td v-for="s in states" :key="s.key">
                <XButton v-if="!has(s.key, 'invalid', 'modified')" variant="ok" size="sm" :class="s.cls" :disabled="s.key === 'disabled'">Обновить панель: 3.0.1</XButton>
              </td>
            </tr>
            <tr>
              <th><b>Button</b><code>danger</code></th>
              <td v-for="s in states" :key="s.key">
                <XButton v-if="!has(s.key, 'invalid', 'modified')" variant="danger" :class="s.cls" :disabled="s.key === 'disabled'">Сбросить</XButton>
              </td>
            </tr>
            <tr>
              <th><b>Button</b><code>icon · ghost · sm</code></th>
              <td v-for="s in states" :key="s.key">
                <div v-if="!has(s.key, 'invalid', 'modified')" class="kit__row">
                  <XButton icon="refresh" label="Обновить" :class="s.cls" :disabled="s.key === 'disabled'" />
                  <XButton variant="ghost" size="sm" icon="dots" label="Ещё" :class="s.cls" :disabled="s.key === 'disabled'" />
                  <XButton variant="ghost-danger" size="sm" icon="delete" label="Удалить" :class="s.cls" :disabled="s.key === 'disabled'" />
                </div>
              </td>
            </tr>
            <tr>
              <th><b>Input</b><code>text</code></th>
              <td v-for="s in states" :key="s.key">
                <XInput
                  :model-value="s.key === 'invalid' ? 'panel' : 'Germany-VLESS'"
                  :class="s.cls"
                  :disabled="s.key === 'disabled'"
                  :invalid="s.key === 'invalid'"
                  :modified="s.key === 'modified'"
                />
              </td>
            </tr>
            <tr>
              <th><b>Input</b><code>mono · search</code></th>
              <td v-for="s in states" :key="s.key">
                <XInput
                  mono
                  type="search"
                  icon="search"
                  placeholder="Поиск"
                  :model-value="s.key === 'focus' ? 'cdn' : ''"
                  :class="s.cls"
                  :disabled="s.key === 'disabled'"
                  :invalid="s.key === 'invalid'"
                  :modified="s.key === 'modified'"
                />
              </td>
            </tr>
            <tr>
              <th><b>Password</b></th>
              <td v-for="s in states" :key="s.key">
                <XInput
                  type="password"
                  model-value="secret-value"
                  :class="s.cls"
                  :disabled="s.key === 'disabled'"
                  :invalid="s.key === 'invalid'"
                  :modified="s.key === 'modified'"
                />
              </td>
            </tr>
            <tr>
              <th><b>Number</b><code>unit</code></th>
              <td v-for="s in states" :key="s.key">
                <XNumber
                  :model-value="s.key === 'invalid' ? 70000 : 2053"
                  unit="порт"
                  :class="s.cls"
                  :disabled="s.key === 'disabled'"
                  :invalid="s.key === 'invalid'"
                  :modified="s.key === 'modified'"
                />
              </td>
            </tr>
            <tr>
              <th><b>Select</b></th>
              <td v-for="s in states" :key="s.key">
                <XSelect
                  :options="languages"
                  model-value="ru"
                  :class="s.cls"
                  :disabled="s.key === 'disabled'"
                  :invalid="s.key === 'invalid'"
                  :modified="s.key === 'modified'"
                />
              </td>
            </tr>
            <tr>
              <th><b>Tag input</b></th>
              <td v-for="s in states" :key="s.key">
                <XTagInput
                  :model-value="['geoip:private', 'geoip:ru']"
                  mono
                  :class="s.cls"
                  :disabled="s.key === 'disabled'"
                  :invalid="s.key === 'invalid'"
                  :modified="s.key === 'modified'"
                />
              </td>
            </tr>
            <tr>
              <th><b>Textarea</b></th>
              <td v-for="s in states" :key="s.key">
                <XTextarea
                  :model-value="s.key === 'invalid' ? badRouting : 'Текст объявления'"
                  :rows="2"
                  :class="s.cls"
                  :disabled="s.key === 'disabled'"
                  :invalid="s.key === 'invalid'"
                  :modified="s.key === 'modified'"
                />
              </td>
            </tr>
            <tr>
              <th><b>Switch</b><code>38×22 · моб. 44×26</code></th>
              <td v-for="s in states" :key="s.key">
                <div class="kit__row">
                  <XSwitch
                    :model-value="s.key !== 'invalid'"
                    :class="s.cls"
                    :disabled="s.key === 'disabled'"
                    :invalid="s.key === 'invalid'"
                    :modified="s.key === 'modified'"
                    label="Включить"
                  />
                  <XSwitch v-if="!has(s.key, 'invalid', 'modified')" :class="s.cls" :disabled="s.key === 'disabled'" label="Выключено" />
                </div>
              </td>
            </tr>
            <tr>
              <th><b>Checkbox</b></th>
              <td v-for="s in states" :key="s.key">
                <div v-if="s.key !== 'modified'" class="kit__row">
                  <XCheckbox :model-value="true" :class="s.cls" :disabled="s.key === 'disabled'" :invalid="s.key === 'invalid'" />
                  <XCheckbox :class="s.cls" :disabled="s.key === 'disabled'" :invalid="s.key === 'invalid'" />
                  <XCheckbox indeterminate :class="s.cls" :disabled="s.key === 'disabled'" />
                </div>
              </td>
            </tr>
            <tr>
              <th><b>Section</b><code>заголовок</code></th>
              <td v-for="s in states" :key="s.key">
                <XSection v-if="!has(s.key, 'disabled', 'invalid', 'modified')" title="Сертификаты" :open="false" :class="s.cls" />
              </td>
            </tr>
            <tr>
              <th><b>Chip</b><code>interactive</code></th>
              <td v-for="s in states" :key="s.key">
                <XChip v-if="!has(s.key, 'disabled', 'invalid', 'modified')" tone="ok" mono interactive :class="s.cls">45,2 ГБ / 100 ГБ</XChip>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </XCard>

    <!-- Variants -->
    <XCard title="Варианты">
      <div class="kit__variants">
        <div class="kit__variant">
          <h3 class="kit__h3">Chip · статусы</h3>
          <div class="kit__row kit__row--wrap">
            <XChip tone="ok" icon="user" mono>10</XChip>
            <XChip tone="off" mono>1</XChip>
            <XChip tone="err" mono>2</XChip>
            <XChip tone="warn" mono>2</XChip>
            <XChip tone="ok" dot mono>4</XChip>
            <XChip tone="unl" icon="infinity" mono>3</XChip>
            <XChip tone="neutral" mono>12 ГБ / ∞</XChip>
          </div>
        </div>
        <div class="kit__variant">
          <h3 class="kit__h3">Chip · срок</h3>
          <div class="kit__row kit__row--wrap">
            <XChip tone="ok">через 41 д</XChip>
            <XChip tone="warn">через 3 д</XChip>
            <XChip tone="err">5 д назад</XChip>
            <XChip tone="off">через 60 д</XChip>
            <XChip tone="ok" dot>Онлайн</XChip>
            <XChip>Офлайн</XChip>
          </div>
        </div>
        <div class="kit__variant">
          <h3 class="kit__h3">Usage pill</h3>
          <div class="kit__stack">
            <XUsagePill used="18,4 ГБ" limit="50 ГБ" :ratio="0.37" />
            <XUsagePill used="46,8 ГБ" limit="50 ГБ" :ratio="0.94" tone="warn" />
            <XUsagePill used="50 ГБ" limit="50 ГБ" :ratio="1" tone="err" exhausted />
            <XUsagePill used="0 Б" limit="30 ГБ" :ratio="0" disabled />
            <XUsagePill used="100,5 ГБ" limit="∞" :ratio="null" />
            <XUsagePill used="через 9 д" limit="30 д" :ratio="0.7" textual />
            <div style="width: 210px"><XUsagePill used="14,2 ГБ" limit="100 ГБ" :ratio="0.14" block interactive /></div>
            <div style="width: 268px"><XUsagePill used="14,2 ГБ" limit="100 ГБ" :ratio="0.14" block large /></div>
          </div>
        </div>
        <div class="kit__variant">
          <h3 class="kit__h3">Status dot · badge · counter</h3>
          <div class="kit__row kit__row--wrap">
            <XStatusDot tone="ok" label="ok" />
            <XStatusDot tone="warn" label="warn" />
            <XStatusDot tone="err" label="err" />
            <XStatusDot tone="unl" label="unlimited" />
            <XStatusDot tone="off" label="off" />
            <XStatusDot tone="ok" pulse label="online" />
            <XStatusDot tone="err" blink label="blink" />
            <XStatusDot size="sm" blink label="sm" />
            <XStatusDot tone="err" label="Исчерпано" tip />
            <XStatusDot tone="ok" pulse label="Онлайн" tip />
          </div>
          <div class="kit__row kit__row--wrap">
            <XBadge mono>vless</XBadge>
            <XBadge mono>shadowsocks</XBadge>
            <XBadge kind="outline" mono>xhttp</XBadge>
            <XBadge kind="outline" mono>grpc</XBadge>
            <XBadge kind="accent" mono>reality</XBadge>
            <XBadge kind="accent">NEW</XBadge>
            <XCounter>12</XCounter>
            <XCounter accent>3</XCounter>
          </div>
        </div>
        <div class="kit__variant">
          <h3 class="kit__h3">Segmented · pager</h3>
          <div class="kit__row kit__row--wrap">
            <XSegmented
              v-model="view"
              label="Фильтр"
              :options="[{ value: 'all', label: 'Все' }, { value: 'online', label: 'Онлайн' }, { value: 'depleted', label: 'Исчерпано' }]"
            />
            <XSegmented
              v-model="mode"
              icon-only
              label="Режим"
              :options="[{ value: 'search', label: 'Поиск', icon: 'search' }, { value: 'filter', label: 'Фильтр', icon: 'filter' }]"
            />
          </div>
          <XPager v-model="page" :pages="3" />
          <XPager v-model="page" :pages="45" />
        </div>
        <div class="kit__variant">
          <h3 class="kit__h3">Notice</h3>
          <div class="kit__stack">
            <XNotice tone="info">Изменения применятся после перезапуска Xray.</XNotice>
            <XNotice tone="warn" title="Сертификат истекает">Осталось 6 дней. Продлите его до 18 октября.</XNotice>
            <XNotice tone="err" title="Ошибка при запуске Xray">failed to listen on 0.0.0.0:443: address already in use</XNotice>
            <XNotice tone="ok">Настройки сохранены.</XNotice>
            <XFieldNote tone="info">В правилах маршрутизации используйте значение как ext:файл.dat:тег (замените тег).</XFieldNote>
            <XNotice tone="warn" title="Соединение не защищено (HTTP)" compact>Не вводите конфиденциальные данные</XNotice>
          </div>
        </div>
        <div class="kit__variant">
          <h3 class="kit__h3">Кнопки с меню · кнопка-чекбокс · подпись</h3>
          <div class="kit__row kit__row--wrap">
            <XButton caret><span class="x-caps">Строк</span> 100</XButton>
            <XButton size="sm" mono caret>за 2 мин</XButton>
            <XCheckbox v-model="boxedCheck" boxed>SysLog</XCheckbox>
            <XCheckbox v-model="typeChecks.direct" boxed tone="ok">Direct</XCheckbox>
            <XCheckbox v-model="typeChecks.blocked" boxed tone="err">Blocked</XCheckbox>
            <XCheckbox v-model="typeChecks.proxy" boxed tone="upload">Proxy</XCheckbox>
            <XButton icon="download" phone-icon-only>Скачать</XButton>
            <XButton variant="soft" size="sm" phone-block>Обновить панель</XButton>
            <XButton variant="primary" :success="successDemo" @click="successDemo = !successDemo">Успех</XButton>
          </div>
          <div class="kit__row kit__row--wrap">
            <span class="x-caps">Подпись 10</span>
            <span class="x-caps x-caps--lg">Подпись 11</span>
          </div>
        </div>
        <div class="kit__variant">
          <h3 class="kit__h3">Группа кнопок · круглая кнопка · подтверждение · результат теста</h3>
          <div class="kit__row kit__row--wrap">
            <XButtonGroup label="Трафик">
              <XButton icon="refresh" label="Обновить" />
              <XPopConfirm v-slot="{ props }" text="Вы уверены, что хотите сбросить трафик?" cta="Сбросить">
                <XButton v-bind="props" icon="reset" label="Сбросить трафик" />
              </XPopConfirm>
            </XButtonGroup>
            <XChip mono>vless</XChip>
            <XChip mono outline>xhttp</XChip>
            <XChip tone="ok" mono outline>reality</XChip>
          </div>
          <XKeyValue :rows="[{ label: 'Access Token', value: 'fb4d5e78-31cd-44c5-ad60-0084c5af7c2a' }, { label: 'Device ID', value: 't.73479747-81ad-4783-ac51-9638e62ddb34' }]" />
          <XDivider label="Статус исходящего подключения" />
          <XSelect v-model="demoCountry" :options="[{ value: 1, label: 'Georgia (GE)' }, { value: 2, label: 'Germany (DE)' }, { value: 3, label: 'Netherlands (NL)' }]" search placeholder="Страна" />
          <XSelect v-model="demoProto" :options="[{ value: 'vless', label: 'vless' }, { value: 'mtproto', label: 'mtproto', disabled: true, note: 'Подключение mtproto может быть только одно' }]" mono />
          <XDateTime v-model="demoDate" />
          <div class="kit__row kit__row--wrap">
            <XTestResult state="idle" />
            <XTestResult state="testing" />
            <XTestResult state="ok" :delay="124" :status="204" />
            <XTestResult state="fail" error="dial udp 185.130.184.31:51820: i/o timeout" />
          </div>
        </div>
        <div class="kit__variant">
          <h3 class="kit__h3">Строка-ссылка · пункт меню</h3>
          <div class="kit__stack" style="max-width: 280px; padding: 0 12px">
            <XRowLink icon="doc" label="Журнал панели" />
            <XRowLink icon="database" label="Резервная копия" />
          </div>
          <XNav label="Меню" style="max-width: 236px">
            <XNavItem icon="dashboard" label="Дашборд" to="/kit" />
            <XNavItem icon="sliders" label="Настройки" />
            <XNavItem icon="logout" label="Выход" danger />
          </XNav>
        </div>
        <div class="kit__variant">
          <h3 class="kit__h3">Список действий</h3>
          <XActionList style="max-width: 400px">
            <XActionRow title="Экспорт базы данных" icon="download">Скачать файл <code>.db</code> с резервной копией.</XActionRow>
            <XActionRow title="Импорт базы данных" icon="upload" loading>Загрузить файл <code>.db</code> с устройства.</XActionRow>
          </XActionList>
        </div>
        <div class="kit__variant">
          <h3 class="kit__h3">Пусто · ошибка</h3>
          <XEmpty title="Нет записей" hint="Для выбранного уровня и глубины журнал пуст" />
          <XErrorBox text="failed to listen on 0.0.0.0:443&#10;bind: address already in use" />
        </div>
        <div class="kit__variant">
          <h3 class="kit__h3">Таблица журнала · карточки на телефоне</h3>
          <XDataTable :columns="tableColumns" :rows="tableRows" :row-key="(r) => r.to" :row-tone="(r) => r.tone" :height="140" />
          <div class="kit__stack" style="max-width: 360px">
            <XRecordCard v-for="r in tableRows" :key="r.to" :title="r.to" time="23:58:58" name="alice" detail="inbound-443" :badge="r.out" :tone="r.tone" />
            <XRecordCard loading />
          </div>
        </div>
        <div class="kit__variant">
          <h3 class="kit__h3">Текст с номерами строк</h3>
          <XCodeArea v-model="codeText" :height="160" label="config.json" />
        </div>
        <div class="kit__variant">
          <h3 class="kit__h3">Строка значения</h3>
          <div class="kit__stack" style="max-width: 400px">
            <XInfoRow label="Текущая версия">3.0.0</XInfoRow>
            <XInfoRow label="Последняя версия" accent><XStatusDot size="sm" blink />3.0.1</XInfoRow>
            <XInfoRow label="Панель обновлена" accent><XChip tone="ok">Обновлено</XChip></XInfoRow>
          </div>
        </div>
        <div class="kit__variant">
          <h3 class="kit__h3">Список выбора · загрузка</h3>
          <XPickList :items="pickItems" current-label="Установлена" style="max-width: 400px" />
          <XPickList :items="[]" loading style="max-width: 400px" />
        </div>
        <div class="kit__variant">
          <h3 class="kit__h3">Полоса прокрутки</h3>
          <div class="kit__scroll" tabindex="0">
            <div v-for="n in 12" :key="n" class="mono">{{ String(n).padStart(2, '0') }} · 2026/10/14 12:00:{{ String(n).padStart(2, '0') }} INFO - XRAY: accepted tcp:www.example.com:443 [inbound-443 -> direct]</div>
          </div>
        </div>
      </div>
    </XCard>

    <!-- Live -->
    <XCard title="Страница настроек">
      <div class="kit__stack">
        <XTabs v-model="setTab" :items="setTabs" label="Разделы" />
        <XNotice tone="warn" title="Предупреждение системы безопасности" :items="['Подключение к панели не защищено. Установите SSL-сертификат.', 'Порт панели по умолчанию небезопасен.']" closable>
          Некоторые настройки уязвимы.
        </XNotice>
        <div class="kit__row kit__row--wrap">
          <XStatusText tone="mod">Не сохранено: 2</XStatusText>
          <XStatusText tone="err">Исправьте ошибки: 1 · не сохранено: 2</XStatusText>
          <XButton @click="busyOpen = true">Перезапуск: оверлей на 3 с</XButton>
          <XButton icon="refresh" @click="restartAsk = true">Подтверждение перезапуска</XButton>
          <XButton icon="delete" @click="deleteAsk = true">Подтверждение удаления</XButton>
        </div>
        <XSettingsSection v-model:open="setOpen" title="Основное" :count="6" :modified="2" :errors="1" note="IP по умолчанию">
          <XSettingRow title="Порт панели" description="Порт, на котором работает панель" narrow modified>
            <XNumber v-model="setVals.port" :min="1" :max="65535" modified />
          </XSettingRow>
          <XSettingRow title="Продолжительность сессии" description="Значение в минутах" narrow error="Значение должно быть не меньше 60">
            <XNumber v-model="setVals.session" unit="мин" invalid />
          </XSettingRow>
          <XSettingRow title="Включить JSON-подписку" description="Отдельный адрес с подпиской в формате JSON" inline>
            <XSwitch v-model="setVals.json" label="JSON" />
          </XSettingRow>
          <XSettingDivider label="Расширенный шаблон" />
          <XSettingRow title="Корневой путь URL-адреса панели" description="Должен начинаться и заканчиваться символом /" modified>
            <XInput v-model="setVals.path" mono modified />
            <template #note><XFieldNote>После перезапуска: <code>http://127.0.0.1:2053/panel-x7Q2/</code></XFieldNote></template>
          </XSettingRow>
          <XSettingRow title="Тема" compact>
            <XSegmented v-model="setTheme" :options="[{ value: 'light', label: 'Светлая', icon: 'sun' }, { value: 'dark', label: 'Тёмная', icon: 'moon' }]" block label="Тема" />
          </XSettingRow>
          <XSettingRow title="Правила маршрутизации" description="Длинное значение: растёт до трёх строк, Enter не переносит" tall>
            <XTextarea v-model="setVals.rules" mono :grow="3" />
          </XSettingRow>
          <XSettingRow title="Подключения для LDAP" description="Новые клиенты попадут в выбранные подключения">
            <XMultiSelect v-model="setVals.tags" :options="setInbounds" mono placeholder="Все подключения" empty="Подключений пока нет" />
          </XSettingRow>
          <XSettingRow title="Фрагментация" inline>
            <XSwitch v-model="setVals.frag" label="Фрагментация" />
            <template v-if="setVals.frag" #nested>
              <XNest title="Параметры фрагментации" :columns="140">
                <XField v-slot="f" label="Packets"><XInput v-bind="f" model-value="tlshello" mono /></XField>
                <XField v-slot="f" label="Length"><XInput v-bind="f" model-value="100-200" mono /></XField>
                <XField v-slot="f" label="Interval"><XInput v-bind="f" model-value="10-20" mono /></XField>
              </XNest>
            </template>
          </XSettingRow>
        </XSettingsSection>
        <XActionBar style="position: static; border-radius: 12px">
          <template #state><XStatusText tone="mod">Не сохранено: 2</XStatusText></template>
          <XButton variant="primary">Сохранить</XButton>
          <XButton icon="refresh" disabled>Перезапуск</XButton>
        </XActionBar>
        <XSettingsSection title="Отдельная форма и 2FA" :count="2" note="Отдельная форма">
          <XSettingBlock>
            <div class="kit__row kit__row--wrap">
              <XCodeInput v-model="demoCode" label="Код" style="width: 260px" />
              <XQrCode value="otpauth://totp/Space%20UI:Administrator?secret=JBSWY3DPEHPK3PXP" copy="JBSWY3DPEHPK3PXP" copy-label="Скопировать токен" :size="120" />
            </div>
            <template #note>
              <XFieldNote tone="err">Неверное имя пользователя или пароль</XFieldNote>
              <span>Применяется кнопкой «Подтвердить», отдельно от «Сохранить».</span>
            </template>
            <template #actions><XButton variant="soft" icon="lock">Подтвердить</XButton></template>
          </XSettingBlock>
          <XSettingRow title="Модель примечания" description="Части по порядку, разделитель — сеткой символов" tall>
            <div class="kit__remark">
              <XExample label="Пример примечания:" :value="'#' + demoParts.join(demoSep)" />
              <div class="kit__remark-line">
                <XOrderToggles v-model="demoParts" :options="[{ value: 'Inbound', label: 'Inbound' }, { value: 'Email', label: 'Email' }, { value: 'Other', label: 'Other' }]" mono />
                <div class="kit__remark-separator">
                  <XSelect v-model="demoSep" :options="[' ', '-', '_', '@', ':', '~', '|', ',', '.', '/'].map((v) => ({ value: v, label: v === ' ' ? 'пробел' : v, glyph: v === ' ' ? '␣' : undefined }))" :columns="5" mono />
                </div>
              </div>
            </div>
          </XSettingRow>
          <XSettingRow title="Включить 2FA" description="Подпись состояния слева от переключателя" inline>
            <XSwitch v-model="demoTfa" :text="demoTfa ? 'Включена' : 'Выключена'" label="2FA" />
          </XSettingRow>
        </XSettingsSection>
        <XNest title="Список внутри блока">
          <div style="display: flex; flex-direction: column; gap: 10px">
            <XNestCard title="Noise №1" modified :columns="130">
              <template #actions><XButton variant="ghost-danger" size="sm" icon="delete">Удалить</XButton></template>
              <XField v-slot="a" label="Packet"><XInput v-bind="a" model-value="10-20" mono /></XField>
              <XField v-slot="a" label="Delay (ms)"><XInput v-bind="a" model-value="10-16" mono /></XField>
            </XNestCard>
            <div><XButton variant="dashed" icon="plus">Добавить Noise</XButton></div>
            <XTagInput v-model="demoTags" :presets="[{ value: 'geoip:private', label: 'Private IP' }, { value: 'geoip:ru', label: '🇷🇺 Russia' }]" presets-title="Пресеты" placeholder="Выберите пресет или введите значение" more-placeholder="Добавить…" />
          </div>
        </XNest>
        <XSettingsSkeleton :rows="[2]" />
        <XCard>
          <XEmpty tone="err" icon="alert" title="Произошла ошибка при получении параметров." />
        </XCard>
      </div>
    </XCard>

    <XCard title="Настройки Xray">
      <div class="kit__stack">
        <XTabs model-value="general" :items="[{ key: 'general', label: 'Основное' }, { key: 'routing', label: 'Маршрутизация' }]" label="Разделы, пока грузятся" disabled />
        <XNotice tone="err" title="Предупреждение системы безопасности" closable>Соединение не защищено. Не вводите конфиденциальные данные до установки SSL-сертификата.</XNotice>
        <div class="kit__row kit__row--wrap">
          <XProcessError v-model:open="xrayErrorOpen" title="Ошибка при запуске Xray" :text="xrayStart" />
          <XButton variant="primary" icon="refresh" :loading="xrayBusy" @click="xrayBusy = !xrayBusy">Перезапуск Xray</XButton>
          <span>Линия под вкладками во время перезапуска:</span>
        </div>
        <XProgressLine v-if="xrayBusy" />
        <XSettingsSection v-model:open="xrayOpen" title="Логи" :count="2">
          <XSettingRow title="Логи ошибок" description="«Empty» в списке — обычным шрифтом курсивом">
            <XSelect v-model="xrayLog" :options="[{ value: '', label: 'Empty', plain: true }, { value: 'none', label: 'none' }, { value: './error.log', label: './error.log' }]" mono />
          </XSettingRow>
        </XSettingsSection>
        <XSettingsSection title="Восстановить настройки по умолчанию" padded>
          <XButton variant="danger" icon="reset">Восстановить настройки по умолчанию</XButton>
        </XSettingsSection>
        <XActionBar style="position: static; border-radius: 12px">
          <template #state><XStatusText tone="mod">Не сохранено: 1</XStatusText></template>
          <XButton variant="primary">Сохранить</XButton>
          <XButton variant="primary" icon="refresh" pending>Перезапуск Xray</XButton>
          <XProcessError title="Ошибка при запуске Xray" :text="xrayStart" />
        </XActionBar>
        <XActionBar style="position: static; border-radius: 12px">
          <XButton variant="primary" disabled>Сохранить</XButton>
          <XButton variant="primary" icon="refresh">Перезапуск Xray</XButton>
        </XActionBar>
        <XField v-slot="a" label="Domain" help="Элементы, разделённые запятыми">
          <XTagInput v-bind="a" v-model="demoDomains" :groups="demoCatalog" mono placeholder="e.g. google.com, geosite:cn" />
        </XField>
        <XPairList v-model="demoAttrs" title="Attributes" name-label="Имя" value-label="Значение" mono />
        <XGridTable
          :columns="demoColumns"
          :rows="demoRules"
          :row-key="(r) => r.to"
          :blank="(r, key) => key === 'dest' && !r.dest.length"
          :menu="() => [{ key: 'edit', label: 'Изменить', icon: 'edit' }, { key: 'delete', label: 'Удалить', icon: 'delete', danger: true }]"
          sortable
          numbered
          @move="moveDemo"
        >
          <template #cell-dest="{ row }"><XValueChip label="Domain" :values="row.dest" /></template>
          <template #cell-to="{ row }"><XRouteTarget :tag="row.to" :balancer="row.bal" /></template>
        </XGridTable>
        <XSection title="Пользовательские GeoSite / GeoIP" card>
          <XPlainTable :columns="plainColumns" :rows="plainRows" :row-key="(r) => r.ext" :active="(r) => !!r.busy">
            <template #cell-ext="{ row }"><code>{{ row.ext }}</code></template>
            <template #cell-at="{ row }">{{ row.at }}</template>
            <template #cell-act="{ row }">
              <XInlineButton icon="edit" label="Изменить" />
              <XInlineButton icon="refresh" label="Обновить сейчас" :loading="row.busy" />
              <XInlineButton icon="delete" label="Удалить" danger />
            </template>
          </XPlainTable>
          <XPlainTable :columns="plainColumns" :rows="noPlainRows" :row-key="(r) => r.ext" />
        </XSection>
        <XPartHead title="Транспорт" />
        <XFieldGrid>
          <XField v-slot="a" label="Тип транспорта"><XSelect v-bind="a" v-model="demoNet" :options="[{ value: 'tcp', label: 'TCP (RAW)' }, { value: 'xhttp', label: 'XHTTP' }]" /></XField>
        </XFieldGrid>
        <XPartHead title="Безопасность" />
        <XFieldGrid>
          <XSegmented v-model="demoSec" :options="[{ value: 'none', label: 'Пусто' }, { value: 'reality', label: 'Reality' }, { value: 'tls', label: 'TLS' }]" label="Безопасность" block />
        </XFieldGrid>
        <XFieldGrid top>
          <XField v-slot="a" label="Short IDs (сетка top)"><XTextarea v-bind="a" v-model="demoShortIds" mono :rows="4" /></XField>
          <XField v-slot="a" label="SpiderX"><XInput v-bind="a" model-value="/" mono /></XField>
        </XFieldGrid>
        <XButtonRow>
          <XButton size="sm" variant="primary">Get New Cert</XButton>
          <XButton size="sm" variant="danger">Clear</XButton>
        </XButtonRow>
        <XStateBar>Account is Expired And Disabled</XStateBar>
        <XFactRow label="Протокол"><XChip tone="unl" mono>vless</XChip></XFactRow>
        <XFactRow label="Хост"><XChip tone="warn">Пусто</XChip></XFactRow>
        <XFactRow label="ID">
          <XChip mono truncate>0740d687-743b-4269-ad61-3b98734c039b-0740d687-743b-4269</XChip>
          <XCopyChip text="0740d687-743b-4269-ad61-3b98734c039b" />
        </XFactRow>
        <div class="demo-tiles">
          <XFactTile label="Остаток"><XChip mono>68,3 ГБ</XChip></XFactTile>
          <XFactTile label="Общий расход"><XChip tone="warn" mono>100 ГБ</XChip></XFactTile>
          <XFactTile label="Дата окончания"><XChip icon="infinity" aria-label="Бесконечно" /></XFactTile>
        </div>
        <div class="demo-tiles">
          <XQrCard label="Подписка" tone="unl" value="https://de.example.net:2096/sub/06e819f5e0bcf42c" />
          <XQrCard label="Germany-VLESS-alice" value="vless://0740d687-743b-4269-ad61-3b98734c039b@de.example.net:443?type=tcp#Germany-VLESS-alice" />
        </div>
        <XCopyBlock label="Subscription Link" tone="unl" text="https://de.example.net:2096/sub/06e819f5e0bcf42c" />
        <XCopyBlock label="Config" :text="'[Interface]\nPrivateKey = eCtXsJZ27+4PbhDkHnB923tkUn2Gj59wZw5wFA75MnU=\nAddress = 10.0.0.2/32'">
          <template #actions><XButton size="sm" icon="download">Скачать</XButton></template>
        </XCopyBlock>
        <h3 class="kit__h3">Страница подписки</h3>
        <XKeyValue
          columns
          :rows="[{ label: 'ID подписки', value: 'alice-7f3k2' }, { key: 'status', label: 'Статус' }, { label: 'Общий лимит', value: '100 ГБ' }]"
        >
          <template #value-status><XChip tone="ok">Активна</XChip></template>
        </XKeyValue>
        <XCopyFrame label="Germany-VLESS-alice" text="vless://0740d687-743b-4269-ad61-3b98734c039b@de.example.net:443?type=tcp&security=reality#Germany-VLESS-alice" />
        <XButtonRow>
          <XMenu v-slot="{ props }" :items="[{ key: 'v2box', label: 'V2Box' }, { key: 'happ', label: 'Happ' }]" placement="top">
            <XButton v-bind="props" variant="primary" caret>Android</XButton>
          </XMenu>
          <XPopover label="Настройки" placement="bottom-end">
            <template #default="{ props }"><XButton v-bind="props" icon="gear" label="Настройки" /></template>
            <template #content>Поповер у правого края кнопки</template>
          </XPopover>
        </XButtonRow>
        <XChipBox label="Предпросмотр итоговых email">
          <XChip v-for="e in ['alice_7', 'carol_7', 'dmitry_7', 'igor_7', 'nikita_7']" :key="e" mono>{{ e }}</XChip>
        </XChipBox>
        <XButtonRow>
          <XButton size="sm" @click="toast('ok', 'Успешно скопировано: 5')">Тост ok</XButton>
          <XButton size="sm" @click="toast('warn', 'Нечего копировать: ни одного клиента не выбрано или список источника пуст')">Тост warn</XButton>
          <XButton size="sm" @click="toast('err', 'Ошибки при копировании: alice_7 — Email уже существует')">Тост err</XButton>
        </XButtonRow>
        <XField label="Дата окончания" help="Оставьте пустым, чтобы не ограничивать">
          <template #badge><XChip tone="err">Expired</XChip></template>
          <template #default="a"><XInput v-bind="a" model-value="2026-09-30 00:00:00" mono /></template>
        </XField>
        <XDivider spaced />
        <XCheckGroup v-model="demoSniff" :options="['http', 'tls', 'quic', 'fakedns'].map((v) => ({ value: v, label: v }))" mono label="Sniffing" />
        <XSwitch v-model="demoWeb" boxed label="WEB-прокси" help="Те же клиенты получают вторую ссылку — tg://webproxy." />
        <XFieldNote v-if="demoWeb" attached>WEB-прокси работает, только если сама панель открыта по HTTPS на порту 443.</XFieldNote>
        <XNestCard title="Peer 1">
          <template #actions><XInlineButton icon="delete" danger label="Удалить" /></template>
          <XFieldGrid>
            <XField v-slot="a" label="Секретный ключ"><XInput v-bind="a" model-value="qMdjWiazPPeHXa" mono /></XField>
            <XField v-slot="a" label="Публичный ключ"><XInput v-bind="a" model-value="rRjG2BaATgtBz7" mono /></XField>
          </XFieldGrid>
        </XNestCard>
        <XNestCard title="Peer 2">
          <XField v-slot="a" label="Keep Alive"><XInput v-bind="a" model-value="0" mono /></XField>
        </XNestCard>
        <XField v-slot="a" label="Email" help="Пожалуйста, укажите уникальный Email" action="Сбросить" @action="demoEmail = Math.random().toString(36).slice(2, 11)">
          <XInput v-bind="a" v-model="demoEmail" />
        </XField>
        <XPlainTable :columns="[{ key: 'email', label: 'Email' }, { key: 'id', label: 'ID' }]" :rows="demoManyClients" :row-key="(r) => r.email" :height="200" :phone-height="240">
          <template #cell-email="{ row }">{{ row.email }}</template>
          <template #cell-id="{ row }"><code>{{ row.id }}</code></template>
        </XPlainTable>
        <XCard title="Шаблон конфигурации Xray" description="Карточка без линии под заголовком; переключатель прокручивается, если не помещается." seamless>
          <template #actions>
            <XSegmented v-model="editorPart" :options="[{ value: 'all', label: 'Все' }, { value: 'in', label: 'Входящие подключения' }, { value: 'out', label: 'Исходящие подключения' }, { value: 'rules', label: 'Маршрутизация' }]" scroll />
          </template>
          <XCodeEditor v-model="editorText" :height="240" :phone-height="240" label="JSON" />
        </XCard>

        <h3 class="kit__h3">Окна исходящего подключения, балансировщика и DNS</h3>
        <div class="kit__duo">
          <div class="kit__stack">
            <XInputList v-model="demoIps" title="Allowed IPs" mono />
            <XListHead title="Peers" add>
              <XNestCard title="Peer 1">
                <XField v-slot="a" label="Keep Alive"><XNumber v-bind="a" v-model="demoMtu" :min="0" block /></XField>
              </XNestCard>
            </XListHead>
            <XSwitch v-model="demoFragment" boxed label="Fragment" />
            <XNest v-if="demoFragment" bare>
              <XFieldGrid :cols="3">
                <XField v-slot="a" label="MTU"><XNumber v-bind="a" v-model="demoMtu" :min="0" block /></XField>
                <XField v-slot="a" label="Length"><XInput v-bind="a" model-value="100-200" mono /></XField>
                <XField v-slot="a" label="Interval"><XInput v-bind="a" model-value="10-20" mono /></XField>
                <XField v-slot="a" label="Поле на всю строку (x-wide)" class="x-wide"><XInput v-bind="a" :model-value="demoDoh" mono /></XField>
              </XFieldGrid>
            </XNest>
            <XSwitch v-model="demoMux" boxed label="Noises">
              <template #actions><XAddButton label="Добавить: Noises" /></template>
            </XSwitch>
            <XField v-slot="a" label="Число на всю ширину поля (block)"><XNumber v-bind="a" v-model="demoMtu" :min="0" block /></XField>
            <XField v-slot="a" label="Селекторы: пресеты без своих названий не повторяют значение и не выделяются">
              <XTagInput v-bind="a" v-model="demoSelectors" :groups="demoOutboundTags" mono />
            </XField>
          </div>
          <div class="kit__stack">
            <XCard seamless>
              <template #heading>
                <XSegmented v-model="demoObserver" :options="[{ value: 'observatory', label: 'Observatory' }, { value: 'burstObservatory', label: 'Burst Observatory' }]" label="Наблюдатели" />
              </template>
              <XCodeEditor v-model="demoObserverText" :height="200" :phone-height="200" fit label="JSON" />
            </XCard>
            <XSettingsSection v-model:open="demoDnsOpen" title="DNS" :count="demoServers.length" padded>
              <XGridTable
                :columns="demoServerColumns"
                :rows="demoServers"
                :row-key="(r) => r.address"
                :blank="(r, key) => key === 'domains' && !r.domains.length"
                :menu="() => [{ key: 'edit', label: 'Изменить', icon: 'edit' }, { key: 'delete', label: 'Удалить', icon: 'delete', danger: true }]"
                numbered
                nested
              >
                <template #cell-address="{ row }"><span class="kit__mono">{{ row.address }}</span></template>
                <template #cell-domains="{ row }"><XValueChip :values="row.domains" /></template>
              </XGridTable>
            </XSettingsSection>
            <XSettingsSection v-model:open="demoDnsEmptyOpen" title="Fake DNS" :count="0" padded>
              <XEmpty framed title="Нет добавленных DNS-серверов.">
                <XButton variant="primary" icon="plus">Создать DNS</XButton>
                <XButton icon="templates" label="Шаблоны DNS" />
              </XEmpty>
            </XSettingsSection>
          </div>
        </div>
      </div>
    </XCard>

    <XCard title="Подключения">
      <div class="kit__stack">
        <div class="kit__row kit__row--wrap">
          <XSummaryStat icon="updown" label="Отправлено/получено" value="67,7 ГБ / 409,0 ГБ" />
          <XSummaryStat icon="user" label="Клиенты">
            <div class="kit__row kit__row--wrap">
              <XCountList tone="off" label="Отключено" :items="demoEmails.slice(1, 2)" labelled />
              <XCountList tone="err" label="Исчерпано" :items="demoEmails" labelled />
              <XCountList tone="ok" label="Онлайн" :items="demoEmails.slice(0, 1)" dot labelled />
            </div>
          </XSummaryStat>
        </div>
        <div class="kit__row kit__row--wrap">
          <XCountList tone="warn" label="Почти исчерпано" :items="demoEmails" />
          <XMenu v-slot="{ props }" :items="demoInbMenu" @select="$event === 'enable' && (demoInbOn = !demoInbOn)">
            <XButton v-bind="props" caret>Меню с переключателем</XButton>
          </XMenu>
          <XButton icon="info" label="Информация" />
          <XButton icon="info" mark="pause" label="Информация, отключено" />
          <XButton icon="caret" label="Автообновление" dot />
        </div>
        <XKeyValue plain :rows="[{ label: 'Порт', value: '443' }, { key: 'proto', label: 'Протокол' }]" style="max-width: 320px">
          <template #value-proto><XChip mono>vless</XChip><XChip mono outline>tcp</XChip></template>
        </XKeyValue>
        <XExpandTable v-model:open="demoInbOpen" :columns="demoInbColumns" :phone-columns="demoInbPhone" :rows="demoInbRows" :row-key="(r) => r.id" :expandable="(r) => r.id === 1" :muted="(r) => !r.on">
          <template #cell-remark="{ row }">{{ row.remark }}</template>
          <template #cell-port="{ row }"><span class="kit__mono">{{ row.port }}</span></template>
          <template #phone-remark="{ row, open, expandable, toggle }">
            <XRowToggle :open="open" :expandable="expandable" @toggle="toggle"><XMark :text="row.remark" query="ланд" /></XRowToggle>
          </template>
          <template #phone-port="{ row }"><span class="kit__mono">{{ row.port }}</span></template>
          <template #expanded>
            <XInnerTable :columns="demoInnerColumns" :phone-columns="demoInnerColumns.slice(0, 2)" :rows="demoInnerRows" :row-key="(r) => r.email">
              <template #cell-acts>
                <div class="kit__row">
                  <XInlineButton icon="edit" label="Редактировать клиента" tip />
                  <XPopConfirm v-slot="{ props: trigger }" text="Вы уверены, что хотите удалить клиента?" cta="Удалить">
                    <XInlineButton v-bind="trigger" icon="delete" label="Удалить" danger tip />
                  </XPopConfirm>
                </div>
              </template>
              <template #cell-email="{ row }"><XMark :text="row.email" query="ali" /></template>
              <template #phone-acts><XInlineButton icon="dots" label="Меню" /></template>
              <template #phone-email="{ row }"><XMark :text="row.email" query="ali" /></template>
            </XInnerTable>
          </template>
          <template #foot><XPager v-model="demoInbPage" v-model:size="demoInbSize" :pages="3" :sizes="[25, 50, 70]" /></template>
        </XExpandTable>
      </div>
    </XCard>
    <XConfirm v-model:open="deleteAsk" title="Удалить подключение #3" cta="Удалить" danger @confirm="deleteAsk = false">
      Вы уверены, что хотите удалить подключение?
    </XConfirm>
    <XConfirm v-model:open="restartAsk" icon="refresh" title="Перезапуск панели" cta="Да" @confirm="restartAsk = false">
      Вы уверены, что хотите перезапустить панель? Если панель будет недоступна, проверьте лог сервера
    </XConfirm>
    <XBusyOverlay :open="busyOpen" title="Перезапуск панели" text="Ожидание ответа панели…" />

    <div class="kit__live">
      <XCard title="Форма">
        <template #actions>
          <XTooltip text="Подсказка появляется при наведении и при фокусе с клавиатуры">
            <XButton variant="ghost" size="sm" icon="info" label="Справка" />
          </XTooltip>
          <XMenu v-slot="{ props }" :items="menuItems" @select="lastAction = $event">
            <XButton v-bind="props" variant="ghost" size="sm" icon="dots" label="Действия" />
          </XMenu>
        </template>
        <div class="kit__form">
          <XField layout="row" label="Включить" hint="Подключение принимает клиентов">
            <XSwitch v-model="form.enabled" label="Включить" />
          </XField>
          <div class="kit__grid">
            <XField v-slot="f" label="Примечание" required>
              <XInput v-bind="f" v-model="form.remark" />
            </XField>
            <XField v-slot="f" label="Порт" :error="form.port != null && form.port > 65535 ? 'Порт от 1 до 65535' : undefined">
              <XNumber v-bind="f" v-model="form.port" :min="1" :max="65535" :invalid="form.port != null && form.port > 65535" style="width: 100%" />
            </XField>
            <XField v-slot="f" label="Язык">
              <XSelect v-bind="f" v-model="form.lang" :options="languages" />
            </XField>
            <XField v-slot="f" label="Стратегия домена" hint="Как Xray определяет IP-адрес">
              <XSelect v-bind="f" v-model="form.strategy" :options="strategies" mono />
            </XField>
          </div>
          <XField v-slot="f" label="Пароль">
            <XInput v-bind="f" v-model="form.password" type="password" mono autocomplete="new-password" />
          </XField>
          <XField v-slot="f" label="Правила" hint="Enter — из списка, запятая — свой вариант">
            <XTagInput v-bind="f" v-model="form.tags" :presets="geo" mono placeholder="geoip:… или geosite:…" />
          </XField>
          <XField v-slot="f" label="Объявление">
            <XTextarea v-bind="f" v-model="form.note" />
          </XField>
          <XSection v-model:open="sectionOpen" title="Sniffing" description="Определение протокола по первым байтам соединения.">
            <template #meta><XSwitch v-model="form.sniffing" label="Sniffing" /></template>
            <XField layout="row" label="Только маршрутизация">
              <XSwitch v-model="form.routeOnly" label="Только маршрутизация" />
            </XField>
            <XCheckbox v-model="form.agree">Переопределять адрес назначения</XCheckbox>
          </XSection>
          <div class="kit__row kit__row--end">
            <XButton>Отмена</XButton>
            <XButton variant="primary" icon="check">Сохранить</XButton>
          </div>
        </div>
      </XCard>

      <XCard title="Окна и поведение">
        <div class="kit__stack">
          <p class="kit__p">Последнее действие меню: <code>{{ lastAction }}</code></p>
          <div class="kit__row kit__row--wrap">
            <XSegmented
              v-model="sheetSize"
              label="Размер окна"
              :options="[{ value: 'sm', label: 'sm 440' }, { value: 'md', label: 'md 560' }, { value: 'lg', label: 'lg 720' }, { value: 'xl', label: 'xl 800' }]"
            />
            <XButton variant="primary" @click="modalOpen = true">Открыть окно</XButton>
            <XButton @click="logOpen = true">Окно на весь экран телефона</XButton>
            <XButton @click="confirmOpen = true">Подтверждение</XButton>
          </div>
          <div class="kit__row kit__row--wrap">
            <XMenu v-slot="{ props }" :items="levelItems" title="Уровень" placement="bottom-start" sheet @select="level = $event">
              <XButton v-bind="props" caret>Меню-лист на телефоне</XButton>
            </XMenu>
          </div>
          <XField layout="row" label="Оптимистичный переключатель" hint="Спиннер на время запроса">
            <XSwitch v-model="sw.b" :loading="sw.loading" label="Применить" @update:model-value="applySwitch" />
          </XField>
        </div>
      </XCard>
    </div>

    <XModal v-model:open="modalOpen" title="Создать подключение" :size="sheetSize" cancel>
      <XField v-slot="f" label="Примечание">
        <XInput v-bind="f" v-model="form.remark" autofocus />
      </XField>
      <XField v-slot="f" label="Протокол">
        <XSelect v-bind="f" v-model="form.strategy" :options="strategies" />
      </XField>
      <XField layout="row" label="Включить">
        <XSwitch v-model="form.enabled" label="Включить" />
      </XField>
      <template #footer>
        <XButton variant="primary" @click="modalOpen = false">Создать</XButton>
      </template>
    </XModal>

    <XConfirm v-model:open="confirmOpen" title="Восстановить базу данных?" cta="Восстановить" :busy="confirmBusy" busy-label="Восстановление…" @confirm="runConfirm">
      Текущая база данных будет заменена файлом backup.db, после этого панель перезапустится.
    </XConfirm>

    <XModal v-model:open="logOpen" title="Журнал панели" size="xl" flush full>
      <template #subtitle>Подзаголовок · <span class="kit__mono">flush</span>: тело без отступов, <span class="kit__mono">full</span>: весь экран телефона</template>
      <XEmpty title="Нет записей" hint="Тело окна рисует сам экран" />
    </XModal>
  </div>
</template>

<style scoped>
.demo-tiles {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.kit {
  display: flex;
  flex-direction: column;
  gap: 24px;
  max-width: 1280px;
  margin: 0 auto;
  padding: 32px 24px 56px;
}

.kit__top {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
}

.kit__title {
  margin: 0;
  font: 600 var(--fs-24) / var(--lh-tight) var(--font-sans);
  letter-spacing: -.4px;
}

.kit__sub,
.kit__p {
  margin: 4px 0 0;
  color: var(--mut);
}

.kit__h3 {
  margin: 0 0 10px;
  color: var(--dim);
  font: 500 var(--fs-11) / 1 var(--font-mono);
  letter-spacing: .07em;
  text-transform: uppercase;
}

.kit__tokens {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 24px;
}

.kit__swatches {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.kit__swatch {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  font-size: var(--fs-10);
  color: var(--dim);
}

.kit__swatch-color {
  width: 44px;
  height: 32px;
  border: 1px solid var(--line);
  border-radius: var(--r-8);
}

.kit__radius {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border: 1px solid var(--acc-line);
  background: var(--acc-soft);
  color: var(--acc-t);
  font: 500 var(--fs-11) var(--font-mono);
}

.kit__fonts {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.kit__mono {
  margin-top: 6px;
  color: var(--fg2);
}

.kit__spacing {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.kit__space {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--fs-10);
  color: var(--dim);
}

.kit__space span {
  height: 8px;
  border-radius: var(--r-pill);
  background: var(--acc);
}

.kit__table-wrap {
  overflow-x: auto;
}

.kit__table {
  width: 100%;
  min-width: 1180px;
  border-collapse: collapse;
}

.kit__table th,
.kit__table td {
  padding: 12px 14px;
  border-bottom: 1px solid var(--div);
  text-align: left;
  vertical-align: middle;
}

.kit__table td {
  width: 13%;
  border-left: 1px solid var(--div);
}

.kit__table thead th {
  color: var(--dim);
  font: 500 var(--fs-10) / 1 var(--font-mono);
  letter-spacing: .07em;
  text-transform: uppercase;
}

.kit__table tbody th {
  min-width: 150px;
}

.kit__table tbody th b {
  display: block;
  font-weight: 600;
}

.kit__table tbody th code {
  color: var(--dim);
  font-size: var(--fs-11);
}

.kit__table tbody tr:last-child > * {
  border-bottom: 0;
}

.kit__row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.kit__row--wrap {
  flex-wrap: wrap;
}

.kit__row--end {
  justify-content: flex-end;
}

.kit__stack {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
}

.kit__stack > .x-notice,
.kit__stack > .x-field {
  align-self: stretch;
}

.kit__variants {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(340px, 1fr));
  gap: 32px;
}

.kit__variant {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.kit__scroll {
  max-width: 400px;
  height: 140px;
  padding: 8px 12px;
  overflow: auto;
  border: 1px solid var(--line);
  border-radius: var(--r-12);
  color: var(--mut);
  font-size: var(--fs-12);
  line-height: var(--lh-code);
  white-space: nowrap;
}

.kit__live {
  display: grid;
  grid-template-columns: minmax(0, 3fr) minmax(0, 2fr);
  gap: 24px;
  align-items: start;
}

.kit__form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.kit__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

/* Two columns of demos, one on narrow screens. */
.kit__duo {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: start;
  gap: 24px;
}

@media (max-width: 960px) {
  .kit__live,
  .kit__duo {
    grid-template-columns: minmax(0, 1fr);
  }
}

.kit__remark {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.kit__remark-line {
  display: flex;
  gap: 8px;
}

.kit__remark-separator {
  flex: none;
  width: 98px;
}

@media (max-width: 767px) {
  .kit {
    padding: 20px 16px 40px;
  }

  /* As on the settings tab: the parts take the whole line, the separator goes under them. */
  .kit__remark-line {
    flex-direction: column;
  }

  .kit__grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .kit__variants {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
