/**
 * The panel's mark: a dotted Earth turning on the kit's accent green (--acc of
 * the light theme, which reads in both themes). One geometry serves the static
 * mark in the menu, the favicon (built into the page, see vite.config.ts) and
 * the turning mark on the login page. Nothing here touches the DOM, so the
 * build can use it too.
 */

type Vec = [number, number, number]

/** Colours of the mark; it keeps them in both themes, the favicon has no theme. */
export const globeColors = {
  tile: '#0A7A59',
  land: '#ffffff',
  sea: 'rgba(255,255,255,.16)',
}

export interface GlobeDot { x: number, y: number, r: number, color: string }
export interface GlobeFrame { size: number, radius: number, dots: GlobeDot[] }

/**
 * Land of Natural Earth 1:110m (public domain) on a 2° grid, 180 × 90 cells
 * from 180°W and 90°N, one bit a cell, low bit first.
 */
const MASK = 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOAfAP8DAAAAAAAAAAAAAAAAAAAAAADo//z//wcAAAAAAAACAAAAAAAAAAAAhvvw//8PAPABAAAAwAMAAAAAAAAAwADkw////wEABAAAAABgAAAAAAAAAADAUT8A/v8PAAAAAAwA/j8AuAEAAAAAcBHtDcD/fwAAAAAwAPz/fwMAABCAAQD5w/wD8P8FAAAMAIL7//9//xOAAP/ff0668QD/HwAA+A8At///////v3/w/////x8++B8AAOD/1//7////////jP////+/+AE/gAcAn9f//////////w/w/////wEs4AEAAHz+////////////gL////8HeAAcAADg5///////////9ADgAf7/f4AnAAAAAH78////////H0QAAAiA//8f8AcAAIBB4////////38ADwAQAOD//5//AQAAHAT/////////A3AAAAAA/v//+T8AAGDz//////////8DAQAAAMD/////AwAAsP//////////LwAAAAAA6P///2IAAAD+//////////8CAAAAAAD///8/CAAA4P//////////JwAAAAAA8P///wYAAAD+/unz/////z8AAAAAAAD///8HAAAA/pgPPP//////MQAAAAAA8P//PwAAAMBD9v7n/////wcBAAAAAAD///8AAAAAPkD7f/7///8hEAAAAAAA4P//DwAAAIDhAv/n////f8YAAAAAAAD8//8AAAAA+AdE//////8jDwAAAAAAgP//AwAAAMD/APD/////PxgAAAAAAADw/x8AAAAA/n/v//////8HAAAAAAAAAPwDAgAAAOD////7////fwAAAAAAAACgHyAAAACA//9/f/7///8DAAAAAAAAAPQBAAAAAPj//+cv+P//PwAAAAAAAAAAHjAAAADA/////g/+//8EAAAAAAAAAOBhCAAAAP7//99/4D//AAAAAAAAAAAAPAMEAADA////+Qf84BcAAAAAAAAAAAA/AAAAAPz//58fgAf+QAAAAAAAAAAAAA8AAADg////ewA4gA8EAAAAAAAAAADAAAAAAPz//38BgAP4QQAAAAAAAAAAAAgPAADA////zwAwgAwQAAAAAAAAAAAA9Q8AAPj///8HAAVIAAAAAAAAAAAAAID/AQAA////fwBAAAAQAAAAAAAAAAAA+P8AAGDh//8DAAA0GAAAAAAAAAAAAID/HwAAAPj/HwAAgMIBAAAAAAAAAAAA/P8BAACA//8AAAAYXgAAAAAAAAAAAMD/fwAAAPz/BwAAAOOBAQAAAAAAAAAA/P8/AACA/z8AAABgbtQBAAAAAAAAAOD//w8AAPD/AwAAAAQIeAAAAAAAAAAA/P//AQAA/z8AAACAA4APAQAAAAAAAID//w8AAPD/AwAAAAARsEAAAAAAAAAA+P9/AAAA/j8AAAAAAAAAAAAAAAAAAAD//wcAAPD/QwAAAACAIwAAAAAAAAAA8P9/AAAA/z8EAAAAAD8GIAAAAAAAAAD8/wMAAPD/cQAAAAD4ZwAAAAAAAAAAgP8/AAAA/w8HAAAAgP8HAAAAAAAAAAD4/wMAAOD/MAAAAAD//wEBAAAAAAAAgP8PAAAA/g8DAAAA+P8fAAAAAAAAAAD4PwAAAOB/EAAAAID//wMAAAAAAAAAgP8DAAAA/AMAAAAA+P9/AAAAAAAAAAD8HwAAAMA/AAAAAID//wcAAAAAAAAAwP8BAAAA+AEAAAAA8P9/AAAAAAAAAAD8DwAAAIAPAAAAAAAP/gMAAAAAAAAAwB8AAAAAAAAAAAAAEIAfAAEAAAAAAAD+AwAAAAAAAAAAAAAA8AEgAAAAAAAA4AcAAAAAAAAAAAAAAAAAAAYAAAAAAABeAAAAAAAAAAAAAAAAwAAwAAAAAAAAwAMAAAAAAAAAAAAAAAAIgAEAAAAAAAAeAAAAAAAAAAAAAAAAAAAMAAAAAAAA4AEAAAAAAAAAAAAAAAAAAAAAAAAAAAAPAAAAAAAAAAABAAAAAAAAAAAAAAAAcAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOAAAAAAAAAAAAAAAAAAAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMAAAAAAAB4AEDwn/8HAAAAAAAAAAAwAAAAAACA/P/h/////w8AAAAAAAAAwAcAAAD4////z///////PwAAAAAAHALwAACA//////////////8HAADw/y///wMAAP7/////////////HwAA+P///38AAID///////////////8AAPL/////BwAO////////////////DwAA8P////8HEPD//////////////z8AAOD/////////////////////////H/AfwP//////////////////////////////////////////////////////////////////////////////////////'
const bits = Uint8Array.from(atob(MASK), (c) => c.charCodeAt(0))

