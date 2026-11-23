// Settings the panel puts into index.html (see web/spa). The Vite dev server
// serves index.html as is, so development falls back to the defaults.

export interface AppConfig {
  /** Panel base path, e.g. "/" or "/secret/"; the server endpoints live under it. */
  base: string
  /** Where this frontend is served: the panel base path, e.g. "/secret/". */
  mount: string
  loggedIn: boolean
  version: string
  /** The panel's time zone by its IANA name, which its windows show dates in; none on the subscription page. */
  zone?: string
}

function read(): AppConfig {
  const fallback: AppConfig = { base: '/', mount: '/', loggedIn: true, version: 'dev' }
  const raw = document.querySelector<HTMLMetaElement>('meta[name="app-config"]')?.content
  if (!raw) return fallback
  try {
    return { ...fallback, ...(JSON.parse(raw) as Partial<AppConfig>) }
  } catch {
    return fallback
  }
}

export const config = read()

/** URL of a server endpoint, e.g. serverUrl('panel/api/server/status'). */
export function serverUrl(path: string) {
  return config.base + path.replace(/^\//, '')
}
