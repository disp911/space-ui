// An inbound as the inbound window edits it: Xray's own JSON for settings,
// streamSettings and sniffing beside the panel's fields, with the rules of the
// old panel's Inbound class (inbound.js of the 2.x panel) for what goes
// with what, the defaults it filled in, the fresh keys a new inbound or client
// gets and the cleanup it did on save.

import { api } from '../../api'
import { wireguardKeys } from '../../wireguard'
import { uuid } from '../xray/outbound'
import type { DbInbound, Protocol } from './model'

type Json = Record<string, any>

export { alpns, domainStrategies, fingerprints, flows, xhttpModes } from '../xray/outbound'

// --- Choices -----------------------------------------------------------------------

export const ssMethods = [
  'chacha20-poly1305',
  'chacha20-ietf-poly1305',
  'xchacha20-ietf-poly1305',
  '2022-blake3-aes-128-gcm',
  '2022-blake3-aes-256-gcm',
  '2022-blake3-chacha20-poly1305',
]
export const tlsVersions = ['1.0', '1.1', '1.2', '1.3']
export const cipherSuites = [
  'TLS_AES_128_GCM_SHA256',
  'TLS_AES_256_GCM_SHA384',
  'TLS_CHACHA20_POLY1305_SHA256',
  'TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA',
  'TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA',
  'TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA',
  'TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA',
  'TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256',
  'TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384',
  'TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256',
  'TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384',
  'TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256',
  'TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256',
]
export const certUsages = ['encipherment', 'verify', 'issue']
export const sniffingTargets = ['http', 'tls', 'quic', 'fakedns']
export const tcpCongestions = ['bbr', 'cubic', 'reno']
export const vmessSecurities = ['aes-128-gcm', 'chacha20-poly1305', 'auto']
export const trafficResets = ['never', 'hourly', 'daily', 'weekly', 'monthly']
export const networks = [
  { value: 'tcp', label: 'TCP (RAW)' },
  { value: 'kcp', label: 'mKCP' },
  { value: 'ws', label: 'WebSocket' },
  { value: 'grpc', label: 'gRPC' },
  { value: 'httpupgrade', label: 'HTTPUpgrade' },
  { value: 'xhttp', label: 'XHTTP' },
]

/** Sites a new Reality inbound borrows its target and SNI from, as the old panel picked one. */
export const realityTargets = [
  { target: 'www.amazon.com:443', sni: 'www.amazon.com' },
  { target: 'aws.amazon.com:443', sni: 'aws.amazon.com' },
  { target: 'www.oracle.com:443', sni: 'www.oracle.com' },
  { target: 'www.nvidia.com:443', sni: 'www.nvidia.com' },
  { target: 'www.amd.com:443', sni: 'www.amd.com' },
  { target: 'www.intel.com:443', sni: 'www.intel.com' },
  { target: 'www.tesla.com:443', sni: 'www.tesla.com' },
  { target: 'www.sony.com:443', sni: 'www.sony.com' },
]

// --- Randomness --------------------------------------------------------------------

const digits = '0123456789'
const lower = 'abcdefghijklmnopqrstuvwxyz'
const upper = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'
const hex = '0123456789abcdef'

function pick(count: number, chars: string) {
  return Array.from(crypto.getRandomValues(new Uint32Array(count)), (v) => chars[v % chars.length]).join('')
}

const randomBase64 = (bytes: number) => btoa(String.fromCharCode(...crypto.getRandomValues(new Uint8Array(bytes))))

export const randomSeq = (count: number) => pick(count, digits + lower + upper)
export const randomLowerNum = (count: number) => pick(count, digits + lower)
export const randomHex = (count: number) => pick(count, hex)
export const randomInt = (min: number, max: number) => min + (crypto.getRandomValues(new Uint32Array(1))[0]! % (max - min + 1))

/** A port for a new inbound, as the old panel chose one. */
export const randomPort = () => randomInt(10000, 60000)

/** Short IDs of every even length from 2 to 16 hex digits, in random order. */
export function randomShortIds() {
  return [2, 4, 6, 8, 10, 12, 14, 16].sort(() => Math.random() - 0.5).map((n) => randomHex(n))
}

