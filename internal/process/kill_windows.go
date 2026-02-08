//go:build windows

package process

import (
	"os"
	"os/exec"
	"strconv"
)

// Kill terminates a process on Windows.
func Kill(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}

// ForceKill immediately terminates a process and all its descendants on Windows.
// Uses taskkill /T to kill the entire process tree.
func ForceKill(pid int) error {
	// taskkill /T kills the process and all child processes
	// taskkill /F forces termination
	err := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run()
	if err != nil {
		// Fall back to killing just the process
		return Kill(pid)
	}
	return nil
}
