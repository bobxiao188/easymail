<template>
  <div id="toast-container"></div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'

const TOAST_DURATION = 3000

function createToastElement(type: 'success' | 'error' | 'warning' | 'info', message: string) {
  const config: Record<string, { bg: string; icon: string }> = {
    success: { bg: 'bg-emerald-500', icon: '✓' },
    error: { bg: 'bg-red-500', icon: '✕' },
    warning: { bg: 'bg-amber-500', icon: '⚠' },
    info: { bg: 'bg-blue-500', icon: 'ℹ' },
  }

  const container = document.getElementById('toast-container')
  if (!container) return

  const existing = container.children.length
  const top = existing * 72

  const { bg, icon } = config[type]

  const el = document.createElement('div')
  el.className = `flex items-center gap-2.5 px-4 py-3 rounded-xl shadow-lg text-sm font-medium text-white min-w-[280px] max-w-[400px] animate-fade-in-up ${bg}`
  el.style.marginTop = existing > 0 ? `${top}px` : '0'
  el.innerHTML = `<span style="font-size:16px;font-weight:bold;flex-shrink:0">${icon}</span><span>${message}</span>`
  container.appendChild(el)

  setTimeout(() => {
    el.style.opacity = '0'
    el.style.transition = 'opacity 0.3s'
    setTimeout(() => { el.remove() }, 300)
  }, TOAST_DURATION)
}

// Initialize window.$toast when component is mounted
onMounted(() => {
  window.$toast = {
    success: (message: string) => createToastElement('success', message),
    error: (message: string) => createToastElement('error', message),
    warning: (message: string) => createToastElement('warning', message),
    info: (message: string) => createToastElement('info', message),
  }
})
</script>