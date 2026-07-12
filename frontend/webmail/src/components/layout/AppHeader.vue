<!-- src/components/layout/AppHeader.vue -->
<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { useSettingStore } from '../../stores/setting'
import { useI18n } from 'vue-i18n'
import { MagnifyingGlassIcon, Cog6ToothIcon, PlusIcon, SunIcon, MoonIcon } from '@heroicons/vue/24/outline'
import ConfirmModal from '../../components/ConfirmModal.vue'
import { applyTheme } from '../../utils/theme'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const settingStore = useSettingStore()

// Confirm modal state
const showLogoutConfirm = ref(false)

// Theme
const theme = computed(() => settingStore.settings.theme || 'light')
const logoSrc = computed(() => theme.value === 'dark' ? '/logo-dark.png' : '/logo-light.png')

function openCompose() {
  router.push({ name: 'Compose' })
}

function openSettings() {
  router.push({ name: 'Settings' })
}

function openLogoutConfirm() {
  showLogoutConfirm.value = true
}

async function handleLogout() {
  try {
    await authStore.logoutAction()
    router.push({ name: 'Login' })
  } catch (error) {
    console.error('Logout error:', error)
  }
}

// Get user initial - use computed to reactively update when user changes
const userInitial = computed(() => {
  const name = authStore.user?.name
  if (name) return name.charAt(0).toUpperCase()
  const email = authStore.user?.email
  if (email) {
    const localPart = email.split('@')[0]
    return localPart.charAt(0).toUpperCase()
  }
})

// Display name for the header: name > email local part, truncated to 8 chars with ".." in the middle
const displayName = computed(() => {
  const raw = authStore.userName || (authStore.user?.email ? authStore.user.email.split('@')[0] : '')
  if (!raw) return ''
  if (raw.length <= 8) return raw
  const left = Math.floor((8 - 2) / 2)
  const right = 8 - left - 2
  return raw.slice(0, left) + '..' + raw.slice(-right)
})

// Change theme
async function changeTheme(newTheme: 'light' | 'dark') {
  settingStore.settings.theme = newTheme
  applyTheme(newTheme)
  await settingStore.saveSettings()
}
</script>

<template>
  <header class="h-14 flex items-center bg-blue-600 border-b border-blue-700 shrink-0">
    <!-- Left: Logo -->
    <div class="flex items-center px-4 flex-shrink-0 w-48">
      <img :src="logoSrc" alt="EasyMail" class="h-8" />
      <span class="text-2xl font-bold text-white px-2">{{ t('app.name') }}</span>
    </div>

    <!-- Center: New Mail + Search Bar -->
    <div class="flex-1 flex items-center gap-4 px-12">
      <!-- New Mail Button -->
      <button 
        @click="openCompose"
        class="flex items-center gap-2 bg-white/20 hover:bg-white/30 text-white px-4 h-9 rounded-md text-sm font-semibold transition-colors flex-shrink-0"
      >
        <PlusIcon class="w-4 h-4" />
        <span class="hidden sm:inline">{{ t('app.newMail') }}</span>
      </button>

      <!-- Search Bar -->
      <div class="flex-1 max-w-xl">
        <div class="relative">
          <MagnifyingGlassIcon class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-white/60" />
          <input 
            type="text" 
            :placeholder="t('app.searchPlaceholder')" 
            class="w-full pl-10 pr-4 h-10 rounded-full bg-white/20 border border-white/30 text-white text-sm placeholder-white/60 outline-none focus:bg-white/30 focus:border-white/50 transition-all"
          />
        </div>
      </div>
    </div>

    <!-- Right: User Actions -->
    <div class="flex items-center gap-2 px-4 flex-shrink-0 w-48 justify-end">
      <!-- Theme Switch -->
      <div class="flex items-center gap-1">
        <button
          @click="changeTheme('light')"
          :class="['p-2 rounded-full transition-colors', theme === 'light' ? 'bg-white/30 text-white' : 'hover:bg-white/20 text-white']"
          :title="t('app.light')"
        >
          <SunIcon class="w-5 h-5" />
        </button>
        <button
          @click="changeTheme('dark')"
          :class="['p-2 rounded-full transition-colors', theme === 'dark' ? 'bg-white/30 text-white' : 'hover:bg-white/20 text-white']"
          :title="t('app.dark')"
        >
          <MoonIcon class="w-5 h-5" />
        </button>
      </div>

      <!-- GitHub -->
      <a
        href="https://github.com/bobxiao188/easymail"
        target="_blank"
        rel="noopener noreferrer"
        class="p-2 rounded-full hover:bg-white/20 text-white transition-colors"
        title="GitHub"
      >
        <svg viewBox="0 0 24 24" class="w-5 h-5" fill="currentColor">
          <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
        </svg>
      </a>

      <!-- Settings -->
      <button 
        @click="openSettings"
        class="p-2.5 rounded-full hover:bg-white/20 text-white transition-colors"
        :title="t('app.settings')"
      >
        <Cog6ToothIcon class="w-5 h-5" />
      </button>

      <!-- User Avatar -->
      <div 
        @click="openLogoutConfirm"
        class="flex items-center gap-2 cursor-pointer group"
        :title="authStore.userName || authStore.userEmail || t('app.logout')"
      >
        <div class="w-9 h-9 rounded-full bg-white/20 hover:bg-white/30 text-white flex items-center justify-center text-sm font-bold transition-colors">
          {{ userInitial }}
        </div>
        <span v-if="displayName" class="hidden sm:inline text-sm text-white/80 group-hover:text-white transition-colors">
          {{ displayName }}
        </span>
      </div>
    </div>
  </header>

  <!-- Logout Confirm Modal -->
  <ConfirmModal
    v-model="showLogoutConfirm"
    :title="t('app.confirmLogout')"
    :message="t('app.logoutMessage')"
    :confirm-text="t('app.logout')"
    :loading="false"
    @confirm="handleLogout"
    @close="showLogoutConfirm = false"
  />
</template>