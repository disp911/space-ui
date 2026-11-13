// An outbound as the outbound window edits it: Xray's own JSON, with the
// rules of the old panel's Outbound class for what goes with what, the
// defaults it filled in, the cleanup it did on save and the links it read.

import type { Outbound } from './model'

type Json = Record<string, any>

export const protocols = ['freedom', 'blackhole', 'dns', 'vmess', 'vless', 'trojan', 'shadowsocks', 'wireguard', 'hysteria', 'socks', 'http'] as const
export type Protocol = (typeof protocols)[number]

export const ssMethods = [
  'aes-256-gcm',
  'aes-128-gcm',
  'chacha20-poly1305',
  'chacha20-ietf-poly1305',
  'xchacha20-poly1305',
  'xchacha20-ietf-poly1305',
  '2022-blake3-aes-128-gcm',
  '2022-blake3-aes-256-gcm',
  '2022-blake3-chacha20-poly1305',
]
export const flows = ['xtls-rprx-vision', 'xtls-rprx-vision-udp443']
export const fingerprints = ['chrome', 'firefox', 'safari', 'ios', 'android', 'edge', '360', 'qq', 'random', 'randomized', 'randomizednoalpn', 'unsafe']
export const alpns = ['h3', 'h2', 'http/1.1']
export const domainStrategies = ['AsIs', 'UseIP', 'UseIPv4', 'UseIPv6', 'UseIPv6v4', 'UseIPv4v6', 'ForceIP', 'ForceIPv6v4', 'ForceIPv6', 'ForceIPv4v6', 'ForceIPv4']
export const vmessSecurities = ['aes-128-gcm', 'chacha20-poly1305', 'auto']
export const xhttpModes = ['auto', 'packet-up', 'stream-up', 'stream-one']
export const tcpCongestions = ['bbr', 'cubic', 'reno']
export const addressPortStrategies = ['none', 'srvportonly', 'srvaddressonly', 'srvportandaddress', 'txtportonly', 'txtaddressonly', 'txtportandaddress']
export const forwardedForHeaders = ['CF-Connecting-IP', 'X-Real-IP', 'True-Client-IP', 'X-Client-IP']
export const networks = [
  { value: 'tcp', label: 'TCP (RAW)' },
  { value: 'kcp', label: 'mKCP' },
  { value: 'ws', label: 'WebSocket' },
  { value: 'grpc', label: 'gRPC' },
  { value: 'httpupgrade', label: 'HTTPUpgrade' },
  { value: 'xhttp', label: 'XHTTP' },
]

