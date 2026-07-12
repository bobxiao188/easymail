import { createApp } from 'vue'
import App from './App.vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import router from './router'
import { createPinia } from 'pinia'
import { setUnauthorizedHandler } from './api'
import { useAuthStore } from './stores/auth'
import './styles/global.css'
import i18n from './i18n'

const app = createApp(App)

// 全局注册Element Plus图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(ElementPlus)
app.use(createPinia())
app.use(router)
app.use(i18n)

setUnauthorizedHandler(() => {
  const auth = useAuthStore()
  auth.logout()
  if (router.currentRoute.value.name !== 'Login') {
    void router.push({
      name: 'Login',
      query: { redirect: router.currentRoute.value.fullPath }
    })
  }
})

app.mount('#app')