//go:build windows

package fsutil

import (
	"syscall"
	"unsafe"
)

const moveFileReplaceExisting = 0x1

var moveFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

func replaceFile(from, to string) error {
	fromPtr, err := syscall.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	toPtr, err := syscall.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	if ok, _, callErr := moveFileEx.Call(
		uintptr(unsafe.Pointer(fromPtr)),
		uintptr(unsafe.Pointer(toPtr)),
		moveFileReplaceExisting,
	); ok == 0 {
		return callErr
	}
	return nil
}
