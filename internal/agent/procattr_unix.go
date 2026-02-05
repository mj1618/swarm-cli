//go:build !windows

package agent

import (
	"os/exec"
	"syscall"
)

// setProcAttr sets Unix-specific process attributes.
// This makes the command its own process group leader, allowing
// ForceKill to terminate the entire process group including child processes.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
