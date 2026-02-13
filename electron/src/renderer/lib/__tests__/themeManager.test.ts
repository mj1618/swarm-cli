import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  getTheme,
  setTheme,
  getEffectiveTheme,
  applyTheme,
  onThemeChange,
  initThemeManager,
} from '../themeManager'

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: vi.fn<(key: string) => string | null>((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key]
    }),
    clear: vi.fn(() => {
      store = {}
    }),
  }
})()

// Mock matchMedia
const createMatchMediaMock = (matches: boolean) => {
  const listeners: Array<(e: MediaQueryListEvent) => void> = []
  return vi.fn().mockImplementation((query: string) => ({
    matches,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn((event: string, cb: (e: MediaQueryListEvent) => void) => {
      if (event === 'change') listeners.push(cb)
    }),
    removeEventListener: vi.fn((event: string, cb: (e: MediaQueryListEvent) => void) => {
      if (event === 'change') {
        const idx = listeners.indexOf(cb)
        if (idx > -1) listeners.splice(idx, 1)
      }
    }),
    dispatchEvent: vi.fn(),
    // Expose listeners for testing
    _listeners: listeners,
    _triggerChange: (newMatches: boolean) => {
      listeners.forEach((cb) => cb({ matches: newMatches } as MediaQueryListEvent))
    },
  }))
}

// Mock document.documentElement.classList
const classListMock = {
  add: vi.fn(),
  remove: vi.fn(),
  contains: vi.fn(),
  toggle: vi.fn(),
}

