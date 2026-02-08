//go:build !windows

package agent

import (
	"os/exec"
)

// setProcAttr sets Unix-specific process attributes.
// We intentionally do NOT set Setpgid here. The detached swarm-cli parent
// uses Setsid (which creates a new session and process group), and we want
// the agent CLI to inherit that process group. This way, ForceKill(-pid)
// on the swarm-cli PID kills the entire group including the agent CLI
// and any children it spawns.
func setProcAttr(cmd *exec.Cmd) {
	// No Setpgid — inherit parent's process group so signals propagate correctly
}
