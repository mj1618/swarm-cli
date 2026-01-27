//go:build !windows

package detach

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// StartDetached starts the current process in detached mode.
// It re-executes the binary with the same arguments plus the internal detached flag.
// Returns the PID of the detached process and the log file path.
func StartDetached(args []string, logFile string, workingDir string) (int, error) {
	// Get the current executable
	executable, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create/open the log file
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to create log file: %w", err)
	}

	// Build the command with the internal flag
	cmd := exec.Command(executable, args...)
	cmd.Dir = workingDir
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.Stdin = nil

	// Set up process attributes for detaching (Unix-specific)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create a new session
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		f.Close()
		return 0, fmt.Errorf("failed to start detached process: %w", err)
	}

	// Don't wait for the process - it's detached
	// The file handle will be inherited by the child process

	return cmd.Process.Pid, nil
}