describe('themeManager', () => {
  beforeEach(() => {
    // Reset localStorage mock
    localStorageMock.clear()
    localStorageMock.getItem.mockClear()
    localStorageMock.setItem.mockClear()

    // Setup global mocks
    Object.defineProperty(global, 'localStorage', {
      value: localStorageMock,
      writable: true,
    })

    // Mock document.documentElement.classList
    Object.defineProperty(document, 'documentElement', {
      value: { classList: classListMock },
      writable: true,
    })
    classListMock.add.mockClear()
    classListMock.remove.mockClear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('getTheme', () => {
    it('returns "system" when localStorage is empty', () => {
      localStorageMock.getItem.mockReturnValue(null)

      const result = getTheme()

      expect(result).toBe('system')
      expect(localStorageMock.getItem).toHaveBeenCalledWith('swarm-theme')
    })

    it('returns "dark" when localStorage has "dark"', () => {
      localStorageMock.getItem.mockReturnValue('dark')

      const result = getTheme()

      expect(result).toBe('dark')
    })

    it('returns "light" when localStorage has "light"', () => {
      localStorageMock.getItem.mockReturnValue('light')

      const result = getTheme()

      expect(result).toBe('light')
    })

    it('returns "system" when localStorage has "system"', () => {
      localStorageMock.getItem.mockReturnValue('system')

      const result = getTheme()

      expect(result).toBe('system')
    })

    it('returns "system" for invalid stored values', () => {
      localStorageMock.getItem.mockReturnValue('invalid-theme')

      const result = getTheme()

      expect(result).toBe('system')
    })

    it('returns "system" for empty string', () => {
      localStorageMock.getItem.mockReturnValue('')

      const result = getTheme()

      expect(result).toBe('system')
    })

    it('returns "system" for numeric value', () => {
      localStorageMock.getItem.mockReturnValue('123')

      const result = getTheme()

      expect(result).toBe('system')
    })

    it('returns "system" for null-like string', () => {
      localStorageMock.getItem.mockReturnValue('null')

      const result = getTheme()

      expect(result).toBe('system')
    })
  })

  describe('setTheme', () => {
    beforeEach(() => {
      // Mock matchMedia for applyTheme calls within setTheme
      Object.defineProperty(window, 'matchMedia', {
        value: createMatchMediaMock(true),
        writable: true,
      })
    })

    it('persists "dark" to localStorage', () => {
      setTheme('dark')

      expect(localStorageMock.setItem).toHaveBeenCalledWith('swarm-theme', 'dark')
    })

    it('persists "light" to localStorage', () => {
      setTheme('light')

      expect(localStorageMock.setItem).toHaveBeenCalledWith('swarm-theme', 'light')
    })

    it('persists "system" to localStorage', () => {
      setTheme('system')

      expect(localStorageMock.setItem).toHaveBeenCalledWith('swarm-theme', 'system')
    })

    it('calls applyTheme after setting', () => {
      setTheme('dark')

      expect(classListMock.add).toHaveBeenCalledWith('dark')
      expect(classListMock.remove).toHaveBeenCalledWith('light')
    })

    it('notifies listeners when theme changes', () => {
      const listener = vi.fn()
      onThemeChange(listener)

      setTheme('light')

      expect(listener).toHaveBeenCalled()
    })
  })

  describe('getEffectiveTheme', () => {
    it('resolves "system" to "dark" when system prefers dark', () => {
      localStorageMock.getItem.mockReturnValue('system')
      Object.defineProperty(window, 'matchMedia', {
        value: createMatchMediaMock(true), // prefers dark
        writable: true,
      })

      const result = getEffectiveTheme()

      expect(result).toBe('dark')
    })

    it('resolves "system" to "light" when system prefers light', () => {
      localStorageMock.getItem.mockReturnValue('system')
      Object.defineProperty(window, 'matchMedia', {
        value: createMatchMediaMock(false), // prefers light
        writable: true,
      })

      const result = getEffectiveTheme()

      expect(result).toBe('light')
    })

    it('returns "dark" directly when preference is "dark"', () => {
      localStorageMock.getItem.mockReturnValue('dark')
      Object.defineProperty(window, 'matchMedia', {
        value: createMatchMediaMock(false), // system prefers light, but user chose dark
        writable: true,
      })

      const result = getEffectiveTheme()

      expect(result).toBe('dark')
    })

    it('returns "light" directly when preference is "light"', () => {
      localStorageMock.getItem.mockReturnValue('light')
      Object.defineProperty(window, 'matchMedia', {
        value: createMatchMediaMock(true), // system prefers dark, but user chose light
        writable: true,
      })

      const result = getEffectiveTheme()

      expect(result).toBe('light')
    })

    it('defaults to "dark" when matchMedia is unavailable and preference is "system"', () => {
      localStorageMock.getItem.mockReturnValue('system')
      Object.defineProperty(window, 'matchMedia', {
        value: undefined,
        writable: true,
      })

      const result = getEffectiveTheme()

      expect(result).toBe('dark')
    })
  })

  describe('applyTheme', () => {
    beforeEach(() => {
      Object.defineProperty(window, 'matchMedia', {
        value: createMatchMediaMock(true),
        writable: true,
      })
    })

    it('adds "dark" class and removes "light" for dark theme', () => {
      applyTheme('dark')

      expect(classListMock.add).toHaveBeenCalledWith('dark')
      expect(classListMock.remove).toHaveBeenCalledWith('light')
    })

    it('adds "light" class and removes "dark" for light theme', () => {
      applyTheme('light')

      expect(classListMock.add).toHaveBeenCalledWith('light')
      expect(classListMock.remove).toHaveBeenCalledWith('dark')
    })

    it('resolves "system" to effective theme based on matchMedia', () => {
      // matchMedia returns dark preference
      Object.defineProperty(window, 'matchMedia', {
        value: createMatchMediaMock(true),
        writable: true,
      })

      applyTheme('system')

      expect(classListMock.add).toHaveBeenCalledWith('dark')
      expect(classListMock.remove).toHaveBeenCalledWith('light')
    })

    it('uses stored preference when no argument provided', () => {
      localStorageMock.getItem.mockReturnValue('light')

      applyTheme()

      expect(classListMock.add).toHaveBeenCalledWith('light')
      expect(classListMock.remove).toHaveBeenCalledWith('dark')
    })
  })

  describe('onThemeChange', () => {
    beforeEach(() => {
      Object.defineProperty(window, 'matchMedia', {
        value: createMatchMediaMock(true),
        writable: true,
      })
    })

    it('returns an unsubscribe function', () => {
      const listener = vi.fn()

      const unsubscribe = onThemeChange(listener)

      expect(typeof unsubscribe).toBe('function')
    })

    it('listener is called when theme changes via setTheme', () => {
      const listener = vi.fn()
      onThemeChange(listener)

      setTheme('dark')

      expect(listener).toHaveBeenCalled()
    })

    it('listener receives the effective theme', () => {
      localStorageMock.getItem.mockReturnValue('dark')
      const listener = vi.fn()
      onThemeChange(listener)

      setTheme('dark')

      expect(listener).toHaveBeenCalledWith('dark')
    })

    it('listener is not called after unsubscribing', () => {
      const listener = vi.fn()
      const unsubscribe = onThemeChange(listener)

      unsubscribe()
      setTheme('dark')

      expect(listener).not.toHaveBeenCalled()
    })

    it('multiple listeners are all notified', () => {
      const listener1 = vi.fn()
      const listener2 = vi.fn()
      onThemeChange(listener1)
      onThemeChange(listener2)

      setTheme('light')

      expect(listener1).toHaveBeenCalled()
      expect(listener2).toHaveBeenCalled()
    })

    it('only unsubscribed listener is removed, others remain', () => {
      const listener1 = vi.fn()
      const listener2 = vi.fn()
      const unsubscribe1 = onThemeChange(listener1)
      onThemeChange(listener2)

      unsubscribe1()
      setTheme('dark')

      expect(listener1).not.toHaveBeenCalled()
      expect(listener2).toHaveBeenCalled()
    })
  })

  describe('initThemeManager', () => {
    it('applies initial theme on init', () => {
      localStorageMock.getItem.mockReturnValue('dark')
      Object.defineProperty(window, 'matchMedia', {
        value: createMatchMediaMock(true),
        writable: true,
      })

      initThemeManager()

      expect(classListMock.add).toHaveBeenCalledWith('dark')
    })

    it('returns a cleanup function', () => {
      Object.defineProperty(window, 'matchMedia', {
        value: createMatchMediaMock(true),
        writable: true,
      })

      const cleanup = initThemeManager()

      expect(typeof cleanup).toBe('function')
    })

    it('listens for system preference changes', () => {
      const addEventListenerSpy = vi.fn()
      const matchMediaMock = vi.fn().mockReturnValue({
        matches: true,
        media: '(prefers-color-scheme: dark)',
        addEventListener: addEventListenerSpy,
        removeEventListener: vi.fn(),
      })
      Object.defineProperty(window, 'matchMedia', {
        value: matchMediaMock,
        writable: true,
      })

      initThemeManager()

      expect(addEventListenerSpy).toHaveBeenCalledWith('change', expect.any(Function))
    })

    it('cleanup removes system preference listener', () => {
      const addEventListenerSpy = vi.fn()
      const removeEventListenerSpy = vi.fn()
      const matchMediaMock = vi.fn().mockReturnValue({
        matches: true,
        media: '(prefers-color-scheme: dark)',
        addEventListener: addEventListenerSpy,
        removeEventListener: removeEventListenerSpy,
      })
      Object.defineProperty(window, 'matchMedia', {
        value: matchMediaMock,
        writable: true,
      })

      const cleanup = initThemeManager()
      cleanup()

      expect(removeEventListenerSpy).toHaveBeenCalledWith('change', expect.any(Function))
    })
  })
})
