//go:build (linux || darwin) && (amd64 || arm64)

package fsutil

import (
	"os"
	"runtime"
	"syscall"
)

func nativePublishOpen(parent *os.File, name string, directory, movable bool) (*os.File, error) {
	flags := syscall.O_RDONLY | syscall.O_CLOEXEC | syscall.O_NOFOLLOW | syscall.O_NONBLOCK
	if directory {
		flags |= syscall.O_DIRECTORY
	}
	fd, err := publishOpenat(int(parent.Fd()), name, flags)
	runtime.KeepAlive(parent)
	if err != nil {
		return nil, &os.PathError{Op: "publication openat", Path: name, Err: err}
	}
	return os.NewFile(uintptr(fd), name), nil
}

func nativePublishSync(file *os.File) error {
	return file.Sync()
}
