// The time zones the panel's schedule can keep to: the IANA names the browser
// knows, each with its offset from UTC now, so the list can be searched by
// either. The server carries the same database (time/tzdata), so any of
// them loads there too.

import type { Option } from '../../kit'

/** The zone's offset from UTC at the moment, as «UTC+3» or «UTC+5:30»; «UTC» when none. */
export function utcOffset(zone: string, at = new Date()): string {
  try {
    const part = new Intl.DateTimeFormat('en-US', { timeZone: zone, timeZoneName: 'shortOffset' })
      .formatToParts(at)
      .find((p) => p.type === 'timeZoneName')?.value
    return part ? part.replace('GMT', 'UTC') : ''
  } catch {
    return ''
  }
}

/**
 * The zones to pick from: the server's own and UTC first, then every IANA
 * zone with its offset. A saved zone the browser does not list, such as an
 * old alias, stays among them so the field shows it.
 */
export function timeZoneOptions(serverLabel: string, saved: string): Option[] {
  const zones = typeof Intl.supportedValuesOf === 'function' ? Intl.supportedValuesOf('timeZone') : []
  const named = zones.filter((z) => z !== 'UTC')
  if (saved && saved !== 'Local' && saved !== 'UTC' && !named.includes(saved)) named.unshift(saved)
  return [
    { value: 'Local', label: serverLabel },
    { value: 'UTC', label: 'UTC' },
    ...named.map((z) => {
      const offset = utcOffset(z)
      return { value: z, label: offset ? `${z} · ${offset}` : z }
    }),
  ]
}
