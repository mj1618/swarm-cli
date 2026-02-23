import { useEffect, useCallback, useRef } from 'react'

interface AboutDialogProps {
  open: boolean
  onClose: () => void
}

export default function AboutDialog({ open, onClose }: AboutDialogProps) {
  const backdropRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        e.stopPropagation()
        onClose()
      }
    }
    document.addEventListener('keydown', handleKeyDown, true)
    return () => document.removeEventListener('keydown', handleKeyDown, true)
  }, [open, onClose])

  const handleBackdropClick = useCallback((e: React.MouseEvent) => {
    if (e.target === backdropRef.current) {
      onClose()
    }
  }, [onClose])

  const handleExternalLink = useCallback((url: string) => {
    // Use the shell API exposed via preload if available, otherwise fallback to window.open
    window.open(url, '_blank', 'noopener,noreferrer')
  }, [])

  if (!open) return null

  return (
    <div
      ref={backdropRef}
      className="fixed inset-0 z-[100] flex items-start justify-center pt-[20vh] bg-black/50"
      onClick={handleBackdropClick}
    >
      <div className="w-full max-w-sm bg-card border border-border rounded-lg shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <h2 className="text-sm font-semibold text-foreground">About Swarm Desktop</h2>
          <button
            onClick={onClose}
            className="text-muted-foreground hover:text-foreground text-lg leading-none"
          >
            ×
          </button>
        </div>

        {/* Body */}
        <div className="px-4 py-6 flex flex-col items-center text-center">
          {/* App Icon / Logo */}
          <div className="w-16 h-16 rounded-xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center mb-4 shadow-lg">
            <svg
              className="w-10 h-10 text-white"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
              />
            </svg>
          </div>

          {/* Title & Version */}
          <h1 className="text-lg font-semibold text-foreground">Swarm Desktop</h1>
          <p className="text-sm text-muted-foreground mt-1">Version 1.0.0</p>

          {/* Description */}
          <p className="text-sm text-muted-foreground mt-4 px-2">
            A visual interface for managing swarm-cli pipelines and AI agents.
          </p>

          {/* Links */}
          <div className="flex items-center gap-4 mt-6">
            <button
              onClick={() => handleExternalLink('https://github.com/mj1618/swarm-cli#readme')}
              className="text-sm text-primary hover:text-primary/80 transition-colors"
            >
              Documentation
            </button>
            <span className="text-muted-foreground">•</span>
            <button
              onClick={() => handleExternalLink('https://github.com/mj1618/swarm-cli')}
              className="text-sm text-primary hover:text-primary/80 transition-colors"
            >
              GitHub
            </button>
          </div>
        </div>

        {/* Footer */}
        <div className="px-4 py-3 border-t border-border bg-secondary/30 text-center">
          <p className="text-xs text-muted-foreground">
            Built with Electron, React, and Tailwind CSS
          </p>
        </div>
      </div>
    </div>
  )
}
