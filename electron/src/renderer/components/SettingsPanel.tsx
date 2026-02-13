import { useState, useEffect, useCallback } from 'react'
import { isSoundEnabled, setSoundEnabled, getSoundVolume, setSoundVolume, playSuccess } from '../lib/soundManager'
import { getTheme, setTheme } from '../lib/themeManager'
import type { ThemePreference } from '../lib/themeManager'

interface SettingsPanelProps {
  onClose: () => void
  onToast: (type: 'success' | 'error', message: string) => void
}

const BACKENDS = ['claude-code', 'cursor']
const MODELS = ['opus', 'sonnet', 'haiku']

export default function SettingsPanel({ onClose, onToast }: SettingsPanelProps) {
  const [backend, setBackend] = useState('')
  const [model, setModel] = useState('')
  const [statePath, setStatePath] = useState('')
  const [logsDir, setLogsDir] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [originalBackend, setOriginalBackend] = useState('')
  const [originalModel, setOriginalModel] = useState('')
  const [systemNotifications, setSystemNotifications] = useState(() =>
    localStorage.getItem('swarm-system-notifications') !== 'false'
  )
  const [soundAlerts, setSoundAlerts] = useState(isSoundEnabled)
  const [soundVolume, setSoundVolumeState] = useState(getSoundVolume)
  const [themePreference, setThemePreference] = useState<ThemePreference>(getTheme)

  useEffect(() => {
    window.settings.read().then(result => {
      if (result.error) {
        onToast('error', `Failed to load settings: ${result.error}`)
        setLoading(false)
        return
      }
      setBackend(result.config.backend)
      setModel(result.config.model)
      setStatePath(result.config.statePath)
      setLogsDir(result.config.logsDir)
      setOriginalBackend(result.config.backend)
      setOriginalModel(result.config.model)
      setLoading(false)
    })
  }, [onToast])

  useEffect(() => {
    setDirty(backend !== originalBackend || model !== originalModel)
  }, [backend, model, originalBackend, originalModel])

  // Close on Escape
  useEffect(() => {
    function handleEscape(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [onClose])

  const handleSave = useCallback(async () => {
    setSaving(true)
    const updates: { backend?: string; model?: string } = {}
    if (backend !== originalBackend) updates.backend = backend
    if (model !== originalModel) updates.model = model

    const result = await window.settings.write(updates)
    setSaving(false)

    if (result.error) {
      onToast('error', `Failed to save settings: ${result.error}`)
    } else {
      setOriginalBackend(backend)
      setOriginalModel(model)
      setDirty(false)
      onToast('success', 'Settings saved')
    }
  }, [backend, model, originalBackend, originalModel, onToast])

  const labelClass = 'text-xs font-semibold text-muted-foreground mb-1.5 block'
  const inputClass = 'w-full bg-background border border-border rounded px-2 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-primary'

  return (
    <div className="flex-1 flex flex-col min-w-0">
      {/* Header */}
      <div className="p-3 border-b border-border flex items-center justify-between">
        <h2 className="text-sm font-semibold text-foreground">Settings</h2>
        <button
          onClick={onClose}
          className="text-muted-foreground hover:text-foreground transition-colors text-lg leading-none px-1"
          aria-label="Close settings"
        >
          &times;
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6">
        {loading ? (
          <p className="text-sm text-muted-foreground">Loading settings...</p>
        ) : (
          <div className="max-w-md mx-auto space-y-6">
            {/* Appearance */}
            <div>
              <label className={labelClass}>Appearance</label>
              <div className="flex gap-2">
                {(['system', 'dark', 'light'] as const).map(t => (
                  <button
                    key={t}
                    onClick={() => {
                      setThemePreference(t)
                      setTheme(t)
                    }}
                    className={`flex-1 px-3 py-2 text-sm rounded border transition-colors ${
                      themePreference === t
                        ? 'bg-primary/20 border-primary text-primary'
                        : 'border-border text-muted-foreground hover:text-foreground hover:border-muted-foreground'
                    }`}
                  >
                    {t === 'system' ? 'System' : t === 'dark' ? 'Dark' : 'Light'}
                  </button>
                ))}
              </div>
              <p className="text-xs text-muted-foreground mt-1.5">
                System follows your OS preference
              </p>
            </div>

            {/* Backend */}
            <div>
              <label className={labelClass}>Backend</label>
              <div className="flex gap-3">
                {BACKENDS.map(b => (
                  <label
                    key={b}
                    className={`flex items-center gap-2 cursor-pointer text-sm px-3 py-2 rounded border transition-colors ${
                      backend === b
                        ? 'bg-primary/20 border-primary text-primary'
                        : 'border-border text-muted-foreground hover:text-foreground hover:border-muted-foreground'
                    }`}
                  >
                    <input
                      type="radio"
                      name="backend"
                      value={b}
                      checked={backend === b}
                      onChange={() => setBackend(b)}
                      className="sr-only"
                    />
                    <span className={`w-3 h-3 rounded-full border-2 flex items-center justify-center ${
                      backend === b ? 'border-primary' : 'border-muted-foreground'
                    }`}>
                      {backend === b && <span className="w-1.5 h-1.5 rounded-full bg-primary" />}
                    </span>
                    {b === 'claude-code' ? 'Claude Code' : 'Cursor'}
                  </label>
                ))}
              </div>
            </div>

            {/* Default Model */}
            <div>
              <label className={labelClass}>Default Model</label>
              <select
                value={model}
                onChange={e => setModel(e.target.value)}
                className={inputClass}
              >
                {MODELS.map(m => (
                  <option key={m} value={m}>{m}</option>
                ))}
              </select>
            </div>

            {/* State Path (read-only) */}
            <div>
              <label className={labelClass}>State Path</label>
              <div className="bg-background/50 border border-border rounded px-2 py-1.5 text-sm text-muted-foreground font-mono">
                {statePath}
              </div>
            </div>

            {/* Logs Directory (read-only) */}
            <div>
              <label className={labelClass}>Logs Directory</label>
              <div className="bg-background/50 border border-border rounded px-2 py-1.5 text-sm text-muted-foreground font-mono">
                {logsDir}
              </div>
            </div>

            {/* System Notifications */}
            <div>
              <label className={labelClass}>Notifications</label>
              <label className="flex items-center gap-3 cursor-pointer">
                <button
                  role="switch"
                  aria-checked={systemNotifications}
                  onClick={() => {
                    const next = !systemNotifications
                    setSystemNotifications(next)
                    localStorage.setItem('swarm-system-notifications', String(next))
                  }}
                  className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                    systemNotifications ? 'bg-primary' : 'bg-muted-foreground/30'
                  }`}
                >
                  <span
                    className={`inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform ${
                      systemNotifications ? 'translate-x-[18px]' : 'translate-x-[3px]'
                    }`}
                  />
                </button>
                <span className="text-sm text-foreground">
                  System notifications when agents complete or fail
                </span>
              </label>
              <p className="text-xs text-muted-foreground mt-1.5">
                Only fires when the window is not focused
              </p>
            </div>

            {/* Sound Alerts */}
            <div>
              <label className={labelClass}>Sound Alerts</label>
              <label className="flex items-center gap-3 cursor-pointer">
                <button
                  role="switch"
                  aria-checked={soundAlerts}
                  onClick={() => {
                    const next = !soundAlerts
                    setSoundAlerts(next)
                    setSoundEnabled(next)
                    if (next) playSuccess()
                  }}
                  className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                    soundAlerts ? 'bg-primary' : 'bg-muted-foreground/30'
                  }`}
                >
                  <span
                    className={`inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform ${
                      soundAlerts ? 'translate-x-[18px]' : 'translate-x-[3px]'
                    }`}
                  />
                </button>
                <span className="text-sm text-foreground">
                  Play sounds when agents complete or fail
                </span>
              </label>
              {soundAlerts && (
                <div className="mt-3 flex items-center gap-3">
                  <span className="text-xs text-muted-foreground w-12">Volume</span>
                  <input
                    type="range"
                    min={0}
                    max={100}
                    value={soundVolume}
                    onChange={e => {
                      const v = parseInt(e.target.value, 10)
                      setSoundVolumeState(v)
                      setSoundVolume(v)
                    }}
                    className="flex-1 h-1.5 accent-primary cursor-pointer"
                  />
                  <span className="text-xs text-muted-foreground w-8 text-right">{soundVolume}%</span>
                </div>
              )}
            </div>

            {/* Save Button */}
            <div className="pt-2">
              <button
                onClick={handleSave}
                disabled={saving || !dirty}
                className="text-xs px-4 py-1.5 bg-primary text-primary-foreground rounded hover:bg-primary/90 disabled:opacity-50 font-medium transition-colors"
              >
                {saving ? 'Saving...' : 'Save Changes'}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