/** A Shadowsocks key: 16 random bytes for 2022-blake3-aes-128-gcm, 32 for the rest. */
export const ssPassword = (method = '2022-blake3-aes-256-gcm') => randomBase64(method === '2022-blake3-aes-128-gcm' ? 16 : 32)

/** An MTProto secret: 16 random bytes as lowercase hex, as telemt takes it. */
export const mtprotoSecret = () => randomHex(32)

export const randomRealityTarget = () => realityTargets[randomInt(0, realityTargets.length - 1)]!

/** Four numbers from 0 to 999 for VLESS Vision padding. */
export const randomVisionSeed = () => Array.from({ length: 4 }, () => randomInt(0, 999))
export const visionSeed = () => [900, 500, 900, 256]

// --- What goes with what -----------------------------------------------------------

/** The panel's own fields of an inbound beside Xray's JSON. */
export interface InboundForm {
  id?: number
  enable: boolean
  remark: string
  listen: string
  port: number | null
  protocol: Protocol
  /** Traffic limit in bytes, 0 without one. */
  total: number
  trafficReset: string
  expiryTime: number
  lastTrafficResetTime: number
  up: number
  down: number
  settings: Json
  /** Absent for the protocols without a transport, External Proxy or Sockopt. */
  streamSettings?: Json
  sniffing: Json
}

export const withClients = (p: Protocol, s: Json = {}) =>
  ['vmess', 'vless', 'trojan', 'hysteria', 'mtproto'].includes(p) || (p === 'shadowsocks' && isSSMultiUser(s.method))

export const isSSMultiUser = (method: string | undefined) => method !== '2022-blake3-chacha20-poly1305'
export const isSS2022 = (method: string | undefined) => String(method ?? '').startsWith('2022')

export const canStream = (p: Protocol) => ['vmess', 'vless', 'trojan', 'shadowsocks', 'hysteria'].includes(p)
export const network = (f: InboundForm): string => f.streamSettings?.network ?? 'tcp'
export const security = (f: InboundForm): string => f.streamSettings?.security ?? 'none'

export function canTls(f: InboundForm) {
  if (f.protocol === 'hysteria') return true
  return ['vmess', 'vless', 'trojan', 'shadowsocks'].includes(f.protocol) && ['tcp', 'ws', 'http', 'grpc', 'httpupgrade', 'xhttp'].includes(network(f))
}

export const canReality = (f: InboundForm) => ['vless', 'trojan'].includes(f.protocol) && ['tcp', 'http', 'grpc', 'xhttp'].includes(network(f))

/** xtls-rprx-vision: VLESS over raw TCP with TLS or Reality. */
export const canFlow = (f: InboundForm) => f.protocol === 'vless' && ['tls', 'reality'].includes(security(f)) && network(f) === 'tcp'

/** Vision padding applies once a client uses a Vision flow. */
export const canVisionSeed = (f: InboundForm) => canFlow(f) && clientsOf(f).some((c) => ['xtls-rprx-vision', 'xtls-rprx-vision-udp443'].includes(c.flow))

/** UDP masks belong to the transports over UDP. */
export const canUdpMask = (f: InboundForm) => ['kcp', 'hysteria'].includes(network(f))

export const clientsOf = (f: InboundForm): Json[] => (Array.isArray(f.settings.clients) ? f.settings.clients : [])

// --- Defaults ----------------------------------------------------------------------

/** A new client of the protocol, with the fresh key it is told apart by. */
export function newClient(protocol: Protocol, method = ''): Json {
  const base = { email: randomLowerNum(8), limitIp: 0, totalGB: 0, expiryTime: 0, enable: true, subId: randomLowerNum(16), comment: '', reset: 0 }
  switch (protocol) {
    case 'vmess':
      return { id: uuid(), security: 'auto', ...base }
    case 'vless':
      return { id: uuid(), flow: '', ...base }
    case 'trojan':
      return { password: randomSeq(10), ...base }
    case 'shadowsocks':
      return { method: isSS2022(method) ? '' : method, password: ssPassword(method), ...base }
    case 'hysteria':
      return { auth: randomSeq(10), ...base }
    case 'mtproto':
      return { id: mtprotoSecret(), ...base }
    default:
      return base
  }
}

