import { describe, it, expect } from 'vitest'
import { parseComposeFile, serializeCompose, composeToFlow, type ComposeFile } from '../yamlParser'

describe('parseComposeFile', () => {
  it('parses valid YAML with tasks', () => {
    const yaml = `
version: "1"
tasks:
  planner:
    prompt: planner
    model: opus
  coder:
    prompt: coder
    depends_on:
      - task: planner
        condition: success
`
    const result = parseComposeFile(yaml)

    expect(result.version).toBe('1')
    expect(result.tasks).toBeDefined()
    expect(Object.keys(result.tasks)).toHaveLength(2)
    expect(result.tasks.planner.prompt).toBe('planner')
    expect(result.tasks.planner.model).toBe('opus')
    expect(result.tasks.coder.depends_on).toHaveLength(1)
  })

  it('parses YAML with pipelines', () => {
    const yaml = `
version: "1"
tasks:
  taskA:
    prompt: a
  taskB:
    prompt: b
pipelines:
  main:
    iterations: 20
    parallelism: 2
    tasks:
      - taskA
      - taskB
`
    const result = parseComposeFile(yaml)

    expect(result.pipelines).toBeDefined()
    expect(result.pipelines?.main.iterations).toBe(20)
    expect(result.pipelines?.main.parallelism).toBe(2)
    expect(result.pipelines?.main.tasks).toEqual(['taskA', 'taskB'])
  })

  it('handles string dependencies (shorthand)', () => {
    const yaml = `
version: "1"
tasks:
  taskA:
    prompt: a
  taskB:
    prompt: b
    depends_on:
      - taskA
`
    const result = parseComposeFile(yaml)

    expect(result.tasks.taskB.depends_on).toEqual(['taskA'])
  })

  it('handles different prompt source types', () => {
    const yaml = `
version: "1"
tasks:
  withPrompt:
    prompt: myPrompt
  withPromptFile:
    prompt-file: ./path/to/prompt.md
  withPromptString:
    prompt-string: "Do something"
`
    const result = parseComposeFile(yaml)

    expect(result.tasks.withPrompt.prompt).toBe('myPrompt')
    expect(result.tasks.withPromptFile['prompt-file']).toBe('./path/to/prompt.md')
    expect(result.tasks.withPromptString['prompt-string']).toBe('Do something')
  })

  it('throws on invalid YAML', () => {
    const invalidYaml = `
version: "1"
tasks:
  broken:
    prompt: [unclosed bracket
`
    expect(() => parseComposeFile(invalidYaml)).toThrow()
  })

  it('parses empty tasks object', () => {
    const yaml = `
version: "1"
tasks: {}
`
    const result = parseComposeFile(yaml)

    expect(result.tasks).toEqual({})
  })
})

describe('serializeCompose', () => {
  it('serializes a compose file back to YAML', () => {
    const compose: ComposeFile = {
      version: '1',
      tasks: {
        planner: {
          prompt: 'planner',
          model: 'opus',
        },
      },
    }

    const result = serializeCompose(compose)

    expect(result).toContain('version:')
    expect(result).toContain('tasks:')
    expect(result).toContain('planner:')
    expect(result).toContain('prompt: planner')
    expect(result).toContain('model: opus')
  })

  it('roundtrips parse -> serialize -> parse', () => {
    const originalYaml = `
version: "1"
tasks:
  taskA:
    prompt: a
    model: sonnet
  taskB:
    prompt: b
    depends_on:
      - task: taskA
        condition: success
pipelines:
  main:
    iterations: 10
    tasks:
      - taskA
      - taskB
`
    const parsed = parseComposeFile(originalYaml)
    const serialized = serializeCompose(parsed)
    const reparsed = parseComposeFile(serialized)

    expect(reparsed.version).toBe(parsed.version)
    expect(reparsed.tasks.taskA.prompt).toBe(parsed.tasks.taskA.prompt)
    expect(reparsed.tasks.taskB.depends_on).toEqual(parsed.tasks.taskB.depends_on)
    expect(reparsed.pipelines?.main.iterations).toBe(parsed.pipelines?.main.iterations)
  })
})

describe('composeToFlow', () => {
  it('converts tasks to React Flow nodes', () => {
    const compose: ComposeFile = {
      version: '1',
      tasks: {
        planner: { prompt: 'planner', model: 'opus' },
        coder: { prompt: 'coder' },
      },
    }

    const { nodes } = composeToFlow(compose)

    expect(nodes).toHaveLength(2)
    expect(nodes.find((n) => n.id === 'planner')).toBeDefined()
    expect(nodes.find((n) => n.id === 'coder')).toBeDefined()

    const plannerNode = nodes.find((n) => n.id === 'planner')!
    expect(plannerNode.data.label).toBe('planner')
    expect(plannerNode.data.promptSource).toBe('planner')
    expect(plannerNode.data.model).toBe('opus')
    expect(plannerNode.type).toBe('taskNode')
  })

  it('creates edges from dependencies', () => {
    const compose: ComposeFile = {
      version: '1',
      tasks: {
        taskA: { prompt: 'a' },
        taskB: {
          prompt: 'b',
          depends_on: [{ task: 'taskA', condition: 'success' }],
        },
      },
    }

    const { edges } = composeToFlow(compose)

    expect(edges).toHaveLength(1)
    expect(edges[0].source).toBe('taskA')
    expect(edges[0].target).toBe('taskB')
    expect(edges[0].label).toBe('success')
  })

  it('handles string dependencies (converts to success condition)', () => {
    const compose: ComposeFile = {
      version: '1',
      tasks: {
        taskA: { prompt: 'a' },
        taskB: {
          prompt: 'b',
          depends_on: ['taskA'],
        },
      },
    }

    const { edges } = composeToFlow(compose)

    expect(edges).toHaveLength(1)
    expect(edges[0].label).toBe('success')
  })

  it('uses saved positions when provided', () => {
    const compose: ComposeFile = {
      version: '1',
      tasks: {
        taskA: { prompt: 'a' },
      },
    }
    const savedPositions = {
      taskA: { x: 100, y: 200 },
    }

    const { nodes } = composeToFlow(compose, savedPositions)

    expect(nodes[0].position).toEqual({ x: 100, y: 200 })
  })

  it('handles empty tasks', () => {
    const compose: ComposeFile = {
      version: '1',
      tasks: {},
    }

    const { nodes, edges } = composeToFlow(compose)

    expect(nodes).toHaveLength(0)
    expect(edges).toHaveLength(0)
  })

  it('ignores dependencies to non-existent tasks', () => {
    const compose: ComposeFile = {
      version: '1',
      tasks: {
        taskA: {
          prompt: 'a',
          depends_on: [{ task: 'nonExistent', condition: 'success' }],
        },
      },
    }

    const { edges } = composeToFlow(compose)

    expect(edges).toHaveLength(0)
  })

  it('detects prompt source from prompt-file', () => {
    const compose: ComposeFile = {
      version: '1',
      tasks: {
        taskA: { 'prompt-file': './custom/path.md' },
      },
    }

    const { nodes } = composeToFlow(compose)

    expect(nodes[0].data.promptSource).toBe('./custom/path.md')
  })

  it('detects prompt source from prompt-string as inline', () => {
    const compose: ComposeFile = {
      version: '1',
      tasks: {
        taskA: { 'prompt-string': 'Do something' },
      },
    }

    const { nodes } = composeToFlow(compose)

    expect(nodes[0].data.promptSource).toBe('inline')
  })
})
