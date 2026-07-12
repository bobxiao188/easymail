import axios from 'axios'
import { getCookie } from '../utils/cookies'

// Create axios instance
const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Get current language and map to backend language tag
function getCurrentLanguage(): string {
  const savedLanguage = getCookie('language') || 'en'
  // Map frontend locale (zh/en) to backend language tag (zh-CN/en)
  return savedLanguage === 'zh' ? 'zh-CN' : 'en'
}

// Request interceptor
apiClient.interceptors.request.use(
  (config) => {
    // Get token from localStorage and add to request headers
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }

    // Add Accept-Language header so backend can return localized error messages
    config.headers['Accept-Language'] = getCurrentLanguage()

    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    // Unified error handling
    console.error('API Error:', error)
    
    // If 401 error received, clear token and redirect to login page
    // Exclude login endpoint to avoid redirect loop
    if (error.response?.status === 401 && !error.config?.url?.includes('/v1/auth/login')) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    
    // Extract backend error message from JSON response body
    const backendMessage = error.response?.data?.message
    if (backendMessage) {
      return Promise.reject(new Error(backendMessage))
    }
    
    return Promise.reject(error)
  }
)

export default apiClient
