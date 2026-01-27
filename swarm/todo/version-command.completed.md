# Add `swarm version` command with build-time version injection

## Problem

The swarm-cli has no `swarm version` command, and the release process doesn't inject version information into the binary. This makes it difficult to:

1. Know which version of swarm-cli is installed
2. Report bugs with version information
3. Verify that updates were installed correctly
4. Debug issues related to specific versions

This is a standard feature for modern CLI tools (docker, kubectl, gh, etc.) and is currently missing.

## Solution

1. Add ldflags to `.goreleaser.yaml` to inject version, commit, and build date at compile time
2. Create a `version` package to hold these variables
3. Add a `swarm version` command that displays version information

### Proposed API

```bash
# Show version information
swarm version

# Output:
swarm version 1.2.3
  Commit:     abc1234
  Built:      2025-01-28T10:30:00Z
  Go version: go1.21.0
  OS/Arch:    darwin/arm64

# Short version (useful for scripting)
swarm version --short
# Output: 1.2.3

# JSON output (for automation)
swarm version --format json
```

## Files to create/change

- Create `internal/version/version.go` - version variables and info struct
- Create `cmd/version.go` - version command implementation
- Update `.goreleaser.yaml` - add ldflags for version injection
- Update `cmd/root.go` - add version command

## Implementation details

### internal/version/version.go

```go
package version

import (
	"fmt"
	"runtime"
)

// These variables are set at build time via ldflags
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// Info holds version information
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// GetInfo returns the current version information
func GetInfo() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String returns a formatted version string
func (i Info) String() string {
	return fmt.Sprintf(`swarm version %s
  Commit:     %s
  Built:      %s
  Go version: %s
  OS/Arch:    %s/%s`,
		i.Version, i.Commit, i.BuildDate, i.GoVersion, i.OS, i.Arch)
}
```

### cmd/version.go

```go
package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/matt/swarm-cli/internal/version"
	"github.com/spf13/cobra"
)

var (
	versionShort  bool
	versionFormat string
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print the version number, commit hash, build date, and runtime information for swarm-cli.`,
	Example: `  # Show full version information
  swarm version

  # Show only version number
  swarm version --short

  # Output as JSON
  swarm version --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		info := version.GetInfo()

		if versionShort {
			fmt.Println(info.Version)
			return nil
		}

		if versionFormat == "json" {
			output, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal version info: %w", err)
			}
			fmt.Println(string(output))
			return nil
		}

		fmt.Println(info.String())
		return nil
	},
}

func init() {
	versionCmd.Flags().BoolVarP(&versionShort, "short", "s", false, "Print only the version number")
	versionCmd.Flags().StringVar(&versionFormat, "format", "", "Output format: json or text (default)")
	rootCmd.AddCommand(versionCmd)
}
```

### .goreleaser.yaml changes

Update the `ldflags` section in the builds configuration:

```yaml
builds:
  - main: .
    binary: swarm
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X github.com/matt/swarm-cli/internal/version.Version={{.Version}}
      - -X github.com/matt/swarm-cli/internal/version.Commit={{.ShortCommit}}
      - -X github.com/matt/swarm-cli/internal/version.BuildDate={{.Date}}
```

### cmd/root.go changes

Add the version command to init():

```go
func init() {
	// ... existing flags ...
	rootCmd.AddCommand(versionCmd)
}
```

### Optional: Add version to root command

Optionally, also set the root command's version for `swarm --version` support:

```go
// In cmd/root.go
import "github.com/matt/swarm-cli/internal/version"

func init() {
	rootCmd.Version = version.Version
	// ... rest of init
}
```

This enables `swarm --version` as an alias for `swarm version --short`.

## Edge cases

1. **Development builds**: When built without ldflags (e.g., `go build`), version shows "dev", commit shows "unknown", and build date shows "unknown". This is intentional and helps identify unofficial builds.

2. **Snapshot builds**: GoReleaser uses `{{ .ShortCommit }}` for snapshot versions, which is reflected correctly in the version output.

3. **JSON output with special characters**: The Info struct uses proper JSON tags, so output is always valid JSON.

4. **Cross-compilation**: `runtime.GOOS` and `runtime.GOARCH` reflect the target platform, not the build platform, which is the desired behavior.

## Testing

### Manual testing

```bash
# Build without ldflags (development)
go build -o swarm .
./swarm version
# Should show "dev" version

# Build with ldflags (simulating release)
go build -ldflags "-X github.com/matt/swarm-cli/internal/version.Version=1.0.0 -X github.com/matt/swarm-cli/internal/version.Commit=abc1234 -X github.com/matt/swarm-cli/internal/version.BuildDate=2025-01-28" -o swarm .
./swarm version
# Should show injected values
```

### Unit tests

Add `cmd/version_test.go`:

```go
package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/matt/swarm-cli/internal/version"
)

func TestVersionCommand(t *testing.T) {
	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "swarm version") {
		t.Errorf("expected version output, got: %s", output)
	}
}

func TestVersionShort(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version", "--short"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version --short failed: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != version.Version {
		t.Errorf("expected %q, got %q", version.Version, output)
	}
}
```

## Acceptance criteria

- `swarm version` displays version, commit, build date, Go version, and OS/Arch
- `swarm version --short` displays only the version number
- `swarm version --format json` outputs valid JSON with all version fields
- Development builds (without ldflags) show "dev" as version
- Release builds (via goreleaser) show the correct git tag version
- `swarm --version` works as an alias for short version output
- No errors when version variables are not injected at build time

---

## Completion Notes (Agent 118d3fa6)

**Completed on:** 2025-01-28

**Files created:**
- `internal/version/version.go` - Version package with Info struct and GetInfo() function
- `cmd/version.go` - Version command implementation with --short and --format flags

**Files modified:**
- `cmd/root.go` - Added versionCmd and set rootCmd.Version for --version flag
- `.goreleaser.yaml` - Added ldflags for Version, Commit, and BuildDate injection

**All acceptance criteria met:**
- `swarm version` shows full version info
- `swarm version --short` shows just version number
- `swarm version --format json` outputs valid JSON
- Dev builds show "dev" version
- `swarm --version` works as alias
- ldflags injection tested and working
