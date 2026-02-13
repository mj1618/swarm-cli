import { useState, useEffect, type ReactNode } from 'react'

interface FileViewerProps {
  filePath: string
}

function isYamlFile(filePath: string): boolean {
  const ext = filePath.split('.').pop()?.toLowerCase()
  return ext === 'yaml' || ext === 'yml'
}

function highlightYamlLine(line: string): ReactNode {
  // Comment lines
  if (/^\s*#/.test(line)) {
    return <span className="text-muted-foreground/70 italic">{line}</span>
  }

  // Key-value lines (key: value)
  const kvMatch = line.match(/^(\s*)([\w.-]+)(\s*:\s*)(.*)$/)
  if (kvMatch) {
    const [, indent, key, colon, value] = kvMatch
    return (
      <>
        {indent}
        <span className="text-blue-400">{key}</span>
        <span className="text-muted-foreground">{colon}</span>
        {highlightYamlValue(value)}
      </>
    )
  }

  // List items (- value)
  const listMatch = line.match(/^(\s*)(- )(.*)$/)
  if (listMatch) {
    const [, indent, dash, value] = listMatch
    return (
      <>
        {indent}
        <span className="text-orange-400">{dash}</span>
        {highlightYamlValue(value)}
      </>
    )
  }

  return line
}

function highlightYamlValue(value: string): ReactNode {
  if (!value) return null

  // Quoted strings
  if (/^["'].*["']$/.test(value)) {
    return <span className="text-green-400">{value}</span>
  }
  // Numbers
  if (/^\d+(\.\d+)?$/.test(value)) {
    return <span className="text-purple-400">{value}</span>
  }
  // Booleans
  if (/^(true|false|yes|no)$/i.test(value)) {
    return <span className="text-purple-400">{value}</span>
  }
  // null
  if (/^(null|~)$/i.test(value)) {
    return <span className="text-red-400">{value}</span>
  }
  // Inline comment after value
  const commentMatch = value.match(/^(.+?)(\s+#.*)$/)
  if (commentMatch) {
    return (
      <>
        <span className="text-yellow-300">{commentMatch[1]}</span>
        <span className="text-muted-foreground/70 italic">{commentMatch[2]}</span>
      </>
    )
  }
  return <span className="text-yellow-300">{value}</span>
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
    default:
      return { label: 'Text', color: 'bg-muted text-muted-foreground' }
  }
}

function getFileName(filePath: string): string {
  return filePath.split('/').pop() || filePath
}

export default function FileViewer({ filePath }: FileViewerProps) {
  const [content, setContent] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    setContent(null)

    window.fs.readfile(filePath).then((result) => {
      if (cancelled) return
      if (result.error) {
        setError(result.error)
      } else {
        setContent(result.content)
      }
      setLoading(false)
    }).catch(() => {
      if (cancelled) return
      setError('Failed to read file')
      setLoading(false)
    })

    return () => { cancelled = true }
  }, [filePath])

  const fileType = getFileType(filePath)
  const fileName = getFileName(filePath)
  const lines = content?.split('\n') ?? []

  return (
    <div className="flex-1 flex flex-col min-h-0">
      {/* Header */}
      <div className="p-3 border-b border-border flex items-center gap-2">
        <span className={`text-xs px-1.5 py-0.5 rounded font-medium ${fileType.color}`}>
          {fileType.label}
        </span>
        <span className="text-sm font-medium text-foreground truncate">{fileName}</span>
        <span className="text-xs text-muted-foreground truncate ml-auto">{filePath}</span>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto">
        {loading ? (
          <div className="p-4 text-sm text-muted-foreground">Loading...</div>
        ) : error ? (
          <div className="p-4 text-sm text-red-400">{error}</div>
        ) : (
          <pre className="text-sm font-mono leading-relaxed">
            <table className="w-full border-collapse">
              <tbody>
                {lines.map((line, i) => (
                  <tr key={i} className="hover:bg-accent/30">
                    <td className="px-3 py-0 text-right text-muted-foreground/50 select-none w-12 align-top">
                      {i + 1}
                    </td>
                    <td className="px-3 py-0 whitespace-pre-wrap break-all text-foreground">
                      {line || '\n'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </pre>
        )}
      </div>
    </div>
  )
}
