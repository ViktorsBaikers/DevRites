package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AtomicWrite writes data to path via a temp file in the same directory followed
// by an atomic rename, so a concurrent reader — or a writer killed mid-write —
// never observes a half-written structured file. The temp file is fsync'd before
// the rename so the content is durable before it becomes visible.
func AtomicWrite(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("atomic write %s: %w", path, err)
	}
	tmpName := tmp.Name()
	// On any failure, don't leave the temp file behind.
	defer func() {
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("atomic write %s: %w", path, err)
	}
	if err = tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("atomic write %s: %w", path, err)
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("atomic write %s: %w", path, err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("atomic write %s: %w", path, err)
	}
	if err = os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("atomic write %s: %w", path, err)
	}
	return nil
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
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("append log %s: %w", path, err)
	}
	defer f.Close()
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
	dir := featureDir(root, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("lock feature %q: %w", slug, err)
	}
	l, err := acquireLock(filepath.Join(dir, ".lock"))
	if err != nil {
		return fmt.Errorf("lock feature %q: %w", slug, err)
	}
	defer l.release()
	return fn()
}
