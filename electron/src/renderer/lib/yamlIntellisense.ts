import type * as monaco from 'monaco-editor'

// Known keys at each level of swarm.yaml
const TOP_LEVEL_KEYS = ['version', 'tasks', 'pipelines']
const TASK_KEYS = ['prompt', 'prompt-file', 'prompt-string', 'model', 'prefix', 'suffix', 'depends_on']
const PIPELINE_KEYS = ['iterations', 'parallelism', 'tasks']
const CONDITION_VALUES = ['success', 'failure', 'any', 'always']
const MODEL_VALUES = ['opus', 'sonnet', 'haiku']

const HOVER_DOCS: Record<string, string> = {
  prompt: 'Name of a prompt file from swarm/prompts/ (without .md extension)',
  'prompt-file': 'Path to a prompt file relative to the project root',
  'prompt-string': 'Inline prompt string',
  model: 'Model to use for this task (overrides default)',
  prefix: 'Text prepended to the prompt before sending to the agent',
  suffix: 'Text appended to the prompt before sending to the agent',
  iterations: 'Number of iterations to run for this pipeline',
  parallelism: 'Maximum concurrent agents for this pipeline',
  depends_on: 'List of task dependencies with conditions',
  condition: 'When to trigger: success, failure, any, or always',
  version: 'Compose file format version',
  tasks: 'Map of task names to their definitions',
  pipelines: 'Map of pipeline names to their definitions',
}

/** Check if a file path looks like a swarm YAML file */
export function isSwarmYaml(filePath: string): boolean {
  return filePath.endsWith('swarm.yaml') || filePath.endsWith('swarm.yml')
}

/** Extract all task names from YAML content using line-based heuristics */
function extractTaskNames(content: string): string[] {
  const lines = content.split('\n')
  const names: string[] = []
  let inTasksBlock = false

  for (const line of lines) {
    // Detect top-level "tasks:" key
    if (/^tasks:\s*$/.test(line)) {
      inTasksBlock = true
      continue
    }
    // Exit tasks block when another top-level key appears
    if (inTasksBlock && /^\S/.test(line) && !line.startsWith('#')) {
      inTasksBlock = false
      continue
    }
    // Task names are at 2-space indent under tasks:
    if (inTasksBlock) {
      const match = line.match(/^ {2}([a-zA-Z0-9_-]+):\s*$/)
      if (match) {
        names.push(match[1])
      }
    }
  }

  return names
}

/** Determine cursor context from line content and position */
interface CursorContext {
  type: 'task-value' | 'prompt-value' | 'condition-value' | 'model-value' | 'depends-on-task' | 'unknown'
}

function getCursorContext(model: monaco.editor.ITextModel, position: monaco.Position): CursorContext {
  const lineContent = model.getLineContent(position.lineNumber)
  const trimmed = lineContent.trimStart()

  // condition: value
  if (trimmed.startsWith('condition:')) {
    return { type: 'condition-value' }
  }

  // task: value inside depends_on list
  if (trimmed.startsWith('task:') || trimmed.startsWith('- task:')) {
    return { type: 'depends-on-task' }
  }

  // prompt: value (but not prompt-file or prompt-string)
  if (/^prompt:\s/.test(trimmed) || trimmed === 'prompt:') {
    return { type: 'prompt-value' }
  }

  // model: value
  if (trimmed.startsWith('model:')) {
    return { type: 'model-value' }
  }

  // Check if we're in a depends_on list item that's just a string (- taskname)
  // Look upward for depends_on:
  if (trimmed.startsWith('- ') && !trimmed.includes(':')) {
    for (let i = position.lineNumber - 1; i >= 1; i--) {
      const prevLine = model.getLineContent(i).trimStart()
      if (prevLine.startsWith('depends_on:')) {
        return { type: 'depends-on-task' }
      }
      // If we hit a non-list, non-empty line that's not indented enough, stop
      if (prevLine && !prevLine.startsWith('-') && !prevLine.startsWith('#')) {
        break
      }
    }
  }

  return { type: 'unknown' }
}

