import type { ComposeFile, TaskDef, TaskDependency, PipelineDef } from './yamlParser'

export interface TaskFormData {
  promptType: 'prompt' | 'prompt-file' | 'prompt-string'
  promptValue: string
  model: string
  prefix: string
  suffix: string
  dependencies: TaskDependency[]
}

export function applyTaskEdits(compose: ComposeFile, taskName: string, form: TaskFormData): ComposeFile {
  const updated = structuredClone(compose)
  const task: TaskDef = updated.tasks[taskName] ?? {}

  // Clear all prompt fields, then set the one matching the form
  delete task.prompt
  delete task['prompt-file']
  delete task['prompt-string']

  if (form.promptValue.trim()) {
    if (form.promptType === 'prompt') {
      task.prompt = form.promptValue.trim()
    } else if (form.promptType === 'prompt-file') {
      task['prompt-file'] = form.promptValue.trim()
    } else {
      task['prompt-string'] = form.promptValue.trim()
    }
  }

  // Model — empty string means inherit (remove key)
  if (form.model && form.model !== 'inherit') {
    task.model = form.model
  } else {
    delete task.model
  }

  // Prefix / suffix
  if (form.prefix.trim()) {
    task.prefix = form.prefix.trim()
  } else {
    delete task.prefix
  }

  if (form.suffix.trim()) {
    task.suffix = form.suffix.trim()
  } else {
    delete task.suffix
  }

  // Dependencies
  if (form.dependencies.length > 0) {
    task.depends_on = form.dependencies.map(dep => {
      if (dep.condition === 'success') {
        return dep.task // Simple string form for the default condition
      }
      return { task: dep.task, condition: dep.condition }
    })
  } else {
    delete task.depends_on
  }

  updated.tasks[taskName] = task
  return updated
}

export function addDependency(
  compose: ComposeFile,
  targetTask: string,
  sourceTask: string,
  condition: TaskDependency['condition'],
): ComposeFile {
  const updated = structuredClone(compose)
  const task = updated.tasks[targetTask]
  if (!task) return updated

  if (!task.depends_on) task.depends_on = []

  // Prevent duplicates (same source task)
  const existing = task.depends_on.find(dep =>
    typeof dep === 'string' ? dep === sourceTask : dep.task === sourceTask
  )
  if (existing) return updated

  // Use simple string form for "success" (default), object form otherwise
  if (condition === 'success') {
    task.depends_on.push(sourceTask)
  } else {
    task.depends_on.push({ task: sourceTask, condition })
  }

  return updated
}

export function applyPipelineEdits(
  compose: ComposeFile,
  pipelineName: string,
  updates: { iterations?: number; parallelism?: number; tasks?: string[] },
): ComposeFile {
  const updated = structuredClone(compose)
  if (!updated.pipelines) updated.pipelines = {}

  const pipeline: PipelineDef = updated.pipelines[pipelineName] ?? {}

  if (updates.iterations !== undefined && updates.iterations > 0) {
    pipeline.iterations = updates.iterations
  } else {
    delete pipeline.iterations
  }
  if (updates.parallelism !== undefined && updates.parallelism > 0) {
    pipeline.parallelism = updates.parallelism
  } else {
    delete pipeline.parallelism
  }
  if (updates.tasks !== undefined) {
    if (updates.tasks.length > 0) {
      pipeline.tasks = updates.tasks
    } else {
      delete pipeline.tasks
    }
  }

  updated.pipelines[pipelineName] = pipeline
  return updated
}

export function deletePipeline(compose: ComposeFile, pipelineName: string): ComposeFile {
  const updated = structuredClone(compose)
  if (updated.pipelines) {
    delete updated.pipelines[pipelineName]
    if (Object.keys(updated.pipelines).length === 0) {
      delete updated.pipelines
    }
  }
  return updated
}

