//go:build windows

package state

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// fileLock provides cross-process file locking using Windows LockFileEx.
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

	// Use LockFileEx with LOCKFILE_EXCLUSIVE_LOCK for exclusive locking
	// Lock the entire file (offset 0, length MaxUint32)
	ol := &windows.Overlapped{}
	err = windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK,
		0,
		1,
		0,
		ol,
	)
	if err != nil {
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
	// Unlock the file
	ol := &windows.Overlapped{}
	windows.UnlockFileEx(
		windows.Handle(fl.file.Fd()),
		0,
		1,
		0,
		ol,
	)
	err := fl.file.Close()
	fl.file = nil
	return err
}

// isProcessRunning checks if a process with the given PID is still running.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	// On Windows, try to open the process with minimal permissions
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	// Check if process has exited
	var exitCode uint32
	err = windows.GetExitCodeProcess(handle, &exitCode)
	if err != nil {
		return false
	}

	// STILL_ACTIVE means the process is still running
	return exitCode == 259 // STILL_ACTIVE = 259
}
