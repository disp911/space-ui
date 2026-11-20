// The lists of domains and subnets the panel serves to a router as sing-box
// rule-sets (web/service/rule_list.go): what the API gives and the
// checks the server makes, so a window can show a wrong line before saving.

export type ListKind = 'domains' | 'subnets'

/** The connections a subnets section is kept to: '' every one, else one network's. */
export type ListNetwork = '' | 'tcp' | 'udp'

/** A named part of a list, e.g. a service's domains, that can be switched off and keeps its entries. */
export interface RuleListSection {
  name: string
  enabled: boolean
  /** One entry a line, as the server normalized them. */
  entries: string
  count: number
  /** A subnets section's network and ports, each a port or a range ("19000-20000"); none mean every one. */
  network?: 'tcp' | 'udp'
  ports?: string[]
}

/** A section as the window's JSON shows it: its entries one a string. */
export interface RawSection {
  name: string
  enabled: boolean
  network?: 'tcp' | 'udp'
  ports?: string[]
  entries: string[]
}

const strings = (v: unknown): v is string[] => Array.isArray(v) && v.every((e) => typeof e === 'string')

/** Sections read from parsed JSON; null when it is not a list of them. enabled is true unless false. */
export function readRawSections(v: unknown): RawSection[] | null {
  if (!Array.isArray(v)) return null
  const out: RawSection[] = []
  for (const s of v) {
    if (!s || typeof s !== 'object' || Array.isArray(s)) return null
    const { name, enabled = true, network, ports, entries = [] } = s as Record<string, unknown>
    if (typeof name !== 'string' || typeof enabled !== 'boolean' || !strings(entries)) return null
    if (network !== undefined && network !== 'tcp' && network !== 'udp') return null
    if (ports !== undefined && !strings(ports)) return null
    const kept = ports as string[] | undefined
    out.push({ name, enabled, ...(network ? { network } : {}), ...(kept?.length ? { ports: kept } : {}), entries })
  }
  return out
}

export interface RuleList {
  id: number
  name: string
  kind: ListKind
  sections: RuleListSection[] | null
  /** Entries in the rule-set: those of the enabled sections, each once. */
  count: number
  createdAt: number
  updatedAt: number
}

export interface ListsReply {
  /** The address the website answers at; a list's address starts with it. */
  base: string
  lists: RuleList[] | null
}

/** A list's name is its file's: routers name the rule-set after it. */
export const listNamePattern = /^[a-z0-9][a-z0-9_-]{0,62}$/

/** A list's entries, its sections' together. */
export const maxEntries = 200000
export const maxSections = 100
/** A section's name, in characters. */
export const maxSectionName = 64
/** A section's ports and port ranges. */
export const maxPorts = 64

/**
 * A section's ports as the window takes them, split by commas, each a port
 * ("3478") or a range ("19000-20000"), kept as the server keeps them: without
 * repeats, a range of one port as that port. The first that is neither, or
 * that there are too many.
 */
export function readPorts(text: string | string[]): { ports: string[]; bad?: string; tooMany?: boolean } {
  const parts = (Array.isArray(text) ? text : text.split(',')).map((p) => p.trim()).filter(Boolean)
  const port = (s: string) => (/^[1-9]\d{0,4}$/.test(s) && Number(s) <= 65535 ? Number(s) : null)
  const ports: string[] = []
  for (const part of parts) {
    const [from = '', to, extra] = part.split('-').map((p) => p.trim())
    const a = port(from)
    const b = to === undefined ? a : port(to)
    if (extra !== undefined || a === null || b === null || a > b) return { ports, bad: part }
    const kept = a === b ? String(a) : `${a}-${b}`
    if (!ports.includes(kept)) ports.push(kept)
  }
  return { ports, tooMany: ports.length > maxPorts }
}

/** Where a router fetches the list: <base><path><secret>/<name>.srs. */
export function listUrl(base: string, path: string, secret: string, name: string) {
  return `${base.replace(/\/$/, '')}${path}${secret}/${name}.srs`
}

const label = /^[a-z0-9_]([a-z0-9_-]{0,61}[a-z0-9_])?$/

/** A name as a domain list keeps it, punycode for other scripts; null when it is no domain. */
export function normalizeDomain(raw: string): string | null {
  let s = raw.trim().toLowerCase()
  if (s.startsWith('*.')) s = s.slice(2)
  if (s.startsWith('.')) s = s.slice(1)
  if (s.endsWith('.')) s = s.slice(0, -1)
  if (!s || s.length > 253) return null
  if (/[^\x20-\x7e]/.test(s)) {
    if (s.includes('_')) return null
    try {
      s = new URL(`http://${s}`).hostname
    } catch {
      return null
    }
  }
  if (s.length > 253) return null
  return s.split('.').every((l) => label.test(l)) ? s : null
}

const octet = '(25[0-5]|2[0-4]\\d|1\\d\\d|[1-9]?\\d)'
const ipv4 = new RegExp(`^${octet}(\\.${octet}){3}$`)

