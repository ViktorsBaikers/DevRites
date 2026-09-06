package main_test

// CLI black-box tests: build the real binary, run it as a subprocess against a
// fixture .devrites/, and assert stdout + exit code. This is the primary test
// seam: it exercises external behavior only, never internals.
//
// Each test runs against a fresh COPY of the fixture in a temp dir, so a test
// may hand-edit files without disturbing the committed fixture or another test.

import (
	"bytes"
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
		_ = os.RemoveAll(dir)
		panic("go build failed: " + err.Error() + "\n" + string(out))
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
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
	return runDevritesIO(t, root, "", nil, args...)
}

func runDevritesIO(t *testing.T, root, stdin string, extraEnv []string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(append(os.Environ(), "DEVRITES_ROOT="+root), extraEnv...)
	cmd.Stdin = strings.NewReader(stdin)
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

func TestUnknownCommandExitsNonZero(t *testing.T) {
	root := newWorkspace(t)
	if _, _, code := runDevrites(t, root, "frobnicate"); code == 0 {
		t.Fatalf("exit = 0, want non-zero for an unknown command")
	}
}

func TestCheckCandidateOutputAndMalformedFailure(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	workspace := filepath.Join(root, "work", "feature")
	testutil.WriteFile(t, filepath.Join(project, "source.go"), "package source\n")
	manifest := "# Touched files\n\n## Touched files\nCandidate paths are declared below.\n\n## Candidate manifest\n| State | File | Slice | Reason |\n| --- | --- | --- | --- |\n| present | `source.go` | S-1 | Implementation. |\n"
	testutil.WriteFile(t, filepath.Join(workspace, "touched-files.md"), manifest)

	out, errOut, code := runDevrites(t, root, "check", "candidate", "feature")
	if code != 0 || errOut != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errOut)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "candidate-sha256: ") || lines[1] != "candidate-files: 1" {
		t.Fatalf("stdout=%q", out)
	}

	testutil.WriteFile(t, filepath.Join(workspace, "touched-files.md"), "# Touched files\n\n## Touched files\n- `source.go`\n")
	out, errOut, code = runDevrites(t, root, "check", "candidate", "feature")
	if code != 3 || out != "" || !strings.Contains(errOut, "candidate manifest") {
		t.Fatalf("malformed: code=%d stdout=%q stderr=%q", code, out, errOut)
	}
}
