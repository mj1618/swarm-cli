//go:build !windows

package state

import (
	"fmt"
	"os"
	"syscall"
)

// fileLock provides cross-process file locking using flock.
type fileLock struct {
	path string
	file *os.File
}

// newFileLock creates a new file lock.
func newFileLock(path string) *fileLock {
	return &fileLock{path: path}
}

// Lock acquires an exclusive lock on the file.
func (fl *fileLock) Lock() error {
	f, err := os.OpenFile(fl.path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}
	fl.file = f

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	return nil
}

// Unlock releases the lock and closes the file.
func (fl *fileLock) Unlock() error {
	if fl.file == nil {
		return nil
	}
	// Unlock and close
	syscall.Flock(int(fl.file.Fd()), syscall.LOCK_UN)
	err := fl.file.Close()
	fl.file = nil
	return err
}

// isProcessRunning checks if a process with the given PID is still running.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, sending signal 0 checks if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
