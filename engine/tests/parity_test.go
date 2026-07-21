package main_test

// Golden CLI harness: run `devrites-engine <args>` against a fixture and compare its
// stdout + exit code to a recorded snapshot under testdata/golden/. The snapshots
// were captured from the commands once they were proven correct, so a later change
// that alters observable behaviour fails here. Regenerate them deliberately with
// UPDATE_GOLDEN=1 (e.g. `UPDATE_GOLDEN=1 go test ./...`).
//
// stderr is intentionally NOT captured: it is diagnostic, not contract. Only
// stdout (the output the hook/command consumer reads) and the exit code are.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// parityCase describes one fixture run of the devrites-engine binary.
type parityCase struct {
	workdir string   // cwd the command runs in
	env     []string // extra environment
	stdin   string   // payload piped to stdin (hooks read stdin)
	goArgs  []string // args to the built devrites-engine binary (binPath is prepended)
}

// assertEqual runs the command and compares its stdout + exit code to the golden
// snapshot for the current subtest.
func (c parityCase) assertEqual(t *testing.T) {
	t.Helper()
	out, code := runArgv(t, c.workdir, c.env, c.stdin, binPath, c.goArgs...)
	assertGolden(t, out, code)
}

// assertGolden compares a command's output to the golden file for the current
// subtest name; assertGoldenKey does the same under an explicit key (for tests
// that snapshot more than one artifact, e.g. stdout plus a rewritten file).
func assertGolden(t *testing.T, stdout string, code int) {
	t.Helper()
	assertGoldenKey(t, t.Name(), fmt.Sprintf("exit %d\n%s", code, stdout))
}

func assertGoldenKey(t *testing.T, key, got string) {
	t.Helper()
	path := filepath.Join(engineRoot, "testdata", "golden", key+".golden")
	if os.Getenv("UPDATE_GOLDEN") != "" {
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
		t.Fatalf("missing golden %s — regenerate with UPDATE_GOLDEN=1: %v", path, err)
	}
	wantText := strings.ReplaceAll(string(want), "\r\n", "\n")
	if got != wantText {
		t.Errorf("golden mismatch for %s\n got: %q\nwant: %q", key, got, wantText)
	}
}

// runArgv runs one command in workdir with extra env, returning stdout and exit
// code. A non-ExitError failure (e.g. binary not found) fails the test.
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
	return out.String(), code
}

// TestParityReviewerReadonly checks `hook reviewer-readonly` against golden
// snapshots for each payload: the silent no-ops, the full enforce-mode deny, the
// observe-mode side effect (asserted only on its empty stdout+exit), and the
// agent-required gate. The golden check compares stdout+exit, not files.
func TestParityReviewerReadonly(t *testing.T) {
	cases := []struct {
		name  string
		env   []string
		stdin string
	}{
		{"non-tool-input", nil, "not even json"},
		{"non-bash-tool", nil, `{"tool_name":"Read","tool_input":{"file_path":"x"}}`},
		{"safe-bash-command", nil, `{"tool_name":"Bash","tool_input":{"command":"ls -la"}}`},
		{"enforce-deny-mutating", []string{"DEVRITES_REVIEWER_RO=enforce"}, `{"tool_name":"Bash","tool_input":{"command":"rm -rf foo"}}`},
		{"enforce-deny-git-commit", []string{"DEVRITES_REVIEWER_RO=enforce"}, `{"tool_name":"Bash","tool_input":{"command":"git commit -m x"}}`},
		{"observe-mutating-silent", nil, `{"tool_name":"Bash","tool_input":{"command":"rm -rf foo"}}`},
		// The agent-required gate reads agent_type ONLY (like the bash node parse),
		// so a subagent_type-only payload has no identity and is allowed.
		{"agent-required-subagent-type-only-allows",
			[]string{"DEVRITES_REVIEWER_RO=enforce", "DEVRITES_REVIEWER_AGENT_REQUIRED=1"},
			`{"tool_name":"Bash","tool_input":{"command":"rm -rf foo"},"subagent_type":"devrites-code-reviewer"}`},
		{"agent-required-devrites-agent-denies",
			[]string{"DEVRITES_REVIEWER_RO=enforce", "DEVRITES_REVIEWER_AGENT_REQUIRED=1"},
			`{"tool_name":"Bash","tool_input":{"command":"rm -rf foo"},"agent_type":"devrites-code-reviewer"}`},
		{"agent-required-non-devrites-allows",
			[]string{"DEVRITES_REVIEWER_RO=enforce", "DEVRITES_REVIEWER_AGENT_REQUIRED=1"},
			`{"tool_name":"Bash","tool_input":{"command":"rm -rf foo"},"agent_type":"Explore"}`},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			c := parityCase{
				workdir: dir,
				env:     append([]string{"DEVRITES_ROOT="}, tc.env...),
				stdin:   tc.stdin,
				goArgs:  []string{"hook", "reviewer-readonly", "--harness=claude"},
			}
			c.assertEqual(t)
		})
	}
}

