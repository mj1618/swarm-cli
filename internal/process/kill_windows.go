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