export function createCompletionProvider(
  getPromptNames: () => Promise<string[]>,
): monaco.languages.CompletionItemProvider {
  return {
    triggerCharacters: [' ', ':'],
    provideCompletionItems: async (model, position) => {
      const ctx = getCursorContext(model, position)
      const word = model.getWordUntilPosition(position)
      const range = {
        startLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endLineNumber: position.lineNumber,
        endColumn: word.endColumn,
      }

      const suggestions: monaco.languages.CompletionItem[] = []

      if (ctx.type === 'condition-value') {
        for (const val of CONDITION_VALUES) {
          suggestions.push({
            label: val,
            kind: 12, // monaco.languages.CompletionItemKind.Value
            insertText: val,
            range,
            detail: 'Dependency condition',
          })
        }
      } else if (ctx.type === 'model-value') {
        for (const val of MODEL_VALUES) {
          suggestions.push({
            label: val,
            kind: 12,
            insertText: val,
            range,
            detail: 'Model name',
          })
        }
      } else if (ctx.type === 'prompt-value') {
        try {
          const prompts = await getPromptNames()
          for (const name of prompts) {
            suggestions.push({
              label: name,
              kind: 16, // monaco.languages.CompletionItemKind.File
              insertText: name,
              range,
              detail: 'Prompt file',
            })
          }
        } catch {
          // Silently fail — prompts unavailable
        }
      } else if (ctx.type === 'depends-on-task') {
        const taskNames = extractTaskNames(model.getValue())
        for (const name of taskNames) {
          suggestions.push({
            label: name,
            kind: 7, // monaco.languages.CompletionItemKind.Class (used for task names)
            insertText: name,
            range,
            detail: 'Task name',
          })
        }
      }

      return { suggestions }
    },
  }
}

export function createHoverProvider(): monaco.languages.HoverProvider {
  return {
    provideHover: (model, position) => {
      const lineContent = model.getLineContent(position.lineNumber)
      // Find a key on this line: key: or key-with-dashes:
      const keyMatch = lineContent.match(/^\s*-?\s*([a-zA-Z_-]+)\s*:/)
      if (!keyMatch) return null

      const key = keyMatch[1]
      const doc = HOVER_DOCS[key]
      if (!doc) return null

      // Make sure cursor is over the key
      const keyStart = lineContent.indexOf(key) + 1
      const keyEnd = keyStart + key.length
      if (position.column < keyStart || position.column > keyEnd) return null

      return {
        range: {
          startLineNumber: position.lineNumber,
          startColumn: keyStart,
          endLineNumber: position.lineNumber,
          endColumn: keyEnd,
        },
        contents: [
          { value: `**${key}**` },
          { value: doc },
        ],
      }
    },
  }
}

