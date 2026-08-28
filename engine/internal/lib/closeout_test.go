package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
	writeFile(t, filepath.Join(root, "work", "feat", "state.md"), "inside\n- Schema: 3\n")

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
	writeFile(t, filepath.Join(root, "work", "feat", "state.md"), "inside\n- Schema: 3\n")
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
	writeFile(t, filepath.Join(root, "work", "feat", "state.md"), "inside\n- Schema: 3\n")
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