/** A WireGuard peer with its own keys and the next address of 10.0.0.0/24. */
export function newPeer(index: number): Json {
  const keys = wireguardKeys()
  return { privateKey: keys.privateKey, publicKey: keys.publicKey, preSharedKey: '', allowedIPs: [`10.0.0.${index + 2}/32`], keepAlive: 0 }
}

/** An address as Xray's allowedIPs take it, in CIDR: one host, /32 for IPv4 and /128 for IPv6, when no mask is given. */
export const withMask = (a: string) => (a && !a.includes('/') ? `${a}/${a.includes(':') ? 128 : 32}` : a)

export const newAccount = () => ({ user: randomSeq(10), pass: randomSeq(10) })
export const newFallback = () => ({ name: '', alpn: '', path: '', dest: '', xver: 0 })

export function defaultSettings(protocol: Protocol): Json {
  switch (protocol) {
    case 'vmess':
      return { clients: [newClient('vmess')] }
    case 'vless':
      return { clients: [newClient('vless')], decryption: 'none', encryption: 'none', fallbacks: [], testseed: visionSeed() }
    case 'trojan':
      return { clients: [newClient('trojan')], fallbacks: [] }
    case 'shadowsocks': {
      const method = '2022-blake3-aes-256-gcm'
      return { method, password: ssPassword(method), network: 'tcp,udp', clients: [newClient('shadowsocks', method)] }
    }
    case 'hysteria':
      return { version: 2, clients: [newClient('hysteria')] }
    case 'mtproto':
      return { tlsDomain: '', clients: [newClient('mtproto')], web: { enable: false, host: '', path: '', publicIp: '' } }
    case 'tunnel':
      return { address: '', portMap: {}, network: 'tcp,udp', followRedirect: false }
    case 'mixed':
      return { auth: 'password', accounts: [newAccount()], udp: false, ip: '127.0.0.1' }
    case 'http':
      return { accounts: [newAccount()], allowTransparent: false }
    case 'wireguard':
      return { mtu: 1420, secretKey: wireguardKeys().privateKey, peers: [newPeer(0)], noKernelTun: false }
    case 'tun':
      return { name: 'xray0', mtu: 1500, gateway: [], dns: [], userLevel: 0, autoSystemRoutingTable: [], autoOutboundsInterface: 'auto' }
    default:
      return {}
  }
}

/** A Shadowsocks method changed: a fresh key for it, and clients that fit it, as the old panel did. */
export function setSSMethod(s: Json, method: string) {
  s.method = method
  s.password = ssPassword(method)
  if (!isSSMultiUser(method)) {
    s.clients = []
    return
  }
  if (!Array.isArray(s.clients) || !s.clients.length) s.clients = [newClient('shadowsocks', method)]
  for (const c of s.clients) {
    c.method = isSS2022(method) ? '' : method
    c.password = ssPassword(method)
  }
}

export const defaultTcpRequest = () => ({ version: '1.1', method: 'GET', path: ['/'], headers: {} })
export const defaultTcpResponse = () => ({ version: '1.1', status: '200', reason: 'OK', headers: {} })

export function defaultNetwork(net: string): Json {
  switch (net) {
    case 'tcp':
      return { acceptProxyProtocol: false, header: { type: 'none' } }
    case 'kcp':
      return { mtu: 1350, tti: 20, uplinkCapacity: 5, downlinkCapacity: 20, cwndMultiplier: 1, maxSendingWindow: 2097152 }
    case 'ws':
      return { acceptProxyProtocol: false, path: '/', host: '', headers: {}, heartbeatPeriod: 0 }
    case 'grpc':
      return { serviceName: '', authority: '', multiMode: false }
    case 'httpupgrade':
      return { acceptProxyProtocol: false, path: '/', host: '', headers: {} }
    case 'xhttp':
      return {
        path: '/',
        host: '',
        headers: {},
        scMaxBufferedPosts: 30,
        scMaxEachPostBytes: '1000000',
        scStreamUpServerSecs: '20-80',
        noSSEHeader: false,
        xPaddingBytes: '100-1000',
        mode: 'auto',
        xPaddingObfsMode: false,
        xPaddingKey: '',
        xPaddingHeader: '',
        xPaddingPlacement: '',
        xPaddingMethod: '',
        uplinkHTTPMethod: '',
        sessionIDPlacement: '',
        sessionIDKey: '',
        seqPlacement: '',
        seqKey: '',
        uplinkDataPlacement: '',
        uplinkDataKey: '',
        uplinkChunkSize: 0,
        sessionIDTable: '',
        sessionIDLength: '',
        serverMaxHeaderBytes: 0,
        noGRPCHeader: false,
        scMinPostsIntervalMs: '',
      }
    case 'hysteria':
      return { version: 2, auth: '', udpIdleTimeout: 60 }
    default:
      return {}
  }
}

