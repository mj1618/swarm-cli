# Add `swarm completion` command for shell completions

## Problem

Users who use swarm-cli frequently have no shell completion support. This means:

1. Tab-completing command names like `swarm li<TAB>` → `swarm list` doesn't work
2. Tab-completing flags like `swarm run --<TAB>` doesn't show available options
3. Tab-completing agent IDs/names for commands like `swarm kill <TAB>` isn't possible
4. Tab-completing prompt names for `swarm run -p <TAB>` isn't available

Shell completion is a standard feature for modern CLI tools and significantly improves usability and discoverability. Tools like `kubectl`, `docker`, and `gh` all provide this functionality.

## Solution

Add a `swarm completion` command that generates shell completion scripts for bash, zsh, fish, and PowerShell. Cobra (the CLI framework used) has built-in support for generating these scripts.

### Proposed API

```bash
# Generate bash completion script
swarm completion bash

# Generate zsh completion script  
swarm completion zsh

# Generate fish completion script
swarm completion fish

# Generate PowerShell completion script
swarm completion powershell
```

### Installation instructions (output by each subcommand)

**Bash:**
```bash
# Add to ~/.bashrc:
source <(swarm completion bash)

# Or install permanently:
swarm completion bash > /etc/bash_completion.d/swarm
```

**Zsh:**
```bash
# Add to ~/.zshrc (before compinit):
source <(swarm completion zsh)

# Or install to a directory in $fpath:
swarm completion zsh > "${fpath[1]}/_swarm"
```

**Fish:**
```bash
swarm completion fish | source

# Or install permanently:
swarm completion fish > ~/.config/fish/completions/swarm.fish
```

## Files to create/change

- Create `cmd/completion.go` - new command implementation
- Update `README.md` - add shell completion section

## Implementation details

### cmd/completion.go

```go
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for swarm-cli.

To load completions:

Bash:
  $ source <(swarm completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ swarm completion bash > /etc/bash_completion.d/swarm
  # macOS:
  $ swarm completion bash > $(brew --prefix)/etc/bash_completion.d/swarm

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ swarm completion zsh > "${fpath[1]}/_swarm"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ swarm completion fish | source

  # To load completions for each session, execute once:
  $ swarm completion fish > ~/.config/fish/completions/swarm.fish

PowerShell:
  PS> swarm completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> swarm completion powershell > swarm.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
```

### Dynamic completions for agent IDs/names

Enhance commands that take agent identifiers to provide dynamic completion. This requires adding custom completion functions.

Update commands like `kill`, `stop`, `start`, `logs`, `inspect`, `update` to include:

```go
// In cmd/kill.go, cmd/logs.go, etc.
func init() {
	// ... existing flags ...
	
	// Add dynamic completion for agent identifier
	killCmd.ValidArgsFunction = completeAgentIdentifier
}

// Add to cmd/completion.go or a shared location
func completeAgentIdentifier(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Get state manager - use global scope to show all agents
	mgr, err := state.NewManagerWithScope(scope.ScopeGlobal, "")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	agents, err := mgr.List(false) // Include terminated for some commands
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, agent := range agents {
		// Add ID with description
		completions = append(completions, fmt.Sprintf("%s\t%s (%s)", agent.ID, agent.Prompt, agent.Status))
		// Add name if present
		if agent.Name != "" {
			completions = append(completions, fmt.Sprintf("%s\t%s (%s)", agent.Name, agent.Prompt, agent.Status))
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
```

### Dynamic completions for prompt names

For `swarm run -p <TAB>`:

```go
// Add to cmd/run.go init()
runCmd.RegisterFlagCompletionFunc("prompt", completePromptName)

// Add to cmd/completion.go
func completePromptName(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	promptsDir, err := GetPromptsDir()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	prompts, err := prompt.ListPrompts(promptsDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return prompts, cobra.ShellCompDirectiveNoFileComp
}
```

### Dynamic completions for model names

For `swarm run -m <TAB>` and `swarm update -m <TAB>`:

```go
runCmd.RegisterFlagCompletionFunc("model", completeModelName)

func completeModelName(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Return known models based on backend
	// These could be read from config or hardcoded for common models
	models := []string{
		"opus\tClaude Opus",
		"sonnet\tClaude Sonnet",
		"opus-4.5-thinking\tClaude 4.5 Opus (Thinking)",
		"sonnet-4.5-thinking\tClaude 4.5 Sonnet (Thinking)",
	}
	return models, cobra.ShellCompDirectiveNoFileComp
}
```

## Edge cases

1. **Empty agent list**: Return empty completion list, no error.

2. **Failed state load**: Return empty completions with `cobra.ShellCompDirectiveNoFileComp` to prevent file completion fallback.

3. **Global vs project scope**: For agent completion, use global scope so users can complete agents from any project. The command itself will still respect scope.

4. **Special characters in names**: Agent names with special characters should be properly escaped by Cobra's completion system.

5. **Very long lists**: If there are many agents/prompts, the shell handles pagination. No special handling needed.

## README additions

Add a new section after "Installation":

```markdown
## Shell Completion

swarm-cli supports shell completion for bash, zsh, fish, and PowerShell.

### Bash

```bash
# Add to ~/.bashrc
source <(swarm completion bash)
```

### Zsh

```bash
# Add to ~/.zshrc (before compinit)
source <(swarm completion zsh)
```

### Fish

```bash
swarm completion fish > ~/.config/fish/completions/swarm.fish
```

For more detailed instructions, run `swarm completion --help`.
```

## Acceptance criteria

- `swarm completion bash` outputs valid bash completion script
- `swarm completion zsh` outputs valid zsh completion script
- `swarm completion fish` outputs valid fish completion script
- `swarm completion powershell` outputs valid PowerShell completion script
- Tab completion works for all command names (`swarm li<TAB>` → `swarm list`)
- Tab completion works for all flags (`swarm run --<TAB>` shows all flags)
- Tab completion works for agent IDs/names (`swarm kill <TAB>` shows running agents)
- Tab completion works for prompt names (`swarm run -p <TAB>` shows available prompts)
- README is updated with shell completion installation instructions
- No errors when state file doesn't exist or is corrupted
