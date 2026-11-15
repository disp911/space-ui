import { createPinia } from 'pinia'
import { createApp } from 'vue'
import './styles/tokens.css'
import './styles/base.css'
import './kit/kit.css'
import { watch } from 'vue'
import { handleSessionLost } from './api'
import App from './App.vue'
import { i18n, initialLocale, setLocale } from './i18n'
import { router } from './router'
import { useSession } from './stores/session'
import { useTheme } from './stores/theme'

const app = createApp(App).use(createPinia())
const session = useSession()
useTheme()
handleSessionLost(() => (session.loggedIn = false))
watch(
  () => session.loggedIn,
  (loggedIn) => {
    if (!loggedIn && !router.currentRoute.value.meta.public) {
      const next = router.currentRoute.value.fullPath
      router.replace({ name: 'login', query: next === '/' ? {} : { next } })
    }
  },
)
app.use(i18n).use(router)
await setLocale(initialLocale())
app.mount('#app')