export const defaultMasquerade = () => ({ type: 'proxy', dir: '', url: '', rewriteHost: false, insecure: false, content: '', headers: {}, statusCode: 0 })

/**
 * A TLS certificate as the window edits it: useFile picks between the paths
 * and the content, which is kept as text; both are left as Xray takes them on
 * save (certFromForm).
 */
export const defaultCert = () => ({
  useFile: true,
  certificateFile: '',
  keyFile: '',
  certificate: '',
  key: '',
  oneTimeLoading: false,
  usage: 'encipherment',
  buildChain: false,
})

/** A stored certificate for the window: paths when it has them, else its content as text. */
function certToForm(c: Json): Json {
  const lines = (v: unknown) => (Array.isArray(v) ? v.join('\n') : String(v ?? ''))
  const useFile = 'useFile' in c ? !!c.useFile : 'certificateFile' in c || !('certificate' in c)
  return { ...defaultCert(), ...c, useFile, certificate: lines(c.certificate), key: lines(c.key) }
}

/** A certificate as Xray takes it: the paths, or the content as lines. */
function certFromForm(c: Json): Json {
  const common = { oneTimeLoading: c.oneTimeLoading, usage: c.usage, buildChain: c.buildChain }
  if (c.useFile) return { certificateFile: c.certificateFile, keyFile: c.keyFile, ...common }
  return { certificate: String(c.certificate ?? '').split('\n'), key: String(c.key ?? '').split('\n'), ...common }
}

/** TLS settings for the window: the defaults under what is set, certificates to edit. */
function tlsForm(t: Json = {}): Json {
  const certs: Json[] = Array.isArray(t.certificates) ? t.certificates : []
  return { ...defaultTls(), ...t, settings: { ...defaultTls().settings, ...t.settings }, certificates: certs.length ? certs.map(certToForm) : [defaultCert()] }
}

/** Reality settings for the window: the defaults under what is set. */
function realityForm(r: Json = {}): Json {
  return { ...defaultReality(), ...r, settings: { ...defaultReality().settings, ...r.settings } }
}

/** A list typed as text, as Xray takes it: trimmed, without empty entries. */
const listOf = (v: unknown, sep: RegExp) => (Array.isArray(v) ? v : String(v ?? '').split(sep)).map((x) => String(x).trim()).filter(Boolean)

export const defaultTls = () => ({
  serverName: '',
  minVersion: '1.2',
  maxVersion: '1.3',
  cipherSuites: '',
  rejectUnknownSni: false,
  disableSystemRoot: false,
  enableSessionResumption: false,
  certificates: [defaultCert()],
  alpn: ['h2', 'http/1.1'],
  echServerKeys: '',
  settings: { fingerprint: 'chrome', echConfigList: '' },
})

export function defaultReality() {
  const { target, sni } = randomRealityTarget()
  return {
    show: false,
    xver: 0,
    target,
    serverNames: [sni],
    privateKey: '',
    minClientVer: '',
    maxClientVer: '',
    maxTimediff: 0,
    shortIds: randomShortIds(),
    mldsa65Seed: '',
    settings: { publicKey: '', fingerprint: 'chrome', serverName: '', spiderX: '/', mldsa65Verify: '' },
  }
}

