package docsite

import (
	"fmt"
	"strings"
	"text/template"
)

// theme is how a site looks: its colours, type and the shape of its parts.
type theme struct {
	Hue, Sat     int
	TintHue      int
	TintSat      int
	Radius       int
	Body, Head   string
	Mono         string
	HeadWeight   int
	H1           string
	Header       string // plain, tint or dark
	Hero         string // center or split
	Columns      int
	Side         string // plain, bar or pill
	Width        int
	Underline    bool
	DarkCode     bool
	Wrap         int
	Illustration string
	Mark         string
	TitleSep     string
	TOC          bool
}

var fontSets = []struct{ body, head string }{
	{`system-ui, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif`, ""},
	{`-apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans", Helvetica, Arial, sans-serif`, ""},
	{`"Inter", system-ui, "Segoe UI", Roboto, sans-serif`, ""},
	{`system-ui, -apple-system, "Segoe UI", Roboto, sans-serif`, `Georgia, "Iowan Old Style", "Times New Roman", serif`},
	{`"Source Sans 3", "Source Sans Pro", system-ui, "Segoe UI", sans-serif`, ""},
	{`"Segoe UI", system-ui, -apple-system, Roboto, "Helvetica Neue", Arial, sans-serif`, `system-ui, -apple-system, "Segoe UI", sans-serif`},
	{`Charter, "Bitstream Charter", "Sitka Text", Cambria, serif`, `system-ui, -apple-system, "Segoe UI", sans-serif`},
}

var monoSets = []string{
	`ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace`,
	`"JetBrains Mono", ui-monospace, "Cascadia Code", Menlo, Consolas, monospace`,
	`"SF Mono", "Fira Code", "Roboto Mono", Menlo, Consolas, monospace`,
	`Menlo, Monaco, Consolas, "Courier New", monospace`,
}

// hues are accent colours that read well on white and on near black.
var hues = []int{4, 12, 24, 32, 142, 152, 162, 172, 184, 194, 205, 214, 222, 232, 248, 262, 332, 346}

func newTheme(r *rnd) *theme {
	t := &theme{}
	t.Hue = pick(r, hues) + r.between(-4, 4)
	t.Sat = r.between(55, 78)
	t.TintHue = t.Hue
	if r.chance(0.4) {
		t.TintHue = pick(r, []int{30, 40, 210, 220})
	}
	t.TintSat = r.between(4, 14)
	t.Radius = pick(r, []int{4, 6, 8, 10, 12})
	fs := pick(r, fontSets)
	t.Body, t.Head = fs.body, fs.head
	if t.Head == "" {
		t.Head = t.Body
	}
	t.Mono = pick(r, monoSets)
	t.HeadWeight = pick(r, []int{600, 650, 700})
	t.H1 = pick(r, []string{"2rem", "2.125rem", "2.25rem", "2.375rem"})
	t.Header = pick(r, []string{"plain", "plain", "tint", "dark"})
	t.Hero = pick(r, []string{"center", "split", "split"})
	t.Columns = pick(r, []int{2, 3, 3})
	t.Side = pick(r, []string{"plain", "bar", "pill"})
	t.Width = r.between(46, 52)
	t.Underline = r.chance(0.4)
	t.DarkCode = r.chance(0.5)
	t.Wrap = pick(r, []int{1200, 1280, 1360, 1440})
	t.Illustration = pick(r, illustrations)
	t.Mark = pick(r, marks)
	t.TitleSep = pick(r, []string{" - ", " | ", " · ", " — "})
	t.TOC = r.chance(0.75)
	return t
}

// hsl writes a colour of the accent's hue.
func (t *theme) hsl(sat, light int) string {
	return fmt.Sprintf("hsl(%d %d%% %d%%)", (t.Hue+360)%360, sat, light)
}

func (t *theme) tint(light int) string {
	return fmt.Sprintf("hsl(%d %d%% %d%%)", (t.TintHue+360)%360, t.TintSat, light)
}

// Accent is the accent colour of the light scheme, for the manifest and
// the browser's toolbar.
func (t *theme) Accent() string { return t.hsl(t.Sat, 44) }

func (t *theme) css() string {
	var b strings.Builder
	if err := cssTemplate.Execute(&b, t); err != nil {
		panic(err)
	}
	return b.String()
}

