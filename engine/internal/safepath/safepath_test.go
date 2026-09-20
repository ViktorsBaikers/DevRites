package safepath

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWithinResolvedHandlesExistingSymlinksAndMissingTails(t *testing.T) {
	base := t.TempDir()
	parent := filepath.Join(base, "real")
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(parent, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	escape := filepath.Join(parent, "escape")
	if err := os.Symlink(outside, escape); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		candidate string
		parent    string
		want      bool
	}{
		{"same path", parent, parent, true},
		{"symlinked existing prefix", filepath.Join(link, "new", "file"), parent, true},
		{"missing parent tail", filepath.Join(parent, "new", "file"), filepath.Join(parent, "new"), true},
		{"symlink escape", filepath.Join(escape, "file"), parent, false},
		{"lexical escape", filepath.Join(parent, "..", "outside"), parent, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := WithinResolved(tc.candidate, tc.parent); got != tc.want {
				t.Fatalf("WithinResolved(%q, %q) = %v, want %v", tc.candidate, tc.parent, got, tc.want)
			}
		})
	}
}

func TestWithinResolvedDoesNotInventCaseFolding(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "CaseSensitiveName")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	wrongCase := filepath.Join(filepath.Dir(parent), strings.ToLower(filepath.Base(parent)))
	_, err := os.Stat(wrongCase)
	if err == nil {
		// The host filesystem treats the paths as the same. Native filepath
		// semantics are the authority on that host.
		if !WithinResolved(wrongCase, parent) {
			t.Fatalf("%s considers paths equivalent but containment disagrees", runtime.GOOS)
		}
		return
	}
	if WithinResolved(wrongCase, parent) {
		t.Fatalf("%s filesystem is case-sensitive; containment folded case", runtime.GOOS)
	}
}
