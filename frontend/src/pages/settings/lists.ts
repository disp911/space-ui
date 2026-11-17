// The lists of domains and subnets the panel serves to a router as sing-box
// rule-sets (web/service/rule_list.go): what the API gives and the
// checks the server makes, so a window can show a wrong line before saving.

export type ListKind = 'domains' | 'subnets'

export interface RuleList {
  id: number
  name: string
  kind: ListKind
  /** One entry a line, as the server normalized them. */
  entries: string
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

export const maxEntries = 200000

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

/** The first line that is not an entry of the kind, counted from 1, and how many good entries there are. */
export function checkList(kind: ListKind, text: string): { bad?: { line: number; text: string }; count: number } {
  const seen = new Set<string>()
  let bad: { line: number; text: string } | undefined
  const lines = text.split('\n')
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]!.trim()
    if (!line || line.startsWith('#')) continue
    const entry = kind === 'domains' ? normalizeDomain(line) : isSubnet(line) ? line.toLowerCase() : null
    if (entry === null) bad ??= { line: i + 1, text: line }
    else seen.add(entry)
  }
  return { bad, count: seen.size }
}
