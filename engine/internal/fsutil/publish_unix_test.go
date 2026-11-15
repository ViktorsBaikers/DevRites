//go:build (linux || darwin) && (amd64 || arm64)

package fsutil

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestPublishDirRejectsSpecialEntries(t *testing.T) {
	root := publicationRoot(t)
	stage := publicationStage(t, root, "candidate", "complete")
	if err := syscall.Mkfifo(filepath.Join(root.Name(), stage, "pipe"), 0o600); err != nil {
		t.Fatal(err)
	}
	if published, err := PublishDirNoReplace(root, stage, filepath.Join("work", "target")); published || err == nil {
		t.Fatalf("FIFO accepted: %v, %v", published, err)
	}
	if _, err := root.Lstat(filepath.Join("work", "target")); !os.IsNotExist(err) {
		t.Fatalf("special-entry refusal published target: %v", err)
	}
}

func TestPublishDirUsesRetainedRootAfterRename(t *testing.T) {
	container := t.TempDir()
	original := filepath.Join(container, "root")
	if err := os.Mkdir(original, 0o700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(original)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	for _, dir := range []string{"staging", "work"} {
		if err := root.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	stage := publicationStage(t, root, "candidate", "complete")
	if err := os.Rename(original, filepath.Join(container, "moved")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(original, 0o700); err != nil {
		t.Fatal(err)
	}
	if published, err := PublishDirNoReplace(root, stage, filepath.Join("work", "target")); !published || err != nil {
		t.Fatalf("retained root publish = %v, %v", published, err)
	}
	if err := SyncPublishedDir(root, filepath.Join("work", "target")); err != nil {
		t.Fatal(err)
	}
	assertPublicationBody(t, root, filepath.Join("work", "target"), "complete")
	entries, err := os.ReadDir(original)
	if err != nil || len(entries) != 0 {
		t.Fatalf("used stale root.Name(): %v, %v", entries, err)
	}
}