export const defaultSockopt = () => ({
  acceptProxyProtocol: false,
  tcpFastOpen: false,
  mark: 0,
  tproxy: 'off',
  tcpMptcp: false,
  penetrate: false,
  domainStrategy: 'UseIP',
  tcpMaxSeg: 1440,
  dialerProxy: '',
  tcpKeepAliveInterval: 0,
  tcpKeepAliveIdle: 300,
  tcpUserTimeout: 10000,
  tcpcongestion: 'bbr',
  V6Only: false,
  tcpWindowClamp: 600,
  interface: '',
})

export function defaultTcpMask(type = 'fragment'): Json {
  switch (type) {
    case 'fragment':
      return { type, settings: { packets: 'tlshello', length: '100-200', delay: '', maxSplit: '' } }
    case 'sudoku':
      return { type, settings: { password: '', ascii: '', customTable: '', customTables: [], paddingMin: 0, paddingMax: 0 } }
    case 'header-custom':
      return { type, settings: { clients: [], servers: [] } }
    default:
      return { type, settings: {} }
  }
}

export function defaultUdpMask(type = 'salamander'): Json {
  switch (type) {
    case 'salamander':
      return { type, settings: { password: '' } }
    case 'mkcp-legacy':
      return { type, settings: { header: '', value: '' } }
    case 'xdns':
      return { type, settings: { domains: [{ names: [] }] } }
    case 'header-custom':
      return { type, settings: { client: [], server: [] } }
    case 'noise':
      return { type, settings: { reset: 0, noise: [] } }
    case 'xicmp':
      return { type, settings: { dgram: false, ips: [] } }
    default:
      return { type, settings: {} }
  }
}

/** External Proxy turned on: the panel's own address and the inbound's port, as the old panel began it. */
export const defaultExternalProxy = (port: number | null) => ({ forceTls: 'same', dest: location.hostname, port: port ?? 443, remark: '' })

export const defaultSniffing = () => ({ enabled: false, destOverride: [...sniffingTargets], metadataOnly: false, routeOnly: false })

/** The transport a protocol starts on: raw TCP without security; Hysteria on its own over TLS with h3. */
export function defaultStream(protocol: Protocol): Json | undefined {
  if (protocol === 'hysteria') {
    return { network: 'hysteria', security: 'tls', externalProxy: [], hysteriaSettings: defaultNetwork('hysteria'), tlsSettings: { ...defaultTls(), alpn: ['h3'] } }
  }
  if (canStream(protocol)) return { network: 'tcp', security: 'none', externalProxy: [], tcpSettings: defaultNetwork('tcp') }
  if (protocol === 'mtproto') return { externalProxy: [] }
  return undefined
}

/** A new inbound: VLESS on a random port with one client, as the old panel began. */
export function newInbound(protocol: Protocol = 'vless'): InboundForm {
  return {
    enable: true,
    remark: '',
    listen: '',
    port: randomPort(),
    protocol,
    total: 0,
    trafficReset: 'never',
    expiryTime: 0,
    lastTrafficResetTime: 0,
    up: 0,
    down: 0,
    settings: defaultSettings(protocol),
    streamSettings: defaultStream(protocol),
    sniffing: defaultSniffing(),
  }
}

/** Another protocol chosen: its own settings and transport from scratch, as the old panel did. */
export function setProtocol(f: InboundForm, protocol: Protocol) {
  f.protocol = protocol
  f.settings = defaultSettings(protocol)
  f.streamSettings = defaultStream(protocol)
}

/** Another transport chosen: its settings, and what no longer fits it let go. */
export function setNetwork(f: InboundForm, net: string) {
  const st = (f.streamSettings ??= {})
  st.network = net
  st[`${net}Settings`] = { ...defaultNetwork(net), ...st[`${net}Settings`] }
  fitSecurity(f)
  if (!canUdpMask(f) && st.finalmask) st.finalmask.udp = []
}

/** Another security chosen: its settings, filled in with defaults. */
export function setSecurity(f: InboundForm, sec: string) {
  const st = (f.streamSettings ??= {})
  st.security = sec
  if (sec === 'tls') st.tlsSettings = tlsForm(st.tlsSettings)
  if (sec === 'reality') st.realitySettings = realityForm(st.realitySettings)
  fitSecurity(f)
}

