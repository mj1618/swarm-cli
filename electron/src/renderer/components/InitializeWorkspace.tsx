import { useState, useCallback } from 'react'
import { Folder, Loader2 } from 'lucide-react'

interface InitializeWorkspaceProps {
  projectPath: string | null
  onInitialize: () => Promise<void>
}

export default function InitializeWorkspace({ projectPath, onInitialize }: InitializeWorkspaceProps) {
  const [isInitializing, setIsInitializing] = useState(false)

  const handleInitialize = useCallback(async () => {
    setIsInitializing(true)
    try {
      await onInitialize()
    } finally {
      setIsInitializing(false)
    }
  }, [onInitialize])

  return (
    <div className="flex-1 flex items-center justify-center p-8">
      <div className="text-center max-w-md space-y-6">
        {/* Icon */}
        <div className="mx-auto w-20 h-20 rounded-full bg-secondary/50 flex items-center justify-center">
          <Folder className="w-10 h-10 text-muted-foreground" />
        </div>

        {/* Heading */}
        <h2 className="text-xl font-semibold text-foreground">No swarm project found</h2>

        {/* Description */}
        <p className="text-sm text-muted-foreground leading-relaxed">
          This directory doesn't have a <code className="px-1.5 py-0.5 bg-secondary rounded text-xs font-mono">swarm/</code> folder.
          Initialize a new swarm project to start creating AI agent pipelines.
        </p>

        {/* Project path display */}
        {projectPath && (
          <p className="text-xs text-muted-foreground/70 font-mono truncate px-4">
            {projectPath}
          </p>
        )}

        {/* Initialize button */}
        <button
          onClick={handleInitialize}
          disabled={isInitializing}
          className="inline-flex items-center gap-2 px-5 py-2.5 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors font-medium"
        >
          {isInitializing ? (
            <>
              <Loader2 className="w-4 h-4 animate-spin" />
              Initializing...
            </>
          ) : (
            'Initialize Swarm Project'
          )}
        </button>

        {/* What gets created */}
        <div className="text-left bg-secondary/30 rounded-lg p-4 space-y-2">
          <p className="text-xs font-medium text-muted-foreground">This will create:</p>
          <ul className="text-xs text-muted-foreground/80 space-y-1.5">
            <li className="flex items-start gap-2">
              <span className="text-primary">•</span>
              <span><code className="font-mono">swarm/swarm.yaml</code> — Pipeline configuration</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-primary">•</span>
              <span><code className="font-mono">swarm/prompts/</code> — Directory for prompt files</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-primary">•</span>
              <span><code className="font-mono">swarm/prompts/example.md</code> — Example prompt</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-primary">•</span>
              <span><code className="font-mono">swarm/swarm.toml</code> — CLI settings</span>
            </li>
          </ul>
        </div>
      </div>
    </div>
  )
}