/** A random UUID v4, also over HTTP where crypto.randomUUID is missing. */
export function uuid() {
  const b = crypto.getRandomValues(new Uint8Array(16))
  b[6] = (b[6]! & 0x0f) | 0x40
  b[8] = (b[8]! & 0x3f) | 0x80
  const h = [...b].map((x) => x.toString(16).padStart(2, '0')).join('')
  return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`
}

/** A list from the JSON for v-for, empty when the key holds none. */
export const items = (v: unknown): Json[] => (Array.isArray(v) ? v : [])

// --- What goes with what -----------------------------------------------------------

const proxies = ['vmess', 'vless', 'trojan', 'shadowsocks']

export const canStream = (o: Outbound) => [...proxies, 'hysteria'].includes(o.protocol)
export const hasAddressPort = (o: Outbound) => ['dns', 'vmess', 'vless', 'trojan', 'shadowsocks', 'socks', 'http', 'hysteria'].includes(o.protocol)
export const hasUser = (o: Outbound) => ['socks', 'http'].includes(o.protocol)

export function canTls(o: Outbound) {
  if (o.protocol === 'hysteria') return true
  return proxies.includes(o.protocol) && ['tcp', 'ws', 'http', 'grpc', 'httpupgrade', 'xhttp'].includes(network(o))
}

export const canReality = (o: Outbound) => ['vless', 'trojan'].includes(o.protocol) && ['tcp', 'http', 'grpc', 'xhttp'].includes(network(o))

/** xtls-rprx-vision: VLESS over raw TCP with TLS or Reality. */
export const canFlow = (o: Outbound) => o.protocol === 'vless' && security(o) !== 'none' && network(o) === 'tcp'

export const canVisionSeed = (o: Outbound) => canFlow(o) && flows.includes(flowOf(o))

/** Not with a flow or over XHTTP, and only for the proxies that take it. */
export const canMux = (o: Outbound) => !flowOf(o) && network(o) !== 'xhttp' && [...proxies, 'http', 'socks'].includes(o.protocol)

export const network = (o: Outbound): string => o.streamSettings?.network ?? 'tcp'
export const security = (o: Outbound): string => o.streamSettings?.security ?? 'none'

// --- Where the fields live ---------------------------------------------------------

/**
 * The object that holds address and port: the first of vnext or servers, or
 * settings itself (VLESS in its flat form, DNS, Hysteria).
 */
export function server(o: Outbound): Json {
  const s = (o.settings ??= {})
  if (Array.isArray(s.vnext)) return (s.vnext[0] ??= {})
  if (Array.isArray(s.servers)) return (s.servers[0] ??= {})
  return s
}

/** The user of a vnext entry, or the server entry itself where there is none. */
export function user(o: Outbound): Json {
  const s = server(o)
  if (o.protocol === 'vmess' || (o.protocol === 'vless' && Array.isArray(o.settings?.vnext))) return ((s.users ??= [])[0] ??= {})
  return s
}

const flowOf = (o: Outbound): string => (o.protocol === 'vless' ? (user(o).flow ?? '') : '')

// --- Defaults ----------------------------------------------------------------------

export function defaultSettings(protocol: string): Json {
  switch (protocol) {
    case 'dns':
      return { network: 'udp', address: '', port: 53 }
    case 'vmess':
      return { vnext: [{ address: '', port: 443, users: [{ id: uuid(), security: 'auto' }] }] }
    case 'vless':
      return { address: '', port: 443, id: uuid(), flow: '', encryption: 'none' }
    case 'trojan':
      return { servers: [{ address: '', port: 443, password: '' }] }
    case 'shadowsocks':
      return { servers: [{ address: '', port: 443, password: '', method: ssMethods[0] }] }
    case 'socks':
    case 'http':
      return { servers: [{ address: '', port: 1080, users: [] }] }
    case 'wireguard':
      return { mtu: 1420, secretKey: '', address: [], peers: [defaultPeer()], noKernelTun: false }
    case 'hysteria':
      return { address: '', port: 443, version: 2 }
    default:
      return {}
  }
}

export const defaultPeer = () => ({ publicKey: '', allowedIPs: ['0.0.0.0/0', '::/0'], endpoint: '', keepAlive: 0 })

export function defaultNetwork(net: string): Json {
  switch (net) {
    case 'tcp':
      return { header: { type: 'none' } }
    case 'kcp':
      return { mtu: 1350, tti: 20, uplinkCapacity: 5, downlinkCapacity: 20, cwndMultiplier: 1, maxSendingWindow: 1350 }
    case 'ws':
      return { path: '/', host: '', heartbeatPeriod: 0 }
    case 'grpc':
      return { serviceName: '', authority: '', multiMode: false }
    case 'httpupgrade':
      return { path: '/', host: '' }
    case 'xhttp':
      return { path: '/', host: '', mode: 'auto', noGRPCHeader: false, scMinPostsIntervalMs: '30' }
    case 'hysteria':
      // Congestion, speeds and windows are QUIC params, a hop range a UDP mask.
      return { version: 2, auth: '' }
    default:
      return {}
  }
}

export const defaultXmux = () => ({
  maxConcurrency: '16-32',
  maxConnections: 0,
  cMaxReuseTimes: 0,
  hMaxRequestTimes: '600-900',
  hMaxReusableSecs: '1800-3000',
  hKeepAlivePeriod: 0,
})

export const defaultTls = () => ({ serverName: '', alpn: [], fingerprint: '', echConfigList: '', verifyPeerCertByName: '', pinnedPeerCertSha256: '' })
export const defaultReality = () => ({ publicKey: '', fingerprint: 'chrome', serverName: '', shortId: '', spiderX: '', mldsa65Verify: '' })
export const defaultSockopt = () => ({
  dialerProxy: '',
  tcpFastOpen: false,
  tcpKeepAliveInterval: 0,
  tcpMptcp: false,
  penetrate: false,
  addressPortStrategy: 'none',
  domainStrategy: 'AsIs',
  tcpKeepAliveIdle: 0,
  mark: 0,
  tcpMaxSeg: 0,
  tcpUserTimeout: 0,
  tcpWindowClamp: 0,
  tcpCongestion: '',
  interface: '',
})
export const defaultMux = () => ({ enabled: true, concurrency: 8, xudpConcurrency: 16, xudpProxyUDP443: 'reject' })
export const defaultFinalRule = () => ({ action: 'block', network: '', port: '', ip: [], delay: '', logLevel: '' })
export const defaultDnsRule = () => ({ action: 'direct', qType: '', domain: '' })
export const defaultQuic = () => ({
  congestion: 'bbr',
  initStreamReceiveWindow: 8388608,
  maxStreamReceiveWindow: 8388608,
  initConnectionReceiveWindow: 20971520,
  maxConnectionReceiveWindow: 20971520,
  maxIdleTimeout: 30,
  keepAlivePeriod: 5,
  maxIncomingStreams: 1024,
})

/** A new outbound: VLESS over raw TCP, as the old panel began. */
export function newOutbound(protocol = 'vless'): Outbound {
  const o: Outbound = { tag: '', protocol, settings: defaultSettings(protocol) }
  setProtocolStream(o)
  return o
}

/** A protocol with a transport starts on raw TCP without security; Hysteria on its own transport. */
export function setProtocolStream(o: Outbound) {
  const sockopt = o.streamSettings?.sockopt
  if (o.protocol === 'hysteria') o.streamSettings = { network: 'hysteria', security: 'none', hysteriaSettings: defaultNetwork('hysteria') }
  else if (canStream(o)) o.streamSettings = { network: 'tcp', security: 'none', tcpSettings: defaultNetwork('tcp') }
  else delete o.streamSettings
  if (sockopt) (o.streamSettings ??= {}).sockopt = sockopt
}

// --- Saving ------------------------------------------------------------------------

const empty = (v: unknown) =>
  v === undefined || v === null || v === '' || (Array.isArray(v) && !v.length) || (typeof v === 'object' && !Array.isArray(v) && !Object.keys(v as object).length)

const netKeys = ['tcpSettings', 'kcpSettings', 'wsSettings', 'grpcSettings', 'httpupgradeSettings', 'xhttpSettings', 'hysteriaSettings', 'httpSettings', 'quicSettings', 'splithttpSettings']

/**
 * The outbound as it goes into the template: the settings of the transport and
 * the security in use only, no empty optional fields, Mux only when on.
 */
export function finalize(input: Outbound): Outbound {
  const o: Outbound = JSON.parse(JSON.stringify(input))
  o.tag = String(o.tag ?? '').trim()
  if (empty(o.tag)) delete o.tag
  if (empty(o.sendThrough)) delete o.sendThrough
  if (!o.mux?.enabled || !canMux(o)) delete o.mux

  const st = o.streamSettings
  if (st) {
    // None of them is the system's own.
    for (const k of ['tcpCongestion', 'interface']) if (st.sockopt && empty(st.sockopt[k])) delete st.sockopt[k]
    if (!canStream(o)) {
      // Freedom keeps its masks: Fragment and Noise live there since Xray 26.
      const finalmask = o.protocol === 'freedom' && (!empty(st.finalmask?.tcp) || !empty(st.finalmask?.udp)) ? st.finalmask : undefined
      if (finalmask && empty(finalmask.tcp)) delete finalmask.tcp
      if (finalmask && empty(finalmask.udp)) delete finalmask.udp
      o.streamSettings = { ...(st.sockopt ? { sockopt: st.sockopt } : {}), ...(finalmask ? { finalmask } : {}) }
    } else {
      const keep = `${st.network}Settings`
      for (const k of netKeys) if (k !== keep) delete st[k]
      if (st.security !== 'tls') delete st.tlsSettings
      if (st.security !== 'reality') delete st.realitySettings
      if (st.finalmask && empty(st.finalmask.tcp) && empty(st.finalmask.udp) && !st.finalmask.quicParams) delete st.finalmask
      if (st.sockopt && empty(st.sockopt.trustedXForwardedFor)) delete st.sockopt.trustedXForwardedFor
    }
    if (empty(o.streamSettings)) delete o.streamSettings
  }

  const s = o.settings ?? {}
  if (o.protocol === 'vless') {
    const u = user(o)
    if (empty(u.flow)) {
      delete u.testpre
      delete u.testseed
    }
    if (empty(s.reverse?.tag)) delete s.reverse
  }
  if (o.protocol === 'freedom') {
    // Xray takes a delay and a log level on block rules only, and will not start with them on an allow rule.
    // blockDelay, its old name, would win over delay in Xray, so it goes; the window shows delay only.
    for (const r of Array.isArray(s.finalRules) ? s.finalRules : []) {
      delete r.blockDelay
      for (const k of ['delay', 'logLevel']) if (r.action !== 'block' || empty(r[k])) delete r[k]
    }
    for (const k of ['domainStrategy', 'redirect', 'fragment', 'noises', 'finalRules']) if (empty(s[k])) delete s[k]
  }
  if (o.protocol === 'wireguard') {
    if (empty(s.reserved)) delete s.reserved
    s.remoteDNS = (s.remoteDNS ?? []).map((v: string) => String(v).trim()).filter(Boolean)
    if (empty(s.remoteDNS)) delete s.remoteDNS
    for (const p of s.peers ?? []) if (empty(p.preSharedKey)) delete p.preSharedKey
  }
  if (o.protocol === 'blackhole' && empty(s.response?.type)) delete s.response
  return o
}

/** Bytes of any base64, standard or URL-safe, padded or not, as hex; null when it is none. */
function base64Hex(v: string): string | null {
  try {
    const std = v.replace(/-/g, '+').replace(/_/g, '/').replace(/=+$/, '')
    const bytes = atob(std + '='.repeat((4 - (std.length % 4)) % 4))
    return [...bytes].map((c) => c.charCodeAt(0).toString(16).padStart(2, '0')).join('')
  } catch {
    return null
  }
}

/**
 * A freedom outbound saved with the fragment and noises of its settings, which
 * Xray 26 takes with a warning, gets them as the masks of its stream that took
 * their place: the fragment's interval is the mask's delay, a random noise
 * its rand, base64 hex (the mask reads standard base64 only). The noises'
 * applyTo has no place there. Done in the window: saved, it is the new way.
 */
export function liftFreedomMasks(o: Outbound): Outbound {
  if (o.protocol !== 'freedom' || !o.settings) return o
  const s = o.settings
  const fm = ((o.streamSettings ??= {}).finalmask ??= {})
  if (s.fragment && typeof s.fragment === 'object' && Object.keys(s.fragment).length) {
    const { interval, ...rest } = s.fragment
    ;(fm.tcp ??= []).push({ type: 'fragment', settings: { ...rest, delay: rest.delay ?? interval ?? '' } })
  }
  delete s.fragment
  if (Array.isArray(s.noises) && s.noises.length) {
    const noise = s.noises.flatMap((n: Record<string, any>) => {
      const delay = n.delay ?? ''
      const packet = String(n.packet ?? '').trim()
      if (n.type === 'rand') return [{ type: 'array', rand: packet, randRange: '0-255', delay }]
      if (n.type === 'base64') {
        const hex = base64Hex(packet)
        return hex === null ? [] : [{ type: 'hex', packet: hex, delay }]
      }
      return [{ type: n.type, packet, delay }]
    })
    if (noise.length) (fm.udp ??= []).push({ type: 'noise', settings: { reset: 0, noise } })
  }
  delete s.noises
  if (!fm.tcp && !fm.udp) {
    delete o.streamSettings!.finalmask
    if (!Object.keys(o.streamSettings!).length) delete o.streamSettings
  }
  return o
}

/** Go's IsPrivate, IsLoopback and IsLinkLocalUnicast of an IPv4 address. */
function privateV4(ip: string) {
  const [a, b] = ip.split('.').map(Number) as [number, number]
  return a === 10 || a === 127 || (a === 172 && b >= 16 && b <= 31) || (a === 192 && b === 168) || (a === 169 && b === 254)
}

/** The zones Xray counts as private (common/geodata), and a name without a dot. */
const privateZones = ['lan', 'localdomain', 'example', 'invalid', 'localhost', 'test', 'local', 'home.arpa', 'internal']

/** A server address Xray lets VLESS and Trojan reach without TLS: a private, loopback or link-local IP, or a private name. */
export function privateAddress(address: string): boolean {
  const a = address.trim().toLowerCase().replace(/^\[|\]$/g, '').replace(/\.$/, '')
  if (/^\d+\.\d+\.\d+\.\d+$/.test(a)) return privateV4(a)
  if (a.includes(':')) {
    const mapped = /^::ffff:(\d+\.\d+\.\d+\.\d+)$/.exec(a)
    if (mapped) return privateV4(mapped[1]!)
    if (a === '::1') return true
    const first = a.startsWith('::') ? 0 : parseInt(a.split(':')[0] || '0', 16)
    return (first & 0xfe00) === 0xfc00 || (first & 0xffc0) === 0xfe80
  }
  if (/^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$/.test(a)) return true
  return privateZones.some((z) => a === z || a.endsWith(`.${z}`))
}

/**
 * VLESS or Trojan to a public address with neither TLS nor REALITY, nor
 * VLESS's own encryption: Xray does not start on it ("vless without TLS or
 * other encryption is prohibited unless the server address is a private IP
 * or domain").
 */
export function plainWrong(o: Outbound): boolean {
  if (o.protocol !== 'vless' && o.protocol !== 'trojan') return false
  if (['tls', 'reality'].includes(String(o.streamSettings?.security ?? ''))) return false
  if (o.protocol === 'vless') {
    const enc = String(user(o).encryption ?? '')
    if (enc && enc !== 'none') return false
  }
  return !privateAddress(String(server(o).address ?? ''))
}

/** A blackhole's custom response that is no standard base64: Xray does not start on one. */
export const blackholeWrong = (o: Outbound) => {
  if (o.protocol !== 'blackhole' || o.settings?.response?.type !== 'custom') return false
  const data = String(o.settings.response.customResponseData ?? '')
  if (!data || !/^[A-Za-z0-9+/]*={0,2}$/.test(data) || data.length % 4) return true
  try {
    atob(data)
    return false
  } catch {
    return true
  }
}

/** An IPv4 or IPv6 address as Go's netip reads it: no leading zeros, no mask, no port. */
export function isIp(v: string): boolean {
  if (/^(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}$/.test(v)) return true
  if (!v.includes(':') || /[^0-9a-fA-F:.]/.test(v)) return false
  try {
    new URL(`http://[${v}]/`)
    return true
  } catch {
    return false
  }
}

