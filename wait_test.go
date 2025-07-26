package lockfile_test

import (
	"context"
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	"github.com/gentlemanautomaton/lockfile"
)

func TestWaitParallel(t *testing.T) {
	const parallel = 64
	const rounds = 10

	// Let this test run in parallel with other tests, so that all of the
	// tests fight each other for lock files.
	t.Parallel()

	var wg sync.WaitGroup
	wg.Add(parallel)

	for i := range parallel {
		go func(instance int) {
			defer wg.Done()
			for round := range rounds {
				time.Sleep(time.Millisecond)
				func() {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
					defer cancel()

					lock, err := lockfile.WaitCtx(ctx, testLockFile)
					if err != nil {
						t.Logf("Instance %d: Round %d: Failed to create lock file: %v", instance, round, err)
						t.Fail()
						return
					}
					defer func() {
						if err := lock.Close(); err != nil {
							t.Logf("Instance %d: Round %d: Closing the lock file returned an error: %v", instance, round, err)
							t.Fail()
						}
					}()

					if err := acquire(); err != nil {
						t.Logf("Instance %d: Round %d: Lock Acquired but validation failed: %v", instance, round, err)
						t.Fail()
					} else {
						t.Logf("Instance %d: Round %d: Lock Acquired", instance, round)
					}

					time.Sleep(time.Millisecond * time.Duration(rand.IntN(5)))

					if err := release(); err != nil {
						t.Logf("Instance %d: Round %d: Lock Released but validation failed: %v", instance, round, err)
						t.Fail()
					} else {
						t.Logf("Instance %d: Round %d: Lock Released", instance, round)
					}
				}()
			}
		}(i)
	}

	wg.Wait()
}
