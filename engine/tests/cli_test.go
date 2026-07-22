package main_test

// CLI black-box tests: build the real binary, run it as a subprocess against a
// fixture .devrites/, and assert stdout + exit code. This is the primary test
// seam: it exercises external behavior only, never internals.
//
// Each test runs against a fresh COPY of the fixture in a temp dir, so a test
// may hand-edit files without disturbing the committed fixture or another test.

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

var (
	binPath    string
	engineRoot string
)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	engineRoot = filepath.Clean(filepath.Join(wd, ".."))
	dir, err := os.MkdirTemp("", "devrites-bin-*")
	if err != nil {
		panic(err)
	}
	name := "devrites"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binPath = filepath.Join(dir, name)
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = engineRoot
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
	src, err := filepath.Abs(filepath.Join(engineRoot, "testdata", "fixtures", "basic", "devrites-root"))
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
	b, err := os.ReadFile(filepath.Join(engineRoot, "testdata", "golden", "status", slug+".txt"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
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

func TestSnapshotEmitsDevRitesWorkspaceJSON(t *testing.T) {
	root := newWorkspace(t)
	out, errOut, code := runDevrites(t, root, "snapshot", "auth-tokens")
	if code != 0 {
		t.Fatalf("snapshot exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	var got struct {
		SchemaVersion   string   `json:"schemaVersion"`
		Slug            string   `json:"slug"`
		Phase           string   `json:"phase"`
		RunMode         string   `json:"runMode"`
		Complete        bool     `json:"complete"`
		MissingSections []string `json:"missingSections"`
		NextCommand     string   `json:"nextCommand"`
		NextCommands    struct {
			Verb   string `json:"verb"`
			Claude string `json:"claude"`
			Codex  string `json:"codex"`
		} `json:"nextCommands"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("snapshot output is not JSON: %v\n%s", err, out)
	}
	if got.SchemaVersion != "devrites.workspace.v1" {
		t.Fatalf("schemaVersion = %q, want devrites.workspace.v1", got.SchemaVersion)
	}
	if got.Slug != "auth-tokens" || got.Phase != "build" || got.RunMode != "HITL" {
		t.Fatalf("unexpected identity fields: %+v", got)
	}
	if got.Complete {
		t.Fatalf("auth-tokens fixture should be incomplete before tasks are filled")
	}
	if len(got.MissingSections) != 1 || got.MissingSections[0] != "tasks" {
		t.Fatalf("missingSections = %#v, want [tasks]", got.MissingSections)
	}
	if got.NextCommand != "/rite-build" {
		t.Fatalf("nextCommand = %q, want /rite-build", got.NextCommand)
	}
	if got.NextCommands.Verb != "build" || got.NextCommands.Claude != "/rite-build" || got.NextCommands.Codex != "$rite-build" {
		t.Fatalf("nextCommands = %+v, want build with Claude/Codex forms", got.NextCommands)
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

func TestStatusReflectsHandEdit(t *testing.T) {
	root := newWorkspace(t)
	// Before: tasks is a heading-only stub → incomplete.
	if out, _, _ := runDevrites(t, root, "status", "auth-tokens"); !bytes.Contains([]byte(out), []byte("result: incomplete")) {
		t.Fatalf("precondition failed, expected incomplete, got:\n%s", out)
	}
	// Hand-edit the file to add real content; status reads files directly.
	tasks := filepath.Join(root, "features", "auth-tokens", "tasks.md")
	testutil.AppendFile(t, tasks, "\n- [x] mint\n- [x] verify\n")

	out, _, code := runDevrites(t, root, "status", "auth-tokens")
	if code != 0 {
		t.Fatalf("status exit = %d, want 0", code)
	}
	if !bytes.Contains([]byte(out), []byte("  tasks      present")) {
		t.Errorf("tasks still not present after edit\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("result: complete")) {
		t.Errorf("expected complete after edit\n%s", out)
	}
}
