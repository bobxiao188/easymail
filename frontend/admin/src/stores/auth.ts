import { defineStore } from 'pinia'
import { authApi } from '../api/auth'
import { getCookie, setCookie } from '../utils/cookies'

interface AuthState {
  token: string | null
  userInfo: {
    id: number
    username: string
    nickname: string
    email: string
    avatar?: string
    language: string
    skin: string
    isAdmin: boolean
  } | null
}

export const useAuthStore = defineStore('user', {
  state: (): AuthState => ({
    token: sessionStorage.getItem('token') || null,
    userInfo: null
  }),

  getters: {
    isLoggedIn: (state) => !!state.token,
    language: (state) => state.userInfo?.language || getCookie('language') || 'en',
    skin: (state) => state.userInfo?.skin || getCookie('skin') || 'dark'
  },

  actions: {
    async login(loginForm: { username: string; password: string }) {
      const res: any = await authApi.login(loginForm)
      if (res.code === 0) {
        this.token = res.data.token
        this.userInfo = res.data.user
        sessionStorage.setItem('token', res.data.token)
        const lang = res.data.user?.language || 'en'
        const skin = res.data.user?.skin || 'dark'
        setCookie('language', lang, 30)
        setCookie('skin', skin, 30)
        return res.data
      }
      throw new Error(res.message || 'Login failed')
    },

    async getProfile() {
      const res: any = await authApi.getProfile()
      if (res.code === 0) {
        this.userInfo = res.data
        const lang = res.data?.language || getCookie('language') || 'en'
        const skin = res.data?.skin || getCookie('skin') || 'dark'
        setCookie('language', lang, 30)
        setCookie('skin', skin, 30)
      }
      return res
    },

    async updateLanguage(language: string) {
      const res: any = await authApi.updateLanguage(language)
      if (res.code === 0) {
        if (this.userInfo) {
          this.userInfo.language = language
        }
        setCookie('language', language, 30)
      }
      return res
    },

    async updateSkin(skin: string) {
      const res: any = await authApi.updateSkin(skin)
      if (res.code === 0) {
        if (this.userInfo) {
          this.userInfo.skin = skin
        }
        setCookie('skin', skin, 30)
      }
      return res
    },

    logout() {
      this.token = null
      this.userInfo = null
      sessionStorage.removeItem('token')
    }
  }
})