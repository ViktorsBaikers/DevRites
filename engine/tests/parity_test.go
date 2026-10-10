package main_test

// Golden CLI harness: run `devrites-engine <args>` against a fixture and compare its
// stdout, stderr and exit code to a recorded snapshot under testdata/golden. The
// snapshots were captured from the commands once they were proven correct, so a later
// change that alters observable behaviour fails here. Regenerate them deliberately
// with UPDATE_GOLDEN=1 (e.g. `UPDATE_GOLDEN=1 go test ./...`).
//
// stdout (the output the hook/command consumer reads) and the exit code are the
// contract; stderr is the operator-facing diagnostic and is recorded too, as a
// labelled section after the exit line, whenever the command wrote any. A refusal
// whose explanation is dropped, reworded or moved to stdout then fails here instead
// of passing as an exit code alone.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

var libEnv = []string{"LC_ALL=C", "DEVRITES_ROOT="}

func libRootEnv(work string) []string {
	return []string{"LC_ALL=C", "DEVRITES_ROOT=" + filepath.Join(work, ".devrites")}
}

func writeFile(t *testing.T, workdir, rel, content string) {
	t.Helper()
	testutil.WriteFile(t, filepath.Join(workdir, rel), content)
}

// lastStderr is the stderr of the most recent runArgv call, with the fixture's
// temp workdir masked so the snapshot stays reproducible. assertGolden appends it
// to the compared text when it is non-empty.
var lastStderr string

// assertGolden compares a command's output to the golden file for the current
// subtest name; assertGoldenKey does the same under an explicit key (for tests
// that snapshot more than one artifact, e.g. stdout plus a rewritten file).
func assertGolden(t *testing.T, stdout string, code int) {
	t.Helper()
	got := fmt.Sprintf("exit %d\n%s", code, stdout)
	if lastStderr != "" {
		if !strings.HasSuffix(got, "\n") {
			got += "\n"
		}
		got += "--- stderr ---\n" + lastStderr
	}
	assertGoldenKey(t, t.Name(), got)
}

func assertGoldenKey(t *testing.T, key, got string) {
	t.Helper()
	path := filepath.Join(engineRoot, "testdata", "golden", key+".golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing golden %s: regenerate with UPDATE_GOLDEN=1: %v", path, err)
	}
	wantText := strings.ReplaceAll(string(want), "\r\n", "\n")
	if got != wantText {
		t.Errorf("golden mismatch for %s\n got: %q\nwant: %q", key, got, wantText)
	}
}

// runArgv runs one command in workdir with extra env, returning stdout and exit
// code. A non-ExitError failure (e.g. binary not found) fails the test. The
// command's stderr is kept in lastStderr for assertGolden.
func runArgv(t *testing.T, workdir string, env []string, stdin string, name string, args ...string) (stdout string, code int) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = strings.NewReader(stdin)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("running %s %v: %v", name, args, err)
		}
		code = ee.ExitCode()
	}
	// Some diagnostics embed the fixture's temp workdir (e.g. the close-out
	// clobber refusal). Mask it, and the symlink-resolved form the engine prints
	// instead, so the recorded snapshot stays reproducible.
	stderr := errBuf.String()
	if workdir != "" {
		if resolved, err := filepath.EvalSymlinks(workdir); err == nil && resolved != workdir {
			stderr = strings.ReplaceAll(stderr, resolved, "<workdir>")
		}
		stderr = strings.ReplaceAll(stderr, workdir, "<workdir>")
	}
	lastStderr = stderr
	return out.String(), code
}
