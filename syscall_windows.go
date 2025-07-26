//go:build windows

package lockfile

import (
	"syscall"
)

// createFile opens or creates a file by its name. The file will be opened
// or created with the given access, share mode, create mode, and
// flags/attributes.
//
// The file will be created with a default security descriptor. The handle
// that is returned will not be inheritable.
func createFile(fileName string, access, shareMode, createMode, flagsAndAttributes uint32) (handle syscall.Handle, err error) {
	if len(fileName) == 0 {
		return 0, syscall.EINVAL
	}

	fnp, err := syscall.UTF16PtrFromString(fileName)
	if err != nil {
		return 0, err
	}

	return syscall.CreateFile(fnp, access, shareMode, nil, createMode, flagsAndAttributes, 0)
}