/** Security the transport cannot take falls back to none, and a Vision flow without TLS or Reality goes. */
function fitSecurity(f: InboundForm) {
  const st = f.streamSettings
  if (!st) return
  if ((st.security === 'tls' && !canTls(f)) || (st.security === 'reality' && !canReality(f))) st.security = 'none'
  if (f.protocol === 'vless' && !canFlow(f)) for (const c of clientsOf(f)) c.flow = ''
}

// --- Reading -----------------------------------------------------------------------

const parse = (text: string): Json => {
  if (!text) return {}
  try {
    return JSON.parse(text) as Json
  } catch {
    return {}
  }
}

/**
 * A stored inbound for the window, with the defaults the old panel filled in
 * on reading it: of the transport and security in use, Sniffing's targets and
 * the Vision seed.
 */
export function fromDb(ib: DbInbound): InboundForm {
  const f: InboundForm = {
    id: ib.id,
    enable: ib.enable,
    remark: ib.remark,
    listen: ib.listen,
    port: ib.port,
    protocol: ib.protocol,
    total: ib.total,
    trafficReset: ib.trafficReset || 'never',
    expiryTime: ib.expiryTime,
    lastTrafficResetTime: ib.lastTrafficResetTime,
    up: ib.up,
    down: ib.down,
    settings: parse(ib.settings),
    streamSettings: undefined,
    sniffing: { ...defaultSniffing(), ...parse(ib.sniffing) },
  }
  if (!Array.isArray(f.sniffing.destOverride) || !f.sniffing.destOverride.length) f.sniffing.destOverride = [...sniffingTargets]

  const st = parse(ib.streamSettings)
  if (canStream(f.protocol)) {
    st.network ??= f.protocol === 'hysteria' ? 'hysteria' : 'tcp'
    st.security ??= 'none'
    st[`${st.network}Settings`] = { ...defaultNetwork(st.network), ...st[`${st.network}Settings`] }
    if (st.security === 'tls') st.tlsSettings = tlsForm(st.tlsSettings)
    if (st.security === 'reality') st.realitySettings = realityForm(st.realitySettings)
  }
  if (canStream(f.protocol) || f.protocol === 'mtproto' || Object.keys(st).length) {
    st.externalProxy = Array.isArray(st.externalProxy) ? st.externalProxy : []
    f.streamSettings = st
  }
  if (f.protocol === 'vless' && !(Array.isArray(f.settings.testseed) && f.settings.testseed.length >= 4)) f.settings.testseed = visionSeed()
  if (f.protocol === 'mtproto') f.settings.web = { enable: false, host: '', path: '', publicIp: '', ...f.settings.web }
  return f
}

// --- Saving ------------------------------------------------------------------------

const empty = (v: unknown) =>
  v === undefined || v === null || v === '' || (Array.isArray(v) && !v.length) || (typeof v === 'object' && !Array.isArray(v) && !Object.keys(v as object).length)

const netKeys = ['tcpSettings', 'kcpSettings', 'wsSettings', 'grpcSettings', 'httpupgradeSettings', 'xhttpSettings', 'hysteriaSettings']

/** Headers as Xray takes them: no entry without a name or a value. */
function cleanHeaders(h: Json | undefined) {
  if (!h || typeof h !== 'object') return h
  for (const [k, v] of Object.entries(h)) {
    const values = Array.isArray(v) ? v.filter((x) => !empty(x)) : v
    if (empty(k) || empty(values)) delete h[k]
    else h[k] = values
  }
  return h
}

