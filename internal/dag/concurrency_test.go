package dag

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAcquireReleaseTaskSlot_Unlimited(t *testing.T) {
	ResetTaskSemaphores()
	defer ResetTaskSemaphores()

	// With limit 0, should return immediately without blocking
	start := time.Now()
	AcquireTaskSlot("test-task", 0)
	ReleaseTaskSlot("test-task", 0)
	elapsed := time.Since(start)

	if elapsed > 50*time.Millisecond {
		t.Errorf("unlimited concurrency took too long: %v", elapsed)
	}
}

func TestAcquireReleaseTaskSlot_Limited(t *testing.T) {
	ResetTaskSemaphores()
	defer ResetTaskSemaphores()

	// Acquire a slot
	AcquireTaskSlot("test-limited", 1)

	// Try to acquire another slot in a goroutine - should block
	blocked := make(chan bool, 1)
	acquired := make(chan bool, 1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		blocked <- true
		AcquireTaskSlot("test-limited", 1)
		acquired <- true
	}()

	// Wait a bit and verify it's blocked
	select {
	case <-blocked:
		// Good, goroutine started
	case <-time.After(200 * time.Millisecond):
		t.Error("goroutine didn't signal")
		return
	}

	// Verify the second acquire is still blocked
	select {
	case <-acquired:
		t.Error("second acquire should have blocked")
		return
	case <-time.After(100 * time.Millisecond):
		// Good, it's still blocked
	}

	// Release the first slot
	ReleaseTaskSlot("test-limited", 1)

	// Now the second acquire should complete
	select {
	case <-acquired:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Error("second acquire didn't complete after release")
	}

	// Clean up
	ReleaseTaskSlot("test-limited", 1)
}

func TestConcurrencyLimit_EnforcesLimit(t *testing.T) {
	ResetTaskSemaphores()
	defer ResetTaskSemaphores()

	const limit = 2
	const totalTasks = 5

	var running int32
	var maxRunning int32
	var wg sync.WaitGroup

	for i := 0; i < totalTasks; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			AcquireTaskSlot("limited-task", limit)
			defer ReleaseTaskSlot("limited-task", limit)

			// Track concurrent executions
			current := atomic.AddInt32(&running, 1)
			for {
				old := atomic.LoadInt32(&maxRunning)
				if current <= old || atomic.CompareAndSwapInt32(&maxRunning, old, current) {
					break
				}
			}

			// Simulate work
			time.Sleep(50 * time.Millisecond)

			atomic.AddInt32(&running, -1)
		}()
	}

	wg.Wait()

	if maxRunning > int32(limit) {
		t.Errorf("max concurrent tasks = %d, want <= %d", maxRunning, limit)
	}
}

func TestDifferentTasksIndependent(t *testing.T) {
	ResetTaskSemaphores()
	defer ResetTaskSemaphores()

	// Two different tasks should have independent semaphores
	AcquireTaskSlot("task-a", 1)
	AcquireTaskSlot("task-b", 1) // Should not block

	// Both should be acquired successfully
	ReleaseTaskSlot("task-a", 1)
	ReleaseTaskSlot("task-b", 1)
}

func TestResetTaskSemaphores(t *testing.T) {
	defer ResetTaskSemaphores()

	// Acquire some slots
	AcquireTaskSlot("task-1", 1)
	AcquireTaskSlot("task-2", 1)

	// Reset
	ResetTaskSemaphores()

	// Should be able to acquire again immediately (new semaphores)
	done := make(chan bool, 1)
	go func() {
		AcquireTaskSlot("task-1", 1)
		done <- true
	}()

	select {
	case <-done:
		// Success
		ReleaseTaskSlot("task-1", 1)
	case <-time.After(500 * time.Millisecond):
		t.Error("acquire after reset should not block")
	}
}

func TestCleanupLockFiles(t *testing.T) {
	ResetTaskSemaphores()
	defer ResetTaskSemaphores()

	// Acquire and release a slot to create a lock file
	AcquireTaskSlot("cleanup-test", 1)
	ReleaseTaskSlot("cleanup-test", 1)

	// CleanupLockFiles should remove orphaned files
	CleanupLockFiles()

	// Should be able to acquire after cleanup
	AcquireTaskSlot("cleanup-test", 1)
	ReleaseTaskSlot("cleanup-test", 1)
}
