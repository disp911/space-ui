import vue from '@vitejs/plugin-vue'
import { defineConfig, type Plugin } from 'vite'
import { loadMessages, localeFiles, translationDir, writeSchema, type Locale } from './build/i18n.ts'

const prefix = 'virtual:locale/'

/** Serves the frontend's translations as virtual:locale/<locale> modules. */
function translations(): Plugin {
  return {
    name: 'space-translations',
    buildStart() {
      writeSchema()
      this.addWatchFile(translationDir)
    },
    resolveId(id) {
      return id.startsWith(prefix) ? `\0${id}` : undefined
    },
    load(id) {
      if (!id.startsWith(`\0${prefix}`)) return
      const locale = id.slice(prefix.length + 1) as Locale
      if (!(locale in localeFiles)) throw new Error(`Unknown locale ${locale}`)
      return `export default ${JSON.stringify(loadMessages(locale))}`
    },
    configureServer(server) {
      server.watcher.add(translationDir)
      server.watcher.on('change', (file) => {
        if (!file.endsWith('.toml')) return
        writeSchema()
        for (const locale of Object.keys(localeFiles)) {
          const mod = server.moduleGraph.getModuleById(`\0${prefix}${locale}`)
          if (mod) server.moduleGraph.invalidateModule(mod)
        }
        server.ws.send({ type: 'full-reload' })
      })
    },
  }
}

// In development the API goes to a running panel: PANEL_URL=http://host:port npm run dev
const panel = process.env.PANEL_URL ?? 'http://127.0.0.1:2053'

export default defineConfig({
  plugins: [vue(), translations()],
  // Relative URLs: the panel serves the app under its own base path and adds <base>.
  base: './',
  build: {
    outDir: '../web/app',
    emptyOutDir: true,
    // The panel, and the subscription page that the subscription server serves on its own.
    rolldownOptions: { input: { index: 'index.html', sub: 'sub.html' } },
  },
  server: {
    proxy: {
      '^/(panel|login|logout|getTwoFactorEnable)': {
        target: panel,
        changeOrigin: true,
        // Page loads stay with the app: its /login screen shares the path of the server's endpoint.
        bypass: (req) => (req.method === 'GET' && req.headers.accept?.includes('text/html') ? '/index.html' : undefined),
      },
      '/ws': { target: panel, ws: true, changeOrigin: true },
    },
  },
})
