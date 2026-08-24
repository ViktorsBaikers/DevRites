package parallel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizePathRejectsDirty(t *testing.T) {
	t.Parallel()
	cases := []string{"", " ", "/", "/abs", `C:\windows`, `..`, "a/../b", "../x"}
	for _, raw := range cases {
		if _, err := NormalizePath(raw); err == nil {
			t.Fatalf("NormalizePath(%q) should fail", raw)
		}
	}
}

func TestNormalizePathAcceptsRelative(t *testing.T) {
	t.Parallel()
	got, err := NormalizePath(`src\foo.go`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "src/foo.go" {
		t.Fatalf("got %q", got)
	}
}

func TestCheckPathDisjointOverlap(t *testing.T) {
	t.Parallel()
	_, err := CheckPathDisjoint([]SlicePaths{
		{ID: "a", Paths: []string{"src/a.go"}},
		{ID: "b", Paths: []string{"src/a.go"}},
	}, "")
	if err == nil || !strings.Contains(err.Error(), "overlap") {
		t.Fatalf("expected overlap error, got %v", err)
	}
}

func TestCheckPathDisjointOK(t *testing.T) {
	t.Parallel()
	ids, err := CheckPathDisjoint([]SlicePaths{
		{ID: "a", Paths: []string{"src/a.go"}},
		{ID: "b", Paths: []string{"src/b.go"}},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("ids=%v", ids)
	}
}

func TestCheckPathDisjointRejectsDevritesPaths(t *testing.T) {
	t.Parallel()
	_, err := CheckPathDisjoint([]SlicePaths{
		{ID: "a", Paths: []string{".devrites/work/state.md"}},
		{ID: "b", Paths: []string{"src/b.go"}},
	}, "")
	if err == nil || !strings.Contains(err.Error(), ".devrites") {
		t.Fatalf("expected .devrites rejection error, got %v", err)
	}
}

func TestCheckPathDisjointSymlinkRoot(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "real.go")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.go")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks unavailable")
	}
	_, err := CheckPathDisjoint([]SlicePaths{
		{ID: "a", Paths: []string{"link.go"}},
		{ID: "b", Paths: []string{"other.go"}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got %v", err)
	}
}

func TestParseSlicesJSONShapes(t *testing.T) {
	t.Parallel()
	a, err := ParseSlicesJSON([]byte(`{"slices":[{"id":"a","paths":["x.go"]},{"id":"b","paths":["y.go"]}]}`))
	if err != nil || len(a) != 2 {
		t.Fatalf("object shape: %v %#v", err, a)
	}
	b, err := ParseSlicesJSON([]byte(`[{"id":"a","paths":["x.go"]},{"id":"b","paths":["y.go"]}]`))
	if err != nil || len(b) != 2 {
		t.Fatalf("array shape: %v %#v", err, b)
	}
}
