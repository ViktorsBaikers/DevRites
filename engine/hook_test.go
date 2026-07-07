package main_test

// Issue 03: `hook orient` dual-harness + fail-open, and issue 04's `hook
// stop-gate`. CLI black-box: run the binary as a subprocess against a fixture
// workspace and assert stdout + exit. The parity oracle lives in parity_test.go.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeActive points the workspace's ACTIVE pointer at slug.
func writeActive(t *testing.T, root, slug string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte(slug+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// parseAdditionalContext decodes a SessionStart hook envelope and returns the
// injected additionalContext, failing the test if the shape is wrong.
func parseAdditionalContext(t *testing.T, stdout string) string {
	t.Helper()
	var env struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &env); err != nil {
		t.Fatalf("stdout is not a valid hook envelope: %v\n%s", err, stdout)
	}
	if env.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("hookEventName = %q, want SessionStart", env.HookSpecificOutput.HookEventName)
	}
	return env.HookSpecificOutput.AdditionalContext
}

func TestHookOrientActiveFeatureEmitsDigest(t *testing.T) {
	for _, h := range []string{"claude", "codex"} {
		h := h
		t.Run(h, func(t *testing.T) {
			root := newWorkspace(t)
			writeActive(t, root, "auth-tokens")
			out, errOut, code := runDevrites(t, root, "hook", "orient", "--harness="+h)
			if code != 0 {
				t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
			}
			ctx := parseAdditionalContext(t, out)
			for _, want := range []string{
				`feature "auth-tokens"`,
				"phase: build",
				"result: incomplete (missing: tasks)",
				"/rite-* command",
			} {
				if !strings.Contains(ctx, want) {
					t.Errorf("additionalContext missing %q\n--- got ---\n%s", want, ctx)
				}
			}
		})
	}
}

func TestHookOrientNoActiveFeatureIsSilent(t *testing.T) {
	root := newWorkspace(t) // fixture has no ACTIVE pointer
	out, errOut, code := runDevrites(t, root, "hook", "orient", "--harness=claude")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("stdout = %q, want silent when no feature is active", out)
	}
}

func TestHookOrientStaleActivePointerIsSilent(t *testing.T) {
	root := newWorkspace(t)
	writeActive(t, root, "does-not-exist")
	out, _, code := runDevrites(t, root, "hook", "orient", "--harness=claude")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (a stale pointer must not wedge the session)", code)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("stdout = %q, want silent for a stale ACTIVE pointer", out)
	}
}

func TestHookOrientNonDevRitesDirIsSilent(t *testing.T) {
	// Point at a directory with no .devrites — fail-open: silent, exit 0.
	empty := t.TempDir()
	out, _, code := runDevrites(t, empty, "hook", "orient", "--harness=claude")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 outside a DevRites workspace", code)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("stdout = %q, want silent outside a workspace", out)
	}
}

func TestHookUnknownHarnessExitsUsage(t *testing.T) {
	root := newWorkspace(t)
	_, errOut, code := runDevrites(t, root, "hook", "orient", "--harness=bogus")
	if code != 2 {
		t.Fatalf("exit = %d, want 2 for an unknown harness", code)
	}
	if !strings.Contains(errOut, "unknown harness") {
		t.Errorf("stderr = %q, want it to name the unknown harness", errOut)
	}
}

func TestHookMissingHarnessExitsUsage(t *testing.T) {
	root := newWorkspace(t)
	if _, _, code := runDevrites(t, root, "hook", "orient"); code != 2 {
		t.Fatalf("exit = %d, want 2 when --harness is absent", code)
	}
}
