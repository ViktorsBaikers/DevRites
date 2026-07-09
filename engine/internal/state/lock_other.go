//go:build !unix

package state

import (
	"fmt"
	"os"
)

// fileLock on non-unix platforms (notably Windows) is best-effort: the engine
// still cross-compiles and runs, but without kernel advisory locking. DevRites'
// concurrent-reviewer fan-out runs on macOS/Linux, where lock_unix.go provides
// real flock; this keeps the pure-Go build green on every target.
type fileLock struct{ f *os.File }

func acquireLock(path string) (*fileLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("acquire lock: %w", err)
	}
	return &fileLock{f: f}, nil
}

func (l *fileLock) release() error { return l.f.Close() }
