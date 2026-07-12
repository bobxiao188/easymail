<template>
  <div class="login-container">
    <!-- 粒子背景 -->
    <canvas ref="particleCanvas" class="particle-canvas"></canvas>
    
    <!-- 自定义背景装饰 -->
    <div class="easy-decoration">
      <div class="easy-circle easy-circle-1"></div>
      <div class="easy-circle easy-circle-2"></div>
      <div class="easy-circle easy-circle-3"></div>
    </div>
    
    <!-- 毛玻璃登录卡片 -->
    <el-card class="login-card glass-card">
      <template #header>
        <div class="login-header">
          <div class="logo-container">
              <img :src="logoSrc" alt="EasyMail Admin" class="logo-icon-image">
          </div>
          <p class="subtitle">{{ t('login.title') }}</p>
        </div>
      </template>
      
      <el-form :model="loginForm" :rules="rules" ref="loginFormRef" label-width="0" class="login-form">
        <el-form-item prop="username">
            <el-input
              v-model="loginForm.username"
              :placeholder="t('login.usernamePlaceholder')"
              class="easy-input"
            >
              <template #prefix>
                <el-icon size="20"><UserFilled /></el-icon>
              </template>
            </el-input>
        </el-form-item>

        <el-form-item prop="password">
            <el-input
              v-model="loginForm.password"
              type="password"
              :placeholder="t('login.passwordPlaceholder')"
              class="easy-input"
              @keyup.enter="handleLogin"
              show-password
            >
              <template #prefix>
                <el-icon size="20"><Lock /></el-icon>
              </template>
            </el-input>
        </el-form-item>
        
        <el-form-item>
          <div class="login-button-container">
            <button type="button" class="easy-button login-button" @click="handleLogin" :loading="loading">
              <span v-if="!loading">{{ t('login.login') }}</span>
              <span v-else>{{ t('login.loggingIn') }}</span>
            </button>
          </div>
        </el-form-item>
      </el-form>
      
      <!-- 底部装饰 -->
      <div class="login-footer">
        <div class="easy-line"></div>
        <p class="footer-text">{{ t('login.footerText') }}</p>
        <a
          href="https://github.com/bobxiao188/easymail"
          target="_blank"
          rel="noopener noreferrer"
          class="github-link"
        >
          <svg viewBox="0 0 24 24" class="github-icon" fill="currentColor">
            <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
          </svg>
          <span>github.com/bobxiao188/easymail</span>
        </a>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores'
import { ElMessage, type FormInstance } from 'element-plus'
import { Lock, UserFilled } from '@element-plus/icons-vue'
import { getCookie } from '../utils/cookies'

const router = useRouter()
const authStore = useAuthStore()
const loginFormRef = ref<FormInstance | null>(null)
const loading = ref(false)
const { t } = useI18n()
const particleCanvas = ref<HTMLCanvasElement | null>(null)

const logoSrc = computed(() =>
  document.documentElement.classList.contains('skin-dark') ? '/logo-dark.png' : '/logo-light.png'
)

let animationId: number | null = null
let particles: Particle[] = []

interface Particle {
  x: number
  y: number
  vx: number
  vy: number
  size: number
  opacity: number
  color: string
}

const loginForm = reactive({
  username: '',
  password: '',
  language: ''
})

const rules = {
  username: [
    { required: true, message: t('common.required', { field: t('login.username') }), trigger: 'blur' }
  ],
  password: [
    { required: true, message: t('common.required', { field: t('login.password') }), trigger: 'blur' }
  ]
}

const initParticles = () => {
  const canvas = particleCanvas.value
  if (!canvas) return
  
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  
  canvas.width = window.innerWidth
  canvas.height = window.innerHeight
  
  particles = []
  const particleCount = Math.floor((canvas.width * canvas.height) / 8000)
  
  const isDark = document.documentElement.classList.contains('skin-dark')
  const root = document.documentElement
  const accentColor = getComputedStyle(root).getPropertyValue('--accent').trim() || '#5e6ad2'
  const particleColor = isDark ? '#5e6ad2' : accentColor
  
  for (let i = 0; i < particleCount; i++) {
    particles.push({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      vx: (Math.random() - 0.5) * 0.5,
      vy: (Math.random() - 0.5) * 0.5,
      size: Math.random() * 2 + 0.5,
      opacity: Math.random() * 0.5 + 0.2,
      color: particleColor
    })
  }
}

const animateParticles = () => {
  const canvas = particleCanvas.value
  if (!canvas) return
  
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  
  ctx.clearRect(0, 0, canvas.width, canvas.height)
  
  particles.forEach((particle, index) => {
    particle.x += particle.vx
    particle.y += particle.vy
    
    if (particle.x < 0 || particle.x > canvas.width) particle.vx *= -1
    if (particle.y < 0 || particle.y > canvas.height) particle.vy *= -1
    
    ctx.beginPath()
    ctx.arc(particle.x, particle.y, particle.size, 0, Math.PI * 2)
    ctx.fillStyle = particle.color
    ctx.globalAlpha = particle.opacity
    ctx.fill()
    
    particles.slice(index + 1).forEach(otherParticle => {
      const dx = particle.x - otherParticle.x
      const dy = particle.y - otherParticle.y
      const distance = Math.sqrt(dx * dx + dy * dy)
      
      if (distance < 120) {
        ctx.beginPath()
        ctx.moveTo(particle.x, particle.y)
        ctx.lineTo(otherParticle.x, otherParticle.y)
        ctx.strokeStyle = particle.color
        ctx.globalAlpha = (120 - distance) / 120 * 0.2
        ctx.lineWidth = 0.5
        ctx.stroke()
      }
    })
  })
  
  ctx.globalAlpha = 1
  animationId = requestAnimationFrame(animateParticles)
}

