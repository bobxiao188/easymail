<template>
  <div class="flex w-full min-h-screen flex-col lg:flex-row dark:bg-dark-bg">
    <!-- ====== Left 60% — Brand ====== -->
    <div class="login-brand-side hidden lg:flex lg:w-[60%] flex-col justify-center items-center dark:bg-dark-bg">
      <div class="login-brand-content relative z-[1] max-w-[620px]">
        <div class="inline-flex items-center gap-2.5 mb-10">
          <div class="inline-flex items-center justify-center rounded-[10px] text-white">
            <img :src="logoSrc" class="object-contain max-h-[42px]" alt="EasyMail Logo">
          </div>
          <span class="text-[32px] font-semibold tracking-tight text-gray-800 dark:text-dark-text">{{ t('login.brand') }}</span>
        </div>

        <h1 class="text-[36px] font-semibold tracking-tight leading-[1.25] text-gray-800 dark:text-dark-text mb-4">
          <span class="block">{{ t('login.taglineLine1') }}</span>
          <span class="block">{{ t('login.taglineLine2') }}</span>
          <span class="block">{{ t('login.taglineLine3') }}</span>
        </h1>

        <p class="text-base text-gray-500 dark:text-gray-400 mb-12 leading-relaxed">
          {{ t('login.description') }}
        </p>

        <div class="grid grid-cols-2 gap-5 py-6">
          <!-- Powerful Anti-spam Engine -->
          <div class="group bg-white/80 backdrop-blur-sm border border-gray-200 dark:bg-dark-surface dark:border-dark-border rounded-xl p-4 transition-all duration-300 hover:shadow-lg hover:border-blue-200 hover:-translate-y-0.5">
            <div class="flex items-center gap-2.5 mb-2.5">
              <div class="shrink-0 w-10 h-10 rounded-lg bg-gradient-to-br from-blue-500 to-indigo-600 text-white flex items-center justify-center shadow-sm">
                <svg viewBox="0 0 24 24" class="w-[20px] h-[20px]" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
                </svg>
              </div>
              <h4 class="text-[15px] font-semibold text-gray-800 dark:text-dark-text">{{ t('login.feature1Title') }}</h4>
            </div>
            <p class="text-[13px] text-gray-500 dark:text-gray-400 leading-relaxed mb-2.5">{{ t('login.feature1Desc') }}</p>
            <div class="flex flex-wrap gap-1.5">
              <span class="inline-block px-2 py-0.5 text-[11px] font-medium rounded-md bg-blue-50 text-blue-700">FastText</span>
              <span class="inline-block px-2 py-0.5 text-[11px] font-medium rounded-md bg-indigo-50 text-indigo-700">DistilBERT</span>
              <span class="inline-block px-2 py-0.5 text-[11px] font-medium rounded-md bg-violet-50 text-violet-700">XGBoost</span>
            </div>
          </div>

          <!-- Full Protocol Support -->
          <div class="group bg-white/80 backdrop-blur-sm border border-gray-200 dark:bg-dark-surface dark:border-dark-border rounded-xl p-4 transition-all duration-300 hover:shadow-lg hover:border-blue-200 hover:-translate-y-0.5">
            <div class="flex items-center gap-2.5 mb-2.5">
              <div class="shrink-0 w-10 h-10 rounded-lg bg-gradient-to-br from-emerald-500 to-teal-600 text-white flex items-center justify-center shadow-sm">
                <svg viewBox="0 0 24 24" class="w-[20px] h-[20px]" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
                  <polyline points="22,6 12,13 2,6"/>
                </svg>
              </div>
              <h4 class="text-[15px] font-semibold text-gray-800 dark:text-dark-text">{{ t('login.feature2Title') }}</h4>
            </div>
            <p class="text-[13px] text-gray-500 dark:text-gray-400 leading-relaxed mb-2.5">{{ t('login.feature2Desc') }}</p>
            <div class="flex flex-wrap gap-1.5">
              <span class="inline-block px-2 py-0.5 text-[11px] font-medium rounded-md bg-emerald-50 text-emerald-700">SMTP</span>
              <span class="inline-block px-2 py-0.5 text-[11px] font-medium rounded-md bg-teal-50 text-teal-700">IMAP4</span>
              <span class="inline-block px-2 py-0.5 text-[11px] font-medium rounded-md bg-cyan-50 text-cyan-700">LMTP</span>
            </div>
          </div>

          <!-- High Performance -->
          <div class="group bg-white/80 backdrop-blur-sm border border-gray-200 dark:bg-dark-surface dark:border-dark-border rounded-xl p-4 transition-all duration-300 hover:shadow-lg hover:border-blue-200 hover:-translate-y-0.5">
            <div class="flex items-center gap-2.5 mb-2.5">
              <div class="shrink-0 w-10 h-10 rounded-lg bg-gradient-to-br from-amber-500 to-orange-600 text-white flex items-center justify-center shadow-sm">
                <svg viewBox="0 0 24 24" class="w-[20px] h-[20px]" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
                </svg>
              </div>
              <h4 class="text-[15px] font-semibold text-gray-800 dark:text-dark-text">{{ t('login.feature3Title') }}</h4>
            </div>
            <p class="text-[13px] text-gray-500 dark:text-gray-400 leading-relaxed mb-2.5">{{ t('login.feature3Desc') }}</p>
            <div class="flex flex-wrap gap-1.5">
              <span class="inline-block px-2 py-0.5 text-[11px] font-medium rounded-md bg-amber-50 text-amber-700">Go</span>
              <span class="inline-block px-2 py-0.5 text-[11px] font-medium rounded-md bg-orange-50 text-orange-700">Redis</span>
              <span class="inline-block px-2 py-0.5 text-[11px] font-medium rounded-md bg-yellow-50 text-yellow-700">SQLite</span>
            </div>
          </div>

          <!-- Self-hosted & Secure -->
          <div class="group bg-white/80 backdrop-blur-sm border border-gray-200 dark:bg-dark-surface dark:border-dark-border rounded-xl p-4 transition-all duration-300 hover:shadow-lg hover:border-blue-200 hover:-translate-y-0.5">
            <div class="flex items-center gap-2.5 mb-2.5">
              <div class="shrink-0 w-10 h-10 rounded-lg bg-gradient-to-br from-rose-500 to-pink-600 text-white flex items-center justify-center shadow-sm">
                <svg viewBox="0 0 24 24" class="w-[20px] h-[20px]" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <rect x="3" y="11" width="18" height="11" rx="2"/>
                  <path d="M7 11V7a5 5 0 0110 0v4"/>
                </svg>
              </div>
              <h4 class="text-[15px] font-semibold text-gray-800 dark:text-dark-text">{{ t('login.feature4Title') }}</h4>
            </div>
            <p class="text-[13px] text-gray-500 dark:text-gray-400 leading-relaxed mb-2.5">{{ t('login.feature4Desc') }}</p>
            <div class="flex flex-wrap gap-1.5">
              <span class="inline-block px-2 py-0.5 text-[11px] font-medium rounded-md bg-rose-50 text-rose-700">AGPLv3</span>
              <span class="inline-block px-2 py-0.5 text-[11px] font-medium rounded-md bg-pink-50 text-pink-700">ClamAV</span>
              <span class="inline-block px-2 py-0.5 text-[11px] font-medium rounded-md bg-fuchsia-50 text-fuchsia-700">Webmail</span>
              <span class="inline-block px-2 py-0.5 text-[11px] font-medium rounded-md bg-purple-50 text-purple-700">Admin</span>
            </div>
          </div>
        </div>
      </div>

      <div class="login-footer-note absolute bottom-5 left-14 right-14 flex items-center justify-between text-xs text-gray-400 z-[1]">
        <span>{{ t('login.footer') }}</span>
        <a
          href="https://github.com/bobxiao188/easymail"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex items-center gap-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors no-underline"
        >
          <svg viewBox="0 0 24 24" class="w-4 h-4" fill="currentColor">
            <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
          </svg>
          <span>github.com/bobxiao188/easymail</span>
        </a>
      </div>
    </div>

    <!-- ====== Right 40% — Login Form ====== -->
    <div class="flex-1 flex flex-col justify-center items-center p-12 lg:w-[40%] lg:p-12 relative dark:bg-dark-bg">
      <div class="w-full max-w-[460px]">
        <!-- Language Selector -->
        <div class="flex items-center gap-2 mb-6">
          <button
            @click="changeLanguage('zh')"
            :class="['px-3 py-1.5 text-sm font-medium rounded-lg transition-colors', currentLang === 'zh' ? 'bg-blue-600 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-surface']"
          >
            {{ t('languages.zh') }}
          </button>
          <button
            @click="changeLanguage('en')"
            :class="['px-3 py-1.5 text-sm font-medium rounded-lg transition-colors', currentLang === 'en' ? 'bg-blue-600 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-surface']"
          >
            {{ t('languages.en') }}
          </button>
        </div>

        <p class="text-[32px] font-bold tracking-tight mb-2 text-gray-900 dark:text-dark-text py-6">
          {{ t('login.welcome') }}
        </p>
        <p class="text-[16px] text-gray-500 dark:text-gray-400 mb-10 leading-relaxed py-3">
          {{ t('login.signInDesc') }}
        </p>

        <form @submit.prevent="handleLogin" novalidate>
          <div class="mb-5">
            <label class="block text-[16px] font-semibold text-gray-700 dark:text-gray-300 mb-2" for="email">
              {{ t('login.email') }}
            </label>
            <input
              id="email"
              class="w-full py-3 px-4 text-[16px] text-gray-900 bg-white border border-gray-300 dark:text-dark-text dark:bg-dark-surface dark:border-dark-border rounded-xl outline-none transition-all focus:border-blue-500 focus:ring-4 focus:ring-blue-500/10 placeholder:text-gray-400"
              :class="{ 'border-red-400 focus:border-red-400 focus:ring-red-500/10': errors.email }"
              type="text"
              inputmode="email"
              v-model="email"
              :placeholder="t('login.emailPlaceholder')"
              autocomplete="username"
            />
            <p v-if="errors.email" class="mt-2 text-[13px] text-red-500 flex items-center gap-1">
              <svg class="w-3.5 h-3.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
              </svg>
              {{ errors.email }}
            </p>
          </div>

          <div class="mb-5">
            <label class="block text-[14px] font-semibold text-gray-700 dark:text-gray-300 mb-2" for="password">
              {{ t('login.password') }}
            </label>
            <input
              id="password"
              class="w-full py-3 px-4 text-[16px] text-gray-900 bg-white border border-gray-300 dark:text-dark-text dark:bg-dark-surface dark:border-dark-border rounded-xl outline-none transition-all focus:border-blue-500 focus:ring-4 focus:ring-blue-500/10 placeholder:text-gray-400"
              :class="{ 'border-red-400 focus:border-red-400 focus:ring-red-500/10': errors.password }"
              type="password"
              v-model="password"
              :placeholder="t('login.passwordPlaceholder')"
              autocomplete="current-password"
            />
            <p v-if="errors.password" class="mt-2 text-[13px] text-red-500 flex items-center gap-1">
              <svg class="w-3.5 h-3.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
              </svg>
              {{ errors.password }}
            </p>
          </div>

          <div class="flex items-center justify-between mb-8">
            <label class="flex items-center gap-2.5 cursor-pointer">
              <input type="checkbox" v-model="rememberMe" class="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500 focus:ring-offset-0 accent-blue-600 cursor-pointer" />
              <span class="text-[14px] text-gray-600 dark:text-gray-400">{{ t('login.rememberMe') }}</span>
            </label>
            <a href="#" class="text-[14px] text-blue-600 no-underline font-semibold hover:text-blue-700 transition-colors" @click.prevent v-if="false">
              {{ t('login.forgotPassword') }}
            </a>
          </div>

          <button
            type="submit"
            class="w-full py-3.5 px-5 text-[16px] font-semibold text-white bg-blue-600 rounded-xl shadow-lg shadow-blue-600/25 cursor-pointer transition-all hover:bg-blue-700 hover:shadow-xl hover:shadow-blue-600/30 active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed inline-flex items-center justify-center gap-2"
            :disabled="isLoading"
          >
            <svg v-if="isLoading" class="animate-spin w-5 h-5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
            </svg>
            <template v-if="isLoading">{{ t('login.signingIn') }}</template>
            <template v-else>{{ t('login.signIn') }}</template>
          </button>
        </form>

        <div class="text-center mt-8 text-[14px] text-gray-500 dark:text-gray-400" v-if="false">
          {{ t('login.noAccount') }} <a href="#" class="text-blue-600 no-underline font-semibold hover:text-blue-700 transition-colors">{{ t('login.createAccount') }}</a>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useI18n } from 'vue-i18n'
