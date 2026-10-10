package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteFileAtomicWritesAndOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "file.txt")
	if err := WriteFileAtomic(path, []byte("first\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "first\n" {
		t.Fatalf("content %q", got)
	}
	if runtime.GOOS != "windows" && mustMode(t, path).Perm() != 0o644 {
		t.Fatalf("perm %v", mustMode(t, path))
	}
	// Overwrite in place.
	if err := WriteFileAtomic(path, []byte("second\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != "second\n" {
		t.Fatalf("content after overwrite %q", got)
	}
	// No temp files left behind.
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if name := e.Name(); name != "file.txt" && name != "nested" {
			t.Fatalf("temp file left behind: %s", name)
		}
	}
}

func mustMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode()
}

func TestWriteFileAtomicSurfacesFailure(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The parent path is an existing file, so creating the destination
	// directory must fail and the error must surface to the caller.
	err := WriteFileAtomic(filepath.Join(blocker, "nested", "file.txt"), []byte("x"), 0o644)
	if err == nil {
		t.Fatal("expected the write failure to surface")
	}
	if !strings.Contains(err.Error(), "atomic write") {
		t.Fatalf("error should identify the operation, got %v", err)
	}
}

func TestWriteFileAtomicSyncsParentDirAfterRename(t *testing.T) {
	var calls []string
	var existedAtSync bool
	path := filepath.Join(t.TempDir(), "nested", "file.txt")
	orig := syncDir
	syncDir = func(dir string) error {
		calls = append(calls, dir)
		_, err := os.Stat(path)
		existedAtSync = err == nil
		return nil
	}
	t.Cleanup(func() { syncDir = orig })
	if err := WriteFileAtomic(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0] != filepath.Dir(path) {
		t.Fatalf("syncDir calls = %v, want one call with %q", calls, filepath.Dir(path))
	}
	if !existedAtSync {
		t.Fatal("syncDir ran before the rename")
	}
}

func TestWriteFileAtomicInRefusesSymlinkedDirectoryOutsideRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks needs elevated privileges on Windows")
	}
	base := t.TempDir()
	target := filepath.Join(base, "target")
	outside := filepath.Join(base, "outside")
	for _, dir := range []string{target, outside} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(target, ".claude")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(target)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	if err := WriteFileAtomicIn(root, filepath.Join(".claude", "skills", "x.md"), []byte("x"), 0o644); err == nil {
		t.Fatal("expected the write through a symlink leaving the root to fail")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("outside directory was written to: %v", entries)
	}
}

func TestWriteFileAtomicInWritesOverwritesAndLeavesNoTemp(t *testing.T) {
	target := t.TempDir()
	root, err := os.OpenRoot(target)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	rel := filepath.Join("a", "b", "f.txt")
	for _, content := range []string{"one", "two"} {
		if err := WriteFileAtomicIn(root, rel, []byte(content), 0o640); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(target, rel))
		if err != nil || string(got) != content {
			t.Fatalf("read = %q, %v; want %q", got, err, content)
		}
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(target, rel))
		if err != nil || info.Mode().Perm() != 0o640 {
			t.Fatalf("mode = %v, %v; want 0640", info, err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(target, "a", "b"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("directory entries = %v, %v; want only the target file", entries, err)
	}
}
