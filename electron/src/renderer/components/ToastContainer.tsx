import { useState, useEffect, useCallback, useRef } from 'react'

export type ToastType = 'success' | 'error' | 'warning' | 'info'

export interface Toast {
  id: string
  type: ToastType
  message: string
  timestamp: number
}

const TOAST_DURATION = 5000
const MAX_TOASTS = 5

const icons: Record<ToastType, string> = {
  success: '✓',
  error: '✕',
  warning: '⏹',
  info: '▶',
}

const colorClasses: Record<ToastType, string> = {
  success: 'border-green-500/50 bg-green-500/10 text-green-400',
  error: 'border-red-500/50 bg-red-500/10 text-red-400',
  warning: 'border-yellow-500/50 bg-yellow-500/10 text-yellow-400',
  info: 'border-blue-500/50 bg-blue-500/10 text-blue-400',
}

const iconColorClasses: Record<ToastType, string> = {
  success: 'text-green-400',
  error: 'text-red-400',
  warning: 'text-yellow-400',
  info: 'text-blue-400',
}

export function useToasts() {
  const [toasts, setToasts] = useState<Toast[]>([])
  const timersRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map())

  const removeToast = useCallback((id: string) => {
    const timer = timersRef.current.get(id)
    if (timer) {
      clearTimeout(timer)
      timersRef.current.delete(id)
    }
    setToasts(prev => prev.filter(t => t.id !== id))
  }, [])

  const addToast = useCallback((type: ToastType, message: string) => {
    const id = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`
    const toast: Toast = { id, type, message, timestamp: Date.now() }

    setToasts(prev => {
      const next = [...prev, toast]
      // Trim oldest if over max
      while (next.length > MAX_TOASTS) {
        const removed = next.shift()!
        const timer = timersRef.current.get(removed.id)
        if (timer) {
          clearTimeout(timer)
          timersRef.current.delete(removed.id)
        }
      }
      return next
    })

    const timer = setTimeout(() => {
      timersRef.current.delete(id)
      setToasts(prev => prev.filter(t => t.id !== id))
    }, TOAST_DURATION)
    timersRef.current.set(id, timer)
  }, [])

  // Cleanup all timers on unmount
  useEffect(() => {
    const timers = timersRef.current
    return () => {
      timers.forEach(t => clearTimeout(t))
      timers.clear()
    }
  }, [])

  return { toasts, addToast, removeToast }
}

export default function ToastContainer({
  toasts,
  onDismiss,
}: {
  toasts: Toast[]
  onDismiss: (id: string) => void
}) {
  if (toasts.length === 0) return null

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 pointer-events-none">
      {toasts.map(toast => (
        <ToastItem key={toast.id} toast={toast} onDismiss={onDismiss} />
      ))}
    </div>
  )
}

function ToastItem({ toast, onDismiss }: { toast: Toast; onDismiss: (id: string) => void }) {
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    // Trigger slide-in on next frame
    const raf = requestAnimationFrame(() => setVisible(true))
    return () => cancelAnimationFrame(raf)
  }, [])

  return (
    <div
      className={`pointer-events-auto flex items-center gap-2 px-3 py-2 rounded-md border text-sm shadow-lg backdrop-blur-sm transition-all duration-300 ease-out ${colorClasses[toast.type]} ${visible ? 'translate-x-0 opacity-100' : 'translate-x-8 opacity-0'}`}
      style={{ minWidth: 260, maxWidth: 360 }}
    >
      <span className={`font-bold text-base leading-none ${iconColorClasses[toast.type]}`}>
        {icons[toast.type]}
      </span>
      <span className="flex-1 truncate">{toast.message}</span>
      <button
        onClick={() => onDismiss(toast.id)}
        className="ml-1 opacity-60 hover:opacity-100 transition-opacity text-current"
        aria-label="Dismiss"
      >
        ✕
      </button>
    </div>
  )
}