/** Whether the line is a subnet or a lone address; IPv4-mapped IPv6 and zones are refused, as by the server. */
export function isSubnet(raw: string): boolean {
  const s = raw.trim()
  const [addr = '', bits, extra] = s.split('/')
  if (extra !== undefined) return false
  if (bits !== undefined && !/^(0|[1-9]\d{0,2})$/.test(bits)) return false
  if (ipv4.test(addr)) return bits === undefined || Number(bits) <= 32
  if (!addr.includes(':') || addr.includes('%')) return false
  let host: string
  try {
    host = new URL(`http://[${addr}]`).hostname
  } catch {
    return false
  }
  // ::ffff:a.b.c.d comes back as ::ffff:xxxx:xxxx.
  if (/^\[::ffff:[0-9a-f]{1,4}:[0-9a-f]{1,4}\]$/.test(host)) return false
  return bits === undefined || Number(bits) <= 128
}

/** An address as a number and its length in bits; null when it is none. */
function addrNumber(addr: string): { bits: 32 | 128; n: bigint } | null {
  if (ipv4.test(addr)) return { bits: 32, n: addr.split('.').reduce((n, o) => (n << 8n) | BigInt(o), 0n) }
  let host: string
  try {
    host = new URL(`http://[${addr}]`).hostname.slice(1, -1)
  } catch {
    return null
  }
  // The URL writes the address as hex groups, at most one "::".
  const [head = '', tail] = host.split('::')
  const left = head ? head.split(':') : []
  const right = tail ? tail.split(':') : []
  const groups = tail === undefined ? left : [...left, ...Array<string>(8 - left.length - right.length).fill('0'), ...right]
  return { bits: 128, n: groups.reduce((n, g) => (n << 16n) | BigInt(parseInt(g, 16)), 0n) }
}

/**
 * Whether the subnets' addresses together are one subnet, as sing-box writes
 * a rule's subnets out of a rule-set: 10.0.0.0/25 and 10.0.0.128/25 are
 * 10.0.0.0/24. Lines that are no subnet are left out.
 */
export function isOneSubnet(subnets: string[]): boolean {
  const ranges: { bits: number; from: bigint; to: bigint }[] = []
  for (const s of subnets) {
    if (!isSubnet(s)) continue
    const [addr = '', len] = s.trim().split('/')
    const a = addrNumber(addr)
    if (!a) continue
    const host = BigInt(a.bits - (len === undefined ? a.bits : Number(len)))
    const from = (a.n >> host) << host
    ranges.push({ bits: a.bits, from, to: from + (1n << host) - 1n })
  }
  ranges.sort((x, y) => x.bits - y.bits || (x.from < y.from ? -1 : x.from > y.from ? 1 : 0))
  const merged: typeof ranges = []
  for (const r of ranges) {
    const last = merged[merged.length - 1]
    if (last && last.bits === r.bits && r.from <= last.to + 1n) last.to = r.to > last.to ? r.to : last.to
    else merged.push({ ...r })
  }
  if (merged.length !== 1) return false
  const { from, to } = merged[0]!
  const size = to - from + 1n
  return (size & (size - 1n)) === 0n && from % size === 0n
}

/**
 * The sections of a subnets list whose rule would be one subnet, which a router
 * cannot read, so loading none of the list: the enabled sections kept to the
 * same protocol and ports make one rule, as the server builds them.
 */
export function loneSubnetSections(sections: { enabled: boolean; network: ListNetwork; ports: string[]; entries: string[] }[]): boolean[] {
  const groups = new Map<string, number[]>()
  sections.forEach((s, i) => {
    if (!s.enabled) return
    const key = `${s.network}|${s.ports.join(',')}`
    groups.set(key, [...(groups.get(key) ?? []), i])
  })
  const lone = sections.map(() => false)
  for (const members of groups.values()) {
    const entries = members.flatMap((i) => sections[i]!.entries)
    if (isOneSubnet(entries)) for (const i of members) lone[i] = sections[i]!.entries.some(isSubnet)
  }
  return lone
}

/**
 * The first line that is not an entry of the kind, counted from 1, how many
 * good entries there are and, in a subnet list, its IPv6 entries: how many
 * and the first. Some routers keep a subnet list in an IPv4 set, where an IPv6
 * entry fails to load and may take the entries loaded with it along.
 */
export function checkList(kind: ListKind, text: string): { bad?: { line: number; text: string }; count: number; v6?: { line: number; text: string; count: number } } {
  const seen = new Set<string>()
  let bad: { line: number; text: string } | undefined
  let v6: { line: number; text: string; count: number } | undefined
  const lines = text.split('\n')
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]!.trim()
    if (!line || line.startsWith('#')) continue
    const entry = kind === 'domains' ? normalizeDomain(line) : isSubnet(line) ? line.toLowerCase() : null
    if (entry === null) bad ??= { line: i + 1, text: line }
    else if (!seen.has(entry)) {
      seen.add(entry)
      if (kind === 'subnets' && entry.includes(':')) {
        v6 ??= { line: i + 1, text: line, count: 0 }
        v6.count++
      }
    }
  }
  return { bad, count: seen.size, v6 }
}