/**
 * A WireGuard outbound's tunnel DNS that is no IP: Xray does not start on
 * one (it panics reading it), so the window does not save it.
 */
export const remoteDnsWrong = (o: Outbound) =>
  o.protocol === 'wireguard' && Array.isArray(o.settings?.remoteDNS) && o.settings.remoteDNS.some((v: string) => !!String(v).trim() && !isIp(String(v).trim()))

// --- Links -------------------------------------------------------------------------

/** An outbound from a share link: vmess://, vless://, trojan://, ss://, hysteria2://; null when it is none of them. */
export function fromLink(link: string): Outbound | null {
  const at = link.indexOf('://')
  if (at < 0) return null
  try {
    switch (link.slice(0, at).toLowerCase()) {
      case 'vmess':
        return fromVmess(JSON.parse(decodeBase64(link.slice(at + 3))))
      case 'vless':
      case 'trojan':
      case 'ss':
        return fromParamLink(link)
      case 'hysteria':
      case 'hysteria2':
        return fromHysteria(link)
      default:
        return null
    }
  } catch {
    return null
  }
}

function decodeBase64(text: string) {
  const bin = atob(text.replace(/-/g, '+').replace(/_/g, '/'))
  return new TextDecoder().decode(Uint8Array.from(bin, (c) => c.charCodeAt(0)))
}

