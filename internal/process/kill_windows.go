//go:build windows

package process

import (
	"os"
)

// Kill terminates a process on Windows.
func Kill(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}

// ForceKill immediately terminates a process on Windows.
// On Windows, this is the same as Kill since there's no graceful termination signal.
func ForceKill(pid int) error {
	return Kill(pid)
}
