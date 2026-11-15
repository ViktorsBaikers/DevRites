package state

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

func TestFeatureIdentityLockNeverCreatesWorkspace(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	injected := errors.New("callback refused")
	if err := WithFeatureIdentityLockIn(root, "future", func() error {
		if _, err := root.Lstat("work"); !os.IsNotExist(err) {
			t.Fatalf("identity lock created work: %v", err)
		}
		return injected
	}); !errors.Is(err, injected) {
		t.Fatalf("callback error = %v", err)
	}
	if _, err := root.Lstat("work"); !os.IsNotExist(err) {
		t.Fatalf("identity lock left workspace: %v", err)
	}
	lockPath := filepath.Join(".locks", "features", "future", ".lock")
	before, err := root.Stat(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := WithFeatureLockIn(root, "future", func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	after, err := root.Stat(lockPath)
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("wrapper changed lock identity: %v", err)
	}
	if info, err := root.Lstat(filepath.Join("work", "future")); err != nil || !info.IsDir() {
		t.Fatalf("wrapper failed to create workspace: %v", err)
	}
	if err := root.Mkdir("archive", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := root.Rename(filepath.Join("work", "future"), filepath.Join("archive", "future")); err != nil {
		t.Fatal(err)
	}
	ran := false
	if err := WithFeatureIdentityLockIn(root, "future", func() error { ran = true; return nil }); err == nil || ran {
		t.Fatalf("identity lock accepted archive: err=%v ran=%v", err, ran)
	}
	if _, err := root.Lstat(filepath.Join("work", "future")); !os.IsNotExist(err) {
		t.Fatalf("identity lock recreated archived workspace: %v", err)
	}
	if _, err := root.Stat("."); err != nil {
		t.Fatalf("caller-owned root closed: %v", err)
	}
}

func TestFeatureIdentityLockSharesExistingWriterExclusion(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	const count = 40
	var workers sync.WaitGroup
	var active, value atomic.Int32
	for i := range count {
		workers.Add(1)
		go func() {
			defer workers.Done()
			lock := WithFeatureIdentityLockIn
			if i%2 == 0 {
				lock = WithFeatureLockIn
			}
			if err := lock(root, "future", func() error {
				if active.Add(1) != 1 {
					t.Error("mixed feature locks admitted overlapping writers")
				}
				defer active.Add(-1)
				runtime.Gosched()
				value.Add(1)
				return nil
			}); err != nil {
				t.Errorf("lock: %v", err)
			}
		}()
	}
	workers.Wait()
	if got := value.Load(); got != count {
		t.Fatalf("mixed-lock counter = %d; want %d", got, count)
	}
}

func TestFeatureIdentityLockRejectsInvalidSlugBeforeWrites(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	for _, slug := range []string{"", "../outside", "bad/slug"} {
		ran := false
		if err := WithFeatureIdentityLockIn(root, slug, func() error { ran = true; return nil }); err == nil || ran {
			t.Fatalf("invalid slug %q: err=%v ran=%v", slug, err, ran)
		}
	}
	if _, err := root.Lstat(".locks"); !os.IsNotExist(err) {
		t.Fatalf("invalid slug created lock hierarchy: %v", err)
	}
}
