package lockfile_test

import (
	"fmt"
	"sync/atomic"
)

const testLockFile = "test.lock"

var counter atomic.Int64

// acquire records a lock acquisition by updating an atomic counter.
// It returns an error if it detects a race condition.
func acquire() error {
	if value := counter.Add(1); value != 1 {
		return fmt.Errorf("adding 1 to lock counter returned an unexpected value: %d", value)
	}
	return nil
}

// release records a lock release by updating an atomic counter.
// It returns an error if it detects a race condition.
func release() error {
	if value := counter.Add(-1); value != 0 {
		return fmt.Errorf("subtracting 1 from the lock counter returned an unexpected value: %d", value)

	}
	return nil
}
