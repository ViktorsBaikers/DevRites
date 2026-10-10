package lib

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func workspaceReadFixture(t *testing.T) (root, workspace string) {
	t.Helper()
	root = t.TempDir()
	workspace = filepath.Join(root, "work", "feat")
	if err := os.MkdirAll(workspace, 0o750); err != nil {
		t.Fatal(err)
	}
	return root, workspace
}

func TestReadWorkspaceArtifactReadsRegularFile(t *testing.T) {
	root, workspace := workspaceReadFixture(t)
	if err := os.WriteFile(filepath.Join(workspace, "tasks.md"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := readWorkspaceArtifact(root, "feat", "tasks.md")
	if err != nil || string(raw) != "ok" {
		t.Fatalf("raw=%q err=%v", raw, err)
	}
}

func TestReadWorkspaceArtifactMissingKeepsNotExist(t *testing.T) {
	root, _ := workspaceReadFixture(t)
	if _, err := readWorkspaceArtifact(root, "feat", "tasks.md"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err=%v, want os.ErrNotExist", err)
	}
}

func TestReadWorkspaceArtifactRejectsSymlinkLeaf(t *testing.T) {
	root, workspace := workspaceReadFixture(t)
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(workspace, "tasks.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	raw, err := readWorkspaceArtifact(root, "feat", "tasks.md")
	if err == nil {
		t.Fatalf("symlinked artifact was read: %q", raw)
	}
}

func TestReadWorkspaceArtifactRejectsFIFO(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mkfifo under Git Bash does not create a real FIFO")
	}
	root, workspace := workspaceReadFixture(t)
	if out, err := exec.Command("mkfifo", filepath.Join(workspace, "tasks.md")).CombinedOutput(); err != nil {
		t.Skipf("fifo unavailable: %v: %s", err, out)
	}
	done := make(chan error, 1)
	go func() {
		_, err := readWorkspaceArtifact(root, "feat", "tasks.md")
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO artifact read without error")
		}
	case <-time.After(time.Second):
		t.Fatal("readWorkspaceArtifact blocked on a FIFO")
	}
}

func TestReadWorkspaceArtifactMissingWorkspaceKeepsNotExist(t *testing.T) {
	if _, err := readWorkspaceArtifact(t.TempDir(), "absent", "tasks.md"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err=%v, want os.ErrNotExist", err)
	}
}

func TestReadWorkspaceArtifactRejectsOversize(t *testing.T) {
	root, workspace := workspaceReadFixture(t)
	path := filepath.Join(workspace, "tasks.md")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, workspaceArtifactLimit+1); err != nil {
		t.Fatal(err)
	}
	raw, err := readWorkspaceArtifact(root, "feat", "tasks.md")
	if err == nil {
		t.Fatalf("oversize artifact was read: %d bytes", len(raw))
	}
	if !strings.Contains(err.Error(), "tasks.md") || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("err=%v, want artifact name and limit message", err)
	}
}
