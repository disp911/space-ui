// XHTTP checks that Xray makes when it starts, made in the windows first.

/** Xray's own session id tables (transport/internet/splithttp): a name stands for its characters. */
const tables: Record<string, number> = {
  ALPHABET: 26,
  Alphabet: 52,
  BASE36: 36,
  Base62: 62,
  HEX: 16,
  alphabet: 26,
  base36: 36,
  hex: 16,
  number: 10,
}

/** Xray wants at least 2^31 session ids to pick from. */
const enough = 2n ** 31n

const sizeOf = (table: string) => tables[table] ?? table.length

/** A range as Xray reads one: "6-9", "6" or 6; null when it is none. */
function range(v: unknown): [number, number] | null {
  if (typeof v === 'number') return Number.isInteger(v) ? [v, v] : null
  const m = /^\s*(\d+)\s*(?:-\s*(\d+)\s*)?$/.exec(String(v ?? ''))
  if (!m) return null
  const from = Number(m[1])
  const to = m[2] === undefined ? from : Number(m[2])
  return from <= to ? [from, to] : null
}

/** How many ids a table and a length give: Xray's roomSize. */
function room(size: number, [from, to]: [number, number]) {
  let sum = 0n
  for (let k = from; k <= to; k++) sum += BigInt(size) ** BigInt(k)
  return sum
}

/**
 * A session id table without a length long enough for it: Xray does not
 * start on one ("sessionIDTable or sessionIDLength is too small").
 */
export function sessionIdWrong(ns: Record<string, any> | undefined): boolean {
  const table = String(ns?.sessionIDTable ?? '')
  if (!table) return false
  const r = range(ns?.sessionIDLength)
  return !r || r[0] <= 0 || room(sizeOf(table), r) < enough
}

/** The shortest length that gives a table enough ids, and three more to vary by, e.g. 6-9 for base36. */
export function sessionIdLength(table: string): string {
  const size = sizeOf(table)
  if (size < 2) return ''
  let n = 1
  while (BigInt(size) ** BigInt(n) < enough) n++
  return `${n}-${n + 3}`
}
