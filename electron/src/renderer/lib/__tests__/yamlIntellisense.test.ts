import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  isSwarmYaml,
  extractTaskNames,
  validateSwarmYaml,
  createHoverProvider,
} from '../yamlIntellisense'

describe('isSwarmYaml', () => {
  it('returns true for paths ending in swarm.yaml', () => {
    expect(isSwarmYaml('swarm.yaml')).toBe(true)
    expect(isSwarmYaml('/path/to/swarm.yaml')).toBe(true)
    expect(isSwarmYaml('project/swarm/swarm.yaml')).toBe(true)
  })

  it('returns true for paths ending in swarm.yml', () => {
    expect(isSwarmYaml('swarm.yml')).toBe(true)
    expect(isSwarmYaml('/path/to/swarm.yml')).toBe(true)
  })

  it('returns false for other YAML files', () => {
    expect(isSwarmYaml('config.yaml')).toBe(false)
    expect(isSwarmYaml('docker-compose.yaml')).toBe(false)
    expect(isSwarmYaml('swarm-config.yaml')).toBe(false)
    expect(isSwarmYaml('/path/to/other.yml')).toBe(false)
  })

  it('returns false for non-YAML files', () => {
    expect(isSwarmYaml('swarm.json')).toBe(false)
    expect(isSwarmYaml('swarm.toml')).toBe(false)
    expect(isSwarmYaml('swarm.yaml.bak')).toBe(false)
    expect(isSwarmYaml('readme.md')).toBe(false)
  })
})

describe('extractTaskNames', () => {
  it('extracts task names from valid YAML content', () => {
    const content = `version: "1"
tasks:
  planner:
    prompt: planner
  coder:
    prompt: coder
  reviewer:
    prompt: reviewer`

    const names = extractTaskNames(content)
    expect(names).toEqual(['planner', 'coder', 'reviewer'])
  })

  it('returns empty array for content with no tasks', () => {
    const content = `version: "1"
pipelines:
  main:
    iterations: 10`

    const names = extractTaskNames(content)
    expect(names).toEqual([])
  })

  it('returns empty array for empty content', () => {
    expect(extractTaskNames('')).toEqual([])
  })

  it('returns empty array when tasks section is empty', () => {
    const content = `version: "1"
tasks:`

    const names = extractTaskNames(content)
    expect(names).toEqual([])
  })

  it('handles task names with hyphens and underscores', () => {
    const content = `tasks:
  my-task:
    prompt: a
  another_task:
    prompt: b
  task123:
    prompt: c`

    const names = extractTaskNames(content)
    expect(names).toEqual(['my-task', 'another_task', 'task123'])
  })

  it('ignores comments in tasks section', () => {
    const content = `tasks:
  # This is a comment
  real-task:
    prompt: test`

    const names = extractTaskNames(content)
    expect(names).toEqual(['real-task'])
  })

  it('stops extracting when another top-level key appears', () => {
    const content = `tasks:
  taskA:
    prompt: a
pipelines:
  main:
    tasks:
      - taskA`

    const names = extractTaskNames(content)
    expect(names).toEqual(['taskA'])
  })

  it('handles content with no version key', () => {
    const content = `tasks:
  simple:
    prompt: test`

    const names = extractTaskNames(content)
    expect(names).toEqual(['simple'])
  })
})