function finalStream(f: InboundForm, input: Json): Json | undefined {
  const st: Json = JSON.parse(JSON.stringify(input))
  if (!canStream(f.protocol)) {
    // Without a transport only External Proxy (MTProto) and Sockopt stay.
    const out: Json = {}
    if (f.protocol === 'mtproto') out.externalProxy = st.externalProxy ?? []
    if (st.sockopt) out.sockopt = st.sockopt
    return empty(out) ? undefined : out
  }
  const keep = `${st.network}Settings`
  for (const k of netKeys) if (k !== keep) delete st[k]
  if (st.security !== 'tls') delete st.tlsSettings
  if (st.security !== 'reality') delete st.realitySettings
  if (st.tlsSettings) st.tlsSettings.certificates = (st.tlsSettings.certificates ?? []).map(certFromForm)
  if (st.realitySettings) {
    const r = st.realitySettings
    r.serverNames = listOf(r.serverNames, /,/)
    r.shortIds = listOf(r.shortIds, /[\s,]+/)
  }

  const net = st[keep] ?? {}
  if (st.network === 'tcp' && net.header?.type !== 'http') net.header = { type: net.header?.type ?? 'none' }
  for (const h of [net.headers, net.header?.request?.headers, net.header?.response?.headers, net.masquerade?.headers]) cleanHeaders(h)
  if (st.network === 'hysteria' && empty(net.masquerade)) delete net.masquerade
  if (st.network === 'xhttp') {
    // Xray takes no empty range: an unset one is left out.
    for (const k of ['h2Params', 'sessionIDTable', 'sessionIDLength', 'scMinPostsIntervalMs']) if (empty(net[k])) delete net[k]
  }
  // The TLS a link sets up for an External Proxy row goes with Force TLS only.
  for (const ep of Array.isArray(st.externalProxy) ? st.externalProxy : []) {
    for (const k of ['sni', 'fingerprint', 'alpn']) if (ep.forceTls !== 'tls' || empty(ep[k])) delete ep[k]
  }

  const fm = st.finalmask
  if (fm) {
    if (!canUdpMask(f)) delete fm.udp
    if (f.protocol !== 'hysteria' && st.network !== 'xhttp') delete fm.quicParams
    if (empty(fm.tcp)) delete fm.tcp
    if (empty(fm.udp)) delete fm.udp
    if (empty(fm)) delete st.finalmask
  }
  if (st.sockopt && empty(st.sockopt.trustedXForwardedFor)) delete st.sockopt.trustedXForwardedFor
  return st
}

/**
 * A client as the panel stores it: no Vision flow where the inbound cannot
 * take one, no empty reverse tag, no MTProto limits left at 0. Used for the
 * inbound's own clients and by the client window (addClient, updateClient).
 */
export function finalClient(f: InboundForm, client: Json): Json {
  const c: Json = JSON.parse(JSON.stringify(client))
  if (f.protocol === 'vless') {
    if (!canFlow(f)) c.flow = ''
    if (empty(c.reverse?.tag)) delete c.reverse
  }
  if (f.protocol === 'mtproto') for (const k of ['rateUp', 'rateDown', 'maxConns']) if (!c[k]) delete c[k]
  return c
}

function finalSettings(f: InboundForm): Json {
  const s: Json = JSON.parse(JSON.stringify(f.settings))
  if (Array.isArray(s.clients)) s.clients = s.clients.map((c: Json) => finalClient(f, c))
  switch (f.protocol) {
    case 'vless': {
      const clients = s.clients ?? []
      if (!clients.some((c: Json) => !empty(c.flow))) delete s.testseed
      if (empty(s.fallbacks)) delete s.fallbacks
      if (empty(s.selectedAuth)) delete s.selectedAuth
      break
    }
    case 'trojan':
      s.fallbacks ??= []
      break
    case 'mtproto':
      if (!s.web?.enable && empty(s.web?.host) && empty(s.web?.path)) delete s.web
      break
    case 'mixed':
      if (s.auth !== 'password') delete s.accounts
      break
    case 'wireguard':
      for (const p of s.peers ?? []) {
        if (empty(p.preSharedKey)) delete p.preSharedKey
        p.allowedIPs = (p.allowedIPs ?? []).map(withMask)
      }
      break
    case 'tunnel':
      cleanHeaders(s.portMap)
      break
  }
  return s
}

/**
 * The inbound as the panel stores it (POST panel/api/inbounds/add or
 * update/:id): its fields, and settings, streamSettings and sniffing as JSON
 * text, with the settings of the transport and the security in use only.
 */
