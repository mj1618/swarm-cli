import { describe, it, expect } from 'vitest'
import {
  applyTaskEdits,
  addDependency,
  applyPipelineEdits,
  deletePipeline,
  deleteTask,
  deleteEdge,
  type TaskFormData,
} from '../yamlWriter'
import type { ComposeFile } from '../yamlParser'

function createBaseCompose(): ComposeFile {
  return {
    version: '1',
    tasks: {
      taskA: { prompt: 'a' },
      taskB: { prompt: 'b' },
    },
  }
}

describe('applyTaskEdits', () => {
  it('sets prompt field for prompt type', () => {
    const compose = createBaseCompose()
    const form: TaskFormData = {
      promptType: 'prompt',
      promptValue: 'myPrompt',
      model: '',
      prefix: '',
      suffix: '',
      dependencies: [],
    }

    const result = applyTaskEdits(compose, 'taskA', form)

    expect(result.tasks.taskA.prompt).toBe('myPrompt')
    expect(result.tasks.taskA['prompt-file']).toBeUndefined()
    expect(result.tasks.taskA['prompt-string']).toBeUndefined()
  })

  it('sets prompt-file field for prompt-file type', () => {
    const compose = createBaseCompose()
    const form: TaskFormData = {
      promptType: 'prompt-file',
      promptValue: './path/to/prompt.md',
      model: '',
      prefix: '',
      suffix: '',
      dependencies: [],
    }

    const result = applyTaskEdits(compose, 'taskA', form)

    expect(result.tasks.taskA['prompt-file']).toBe('./path/to/prompt.md')
    expect(result.tasks.taskA.prompt).toBeUndefined()
    expect(result.tasks.taskA['prompt-string']).toBeUndefined()
  })

  it('sets prompt-string field for prompt-string type', () => {
    const compose = createBaseCompose()
    const form: TaskFormData = {
      promptType: 'prompt-string',
      promptValue: 'Do something cool',
      model: '',
      prefix: '',
      suffix: '',
      dependencies: [],
    }

    const result = applyTaskEdits(compose, 'taskA', form)

    expect(result.tasks.taskA['prompt-string']).toBe('Do something cool')
    expect(result.tasks.taskA.prompt).toBeUndefined()
    expect(result.tasks.taskA['prompt-file']).toBeUndefined()
  })

  it('clears prompt fields when value is empty', () => {
    const compose = createBaseCompose()
    compose.tasks.taskA = { prompt: 'oldPrompt' }
    const form: TaskFormData = {
      promptType: 'prompt',
      promptValue: '   ',
      model: '',
      prefix: '',
      suffix: '',
      dependencies: [],
    }

    const result = applyTaskEdits(compose, 'taskA', form)

    expect(result.tasks.taskA.prompt).toBeUndefined()
    expect(result.tasks.taskA['prompt-file']).toBeUndefined()
    expect(result.tasks.taskA['prompt-string']).toBeUndefined()
  })

  it('clears previous prompt type when switching types', () => {
    const compose = createBaseCompose()
    compose.tasks.taskA = { prompt: 'oldPrompt', 'prompt-file': './old.md' }
    const form: TaskFormData = {
      promptType: 'prompt-string',
      promptValue: 'inline content',
      model: '',
      prefix: '',
      suffix: '',
      dependencies: [],
    }

    const result = applyTaskEdits(compose, 'taskA', form)

    expect(result.tasks.taskA['prompt-string']).toBe('inline content')
    expect(result.tasks.taskA.prompt).toBeUndefined()
    expect(result.tasks.taskA['prompt-file']).toBeUndefined()
  })

  it('sets model when provided', () => {
    const compose = createBaseCompose()
    const form: TaskFormData = {
      promptType: 'prompt',
      promptValue: 'myPrompt',
      model: 'opus',
      prefix: '',
      suffix: '',
      dependencies: [],
    }

    const result = applyTaskEdits(compose, 'taskA', form)

    expect(result.tasks.taskA.model).toBe('opus')
  })

  it('removes model when set to inherit', () => {
    const compose = createBaseCompose()
    compose.tasks.taskA.model = 'opus'
    const form: TaskFormData = {
      promptType: 'prompt',
      promptValue: 'myPrompt',
      model: 'inherit',
      prefix: '',
      suffix: '',
      dependencies: [],
    }

    const result = applyTaskEdits(compose, 'taskA', form)

    expect(result.tasks.taskA.model).toBeUndefined()
  })

  it('removes model when empty string', () => {
    const compose = createBaseCompose()
    compose.tasks.taskA.model = 'sonnet'
    const form: TaskFormData = {
      promptType: 'prompt',
      promptValue: 'myPrompt',
      model: '',
      prefix: '',
      suffix: '',
      dependencies: [],
    }

    const result = applyTaskEdits(compose, 'taskA', form)

    expect(result.tasks.taskA.model).toBeUndefined()
  })

  it('sets prefix when provided', () => {
    const compose = createBaseCompose()
    const form: TaskFormData = {
      promptType: 'prompt',
      promptValue: 'myPrompt',
      model: '',
      prefix: 'Before: ',
      suffix: '',
      dependencies: [],
    }

    const result = applyTaskEdits(compose, 'taskA', form)

    expect(result.tasks.taskA.prefix).toBe('Before:')
  })

  it('removes prefix when empty or whitespace', () => {
    const compose = createBaseCompose()
    compose.tasks.taskA.prefix = 'oldPrefix'
    const form: TaskFormData = {
      promptType: 'prompt',
      promptValue: 'myPrompt',
      model: '',
      prefix: '   ',
      suffix: '',
      dependencies: [],
    }

    const result = applyTaskEdits(compose, 'taskA', form)

    expect(result.tasks.taskA.prefix).toBeUndefined()
  })

  it('sets suffix when provided', () => {
    const compose = createBaseCompose()
    const form: TaskFormData = {
      promptType: 'prompt',
      promptValue: 'myPrompt',
      model: '',
      prefix: '',
      suffix: ' After',
      dependencies: [],
    }

    const result = applyTaskEdits(compose, 'taskA', form)

    expect(result.tasks.taskA.suffix).toBe('After')
  })

  it('removes suffix when empty or whitespace', () => {
    const compose = createBaseCompose()
    compose.tasks.taskA.suffix = 'oldSuffix'
    const form: TaskFormData = {
      promptType: 'prompt',
      promptValue: 'myPrompt',
      model: '',
      prefix: '',
      suffix: '',
      dependencies: [],
    }

    const result = applyTaskEdits(compose, 'taskA', form)

    expect(result.tasks.taskA.suffix).toBeUndefined()
  })

  it('sets dependencies with string shorthand for success condition', () => {
    const compose = createBaseCompose()
    const form: TaskFormData = {
      promptType: 'prompt',
      promptValue: 'myPrompt',
      model: '',
      prefix: '',
      suffix: '',
      dependencies: [{ task: 'taskB', condition: 'success' }],
    }

    const result = applyTaskEdits(compose, 'taskA', form)

    expect(result.tasks.taskA.depends_on).toEqual(['taskB'])
  })

  it('sets dependencies with object form for non-success conditions', () => {
    const compose = createBaseCompose()
    const form: TaskFormData = {
      promptType: 'prompt',
      promptValue: 'myPrompt',
      model: '',
      prefix: '',
      suffix: '',
      dependencies: [
        { task: 'taskB', condition: 'failure' },
        { task: 'taskA', condition: 'any' },
      ],
    }

    const result = applyTaskEdits(compose, 'taskB', form)

    expect(result.tasks.taskB.depends_on).toEqual([
      { task: 'taskB', condition: 'failure' },
      { task: 'taskA', condition: 'any' },
    ])
  })

  it('removes depends_on when dependencies array is empty', () => {
    const compose = createBaseCompose()
    compose.tasks.taskA.depends_on = ['taskB']
    const form: TaskFormData = {
      promptType: 'prompt',
      promptValue: 'myPrompt',
      model: '',
      prefix: '',
      suffix: '',
      dependencies: [],
    }

    const result = applyTaskEdits(compose, 'taskA', form)

    expect(result.tasks.taskA.depends_on).toBeUndefined()
  })

  it('creates task if it does not exist', () => {
    const compose = createBaseCompose()
    const form: TaskFormData = {
      promptType: 'prompt',
      promptValue: 'newPrompt',
      model: 'opus',
      prefix: '',
      suffix: '',
      dependencies: [],
    }

    const result = applyTaskEdits(compose, 'newTask', form)

    expect(result.tasks.newTask).toBeDefined()
    expect(result.tasks.newTask.prompt).toBe('newPrompt')
    expect(result.tasks.newTask.model).toBe('opus')
  })

  it('does not mutate original compose', () => {
    const compose = createBaseCompose()
    const form: TaskFormData = {
      promptType: 'prompt',
      promptValue: 'changed',
      model: 'opus',
      prefix: '',
      suffix: '',
      dependencies: [],
    }

    const result = applyTaskEdits(compose, 'taskA', form)

    expect(compose.tasks.taskA.prompt).toBe('a')
    expect(result.tasks.taskA.prompt).toBe('changed')
  })
})

