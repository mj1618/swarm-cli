import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: vi.fn((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key]
    }),
    clear: vi.fn(() => {
      store = {}
    }),
    get length() {
      return Object.keys(store).length
    },
    key: vi.fn((index: number) => Object.keys(store)[index] ?? null),
  }
})()

// Mock AudioContext and related Web Audio API classes
function createMockOscillator() {
  return {
    type: 'sine' as OscillatorType,
    frequency: {
      setValueAtTime: vi.fn(),
    },
    connect: vi.fn(),
    start: vi.fn(),
    stop: vi.fn(),
  }
}

function createMockGainNode() {
  return {
    gain: {
      setValueAtTime: vi.fn(),
      exponentialRampToValueAtTime: vi.fn(),
    },
    connect: vi.fn(),
  }
}

// Track mock calls
const audioContextCalls = {
  createOscillator: vi.fn(),
  createGain: vi.fn(),
}

// Create a mock AudioContext constructor
class MockAudioContext {
  currentTime = 0
  destination = {}

  createOscillator() {
    audioContextCalls.createOscillator()
    return createMockOscillator()
  }

  createGain() {
    audioContextCalls.createGain()
    return createMockGainNode()
  }
}

// Setup global mocks before importing the module
vi.stubGlobal('localStorage', localStorageMock)
vi.stubGlobal('AudioContext', MockAudioContext)