function streamFrom(net: string, get: (k: string) => string | undefined): Json {
  const st: Json = { network: net === 'none' ? 'tcp' : net, security: 'none' }
  const host = get('host') ?? ''
  const path = get('path') ?? ''
  switch (st.network) {
    case 'tcp':
      st.tcpSettings = get('headerType') === 'http'
        ? { header: { type: 'http', request: { headers: { Host: host ? host.split(',') : [] }, path: path ? path.split(',') : ['/'] } } }
        : { header: { type: 'none' } }
      break
    case 'kcp': {
      const kcp = defaultNetwork('kcp')
      const mtu = Number(get('mtu'))
      const tti = Number(get('tti'))
      if (mtu > 0) kcp.mtu = mtu
      if (tti > 0) kcp.tti = tti
      st.kcpSettings = kcp
      break
    }
    case 'ws':
      st.wsSettings = { path: path || '/', host }
      break
    case 'grpc':
      st.grpcSettings = { serviceName: get('serviceName') ?? path, authority: get('authority') ?? '', multiMode: get('mode') === 'multi' }
      break
    case 'httpupgrade':
      st.httpupgradeSettings = { path: path || '/', host }
      break
    case 'xhttp': {
      // extra carries what the client shares with the server: padding, ids, XMUX.
      let extra: Json = {}
      try {
        const e = JSON.parse(get('extra') ?? '{}')
        if (e && typeof e === 'object' && !Array.isArray(e)) extra = e
      } catch {
        // A link with a broken extra still connects with what else it says.
      }
      const xh: Json = { ...extra, path: path || '/', host }
      if (get('mode')) xh.mode = get('mode')
      st.xhttpSettings = xh
      break
    }
  }
  return st
}

