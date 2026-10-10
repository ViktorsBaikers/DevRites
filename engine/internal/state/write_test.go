//go:build unix || windows

package state

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// A waiter that opened the lock file before its directory was renamed away must
// not acquire the archived inode, and must not recreate the directory.
func TestLockOpenedRefusesLockMovedWhileWaiting(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows refuses to rename or remove a path with an open handle, so the stale state cannot be built")
	}
	root := t.TempDir()
	lockPath := filepath.Join(root, "work", "feat", ".lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Dir(lockPath), filepath.Join(root, "archive", "feat")); err != nil {
		t.Fatal(err)
	}

	if l, err := lockOpened(f, lockPath); !errors.Is(err, ErrWorkspaceMoved) {
		if l != nil {
			_ = l.release()
		}
		t.Fatalf("lockOpened on a moved workspace = %v, want ErrWorkspaceMoved", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "work", "feat")); !os.IsNotExist(err) {
		t.Fatalf("work/feat reappeared after a stale lock: %v", err)
	}
}

// A lock file replaced at the same path leaves the opened inode stale too.
func TestLockOpenedRefusesReplacedLockFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows refuses to rename or remove a path with an open handle, so the stale state cannot be built")
	}
	lockPath := filepath.Join(t.TempDir(), ".lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if l, err := lockOpened(f, lockPath); !errors.Is(err, ErrWorkspaceMoved) {
		if l != nil {
			_ = l.release()
		}
		t.Fatalf("lockOpened on a replaced lock file = %v, want ErrWorkspaceMoved", err)
	}
}

func TestWithFeatureLockAcquiresCurrentLockFile(t *testing.T) {
	ran := false
	if err := WithFeatureLock(t.TempDir(), "feat", func() error { ran = true; return nil }); err != nil || !ran {
		t.Fatalf("WithFeatureLock = %v, ran %v, want success", err, ran)
	}
}

// The identity check is shared by every platform's lock, so it is exercised
// without renaming anything: the held file is compared with whatever is at path.
func TestConfirmLockComparesHeldFileWithPath(t *testing.T) {
	dir := t.TempDir()
	open := func(name string) *os.File {
		f, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_RDWR, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	held := open("held.lock")
	other := open("other.lock")
	defer func() { _ = other.Close() }()

	l, err := confirmLock(&fileLock{f: held}, held.Name())
	if err != nil || l == nil {
		t.Fatalf("confirmLock on the current file = %v, %v, want the lock", l, err)
	}
	_ = l.f.Close()

	for name, path := range map[string]string{
		"different file at path": other.Name(),
		"missing path":           filepath.Join(dir, "gone.lock"),
	} {
		f := open("probe-" + filepath.Base(path))
		l, err := confirmLock(&fileLock{f: f}, path)
		if !errors.Is(err, ErrWorkspaceMoved) || l != nil {
			t.Errorf("%s: confirmLock = %v, %v, want ErrWorkspaceMoved and no lock", name, l, err)
		}
		if _, statErr := f.Stat(); statErr == nil {
			t.Errorf("%s: stale lock was not released (file still open)", name)
		}
	}
}

// A slug that is not a single path element must be refused before any directory
// or lock file is created, so the lock sink cannot be steered outside work/.
func TestFeatureLockRefusesInvalidSlug(t *testing.T) {
	for _, slug := range []string{"../evil", "a/b", `a\b`, "..", "."} {
		for name, lock := range map[string]func(root string, ran *bool) error{
			"WithFeatureLock": func(root string, ran *bool) error {
				return WithFeatureLock(root, slug, func() error { *ran = true; return nil })
			},
			"WithFeatureLockRelease": func(root string, ran *bool) error {
				return WithFeatureLockRelease(root, slug, func(func()) error { *ran = true; return nil })
			},
		} {
			root := t.TempDir()
			var ran bool
			err := lock(root, &ran)
			if err == nil || ran {
				t.Errorf("%s(%q) = %v, ran %v, want an error and fn not run", name, slug, err, ran)
			}
			if _, statErr := os.Stat(filepath.Join(root, "evil", ".lock")); statErr == nil {
				t.Errorf("%s(%q) created a lock file outside work/", name, slug)
			}
		}
	}
}
