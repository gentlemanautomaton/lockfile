package lockfile

import (
	"context"
	"math/rand/v2"
	"time"
)

// WaitCtx repeatedly calls [Create] with the given path until a lock file is
// successfully created, a non-temporary error is encountered or the provided
// context is cancelled.
func WaitCtx(ctx context.Context, path string) (*File, error) {
	// Try to create the lock file.
	file, err := Create(path)
	if err == nil {
		return file, nil
	}

	// If the error indicates a non-temporary failure, give up.
	if !IsTemporary(err) {
		return nil, err
	}

	// Repeatedly try to create the lock file until one of three things
	// happens:
	// 1. The lock file is successfully created.
	// 2: A non-temporary error is returned.
	// 3: The provided context is cancelled.
	attempt := 0
	timer := time.NewTimer(randomBackoff(attempt))
	for {
		// Wait for the timer to fire, or the context to be cancelled.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}

		// Try to create the lock file.
		file, err = Create(path)
		if err == nil {
			return file, nil
		}
		if !IsTemporary(err) {
			return nil, err
		}

		// Calculate a new random delay and reset the timer.
		attempt++
		delay := randomBackoff(attempt)
		timer.Reset(delay)
	}
}

// randomBackoff returns a random backoff time betwen 0 and 1 second.
func randomBackoff(attempt int) time.Duration {
	if attempt > 99 {
		attempt = 99
	}
	milliseconds := rand.IntN((1 + attempt) * 10)
	return time.Millisecond * time.Duration(milliseconds)
}
