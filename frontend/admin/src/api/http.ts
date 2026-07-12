import axios, { type AxiosInstance, type InternalAxiosRequestConfig, type AxiosResponse } from 'axios'
import i18n from '../i18n'

export type UnauthorizedHandler = () => void | Promise<void>

let unauthorizedHandler: UnauthorizedHandler | null = null

export function setUnauthorizedHandler(handler: UnauthorizedHandler) {
  unauthorizedHandler = handler
}

const api: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 60000,
  headers: {
    'Content-Type': 'application/json'
  }
})

api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = sessionStorage.getItem('token')
    if (token) {
      config.headers.set('Authorization', `Bearer ${token}`)
    }
    // Send Accept-Language based on current UI locale so the backend returns localized messages
    const locale = i18n.global.locale.value
    config.headers.set('Accept-Language', locale === 'zh' ? 'zh-CN' : 'en')
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

api.interceptors.response.use(
  (response: AxiosResponse) => {
    return response.data
  },
  (error) => {
    if (error.response?.status === 401) {
      sessionStorage.removeItem('token')
      const run = unauthorizedHandler
      if (run) {
        void Promise.resolve(run()).catch(() => {})
      }
    }
    return Promise.reject(error)
  }
)

export default api
