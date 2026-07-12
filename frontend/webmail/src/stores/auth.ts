import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User } from '../types'
import { login, logout, getCurrentUser } from '../api/auth'

export const useAuthStore = defineStore('auth', () => {
  // State
  const user = ref<User | null>(null)
  const token = ref<string>(localStorage.getItem('token') || '')
  const isLoading = ref(false)

  // Computed properties
  const isLoggedIn = computed(() => !!token.value && !!user.value)
  const userEmail = computed(() => user.value?.email || '')
  const userName = computed(() => user.value?.name || '')

  // Methods
  async function loginAction(email: string, password: string) {
    isLoading.value = true
    try {
      const response = await login(email, password)
      token.value = response.data.token
      user.value = response.data.user
      localStorage.setItem('token', response.data.token)
      return response
    } catch (error) {
      throw error
    } finally {
      isLoading.value = false
    }
  }

  async function logoutAction() {
    try {
      await logout()
    } catch (error) {
      console.error('Logout error:', error)
    } finally {
      user.value = null
      token.value = ''
      localStorage.removeItem('token')
    }
  }

  async function fetchUser() {
    if (!token.value) return
    
    try {
      const response = await getCurrentUser()
      user.value = response.data
    } catch (error) {
      // Clear invalid token
      token.value = ''
      localStorage.removeItem('token')
    }
  }

  function clear() {
    user.value = null
    token.value = ''
    localStorage.removeItem('token')
  }

  return {
    // State
    user,
    token,
    isLoading,
    // Computed
    isLoggedIn,
    userEmail,
    userName,
    // Methods
    loginAction,
    logoutAction,
    fetchUser,
    clear
  }
})
