import { describe, it, expect } from 'vitest'
import { validateDag } from '../dagValidation'
import type { ComposeFile } from '../yamlParser'

describe('validateDag', () => {
  describe('cycle detection', () => {
    it('returns empty sets for acyclic DAG', () => {
      const compose: ComposeFile = {
        version: '1',
        tasks: {
          taskA: { prompt: 'a' },
          taskB: {
            prompt: 'b',
            depends_on: [{ task: 'taskA', condition: 'success' }],
          },
          taskC: {
            prompt: 'c',
            depends_on: [{ task: 'taskB', condition: 'success' }],
          },
        },
      }

      const result = validateDag(compose)

      expect(result.cycleNodes.size).toBe(0)
      expect(result.cycleEdges.size).toBe(0)
    })

    it('detects simple 2-node cycle', () => {
      const compose: ComposeFile = {
        version: '1',
        tasks: {
          taskA: {
            prompt: 'a',
            depends_on: [{ task: 'taskB', condition: 'success' }],
          },
          taskB: {
            prompt: 'b',
            depends_on: [{ task: 'taskA', condition: 'success' }],
          },
        },
      }

      const result = validateDag(compose)

      expect(result.cycleNodes.has('taskA')).toBe(true)
      expect(result.cycleNodes.has('taskB')).toBe(true)
      expect(result.cycleEdges.has('taskA->taskB')).toBe(true)
      expect(result.cycleEdges.has('taskB->taskA')).toBe(true)
    })

    it('detects self-referential cycle', () => {
      const compose: ComposeFile = {
        version: '1',
        tasks: {
          taskA: {
            prompt: 'a',
            depends_on: [{ task: 'taskA', condition: 'success' }],
          },
        },
      }

      const result = validateDag(compose)

      expect(result.cycleNodes.has('taskA')).toBe(true)
      expect(result.cycleEdges.has('taskA->taskA')).toBe(true)
    })

    it('detects 3-node cycle', () => {
      const compose: ComposeFile = {
        version: '1',
        tasks: {
          taskA: {
            prompt: 'a',
            depends_on: [{ task: 'taskC', condition: 'success' }],
          },
          taskB: {
            prompt: 'b',
            depends_on: [{ task: 'taskA', condition: 'success' }],
          },
          taskC: {
            prompt: 'c',
            depends_on: [{ task: 'taskB', condition: 'success' }],
          },
        },
      }

      const result = validateDag(compose)

      expect(result.cycleNodes.size).toBe(3)
      expect(result.cycleNodes.has('taskA')).toBe(true)
      expect(result.cycleNodes.has('taskB')).toBe(true)
      expect(result.cycleNodes.has('taskC')).toBe(true)
    })

    it('detects partial cycle (some nodes not in cycle)', () => {
      const compose: ComposeFile = {
        version: '1',
        tasks: {
          root: { prompt: 'root' },
          taskA: {
            prompt: 'a',
            depends_on: [
              { task: 'root', condition: 'success' },
              { task: 'taskB', condition: 'success' },
            ],
          },
          taskB: {
            prompt: 'b',
            depends_on: [{ task: 'taskA', condition: 'success' }],
          },
        },
      }

      const result = validateDag(compose)

      expect(result.cycleNodes.has('root')).toBe(false)
      expect(result.cycleNodes.has('taskA')).toBe(true)
      expect(result.cycleNodes.has('taskB')).toBe(true)
    })

    it('handles string dependencies (shorthand)', () => {
      const compose: ComposeFile = {
        version: '1',
        tasks: {
          taskA: {
            prompt: 'a',
            depends_on: ['taskB'],
          },
          taskB: {
            prompt: 'b',
            depends_on: ['taskA'],
          },
        },
      }

      const result = validateDag(compose)

      expect(result.cycleNodes.size).toBe(2)
    })
  })

  describe('orphan detection', () => {
    it('returns empty set when no pipelines defined', () => {
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

      const result = validateDag(compose)

      expect(result.orphanedTasks.size).toBe(0)
    })

    it('detects orphaned task with dependencies not in pipeline', () => {
      const compose: ComposeFile = {
        version: '1',
        tasks: {
          taskA: { prompt: 'a' },
          taskB: {
            prompt: 'b',
            depends_on: [{ task: 'taskA', condition: 'success' }],
          },
          taskC: { prompt: 'c' },
        },
        pipelines: {
          main: {
            iterations: 10,
            tasks: ['taskA', 'taskC'],
          },
        },
      }

      const result = validateDag(compose)

      expect(result.orphanedTasks.has('taskB')).toBe(true)
      expect(result.orphanedTasks.has('taskA')).toBe(false)
      expect(result.orphanedTasks.has('taskC')).toBe(false)
    })

    it('does not mark task as orphan if in at least one pipeline', () => {
      const compose: ComposeFile = {
        version: '1',
        tasks: {
          taskA: { prompt: 'a' },
          taskB: {
            prompt: 'b',
            depends_on: [{ task: 'taskA', condition: 'success' }],
          },
        },
        pipelines: {
          main: {
            iterations: 10,
            tasks: ['taskA', 'taskB'],
          },
        },
      }

      const result = validateDag(compose)

      expect(result.orphanedTasks.size).toBe(0)
    })

    it('does not mark standalone task (no deps) as orphan', () => {
      const compose: ComposeFile = {
        version: '1',
        tasks: {
          standalone: { prompt: 'standalone' },
          inPipeline: { prompt: 'inPipeline' },
        },
        pipelines: {
          main: {
            iterations: 10,
            tasks: ['inPipeline'],
          },
        },
      }

      const result = validateDag(compose)

      // standalone has no dependencies, so it's not considered orphaned
      expect(result.orphanedTasks.has('standalone')).toBe(false)
    })
  })

  describe('parallel task detection', () => {
    it('returns empty set when no parallel pipelines', () => {
      const compose: ComposeFile = {
        version: '1',
        tasks: {
          taskA: { prompt: 'a' },
        },
        pipelines: {
          main: {
            iterations: 10,
            parallelism: 1,
            tasks: ['taskA'],
          },
        },
      }

      const result = validateDag(compose)

      expect(result.parallelTasks.size).toBe(0)
    })

    it('detects tasks in pipelines with parallelism > 1', () => {
      const compose: ComposeFile = {
        version: '1',
        tasks: {
          taskA: { prompt: 'a' },
          taskB: { prompt: 'b' },
          taskC: { prompt: 'c' },
        },
        pipelines: {
          parallel: {
            iterations: 10,
            parallelism: 3,
            tasks: ['taskA', 'taskB'],
          },
          sequential: {
            iterations: 5,
            tasks: ['taskC'],
          },
        },
      }

      const result = validateDag(compose)

      expect(result.parallelTasks.has('taskA')).toBe(true)
      expect(result.parallelTasks.has('taskB')).toBe(true)
      expect(result.parallelTasks.has('taskC')).toBe(false)
    })

    it('handles pipelines without explicit parallelism (defaults to 1)', () => {
      const compose: ComposeFile = {
        version: '1',
        tasks: {
          taskA: { prompt: 'a' },
        },
        pipelines: {
          main: {
            iterations: 10,
            tasks: ['taskA'],
          },
        },
      }

      const result = validateDag(compose)

      expect(result.parallelTasks.size).toBe(0)
    })
  })

  describe('edge cases', () => {
    it('handles empty tasks object', () => {
      const compose: ComposeFile = {
        version: '1',
        tasks: {},
      }

      const result = validateDag(compose)

      expect(result.cycleNodes.size).toBe(0)
      expect(result.cycleEdges.size).toBe(0)
      expect(result.orphanedTasks.size).toBe(0)
      expect(result.parallelTasks.size).toBe(0)
    })

    it('handles undefined tasks', () => {
      const compose = {
        version: '1',
      } as ComposeFile

      const result = validateDag(compose)

      expect(result.cycleNodes.size).toBe(0)
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

      const result = validateDag(compose)

      // Should not crash or include non-existent task
      expect(result.cycleNodes.size).toBe(0)
    })

    it('handles complex DAG with multiple validation issues', () => {
      const compose: ComposeFile = {
        version: '1',
        tasks: {
          root: { prompt: 'root' },
          cycleA: {
            prompt: 'a',
            depends_on: [{ task: 'cycleB', condition: 'success' }],
          },
          cycleB: {
            prompt: 'b',
            depends_on: [{ task: 'cycleA', condition: 'success' }],
          },
          orphan: {
            prompt: 'orphan',
            depends_on: [{ task: 'root', condition: 'success' }],
          },
          parallel: { prompt: 'parallel' },
        },
        pipelines: {
          parallelPipeline: {
            iterations: 5,
            parallelism: 2,
            tasks: ['parallel', 'root'],
          },
        },
      }

      const result = validateDag(compose)

      // Check cycles
      expect(result.cycleNodes.has('cycleA')).toBe(true)
      expect(result.cycleNodes.has('cycleB')).toBe(true)
      expect(result.cycleNodes.has('root')).toBe(false)

      // Check orphans
      expect(result.orphanedTasks.has('orphan')).toBe(true)
      expect(result.orphanedTasks.has('cycleA')).toBe(true) // has deps but not in pipeline
      expect(result.orphanedTasks.has('cycleB')).toBe(true)

      // Check parallel
      expect(result.parallelTasks.has('parallel')).toBe(true)
      expect(result.parallelTasks.has('root')).toBe(true)
    })
  })
})
