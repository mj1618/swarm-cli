/**
 * Theme Manager
 * 
 * Manages theme state (system, dark, light) with localStorage persistence
 * and system preference detection via prefers-color-scheme.
 */

export type ThemePreference = 'system' | 'dark' | 'light'
export type EffectiveTheme = 'dark' | 'light'

const STORAGE_KEY = 'swarm-theme'

/** Get the stored theme preference */
export function getTheme(): ThemePreference {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'dark' || stored === 'light' || stored === 'system') {
    return stored
  }
  return 'system'
}

/** Set and persist the theme preference */
export function setTheme(theme: ThemePreference): void {
  localStorage.setItem(STORAGE_KEY, theme)
  applyTheme(theme)
  notifyListeners(getEffectiveTheme())
}

/** Get the system's preferred color scheme */
function getSystemTheme(): EffectiveTheme {
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  return 'dark' // Default to dark if matchMedia is unavailable
}

/** Resolve 'system' to the actual theme based on OS preference */
export function getEffectiveTheme(): EffectiveTheme {
  const preference = getTheme()
  if (preference === 'system') {
    return getSystemTheme()
  }
  return preference
}

/** Apply the theme class to the document root */
export function applyTheme(preference?: ThemePreference): void {
  const theme = preference ?? getTheme()
  const effective = theme === 'system' ? getSystemTheme() : theme
  
  if (effective === 'dark') {
    document.documentElement.classList.add('dark')
    document.documentElement.classList.remove('light')
  } else {
    document.documentElement.classList.add('light')
    document.documentElement.classList.remove('dark')
  }
}

// Listener management
type ThemeChangeListener = (theme: EffectiveTheme) => void
const listeners = new Set<ThemeChangeListener>()

/** Subscribe to theme changes */
export function onThemeChange(callback: ThemeChangeListener): () => void {
  listeners.add(callback)
  return () => {
    listeners.delete(callback)
  }
}

function notifyListeners(theme: EffectiveTheme): void {
  listeners.forEach(cb => cb(theme))
}

/** Initialize theme manager - call once at app startup */
export function initThemeManager(): () => void {
  // Apply initial theme
  applyTheme()
  
  // Listen for system preference changes
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  
  const handleSystemChange = () => {
    const preference = getTheme()
    if (preference === 'system') {
      applyTheme('system')
      notifyListeners(getEffectiveTheme())
    }
  }
  
  mediaQuery.addEventListener('change', handleSystemChange)
  
  // Return cleanup function
  return () => {
    mediaQuery.removeEventListener('change', handleSystemChange)
  }
}
