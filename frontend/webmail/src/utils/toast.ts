type ToastType = 'success' | 'error' | 'warning' | 'info'

export function showToast(type: ToastType, message: string) {
  if (window.$toast) {
    window.$toast[type](message)
  } else {
    console.warn(`[Toast] ${type}: ${message}`)
  }
}