// TestParitySubagentOrient checks `hook subagent-orient` against golden snapshots
// for each payload: a non-devrites agent (silent) and a devrites-* agent (the
// fixed context envelope).
func TestParitySubagentOrient(t *testing.T) {
	cases := []struct {
		name  string
		stdin string
	}{
		{"garbage-input", "not json"},
		{"non-devrites-agent", `{"agent_type":"Explore"}`},
		{"devrites-agent", `{"agent_type":"devrites-code-reviewer"}`},
		{"devrites-agent-subagent-type-key", `{"subagent_type":"devrites-security-auditor"}`},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			c := parityCase{
				workdir: dir,
				env:     []string{"DEVRITES_ROOT="},
				stdin:   tc.stdin,
				goArgs:  []string{"hook", "subagent-orient", "--harness=claude"},
			}
			c.assertEqual(t)
		})
	}
}

// TestParityCursor checks `hook cursor` against golden snapshots for each
// active-feature state: the field reads, the open-questions count, the AFK line,
// the RED warning, and the silent no-active path.
func TestParityCursor(t *testing.T) {
	work := t.TempDir()

	writeFeatureFile(t, work, "full", "state.md",
		"- Phase: build\n- Status: running\n- Next step: implement slice 3\n- AFK slices remaining: 2\n")
	writeFeatureFile(t, work, "full", "questions.md",
		"## q-1\nstatus: open\ngate: blocking\n\n## q-2\nstatus: open\ngate: advisory\n")
	// present questions.md with ZERO open → the grep -c "0\n0" quirk.
	writeFeatureFile(t, work, "zerogates", "state.md", "- Phase: spec\n- Status: running\n")
	writeFeatureFile(t, work, "zerogates", "questions.md", "## q-1\nstatus: resolved\ngate: blocking\n")
	// a RED workspace.
	writeFeatureFile(t, work, "red", "state.md", "- Phase: prove\n- Status: running\n")
	writeFeatureFile(t, work, "red", ".red", "npm test\n")
	// a bare dir (no state.md / questions.md).
	makeFeatureDir(t, work, "bare")

	for _, tc := range []struct {
		name, active string
		afk          bool
	}{
		{"full-hitl", "full", false},
		{"full-afk", "full", true},
		{"zerogates", "zerogates", false},
		{"red", "red", false},
		{"bare", "bare", false},
		{"no-active", "", false},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			setActiveAFK(t, work, tc.active, tc.afk)
			(parityCase{
				workdir: work,
				env:     libRootEnv(work),
				goArgs:  []string{"hook", "cursor", "--harness=claude"},
			}).assertEqual(t)
		})
	}
}

