import { createI18n } from 'vue-i18n'
import { getCookie, setCookie } from '../utils/cookies'
import enLocales from './locales/en'
import zhLocales from './locales/zh'
import enViews from './views/en'
import zhViews from './views/zh'

function deepMerge(target: Record<string, any>, source: Record<string, any>): Record<string, any> {
  const output = { ...target }
  for (const key of Object.keys(source)) {
    if (
      source[key] !== null &&
      typeof source[key] === 'object' &&
      !Array.isArray(source[key]) &&
      target[key] !== null &&
      typeof target[key] === 'object' &&
      !Array.isArray(target[key])
    ) {
      output[key] = deepMerge(target[key], source[key])
    } else {
      output[key] = source[key]
    }
  }
  return output
}

const messages = {
  en: deepMerge(enLocales, enViews),
  zh: deepMerge(zhLocales, zhViews)
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
