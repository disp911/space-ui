// Presets the IP and domain fields of a routing rule offer, as the old Xray
// page had them; the geo files the user added come last.

import { api } from '../../api'
import type { Text } from './model'

type Preset = { value: string; label: string }

export const ipPresets: Preset[] = [
  { label: 'Private IPs', value: 'geoip:private' },
  { label: '🇨🇳 China', value: 'geoip:cn' },
  { label: '🇷🇺 Russia', value: 'ext:geoip_RU.dat:ru' },
  { label: '🇻🇳 Vietnam', value: 'geoip:vn' },
  { label: '🇪🇸 Spain', value: 'geoip:es' },
  { label: '🇮🇩 Indonesia', value: 'geoip:id' },
  { label: '🇺🇦 Ukraine', value: 'geoip:ua' },
  { label: '🇹🇷 Türkiye', value: 'geoip:tr' },
  { label: '🇧🇷 Brazil', value: 'geoip:br' },
]

export const domainPresets: { title: Text; options: Preset[] }[] = [
  {
    title: 'xray.catalogRegions',
    options: [
      { label: '🇨🇳 China', value: 'geosite:cn' },
      { label: '🇨🇳 .cn', value: 'regexp:.*\\.cn$' },
      { label: '🇷🇺 Russia', value: 'ext:geosite_RU.dat:ru-available-only-inside' },
      { label: '🇷🇺 .ru', value: 'regexp:.*\\.ru$' },
      { label: '🇷🇺 .su', value: 'regexp:.*\\.su$' },
      { label: '🇷🇺 .рф', value: 'regexp:.*\\.xn--p1ai$' },
      { label: '🇻🇳 .vn', value: 'regexp:.*\\.vn$' },
    ],
  },
  {
    title: 'xray.catalogThreats',
    options: [
      { label: 'Ads All', value: 'geosite:category-ads-all' },
      { label: 'Ads RU 🇷🇺', value: 'ext:geosite_RU.dat:category-ads-all' },
      { label: 'Adult +18', value: 'geosite:category-porn' },
    ],
  },
  {
    title: 'xray.catalogServices',
    options: [
      { label: 'Apple', value: 'geosite:apple' },
      { label: 'Meta', value: 'geosite:meta' },
      { label: 'Google', value: 'geosite:google' },
      { label: 'OpenAI', value: 'geosite:openai' },
      { label: 'Spotify', value: 'geosite:spotify' },
      { label: 'Netflix', value: 'geosite:netflix' },
      { label: 'Reddit', value: 'geosite:reddit' },
      { label: 'Speedtest', value: 'geosite:speedtest' },
    ],
  },
]

interface Alias {
  alias: string
  extExample: string
}

/** The user's own geo files: geoip ones join the IP presets, geosite ones make a group of domains. */
export async function customGeo(suffix: string): Promise<{ ip: Preset[]; domain: Preset[] }> {
  try {
    const out = await api.get<{ geoip?: Alias[]; geosite?: Alias[] } | null>('panel/api/custom-geo/aliases')
    const as = (list?: Alias[]) => (list ?? []).map((x) => ({ label: x.alias + suffix, value: x.extExample }))
    return { ip: as(out?.geoip), domain: as(out?.geosite) }
  } catch {
    return { ip: [], domain: [] }
  }
}