function fromVmess(j: Json): Outbound {
  const st = streamFrom(j.net ?? 'tcp', (k) => (k === 'headerType' ? j.type : k === 'mode' && j.net === 'grpc' ? (j.type === 'multi' ? 'multi' : '') : j[k]))
  if (j.tls === 'tls') {
    st.security = 'tls'
    st.tlsSettings = { ...defaultTls(), serverName: j.sni ?? '', alpn: j.alpn ? String(j.alpn).split(',') : [], fingerprint: j.fp ?? '' }
  }
  return {
    tag: j.ps ?? '',
    protocol: 'vmess',
    settings: { vnext: [{ address: j.add ?? '', port: Number(j.port) || 443, users: [{ id: j.id ?? '', security: j.scy || 'auto' }] }] },
    streamSettings: st,
  }
}

function fromParamLink(link: string): Outbound | null {
  const url = new URL(link)
  const q = (k: string) => url.searchParams.get(k) ?? undefined
  const st = streamFrom(q('type') ?? 'tcp', q)
  const sec = q('security') ?? 'none'
  if (sec === 'tls') {
    st.security = 'tls'
    st.tlsSettings = { ...defaultTls(), serverName: q('sni') ?? '', alpn: q('alpn') ? q('alpn')!.split(',') : [], fingerprint: q('fp') ?? '', echConfigList: q('ech') ?? '' }
  } else if (sec === 'reality') {
    st.security = 'reality'
    st.realitySettings = {
      publicKey: q('pbk') ?? '',
      fingerprint: q('fp') ?? '',
      serverName: q('sni') ?? '',
      shortId: q('sid') ?? '',
      spiderX: q('spx') ?? '',
      mldsa65Verify: q('pqv') ?? '',
    }
  }
  const m = link.match(/^([^@]+):\/\/([^@]+)@(.+):(\d+)(.*)$/)
  if (!m) return null
  const [, scheme, userData, address, portText] = m
  const port = Number(portText)
  const tag = decodeURIComponent(url.hash.slice(1)) || `out-${scheme === 'ss' ? 'shadowsocks' : scheme}-${port}`
  switch (scheme) {
    case 'vless':
      return {
        tag,
        protocol: 'vless',
        settings: { address, port, id: decodeURIComponent(userData!), flow: q('flow') ?? '', encryption: q('encryption') ?? 'none' },
        streamSettings: st,
      }
    case 'trojan':
      return { tag, protocol: 'trojan', settings: { servers: [{ address, port, password: decodeURIComponent(userData!) }] }, streamSettings: st }
    case 'ss': {
      const [method, ...pass] = decodeBase64(decodeURIComponent(userData!)).split(':')
      return {
        tag,
        protocol: 'shadowsocks',
        settings: { servers: [{ address, port, password: pass.join(':'), method }] },
        streamSettings: st,
      }
    }
    default:
      return null
  }
}

function fromHysteria(link: string): Outbound | null {
  const m = link.match(/^hysteria2?:\/\/([^@]+)@([^:?#]+):(\d+)([^#]*)(#.*)?$/)
  if (!m) return null
  const [, password, address, portText, , hash] = m
  const hy = defaultNetwork('hysteria')
  hy.auth = decodeURIComponent(password!)
  const port = Number(portText)
  return {
    tag: hash ? decodeURIComponent(hash.slice(1)) : `out-hysteria-${port}`,
    protocol: 'hysteria',
    settings: { address, port, version: 2 },
    streamSettings: { network: 'hysteria', security: 'none', hysteriaSettings: hy },
  }
}
