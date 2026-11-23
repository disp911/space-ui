// The time zone the panel's windows show their dates in: the panel's, which
// the server puts into the page (config.zone). The subscription page has none
// and keeps the browser's own, as its readers are the clients.

import { config } from '../config'

/** A moment's clock: the month counted from 1. */
export interface ClockParts {
  year: number
  month: number
  day: number
  hour: number
  minute: number
  second: number
}

function formatter(zone: string | undefined) {
  return new Intl.DateTimeFormat('en-US', {
    timeZone: zone,
    hourCycle: 'h23',
    year: 'numeric',
    month: 'numeric',
    day: 'numeric',
    hour: 'numeric',
    minute: 'numeric',
    second: 'numeric',
  })
}

// A zone the browser does not know falls back to the browser's own.
const clock = (() => {
  try {
    return formatter(config.zone || undefined)
  } catch {
    return formatter(undefined)
  }
})()

/** The moment's clock in the panel's zone. */
export function clockOf(ms: number): ClockParts {
  const p: Record<string, number> = {}
  for (const part of clock.formatToParts(new Date(ms))) if (part.type !== 'literal') p[part.type] = Number(part.value)
  return { year: p.year!, month: p.month!, day: p.day!, hour: p.hour!, minute: p.minute!, second: p.second! }
}

/** How far the panel's zone is ahead of UTC at the moment, in milliseconds. */
function offsetAt(ms: number) {
  const c = clockOf(ms)
  return Date.UTC(c.year, c.month - 1, c.day, c.hour, c.minute, c.second) - Math.floor(ms / 1000) * 1000
}

const same = (a: ClockParts, b: ClockParts) =>
  a.year === b.year && a.month === b.month && a.day === b.day && a.hour === b.hour && a.minute === b.minute && a.second === b.second

/**
 * The moment the panel's zone reads as this clock. Taken as UTC first, then
 * moved by the zone's offset; the offset is looked up again where it lands,
 * as a clock change between the two moves it. A clock the zone skips, when
 * the clocks go forward, is taken as the moment after the change, as it
 * would read had they not.
 */
export function momentOf(c: ClockParts): number {
  const asUtc = Date.UTC(c.year, c.month - 1, c.day, c.hour, c.minute, c.second)
  const first = asUtc - offsetAt(asUtc)
  const second = asUtc - offsetAt(first)
  if (same(clockOf(second), c)) return second
  if (same(clockOf(first), c)) return first
  return Math.max(first, second)
}