describe('validateSwarmYaml', () => {
  let mockMonaco: {
    editor: {
      setModelMarkers: ReturnType<typeof vi.fn>
    }
  }
  let mockModel: {
    uri: { toString: () => string }
  }

  beforeEach(() => {
    mockMonaco = {
      editor: {
        setModelMarkers: vi.fn(),
      },
    }
    mockModel = {
      uri: { toString: () => 'file:///test/swarm.yaml' },
    }
  })

  it('detects unknown top-level keys', () => {
    const content = `version: "1"
unknownKey: value
tasks:
  taskA:
    prompt: test`

    validateSwarmYaml(content, mockMonaco as any, mockModel as any)

    expect(mockMonaco.editor.setModelMarkers).toHaveBeenCalledWith(
      mockModel,
      'swarm-yaml',
      expect.arrayContaining([
        expect.objectContaining({
          message: expect.stringContaining('Unknown top-level key "unknownKey"'),
        }),
      ])
    )
  })

  it('detects unknown task keys', () => {
    const content = `tasks:
  taskA:
    prompt: test
    unknownTaskKey: value`

    validateSwarmYaml(content, mockMonaco as any, mockModel as any)

    expect(mockMonaco.editor.setModelMarkers).toHaveBeenCalledWith(
      mockModel,
      'swarm-yaml',
      expect.arrayContaining([
        expect.objectContaining({
          message: expect.stringContaining('Unknown task key "unknownTaskKey"'),
        }),
      ])
    )
  })

  it('validates condition values - rejects invalid', () => {
    const content = `tasks:
  taskA:
    prompt: a
  taskB:
    prompt: b
    depends_on:
      - task: taskA
        condition: invalid_condition`

    validateSwarmYaml(content, mockMonaco as any, mockModel as any)

    expect(mockMonaco.editor.setModelMarkers).toHaveBeenCalledWith(
      mockModel,
      'swarm-yaml',
      expect.arrayContaining([
        expect.objectContaining({
          message: expect.stringContaining('Invalid condition "invalid_condition"'),
          severity: 8, // Error
        }),
      ])
    )
  })

  it('validates condition values - accepts valid', () => {
    const content = `tasks:
  taskA:
    prompt: a
  taskB:
    prompt: b
    depends_on:
      - task: taskA
        condition: success`

    validateSwarmYaml(content, mockMonaco as any, mockModel as any)

    const calls = mockMonaco.editor.setModelMarkers.mock.calls
    const markers = calls[0][2]
    const conditionErrors = markers.filter((m: any) => m.message.includes('condition'))
    expect(conditionErrors).toHaveLength(0)
  })

  it('validates task references in depends_on - object syntax', () => {
    const content = `tasks:
  taskA:
    prompt: a
  taskB:
    prompt: b
    depends_on:
      - task: nonExistentTask
        condition: success`

    validateSwarmYaml(content, mockMonaco as any, mockModel as any)

    expect(mockMonaco.editor.setModelMarkers).toHaveBeenCalledWith(
      mockModel,
      'swarm-yaml',
      expect.arrayContaining([
        expect.objectContaining({
          message: expect.stringContaining('Task "nonExistentTask" not found'),
          severity: 8, // Error
        }),
      ])
    )
  })

  it('validates task references in depends_on - string syntax', () => {
    const content = `tasks:
  taskA:
    prompt: a
  taskB:
    prompt: b
    depends_on:
      - missingTask`

    validateSwarmYaml(content, mockMonaco as any, mockModel as any)

    expect(mockMonaco.editor.setModelMarkers).toHaveBeenCalledWith(
      mockModel,
      'swarm-yaml',
      expect.arrayContaining([
        expect.objectContaining({
          message: expect.stringContaining('Task "missingTask" not found'),
        }),
      ])
    )
  })

  it('validates numeric values for iterations', () => {
    const content = `tasks:
  taskA:
    prompt: a
pipelines:
  main:
    iterations: notANumber
    tasks:
      - taskA`

    validateSwarmYaml(content, mockMonaco as any, mockModel as any)

    expect(mockMonaco.editor.setModelMarkers).toHaveBeenCalledWith(
      mockModel,
      'swarm-yaml',
      expect.arrayContaining([
        expect.objectContaining({
          message: expect.stringContaining('"iterations" must be a number'),
          severity: 8,
        }),
      ])
    )
  })

  it('validates numeric values for parallelism', () => {
    const content = `tasks:
  taskA:
    prompt: a
pipelines:
  main:
    parallelism: abc
    tasks:
      - taskA`

    validateSwarmYaml(content, mockMonaco as any, mockModel as any)

    expect(mockMonaco.editor.setModelMarkers).toHaveBeenCalledWith(
      mockModel,
      'swarm-yaml',
      expect.arrayContaining([
        expect.objectContaining({
          message: expect.stringContaining('"parallelism" must be a number'),
        }),
      ])
    )
  })

  it('accepts valid numeric iterations and parallelism', () => {
    const content = `tasks:
  taskA:
    prompt: a
pipelines:
  main:
    iterations: 10
    parallelism: 2
    tasks:
      - taskA`

    validateSwarmYaml(content, mockMonaco as any, mockModel as any)

    const calls = mockMonaco.editor.setModelMarkers.mock.calls
    const markers = calls[0][2]
    const numericErrors = markers.filter(
      (m: any) => m.message.includes('must be a number')
    )
    expect(numericErrors).toHaveLength(0)
  })

  it('detects unknown pipeline keys', () => {
    const content = `tasks:
  taskA:
    prompt: a
pipelines:
  main:
    unknownPipelineKey: value
    tasks:
      - taskA`

    validateSwarmYaml(content, mockMonaco as any, mockModel as any)

    expect(mockMonaco.editor.setModelMarkers).toHaveBeenCalledWith(
      mockModel,
      'swarm-yaml',
      expect.arrayContaining([
        expect.objectContaining({
          message: expect.stringContaining('Unknown pipeline key "unknownPipelineKey"'),
        }),
      ])
    )
  })

  it('produces no errors for valid content', () => {
    const content = `version: "1"
tasks:
  planner:
    prompt: planner
    model: opus
  coder:
    prompt: coder
    depends_on:
      - task: planner
        condition: success
pipelines:
  main:
    iterations: 10
    parallelism: 2
    tasks:
      - planner
      - coder`

    validateSwarmYaml(content, mockMonaco as any, mockModel as any)

    const calls = mockMonaco.editor.setModelMarkers.mock.calls
    const markers = calls[0][2]
    expect(markers).toHaveLength(0)
  })

  it('handles empty content', () => {
    validateSwarmYaml('', mockMonaco as any, mockModel as any)

    const calls = mockMonaco.editor.setModelMarkers.mock.calls
    const markers = calls[0][2]
    expect(markers).toHaveLength(0)
  })

  it('ignores comments', () => {
    const content = `# Top comment
version: "1"
tasks:
  # Task comment
  taskA:
    prompt: test`

    validateSwarmYaml(content, mockMonaco as any, mockModel as any)

    const calls = mockMonaco.editor.setModelMarkers.mock.calls
    const markers = calls[0][2]
    expect(markers).toHaveLength(0)
  })
})

