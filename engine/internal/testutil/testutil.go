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
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}
