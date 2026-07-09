// Package testutil holds helpers shared across the engine's test packages.
package testutil

import (
	"os"
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