describe('addDependency', () => {
  it('adds a dependency with success condition as string', () => {
    const compose = createBaseCompose()

    const result = addDependency(compose, 'taskB', 'taskA', 'success')

    expect(result.tasks.taskB.depends_on).toEqual(['taskA'])
  })

  it('adds a dependency with non-success condition as object', () => {
    const compose = createBaseCompose()

    const result = addDependency(compose, 'taskB', 'taskA', 'failure')

    expect(result.tasks.taskB.depends_on).toEqual([{ task: 'taskA', condition: 'failure' }])
  })

  it('adds dependency with any condition', () => {
    const compose = createBaseCompose()

    const result = addDependency(compose, 'taskB', 'taskA', 'any')

    expect(result.tasks.taskB.depends_on).toEqual([{ task: 'taskA', condition: 'any' }])
  })

  it('adds dependency with always condition', () => {
    const compose = createBaseCompose()

    const result = addDependency(compose, 'taskB', 'taskA', 'always')

    expect(result.tasks.taskB.depends_on).toEqual([{ task: 'taskA', condition: 'always' }])
  })

  it('prevents duplicate dependencies (string form)', () => {
    const compose = createBaseCompose()
    compose.tasks.taskB.depends_on = ['taskA']

    const result = addDependency(compose, 'taskB', 'taskA', 'success')

    expect(result.tasks.taskB.depends_on).toEqual(['taskA'])
    expect(result.tasks.taskB.depends_on).toHaveLength(1)
  })

  it('prevents duplicate dependencies (object form)', () => {
    const compose = createBaseCompose()
    compose.tasks.taskB.depends_on = [{ task: 'taskA', condition: 'failure' }]

    const result = addDependency(compose, 'taskB', 'taskA', 'success')

    expect(result.tasks.taskB.depends_on).toHaveLength(1)
  })

  it('appends to existing dependencies', () => {
    const compose: ComposeFile = {
      version: '1',
      tasks: {
        taskA: { prompt: 'a' },
        taskB: { prompt: 'b' },
        taskC: { prompt: 'c', depends_on: ['taskA'] },
      },
    }

    const result = addDependency(compose, 'taskC', 'taskB', 'success')

    expect(result.tasks.taskC.depends_on).toEqual(['taskA', 'taskB'])
  })

  it('creates depends_on array if not present', () => {
    const compose = createBaseCompose()
    expect(compose.tasks.taskA.depends_on).toBeUndefined()

    const result = addDependency(compose, 'taskA', 'taskB', 'success')

    expect(result.tasks.taskA.depends_on).toEqual(['taskB'])
  })

  it('returns unchanged compose if target task does not exist', () => {
    const compose = createBaseCompose()

    const result = addDependency(compose, 'nonExistent', 'taskA', 'success')

    expect(result.tasks).toEqual(compose.tasks)
    expect(result.tasks.nonExistent).toBeUndefined()
  })

  it('does not mutate original compose', () => {
    const compose = createBaseCompose()

    const result = addDependency(compose, 'taskB', 'taskA', 'success')

    expect(compose.tasks.taskB.depends_on).toBeUndefined()
    expect(result.tasks.taskB.depends_on).toEqual(['taskA'])
  })
})

