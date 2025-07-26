package lockfile_test

import (
	"context"
	"fmt"
	"time"

	"github.com/gentlemanautomaton/lockfile"
)

func ExampleWaitCtx() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	lock, err := lockfile.WaitCtx(ctx, "test.lock")
	if err != nil {
		fmt.Printf("Lock Acquisition Failed: %v\n", err)
		return
	}

	defer func() {
		if err := lock.Close(); err != nil {
			fmt.Printf("Lock Release Failed: %v\n", err)
		}
	}()

	fmt.Printf("Lock Acquired\n")

	// Output:
	// Lock Acquired
}
