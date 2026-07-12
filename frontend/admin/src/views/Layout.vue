<template>
  <div class="layout-container">
    <el-aside width="240px" class="sidebar glass-card">
      <div class="logo-section">
        <div class="logo-icon">
          <img :src="logoSrc" alt="EasyMail" class="logo-image">
          <span class="logo-text">EasyMail</span>
        </div>
        <a
          href="https://github.com/bobxiao188/easymail"
          target="_blank"
          rel="noopener noreferrer"
          class="gradient-text-primary github-version-link"
        >
          <svg viewBox="0 0 24 24" class="github-icon-inline" fill="currentColor">
            <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
          </svg>
          GitHub
        </a>
      </div>

      <el-menu
        :default-active="activeMenu"
        class="easy-menu"
        router
      >
        <el-menu-item index="/dashboard" class="menu-item">
          <div class="menu-icon">
            <TrendCharts />
          </div>
          <span>{{ t('menu.dashboard') }}</span>
        </el-menu-item>
        <el-menu-item index="/domains" class="menu-item">
          <div class="menu-icon">
            <User />
          </div>
          <span>{{ t('menu.domains') }}</span>
        </el-menu-item>
        <el-menu-item index="/filter/rules" class="menu-item">
          <div class="menu-icon">
            <Filter />
          </div>
          <span>{{ t('menu.FilterRules') }}</span>
        </el-menu-item>
        <el-menu-item index="/classify-models" class="menu-item">
          <div class="menu-icon">
            <Filter />
          </div>
          <span>{{ t('menu.classifyModels') }}</span>
        </el-menu-item>
        <el-menu-item index="/training" class="menu-item">
          <div class="menu-icon">
            <Filter />
          </div>
          <span>{{ t('menu.training') }}</span>
        </el-menu-item>
        <el-menu-item index="/filter/logs" class="menu-item">
          <div class="menu-icon">
            <Document />
          </div>
          <span>{{ t('menu.filterLogs') }}</span>
        </el-menu-item>

        <!-- Postfix management -->
        <el-sub-menu index="/postfix">
          <template #title>
            <div class="menu-icon">
              <Postcard />
            </div>
            <span>{{ t('menu.postfix') }}</span>
          </template>
          <el-menu-item index="/postfix/status">
            <span>{{ t('menu.postfixStatus') }}</span>
          </el-menu-item>
          <el-menu-item index="/postfix/configs">
            <span>{{ t('menu.postfixConfigs') }}</span>
          </el-menu-item>
          <el-menu-item index="/postfix/agents">
            <span>{{ t('menu.postfixAgents') }}</span>
          </el-menu-item>
          <el-menu-item index="/postfix/queue">
            <span>{{ t('menu.postfixQueue') }}</span>
          </el-menu-item>
        </el-sub-menu>
      </el-menu>

      <div class="sidebar-footer">
        <div class="status-indicator">
          <div>
            <a href="https://github.com/bobxiao188/easymail" target="_blank" rel="noopener noreferrer">
              &copy; {{ new Date().getFullYear() }} EasyMail AGPLv3 All rights reserved
            </a>
          </div>
          <div>
            <a href="mailto:3680010825@qq.com" target="_blank" rel="noopener noreferrer">
              3680010825@qq.com
            </a>
          </div>
        </div>
      </div>
    </el-aside>

    <el-container class="main-container">
      <el-header class="header glass-card-light">
        <div class="header-left">
          <h2>{{ pageTitle }}</h2>
        </div>
        <div class="header-right">
          <el-dropdown @command="switchSkin">
            <div class="skin-switcher">
              <el-icon :size="18"><Moon v-if="currentSkin === 'dark'" /><Sunny v-else /></el-icon>
              <span>{{ currentSkin === 'dark' ? t('common.dark') : t('common.light') }}</span>
            </div>
            <template #dropdown>
              <el-dropdown-menu class="easy-dropdown-menu">
                <el-dropdown-item :command="'dark'" :class="{ 'is-active': currentSkin === 'dark' }">
                  <span>{{ t('common.dark') }}</span>
                </el-dropdown-item>
                <el-dropdown-item :command="'light'" :class="{ 'is-active': currentSkin === 'light' }">
                  <span>{{ t('common.light') }}</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-dropdown @command="switchLanguage">
            <div class="language-switcher">
              <el-icon :size="18"><Location /></el-icon>
              <span>{{ currentLocale === 'en' ? 'EN' : '中文' }}</span>
            </div>
            <template #dropdown>
              <el-dropdown-menu class="easy-dropdown-menu">
                <el-dropdown-item :command="'en'" :class="{ 'is-active': currentLocale === 'en' }">
                  <span>English</span>
                </el-dropdown-item>
                <el-dropdown-item :command="'zh'" :class="{ 'is-active': currentLocale === 'zh' }">
                  <span>简体中文</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-dropdown @command="handleCommand">
            <div class="user-info">
              <div class="user-avatar easy-avatar">
                <img
                  v-if="userAvatarSrc"
                  :src="userAvatarSrc"
                  alt=""
                  class="user-avatar-img"
                />
                <template v-else>{{ userInfo?.nickname?.[0] || 'U' }}</template>
              </div>
              <div class="user-details">
                <span class="user-name">{{ userInfo?.nickname || t('common.user') }}</span>
                <span class="user-role">{{ userInfo?.isAdmin ? t('common.admin') : t('common.user') }}</span>
              </div>
              <svg class="dropdown-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="6 9 12 15 18 9"/>
              </svg>
            </div>
            <template #dropdown>
              <el-dropdown-menu class="easy-dropdown-menu">
                <el-dropdown-item command="profile">
                  <div class="dropdown-item-content">
                    <el-icon :size="18"><User /></el-icon>
                    <span>{{ t('menu.profile') }}</span>
                  </div>
                </el-dropdown-item>
                <el-dropdown-item command="logout">
                  <div class="dropdown-item-content">
                    <el-icon :size="18"><SwitchButton /></el-icon>
                    <span>{{ t('common.logout') }}</span>
                  </div>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main class="main-content">
        <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
            <component :is="Component" />
        </transition>
        </router-view>
      </el-main>
    </el-container>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'
