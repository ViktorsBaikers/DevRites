//go:build windows

package state

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const lockFileExclusiveLock = 0x2

var (
	lockFileEx   = syscall.NewLazyDLL("kernel32.dll").NewProc("LockFileEx")
	unlockFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("UnlockFileEx")
)

type fileLock struct {
	f  *os.File
	ov syscall.Overlapped
}

func acquireLock(path string) (*fileLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("acquire lock: %w", err)
	}
	l := &fileLock{f: f}
	if ok, _, callErr := lockFileEx.Call(
		f.Fd(),
		lockFileExclusiveLock,
		0,
		1,
		0,
		uintptr(unsafe.Pointer(&l.ov)),
	); ok == 0 {
		_ = f.Close()
		return nil, fmt.Errorf("lock file %s: %w", path, callErr)
	}
	return l, nil
}

func (l *fileLock) release() error {
	if ok, _, callErr := unlockFileEx.Call(
		l.f.Fd(),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(&l.ov)),
	); ok == 0 {
		_ = l.f.Close()
		return callErr
	}
	return l.f.Close()
}