describe('applyPipelineEdits', () => {
  it('sets iterations when provided', () => {
    const compose = createBaseCompose()

    const result = applyPipelineEdits(compose, 'main', { iterations: 10 })

    expect(result.pipelines?.main.iterations).toBe(10)
  })

  it('removes iterations when set to 0 or undefined', () => {
    const compose = createBaseCompose()
    compose.pipelines = { main: { iterations: 10 } }

    const result = applyPipelineEdits(compose, 'main', { iterations: 0 })

    expect(result.pipelines?.main.iterations).toBeUndefined()
  })

  it('sets parallelism when provided', () => {
    const compose = createBaseCompose()

    const result = applyPipelineEdits(compose, 'main', { parallelism: 4 })

    expect(result.pipelines?.main.parallelism).toBe(4)
  })

  it('removes parallelism when set to 0 or undefined', () => {
    const compose = createBaseCompose()
    compose.pipelines = { main: { parallelism: 4 } }

    const result = applyPipelineEdits(compose, 'main', { parallelism: 0 })

    expect(result.pipelines?.main.parallelism).toBeUndefined()
  })

  it('sets tasks list when provided', () => {
    const compose = createBaseCompose()

    const result = applyPipelineEdits(compose, 'main', { tasks: ['taskA', 'taskB'] })

    expect(result.pipelines?.main.tasks).toEqual(['taskA', 'taskB'])
  })

  it('removes tasks when set to empty array', () => {
    const compose = createBaseCompose()
    compose.pipelines = { main: { tasks: ['taskA'] } }

    const result = applyPipelineEdits(compose, 'main', { tasks: [] })

    expect(result.pipelines?.main.tasks).toBeUndefined()
  })

  it('creates pipeline if it does not exist', () => {
    const compose = createBaseCompose()

    const result = applyPipelineEdits(compose, 'newPipeline', { iterations: 5 })

    expect(result.pipelines?.newPipeline).toBeDefined()
    expect(result.pipelines?.newPipeline.iterations).toBe(5)
  })

  it('creates pipelines object if not present', () => {
    const compose = createBaseCompose()
    expect(compose.pipelines).toBeUndefined()

    const result = applyPipelineEdits(compose, 'main', { iterations: 10 })

    expect(result.pipelines).toBeDefined()
    expect(result.pipelines?.main.iterations).toBe(10)
  })

  it('clears iterations/parallelism when not provided but preserves tasks', () => {
    const compose = createBaseCompose()
    compose.pipelines = { main: { iterations: 5, parallelism: 2, tasks: ['taskA'] } }

    // iterations/parallelism get cleared when not provided, but tasks is preserved
    const result = applyPipelineEdits(compose, 'main', { iterations: 10 })

    expect(result.pipelines?.main.iterations).toBe(10)
    expect(result.pipelines?.main.parallelism).toBeUndefined()
    expect(result.pipelines?.main.tasks).toEqual(['taskA']) // tasks preserved when not in updates
  })

  it('can update multiple fields at once', () => {
    const compose = createBaseCompose()
    compose.pipelines = { main: { iterations: 5 } }

    const result = applyPipelineEdits(compose, 'main', {
      iterations: 10,
      parallelism: 4,
      tasks: ['taskA', 'taskB'],
    })

    expect(result.pipelines?.main.iterations).toBe(10)
    expect(result.pipelines?.main.parallelism).toBe(4)
    expect(result.pipelines?.main.tasks).toEqual(['taskA', 'taskB'])
  })

  it('does not mutate original compose', () => {
    const compose = createBaseCompose()

    const result = applyPipelineEdits(compose, 'main', { iterations: 10 })

    expect(compose.pipelines).toBeUndefined()
    expect(result.pipelines?.main.iterations).toBe(10)
  })
})