import { ElMessage } from 'element-plus'
import { setLocale } from '../i18n'
import { Location, Moon, Sunny, Postcard } from '@element-plus/icons-vue'
import { setCookie } from '../utils/cookies'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const { t, locale } = useI18n()

const currentLocale = ref(locale.value)
const currentSkin = ref('dark')

const logoSrc = computed(() =>
  currentSkin.value === 'dark' ? '/logo-dark.png' : '/logo-light.png'
)

const switchLanguage = async (lang: string) => {
  currentLocale.value = lang
  setLocale(lang)
  if (authStore.isLoggedIn) {
    try {
      await authStore.updateLanguage(lang)
    } catch (error) {
      console.error('Failed to save language preference:', error)
    }
  }
}

const switchSkin = async (skin: string) => {
  currentSkin.value = skin
  document.documentElement.classList.remove('skin-dark', 'skin-light')
  document.documentElement.classList.add(`skin-${skin}`)
  setCookie('skin', skin, 30)
  if (authStore.isLoggedIn) {
    try {
      await authStore.updateSkin(skin)
    } catch (error) {
      console.error('Failed to save skin preference:', error)
    }
  }
}

const userInfo = computed(() => authStore.userInfo)

const userAvatarSrc = computed(() => {
  const a = userInfo.value?.avatar
  if (!a || typeof a !== 'string') return ''
  if (a.startsWith('data:image/') || a.startsWith('http://') || a.startsWith('https://')) return a
  return ''
})

const activeMenu = computed(() => route.path)

const pageTitle = computed(() => {
  const titles: Record<string, string> = {
    '/dashboard': t('menu.dashboard'),
    '/domains': t('menu.domains'),
    '/profile': t('menu.profile'),
    '/filter/rules': t('menu.FilterRules'),
    '/classify-models': t('menu.classifyModels'),
    '/training': t('menu.training'),
    '/filter/logs': t('menu.filterLogs'),
    '/postfix/status': t('menu.postfixStatus'),
    '/postfix/configs': t('menu.postfixConfigs'),
    '/postfix/agents': t('menu.postfixAgents'),
    '/postfix/queue': t('menu.postfixQueue')
  }
  return titles[route.path] || ''
})

onMounted(async () => {
  currentSkin.value = authStore.skin
  document.documentElement.classList.remove('skin-dark', 'skin-light')
  document.documentElement.classList.add(`skin-${currentSkin.value}`)
  
  if (authStore.isLoggedIn && !authStore.userInfo) {
    try {
      await authStore.getProfile()
      currentSkin.value = authStore.skin
      document.documentElement.classList.remove('skin-dark', 'skin-light')
      document.documentElement.classList.add(`skin-${currentSkin.value}`)
    } catch (error) {
      console.error('Failed to get profile:', error)
    }
  }
})

const handleCommand = async (command: string) => {
  if (command === 'profile') {
    router.push('/profile')
    return
  }
  if (command === 'logout') {
    await authStore.logout()
    ElMessage.success(t('common.logoutSuccess'))
    router.push('/login')
  }
}
</script>
<style scoped>
.layout-container {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background: transparent;
}

.sidebar {
  height: 100%;
  border: none;
  border-right: 1px solid var(--border-default);
  display: flex;
  flex-direction: column;
  padding: 0;
  background: var(--glass-bg);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
}

.logo-section {
  padding: 30px 20px;
  text-align: center;
  border-bottom: 1px solid var(--border-default);
}

.logo-icon {
  margin: 0 auto 15px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.logo-image {
  width: 100%;
  height: 42px;
  object-fit: contain;
}

.logo-text {
  font-size: 24px;
  font-weight: 700;
  letter-spacing: 1px;
  color: var(--foreground);
}

.gradient-text-primary {
  font-size: 12px;
  color: var(--foreground-muted);
  letter-spacing: 0.5px;
  font-weight: 100;
}

.github-version-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  text-decoration: none;
  transition: color 0.2s;
}

.github-version-link:hover {
  color: var(--accent);
}

.github-icon-inline {
  width: 14px;
  height: 14px;
}

