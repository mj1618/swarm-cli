package dag

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// lockDir is the directory for cross-process lock files.
var lockDir = filepath.Join(os.TempDir(), "swarm", "locks")

// activeLocks tracks file handles for locks held by this process.
var (
	activeLocks   = make(map[string][]*os.File)
	activeLocksMu sync.Mutex
)

// AcquireTaskSlot blocks until a slot is available for the task.
// Uses file-based locking to coordinate across multiple processes.
// Returns immediately if limit is 0 (unlimited).
func AcquireTaskSlot(taskName string, limit int) {
	if limit <= 0 {
		return
	}

	// Ensure lock directory exists
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		// Fall back to no locking if we can't create the directory
		return
	}

	// Try to acquire one of the available slots
	for {
		for slot := 0; slot < limit; slot++ {
			lockPath := filepath.Join(lockDir, fmt.Sprintf("%s.%d.lock", taskName, slot))
			f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
			if err != nil {
				continue
			}

			// Try non-blocking exclusive lock
			err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
			if err == nil {
				// Successfully acquired lock
				activeLocksMu.Lock()
				activeLocks[taskName] = append(activeLocks[taskName], f)
				activeLocksMu.Unlock()
				return
			}

			// Couldn't get this slot, close and try next
			f.Close()
		}

		// All slots busy, wait and retry
		time.Sleep(100 * time.Millisecond)
	}
}

// ReleaseTaskSlot releases a slot for the task.
// Does nothing if limit is 0 (unlimited).
func ReleaseTaskSlot(taskName string, limit int) {
	if limit <= 0 {
		return
	}

	activeLocksMu.Lock()
	defer activeLocksMu.Unlock()

	locks := activeLocks[taskName]
	if len(locks) == 0 {
		return
	}

	// Release the most recently acquired lock (LIFO)
	f := locks[len(locks)-1]
	activeLocks[taskName] = locks[:len(locks)-1]

	// Unlock and close
	syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	f.Close()
}

// ResetTaskSemaphores releases all locks. Used for testing and cleanup.
func ResetTaskSemaphores() {
	activeLocksMu.Lock()
	defer activeLocksMu.Unlock()

	for _, locks := range activeLocks {
		for _, f := range locks {
			syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			f.Close()
		}
	}
	activeLocks = make(map[string][]*os.File)
}

// CleanupLockFiles removes stale lock files. Called on startup.
func CleanupLockFiles() {
	// Lock files are automatically released when processes exit,
	// so we just need to remove any orphaned files.
	// We can safely remove files that aren't locked.
	entries, err := os.ReadDir(lockDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		lockPath := filepath.Join(lockDir, entry.Name())
		f, err := os.OpenFile(lockPath, os.O_RDWR, 0644)
		if err != nil {
			continue
		}

		// Try non-blocking lock - if we get it, file is orphaned
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			// Got the lock, file was orphaned - unlock and remove
			syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			f.Close()
			os.Remove(lockPath)
		} else {
			// File is actively locked by another process
			f.Close()
		}
	}
}