describe('deletePipeline', () => {
  it('removes the specified pipeline', () => {
    const compose = createBaseCompose()
    compose.pipelines = { main: { iterations: 10 }, other: { iterations: 5 } }

    const result = deletePipeline(compose, 'main')

    expect(result.pipelines?.main).toBeUndefined()
    expect(result.pipelines?.other).toBeDefined()
  })

  it('removes pipelines object when last pipeline is deleted', () => {
    const compose = createBaseCompose()
    compose.pipelines = { main: { iterations: 10 } }

    const result = deletePipeline(compose, 'main')

    expect(result.pipelines).toBeUndefined()
  })

  it('handles non-existent pipeline gracefully', () => {
    const compose = createBaseCompose()
    compose.pipelines = { main: { iterations: 10 } }

    const result = deletePipeline(compose, 'nonExistent')

    expect(result.pipelines?.main).toBeDefined()
  })

  it('handles missing pipelines object gracefully', () => {
    const compose = createBaseCompose()

    const result = deletePipeline(compose, 'main')

    expect(result.pipelines).toBeUndefined()
  })

  it('does not mutate original compose', () => {
    const compose = createBaseCompose()
    compose.pipelines = { main: { iterations: 10 } }

    const result = deletePipeline(compose, 'main')

    expect(compose.pipelines?.main).toBeDefined()
    expect(result.pipelines).toBeUndefined()
  })
})

