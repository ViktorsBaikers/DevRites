//go:build unix

package state

import (
	"fmt"
	"os"
	"syscall"
)

// fileLock is an exclusive advisory lock held on an open lock file. On unix it
// uses flock(2), which the kernel releases automatically when the file is closed
// or the process exits, so a crashed writer never strands the lock.
type fileLock struct{ f *os.File }

func acquireLock(path string) (*fileLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("acquire lock: %w", err)
	}
	// LOCK_EX blocks until the lock is available; contending devrites-engine processes
	// queue here rather than racing the read-modify-write.
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("flock %s: %w", path, err)
	}
	return &fileLock{f: f}, nil
}

func (l *fileLock) release() error {
	// Closing the descriptor releases the flock; unlock explicitly first so the
	// intent is clear and the lock is gone before any close error is considered.
	_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	return l.f.Close()
}
