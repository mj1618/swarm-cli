# Implement `swarm config set-model` command

## Problem

The README documents `swarm config set-model` as a working command:

```bash
# Set the default model for claude-code backend
swarm config set-model opus

# Set the default model for cursor backend
swarm config set-model opus-4.5-thinking
```

However, this command does not exist in the codebase. The `cmd/config.go` file only implements:
- `swarm config show`
- `swarm config path`
- `swarm config set-backend`

When users try to run `swarm config set-model opus`, they get an error:

```
Error: unknown command "set-model" for "swarm config"
```

This is a documentation/implementation mismatch that confuses users.

## Solution

Add a `set-model` subcommand to `swarm config` that updates the default model in the config file, following the same pattern as the existing `set-backend` command.

### Proposed API

```bash
# Set default model (updates project config by default)
swarm config set-model opus
swarm config set-model claude-sonnet-4-20250514

# Update global config instead
swarm config set-model opus --global
```

## Files to change

- `cmd/config.go` - add `configSetModelCmd` command

## Implementation details

### config.go additions

```go
var configSetModelCmd = &cobra.Command{
    Use:   "set-model [model]",
    Short: "Set the default model",
    Long: `Set the default model for agent runs.

The model is used when no --model flag is specified on the run command.
By default, updates the project config (swarm/.swarm.toml). Use --global to update the global config.`,
    Example: `  # Set default model for project
  swarm config set-model opus

  # Set model in global config
  swarm config set-model claude-sonnet-4-20250514 --global`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        model := args[0]

        // Determine config path
        var configPath string
        var err error
        if configGlobal {
            configPath, err = config.GlobalConfigPath()
            if err != nil {
                return fmt.Errorf("failed to determine global config path: %w", err)
            }
        } else {
            configPath = config.ProjectConfigPath()
        }

        // Load existing config or start with defaults
        cfg := config.DefaultConfig()
        if _, err := os.Stat(configPath); err == nil {
            loadedCfg, err := config.Load()
            if err != nil {
                return fmt.Errorf("failed to load existing config: %w", err)
            }
            cfg = loadedCfg
        }

        // Update the model
        cfg.Model = model

        // Create parent directory if needed
        dir := filepath.Dir(configPath)
        if err := os.MkdirAll(dir, 0755); err != nil {
            return fmt.Errorf("failed to create directory %s: %w", dir, err)
        }

        // Write updated config
        if err := os.WriteFile(configPath, []byte(cfg.ToTOML()), 0644); err != nil {
            return fmt.Errorf("failed to write config file: %w", err)
        }

        fmt.Printf("Default model set to %q\n", model)
        fmt.Printf("Updated config: %s\n", configPath)
        return nil
    },
}

// In init():
configCmd.AddCommand(configSetModelCmd)
configSetModelCmd.Flags().BoolVarP(&configGlobal, "global", "g", false, "Update global config instead of project config")
```

### Output examples

Success:
```
$ swarm config set-model opus
Default model set to "opus"
Updated config: /path/to/project/swarm/.swarm.toml
```

With global flag:
```
$ swarm config set-model claude-sonnet-4-20250514 --global
Default model set to "claude-sonnet-4-20250514"
Updated config: /home/user/.config/swarm/config.toml
```

Verify with show:
```
$ swarm config show
# Effective configuration (merged from all sources)

backend = "claude-code"
model = "opus"
iterations = 1
...
```

## Edge cases

1. **Empty model string**: Reject with an error - model must be non-empty.

2. **Invalid model name**: Don't validate model names - let the backend CLI handle validation. Different backends support different models, and new models may be added over time.

3. **Config file doesn't exist**: Create it with just the model setting (following `set-backend` behavior).

4. **Preserve other settings**: When updating model, don't overwrite other existing config values like `backend` or `iterations`.

## Acceptance criteria

- `swarm config set-model <model>` updates the project config file
- `swarm config set-model <model> --global` updates the global config file
- Existing config values (backend, iterations, etc.) are preserved
- Config file is created if it doesn't exist
- `swarm config show` reflects the updated model after running the command
- Command follows the same pattern/style as `set-backend`
- `swarm run` without `--model` uses the configured default model

---

## Completion Notes (Agent cd59a862)

**Status:** Completed

**Changes made:**
- Added `configSetModelCmd` to `cmd/config.go` following the existing `set-backend` pattern
- Command supports `-g`/`--global` flag to update global config instead of project config
- Empty model strings are rejected with a clear error message
- Model names are not validated (left to backend CLI to validate)
- Existing config values are preserved when updating model

**Testing performed:**
- `swarm config set-model --help` - displays correct help text
- `swarm config set-model sonnet-test` - successfully updates project config
- `swarm config set-model ""` - correctly rejects empty model with error
- `swarm config show` - shows updated model after running set-model

**Note:** There is a pre-existing test failure in `TestLoadWithProjectOverride` due to test isolation issues - the test doesn't mock the global config path, so it's affected by the user's global config file at `~/Library/Application Support/swarm/config.toml`. This is unrelated to this implementation.
