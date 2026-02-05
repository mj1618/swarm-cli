//go:build !windows

package process

import (
	"syscall"
)

// Kill sends a termination signal to a process.
func Kill(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}

// ForceKill sends SIGKILL to immediately terminate a process and its entire process group.
// This ensures child processes (like agent CLIs) are also killed.
func ForceKill(pid int) error {
	// First try to kill the process group (negative PID)
	// This will kill all processes in the same process group
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		// If process group kill fails (e.g., not a group leader), fall back to killing just the process
		return syscall.Kill(pid, syscall.SIGKILL)
	}
	return nil
}
