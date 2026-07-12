import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './style.css'
import App from './App.vue'
import router from './router'
import i18n from './i18n'
import dayjs from 'dayjs'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)
app.use(i18n)
app.mount('#app')

// Sync dayjs locale with i18n
import zhcn from 'dayjs/locale/zh-cn'
const syncDayjsLocale = () => {
  if (i18n.global.locale.value === 'zh') {
    dayjs.locale('zh-cn', zhcn)
  } else {
    dayjs.locale('en')
  }
}
syncDayjsLocale()
// Watch for locale changes
import { watch } from 'vue'
watch(() => i18n.global.locale.value, () => {
  syncDayjsLocale()
})