describe('createHoverProvider', () => {
  // Helper type for hover result - the implementation returns synchronously
  type HoverResult = { range: unknown; contents: Array<{ value: string }> } | null

  it('provides hover for known keys', () => {
    const provider = createHoverProvider()
    const mockModel = {
      getLineContent: vi.fn().mockReturnValue('    prompt: test'),
    }
    const position = { lineNumber: 1, column: 8 } // Over "prompt"

    const result = provider.provideHover!(mockModel as any, position as any, {} as any) as HoverResult

    expect(result).not.toBeNull()
    expect(result?.contents).toBeDefined()
    expect(result?.contents).toHaveLength(2)
  })

  it('returns null for unknown keys', () => {
    const provider = createHoverProvider()
    const mockModel = {
      getLineContent: vi.fn().mockReturnValue('    unknownKey: test'),
    }
    const position = { lineNumber: 1, column: 10 }

    const result = provider.provideHover!(mockModel as any, position as any, {} as any) as HoverResult

    expect(result).toBeNull()
  })

  it('returns null when cursor is not over the key', () => {
    const provider = createHoverProvider()
    const mockModel = {
      getLineContent: vi.fn().mockReturnValue('    prompt: somevalue'),
    }
    const position = { lineNumber: 1, column: 18 } // Over "somevalue"

    const result = provider.provideHover!(mockModel as any, position as any, {} as any) as HoverResult

    expect(result).toBeNull()
  })

  it('returns null for lines without keys', () => {
    const provider = createHoverProvider()
    const mockModel = {
      getLineContent: vi.fn().mockReturnValue('      - taskA'),
    }
    const position = { lineNumber: 1, column: 10 }

    const result = provider.provideHover!(mockModel as any, position as any, {} as any) as HoverResult

    expect(result).toBeNull()
  })

  it('provides hover for depends_on key', () => {
    const provider = createHoverProvider()
    const mockModel = {
      getLineContent: vi.fn().mockReturnValue('    depends_on:'),
    }
    const position = { lineNumber: 1, column: 10 }

    const result = provider.provideHover!(mockModel as any, position as any, {} as any) as HoverResult

    expect(result).not.toBeNull()
    expect(result?.contents[0]).toEqual({ value: '**depends_on**' })
  })

  it('provides hover for condition key in list items', () => {
    const provider = createHoverProvider()
    const mockModel = {
      getLineContent: vi.fn().mockReturnValue('        - condition: success'),
    }
    const position = { lineNumber: 1, column: 14 }

    const result = provider.provideHover!(mockModel as any, position as any, {} as any) as HoverResult

    expect(result).not.toBeNull()
    expect(result?.contents[0]).toEqual({ value: '**condition**' })
  })

  it('provides hover for model key', () => {
    const provider = createHoverProvider()
    const mockModel = {
      getLineContent: vi.fn().mockReturnValue('    model: opus'),
    }
    const position = { lineNumber: 1, column: 7 }

    const result = provider.provideHover!(mockModel as any, position as any, {} as any) as HoverResult

    expect(result).not.toBeNull()
    expect(result?.contents).toContainEqual({ value: '**model**' })
  })

  it('provides hover for iterations key', () => {
    const provider = createHoverProvider()
    const mockModel = {
      getLineContent: vi.fn().mockReturnValue('    iterations: 10'),
    }
    const position = { lineNumber: 1, column: 10 }

    const result = provider.provideHover!(mockModel as any, position as any, {} as any) as HoverResult

    expect(result).not.toBeNull()
    expect(result?.contents).toContainEqual({ value: '**iterations**' })
  })
})
