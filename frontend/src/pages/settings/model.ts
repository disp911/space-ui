// The panel's settings as POST panel/setting/all returns them and
// panel/setting/update takes them back (web/entity AllSetting).

import type { MessageSchema } from '../../i18n/schema.gen'

/** A message of the settings screen. */
export type Text = `settings.${keyof MessageSchema['settings']}`

export interface AllSetting {
  webListen: string
  webDomain: string
  webPort: number
  webCertFile: string
  webKeyFile: string
  webBasePath: string
  sessionMaxAge: number

  pageSize: number
  expireDiff: number
  trafficDiff: number
  remarkModel: string

  timeLocation: string
  twoFactorEnable: boolean
  twoFactorToken: string

  subEnable: boolean
  subJsonEnable: boolean
  subTitle: string
  subSupportUrl: string
  subProfileUrl: string
  subAnnounce: string
  subEnableRouting: boolean
  subRoutingRules: string
  subListen: string
  subPort: number
  subPath: string
  subDomain: string
  subCertFile: string
  subKeyFile: string
  subUpdates: number
  externalTrafficInformEnable: boolean
  externalTrafficInformURI: string
  restartXrayOnClientDisable: boolean
  subEncrypt: boolean
  subShowInfo: boolean
  subURI: string
  subJsonPath: string
  subJsonURI: string
  subClashEnable: boolean
  subClashPath: string
  subClashURI: string
  subJsonFragment: string
  subJsonNoises: string
  subJsonMux: string
  subJsonRules: string

  listsPath: string
  listsSecret: string

  ldapEnable: boolean
  ldapHost: string
  ldapPort: number
  ldapUseTLS: boolean
  ldapBindDN: string
  ldapPassword: string
  ldapBaseDN: string
  ldapUserFilter: string
  ldapUserAttr: string
  ldapVlessField: string
  ldapSyncCron: string
  ldapFlagField: string
  ldapTruthyValues: string
  ldapInvertFlag: boolean
  ldapInboundTags: string
  ldapAutoCreate: boolean
  ldapAutoDelete: boolean
  ldapDefaultTotalGB: number
  ldapDefaultExpiryDays: number
  ldapDefaultLimitIP: number
}

export type SettingKey = keyof AllSetting

export type TabKey = 'panel' | 'site' | 'ui' | 'account' | 'ldap' | 'sub' | 'formats' | 'lists'

/** The tabs in order, with the settings each edits: its dot and the status line count changes and errors among them. */
export const tabs: { key: TabKey; label: Text; keys: SettingKey[] }[] = [
  {
    key: 'panel',
    label: 'settings.tabPanel',
    keys: [
      'webListen', 'webDomain', 'webPort', 'webBasePath', 'sessionMaxAge',
      'restartXrayOnClientDisable', 'externalTrafficInformEnable', 'externalTrafficInformURI',
    ],
  },
  { key: 'site', label: 'settings.tabSite', keys: [] },
  { key: 'ui', label: 'settings.tabUi', keys: ['pageSize', 'timeLocation', 'expireDiff', 'trafficDiff'] },
  { key: 'account', label: 'settings.tabAccount', keys: [] },
  {
    key: 'ldap',
    label: 'settings.tabLdap',
    keys: [
      'ldapEnable', 'ldapHost', 'ldapPort', 'ldapUseTLS', 'ldapBindDN', 'ldapPassword', 'ldapBaseDN', 'ldapUserFilter', 'ldapUserAttr',
      'ldapVlessField', 'ldapFlagField', 'ldapTruthyValues', 'ldapInvertFlag', 'ldapSyncCron', 'ldapInboundTags',
      'ldapAutoCreate', 'ldapAutoDelete', 'ldapDefaultTotalGB', 'ldapDefaultExpiryDays', 'ldapDefaultLimitIP',
    ],
  },
  {
    key: 'sub',
    label: 'settings.tabSub',
    keys: [
      'remarkModel', 'subEnable', 'subJsonEnable', 'subClashEnable', 'subListen', 'subDomain', 'subPort', 'subPath', 'subURI',
      'subEncrypt', 'subShowInfo', 'subTitle', 'subSupportUrl', 'subProfileUrl', 'subAnnounce', 'subEnableRouting', 'subRoutingRules',
      'subCertFile', 'subKeyFile', 'subUpdates',
    ],
  },
  {
    key: 'formats',
    label: 'settings.tabFormats',
    keys: ['subJsonPath', 'subJsonURI', 'subClashPath', 'subClashURI', 'subJsonFragment', 'subJsonNoises', 'subJsonMux', 'subJsonRules'],
  },
  { key: 'lists', label: 'settings.tabLists', keys: ['listsPath', 'listsSecret'] },
]

/** The URI path a subscription is served at: the reverse proxy's when one is set. */
function subPathOf(uri: string, path: string) {
  try {
    return uri ? new URL(uri).pathname : path
  } catch {
    return path
  }
}

