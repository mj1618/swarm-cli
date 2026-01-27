//go:build !windows

package process

import (
	"syscall"
)

// Kill sends a termination signal to a process.
func Kill(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}
