package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/fsutil"
)

// ErrWorkspaceMoved reports that the lock file was moved or removed while the
// caller waited for the lock, so the lock no longer guards the path it was
// requested for.
var ErrWorkspaceMoved = errors.New("lock file moved or removed while waiting")

// confirmLock checks that the lock l just took still guards path. A waiter that
// opened the lock file before the directory holding it was renamed away would
// otherwise hold the archived file while new writers contend on a fresh one at
// the same path, so exclusion is lost. A stale lock is released and reported,
// never retried: a retry would recreate the moved directory.
func confirmLock(l *fileLock, path string) (*fileLock, error) {
	held, err := l.f.Stat()
	if err == nil {
		var cur os.FileInfo
		cur, err = os.Stat(path)
		if errors.Is(err, fs.ErrNotExist) || (err == nil && !os.SameFile(held, cur)) {
			err = ErrWorkspaceMoved
		}
	}
	if err != nil {
		_ = l.release()
		return nil, err
	}
	return l, nil
}

// AtomicWrite writes data to path via a temp file in the same directory followed
// by an atomic rename, so a concurrent reader, or a writer killed mid-write:
// never observes a half-written structured file. The temp file is fsync'd before
// the rename so the content is durable before it becomes visible.
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		return fmt.Errorf("atomic write %s: %w", path, err)
	}
	return fsutil.WriteFileAtomic(path, data, perm)
}

// AppendLog appends one newline-terminated record to path using O_APPEND, so
// concurrent short-lived writers never lose or interleave records: each write is
// positioned at the current end of file by the kernel, and a small record (well
// under PIPE_BUF) lands as a single atomic write. The file and any parent
// directories are created on demand.
func AppendLog(path, record string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("append log %s: %w", path, err)
	}
	// #nosec G304 -- workspace artifact path inside the .devrites tree
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("append log %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(strings.TrimRight(record, "\n") + "\n"); err != nil {
		return fmt.Errorf("append log %s: %w", path, err)
	}
	return nil
}

// WithFeatureLock runs fn while holding an exclusive, advisory, per-feature lock,
// serializing read-modify-write across concurrent devrites-engine processes so two
// writers never corrupt a feature's files. On unix the lock is flock-based and
// is released automatically even if the process dies mid-operation (crash-safe);
// see lock_unix.go / lock_other.go.
func WithFeatureLock(root, slug string, fn func() error) error {
	lockPath, err := featureLockPath(root, slug)
	if err != nil {
		return err
	}
	return WithLock(lockPath, fn)
}

// WithLock runs fn while holding the same exclusive advisory lock as
// WithFeatureLock, keyed on lockPath, for shared ledgers outside one feature.
func WithLock(lockPath string, fn func() error) error {
	return withLock(lockPath, func(func()) error { return fn() })
}

// WithFeatureLockRelease is WithFeatureLock for a caller that must drop the lock
// before fn returns, for example because the platform refuses to rename the
// directory that holds the open lock file. fn receives release, which is safe to
// call more than once; the lock is also released when fn returns.
func WithFeatureLockRelease(root, slug string, fn func(release func()) error) error {
	lockPath, err := featureLockPath(root, slug)
	if err != nil {
		return err
	}
	return withLock(lockPath, fn)
}

// featureLockPath validates slug and creates the feature directory, returning
// the lock file path inside it.
func featureLockPath(root, slug string) (string, error) {
	if !devritespaths.ValidSlug(slug) {
		return "", fmt.Errorf("lock feature %q: not a feature slug", slug)
	}
	dir := featureDir(root, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("lock feature %q: %w", slug, err)
	}
	return filepath.Join(dir, ".lock"), nil
}

func withLock(lockPath string, fn func(release func()) error) error {
	l, err := acquireLock(lockPath)
	if err != nil {
		return fmt.Errorf("lock %s: %w", lockPath, err)
	}
	var once sync.Once
	release := func() { once.Do(func() { _ = l.release() }) }
	defer release()
	return fn(release)
}
