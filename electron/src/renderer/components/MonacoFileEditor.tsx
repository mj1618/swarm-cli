import { useState, useEffect, useRef, useCallback } from 'react'
import Editor, { type OnMount } from '@monaco-editor/react'
import type * as monaco from 'monaco-editor'
import {
  isSwarmYaml,
  createCompletionProvider,
  createHoverProvider,
  validateSwarmYaml,
} from '../lib/yamlIntellisense'

import type { EffectiveTheme } from '../lib/themeManager'

interface MonacoFileEditorProps {
  filePath: string
  theme?: EffectiveTheme
  onDirtyChange?: (filePath: string, isDirty: boolean) => void
  /** Increment to trigger a save operation */
  triggerSave?: number
  /** Called after save completes (success or failure) */
  onSaveComplete?: () => void
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

function isPromptFile(filePath: string): boolean {
  return filePath.includes('/prompts/') && filePath.endsWith('.md')
}

/** Track whether YAML IntelliSense providers have been registered (global, once per language) */
let yamlProvidersRegistered = false

/** Inject CSS for template decoration classes (once) */
let stylesInjected = false
function injectDecorationStyles() {
  if (stylesInjected) return
  stylesInjected = true
  const style = document.createElement('style')
  style.textContent = `
    .template-include-decoration {
      color: #67e8f9 !important;
      text-decoration: underline;
      text-decoration-color: #22d3ee;
      font-style: italic;
    }
    .template-variable-decoration {
      color: #c084fc !important;
      background-color: rgba(192, 132, 252, 0.1);
      border-radius: 2px;
    }
  `
  document.head.appendChild(style)
}

function computeDecorations(
  model: monaco.editor.ITextModel,
): monaco.editor.IModelDeltaDecoration[] {
  const decorations: monaco.editor.IModelDeltaDecoration[] = []
  const text = model.getValue()
  const lines = text.split('\n')

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const lineNumber = i + 1

    // Match all {{...}} patterns
    const pattern = /\{\{([^}]+)\}\}/g
    let match: RegExpExecArray | null
    while ((match = pattern.exec(line)) !== null) {
      const startCol = match.index + 1
      const endCol = match.index + match[0].length + 1
      const inner = match[1].trim()

      const isInclude = inner.startsWith('include:')
      decorations.push({
        range: {
          startLineNumber: lineNumber,
          startColumn: startCol,
          endLineNumber: lineNumber,
          endColumn: endCol,
        } as monaco.IRange,
        options: {
          inlineClassName: isInclude
            ? 'template-include-decoration'
            : 'template-variable-decoration',
          hoverMessage: isInclude
            ? { value: `Include directive: \`${inner.slice(8).trim()}\`` }
            : { value: `Template variable: \`${inner}\`` },
        },
      })
    }
  }

  return decorations
}