export function toForm(f: InboundForm): FormData {
  const st = f.streamSettings ? finalStream(f, f.streamSettings) : undefined
  const sniffing: Json = JSON.parse(JSON.stringify(f.sniffing))
  for (const k of ['ipsExcluded', 'domainsExcluded']) if (empty(sniffing[k])) delete sniffing[k]

  const fields: Record<string, unknown> = {
    up: f.up,
    down: f.down,
    total: f.total,
    remark: f.remark,
    enable: f.enable,
    expiryTime: f.expiryTime,
    trafficReset: f.trafficReset,
    lastTrafficResetTime: f.lastTrafficResetTime,
    listen: f.listen,
    port: f.port ?? 0,
    protocol: f.protocol,
    settings: JSON.stringify(finalSettings(f), null, 2),
    sniffing: JSON.stringify(sniffing, null, 2),
  }
  if (st) fields.streamSettings = JSON.stringify(st, null, 2)
  const form = new FormData()
  for (const [k, v] of Object.entries(fields)) form.set(k, String(v))
  return form
}

/**
 * A copy of a stored inbound for clone: the same transport, security and
 * Sniffing, new settings without the clients, another port, no listen address
 * and no traffic, as the old panel cloned.
 */
export function cloneOf(ib: DbInbound): InboundForm {
  const f = fromDb(ib)
  return { ...f, id: undefined, remark: `${ib.remark} - Cloned`, listen: '', port: randomPort(), up: 0, down: 0, settings: defaultSettings(ib.protocol) }
}

// --- Keys from the server ----------------------------------------------------------

/** A Reality key pair from Xray on the server. */
export const newX25519 = () => api.get<{ privateKey: string; publicKey: string }>('panel/api/server/getNewX25519Cert')

/** An ML-DSA-65 seed and its verify key. */
export const newMldsa65 = () => api.get<{ seed: string; verify: string }>('panel/api/server/getNewmldsa65')

/** ECH server keys and the config list for the SNI. */
export function newEch(sni: string) {
  const form = new FormData()
  form.set('sni', sni)
  return api.post<{ echServerKeys: string; echConfigList: string }>('panel/api/server/getNewEchCert', form)
}

/** VLESS Encryption pairs, one per authentication, labelled as the server names them. */
export const newVlessEnc = () => api.get<{ auths?: { label: string; decryption: string; encryption: string }[] }>('panel/api/server/getNewVlessEnc')

/** What TUN's autoSystemWfpBlockLeak takes: dns, windnsclient, misconfigtun or a port as tcp/853 or udp/53. */
export function wfpLeakWrong(values: unknown): boolean {
  if (!Array.isArray(values)) return false
  return values.some((v) => {
    const s = String(v).trim().toLowerCase()
    if (['dns', 'windnsclient', 'misconfigtun'].includes(s)) return false
    const m = /^(tcp|udp)\/(\d+)$/.exec(s)
    return !m || Number(m[2]) < 1 || Number(m[2]) > 65535
  })
}

const filledList = (v: unknown) => Array.isArray(v) && v.some((x) => String(x).trim())

/**
 * What a TUN's settings need beside each other, as Xray checks them where they
 * take effect: DNS to the gateway needs a gateway (Linux), blocking leaks
 * needs the routing table (Windows), blocking the system's DNS needs DNS
 * servers on the TUN (Windows). Each names the field that is missing.
 */
export function tunMissing(s: Record<string, any> | undefined) {
  const leaks: string[] = Array.isArray(s?.autoSystemWfpBlockLeak) ? s.autoSystemWfpBlockLeak.map((v: unknown) => String(v).trim().toLowerCase()) : []
  return {
    gateway: !!s?.autoSystemDnsToGateway && !filledList(s?.gateway),
    routingTable: leaks.length > 0 && !filledList(s?.autoSystemRoutingTable),
    dns: leaks.some((v) => ['dns', 'udp/53', 'tcp/53', 'windnsclient'].includes(v)) && !filledList(s?.dns),
  }
}

/** A TUN Xray does not start with. */
export const tunWrong = (s: Record<string, any> | undefined) => wfpLeakWrong(s?.autoSystemWfpBlockLeak) || Object.values(tunMissing(s)).some(Boolean)
