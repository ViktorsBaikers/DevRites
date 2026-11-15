//go:build darwin && (amd64 || arm64)

package fsutil

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

// XNU bsd/kern/syscalls.master: openat=463, renameatx_np=488.
// syscall.Syscall6 uses Darwin's native trap ABI with CGO_ENABLED=0.
func publishOpenat(parent int, name string, flags int) (int, error) {
	p, err := syscall.BytePtrFromString(name)
	if err != nil {
		return -1, err
	}
	fd, _, errno := syscall.Syscall6(463, uintptr(parent), uintptr(unsafe.Pointer(p)), uintptr(flags), 0, 0, 0)
	if errno != 0 {
		return -1, errno
	}
	return int(fd), nil
}

func nativePublishRename(sourceParent, stage, targetParent *os.File, from, to string) error {
	old, err := syscall.BytePtrFromString(from)
	if err != nil {
		return err
	}
	new, err := syscall.BytePtrFromString(to)
	if err != nil {
		return err
	}
	const renameExcl = 4 // XNU bsd/sys/stdio.h RENAME_EXCL
	_, _, errno := syscall.Syscall6(488, sourceParent.Fd(), uintptr(unsafe.Pointer(old)), targetParent.Fd(), uintptr(unsafe.Pointer(new)), renameExcl, 0)
	runtime.KeepAlive(sourceParent)
	runtime.KeepAlive(targetParent)
	if errno != 0 {
		return &os.LinkError{Op: "renameatx_np excl", Old: from, New: to, Err: errno}
	}
	return nil
}