export default function MonacoFileEditor({ filePath, theme = 'dark', onDirtyChange, triggerSave, onSaveComplete }: MonacoFileEditorProps) {
  const monacoTheme = theme === 'dark' ? 'vs-dark' : 'vs'
  const [content, setContent] = useState<string | null>(null)
  const [savedContent, setSavedContent] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [showPreview, setShowPreview] = useState(false)
  const [previewContent, setPreviewContent] = useState<string | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [previewError, setPreviewError] = useState<string | null>(null)
  const editorRef = useRef<monaco.editor.IStandaloneCodeEditor | null>(null)
  const decorationsRef = useRef<string[]>([])
  const isDirtyRef = useRef(false)
  const saveRef = useRef<() => void>(() => {})
  const onDirtyChangeRef = useRef(onDirtyChange)
  onDirtyChangeRef.current = onDirtyChange

  const language = getLanguage(filePath)
  const tabSize = getTabSize(language)
  const fileType = getFileType(filePath)
  const fileName = filePath.split('/').pop() || filePath
  const readOnly = isReadOnly(filePath)
  const isPrompt = isPromptFile(filePath)
  const isDirty = content !== null && savedContent !== null && content !== savedContent
  isDirtyRef.current = isDirty

  // Report dirty state changes to parent
  useEffect(() => {
    onDirtyChangeRef.current?.(filePath, isDirty)
  }, [filePath, isDirty])

  // Clean up dirty state when component unmounts
  useEffect(() => {
    return () => {
      onDirtyChangeRef.current?.(filePath, false)
    }
  }, [filePath])

  // Handle save trigger from parent (used for save-and-close flow)
  const prevTriggerSave = useRef(triggerSave)
  useEffect(() => {
    if (triggerSave !== undefined && triggerSave !== prevTriggerSave.current) {
      prevTriggerSave.current = triggerSave
      if (isDirtyRef.current) {
        // Trigger save and notify completion
        const doSave = async () => {
          await saveRef.current()
          onSaveComplete?.()
        }
        doSave()
      } else {
        // No changes to save, still call completion
        onSaveComplete?.()
      }
    }
  }, [triggerSave, onSaveComplete])

  // Load file content
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    setContent(null)
    setSavedContent(null)
    setSaveError(null)
    setShowPreview(false)
    setPreviewContent(null)

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
      // Match by full path or by trailing path suffix
      if (data.path === filePath || data.path.endsWith('/' + filePath) || filePath.endsWith('/' + data.path)) {
        if (!isDirtyRef.current) {
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
  }, [filePath])

  // Load preview content
  const loadPreview = useCallback(async () => {
    setPreviewLoading(true)
    setPreviewError(null)
    try {
      const result = await window.promptResolver.resolve(filePath)
      if (result.error) {
        setPreviewError(result.error)
      } else {
        setPreviewContent(result.content)
      }
    } catch {
      setPreviewError('Failed to resolve prompt')
    }
    setPreviewLoading(false)
  }, [filePath])

  // Refresh preview when toggled on or content saved
  useEffect(() => {
    if (showPreview && isPrompt) {
      loadPreview()
    }
  }, [showPreview, isPrompt, savedContent, loadPreview])

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
  saveRef.current = handleSave

  const isSwarm = isSwarmYaml(filePath)

  // Register Cmd+S / Ctrl+S
  const handleEditorMount: OnMount = useCallback((editor, monacoInstance) => {
    editorRef.current = editor

    if (isPrompt) {
      injectDecorationStyles()
      // Apply decorations on mount
      const model = editor.getModel()
      if (model) {
        const newDecorations = computeDecorations(model)
        decorationsRef.current = editor.deltaDecorations([], newDecorations)

        // Re-apply decorations on content change
        model.onDidChangeContent(() => {
          const updated = computeDecorations(model)
          decorationsRef.current = editor.deltaDecorations(decorationsRef.current, updated)
        })
      }
    }

    // Register YAML IntelliSense providers (once globally) and per-file validation
    if (isSwarm) {
      if (!yamlProvidersRegistered) {
        yamlProvidersRegistered = true

        const getPromptNames = async (): Promise<string[]> => {
          try {
            const result = await window.fs.listprompts()
            if (result.prompts) {
              return result.prompts.map((p) => p.replace(/\.md$/, ''))
            }
          } catch {
            // ignore
          }
          return []
        }

        monacoInstance.languages.registerCompletionItemProvider('yaml', createCompletionProvider(getPromptNames))
        monacoInstance.languages.registerHoverProvider('yaml', createHoverProvider())
      }

      // Run validation on mount and on each change
      const model = editor.getModel()
      if (model) {
        validateSwarmYaml(model.getValue(), monacoInstance, model)
        model.onDidChangeContent(() => {
          validateSwarmYaml(model.getValue(), monacoInstance, model)
        })
      }
    }

    editor.addCommand(
      // Monaco KeyMod.CtrlCmd | Monaco KeyCode.KeyS
      2048 | 49, // CtrlCmd + KeyS
      () => { saveRef.current() },
    )
  }, [isPrompt, isSwarm])

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
        {isMarkdown && (
          <button
            onClick={() => setShowPreview((v) => !v)}
            className={`text-xs px-2 py-1 rounded transition-colors ${
              showPreview
                ? 'bg-cyan-500/20 text-cyan-300 hover:bg-cyan-500/30'
                : 'bg-muted text-muted-foreground hover:bg-muted/80'
            }`}
            title={showPreview ? 'Hide preview' : 'Show preview'}
          >
            {showPreview ? 'Hide Preview' : 'Preview'}
          </button>
        )}
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

      {/* Editor + Preview */}
      <div className={`flex-1 min-h-0 flex ${showPreview ? 'flex-row' : ''}`}>
        {/* Monaco Editor */}
        <div className={`${showPreview ? 'w-1/2 border-r border-border' : 'flex-1'} min-h-0`}>
          <Editor
            language={language}
            value={content ?? ''}
            theme={monacoTheme}
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

        {/* Preview Panel */}
        {showPreview && (
          <div className="w-1/2 min-h-0 flex flex-col">
            <div className="px-3 py-1.5 border-b border-border flex items-center gap-2">
              <span className="text-xs text-muted-foreground">Markdown Preview</span>
              {isPrompt && (
                <>
                  <span className="text-xs text-muted-foreground">|</span>
                  <button
                    onClick={loadPreview}
                    disabled={previewLoading}
                    className="text-xs px-1.5 py-0.5 rounded bg-muted text-muted-foreground hover:bg-muted/80 transition-colors disabled:opacity-50"
                    title="Resolve template includes and show result"
                  >
                    {previewLoading ? 'Resolving...' : 'Resolve Includes'}
                  </button>
                </>
              )}
            </div>
            <div className="flex-1 min-h-0 overflow-auto bg-[#1e1e1e]">
              {previewError ? (
                <div className="p-4 text-sm text-red-400">{previewError}</div>
              ) : previewContent ? (
                /* Show resolved content when available (prompt files) */
                <div
                  className="p-4 prose prose-sm prose-invert max-w-none
                    prose-headings:text-[#d4d4d4] prose-headings:font-semibold
                    prose-p:text-[#d4d4d4] prose-p:leading-relaxed
                    prose-a:text-cyan-400 prose-a:no-underline hover:prose-a:underline
                    prose-code:text-purple-300 prose-code:bg-[#2d2d2d] prose-code:px-1 prose-code:py-0.5 prose-code:rounded prose-code:before:content-none prose-code:after:content-none
                    prose-pre:bg-[#2d2d2d] prose-pre:border prose-pre:border-[#3d3d3d]
                    prose-blockquote:border-l-cyan-500 prose-blockquote:text-[#a0a0a0]
                    prose-strong:text-[#e0e0e0]
                    prose-ul:text-[#d4d4d4] prose-ol:text-[#d4d4d4]
                    prose-li:text-[#d4d4d4]"
                  dangerouslySetInnerHTML={{ __html: marked(previewContent) as string }}
                />
              ) : (
                /* Show live markdown preview */
                <div
                  className="p-4 prose prose-sm prose-invert max-w-none
                    prose-headings:text-[#d4d4d4] prose-headings:font-semibold
                    prose-p:text-[#d4d4d4] prose-p:leading-relaxed
                    prose-a:text-cyan-400 prose-a:no-underline hover:prose-a:underline
                    prose-code:text-purple-300 prose-code:bg-[#2d2d2d] prose-code:px-1 prose-code:py-0.5 prose-code:rounded prose-code:before:content-none prose-code:after:content-none
                    prose-pre:bg-[#2d2d2d] prose-pre:border prose-pre:border-[#3d3d3d]
                    prose-blockquote:border-l-cyan-500 prose-blockquote:text-[#a0a0a0]
                    prose-strong:text-[#e0e0e0]
                    prose-ul:text-[#d4d4d4] prose-ol:text-[#d4d4d4]
                    prose-li:text-[#d4d4d4]"
                  dangerouslySetInnerHTML={{ __html: renderedMarkdown as string }}
                />
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
