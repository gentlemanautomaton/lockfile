//go:build windows

package lockfile

import (
	"os"
	"sync"
	"syscall"
)

// File is an open lock file.
type File struct {
	mutex sync.Mutex
	file  *os.File
}

// Create attempts to create a lock file with the given path.
//
// It uses an exclusive file lock to prevent competing processes from
// acquiring a lock while the file is open. The file is marked as temporary.
//
// If successful, it returns a [File] that wraps the underlying [os.File].
//
// The [File] will stay open and locked until [File.Close] is called, which
// will delete the lock file and release system resources that are associated
// with it.
//
// If the file already exists, it returns [os.ErrExists].
//
// If the file already exists but is marked for deletion, it returns
// [os.ErrPermission]. Unfortunately, this case is indistinguishable from
// regular access denied errors, due to the design of the underlying API
// calls.
func Create(path string) (*File, error) {
	const (
		FILE_ATTRIBUTE_TEMPORARY  = 0x00000100
		FILE_FLAG_DELETE_ON_CLOSE = 0x04000000
	)

	// FIXME: Handle long file paths by prefixing them with the extended path
	// prefix (\\?\). The standard library does this with [os.fixLongPath],
	// which sadly is not exposed.

	handle, err := createFile(path, syscall.GENERIC_READ, 0, syscall.CREATE_NEW, FILE_ATTRIBUTE_TEMPORARY|FILE_FLAG_DELETE_ON_CLOSE)
	if err != nil {
		if errno, ok := err.(syscall.Errno); ok {
			switch errno {
			case syscall.ERROR_FILE_EXISTS:
				return nil, os.ErrExist
			case syscall.ERROR_ACCESS_DENIED:
				// This can happen if the file is pending deletion, but
				// it can also happen if we don't have the necessary
				// privileges to create the file.
				return nil, os.ErrPermission
			}
		}
		return nil, err
	}

	return &File{
		file: os.NewFile(uintptr(handle), path),
	}, nil
}

// Close deletes the lock file.
func (f *File) Close() error {
	// Hold a lock so that this call is threadsafe.
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// If the file has already been closed, we're done.
	if f.file == nil {
		return os.ErrClosed
	}

	// Close the file.
	err := f.file.Close()
	f.file = nil

	return err
}