function land(lat: number, lon: number): boolean {
  const j = Math.min(89, Math.floor((90 - lat) / 2))
  const i = Math.min(179, Math.floor((((lon + 180) % 360) + 360) % 360 / 2))
  const k = j * 180 + i
  return ((bits[k >> 3]! >> (k & 7)) & 1) === 1
}

/** Dots in rows of latitude, `step` degrees apart, each with its place and kind. */
const grids = new Map<number, { p: Vec, land: boolean }[]>()
function grid(step: number) {
  let g = grids.get(step)
  if (g) return g
  g = []
  for (let lat = -90 + step / 2; lat < 90; lat += step) {
    const a = (lat * Math.PI) / 180
    const n = Math.max(1, Math.round((360 / step) * Math.cos(a)))
    for (let m = 0; m < n; m++) {
      const lon = -180 + (360 * (m + 0.5)) / n
      const o = (lon * Math.PI) / 180
      g.push({ p: [Math.cos(a) * Math.sin(o), Math.sin(a), Math.cos(a) * Math.cos(o)], land: land(lat, lon) })
    }
  }
  grids.set(step, g)
  return g
}

/** Turns the globe by `angle` round its axis, its north tipped towards the viewer. */
function turn(p: Vec, angle: number): Vec {
  const c = Math.cos(angle)
  const s = Math.sin(angle)
  const x = p[0] * c + p[2] * s
  const z = -p[0] * s + p[2] * c
  const ct = Math.cos(0.4)
  const st = Math.sin(0.4)
  return [x, p[1] * ct - z * st, p[1] * st + z * ct]
}

/** The resting pose (the static mark and the favicon): Europe and Africa face the viewer. */
export const restAngle = -0.3

/** The mark at `size` px, turned by `angle`. */
export function globeFrame(size: number, angle = restAngle): GlobeFrame {
  const k = globeColors
  const c = size / 2
  const R = size * 0.4
  const step = Math.min(7.5, Math.max(2.5, (1.25 / R) * (180 / Math.PI)))
  const gap = ((step * Math.PI) / 180) * R
  const dots: GlobeDot[] = []
  for (const { p, land: isLand } of grid(step)) {
    const q = turn(p, angle)
    if (q[2] < 0) continue
    const depth = 0.55 + 0.45 * q[2]
    dots.push({ x: c + q[0] * R, y: c - q[1] * R, r: gap * (isLand ? 0.36 : 0.22) * depth, color: isLand ? k.land : k.sea })
  }
  return { size, radius: R, dots }
}

const n = (v: number) => String(Math.round(v * 100) / 100)

/** The resting mark as an SVG document, 32 px across. */
export function globeSvg(): string {
  const f = globeFrame(32)
  const parts = [
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">`,
    `<rect width="32" height="32" rx="9" fill="${globeColors.tile}"/>`,
    ...f.dots.map((d) => `<circle cx="${n(d.x)}" cy="${n(d.y)}" r="${n(d.r)}" fill="${d.color}"/>`),
    '</svg>',
  ]
  return parts.join('')
}

/** The resting mark as a data: URL, for the favicon built into the page. */
export function globeIconUrl(): string {
  return `data:image/svg+xml,${encodeURIComponent(globeSvg())}`
}
