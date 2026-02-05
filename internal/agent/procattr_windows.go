//go:build windows

package agent

import (
	"os/exec"
)

// setProcAttr sets Windows-specific process attributes.
// On Windows, process groups work differently and we rely on
// the standard process termination behavior.
func setProcAttr(cmd *exec.Cmd) {
	// No special attributes needed on Windows
}
