package lib

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/devrites/devrites/internal/state"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func runCloseOut(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := CloseOut(root, args, stdout, stderr)
	return code, stdout.String() + stderr.String()
}

func TestCloseOutRejectsRootOverrideArgument(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "work", "feat", "state.md"), "inside\n- Schema: 4\n")

	code, out := runCloseOut(t, root, "feat", t.TempDir())
	if code != 4 || !strings.Contains(out, "usage: devrites-engine state close <slug>") {
		t.Fatalf("state close override = %d, want usage refusal\n%s", code, out)
	}
	if !isFile(filepath.Join(root, "work", "feat", "state.md")) {
		t.Fatal("state close mutated the workspace after an extra root argument")
	}
}

func TestCloseOutRejectsWorkspaceSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "work", "feat")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	writeFile(t, filepath.Join(outside, "state.md"), "outside\n")

	code, out := runCloseOut(t, root, "feat")
	if code != 1 {
		t.Fatalf("close-out = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "invalid workspace") {
		t.Fatalf("missing workspace diagnostic:\n%s", out)
	}
	if !isFile(filepath.Join(outside, "state.md")) {
		t.Fatal("close-out moved a workspace through an external symlink")
	}
}

func TestCloseOutRejectsArchiveSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFile(t, filepath.Join(root, "work", "feat", "state.md"), "inside\n- Schema: 4\n")
	if err := os.Symlink(outside, filepath.Join(root, "archive")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	code, out := runCloseOut(t, root, "feat")
	if code != 1 {
		t.Fatalf("close-out = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "invalid archive") {
		t.Fatalf("missing archive diagnostic:\n%s", out)
	}
	if !isFile(filepath.Join(root, "work", "feat", "state.md")) {
		t.Fatal("close-out moved the workspace through an external archive symlink")
	}
	if entries, err := os.ReadDir(outside); err != nil {
		t.Fatal(err)
	} else if len(entries) != 0 {
		t.Fatalf("outside archive changed: %v", entries)
	}
}

func TestCloseOutRollsBackArchiveWhenActiveCannotBeCleared(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission model differs on Windows")
	}
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "work", "feat", "state.md"), "inside\n- Schema: 4\n")
	writeFile(t, filepath.Join(root, "ACTIVE"), "feat\n")
	if err := os.MkdirAll(filepath.Join(root, "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(root, 0o755) }()

	code, out := runCloseOut(t, root, "feat")
	if code != 1 {
		t.Fatalf("close-out = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "archive move rolled back") {
		t.Fatalf("missing rollback diagnostic:\n%s", out)
	}
	if !isFile(filepath.Join(root, "work", "feat", "state.md")) {
		t.Fatal("workspace was not restored after ACTIVE clear failed")
	}
	if _, err := os.Lstat(filepath.Join(root, "archive", "feat")); !os.IsNotExist(err) {
		t.Fatalf("archive remained after rollback: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "ACTIVE"))
	if err != nil || string(data) != "feat\n" {
		t.Fatalf("ACTIVE changed after rollback: %q, %v", data, err)
	}
}

// TestCloseOutSerializesRenameBehindFeatureLock holds the per-feature lock in a
// concurrent writer and requires close-out to wait for it. Without the lock the
// rename lands while the writer is mid-save, and the writer's MkdirAll re-creates
// work/<slug> after the workspace has been archived.
//
// The barrier is channel-based: `locked` proves the writer owns the lock before
// close-out starts, and `release` hands the workspace back only after close-out
// has stayed blocked for the bounded wait below. That wait never signals
// success: success is proven by the archive carrying the writer's file and
// work/<slug> being gone. It only bounds how long an unsynchronized rename may
// take to appear, so a missing lock fails fast instead of hanging.
func TestCloseOutSerializesRenameBehindFeatureLock(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "work", "feat", "state.md"), "inside\n- Schema: 4\n")
	writeFile(t, filepath.Join(root, "ACTIVE"), "feat\n")

	// dispatch.json is what a concurrent `dispatch` mid-save writes; it is
	// written the way saveDispatch writes it, MkdirAll included, so the
	// resurrection path is exercised rather than simulated.
	dispatchPath := filepath.Join(root, "work", "feat", "dispatch.json")
	locked := make(chan struct{})
	release := make(chan struct{})
	wrote := make(chan error, 1)
	writerDone := make(chan error, 1)
	go func() {
		writerDone <- state.WithFeatureLock(root, "feat", func() error {
			close(locked)
			<-release
			// Still holding the lock, as dispatch does across its load-modify-save.
			if err := os.MkdirAll(filepath.Dir(dispatchPath), 0o755); err != nil {
				wrote <- err
				return err
			}
			wrote <- state.AtomicWrite(dispatchPath, []byte(`{"phase":"start"}`), 0o600)
			return nil
		})
	}()
	<-locked

	closeDone := make(chan int, 1)
	go func() { closeDone <- CloseOut(root, []string{"feat"}, io.Discard, io.Discard) }()

	select {
	case code := <-closeDone:
		t.Fatalf("state close returned %d while an in-process writer held %s: "+
			"the rename is not serialized behind the feature lock, so a concurrent "+
			"writer can re-create work/feat after the workspace has been archived",
			code, filepath.Join("work", "feat", ".lock"))
	case <-time.After(750 * time.Millisecond):
		// Still blocked on the lock: correct.
	}

	// Release the writer only once close-out is demonstrably parked, then let the
	// writer's save land. With the lock held across the rename, the rename cannot
	// have happened yet, so this file is archived rather than left behind.
	close(release)
	if err := <-wrote; err != nil {
		t.Fatalf("concurrent dispatch save: %v", err)
	}
	if err := <-writerDone; err != nil {
		t.Fatalf("concurrent writer lock: %v", err)
	}

	var code int
	select {
	case code = <-closeDone:
	case <-time.After(10 * time.Second):
		t.Fatal("state close did not finish after the feature lock was released")
	}
	if code != 0 {
		t.Fatalf("state close = %d, want 0 after the concurrent writer released the lock\n", code)
	}
	if !isFile(filepath.Join(root, "archive", "feat", "dispatch.json")) {
		t.Fatal("concurrent dispatch.json was not archived: the rename did not run after the writer's save")
	}
	if _, err := os.Lstat(filepath.Join(root, "work", "feat")); !os.IsNotExist(err) {
		t.Fatalf("work/feat was re-created by the concurrent writer after close-out: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(root, "ACTIVE")); err != nil || len(data) != 0 {
		t.Fatalf("ACTIVE = %q (%v), want cleared", data, err)
	}
}

// TestCloseOutReleasesFeatureLockBeforeRenameWhenPlatformRequires injects the
// Windows decision and checks, from inside the rename, that the feature lock is
// free: another locker acquires it while the workspace still exists. With the
// lock still held at the rename that locker stays blocked.
func TestCloseOutReleasesFeatureLockBeforeRenameWhenPlatformRequires(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "work", "feat", "state.md"), "inside\n- Schema: 4\n")
	writeFile(t, filepath.Join(root, "ACTIVE"), "feat\n")

	origRelease, origRename := releaseLockBeforeRename, renameWorkspace
	t.Cleanup(func() { releaseLockBeforeRename, renameWorkspace = origRelease, origRename })
	releaseLockBeforeRename = true

	var lockFreeAtRename bool
	renameWorkspace = func(from, to string) error {
		acquired := make(chan struct{})
		done := make(chan error, 1)
		go func() {
			done <- state.WithFeatureLock(root, "feat", func() error {
				close(acquired)
				return nil
			})
		}()
		select {
		case <-acquired:
			lockFreeAtRename = true
			<-done
		case <-time.After(750 * time.Millisecond):
			// Still blocked: the rename is about to run with the lock held.
			// The locker is drained after CloseOut returns.
			t.Cleanup(func() { <-done })
		}
		return os.Rename(from, to)
	}

	code, out := runCloseOut(t, root, "feat")
	if code != 0 {
		t.Fatalf("state close = %d, want 0\n%s", code, out)
	}
	if !lockFreeAtRename {
		t.Fatal("the feature lock was still held when the workspace was renamed")
	}
	if !isFile(filepath.Join(root, "archive", "feat", "state.md")) {
		t.Fatal("workspace was not archived")
	}
}

func TestCloseOutMissingWorkspaceNamesResolvedPath(t *testing.T) {
	root := t.TempDir()

	code, out := runCloseOut(t, root, "ghost")
	want := filepath.Join(root, "work", "ghost")
	if code != 4 || !strings.Contains(out, "ghost") || !strings.Contains(out, want) {
		t.Fatalf("state close missing workspace = %d, want exit 4 naming %q\n%s", code, want, out)
	}
}