.logo-section h3 {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  letter-spacing: 1px;
}

.easy-menu {
  flex: 1;
  border: none;
  background: transparent;
  padding: 20px 10px;
}

.easy-menu :deep(.el-menu-item) {
  margin-bottom: 8px;
  border-radius: 8px;
  color: var(--foreground-muted);
  transition: all 200ms cubic-bezier(0.16, 1, 0.3, 1);
  padding: 0 15px;
  height: 48px;
  display: flex;
  align-items: center;
}

.easy-menu :deep(.el-menu-item:hover) {
  background: var(--surface-hover);
  color: var(--foreground);
}

.easy-menu :deep(.el-menu-item.is-active) {
  background: var(--accent-20);
  color: var(--foreground);
  box-shadow: 0 0 20px var(--accent-glow);
}

.easy-menu :deep(.el-sub-menu) {
  margin-bottom: 8px;
}

.easy-menu :deep(.el-sub-menu__title) {
  height: 48px;
  padding: 0 15px;
  border-radius: 8px;
  color: var(--foreground-muted);
  transition: all 200ms cubic-bezier(0.16, 1, 0.3, 1);
  display: flex;
  align-items: center;
  gap: 12px;
}

.easy-menu :deep(.el-sub-menu__title:hover) {
  background: var(--surface-hover);
  color: var(--foreground);
}

.easy-menu :deep(.el-sub-menu.is-active .el-sub-menu__title) {
  background: var(--accent-20);
  color: var(--foreground);
  box-shadow: 0 0 20px var(--accent-glow);
}

.easy-menu :deep(.el-sub-menu .el-menu) {
  background: transparent;
  border: none;
}

.easy-menu :deep(.el-sub-menu .el-menu .el-menu-item) {
  margin-top: 10px;
  margin-bottom: 10px;
  padding-left: 30px;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.menu-icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.menu-icon svg {
  width: 20px;
  height: 20px;
}

.sidebar-footer {
  padding: 20px;
  border-top: 1px solid var(--border-default);
}

.status-indicator {
  text-align: center;
  gap: 8px;
  color: var(--foreground-muted);
  font-size: 12px;
}

.status-indicator a {
  color: inherit;
  text-decoration: none;
  transition: color 0.2s;
}

.status-indicator a:hover {
  color: var(--accent);
}

.status-dot {
  width: 8px;
  height: 8px;
  background: #10b981;
  border-radius: 50%;
  box-shadow: 0 0 10px rgba(16, 185, 129, 0.5);
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.7; transform: scale(1.1); }
}

.main-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.header {
  height: 70px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 30px;
  margin: 20px 30px 0;
  border-radius: 12px;
  border: none;
  background: rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border: 1px solid var(--border-default);
  box-shadow: var(--shadow-card);
}

.header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--foreground);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 20px;
}

.skin-switcher {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border-radius: 8px;
  cursor: pointer;
}

.skin-switcher span {
  font-size: 13px;
  font-weight: 500;
}

.skin-switcher:hover {
  background: var(--surface-hover);
  color: var(--foreground);
  border-color: var(--border-hover);
}

.language-switcher {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border-radius: 8px;
  cursor: pointer;
  color: var(--foreground-muted);
  transition: all 200ms ease;
  border: 1px solid transparent;
}

.language-switcher:hover {
  background: var(--surface-hover);
  color: var(--foreground);
  border-color: var(--border-hover);
}

.language-switcher span {
  font-size: 13px;
  font-weight: 500;
}

.is-active {
  color: var(--accent) !important;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 16px;
  border-radius: 10px;
  transition: all 0.3s ease;
  border: 1px solid transparent;
  cursor: pointer;
}

.user-info:hover {
  background: var(--surface-hover);
  border-color: var(--border-hover);
}

.user-avatar {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 600;
  color: white;
  border-radius: 20px;
  overflow: hidden;
}

.user-avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.user-details {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.user-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--foreground);
}

.user-role {
  font-size: 12px;
  color: var(--foreground-muted);
}

.dropdown-arrow {
  width: 16px;
  height: 16px;
  color: var(--foreground-muted);
  transition: transform 200ms cubic-bezier(0.16, 1, 0.3, 1);
}

.user-info:hover .dropdown-arrow {
  transform: rotate(180deg);
}

.easy-dropdown-menu {
  background: var(--glass-bg);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  padding: 8px;
  min-width: 180px;
  box-shadow: var(--shadow-card);
}

.easy-dropdown-menu :deep(.el-dropdown-menu__item) {
  color: var(--foreground);
  padding: 10px 12px;
  border-radius: 6px;
  transition: all 150ms ease;
}

.easy-dropdown-menu :deep(.el-dropdown-menu__item:hover) {
  background: var(--accent-10);
  color: var(--accent-bright);
}

.dropdown-item-content {
  display: flex;
  align-items: center;
  gap: 10px;
}

.dropdown-item-content svg {
  width: 18px;
  height: 18px;
  color: var(--foreground-muted);
}

.main-content {
  flex: 1;
  padding: 20px 30px 30px;
  overflow-y: auto;
  background: transparent;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.fade-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.fade-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>
