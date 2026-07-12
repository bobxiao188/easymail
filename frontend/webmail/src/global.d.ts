// Global type declarations
declare global {
  interface Window {
    $toast?: {
      success: (message: string, duration?: number) => void
      error: (message: string, duration?: number) => void
      warning: (message: string, duration?: number) => void
      info: (message: string, duration?: number) => void
    }
  }
}

export {}
