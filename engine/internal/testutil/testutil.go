// Package testutil holds helpers shared across the engine's test packages.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// CopyTree recursively copies the directory tree at src into dst, creating dst
// and any parents. It is used to give each test an isolated, writable copy of a
// read-only fixture.
func CopyTree(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
}

// WriteFile writes data to path with mode 0644, creating parent directories.
func WriteFile(t *testing.T, path, data string) {
	t.Helper()
	WriteFileMode(t, path, data, 0o644)
}

// WriteExecutable writes data to path with mode 0755, creating parent directories.
func WriteExecutable(t *testing.T, path, data string) {
	t.Helper()
	WriteFileMode(t, path, data, 0o755)
}

// WriteFileMode writes data to path with the given permissions, creating parent
// directories. It fails the test on any error.
func WriteFileMode(t *testing.T, path, data string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), perm); err != nil {
		t.Fatal(err)
	}
}

// AppendFile appends data to an existing file, failing the test if the file is
// missing or unwritable.
func AppendFile(t *testing.T, path, data string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(data); err != nil {
		t.Fatal(err)
	}
}

// ReadFile returns the contents of path, failing the test on any error.
func ReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
