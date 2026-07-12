/**
 * Apply theme to the application
 * Supports: light (default), dark
 */

export function applyTheme(theme: string): void {
  const root = document.documentElement

  if (theme === 'dark') {
    root.classList.add('dark')
    root.style.colorScheme = 'dark'
  } else {
    root.classList.remove('dark')
    root.style.colorScheme = 'light'
  }
}