import { setLocale } from '../i18n'
import { showToast } from '../utils/toast'

const { t, locale } = useI18n()
const router = useRouter()
const authStore = useAuthStore()

const currentLang = ref<'zh' | 'en'>(locale.value as 'zh' | 'en')

// Logo switches based on theme (dark class on <html>)
const logoSrc = computed(() =>
  document.documentElement.classList.contains('dark') ? '/logo-dark.png' : '/logo-light.png'
)

// Change language
function changeLanguage(lang: 'zh' | 'en') {
  currentLang.value = lang
  setLocale(lang)
  locale.value = lang
}

onMounted(() => {
  currentLang.value = locale.value as 'zh' | 'en'
})

const email = ref('')
const password = ref('')
const rememberMe = ref(true)
const isLoading = ref(false)
const errors = ref({ email: '', password: '' })

function validate() {
  errors.value = { email: '', password: '' }

  if (!email.value) {
    errors.value.email = t('login.enterEmail')
    return false
  }

  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!emailRegex.test(email.value)) {
    errors.value.email = t('login.invalidEmail')
    return false
  }

  if (!password.value) {
    errors.value.password = t('login.enterPassword')
    return false
  }

  if (password.value.length < 6) {
    errors.value.password = t('login.passwordLength')
    return false
  }

  return true
}