// TestParityStatusline checks `hook statusline` against golden snapshots for each
// active-feature state.
func TestParityStatusline(t *testing.T) {
	work := t.TempDir()

	writeFeatureFile(t, work, "full", "state.md", "- Phase: build\n- Status: running\n")
	writeFeatureFile(t, work, "full", "questions.md",
		"## q-1\nstatus: open\ngate: blocking\n\n## q-2\nstatus: open\ngate: advisory\n")
	writeFeatureFile(t, work, "nophase", "state.md", "- Status: running\n") // phase → "?"
	writeFeatureFile(t, work, "red", "state.md", "- Phase: prove\n")
	writeFeatureFile(t, work, "red", ".red", "npm test\n")

	for _, tc := range []struct {
		name, active string
		afk          bool
	}{
		{"full-hitl", "full", false},
		{"full-afk", "full", true},
		{"nophase", "nophase", false},
		{"red", "red", false},
		{"no-active", "", false},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			setActiveAFK(t, work, tc.active, tc.afk)
			(parityCase{
				workdir: work,
				env:     libRootEnv(work),
				stdin:   `{"session":"x"}`, // statusline drains stdin
				goArgs:  []string{"hook", "statusline", "--harness=claude"},
			}).assertEqual(t)
		})
	}
}

// TestParityRedwatch checks `hook redwatch` against golden snapshots for each
// payload: the test-command gate, the RED additionalContext, the green clear, and
// the silent non-test / non-Bash paths. Each case uses its own slug so the .red
// side effects don't cross-contaminate.
func TestParityRedwatch(t *testing.T) {
	work := t.TempDir()
	for _, slug := range []string{"redf", "redp", "nont", "nonb"} {
		makeFeatureDir(t, work, slug)
	}

	for _, tc := range []struct {
		name, active, stdin string
	}{
		{"fail-writes-red", "redf", `{"tool_name":"Bash","tool_input":{"command":"npm test"},"tool_response":"Tests: 1 failed, 3 passed"}`},
		{"pass-clears-red", "redp", `{"tool_name":"Bash","tool_input":{"command":"go test ./..."},"tool_response":"ok  all tests passed"}`},
		{"non-test-command", "nont", `{"tool_name":"Bash","tool_input":{"command":"ls -la"},"tool_response":"file"}`},
		{"non-bash-tool", "nonb", `{"tool_name":"Read","tool_input":{"file_path":"x"},"tool_response":"y"}`},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			setActiveAFK(t, work, tc.active, false)
			(parityCase{
				workdir: work,
				env:     libRootEnv(work),
				stdin:   tc.stdin,
				goArgs:  []string{"hook", "redwatch", "--harness=claude"},
			}).assertEqual(t)
		})
	}
}

// TestParityA1Guard checks `hook a1-guard` against golden snapshots for each
// payload: it blocks a MAIN-thread source edit only when the build window is open
// and enforce is on, and allows subagent edits, .devrites bookkeeping edits, an
// inline-fallback window, and a closed window.
func TestParityA1Guard(t *testing.T) {
	work := t.TempDir()
	makeFeatureDir(t, work, "win")
	writeFeatureFile(t, work, "win", ".reconcile-base", "snapshot\n")
	makeFeatureDir(t, work, "inline")
	writeFeatureFile(t, work, "inline", ".reconcile-base", "snapshot\n")
	writeFeatureFile(t, work, "inline", ".reconcile-inline", "")
	makeFeatureDir(t, work, "nowin") // no .reconcile-base → window closed

	enforce := []string{"DEVRITES_A1_HOOK=enforce"}
	for _, tc := range []struct {
		name, active string
		env          []string
		stdin        string
	}{
		{"source-edit-enforce-denies", "win", enforce, `{"tool_name":"Edit","tool_input":{"file_path":"src/app.js"}}`},
		{"subagent-allowed", "win", enforce, `{"tool_name":"Edit","tool_input":{"file_path":"src/app.js"},"agent_id":"abc"}`},
		{"devrites-edit-allowed", "win", enforce, `{"tool_name":"Edit","tool_input":{"file_path":".devrites/features/win/state.md"}}`},
		{"non-edit-tool", "win", enforce, `{"tool_name":"Bash","tool_input":{"command":"ls"}}`},
		{"observe-mode-silent", "win", nil, `{"tool_name":"Edit","tool_input":{"file_path":"src/app.js"}}`},
		{"inline-fallback-silent", "inline", enforce, `{"tool_name":"Edit","tool_input":{"file_path":"src/app.js"}}`},
		{"no-window-silent", "nowin", enforce, `{"tool_name":"Edit","tool_input":{"file_path":"src/app.js"}}`},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			setActiveAFK(t, work, tc.active, false)
			(parityCase{
				workdir: work,
				env:     append(libRootEnv(work), tc.env...),
				stdin:   tc.stdin,
				goArgs:  []string{"hook", "a1-guard", "--harness=claude"},
			}).assertEqual(t)
		})
	}
}

