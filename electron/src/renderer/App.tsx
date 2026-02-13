import { useState, useEffect, useCallback, useMemo } from 'react'
import { ReactFlowProvider } from '@xyflow/react'
import FileTree from './components/FileTree'
import FileViewer from './components/FileViewer'
import DagCanvas from './components/DagCanvas'
import AgentPanel from './components/AgentPanel'
import ConsolePanel from './components/ConsolePanel'

function isYamlFile(filePath: string): boolean {
  const ext = filePath.split('.').pop()?.toLowerCase()
  return ext === 'yaml' || ext === 'yml'
}

function App() {
  const [selectedFile, setSelectedFile] = useState<string | null>(null)
  const [defaultYamlContent, setDefaultYamlContent] = useState<string | null>(null)
  const [defaultYamlLoading, setDefaultYamlLoading] = useState(true)
  const [defaultYamlError, setDefaultYamlError] = useState<string | null>(null)
  const [selectedYamlContent, setSelectedYamlContent] = useState<string | null>(null)
  const [selectedYamlLoading, setSelectedYamlLoading] = useState(false)
  const [selectedYamlError, setSelectedYamlError] = useState<string | null>(null)

  const handleSelectFile = useCallback((filePath: string) => {
    setSelectedFile(filePath)
  }, [])

  // Load default swarm.yaml for when no file is selected
  useEffect(() => {
    window.fs.readfile('swarm/swarm.yaml').then((result) => {
      if (result.error) {
        setDefaultYamlError(result.error)
      } else {
        setDefaultYamlContent(result.content)
      }
      setDefaultYamlLoading(false)
    }).catch(() => {
      setDefaultYamlError('Failed to read swarm.yaml')
      setDefaultYamlLoading(false)
    })
  }, [])

  // Load selected YAML file content when a YAML file is selected
  useEffect(() => {
    if (!selectedFile || !isYamlFile(selectedFile)) {
      setSelectedYamlContent(null)
      setSelectedYamlError(null)
      setSelectedYamlLoading(false)
      return
    }

    setSelectedYamlLoading(true)
    setSelectedYamlError(null)
    setSelectedYamlContent(null)

    window.fs.readfile(selectedFile).then((result) => {
      if (result.error) {
        setSelectedYamlError(result.error)
      } else {
        setSelectedYamlContent(result.content)
      }
      setSelectedYamlLoading(false)
    }).catch(() => {
      setSelectedYamlError('Failed to read file')
      setSelectedYamlLoading(false)
    })
  }, [selectedFile])

  const selectedIsYaml = selectedFile ? isYamlFile(selectedFile) : false

  const dagLabel = useMemo(() => {
    if (!selectedFile) return 'DAG Editor'
    return selectedFile.split('/').pop() || 'DAG Editor'
  }, [selectedFile])

  return (
    <div className="h-full flex flex-col">
      {/* Title bar drag region */}
      <div className="h-8 bg-background border-b border-border flex items-center px-4 drag-region">
        <span className="text-sm font-medium text-muted-foreground ml-16">Swarm Desktop</span>
      </div>

      {/* Main content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left sidebar - File tree */}
        <div className="w-64 border-r border-border bg-secondary/30 flex flex-col">
          <FileTree selectedPath={selectedFile} onSelectFile={handleSelectFile} />
        </div>

        {/* Center - File viewer or DAG canvas */}
        <div className="flex-1 flex flex-col min-w-0">
          {selectedFile && !selectedIsYaml ? (
            <FileViewer filePath={selectedFile} />
          ) : (
            <>
              <div className="p-3 border-b border-border">
                <h2 className="text-sm font-semibold text-foreground">{dagLabel}</h2>
              </div>
              <ReactFlowProvider>
                <DagCanvas
                  yamlContent={selectedIsYaml ? selectedYamlContent : defaultYamlContent}
                  loading={selectedIsYaml ? selectedYamlLoading : defaultYamlLoading}
                  error={selectedIsYaml ? selectedYamlError : defaultYamlError}
                />
              </ReactFlowProvider>
            </>
          )}
        </div>

        {/* Right sidebar - Agent panel */}
        <div className="w-72 border-l border-border bg-secondary/30 flex flex-col">
          <AgentPanel />
        </div>
      </div>

      {/* Bottom - Console */}
      <div className="h-48 border-t border-border bg-background flex flex-col">
        <ConsolePanel />
      </div>
    </div>
  )
}

export default App
