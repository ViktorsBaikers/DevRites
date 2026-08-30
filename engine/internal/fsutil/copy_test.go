package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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
	if !PermissionsMatch(mustMode(t, path), 0o644) {
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

func TestFileModTime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	modified, ok := FileModTime(path)
	if !ok {
		t.Fatal("regular file should report mod time")
	}
	if modified <= 0 {
		t.Fatalf("mod time %d", modified)
	}
	if _, ok := FileModTime(filepath.Join(dir, "missing.txt")); ok {
		t.Fatal("missing file should report !ok")
	}
	if _, ok := FileModTime(dir); ok {
		t.Fatal("directory should report !ok")
	}
}

func TestNewestModTime(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "old.txt")
	new := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(old, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(new, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Hour)
	if err := os.Chtimes(new, future, future); err != nil {
		t.Skipf("cannot set file times on this platform: %v", err)
	}
	newest, ok := NewestModTime(old, new)
	if !ok {
		t.Fatal("expected ok")
	}
	want, _ := FileModTime(new)
	if newest != want {
		t.Fatalf("newest=%d want %d", newest, want)
	}
	if _, ok := NewestModTime(filepath.Join(dir, "missing.txt")); ok {
		t.Fatal("missing-only inputs should report !ok")
	}
}

func TestCopyTreeMissingSourceIsNoOp(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "dst")
	if err := CopyTree(filepath.Join(dir, "missing"), dst); err != nil {
		t.Fatalf("missing src must not error: %v", err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("dst must not be created, got %v", err)
	}
}

func TestCopyTreeFileAndTree(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "root.txt"), []byte("root\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "leaf.txt"), []byte("leaf\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "dst")
	if err := CopyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	for rel, want := range map[string]string{
		"root.txt":     "root\n",
		"sub/leaf.txt": "leaf\n",
	} {
		got, err := os.ReadFile(filepath.Join(dst, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		if string(got) != want {
			t.Fatalf("%s: %q want %q", rel, got, want)
		}
	}

	// Copying a single file copies its contents.
	single := filepath.Join(dir, "single.txt")
	if err := os.WriteFile(single, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	singleDst := filepath.Join(dir, "single-copy.txt")
	if err := CopyTree(single, singleDst); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(singleDst); string(got) != "one\n" {
		t.Fatalf("single copy %q", got)
	}

	// Recopying overwrites existing files.
	if err := os.WriteFile(filepath.Join(src, "root.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dst, "root.txt")); string(got) != "changed\n" {
		t.Fatalf("overwrite %q", got)
	}
}

func TestPermissionsMatch(t *testing.T) {
	if !PermissionsMatch(0o644, 0o644) {
		t.Fatal("matching perms must match")
	}
	if runtime.GOOS != "windows" && PermissionsMatch(0o600, 0o644) {
		t.Fatal("differing POSIX perms must not match")
	}
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
