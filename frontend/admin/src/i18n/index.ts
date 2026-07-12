import { createI18n } from 'vue-i18n'
import { getCookie, setCookie } from '../utils/cookies'
import enLocales from './locales/en'
import zhLocales from './locales/zh'
import enViews from './views/en'
import zhViews from './views/zh'

const messages = {
  en: {
    ...enLocales,
    ...enViews
  },
  zh: {
    ...zhLocales,
    ...zhViews
  }
}

const savedLanguage = getCookie('language') || 'en'

const i18n = createI18n({
  legacy: false,
  locale: savedLanguage,
  fallbackLocale: 'en',
  messages
})

export const setLocale = (locale: string) => {
  i18n.global.locale.value = locale as 'en' | 'zh'
  setCookie('language', locale, 365)
}

export default i18n