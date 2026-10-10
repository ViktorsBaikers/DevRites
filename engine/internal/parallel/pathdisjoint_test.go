package parallel

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizePathRejectsDirty(t *testing.T) {
	t.Parallel()
	cases := []string{"", " ", "/", "/abs", `C:\windows`, `..`, "a/../b", "../x",
		// Segment-edge whitespace previously survived normalization
		// ("0 /." collapsed to "0 "), making the mapping non-idempotent.
		"0 /.", "src /a.go", "./ \n/x"}
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
	for _, c := range [][2]string{{"src", "src/a.go"}, {"src/a.go", "src"}, {"src/x", "src/x/y/z.go"}} {
		_, err := CheckPathDisjoint([]SlicePaths{
			{ID: "a", Paths: []string{c[0]}},
			{ID: "b", Paths: []string{c[1]}},
		}, "")
		if err == nil || !strings.Contains(err.Error(), "overlap") {
			t.Fatalf("%q vs %q: expected overlap error, got %v", c[0], c[1], err)
		}
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

// A symlinked directory inside a claimed path must also be rejected: the
// leaf Lstat alone misses it, and a worktree would then write outside itself.
func TestCheckPathDisjointSymlinkedDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "outside")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "linkdir")); err != nil {
		t.Skip("symlinks unavailable")
	}
	_, err := CheckPathDisjoint([]SlicePaths{
		{ID: "a", Paths: []string{"linkdir/x.go"}},
		{ID: "b", Paths: []string{"other.go"}},
	}, dir)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error for symlinked dir, got %v", err)
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

func TestCheckPathDisjointOverlapSentinel(t *testing.T) {
	t.Parallel()
	_, err := CheckPathDisjoint([]SlicePaths{
		{ID: "a", Paths: []string{"src/a.go"}},
		{ID: "b", Paths: []string{"src/a.go"}},
	}, "")
	if !errors.Is(err, ErrPathOverlap) {
		t.Fatalf("expected ErrPathOverlap, got %v", err)
	}
}

// A hard refusal whose message merely mentions "overlap" (here via the slice
// id) must not be classified as a path overlap.
func TestCheckPathDisjointSymlinkNotOverlap(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "real")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "linkdir")); err != nil {
		t.Skip("symlinks unavailable")
	}
	slices := []SlicePaths{
		{ID: "overlap-a", Paths: []string{"linkdir/x.go"}},
		{ID: "b", Paths: []string{"other.go"}},
	}
	_, err := CheckPathDisjoint(slices, dir)
	if err == nil || errors.Is(err, ErrPathOverlap) {
		t.Fatalf("expected non-overlap refusal, got %v", err)
	}
	if _, err := SelectGreedy(2, slices, dir); err == nil || errors.Is(err, ErrPathOverlap) {
		t.Fatalf("SelectGreedy must return the hard refusal, got %v", err)
	}
}

// SelectGreedy must return a non-overlap refusal from the trial check even when
// its message mentions "overlap", and must still skip a real overlap.
func TestSelectGreedyTrialErrorClassification(t *testing.T) {
	orig := checkPathDisjoint
	t.Cleanup(func() { checkPathDisjoint = orig })
	ready := []SlicePaths{
		{ID: "a", Paths: []string{"a.go"}},
		{ID: "b", Paths: []string{"b.go"}},
	}

	hard := errors.New("slice overlap-a: cannot verify paths")
	checkPathDisjoint = func([]SlicePaths, string) ([]string, error) { return nil, hard }
	if _, err := SelectGreedy(2, ready, ""); !errors.Is(err, hard) {
		t.Fatalf("expected the hard refusal to be returned, got %v", err)
	}

	checkPathDisjoint = func([]SlicePaths, string) ([]string, error) {
		return nil, fmt.Errorf("%w: x", ErrPathOverlap)
	}
	got, err := SelectGreedy(2, ready, "")
	if err != nil || len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("real overlap must be skipped, got %v, %v", got, err)
	}
}