export function validateSwarmYaml(
  content: string,
  monacoInstance: typeof monaco,
  model: monaco.editor.ITextModel,
): void {
  const markers: monaco.editor.IMarkerData[] = []
  const lines = content.split('\n')

  const taskNames = extractTaskNames(content)

  // Track which section we're in
  let section: 'top' | 'tasks' | 'pipelines' | 'task-body' | 'pipeline-body' | 'depends-on' = 'top'

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const lineNumber = i + 1

    // Skip empty lines and comments
    if (!line.trim() || line.trim().startsWith('#')) continue

    const indent = line.length - line.trimStart().length
    const trimmed = line.trimStart()

    // Top-level key detection (no indent)
    if (indent === 0 && trimmed.includes(':')) {
      const topKey = trimmed.split(':')[0].trim()
      if (!TOP_LEVEL_KEYS.includes(topKey)) {
        markers.push({
          severity: 4, // monaco.MarkerSeverity.Warning
          message: `Unknown top-level key "${topKey}". Expected: ${TOP_LEVEL_KEYS.join(', ')}`,
          startLineNumber: lineNumber,
          startColumn: 1,
          endLineNumber: lineNumber,
          endColumn: topKey.length + 1,
        })
      }
      if (topKey === 'tasks') section = 'tasks'
      else if (topKey === 'pipelines') section = 'pipelines'
      else section = 'top'
      continue
    }

    // Inside tasks section
    if (section === 'tasks' || section === 'task-body' || section === 'depends-on') {
      // Task name at indent 2
      if (indent === 2 && trimmed.match(/^[a-zA-Z0-9_-]+:\s*$/)) {
        section = 'task-body'
        continue
      }

      // Task property at indent 4
      if (indent === 4 && section === 'task-body' && trimmed.includes(':')) {
        const key = trimmed.split(':')[0].trim().replace(/^-\s*/, '')
        if (key === 'depends_on') {
          section = 'depends-on'
          continue
        }
        if (!TASK_KEYS.includes(key)) {
          markers.push({
            severity: 4,
            message: `Unknown task key "${key}". Expected: ${TASK_KEYS.join(', ')}`,
            startLineNumber: lineNumber,
            startColumn: indent + 1,
            endLineNumber: lineNumber,
            endColumn: indent + key.length + 1,
          })
        }
        continue
      }

      // depends_on list items
      if (section === 'depends-on' && trimmed.startsWith('-')) {
        // Reset back to task-body when indent drops
        if (indent <= 4 && !trimmed.startsWith('-')) {
          section = 'task-body'
        }
      }

      // Validate condition values
      if (trimmed.startsWith('condition:')) {
        const val = trimmed.split(':').slice(1).join(':').trim()
        if (val && !CONDITION_VALUES.includes(val)) {
          const colStart = line.indexOf(val) + 1
          markers.push({
            severity: 8, // monaco.MarkerSeverity.Error
            message: `Invalid condition "${val}". Must be one of: ${CONDITION_VALUES.join(', ')}`,
            startLineNumber: lineNumber,
            startColumn: colStart,
            endLineNumber: lineNumber,
            endColumn: colStart + val.length,
          })
        }
      }

      // Validate task references in depends_on
      if (trimmed.startsWith('task:') || trimmed.startsWith('- task:')) {
        const val = trimmed.replace(/^-?\s*task:\s*/, '').trim()
        if (val && !taskNames.includes(val)) {
          const colStart = line.indexOf(val) + 1
          markers.push({
            severity: 8,
            message: `Task "${val}" not found. Available tasks: ${taskNames.join(', ')}`,
            startLineNumber: lineNumber,
            startColumn: colStart,
            endLineNumber: lineNumber,
            endColumn: colStart + val.length,
          })
        }
      }

      // Simple string depends_on items (- taskname)
      if (section === 'depends-on' && trimmed.startsWith('- ') && !trimmed.includes(':')) {
        const val = trimmed.slice(2).trim()
        if (val && !taskNames.includes(val)) {
          const colStart = line.indexOf(val) + 1
          markers.push({
            severity: 8,
            message: `Task "${val}" not found. Available tasks: ${taskNames.join(', ')}`,
            startLineNumber: lineNumber,
            startColumn: colStart,
            endLineNumber: lineNumber,
            endColumn: colStart + val.length,
          })
        }
      }
    }

    // Inside pipelines section
    if (section === 'pipelines' || section === 'pipeline-body') {
      if (indent === 2 && trimmed.match(/^[a-zA-Z0-9_-]+:\s*$/)) {
        section = 'pipeline-body'
        continue
      }

      if (indent === 4 && section === 'pipeline-body' && trimmed.includes(':')) {
        const key = trimmed.split(':')[0].trim()
        if (!PIPELINE_KEYS.includes(key)) {
          markers.push({
            severity: 4,
            message: `Unknown pipeline key "${key}". Expected: ${PIPELINE_KEYS.join(', ')}`,
            startLineNumber: lineNumber,
            startColumn: indent + 1,
            endLineNumber: lineNumber,
            endColumn: indent + key.length + 1,
          })
        }

        // Validate numeric values
        if (key === 'iterations' || key === 'parallelism') {
          const val = trimmed.split(':').slice(1).join(':').trim()
          if (val && isNaN(Number(val))) {
            const colStart = line.indexOf(val) + 1
            markers.push({
              severity: 8,
              message: `"${key}" must be a number, got "${val}"`,
              startLineNumber: lineNumber,
              startColumn: colStart,
              endLineNumber: lineNumber,
              endColumn: colStart + val.length,
            })
          }
        }
      }
    }
  }

  monacoInstance.editor.setModelMarkers(model, 'swarm-yaml', markers)
}

export function clearSwarmMarkers(
  monacoInstance: typeof monaco,
  model: monaco.editor.ITextModel,
): void {
  monacoInstance.editor.setModelMarkers(model, 'swarm-yaml', [])
}