var cssTemplate = template.Must(template.New("css").Parse(`:root {
  --bg: #fff;
  --bg2: {{.Tint 97}};
  --text: {{.Tint 13}};
  --muted: {{.Tint 40}};
  --border: {{.Tint 89}};
  --accent: {{.Acc 44}};
  --accent-hover: {{.Acc 36}};
  --accent-soft: {{.AccSoft 95}};
  --accent-ink: #fff;
  --code-bg: {{if .DarkCode}}{{.Tint 12}}{{else}}{{.Tint 96}}{{end}};
  --code-text: {{if .DarkCode}}{{.Tint 90}}{{else}}{{.Tint 18}}{{end}};
  --code-dim: {{if .DarkCode}}{{.Tint 58}}{{else}}{{.Tint 50}}{{end}};
  --code-key: {{if .DarkCode}}{{.Acc 74}}{{else}}{{.Acc 38}}{{end}};
  --inline-code: {{.Tint 94}};
  --header-bg: {{if eq .Header "tint"}}{{.AccSoft 96}}{{else if eq .Header "dark"}}{{.Tint 11}}{{else}}rgba(255, 255, 255, .92){{end}};
  --header-text: {{if eq .Header "dark"}}{{.Tint 92}}{{else}}{{.Tint 13}}{{end}};
  --header-muted: {{if eq .Header "dark"}}{{.Tint 70}}{{else}}{{.Tint 40}}{{end}};
  --header-field: {{if eq .Header "dark"}}{{.Tint 18}}{{else}}{{.Tint 96}}{{end}};
  --header-border: {{if eq .Header "dark"}}{{.Tint 22}}{{else}}{{.Tint 89}}{{end}};
  --radius: {{.Radius}}px;
  --font: {{.Body}};
  --font-head: {{.Head}};
  --font-mono: {{.Mono}};
  color-scheme: light dark;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: {{.Tint 8}};
    --bg2: {{.Tint 11}};
    --text: {{.Tint 90}};
    --muted: {{.Tint 66}};
    --border: {{.Tint 20}};
    --accent: {{.Acc 68}};
    --accent-hover: {{.Acc 76}};
    --accent-soft: {{.AccSoft 16}};
    --accent-ink: {{.Tint 8}};
    --code-bg: {{.Tint 12}};
    --code-text: {{.Tint 88}};
    --code-dim: {{.Tint 56}};
    --code-key: {{.Acc 74}};
    --inline-code: {{.Tint 16}};
    --header-bg: {{if eq .Header "tint"}}{{.AccSoft 11}}{{else if eq .Header "dark"}}{{.Tint 5}}{{else}}{{.TintA 8 92}}{{end}};
    --header-text: {{.Tint 90}};
    --header-muted: {{.Tint 66}};
    --header-field: {{.Tint 14}};
    --header-border: {{.Tint 20}};
  }
}
*, *::before, *::after { box-sizing: border-box; }
html { -webkit-text-size-adjust: 100%; scroll-padding-top: 80px; }
body { margin: 0; background: var(--bg); color: var(--text); font: 16px/1.65 var(--font); }
a { color: var(--accent); text-decoration: {{if .Underline}}underline{{else}}none{{end}}; text-underline-offset: 2px; }
a:hover { color: var(--accent-hover); text-decoration: underline; }
img, svg { max-width: 100%; }
h1, h2, h3 { font-family: var(--font-head); font-weight: {{.HeadWeight}}; line-height: 1.25; letter-spacing: -.01em; font-variant-numeric: lining-nums; }
code, kbd, pre { font-family: var(--font-mono); font-size: .875em; }
.wrap { max-width: {{.Wrap}}px; margin: 0 auto; padding: 0 24px; }

.top { position: sticky; top: 0; z-index: 10; background: var(--header-bg); color: var(--header-text); border-bottom: 1px solid var(--header-border); backdrop-filter: saturate(1.4) blur(8px); }
.top .wrap { display: flex; align-items: center; gap: 28px; height: 60px; }
.brand { display: flex; align-items: center; gap: 10px; color: inherit; font: {{.HeadWeight}} 1.125rem var(--font-head); text-decoration: none; }
.brand:hover { color: inherit; text-decoration: none; }
.brand svg { width: 28px; height: 28px; flex: none; }
.main-nav { display: flex; gap: 22px; }
.main-nav a { color: var(--header-muted); font-size: .9375rem; text-decoration: none; }
.main-nav a:hover, .main-nav a[aria-current] { color: var(--header-text); text-decoration: none; }
.search { position: relative; margin-left: auto; width: 240px; }
.search input { width: 100%; height: 36px; padding: 0 34px 0 12px; border: 1px solid var(--header-border); border-radius: var(--radius); background: var(--header-field); color: var(--header-text); font: inherit; font-size: .875rem; }
.search input::placeholder { color: var(--header-muted); }
.search input:focus { outline: 2px solid var(--accent); outline-offset: -1px; }
.search kbd { position: absolute; right: 8px; top: 8px; padding: 0 6px; border: 1px solid var(--header-border); border-radius: 4px; color: var(--header-muted); font-size: .75rem; line-height: 18px; }
.hits { position: absolute; right: 0; top: 42px; width: min(420px, 90vw); margin: 0; padding: 6px; list-style: none; background: var(--bg); color: var(--text); border: 1px solid var(--border); border-radius: var(--radius); box-shadow: 0 12px 32px rgba(0, 0, 0, .14); }
.hits a { display: block; padding: 8px 10px; border-radius: calc(var(--radius) - 2px); color: inherit; text-decoration: none; }
.hits a:hover, .hits a.sel { background: var(--accent-soft); }
.hits b { display: block; font-weight: 600; }
.hits span { display: block; color: var(--muted); font-size: .8125rem; line-height: 1.45; }
.hits .none { padding: 8px 10px; color: var(--muted); font-size: .875rem; }
.menu { display: none; margin-left: 4px; padding: 6px; background: none; border: 0; color: inherit; cursor: pointer; }
.menu svg { display: block; width: 22px; height: 22px; }

.btn { display: inline-flex; align-items: center; height: 44px; padding: 0 20px; border-radius: var(--radius); background: var(--accent); color: var(--accent-ink); font-weight: 600; text-decoration: none; }
.btn:hover { background: var(--accent-hover); color: var(--accent-ink); text-decoration: none; }
.btn.alt { background: transparent; color: var(--text); box-shadow: inset 0 0 0 1px var(--border); }
.btn.alt:hover { background: var(--bg2); color: var(--text); }

.hero { padding: 88px 0 72px; {{if eq .Header "tint"}}background: linear-gradient(var(--accent-soft), var(--bg));{{end}} }
.hero .wrap { {{if eq .Hero "split"}}display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 56px; align-items: center;{{else}}text-align: center;{{end}} }
.hero h1 { margin: 0 0 20px; font-size: clamp(2.25rem, 4.5vw, 3.25rem); line-height: 1.12; letter-spacing: -.025em; {{if eq .Hero "center"}}max-width: 820px; margin-left: auto; margin-right: auto;{{end}} }
.hero p.lead { margin: 0 0 32px; color: var(--muted); font-size: 1.1875rem; {{if eq .Hero "center"}}max-width: 680px; margin-left: auto; margin-right: auto;{{end}} }
.hero .actions { display: flex; flex-wrap: wrap; gap: 12px; {{if eq .Hero "center"}}justify-content: center;{{end}} }
.hero .install { max-width: 100%; margin-top: 28px; {{if eq .Hero "center"}}display: inline-block;{{end}} }
.hero .install pre { margin: 0; text-align: left; }
.hero .art { {{if eq .Hero "center"}}max-width: 640px; margin: 56px auto 0;{{end}} }
.hero .art svg { display: block; width: 100%; height: auto; }
.release { display: inline-flex; gap: 8px; align-items: center; margin-bottom: 24px; padding: 4px 12px 4px 4px; border: 1px solid var(--border); border-radius: 999px; color: var(--muted); font-size: .875rem; text-decoration: none; }
.release b { padding: 2px 10px; border-radius: 999px; background: var(--accent-soft); color: var(--accent); font-weight: 600; }
.release:hover { color: var(--text); text-decoration: none; }

.band { padding: 72px 0; border-top: 1px solid var(--border); }
.band.alt { background: var(--bg2); }
.band h2 { margin: 0 0 12px; font-size: 1.875rem; text-align: center; }
.band p.sub { max-width: 640px; margin: 0 auto 48px; color: var(--muted); text-align: center; }
.features { display: grid; grid-template-columns: repeat({{.Columns}}, minmax(0, 1fr)); gap: 24px; }
.feature { padding: 24px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--bg); }
.feature .icon { display: grid; place-items: center; width: 40px; height: 40px; margin-bottom: 16px; border-radius: calc(var(--radius) - 2px); background: var(--accent-soft); color: var(--accent); }
.feature .icon svg { width: 22px; height: 22px; }
.feature h3 { margin: 0 0 8px; font-size: 1.0625rem; }
.feature p { margin: 0; color: var(--muted); font-size: .9375rem; }
.feature code { padding: .125em .375em; border-radius: 4px; background: var(--inline-code); color: var(--text); }
.example { display: grid; grid-template-columns: minmax(0, 5fr) minmax(0, 6fr); gap: 48px; align-items: start; }
.example h2 { text-align: left; }
.example p { color: var(--muted); }
.example pre { margin: 0; }

.docs { display: grid; grid-template-columns: 236px minmax(0, 1fr){{if .TOC}} 208px{{end}}; gap: 48px; }
.side { position: sticky; top: 61px; align-self: start; max-height: calc(100vh - 61px); overflow-y: auto; scrollbar-gutter: stable; padding: 28px 8px 40px 0; font-size: .9375rem; }
.side p { margin: 20px 0 6px; color: var(--text); font-size: .75rem; font-weight: 700; letter-spacing: .06em; text-transform: uppercase; }
.side p:first-child { margin-top: 0; }
.side ul { margin: 0; padding: 0; list-style: none; }
.side a { display: block; padding: 5px 10px; color: var(--muted); text-decoration: none; {{if eq .Side "bar"}}border-left: 2px solid var(--border); border-radius: 0;{{else}}border-radius: calc(var(--radius) - 2px);{{end}} }
.side a:hover { color: var(--text); text-decoration: none; }
.side a[aria-current] { color: var(--accent); font-weight: 600; {{if eq .Side "bar"}}border-left-color: var(--accent);{{else if eq .Side "pill"}}background: var(--accent-soft);{{end}} }
.toc { position: sticky; top: 61px; align-self: start; padding: 32px 0; font-size: .8125rem; }
.toc p { margin: 0 0 8px; font-weight: 600; }
.toc ul { margin: 0; padding: 0; list-style: none; }
.toc li { margin: 4px 0; }
.toc li.sub { padding-left: 12px; }
.toc a { color: var(--muted); text-decoration: none; }
.toc a:hover { color: var(--accent); }

.doc { min-width: 0; padding: 32px 0 64px; }
.doc .prose { max-width: {{.Width}}rem; }
.crumbs { margin-bottom: 8px; color: var(--muted); font-size: .875rem; }
.crumbs a { color: var(--muted); }
.prose h1 { margin: 0 0 20px; font-size: {{.H1}}; letter-spacing: -.02em; }
.prose h2 { margin: 44px 0 14px; padding-top: 4px; font-size: 1.5rem; }
.prose h3 { margin: 32px 0 10px; font-size: 1.1875rem; }
.prose h2:first-of-type { margin-top: 32px; }
.prose p, .prose ul, .prose ol { margin: 0 0 16px; }
.prose li { margin: 4px 0; }
.prose ul, .prose ol { padding-left: 24px; }
.anchor { float: left; margin-left: -1em; padding-right: .25em; color: var(--muted); opacity: 0; text-decoration: none; }
h2:hover .anchor, h3:hover .anchor { opacity: 1; }
.prose code { padding: .125em .375em; border-radius: 4px; background: var(--inline-code); }
.prose a code { color: inherit; }
.prose h2 code, .prose h3 code { padding: 0; background: none; font-size: .9em; }
pre.code { position: relative; margin: 0 0 20px; padding: 16px 20px; overflow-x: auto; border-radius: var(--radius); background: var(--code-bg); color: var(--code-text); line-height: 1.6; {{if not .DarkCode}}border: 1px solid var(--border);{{end}} }
pre.code code { padding: 0; background: none; font-size: .8125rem; }
pre .c { color: var(--code-dim); }
pre .p { color: var(--code-dim); user-select: none; }
pre .k { color: var(--code-key); }
.copy { position: absolute; top: 8px; right: 8px; padding: 2px 8px; border: 1px solid transparent; border-radius: 4px; background: none; color: var(--code-dim); font: 12px/18px var(--font); cursor: pointer; opacity: 0; }
pre:hover .copy, .copy:focus { opacity: 1; border-color: var(--code-dim); }
.note { margin: 0 0 20px; padding: 12px 16px; border-left: 3px solid var(--accent); border-radius: 0 var(--radius) var(--radius) 0; background: var(--accent-soft); }
.note p { margin: 0; }
.table { margin: 0 0 20px; overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: .9375rem; }
th, td { padding: 8px 12px; border-bottom: 1px solid var(--border); text-align: left; vertical-align: top; }
th { font-weight: 600; white-space: nowrap; }
td code { white-space: nowrap; }
figure.diagram { margin: 24px 0 28px; padding: 20px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--bg2); }
figure.diagram svg { display: block; width: 100%; height: auto; }

.feedback { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; max-width: {{.Width}}rem; margin-top: 48px; padding: 16px 0; border-top: 1px solid var(--border); color: var(--muted); font-size: .9375rem; }
.feedback button { height: 32px; padding: 0 14px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--bg); color: var(--text); font: inherit; cursor: pointer; }
.feedback button:hover { border-color: var(--accent); color: var(--accent); }
.feedback .edited { margin-left: auto; font-size: .8125rem; }
.pager { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; max-width: {{.Width}}rem; margin-top: 16px; }
.pager a { padding: 12px 16px; border: 1px solid var(--border); border-radius: var(--radius); color: var(--text); text-decoration: none; }
.pager a:hover { border-color: var(--accent); text-decoration: none; }
.pager small { display: block; color: var(--muted); font-size: .8125rem; }
.pager .next { grid-column: 2; text-align: right; }

.plain { max-width: 820px; padding-top: 48px; padding-bottom: 72px; }
.plain h1 { font-size: {{.H1}}; margin: 0 0 12px; }
.plain .lead { margin: 0 0 32px; color: var(--muted); font-size: 1.125rem; }
.notfound { padding: 120px 24px; text-align: center; }
.notfound .code { color: var(--accent); font: 700 4.5rem/1 var(--font-head); }
.notfound h1 { margin: 16px 0 8px; }
.notfound p { color: var(--muted); }

.foot { padding: 40px 0; border-top: 1px solid var(--border); color: var(--muted); font-size: .875rem; }
.foot .wrap { display: flex; flex-wrap: wrap; gap: 12px 32px; justify-content: space-between; }
.foot nav { display: flex; gap: 20px; }
.foot a { color: var(--muted); }
.foot a:hover { color: var(--text); }

@media (max-width: 1180px) {
  .docs { grid-template-columns: 220px minmax(0, 1fr); }
  .toc { display: none; }
}
@media (max-width: 900px) {
  .hero .wrap, .example, .features { grid-template-columns: minmax(0, 1fr); }
  .docs { display: block; }
  .side { display: none; position: fixed; inset: 61px 0 0; z-index: 9; max-height: none; padding: 24px; background: var(--bg); }
  .menu-open .side { display: block; }
  .menu { display: block; }
  .main-nav { display: none; }
  .search { width: auto; flex: 1; }
  .search kbd { display: none; }
}
@media (max-width: 560px) {
  .wrap { padding: 0 16px; }
  .top .wrap { gap: 14px; }
  .brand span { display: none; }
  .hero { padding: 56px 0 48px; }
  .band { padding: 56px 0; }
  .pager { grid-template-columns: 1fr; }
  .pager .next { grid-column: 1; }
}
`))

// Tint, TintA, Acc and AccSoft are the colours the style sheet is written
// with: neutrals tinted by the theme, the accent and a pale accent.
func (t *theme) Tint(light int) string { return t.tint(light) }

func (t *theme) TintA(light, alpha int) string {
	return fmt.Sprintf("hsl(%d %d%% %d%% / .%d)", (t.TintHue+360)%360, t.TintSat, light, alpha)
}

func (t *theme) Acc(light int) string { return t.hsl(t.Sat, light) }

func (t *theme) AccSoft(light int) string { return t.hsl(max(t.Sat-20, 30), light) }
