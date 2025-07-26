//go:build !windows

package lockfile

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
)

// Lennart Poettering provides a helpful overview of the hazards of file
// locking in Linux here:
// https://0pointer.net/blog/projects/locking.html
//
// We follow an algorithm similar to the one described by Guido U. Draheim
// in his answer to the Stack Overflow question "flock(): removing locked
// file without race condition?", which can be found here:
// https://stackoverflow.com/questions/17708885/flock-removing-locked-file-without-race-condition/51070775#51070775

// File is an open lock file.
type File struct {
	path  string
	mutex sync.Mutex
	file  *os.File
}

// Create attempts to create a lock file with the given path.
//
// It uses the flock system call to lock the file, which acquires an advisory
// lock. This means that the lock is only effective if all processes competing
// for the lock file also acquire the same kind of file lock.
//
// If successful, it returns a [File] that wraps the underlying [os.File].
//
// The [File] will stay open and locked until [File.Close] is called, which
// will delete the lock file and release system resources that are associated
// with it.
//
// If the lock file already exists, it returns [os.ErrExists].
func Create(path string) (*File, error) {
	for {
		// Create the lock file if it doesn't exist.
		//
		// Note that we could race with another process here, so this might open
		// a lock file that was created by another process.
		//
		// Note also that we don't make this world readable. This prevents
		// unprivileged processes from taking a lock on this file, which could
		// result in a denial-of-service attack if they never release it.
		file, err := os.OpenFile(path, os.O_CREATE, 0400)
		if err != nil {
			return nil, err
		}

		// Try to lock the file with the flock system call.
		//
		// This locks the whole file. Unlike the posix file locking calls, the
		// lock acquired by flock is attached to the provided file descriptor, not
		// the calling process as a whole.
		//
		// If we get a [syscall.EWOULDBLOCK] error, it means that someone else got
		// the advisory lock before we did. In that case, they will be responsible
		// for deleting the file when they are done with it.
		//
		// https://man7.org/linux/man-pages/man2/flock.2.html
		fd := int(file.Fd())
		if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			file.Close()
			switch {
			case errors.Is(err, syscall.EWOULDBLOCK):
				return nil, os.ErrExist
			default:
				return nil, err
			}
		}

		// Make sure that the file is empty and the number of links to the
		// file is non-zero.
		//
		// The number of links can be zero if another process opened, locked and
		// deleted the lock file between our open and flock calls.
		//
		// If we detect this case, we start over and try again.
		fi, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to stat lock file \"%s\" after creation: %w", path, err)
		}

		if fi.Size() != 0 {
			return nil, fmt.Errorf("the lock file \"%s\" is not empty", path)
		}

		if stat, ok := fi.Sys().(*syscall.Stat_t); !ok || stat == nil {
			file.Close()
			return nil, fmt.Errorf("the os.Stat call for lock file \"%s\" returned an unexpected data type", path)
		} else if stat.Nlink == 0 {
			file.Close()
			continue // We lost this race. Try again.
		}

		return &File{
			path: path,
			file: file,
		}, nil
	}
}

// Close deletes the lock file. It returns an error if it is unable to do
// so, or if the underlying file handle could not be closed.
//
// It returns [os.ErrClosed] if the function has already been called.
func (f *File) Close() (err error) {
	// Hold a lock so that this call is threadsafe.
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// If the file has already been closed, we're done.
	if f.file == nil {
		return os.ErrClosed
	}

	// Always close the file handle when we're done. This will automatically
	// release the file lock at the same time.
	//
	// It's very important that this happens after the file is unlinked. To
	// do otherwise can lead to race conditions.
	defer func() {
		closeErr := f.file.Close()
		f.file = nil
		if err == nil {
			err = closeErr
		}
	}()

	// If the file is still at the expected file path, unlink it.
	fi1, err := f.file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat opened lock file \"%s\" before deletion: %w", f.path, err)
	}

	fi2, err := os.Stat(f.path)
	if err != nil {
		return fmt.Errorf("failed to stat existing lock file \"%s\" by its before deletion: %w", f.path, err)
	}

	if !os.SameFile(fi1, fi2) {
		// The lock file was probably renamed. That's not good, but there's not
		// much we can do about it.
		return fmt.Errorf("failed to unlink lock file \"%s\": the file was moved or deleted", f.path)
	}

	// Unlink the file.
	if err := syscall.Unlink(f.path); err != nil {
		return fmt.Errorf("failed to unlink lock file \"%s\": %w", f.path, err)
	}

	return nil
}
