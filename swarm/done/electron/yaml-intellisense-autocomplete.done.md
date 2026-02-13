# Task: Add YAML IntelliSense and Autocomplete for swarm.yaml

**Phase:** 5 - Polish
**Priority:** Medium

## Goal

When editing `swarm.yaml` in the Monaco editor, provide IntelliSense features:

1. **Autocomplete for task names** in `depends_on[].task` fields — suggest existing task names
2. **Autocomplete for prompt names** in `prompt:` fields — suggest filenames from `swarm/prompts/`
3. **Hover documentation** for known fields (e.g., `iterations`, `parallelism`, `condition`, `prefix`, `suffix`)
4. **Schema validation markers** — red squiggles on unknown keys or invalid values (e.g., non-numeric `iterations`, invalid `condition` values)

## What to Build

### 1. Monaco Completion Provider for YAML

Register a `CompletionItemProvider` for the `yaml` language in MonacoFileEditor. The provider should:

- Detect when the cursor is inside a `depends_on` block's `task:` value → suggest all task names from the same file
- Detect when the cursor is at a `prompt:` value → fetch prompt file names from `swarm/prompts/` via the existing `window.fs.readdir()` IPC and suggest them (stripped of `.md` extension)
- Detect when the cursor is at a `condition:` value → suggest `success`, `failure`, `any`, `always`
- Detect when the cursor is at a `model:` value → suggest `opus`, `sonnet`, `haiku`

### 2. Monaco Hover Provider for YAML

Register a `HoverProvider` for the `yaml` language that shows documentation when hovering known keys:

| Key | Documentation |
|-----|---------------|
| `prompt` | "Name of a prompt file from swarm/prompts/ (without .md extension)" |
| `prompt-file` | "Path to a prompt file relative to the project root" |
| `prompt-string` | "Inline prompt string" |
| `model` | "Model to use for this task (overrides default)" |
| `prefix` | "Text prepended to the prompt before sending to the agent" |
| `suffix` | "Text appended to the prompt before sending to the agent" |
| `iterations` | "Number of iterations to run for this pipeline" |
| `parallelism` | "Maximum concurrent agents for this pipeline" |
| `depends_on` | "List of task dependencies with conditions" |
| `condition` | "When to trigger: success, failure, any, or always" |

### 3. Monaco Diagnostic Markers for YAML

After each edit, parse the YAML and set diagnostic markers via `monaco.editor.setModelMarkers()`:

- Unknown top-level keys (not `version`, `tasks`, `pipelines`)
- Unknown task-level keys (not `prompt`, `prompt-file`, `prompt-string`, `model`, `prefix`, `suffix`, `depends_on`)
- Invalid `condition` values (not one of `success`, `failure`, `any`, `always`)
- `depends_on[].task` referencing a non-existent task name
- Non-numeric `iterations` or `parallelism` values

## Files

### Modify
- **`electron/src/renderer/components/MonacoFileEditor.tsx`**
  - Register completion provider, hover provider, and diagnostics when the file is a YAML file (specifically when the path ends in `swarm.yaml` or is in the swarm directory)
  - Use `editor.onDidChangeModelContent` to re-run validation after edits

### Possibly Create
- **`electron/src/renderer/lib/yamlIntellisense.ts`** (optional)
  - Extract the completion/hover/validation logic into a separate module to keep MonacoFileEditor manageable

## Dependencies

- Monaco editor integration (completed)
- File tree / `window.fs.readdir()` IPC (completed)
- `yamlParser.ts` parsing utilities (completed)

## Acceptance Criteria

1. When editing `swarm.yaml`, typing in a `prompt:` field shows autocomplete suggestions from `swarm/prompts/` directory
2. When editing `depends_on`, typing in a `task:` field shows autocomplete suggestions of other task names in the file
3. `condition:` fields suggest `success`, `failure`, `any`, `always`
4. Hovering over known YAML keys shows documentation tooltips
5. Invalid `condition` values and references to non-existent tasks show red squiggles
6. App builds with `npm run build`
7. No regressions to the existing Monaco editor behavior for non-YAML files

## Notes

- Monaco's `registerCompletionItemProvider` and `registerHoverProvider` are registered globally for a language, so use the `model.uri` or file path to scope behavior to swarm YAML files only
- The existing `parseComposeFile()` from `yamlParser.ts` can be reused for validation
- Prompt names can be fetched via `window.fs.readdir(swarmRoot + '/prompts')` — this IPC handler already exists
- Keep the YAML position detection simple — use line-based heuristics (check indentation + preceding key names) rather than building a full YAML AST cursor
- Reference: Monaco CompletionItemProvider docs — `provideCompletionItems(model, position, context, token)`

## Completion Notes

Implemented all 4 IntelliSense features:

1. **Autocomplete**: Created `yamlIntellisense.ts` with a `CompletionItemProvider` that provides context-aware suggestions for `prompt:` (fetches prompt names from `window.fs.listprompts()`), `condition:` (success/failure/any/always), `model:` (opus/sonnet/haiku), and `depends_on` task references (extracts task names from the YAML content).

2. **Hover docs**: `HoverProvider` shows documentation tooltips for all known swarm YAML keys (prompt, prompt-file, prompt-string, model, prefix, suffix, iterations, parallelism, depends_on, condition, version, tasks, pipelines).

3. **Validation markers**: `validateSwarmYaml()` runs on mount and on every content change, producing diagnostic markers for unknown top-level keys, unknown task-level keys, invalid condition values, non-existent task references in depends_on, unknown pipeline keys, and non-numeric iterations/parallelism values.

4. **Integration**: Providers are registered once globally for the `yaml` language in `MonacoFileEditor.tsx`. Validation is scoped to files matching `swarm.yaml`/`swarm.yml` via the `isSwarmYaml()` check.

Files modified:
- `electron/src/renderer/lib/yamlIntellisense.ts` (new)
- `electron/src/renderer/components/MonacoFileEditor.tsx` (modified)
