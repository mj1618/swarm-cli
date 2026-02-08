//go:build !windows

package process

import (
	"bytes"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// Kill sends a termination signal to a process.
func Kill(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}

// ForceKill sends SIGKILL to immediately terminate a process, its process group,
// and all descendant processes. This ensures child processes (like agent CLIs)
// are killed even if they created their own process groups.
func ForceKill(pid int) error {
	// First, recursively find and kill all descendant processes.
	// This handles cases where child processes (e.g., claude/cursor CLIs)
	// have created their own process groups and wouldn't be killed by
	// a process group signal alone.
	killDescendants(pid)

	// Kill the process group (negative PID) to catch any remaining
	// processes in the same group
	_ = syscall.Kill(-pid, syscall.SIGKILL)

	// Kill the process itself
	return syscall.Kill(pid, syscall.SIGKILL)
}

// killDescendants recursively finds and kills all descendant processes of the
// given PID using pgrep. Children are killed before their parents to prevent
// orphan re-parenting issues.
func killDescendants(pid int) {
	children := findChildPIDs(pid)
	for _, child := range children {
		killDescendants(child)
		// Kill the child's process group first (in case it's a group leader)
		_ = syscall.Kill(-child, syscall.SIGKILL)
		_ = syscall.Kill(child, syscall.SIGKILL)
	}
}

// findChildPIDs returns the PIDs of all direct child processes of the given PID.
func findChildPIDs(pid int) []int {
	out, err := exec.Command("pgrep", "-P", strconv.Itoa(pid)).Output()
	if err != nil {
		return nil
	}

	var pids []int
	for _, line := range strings.Split(strings.TrimSpace(string(bytes.TrimSpace(out))), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if p, err := strconv.Atoi(line); err == nil {
			pids = append(pids, p)
		}
	}
	return pids
}