const handleResize = () => {
  initParticles()
}

onMounted(() => {
  const skin = getCookie('skin') || 'dark'
  document.documentElement.classList.remove('skin-dark', 'skin-light')
  document.documentElement.classList.add(`skin-${skin}`)
  
  initParticles()
  animateParticles()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  if (animationId) {
    cancelAnimationFrame(animationId)
  }
  window.removeEventListener('resize', handleResize)
})

const handleLogin = async () => {
  if (!loginFormRef.value) return

  loginForm.language = getCookie('language') || 'en'

  try {
    const valid = await loginFormRef.value.validate()
    if (!valid) return

    loading.value = true
    await authStore.login(loginForm)
    ElMessage.success(t('login.loginSuccess'))
    await router.push({ name: 'Domains' })
  } catch (error: any) {
    let errorMessage = t('login.loginFailed')
    if (error.response?.data?.message) {
      errorMessage = error.response.data.message
    } else if (error.message) {
      errorMessage = error.message
    }
    ElMessage.error(errorMessage)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  position: relative;
  overflow: hidden;
  background: var(--background-base);
}

/* 粒子画布 */
.particle-canvas {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 0;
}

/* 自定义背景装饰 */
.easy-decoration {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 0;
}

.easy-circle {
  position: absolute;
  border-radius: 50%;
  background: radial-gradient(circle, var(--accent-15) 0%, transparent 70%);
  animation: float 6s ease-in-out infinite;
}

.easy-circle-1 {
  width: 400px;
  height: 400px;
  top: -100px;
  left: -100px;
  animation-delay: 0s;
}

.easy-circle-2 {
  width: 300px;
  height: 300px;
  bottom: -50px;
  right: -50px;
  animation-delay: 2s;
  background: radial-gradient(circle, var(--accent-15) 0%, transparent 70%);
}

.easy-circle-3 {
  width: 200px;
  height: 200px;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  animation-delay: 4s;
  background: radial-gradient(circle, var(--accent-10) 0%, transparent 70%);
}

@keyframes float {
  0%, 100% {
    transform: translateY(0) scale(1);
    opacity: 0.5;
  }
  50% {
    transform: translateY(-20px) scale(1.1);
    opacity: 0.8;
  }
}

.logo-icon-image {
  height: 42px;
  object-fit: contain;
  display: flex;
  align-items: center;
}

/* 登录卡片 */
.login-card {
  width: 420px;
  position: relative;
  z-index: 1;
  border: 1px solid var(--border-default);
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25), 0 0 0 1px rgba(0, 0, 0, 0.05);
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.1) 0%, rgba(255, 255, 255, 0.05) 100%);
  backdrop-filter: blur(30px);
  -webkit-backdrop-filter: blur(30px);
}

.login-card :deep(.el-card__header) {
  background: transparent;
  border-bottom: 1px solid var(--border-default);
  padding: 30px 30px 20px;
}

.login-card :deep(.el-card__body) {
  padding: 20px 30px 30px;
}

/* 登录头部 */
.login-header {
  text-align: center;
}

.logo-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 10px;
}

.logo-icon {
  width: 60px;
  height: 60px;
  background: var(--accent-10);
  border: 2px solid var(--accent-30);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 15px;
  box-shadow: 0 0 30px var(--accent-glow);
}

.subtitle {
  margin: 8px 0 0;
  color: var(--foreground-muted);
  font-size: 20px;
  letter-spacing: 0.5px;
}

/* 登录表单 */
.login-form {
  margin-top: 20px;
}

.input-icon svg {
  width: 20px;
  height: 20px;
}

.login-button-container {
  width: 100%;
  display: flex;
  justify-content: center;
}

/* 登录按钮 */
.login-button {
  width: 120px;
  height: 44px;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0.5px;
  border-radius: 8px;
  margin-top: 10px;
  position: relative;
  overflow: hidden;
}

.login-button::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, var(--accent-20), transparent);
  transition: left 0.5s ease;
}

.login-button:hover::before {
  left: 100%;
}

/* 底部装饰 */
.login-footer {
  margin-top: 25px;
  text-align: center;
}

.easy-line {
  width: 60px;
  height: 2px;
  background: linear-gradient(90deg, transparent, var(--accent), transparent);
  margin: 0 auto 15px;
}

.footer-text {
  color: var(--foreground-muted);
  font-size: 12px;
  letter-spacing: 2px;
  margin: 0 0 12px;
}

.github-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--foreground-muted);
  font-size: 12px;
  text-decoration: none;
  transition: color 0.2s;
}

.github-link:hover {
  color: var(--accent);
}

.github-icon {
  width: 16px;
  height: 16px;
}

/* 表单项间距 */
.login-form :deep(.el-form-item) {
  margin-bottom: 20px;
}

.login-form :deep(.el-form-item:last-child) {
  margin-bottom: 0;
}
</style>