describe('soundManager', () => {
  beforeEach(async () => {
    // Clear localStorage mock
    localStorageMock.clear()
    vi.clearAllMocks()
    
    // Reset audio context call tracking
    audioContextCalls.createOscillator.mockClear()
    audioContextCalls.createGain.mockClear()

    // Reset modules to clear the audioCtx singleton
    vi.resetModules()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  describe('isSoundEnabled', () => {
    it('returns true by default when localStorage is empty', async () => {
      const { isSoundEnabled } = await import('../soundManager')

      expect(isSoundEnabled()).toBe(true)
      expect(localStorageMock.getItem).toHaveBeenCalledWith('swarm-sound-alerts-enabled')
    })

    it('returns false when localStorage has "false"', async () => {
      localStorageMock.setItem('swarm-sound-alerts-enabled', 'false')
      const { isSoundEnabled } = await import('../soundManager')

      expect(isSoundEnabled()).toBe(false)
    })

    it('returns true when localStorage has "true"', async () => {
      localStorageMock.setItem('swarm-sound-alerts-enabled', 'true')
      const { isSoundEnabled } = await import('../soundManager')

      expect(isSoundEnabled()).toBe(true)
    })

    it('returns true for any other localStorage value', async () => {
      localStorageMock.setItem('swarm-sound-alerts-enabled', 'random-value')
      const { isSoundEnabled } = await import('../soundManager')

      expect(isSoundEnabled()).toBe(true)
    })

    it('returns true for empty string localStorage value', async () => {
      localStorageMock.setItem('swarm-sound-alerts-enabled', '')
      const { isSoundEnabled } = await import('../soundManager')

      expect(isSoundEnabled()).toBe(true)
    })
  })

  describe('setSoundEnabled', () => {
    it('persists "true" to localStorage when enabled', async () => {
      const { setSoundEnabled } = await import('../soundManager')

      setSoundEnabled(true)

      expect(localStorageMock.setItem).toHaveBeenCalledWith('swarm-sound-alerts-enabled', 'true')
    })

    it('persists "false" to localStorage when disabled', async () => {
      const { setSoundEnabled } = await import('../soundManager')

      setSoundEnabled(false)

      expect(localStorageMock.setItem).toHaveBeenCalledWith('swarm-sound-alerts-enabled', 'false')
    })
  })

  describe('getSoundVolume', () => {
    it('returns default 80 when localStorage is empty', async () => {
      const { getSoundVolume } = await import('../soundManager')

      expect(getSoundVolume()).toBe(80)
      expect(localStorageMock.getItem).toHaveBeenCalledWith('swarm-sound-alerts-volume')
    })

    it('returns stored value when valid (within 0-100)', async () => {
      localStorageMock.setItem('swarm-sound-alerts-volume', '50')
      const { getSoundVolume } = await import('../soundManager')

      expect(getSoundVolume()).toBe(50)
    })

    it('returns 0 when stored value is 0', async () => {
      localStorageMock.setItem('swarm-sound-alerts-volume', '0')
      const { getSoundVolume } = await import('../soundManager')

      expect(getSoundVolume()).toBe(0)
    })

    it('returns 100 when stored value is 100', async () => {
      localStorageMock.setItem('swarm-sound-alerts-volume', '100')
      const { getSoundVolume } = await import('../soundManager')

      expect(getSoundVolume()).toBe(100)
    })

    it('clamps values above 100 to 100', async () => {
      localStorageMock.setItem('swarm-sound-alerts-volume', '150')
      const { getSoundVolume } = await import('../soundManager')

      expect(getSoundVolume()).toBe(100)
    })

    it('clamps values below 0 to 0', async () => {
      localStorageMock.setItem('swarm-sound-alerts-volume', '-50')
      const { getSoundVolume } = await import('../soundManager')

      expect(getSoundVolume()).toBe(0)
    })

    it('returns 80 for invalid/NaN values', async () => {
      localStorageMock.setItem('swarm-sound-alerts-volume', 'not-a-number')
      const { getSoundVolume } = await import('../soundManager')

      expect(getSoundVolume()).toBe(80)
    })

    it('returns 80 for empty string value', async () => {
      localStorageMock.setItem('swarm-sound-alerts-volume', '')
      const { getSoundVolume } = await import('../soundManager')

      expect(getSoundVolume()).toBe(80)
    })

    it('parses float strings as integers', async () => {
      localStorageMock.setItem('swarm-sound-alerts-volume', '75.9')
      const { getSoundVolume } = await import('../soundManager')

      expect(getSoundVolume()).toBe(75)
    })
  })

  describe('setSoundVolume', () => {
    it('persists volume to localStorage', async () => {
      const { setSoundVolume } = await import('../soundManager')

      setSoundVolume(60)

      expect(localStorageMock.setItem).toHaveBeenCalledWith('swarm-sound-alerts-volume', '60')
    })

    it('clamps values above 100 before storing', async () => {
      const { setSoundVolume } = await import('../soundManager')

      setSoundVolume(200)

      expect(localStorageMock.setItem).toHaveBeenCalledWith('swarm-sound-alerts-volume', '100')
    })

    it('clamps values below 0 before storing', async () => {
      const { setSoundVolume } = await import('../soundManager')

      setSoundVolume(-30)

      expect(localStorageMock.setItem).toHaveBeenCalledWith('swarm-sound-alerts-volume', '0')
    })

    it('stores 0 correctly', async () => {
      const { setSoundVolume } = await import('../soundManager')

      setSoundVolume(0)

      expect(localStorageMock.setItem).toHaveBeenCalledWith('swarm-sound-alerts-volume', '0')
    })

    it('stores 100 correctly', async () => {
      const { setSoundVolume } = await import('../soundManager')

      setSoundVolume(100)

      expect(localStorageMock.setItem).toHaveBeenCalledWith('swarm-sound-alerts-volume', '100')
    })
  })

  describe('playSuccess', () => {
    it('does not play when sound is disabled', async () => {
      localStorageMock.setItem('swarm-sound-alerts-enabled', 'false')
      const { playSuccess } = await import('../soundManager')

      playSuccess()

      expect(audioContextCalls.createOscillator).not.toHaveBeenCalled()
      expect(audioContextCalls.createGain).not.toHaveBeenCalled()
    })

    it('uses AudioContext when sound is enabled', async () => {
      const { playSuccess } = await import('../soundManager')

      playSuccess()

      expect(audioContextCalls.createGain).toHaveBeenCalled()
      expect(audioContextCalls.createOscillator).toHaveBeenCalled()
    })

    it('creates two oscillators and two gain nodes per call', async () => {
      const { playSuccess } = await import('../soundManager')

      playSuccess()

      // The playTone function creates 2 oscillators and 2 gain nodes
      expect(audioContextCalls.createOscillator).toHaveBeenCalledTimes(2)
      expect(audioContextCalls.createGain).toHaveBeenCalledTimes(2)
    })
  })

  describe('playFailure', () => {
    it('does not play when sound is disabled', async () => {
      localStorageMock.setItem('swarm-sound-alerts-enabled', 'false')
      const { playFailure } = await import('../soundManager')

      playFailure()

      expect(audioContextCalls.createOscillator).not.toHaveBeenCalled()
      expect(audioContextCalls.createGain).not.toHaveBeenCalled()
    })

    it('uses AudioContext when sound is enabled', async () => {
      const { playFailure } = await import('../soundManager')

      playFailure()

      expect(audioContextCalls.createGain).toHaveBeenCalled()
      expect(audioContextCalls.createOscillator).toHaveBeenCalled()
    })

    it('creates two oscillators and two gain nodes per call', async () => {
      const { playFailure } = await import('../soundManager')

      playFailure()

      // The playTone function creates 2 oscillators and 2 gain nodes
      expect(audioContextCalls.createOscillator).toHaveBeenCalledTimes(2)
      expect(audioContextCalls.createGain).toHaveBeenCalledTimes(2)
    })
  })

  describe('playSuccess and playFailure together', () => {
    it('both functions use the AudioContext methods', async () => {
      const { playSuccess, playFailure } = await import('../soundManager')

      playSuccess()
      playFailure()

      // Both calls should have triggered AudioContext methods
      // Each call creates 2 oscillators and 2 gain nodes
      expect(audioContextCalls.createOscillator).toHaveBeenCalledTimes(4)
      expect(audioContextCalls.createGain).toHaveBeenCalledTimes(4)
    })
  })
})
