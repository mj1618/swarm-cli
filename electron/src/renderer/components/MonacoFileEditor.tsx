import { useState, useEffect, useRef, useCallback } from 'react'
import Editor, { type OnMount } from '@monaco-editor/react'
import type * as monaco from 'monaco-editor'

interface MonacoFileEditorProps {
  filePath: string
}

function getLanguage(filePath: string): string {
  const ext = filePath.split('.').pop()?.toLowerCase()
  switch (ext) {
    case 'yaml':
    case 'yml':
      return 'yaml'
    case 'md':
      return 'markdown'
    case 'toml':
      return 'ini'
    case 'json':
      return 'json'
    case 'ts':
    case 'tsx':
      return 'typescript'
    case 'js':
    case 'jsx':
      return 'javascript'
    case 'go':
      return 'go'
    case 'log':
      return 'plaintext'
    default:
      return 'plaintext'
  }
}

function getTabSize(language: string): number {
  switch (language) {
    case 'yaml':
    case 'json':
      return 2
    case 'go':
      return 4
    default:
      return 2
  }
}

function getFileType(filePath: string): { label: string; color: string } {
  const ext = filePath.split('.').pop()?.toLowerCase()
  switch (ext) {
    case 'yaml':
    case 'yml':
      return { label: 'YAML', color: 'bg-yellow-500/20 text-yellow-300' }
    case 'md':
      return { label: 'Markdown', color: 'bg-green-500/20 text-green-300' }
    case 'toml':
      return { label: 'Config', color: 'bg-orange-500/20 text-orange-300' }
    case 'log':
      return { label: 'Log', color: 'bg-gray-500/20 text-gray-300' }
    case 'json':
      return { label: 'JSON', color: 'bg-blue-500/20 text-blue-300' }
    case 'ts':
    case 'tsx':
      return { label: 'TypeScript', color: 'bg-blue-500/20 text-blue-300' }
    case 'js':
    case 'jsx':
      return { label: 'JavaScript', color: 'bg-yellow-500/20 text-yellow-300' }
    case 'go':
      return { label: 'Go', color: 'bg-cyan-500/20 text-cyan-300' }
    default:
      return { label: 'Text', color: 'bg-muted text-muted-foreground' }
  }
}

function isReadOnly(filePath: string): boolean {
  const ext = filePath.split('.').pop()?.toLowerCase()
  return ext === 'log'
}

export default function MonacoFileEditor({ filePath }: MonacoFileEditorProps) {
  const [content, setContent] = useState<string | null>(null)
  const [savedContent, setSavedContent] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const editorRef = useRef<monaco.editor.IStandaloneCodeEditor | null>(null)

  const language = getLanguage(filePath)
  const tabSize = getTabSize(language)
  const fileType = getFileType(filePath)
  const fileName = filePath.split('/').pop() || filePath
  const readOnly = isReadOnly(filePath)
  const isDirty = content !== null && savedContent !== null && content !== savedContent

  // Load file content
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    setContent(null)
    setSavedContent(null)
    setSaveError(null)

    window.fs.readfile(filePath).then((result) => {
      if (cancelled) return
      if (result.error) {
        setError(result.error)
      } else {
        setContent(result.content)
        setSavedContent(result.content)
      }
      setLoading(false)
    }).catch(() => {
      if (cancelled) return
      setError('Failed to read file')
      setLoading(false)
    })

    return () => { cancelled = true }
  }, [filePath])

  // Watch for external file changes
  useEffect(() => {
    const unsubscribe = window.fs.onChanged((data) => {
      if (data.path.endsWith(filePath) || filePath.endsWith(data.path)) {
        // Reload if not dirty
        if (!isDirty) {
          window.fs.readfile(filePath).then((result) => {
            if (!result.error) {
              setContent(result.content)
              setSavedContent(result.content)
            }
          })
        }
      }
    })
    return unsubscribe
  }, [filePath, isDirty])

  const handleSave = useCallback(async () => {
    if (content === null || readOnly) return
    setSaving(true)
    setSaveError(null)
    const result = await window.fs.writefile(filePath, content)
    if (result.error) {
      setSaveError(result.error)
    } else {
      setSavedContent(content)
    }
    setSaving(false)
  }, [content, filePath, readOnly])

  // Register Cmd+S / Ctrl+S
  const handleEditorMount: OnMount = useCallback((editor) => {
    editorRef.current = editor
    editor.addCommand(
      // Monaco KeyMod.CtrlCmd | Monaco KeyCode.KeyS
      2048 | 49, // CtrlCmd + KeyS
      () => { handleSave() },
    )
  }, [handleSave])

  const handleChange = useCallback((value: string | undefined) => {
    if (value !== undefined) {
      setContent(value)
    }
  }, [])

  if (loading) {
    return (
      <div className="flex-1 flex flex-col min-h-0">
        <div className="p-3 border-b border-border flex items-center gap-2">
          <span className={`text-xs px-1.5 py-0.5 rounded font-medium ${fileType.color}`}>
            {fileType.label}
          </span>
          <span className="text-sm font-medium text-foreground truncate">{fileName}</span>
        </div>
        <div className="flex-1 flex items-center justify-center">
          <span className="text-sm text-muted-foreground">Loading...</span>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex-1 flex flex-col min-h-0">
        <div className="p-3 border-b border-border flex items-center gap-2">
          <span className={`text-xs px-1.5 py-0.5 rounded font-medium ${fileType.color}`}>
            {fileType.label}
          </span>
          <span className="text-sm font-medium text-foreground truncate">{fileName}</span>
        </div>
        <div className="flex-1 flex items-center justify-center">
          <span className="text-sm text-red-400">{error}</span>
        </div>
      </div>
    )
  }

  return (
    <div className="flex-1 flex flex-col min-h-0">
      {/* Header */}
      <div className="p-3 border-b border-border flex items-center gap-2">
        <span className={`text-xs px-1.5 py-0.5 rounded font-medium ${fileType.color}`}>
          {fileType.label}
        </span>
        <span className="text-sm font-medium text-foreground truncate">
          {fileName}
          {isDirty && <span className="text-orange-400 ml-1" title="Unsaved changes">&bull;</span>}
        </span>
        <span className="text-xs text-muted-foreground truncate ml-auto">{filePath}</span>
        {!readOnly && (
          <>
            {saveError && (
              <span className="text-xs text-red-400">{saveError}</span>
            )}
            <button
              onClick={handleSave}
              disabled={!isDirty || saving}
              className="text-xs px-2 py-1 rounded bg-primary/20 text-primary hover:bg-primary/30 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            >
              {saving ? 'Saving...' : 'Save'}
            </button>
          </>
        )}
        {readOnly && (
          <span className="text-xs px-1.5 py-0.5 rounded bg-muted text-muted-foreground">Read-only</span>
        )}
      </div>

      {/* Monaco Editor */}
      <div className="flex-1 min-h-0">
        <Editor
          language={language}
          value={content ?? ''}
          theme="vs-dark"
          onChange={handleChange}
          onMount={handleEditorMount}
          options={{
            readOnly,
            minimap: { enabled: false },
            wordWrap: language === 'markdown' ? 'on' : 'off',
            fontSize: 13,
            scrollBeyondLastLine: false,
            automaticLayout: true,
            tabSize,
            lineNumbers: 'on',
            renderLineHighlight: 'line',
            bracketPairColorization: { enabled: true },
            padding: { top: 8 },
          }}
        />
      </div>
    </div>
  )
}
