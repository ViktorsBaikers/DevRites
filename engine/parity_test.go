package main_test

// The parity oracle: a reusable harness that runs a shared fixture through BOTH
// a bash implementation and the Go command and diffs their stdout + exit code.
// It is the transitional proof that a Go port matches the script it replaces;
// later command ports (issues 08/09) reuse parityCase unchanged, swapping in
// their own bash script and go args.
//
// stderr is intentionally NOT compared: it is diagnostic, not contract. Only
// stdout (the hook/command output the harness consumes) and the exit code are.

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// parityCase describes one shared-fixture comparison.
type parityCase struct {
	workdir string   // cwd both commands run in
	env     []string // extra environment for both commands
	stdin   string   // payload piped to both commands' stdin (hooks read stdin)
	bash    []string // full argv of the bash implementation (e.g. bash, script.sh)
	goArgs  []string // args to the built devrites binary (binPath is prepended)
}

// parityResult is the raw outcome of running both sides.
type parityResult struct {
	bashOut  string
	bashCode int
	goOut    string
	goCode   int
}

func (r parityResult) equal() bool {
	return r.bashOut == r.goOut && r.bashCode == r.goCode
}

// run executes both implementations and returns their outputs and exit codes.
func (c parityCase) run(t *testing.T) parityResult {
	t.Helper()
	bo, bc := runArgv(t, c.workdir, c.env, c.stdin, c.bash[0], c.bash[1:]...)
	go_, gc := runArgv(t, c.workdir, c.env, c.stdin, binPath, c.goArgs...)
	return parityResult{bashOut: bo, bashCode: bc, goOut: go_, goCode: gc}
}

// assertEqual runs the case and fails the test unless bash and Go agree on
// stdout and exit code. This is what a real port test calls.
func (c parityCase) assertEqual(t *testing.T) {
	t.Helper()
	r := c.run(t)
	if !r.equal() {
		t.Errorf("parity mismatch\n bash: exit=%d out=%q\n   go: exit=%d out=%q",
			r.bashCode, r.bashOut, r.goCode, r.goOut)
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

// repoRoot is the DevRites repo root (the engine's parent), where the legacy
// pack scripts live.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func requireBash(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available; skipping parity oracle")
	}
}

// requireNode skips a parity case whose bash side parses its payload with node
// (the shipped hooks do). Without node the bash hook fails open and stays silent,
// so a parity assertion against a decision path would be meaningless.
func requireNode(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available; skipping node-dependent parity case")
	}
}

// legacyHook returns the path to a shipped bash hook, skipping if it is absent
// (e.g. after the cutover that deletes the scripts).
func legacyHook(t *testing.T, name string) string {
	t.Helper()
	script := filepath.Join(repoRoot(t), "pack", ".claude", "hooks", name)
	if _, err := os.Stat(script); err != nil {
		t.Skipf("legacy hook %s not present: %v", name, err)
	}
	return script
}

// TestParityReviewerReadonly proves `hook reviewer-readonly` matches
// devrites-reviewer-readonly.sh on the paths whose behavior is independent of the
// state-file layout: the silent no-ops AND the full enforce-mode deny (a fixed
// JSON decision that reads no workspace state). The observe-mode side effect
// writes to different log paths pre/post-migration, so only its stdout+exit
// (empty, 0) is asserted — the parity oracle compares stdout+exit, not files.
func TestParityReviewerReadonly(t *testing.T) {
	requireBash(t)
	script := legacyHook(t, "devrites-reviewer-readonly.sh")

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
			requireNode(t) // the bash side parses the payload with node
			dir := t.TempDir()
			c := parityCase{
				workdir: dir,
				env:     append([]string{"DEVRITES_ROOT="}, tc.env...),
				stdin:   tc.stdin,
				bash:    []string{"bash", script},
				goArgs:  []string{"hook", "reviewer-readonly", "--harness=claude"},
			}
			c.assertEqual(t)
		})
	}
}

// TestParitySubagentOrient proves `hook subagent-orient` matches
// devrites-subagent-orient.sh on both paths: a non-devrites agent (silent) and a
// devrites-* agent (the fixed context envelope, byte-for-byte). The injected text
// is embedded from the same source file the script reads, so the two cannot drift.
func TestParitySubagentOrient(t *testing.T) {
	requireBash(t)
	script := legacyHook(t, "devrites-subagent-orient.sh")

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
			requireNode(t) // the bash side parses the payload with node
			dir := t.TempDir()
			c := parityCase{
				workdir: dir,
				env:     []string{"DEVRITES_ROOT="},
				stdin:   tc.stdin,
				bash:    []string{"bash", script},
				goArgs:  []string{"hook", "subagent-orient", "--harness=claude"},
			}
			c.assertEqual(t)
		})
	}
}

// TestParityOracleSelfCheck proves the harness actually detects agreement and
// disagreement — otherwise a green parity test would be meaningless. It checks
// the comparison logic (equal()) and that runArgv captures a real command's
// stdout and exit code.
func TestParityOracleSelfCheck(t *testing.T) {
	if !(parityResult{bashOut: "x", bashCode: 0, goOut: "x", goCode: 0}).equal() {
		t.Error("equal() must report identical stdout+exit as equal")
	}
	if (parityResult{bashOut: "x", bashCode: 0, goOut: "y", goCode: 0}).equal() {
		t.Error("equal() must report differing stdout as unequal")
	}
	if (parityResult{bashOut: "x", bashCode: 0, goOut: "x", goCode: 1}).equal() {
		t.Error("equal() must report differing exit code as unequal")
	}

	requireBash(t)
	out, code := runArgv(t, t.TempDir(), nil, "", "bash", "-c", `printf 'hi\n'; exit 3`)
	if out != "hi\n" || code != 3 {
		t.Errorf("runArgv captured out=%q code=%d, want %q and 3", out, code, "hi\n")
	}
}

// TestParityOrientSilentPath is a REAL parity assertion against the legacy
// devrites-orient.sh: outside any DevRites workspace, both the bash hook and the
// Go `hook orient` must be byte-identical — exit 0, no stdout. This is the
// common session-start case (a project with no active feature). Full byte-parity
// on the active-workspace path is deferred to cutover (issue 11), because the
// legacy script reads the pre-migration .devrites/work/ layout while the engine
// reads the new schema.
func TestParityOrientSilentPath(t *testing.T) {
	requireBash(t)
	script := filepath.Join(repoRoot(t), "pack", ".claude", "hooks", "devrites-orient.sh")
	if _, err := os.Stat(script); err != nil {
		t.Skipf("legacy orient script not present: %v", err)
	}
	dir := t.TempDir() // a directory with no .devrites

	c := parityCase{
		workdir: dir,
		// Ensure DEVRITES_ROOT does not leak in and make the Go side resolve a
		// real workspace; both sides must see "not a DevRites project".
		env:    []string{"DEVRITES_ROOT="},
		bash:   []string{"bash", script},
		goArgs: []string{"hook", "orient", "--harness=claude"},
	}
	c.assertEqual(t)
}
