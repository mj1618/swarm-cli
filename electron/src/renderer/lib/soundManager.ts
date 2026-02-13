const STORAGE_KEY_ENABLED = 'swarm-sound-alerts-enabled'
const STORAGE_KEY_VOLUME = 'swarm-sound-alerts-volume'

let audioCtx: AudioContext | null = null

function getAudioContext(): AudioContext {
  if (!audioCtx) {
    audioCtx = new AudioContext()
  }
  return audioCtx
}

export function isSoundEnabled(): boolean {
  return localStorage.getItem(STORAGE_KEY_ENABLED) !== 'false'
}

export function setSoundEnabled(enabled: boolean): void {
  localStorage.setItem(STORAGE_KEY_ENABLED, String(enabled))
}

export function getSoundVolume(): number {
  const raw = localStorage.getItem(STORAGE_KEY_VOLUME)
  if (raw == null) return 80
  const v = parseInt(raw, 10)
  return isNaN(v) ? 80 : Math.max(0, Math.min(100, v))
}

export function setSoundVolume(volume: number): void {
  localStorage.setItem(STORAGE_KEY_VOLUME, String(Math.max(0, Math.min(100, volume))))
}

function playTone(freq1: number, freq2: number, ascending: boolean): void {
  if (!isSoundEnabled()) return

  const ctx = getAudioContext()
  const volume = getSoundVolume() / 100
  const now = ctx.currentTime

  const gain = ctx.createGain()
  gain.connect(ctx.destination)
  gain.gain.setValueAtTime(volume * 0.3, now)
  gain.gain.exponentialRampToValueAtTime(0.001, now + 0.25)

  const osc1 = ctx.createOscillator()
  osc1.type = 'sine'
  osc1.frequency.setValueAtTime(ascending ? freq1 : freq2, now)
  osc1.connect(gain)
  osc1.start(now)
  osc1.stop(now + 0.12)

  const gain2 = ctx.createGain()
  gain2.connect(ctx.destination)
  gain2.gain.setValueAtTime(0.001, now)
  gain2.gain.setValueAtTime(volume * 0.3, now + 0.1)
  gain2.gain.exponentialRampToValueAtTime(0.001, now + 0.3)

  const osc2 = ctx.createOscillator()
  osc2.type = 'sine'
  osc2.frequency.setValueAtTime(ascending ? freq2 : freq1, now + 0.1)
  osc2.connect(gain2)
  osc2.start(now + 0.1)
  osc2.stop(now + 0.25)
}

export function playSuccess(): void {
  playTone(523.25, 659.25, true) // C5 → E5 ascending
}

export function playFailure(): void {
  playTone(329.63, 261.63, false) // E4 → C4 descending
}