describe('deleteTask', () => {
  it('removes the specified task', () => {
    const compose = createBaseCompose()

    const result = deleteTask(compose, 'taskA')

    expect(result.tasks.taskA).toBeUndefined()
    expect(result.tasks.taskB).toBeDefined()
  })

  it('removes references from other tasks depends_on (string form)', () => {
    const compose = createBaseCompose()
    compose.tasks.taskB.depends_on = ['taskA']

    const result = deleteTask(compose, 'taskA')

    expect(result.tasks.taskB.depends_on).toBeUndefined()
  })

  it('removes references from other tasks depends_on (object form)', () => {
    const compose = createBaseCompose()
    compose.tasks.taskB.depends_on = [{ task: 'taskA', condition: 'failure' }]

    const result = deleteTask(compose, 'taskA')

    expect(result.tasks.taskB.depends_on).toBeUndefined()
  })

  it('keeps other dependencies when removing one', () => {
    const compose: ComposeFile = {
      version: '1',
      tasks: {
        taskA: { prompt: 'a' },
        taskB: { prompt: 'b' },
        taskC: { prompt: 'c', depends_on: ['taskA', 'taskB'] },
      },
    }

    const result = deleteTask(compose, 'taskA')

    expect(result.tasks.taskC.depends_on).toEqual(['taskB'])
  })

  it('removes task from pipeline task lists', () => {
    const compose = createBaseCompose()
    compose.pipelines = { main: { tasks: ['taskA', 'taskB'] } }

    const result = deleteTask(compose, 'taskA')

    expect(result.pipelines?.main.tasks).toEqual(['taskB'])
  })

  it('removes tasks array from pipeline when empty', () => {
    const compose = createBaseCompose()
    compose.pipelines = { main: { iterations: 10, tasks: ['taskA'] } }

    const result = deleteTask(compose, 'taskA')

    expect(result.pipelines?.main.tasks).toBeUndefined()
    expect(result.pipelines?.main.iterations).toBe(10)
  })

  it('handles non-existent task gracefully', () => {
    const compose = createBaseCompose()

    const result = deleteTask(compose, 'nonExistent')

    expect(Object.keys(result.tasks)).toHaveLength(2)
  })

  it('does not mutate original compose', () => {
    const compose = createBaseCompose()
    compose.tasks.taskB.depends_on = ['taskA']

    const result = deleteTask(compose, 'taskA')

    expect(compose.tasks.taskA).toBeDefined()
    expect(compose.tasks.taskB.depends_on).toEqual(['taskA'])
    expect(result.tasks.taskA).toBeUndefined()
    expect(result.tasks.taskB.depends_on).toBeUndefined()
  })
})

describe('deleteEdge', () => {
  it('removes edge from depends_on (string form)', () => {
    const compose = createBaseCompose()
    compose.tasks.taskB.depends_on = ['taskA']

    const result = deleteEdge(compose, 'taskA', 'taskB')

    expect(result.tasks.taskB.depends_on).toBeUndefined()
  })

  it('removes edge from depends_on (object form)', () => {
    const compose = createBaseCompose()
    compose.tasks.taskB.depends_on = [{ task: 'taskA', condition: 'failure' }]

    const result = deleteEdge(compose, 'taskA', 'taskB')

    expect(result.tasks.taskB.depends_on).toBeUndefined()
  })

  it('keeps other dependencies when removing one edge', () => {
    const compose: ComposeFile = {
      version: '1',
      tasks: {
        taskA: { prompt: 'a' },
        taskB: { prompt: 'b' },
        taskC: { prompt: 'c', depends_on: ['taskA', { task: 'taskB', condition: 'any' }] },
      },
    }

    const result = deleteEdge(compose, 'taskA', 'taskC')

    expect(result.tasks.taskC.depends_on).toEqual([{ task: 'taskB', condition: 'any' }])
  })

  it('handles non-existent target task gracefully', () => {
    const compose = createBaseCompose()

    const result = deleteEdge(compose, 'taskA', 'nonExistent')

    expect(result).toEqual(compose)
  })

  it('handles target task without depends_on gracefully', () => {
    const compose = createBaseCompose()

    const result = deleteEdge(compose, 'taskA', 'taskB')

    expect(result.tasks.taskB.depends_on).toBeUndefined()
  })

  it('handles non-existent edge gracefully', () => {
    const compose = createBaseCompose()
    compose.tasks.taskB.depends_on = ['taskA']

    const result = deleteEdge(compose, 'nonExistent', 'taskB')

    expect(result.tasks.taskB.depends_on).toEqual(['taskA'])
  })

  it('does not mutate original compose', () => {
    const compose = createBaseCompose()
    compose.tasks.taskB.depends_on = ['taskA']

    const result = deleteEdge(compose, 'taskA', 'taskB')

    expect(compose.tasks.taskB.depends_on).toEqual(['taskA'])
    expect(result.tasks.taskB.depends_on).toBeUndefined()
  })
})
