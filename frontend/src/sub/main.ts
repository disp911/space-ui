import { createPinia } from 'pinia'
import { createApp } from 'vue'
import '../styles/tokens.css'
import '../styles/base.css'
import '../kit/kit.css'
import { i18n, initialLocale, setLocale } from '../i18n'
import { useTheme } from '../stores/theme'
import { loadData } from './data'
import SubPage from './SubPage.vue'

// The subscription page: what a client sees opening its subscription link in
// a browser. The subscription server serves it apart from the panel.
const data = await loadData()
const app = createApp(SubPage, { data }).use(createPinia())
useTheme()
app.use(i18n)
await setLocale(initialLocale())
app.mount('#app')
