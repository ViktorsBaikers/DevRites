package main_test

// Issue 04: completeness gates (readiness, seal) and the stop-gate rest-point
// invariant. CLI black-box against fixture workspaces.

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runDevritesIO is runDevrites with control over stdin and extra environment,
// for the hooks that read a payload (stop-gate) or a mode env var.
func runDevritesIO(t *testing.T, root, stdin string, extraEnv []string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(append(os.Environ(), "DEVRITES_ROOT="+root), extraEnv...)
	cmd.Stdin = strings.NewReader(stdin)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("running %v: %v", args, err)
		}
		code = ee.ExitCode()
	}
	return outBuf.String(), errBuf.String(), code
}

// setPhase rewrites the phase in a feature's feature.md frontmatter.
func setPhase(t *testing.T, root, slug, phase string) {
	t.Helper()
	path := filepath.Join(root, "features", slug, "feature.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\n")
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "phase:") {
			lines[i] = "phase: " + phase
		}
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadinessBlocksIncompletePhase(t *testing.T) {
	root := newWorkspace(t)
	out, _, code := runDevrites(t, root, "readiness", "auth-tokens") // build phase, tasks empty
	if code != 3 {
		t.Fatalf("exit = %d, want 3 (blocked)", code)
	}
	if !strings.Contains(out, `result: blocked (missing to leave "build": tasks)`) {
		t.Errorf("stdout missing the block line\n%s", out)
	}
	if !strings.Contains(out, "devrites readiness auth-tokens") {
		t.Errorf("stdout missing the actionable retry step\n%s", out)
	}
}

func TestReadinessPassesCompletePhase(t *testing.T) {
	root := newWorkspace(t)
	out, _, code := runDevrites(t, root, "readiness", "search-ranking") // spec phase, spec present
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (pass)", code)
	}
	if !strings.Contains(out, "result: pass") {
		t.Errorf("stdout not a pass\n%s", out)
	}
	// A not-yet-required section (proof, empty at the spec phase) must not appear
	// as a blocker.
	if strings.Contains(out, "blocked") {
		t.Errorf("an empty not-yet-required section blocked readiness\n%s", out)
	}
}

func TestSealBlocksWhenSealSectionsMissing(t *testing.T) {
	root := newWorkspace(t)
	out, _, code := runDevrites(t, root, "seal", "auth-tokens")
	if code != 3 {
		t.Fatalf("exit = %d, want 3 (blocked)", code)
	}
	// auth-tokens: tasks + proof empty; seal requires the full set.
	if !strings.Contains(out, "result: blocked (missing to seal: tasks, proof)") {
		t.Errorf("unexpected seal block line\n%s", out)
	}
}

func TestSealPassesWhenComplete(t *testing.T) {
	root := newWorkspace(t)
	// Fill the two empty sections auth-tokens is missing for a seal. tasks.md is
	// a heading-only stub in the fixture; proof.md is absent (empty), so write it.
	appendFile(t, filepath.Join(root, "features", "auth-tokens", "tasks.md"), "\n- [x] mint\n")
	if err := os.WriteFile(filepath.Join(root, "features", "auth-tokens", "proof.md"),
		[]byte("# Proof\n\nAll acceptance tests pass.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code := runDevrites(t, root, "seal", "auth-tokens")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (pass)\n%s", code, out)
	}
	if !strings.Contains(out, "result: pass") {
		t.Errorf("stdout not a pass\n%s", out)
	}
}

func TestStopGateEnforceBlocksClaimedDoneButUnproven(t *testing.T) {
	root := newWorkspace(t)
	setPhase(t, root, "auth-tokens", "seal") // claims completion
	writeActive(t, root, "auth-tokens")      // proof.md is empty
	out, _, code := runDevritesIO(t, root, "{}", []string{"DEVRITES_STOP_GATE=enforce"}, "hook", "stop-gate", "--harness=claude")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (a block is a decision, not a crash)", code)
	}
	if !strings.Contains(out, `"decision":"block"`) {
		t.Errorf("stop-gate did not block a claimed-done-but-unproven rest point\n%s", out)
	}
	if !strings.Contains(out, "proof.md is empty") {
		t.Errorf("block reason not actionable\n%s", out)
	}
}

func TestStopGateDoesNotBlockNormalInProgress(t *testing.T) {
	root := newWorkspace(t)
	writeActive(t, root, "auth-tokens") // phase build — in progress, not claiming done
	out, _, code := runDevritesIO(t, root, "{}", []string{"DEVRITES_STOP_GATE=enforce"}, "hook", "stop-gate", "--harness=claude")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("stop-gate blocked normal in-progress work\n%s", out)
	}
}

func TestStopGateObserveModeNeverBlocks(t *testing.T) {
	root := newWorkspace(t)
	setPhase(t, root, "auth-tokens", "seal")
	writeActive(t, root, "auth-tokens")
	// Default mode (observe) — even a violated invariant must not emit a block.
	out, _, code := runDevritesIO(t, root, "{}", nil, "hook", "stop-gate", "--harness=claude")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Errorf("observe mode emitted a block: exit=%d out=%q", code, out)
	}
}

func TestStopGateLoopGuardLetsStop(t *testing.T) {
	root := newWorkspace(t)
	setPhase(t, root, "auth-tokens", "seal")
	writeActive(t, root, "auth-tokens")
	// stop_hook_active=true: we already blocked once this cycle; must let it stop.
	out, _, code := runDevritesIO(t, root, `{"stop_hook_active":true}`,
		[]string{"DEVRITES_STOP_GATE=enforce"}, "hook", "stop-gate", "--harness=claude")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Errorf("loop guard failed: exit=%d out=%q", code, out)
	}
}