function getErrorMessage(error: any): string {
  const code = error.response?.data?.code
  const message = error.response?.data?.message

  switch (code) {
    case 400:
      return t('login.fillFields')
    case 401:
      return t('login.invalidCredentials')
    case 500:
      return t('login.serviceUnavailable')
    default:
      return message || t('login.loginFailed')
  }
}

async function handleLogin() {
  if (!validate()) return

  isLoading.value = true
  try {
    await authStore.loginAction(email.value, password.value)
    router.push('/inbox')
  } catch (error: any) {
    const message = getErrorMessage(error)
    showToast('error', message)
  } finally {
    isLoading.value = false
  }
}
</script>

<style scoped>
/* === Left Brand Side Decorative Pseudo-elements === */
.login-brand-side {
  background: #f8fafc;
  flex-direction: column;
  justify-content: center;
  padding: 48px 56px;
  position: relative;
  overflow: hidden;
}
.login-brand-side::before {
  content: '';
  position: absolute;
  top: -120px;
  right: -120px;
  width: 400px;
  height: 400px;
  background: rgba(59, 130, 246, 0.03);
  border-radius: 50%;
  pointer-events: none;
}
.login-brand-side::after {
  content: '';
  position: absolute;
  bottom: -60px;
  left: -60px;
  width: 240px;
  height: 240px;
  background: rgba(59, 130, 246, 0.04);
  border-radius: 50%;
  pointer-events: none;
}

/* === Responsive Adjustments === */
@media (max-width: 900px) {
  .login-brand-side {
    padding: 40px 32px 32px;
  }
  .login-footer-note {
    position: static !important;
    padding-left: 32px;
    padding-bottom: 16px;
  }
}
@media (max-width: 480px) {
  .login-brand-side {
    padding: 28px 20px 24px;
  }
}
</style>
