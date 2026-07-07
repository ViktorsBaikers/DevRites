package main_test

// CLI black-box tests: build the real binary, run it as a subprocess against a
// fixture .devrites/, and assert stdout + exit code. This is the primary test
// seam — it exercises external behavior only, never internals.
//
// Each test runs against a fresh COPY of the fixture in a temp dir, so a test
// may write the index (state.db) or hand-edit a file without disturbing the
// committed fixture or another test.

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "devrites-bin-*")
	if err != nil {
		panic(err)
	}
	binPath = filepath.Join(dir, "devrites")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		panic("go build failed: " + err.Error() + "\n" + string(out))
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// newWorkspace copies the committed fixture workspace into a fresh temp dir and
// returns the path to the copied .devrites root. (The committed fixture dir is
// named devrites-root/ rather than .devrites/ only because the repo-root
// .gitignore ignores every .devrites/ directory; the engine keys off the
// DEVRITES_ROOT path, not the directory name.)
func newWorkspace(t *testing.T) string {
	t.Helper()
	src, err := filepath.Abs(filepath.Join("testdata", "fixtures", "basic", "devrites-root"))
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), ".devrites")
	testutil.CopyTree(t, src, root)
	return root
}

// runDevrites runs the built binary against the given .devrites root and returns
// stdout, stderr, and the exit code.
func runDevrites(t *testing.T, root string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), "DEVRITES_ROOT="+root)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	code = 0
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("running %v: %v", args, err)
		}
		code = ee.ExitCode()
	}
	return outBuf.String(), errBuf.String(), code
}

func goldenStatus(t *testing.T, slug string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "golden", "status", slug+".txt"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestStatusMatchesGolden(t *testing.T) {
	for _, slug := range []string{"auth-tokens", "search-ranking"} {
		slug := slug
		t.Run(slug, func(t *testing.T) {
			root := newWorkspace(t)
			out, errOut, code := runDevrites(t, root, "status", slug)
			if code != 0 {
				t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
			}
			if want := goldenStatus(t, slug); out != want {
				t.Errorf("stdout mismatch\n--- got ---\n%s--- want ---\n%s", out, want)
			}
		})
	}
}

func TestStatusUnknownSlugExitsNonZero(t *testing.T) {
	root := newWorkspace(t)
	out, errOut, code := runDevrites(t, root, "status", "does-not-exist")
	if code == 0 {
		t.Fatalf("exit = 0, want non-zero for unknown slug")
	}
	if out != "" {
		t.Errorf("stdout = %q, want empty on error", out)
	}
	if !bytes.Contains([]byte(errOut), []byte("not found")) {
		t.Errorf("stderr = %q, want it to mention 'not found'", errOut)
	}
}

func TestStatusMissingSlugArgExitsNonZero(t *testing.T) {
	root := newWorkspace(t)
	if _, _, code := runDevrites(t, root, "status"); code == 0 {
		t.Fatalf("exit = 0, want non-zero when slug arg is missing")
	}
}

func TestUnknownCommandExitsNonZero(t *testing.T) {
	root := newWorkspace(t)
	if _, _, code := runDevrites(t, root, "frobnicate"); code == 0 {
		t.Fatalf("exit = 0, want non-zero for an unknown command")
	}
}

// --- issue 02: SQLite index + reindex + staleness ---

func TestReindexBuildsDB(t *testing.T) {
	root := newWorkspace(t)
	_, errOut, code := runDevrites(t, root, "reindex")
	if code != 0 {
		t.Fatalf("reindex exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(root, "state.db")); err != nil {
		t.Errorf("state.db not created: %v", err)
	}
}

func TestIndexServedStatusMatchesGolden(t *testing.T) {
	root := newWorkspace(t)
	// Build the index first, so status is served from it rather than lazily.
	if _, _, code := runDevrites(t, root, "reindex"); code != 0 {
		t.Fatalf("reindex failed")
	}
	out, _, code := runDevrites(t, root, "status", "auth-tokens")
	if code != 0 {
		t.Fatalf("status exit = %d, want 0", code)
	}
	if want := goldenStatus(t, "auth-tokens"); out != want {
		t.Errorf("index-served status != files-only golden\n--- got ---\n%s--- want ---\n%s", out, want)
	}
}

func TestStalenessHealReflectsHandEdit(t *testing.T) {
	root := newWorkspace(t)
	if _, _, code := runDevrites(t, root, "reindex"); code != 0 {
		t.Fatalf("reindex failed")
	}
	// Before: tasks is a heading-only stub → incomplete.
	if out, _, _ := runDevrites(t, root, "status", "auth-tokens"); !bytes.Contains([]byte(out), []byte("result: incomplete")) {
		t.Fatalf("precondition failed, expected incomplete, got:\n%s", out)
	}
	// Hand-edit the file to add real content — no explicit reindex.
	tasks := filepath.Join(root, "features", "auth-tokens", "tasks.md")
	appendFile(t, tasks, "\n- [x] mint\n- [x] verify\n")

	out, _, code := runDevrites(t, root, "status", "auth-tokens")
	if code != 0 {
		t.Fatalf("status exit = %d, want 0", code)
	}
	if !bytes.Contains([]byte(out), []byte("  tasks      present")) {
		t.Errorf("staleness heal: tasks still not present after edit\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("result: complete")) {
		t.Errorf("staleness heal: expected complete after edit\n%s", out)
	}
}

func TestDeleteDBReindexIsByteIdentical(t *testing.T) {
	root := newWorkspace(t)
	first, _, code := runDevrites(t, root, "status", "auth-tokens") // lazily builds the DB
	if code != 0 {
		t.Fatalf("first status exit = %d", code)
	}
	for _, sidecar := range []string{"state.db", "state.db-wal", "state.db-shm"} {
		_ = os.Remove(filepath.Join(root, sidecar))
	}
	if _, _, code := runDevrites(t, root, "reindex"); code != 0 {
		t.Fatalf("reindex failed")
	}
	second, _, code := runDevrites(t, root, "status", "auth-tokens")
	if code != 0 {
		t.Fatalf("second status exit = %d", code)
	}
	if first != second {
		t.Errorf("status changed across delete+reindex\n--- first ---\n%s--- second ---\n%s", first, second)
	}
	if want := goldenStatus(t, "auth-tokens"); second != want {
		t.Errorf("post-reindex status != golden\n%s", second)
	}
}

func TestCorruptDBIsRebuilt(t *testing.T) {
	root := newWorkspace(t)
	if _, _, code := runDevrites(t, root, "reindex"); code != 0 {
		t.Fatalf("reindex failed")
	}
	// Corrupt the DB with garbage; a well-behaved cache must drop and rebuild it.
	if err := os.WriteFile(filepath.Join(root, "state.db"), []byte("not a sqlite database at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errOut, code := runDevrites(t, root, "status", "auth-tokens")
	if code != 0 {
		t.Fatalf("status on corrupt DB exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if want := goldenStatus(t, "auth-tokens"); out != want {
		t.Errorf("status after corrupt-DB rebuild != golden\n%s", out)
	}
}

func appendFile(t *testing.T, path, text string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(text); err != nil {
		t.Fatal(err)
	}
}
