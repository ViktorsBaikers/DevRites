//go:build linux && (amd64 || arm64)

package fsutil

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

func publishOpenat(parent int, name string, flags int) (int, error) {
	return syscall.Openat(parent, name, flags, 0)
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
	// Linux UAPI: x86 syscall_64.tbl and asm-generic/unistd.h.
	trap := uintptr(316)
	if runtime.GOARCH == "arm64" {
		trap = 276
	}
	const renameNoReplace = 1
	_, _, errno := syscall.Syscall6(trap, sourceParent.Fd(), uintptr(unsafe.Pointer(old)), targetParent.Fd(), uintptr(unsafe.Pointer(new)), renameNoReplace, 0)
	runtime.KeepAlive(sourceParent)
	runtime.KeepAlive(targetParent)
	if errno != 0 {
		return &os.LinkError{Op: "renameat2 noreplace", Old: from, New: to, Err: errno}
	}
	return nil
}