/** What the security banner warns about in the saved settings, as translation keys. */
/** reality: the panel answers on 443 behind REALITY and opens no port of its own. */
export function securityAlerts(s: AllSetting, reality = false) {
  const alerts: Text[] = []
  // The certificate may be the panel's own, obtained by itself: what counts is how this page came.
  if (location.protocol !== 'https:') alerts.push('settings.securityTls')
  if (!reality && s.webPort === 2053) alerts.push('settings.securityPort')
  if (s.webBasePath === '/') alerts.push('settings.securityPath')
  if (s.subEnable && subPathOf(s.subURI, s.subPath) === '/sub/') alerts.push('settings.securitySubPath')
  if (s.subJsonEnable && subPathOf(s.subJsonURI, s.subJsonPath) === '/json/') alerts.push('settings.securityJsonPath')
  return alerts
}

/** What a setting must hold: a whole number, within min and max; or a path with its slashes. */
export interface Check {
  min?: number
  max?: number
  /** both: begins and ends with a slash; start: begins with one. An empty path passes. */
  path?: 'both' | 'start'
}

export const checks: Partial<Record<SettingKey, Check>> = {
  webPort: { min: 1, max: 65535 },
  webBasePath: { path: 'both' },
  sessionMaxAge: { min: 60 },
  pageSize: { min: 0 },
  expireDiff: { min: 0 },
  trafficDiff: { min: 0 },
  ldapPort: { min: 1, max: 65535 },
  ldapDefaultTotalGB: { min: 0 },
  ldapDefaultExpiryDays: { min: 0 },
  ldapDefaultLimitIP: { min: 0 },
  subPort: { min: 1, max: 65535 },
  subPath: { path: 'both' },
  subCertFile: { path: 'start' },
  subKeyFile: { path: 'start' },
  subUpdates: { min: 1 },
  subJsonPath: { path: 'both' },
  subClashPath: { path: 'both' },
  listsPath: { path: 'both' },
}

/** What the lists' secret may be, as the server checks it. */
export const listsSecretPattern = /^[A-Za-z0-9]{16,64}$/

/** The concurrency values of the JSON subscription's mux. */
export const muxCheck: Check = { min: -1, max: 1024 }

/** Why each setting fails its check, for the settings as edited. */
export function settingErrors(s: AllSetting) {
  const out: Partial<Record<SettingKey, [Text, Record<string, number>?]>> = {}
  for (const [key, check] of Object.entries(checks) as [SettingKey, Check][]) {
    // A format that is off keeps whatever path it had.
    if ((key === 'subJsonPath' && !s.subJsonEnable) || (key === 'subClashPath' && !s.subClashEnable)) continue
    const fail = checkValue(check, s[key])
    if (fail) out[key] = fail
  }
  if (!out.listsPath) {
    // The lists' folder is the website's own: not the root, and apart from the panel's and the subscriptions' paths.
    const lists = s.listsPath
    const others = [s.webBasePath, s.subPath, s.subJsonPath, s.subClashPath]
    if (lists === '/' || others.some((o) => (o !== '/' && lists.startsWith(o)) || o.startsWith(lists))) out.listsPath = ['settings.errListsPath']
  }
  if (!listsSecretPattern.test(s.listsSecret)) out.listsSecret = ['settings.errListsSecret']
  if (s.subJsonMux) {
    try {
      const mux = JSON.parse(s.subJsonMux) as { concurrency?: unknown; xudpConcurrency?: unknown }
      const fail = checkValue(muxCheck, mux.concurrency) ?? checkValue(muxCheck, mux.xudpConcurrency)
      if (fail) out.subJsonMux = fail
    } catch {
      // not ours to judge: the server keeps what it has
    }
  }
  return out
}

/** Why a value fails its check, as a translation key with its values; null when it passes. */
export function checkValue(c: Check, value: unknown): [Text, Record<string, number>?] | null {
  if (c.path) {
    const s = String(value ?? '')
    if (!s) return null
    if (c.path === 'both' && !(s.startsWith('/') && s.endsWith('/'))) return ['settings.errPathBoth']
    if (c.path === 'start' && !s.startsWith('/')) return ['settings.errPathStart']
    return null
  }
  if (typeof value !== 'number' || !Number.isInteger(value)) return ['settings.errInteger']
  if (c.min != null && c.max != null && (value < c.min || value > c.max)) return ['settings.errRange', { min: c.min, max: c.max }]
  if (c.min != null && value < c.min) return ['settings.errMin', { min: c.min }]
  return null
}

/** The panel's address with these settings, as shown under the fields that make it up. */
export function panelAddress(s: AllSetting) {
  const host = s.webDomain || s.webListen || location.hostname
  return `${location.protocol}//${host}:${s.webPort}${s.webBasePath || '/'}`
}
