// Share links of an inbound and its clients, built as the old panel built them
// (genLink and the rest in inbound.js of the 2.x panel): one per address the
// clients reach the inbound by, named by the panel's remark model, plus the
// WEB proxy link of MTProto and the configs of WireGuard's peers.

import { wireguardKeys } from '../../wireguard'
import { defaultTls, fromDb, withClients, withMask, isSS2022, isSSMultiUser, type InboundForm } from './inbound'
import type { DbInbound } from './model'

type Json = Record<string, any>

/** A link with the name it carries. */
export interface Link {
  remark: string
  link: string
}

/** The remark model the panel falls back to: "-" between the inbound, the email and the External Proxy's remark. */
export const defaultRemarkModel = '-ieo'

// --- Pieces ------------------------------------------------------------------------

/** Base64 of the text as UTF-8; url-safe without padding when asked. */
function base64(text: string, safe = false) {
  const out = btoa(String.fromCharCode(...new TextEncoder().encode(text)))
  return safe ? out.replace(/\+/g, '-').replace(/=/g, '').replace(/\//g, '_') : out
}

const filled = (v: unknown) => v !== undefined && v !== null && v !== ''

/** The first value of a header, its name in any case; undefined without one. */
function headerOf(headers: Json | undefined, name: string): string | undefined {
  for (const [k, v] of Object.entries(headers ?? {})) {
    if (k.toLowerCase() !== name.toLowerCase()) continue
    if (!Array.isArray(v)) return v
    if (v.length) return v[0]
  }
  return undefined
}

const header = (headers: Json | undefined, name: string) => headerOf(headers, name) ?? ''

/** The Host of a transport: its own field when set, else its Host header. */
const hostOf = (s: Json) => (s.host?.length > 0 ? s.host : header(s.headers, 'host'))

/** The address the inbound's links point at: its listen address when a specific one, the panel's host otherwise. */
export const linkAddress = (f: InboundForm) => (filled(f.listen) && f.listen !== '0.0.0.0' ? f.listen : location.hostname)

/** finalmask as share links carry it (fm=…), when it has a value to share. */
function finalMask(st: Json): string {
  const fm = st.finalmask
  if (!fm) return ''
  const out: Json = {}
  if (Array.isArray(fm.tcp) && fm.tcp.length) out.tcp = fm.tcp
  if (Array.isArray(fm.udp) && fm.udp.length) out.udp = fm.udp
  if (fm.quicParams) out.quicParams = fm.quicParams
  const shareable = (v: unknown): boolean => {
    if (v == null) return false
    if (Array.isArray(v)) return v.some(shareable)
    if (typeof v === 'object') return Object.values(v).some(shareable)
    if (typeof v === 'string') return v.length > 0
    return true
  }
  return shareable(out) ? JSON.stringify(out) : ''
}

/**
 * xhttp padding a client must know, or the server turns its handshakes down:
 * x_padding_bytes for sing-box clients and the whole of it in extra for Xray.
 */
function xhttpPadding(x: Json): Json {
  const out: Json = {}
  if (typeof x.xPaddingBytes === 'string' && x.xPaddingBytes.length > 0) out.xPaddingBytes = x.xPaddingBytes
  if (x.xPaddingObfsMode === true) {
    out.xPaddingObfsMode = true
    for (const k of ['xPaddingKey', 'xPaddingHeader', 'xPaddingPlacement', 'xPaddingMethod']) if (typeof x[k] === 'string' && x[k].length > 0) out[k] = x[k]
  }
  return out
}

/** XHTTP settings only the server reads; a link leaves them out. */
const xhttpServerOnly = ['path', 'host', 'mode', 'scMaxBufferedPosts', 'scStreamUpServerSecs', 'noSSEHeader', 'serverMaxHeaderBytes', 'h2Params']

/**
 * XHTTP settings a client must share with the server, for a link's extra:
 * the padding as xhttpPadding has it, the headers but Host (the link's host
 * says it), and whatever else is set. A setting at Xray's default is left
 * out: an empty one, false, 0, or the 1000000 of scMaxEachPostBytes.
 */
export function xhttpExtra(x: Json): Json {
  const out: Json = xhttpPadding(x)
  const headers: Json = {}
  for (const [k, v] of Object.entries(x.headers ?? {})) {
    const value = Array.isArray(v) ? v[0] : v
    if (k.toLowerCase() !== 'host' && filled(k) && filled(value)) headers[k] = value
  }
  if (Object.keys(headers).length) out.headers = headers
  for (const [k, v] of Object.entries(x)) {
    if (xhttpServerOnly.includes(k) || k === 'headers' || k.startsWith('xPadding')) continue
    if (!filled(v) || v === false || v === 0 || v === '0') continue
    if (typeof v === 'object' && !Object.keys(v).length) continue
    if (k === 'scMaxEachPostBytes' && String(v) === '1000000') continue
    out[k] = v
  }
  return out
}

/**
 * The transport's part of a link's query, as VLESS, Trojan and Shadowsocks
 * share it; what goes right after the type (VLESS's encryption) comes in after.
 * Empty values are left out, as the share link standard asks (Xray-core
 * discussion 716): the client takes its default for them.
 */
function transportParams(st: Json, params: Map<string, unknown>, after: Json = {}) {
  const net = st.network
  const s: Json = st[`${net}Settings`] ?? {}
  const set = (k: string, v: unknown) => filled(v) && params.set(k, v)
  params.set('type', net)
  for (const [k, v] of Object.entries(after)) params.set(k, v)
  switch (net) {
    case 'tcp':
      if (s.header?.type === 'http') {
        const request = s.header.request ?? {}
        set('path', (request.path ?? ['/']).join(','))
        set('host', headerOf(request.headers, 'host'))
        params.set('headerType', 'http')
      }
      break
    case 'kcp':
      params.set('mtu', s.mtu)
      params.set('tti', s.tti)
      break
    case 'ws':
    case 'httpupgrade':
      set('path', s.path)
      set('host', hostOf(s))
      break
    case 'grpc':
      set('serviceName', s.serviceName)
      // The standard lets the authority be empty.
      params.set('authority', s.authority)
      if (s.multiMode) params.set('mode', 'multi')
      break
    case 'xhttp': {
      set('path', s.path)
      set('host', hostOf(s))
      params.set('mode', s.mode)
      const extra = xhttpExtra(s)
      if (extra.xPaddingBytes) params.set('x_padding_bytes', extra.xPaddingBytes)
      if (Object.keys(extra).length) params.set('extra', JSON.stringify(extra))
      break
    }
  }
  const fm = finalMask(st)
  if (fm) params.set('fm', fm)
}

/**
 * TLS of the inbound in a link's query, without an empty fingerprint or ALPN
 * (the client then takes chrome and the core's choice); given a flow, VLESS's
 * Vision flow over raw TCP too. The SNI, fingerprint and ALPN an External
 * Proxy row with Force TLS sets come over the inbound's own.
 */
function tlsParams(st: Json, params: Map<string, unknown>, flow?: string, ep?: Json) {
  params.set('security', 'tls')
  if (st.security === 'tls') {
    const t = st.tlsSettings ?? {}
    if (filled(t.settings?.fingerprint)) params.set('fp', t.settings.fingerprint)
    if (t.alpn?.length > 0) params.set('alpn', t.alpn)
    if (filled(t.serverName)) params.set('sni', t.serverName)
    if (t.settings?.echConfigList?.length > 0) params.set('ech', t.settings.echConfigList)
    if (flow !== undefined && st.network === 'tcp' && filled(flow)) params.set('flow', flow)
  }
  for (const [k, v] of Object.entries(epTls(ep))) params.set(k, v)
}

/** The TLS an External Proxy row with Force TLS sets, as a link's sni, fp and alpn; each when given. */
function epTls(ep?: Json): Record<string, string> {
  const out: Record<string, string> = {}
  if (ep?.forceTls !== 'tls') return out
  if (filled(ep.sni)) out.sni = ep.sni
  if (filled(ep.fingerprint)) out.fp = ep.fingerprint
  if (Array.isArray(ep.alpn) && ep.alpn.length) out.alpn = ep.alpn.join(',')
  return out
}

/** Reality of the inbound in a link's query: the first server name and the first short ID. */
function realityParams(st: Json, params: Map<string, unknown>, flow?: string) {
  const r = st.realitySettings ?? {}
  const names = ([] as string[]).concat(r.serverNames ?? []).join(',')
  const ids = ([] as string[]).concat(r.shortIds ?? []).join(',')
  params.set('security', 'reality')
  params.set('pbk', r.settings?.publicKey)
  if (filled(r.settings?.fingerprint)) params.set('fp', r.settings.fingerprint)
  // Without a server name the client takes the address, so an empty one is left out.
  if (names.split(',')[0]) params.set('sni', names.split(',')[0])
  if (ids.length > 0) params.set('sid', ids.split(',')[0])
  if (filled(r.settings?.spiderX)) params.set('spx', r.settings.spiderX)
  if (filled(r.settings?.mldsa65Verify)) params.set('pqv', r.settings.mldsa65Verify)
  if (flow !== undefined && st.network === 'tcp' && filled(flow)) params.set('flow', flow)
}

/** A link of the form scheme://user@address:port?query#remark. */
function urlLink(base: string, params: Map<string, unknown>, remark: string) {
  const url = new URL(base)
  for (const [k, v] of params) url.searchParams.set(k, String(v))
  url.hash = encodeURIComponent(remark)
  return url.toString()
}

const securityOf = (st: Json, forceTls: string) => (forceTls === 'same' ? st.security : forceTls)

// --- One link per protocol ---------------------------------------------------------

function vmessLink(f: InboundForm, address: string, port: number, forceTls: string, remark: string, c: Json, ep?: Json) {
  const st = f.streamSettings ?? {}
  const tls = securityOf(st, forceTls)
  const s: Json = st[`${st.network}Settings`] ?? {}
  const obj: Json = { v: '2', ps: remark, add: address, port, id: c.id, scy: c.security ?? 'auto', net: st.network, tls }
  switch (st.network) {
    case 'tcp':
      obj.type = s.header?.type
      if (s.header?.type === 'http') {
        const request = s.header.request ?? {}
        obj.path = (request.path ?? ['/']).join(',')
        const host = header(request.headers, 'host')
        if (host) obj.host = host
      }
      break
    case 'kcp':
      obj.mtu = s.mtu
      obj.tti = s.tti
      break
    case 'ws':
    case 'httpupgrade':
      obj.path = s.path
      obj.host = hostOf(s)
      break
    case 'grpc':
      obj.path = s.serviceName
      obj.authority = s.authority
      if (s.multiMode) obj.type = 'multi'
      break
    case 'xhttp': {
      obj.path = s.path
      obj.host = hostOf(s)
      obj.type = s.mode
      const pad = xhttpPadding(s)
      if (pad.xPaddingBytes) obj.x_padding_bytes = pad.xPaddingBytes
      delete pad.xPaddingBytes
      Object.assign(obj, pad)
      break
    }
  }
  const fm = finalMask(st)
  if (fm) obj.fm = fm
  if (tls === 'tls') {
    const t = st.tlsSettings ?? defaultTls()
    if (filled(t.serverName)) obj.sni = t.serverName
    if (filled(t.settings?.fingerprint)) obj.fp = t.settings.fingerprint
    if (t.alpn?.length > 0) obj.alpn = t.alpn.join(',')
    Object.assign(obj, epTls(ep))
  }
  return 'vmess://' + base64(JSON.stringify(obj, null, 2))
}

function vlessLink(f: InboundForm, address: string, port: number, forceTls: string, remark: string, c: Json, ep?: Json) {
  const st = f.streamSettings ?? {}
  const params = new Map<string, unknown>()
  transportParams(st, params, { encryption: f.settings.encryption ?? 'none' })
  const sec = securityOf(st, forceTls)
  if (sec === 'tls') tlsParams(st, params, c.flow ?? '', ep)
  else if (sec === 'reality') realityParams(st, params, c.flow ?? '')
  else params.set('security', 'none')
  return urlLink(`vless://${c.id}@${address}:${port}`, params, remark)
}

function trojanLink(f: InboundForm, address: string, port: number, forceTls: string, remark: string, password: string, ep?: Json) {
  const st = f.streamSettings ?? {}
  const params = new Map<string, unknown>()
  transportParams(st, params)
  const sec = securityOf(st, forceTls)
  if (sec === 'tls') tlsParams(st, params, undefined, ep)
  else if (sec === 'reality') realityParams(st, params)
  else params.set('security', 'none')
  return urlLink(`trojan://${password}@${address}:${port}`, params, remark)
}

function ssLink(f: InboundForm, address: string, port: number, forceTls: string, remark: string, clientPassword = '', ep?: Json) {
  const st = f.streamSettings ?? {}
  const s = f.settings
  const params = new Map<string, unknown>()
  transportParams(st, params)
  if (securityOf(st, forceTls) === 'tls') tlsParams(st, params, undefined, ep)
  const password: string[] = []
  if (isSS2022(s.method)) password.push(s.password)
  if (isSSMultiUser(s.method)) password.push(clientPassword)
  return urlLink(`ss://${base64(`${s.method}:${password.join(':')}`, true)}@${address}:${port}`, params, remark)
}

function hysteriaLink(f: InboundForm, address: string, port: number, remark: string, auth: string) {
  const st = f.streamSettings ?? {}
  const t = st.tlsSettings ?? defaultTls()
  const params = new Map<string, unknown>()
  params.set('security', 'tls')
  if (t.settings?.fingerprint?.length > 0) params.set('fp', t.settings.fingerprint)
  if (t.alpn?.length > 0) params.set('alpn', t.alpn)
  if (t.settings?.echConfigList?.length > 0) params.set('ech', t.settings.echConfigList)
  if (t.serverName?.length > 0) params.set('sni', t.serverName)
  const salamander = (Array.isArray(st.finalmask?.udp) ? st.finalmask.udp : []).find((m: Json) => m?.type === 'salamander')
  const obfs = salamander?.settings?.password
  if (typeof obfs === 'string' && obfs.length > 0) {
    params.set('obfs', 'salamander')
    params.set('obfs-password', obfs)
  }
  const fm = finalMask(st)
  if (fm) params.set('fm', fm)
  return urlLink(`${(f.settings.version ?? 2) == 2 ? 'hysteria2' : 'hysteria'}://${auth}@${address}:${port}`, params, remark)
}

/** Telegram's tg://proxy: the ee prefix selects Fake-TLS, the masking domain follows the secret as hex. */
function mtprotoLink(f: InboundForm, address: string, port: number, secret = '') {
  const domain = [...new TextEncoder().encode(f.settings.tlsDomain || '')].map((b) => b.toString(16).padStart(2, '0')).join('')
  return `tg://proxy?server=${address}&port=${port}&secret=ee${secret}${domain}`
}

/**
 * The WEB proxy's tg://webproxy: the host and the secret path, no port as
 * Telegram always uses 443; the secret is 0x70, then the dd-prefixed secret,
 * as unpadded base64url.
 */
function mtprotoWebLink(f: InboundForm, secret = '') {
  const web = f.settings.web ?? {}
  const bytes = [0x70, 0xdd, ...(secret.match(/../g) ?? []).map((h) => parseInt(h, 16))]
  const encoded = btoa(String.fromCharCode(...bytes)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
  return `tg://webproxy?server=${encodeURIComponent(`${web.host}/${web.path}`)}&secret=${encoded}`
}

/** Whether the MTProto inbound's WEB proxy has what its links need. */
const webReady = (f: InboundForm) => !!f.settings.web?.enable && filled(f.settings.web?.host) && filled(f.settings.web?.path)

/**
 * One client's link to the address and port; forceTls "same" keeps the
 * inbound's security. ep is the External Proxy row the link is for.
 */
export function genLink(f: InboundForm, address: string, port: number, forceTls: string, remark: string, client: Json | undefined, ep?: Json): string {
  const c = client ?? {}
  switch (f.protocol) {
    case 'vmess':
      return vmessLink(f, address, port, forceTls, remark, c, ep)
    case 'vless':
      return vlessLink(f, address, port, forceTls, remark, c, ep)
    case 'shadowsocks':
      return ssLink(f, address, port, forceTls, remark, isSSMultiUser(f.settings.method) ? c.password : '', ep)
    case 'trojan':
      return trojanLink(f, address, port, forceTls, remark, c.password, ep)
    case 'hysteria':
      return hysteriaLink(f, address, port, remark, c.auth?.length > 0 ? c.auth : (f.streamSettings?.hysteriaSettings?.auth ?? ''))
    case 'mtproto':
      return mtprotoLink(f, address, port, c.id)
    default:
      return ''
  }
}

// --- All of them -------------------------------------------------------------------

/**
 * A client's links: one to the inbound itself, or one per External Proxy
 * address, and MTProto's WEB proxy; each named by the remark model, its first
 * character the separator and the rest the order of the inbound's remark (i),
 * the email (e) and the External Proxy's remark (o).
 */
export function allLinks(f: InboundForm, remarkModel = defaultRemarkModel, client?: Json): Link[] {
  const sep = remarkModel.charAt(0)
  const order = remarkModel.slice(1).split('')
  const parts: Record<string, string> = { i: f.remark, e: client?.email ?? '', o: '' }
  const name = () => order.map((ch) => parts[ch] ?? '').filter((x) => x.length > 0).join(sep)
  const out: Link[] = []
  const proxies: Json[] = f.streamSettings?.externalProxy ?? []
  if (!proxies.length) {
    const r = name()
    out.push({ remark: r, link: genLink(f, linkAddress(f), f.port ?? 0, 'same', r, client) })
  } else {
    for (const ep of proxies) {
      parts.o = ep.remark ?? ''
      const r = name()
      out.push({ remark: r, link: genLink(f, ep.dest, ep.port, ep.forceTls, r, client, ep) })
    }
  }
  if (f.protocol === 'mtproto' && client && webReady(f)) {
    parts.o = 'WEB'
    out.push({ remark: name(), link: mtprotoWebLink(f, client.id) })
  }
  return out
}

/** The peers, an address without a mask taken as one host. */
const peersOf = (f: InboundForm): Json[] =>
  (Array.isArray(f.settings.peers) ? f.settings.peers : []).map((p: Json) => ({ ...p, allowedIPs: (p.allowedIPs ?? ['10.0.0.2/32']).map(withMask) }))

/** WireGuard's public key, from its secret key. */
export function wireguardPublicKey(secretKey: string) {
  try {
    return secretKey ? wireguardKeys(secretKey).publicKey : ''
  } catch {
    return ''
  }
}

/** WireGuard's settings with the MTU the old panel took when none is set. */
const wireguardSettings = (f: InboundForm): Json => ({ ...f.settings, mtu: f.settings.mtu ?? 1420 })

/** A peer's config file, as wg-quick reads it. */
function wireguardConfig(f: InboundForm, address: string, port: number, remark: string, peer: Json) {
  const s = wireguardSettings(f)
  let txt = '[Interface]\n'
  txt += `PrivateKey = ${peer.privateKey}\n`
  txt += `Address = ${peer.allowedIPs?.[0]}\n`
  txt += 'DNS = 1.1.1.1, 1.0.0.1\n'
  if (s.mtu) txt += `MTU = ${s.mtu}\n`
  txt += `\n# ${remark}\n`
  txt += '[Peer]\n'
  txt += `PublicKey = ${wireguardPublicKey(s.secretKey)}\n`
  txt += 'AllowedIPs = 0.0.0.0/0, ::/0\n'
  txt += `Endpoint = ${address}:${port}`
  if (peer.preSharedKey) txt += `\nPresharedKey = ${peer.preSharedKey}`
  if (peer.keepAlive) txt += `\nPersistentKeepalive = ${peer.keepAlive}\n`
  return txt
}

/** A peer's wireguard:// link. */
function wireguardLink(f: InboundForm, address: string, port: number, remark: string, peer: Json) {
  const s = wireguardSettings(f)
  const url = new URL(`wireguard://${address}:${port}`)
  url.username = peer.privateKey || ''
  const pub = wireguardPublicKey(s.secretKey)
  if (pub) url.searchParams.set('publickey', pub)
  if (Array.isArray(peer.allowedIPs) && peer.allowedIPs[0]) url.searchParams.set('address', peer.allowedIPs[0])
  if (s.mtu) url.searchParams.set('mtu', String(s.mtu))
  url.hash = encodeURIComponent(remark)
  return url.toString()
}


/** Each WireGuard peer's config, named after the inbound and the peer's number. */
export const wireguardConfigs = (f: InboundForm, remarkModel = defaultRemarkModel, address = linkAddress(f)) =>
  peersOf(f).map((p, i) => wireguardConfig(f, address, f.port ?? 0, f.remark + remarkModel.charAt(0) + (i + 1), p))

/** Each WireGuard peer's link, named as its config. */
export const wireguardLinks = (f: InboundForm, remarkModel = defaultRemarkModel, address = linkAddress(f)) =>
  peersOf(f).map((p, i) => wireguardLink(f, address, f.port ?? 0, f.remark + remarkModel.charAt(0) + (i + 1), p))

/**
 * Everything an inbound shares: every client's links; for Shadowsocks
 * with one key the inbound's link; for WireGuard the peers' configs.
 */
export function inboundLinks(f: InboundForm, remarkModel = defaultRemarkModel): string[] {
  if (withClients(f.protocol, f.settings)) {
    const clients: Json[] = Array.isArray(f.settings.clients) ? f.settings.clients : []
    return clients.flatMap((c) => allLinks(f, remarkModel, c).map((l) => l.link))
  }
  if (f.protocol === 'shadowsocks') return [ssLink(f, linkAddress(f), f.port ?? 0, 'same', f.remark)]
  if (f.protocol === 'wireguard') return wireguardConfigs(f, remarkModel)
  return []
}

// --- The server's IPv4 ------------------------------------------------------------

/**
 * A link or a WireGuard config that reaches the server at its public IPv4
 * instead of the address it names, as the old QR window's switch did: the
 * Endpoint of a config, the server of tg://proxy, the address in a VMess
 * link's JSON, the host of the others. MTProto's WEB proxy keeps its domain,
 * which its TLS needs.
 */
export function withIPv4(link: string, ipv4: string): string {
  if (!ipv4) return link
  if (link.startsWith('[Interface]')) return link.replace(/Endpoint = (.+):(\d+)/, `Endpoint = ${ipv4}:$2`)
  if (link.startsWith('tg://webproxy')) return link
  if (link.startsWith('vmess://')) {
    try {
      const obj = JSON.parse(new TextDecoder().decode(Uint8Array.from(atob(link.slice(8)), (ch) => ch.charCodeAt(0))))
      return 'vmess://' + base64(JSON.stringify({ ...obj, add: ipv4 }, null, 2))
    } catch {
      return link
    }
  }
  try {
    const url = new URL(link)
    if (link.startsWith('tg://proxy')) url.searchParams.set('server', ipv4)
    else url.hostname = ipv4
    return url.toString()
  } catch {
    return link
  }
}

// --- Fallbacks ---------------------------------------------------------------------

/**
 * An inbound for its links. One listening on a socket (@…) is a fallback of a
 * VLESS or Trojan inbound over TCP: its clients reach it at that inbound's
 * address and port, with its security, TLS and External Proxy.
 */
export function linkForm(ib: DbInbound, all: DbInbound[]): InboundForm {
  if (!ib.listen?.startsWith('@')) return fromDb(ib)
  const root = all
    .filter((x) => ['trojan', 'vless'].includes(x.protocol))
    .map(fromDb)
    .find((x) => x.streamSettings?.network === 'tcp' && (x.settings.fallbacks ?? []).some((fb: Json) => fb.dest === ib.listen))
  if (!root) return fromDb(ib)
  const st: Json = JSON.parse(ib.streamSettings || '{}')
  const rs = root.streamSettings ?? {}
  st.security = rs.security
  if (rs.tlsSettings) st.tlsSettings = rs.tlsSettings
  else delete st.tlsSettings
  st.externalProxy = rs.externalProxy
  return fromDb({ ...ib, listen: root.listen, port: root.port ?? ib.port, streamSettings: JSON.stringify(st) })
}

/** Subscription links of the clients that have one, each once. */
export function subLinks(clients: Json[], subURI: string): string[] {
  return [...new Set(clients.filter((c) => c.subId?.length > 0).map((c) => subURI + c.subId))]
}
