//go:build unix || windows

package state

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
		root := t.TempDir()
		var ran bool
		err := WithFeatureLock(root, slug, func() error { ran = true; return nil })
		if err == nil || ran {
			t.Errorf("WithFeatureLock(%q) = %v, ran %v, want an error and fn not run", slug, err, ran)
		}
		if _, statErr := os.Stat(filepath.Join(root, "evil", ".lock")); statErr == nil {
			t.Errorf("WithFeatureLock(%q) created a lock file outside work/", slug)
		}
	}
}

// Locking a closed (archived) workspace must not recreate it.
func TestWithFeatureLockRefusesClosedWorkspace(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "archive", "feat"), 0o755); err != nil {
		t.Fatal(err)
	}
	ran := false
	if err := WithFeatureLock(root, "feat", func() error { ran = true; return nil }); err == nil || ran {
		t.Fatalf("lock on a missing workspace: err=%v ran=%v, want refused", err, ran)
	}
	if _, err := os.Lstat(filepath.Join(root, "work", "feat")); !os.IsNotExist(err) {
		t.Fatalf("work/feat recreated by the lock: %v", err)
	}
}

func TestFeatureLockRefusesRecreatedClosedWorkspace(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"archive", "work"} {
		if err := os.MkdirAll(filepath.Join(root, dir, "feat"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ran := false
	if err := WithFeatureLock(root, "feat", func() error { ran = true; return nil }); err == nil || ran {
		t.Fatalf("lock on recreated closed workspace: err=%v ran=%v, want refused", err, ran)
	}
}

func TestFeatureLockAllowsArchiveWhileHeld(t *testing.T) {
	root := t.TempDir()
	err := WithFeatureLock(root, "feat", func() error {
		if err := os.MkdirAll(filepath.Join(root, "archive"), 0o755); err != nil {
			return err
		}
		return os.Rename(filepath.Join(root, "work", "feat"), filepath.Join(root, "archive", "feat"))
	})
	if err != nil {
		t.Fatalf("archive while feature lock held: %v", err)
	}
	if err := WithFeatureLock(root, "feat", func() error {
		t.Error("late writer ran after archive")
		return nil
	}); err == nil {
		t.Fatal("late writer accepted archived workspace")
	}
	if _, err := os.Lstat(filepath.Join(root, "work", "feat")); !os.IsNotExist(err) {
		t.Fatalf("late writer recreated workspace: %v", err)
	}
}

func TestFeatureLockPreservesMaximumLengthSlug(t *testing.T) {
	slug := strings.Repeat("a", 255)
	ran := false
	if err := WithFeatureLock(t.TempDir(), slug, func() error { ran = true; return nil }); err != nil || !ran {
		t.Fatalf("255-byte feature slug: err=%v ran=%v", err, ran)
	}
}

func TestFeatureLockRefusesEscapingParents(t *testing.T) {
	for _, parent := range []string{"work", ".locks"} {
		for _, retained := range []bool{false, true} {
			name := parent + "/path"
			if retained {
				name = parent + "/root"
			}
			t.Run(name, func(t *testing.T) {
				dir, external := t.TempDir(), t.TempDir()
				if err := os.Symlink(external, filepath.Join(dir, parent)); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
				root, err := os.OpenRoot(dir)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = root.Close() }()
				ran := false
				fn := func() error { ran = true; return nil }
				if retained {
					err = WithFeatureLockIn(root, "feat", fn)
				} else {
					err = WithFeatureLock(dir, "feat", fn)
				}
				if err == nil || ran {
					t.Fatalf("escaping %s accepted: err=%v ran=%v", parent, err, ran)
				}
				entries, err := os.ReadDir(external)
				if err != nil {
					t.Fatal(err)
				}
				if len(entries) != 0 {
					t.Fatalf("escaping %s created external entries: %v", parent, entries)
				}
			})
		}
	}
}

func TestWithFeatureLockInCreatesAndRefusesArchivedWorkspace(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	ran := false
	if err := WithFeatureLockIn(root, "feat", func() error {
		ran = true
		info, err := root.Lstat(filepath.Join("work", "feat"))
		if err != nil {
			return err
		}
		if !info.IsDir() {
			t.Fatal("workspace is not a directory")
		}
		return nil
	}); err != nil || !ran {
		t.Fatalf("feature creation: err=%v ran=%v", err, ran)
	}
	lockPath := filepath.Join(".locks", "features", "feat", ".lock")
	before, err := root.Lstat(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	ran = false
	if err := WithFeatureLockIn(root, "feat", func() error { ran = true; return nil }); err != nil || !ran {
		t.Fatalf("existing feature lock: err=%v ran=%v", err, ran)
	}
	after, err := root.Lstat(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("existing feature lock replaced")
	}
	if err := root.MkdirAll("archive", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := root.Rename(filepath.Join("work", "feat"), filepath.Join("archive", "feat")); err != nil {
		t.Fatal(err)
	}
	ran = false
	if err := WithFeatureLockIn(root, "feat", func() error { ran = true; return nil }); err == nil || ran {
		t.Fatalf("archived feature accepted: err=%v ran=%v", err, ran)
	}
	if _, err := root.Lstat(filepath.Join("work", "feat")); !os.IsNotExist(err) {
		t.Fatalf("archived workspace recreated: %v", err)
	}
	if _, err := root.Stat("."); err != nil {
		t.Fatalf("caller-owned root closed: %v", err)
	}
}

func TestWithFeatureLockCreatesMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	ran := false
	if err := WithFeatureLock(root, "feat", func() error {
		ran = true
		info, err := os.Stat(filepath.Join(root, "work", "feat"))
		if err != nil {
			return err
		}
		if !info.IsDir() {
			t.Fatal("workspace is not a directory")
		}
		return nil
	}); err != nil || !ran {
		t.Fatalf("feature creation in missing root: err=%v ran=%v", err, ran)
	}
	if _, err := os.Stat(filepath.Join(root, ".locks", "features", "feat", ".lock")); err != nil {
		t.Fatalf("feature lock not created: %v", err)
	}
}