// TestParityWrightScope checks `hook wright-scope` against golden snapshots for
// each payload: a path in touched-files.md is allowed, one outside it is denied
// under enforce, .devrites edits and non-edit tools are allowed, and the
// agent-required gate skips a non-wright caller.
func TestParityWrightScope(t *testing.T) {
	work := t.TempDir()
	writeFeatureFile(t, work, "ws", "touched-files.md", "# touched\n- `src/app.js`\n")

	enforce := []string{"DEVRITES_WRIGHT_SCOPE=enforce"}
	for _, tc := range []struct {
		name  string
		env   []string
		stdin string
	}{
		{"in-scope-allowed", enforce, `{"tool_name":"Edit","tool_input":{"file_path":"src/app.js"}}`},
		{"out-of-scope-enforce-denies", enforce, `{"tool_name":"Edit","tool_input":{"file_path":"src/other.js"}}`},
		{"devrites-edit-allowed", enforce, `{"tool_name":"Edit","tool_input":{"file_path":".devrites/x/y.md"}}`},
		{"non-edit-tool", enforce, `{"tool_name":"Bash","tool_input":{"command":"ls"}}`},
		{"out-of-scope-observe-silent", nil, `{"tool_name":"Edit","tool_input":{"file_path":"src/other.js"}}`},
		{"agent-required-non-wright", append([]string{"DEVRITES_WRIGHT_AGENT_REQUIRED=1"}, enforce...),
			`{"tool_name":"Edit","tool_input":{"file_path":"src/other.js"},"agent_type":"Explore"}`},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			setActiveAFK(t, work, "ws", false)
			(parityCase{
				workdir: work,
				env:     append(libRootEnv(work), tc.env...),
				stdin:   tc.stdin,
				goArgs:  []string{"hook", "wright-scope", "--harness=claude"},
			}).assertEqual(t)
		})
	}
}

// setActiveAFK points .devrites/ACTIVE at slug (or removes it) and toggles the
// .devrites/AFK sentinel — the two files bash (via CWD) and the migrated Go (via
// <root>) both read from the same shared location.
func setActiveAFK(t *testing.T, work, slug string, afk bool) {
	t.Helper()
	active := filepath.Join(work, ".devrites", "ACTIVE")
	if slug == "" {
		_ = os.Remove(active)
	} else if err := os.WriteFile(active, []byte(slug+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	afkPath := filepath.Join(work, ".devrites", "AFK")
	if afk {
		if err := os.WriteFile(afkPath, []byte("max-slices: 2\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	} else {
		_ = os.Remove(afkPath)
	}
}

// TestParityOrientSilentPath checks `hook orient` against a golden snapshot for
// the common session-start case: outside any DevRites workspace (a directory with
// no .devrites), orient must stay silent — exit 0, no stdout.
func TestParityOrientSilentPath(t *testing.T) {
	dir := t.TempDir() // a directory with no .devrites

	c := parityCase{
		workdir: dir,
		// Ensure DEVRITES_ROOT does not leak in and make the Go side resolve a
		// real workspace; it must see "not a DevRites project".
		env:    []string{"DEVRITES_ROOT="},
		goArgs: []string{"hook", "orient", "--harness=claude"},
	}
	c.assertEqual(t)
}
