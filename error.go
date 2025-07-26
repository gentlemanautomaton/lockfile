package lockfile

import (
	"os"
	"runtime"
)

// IsTemporary returns true if the given error returned by [Create] indicates
// temporary contention of the lock file.
func IsTemporary(err error) bool {
	switch err {
	case os.ErrExist:
		return true
	case os.ErrPermission:
		if runtime.GOOS == "windows" {
			// On Windows, os.ErrPermission can be returned by Create if a
			// previous lock file is in the process of being deleted.
			// Treat it like a temporary error
			return true
		}
	}
	return false
}
