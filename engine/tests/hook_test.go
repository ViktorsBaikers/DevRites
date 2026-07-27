package main_test

// CLI coverage for Issue 03's dual-harness `hook orient` behavior and Issue 04's
// `hook stop-gate`. These tests run the binary against a fixture workspace and
// check stdout and exit status. parity_test.go owns the parity oracle.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
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
				"host-specific DevRites command",
			} {
				if !strings.Contains(ctx, want) {
					t.Errorf("additionalContext missing %q\n--- got ---\n%s", want, ctx)
				}
			}
		})
	}
}

func TestHookOrientNoActiveFeatureNudgesOnceThenSilent(t *testing.T) {
	root := newWorkspace(t) // fixture has no ACTIVE pointer
	out, errOut, code := runDevrites(t, root, "hook", "orient", "--harness=claude")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	ctx := parseAdditionalContext(t, out)
	if !strings.Contains(ctx, "no feature is active") || !strings.Contains(ctx, "/rite") {
		t.Errorf("first blank-slate session should nudge a starting rite, got %q", ctx)
	}
	if _, err := os.Stat(filepath.Join(root, ".first-run-shown")); err != nil {
		t.Fatalf("first-run marker not written: %v", err)
	}
	out, errOut, code = runDevrites(t, root, "hook", "orient", "--harness=claude")
	if code != 0 {
		t.Fatalf("second run exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("second run stdout = %q, want silent once the marker exists", out)
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
	// Point at a directory with no .devrites: fail-open: silent, exit 0.
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

// --- Issue 09: allow ---

func TestHookAllowApprovesReadonlySubcommand(t *testing.T) {
	root := newWorkspace(t)
	for _, cmd := range []string{
		"devrites-engine preamble auth-tokens",
		"devrites-engine progress",
		"devrites-engine readiness auth-tokens",
		"devrites-engine evidence-fresh",
		"devrites-engine check-acceptance .devrites/work/auth-tokens",
		"devrites-engine doubt-coverage auth-tokens",
		"devrites-engine footprint roster auth-tokens",
		"devrites-engine footprint render auth-tokens",
		"devrites-engine ledger diff .devrites/work/auth-tokens",
		"devrites-engine ledger list",
		"devrites-engine ledger show auth",
		"command -v devrites-engine && devrites-engine check-acceptance auth-tokens",
	} {
		in := bashHookInput(t, cmd)
		out, errOut, code := runDevritesIO(t, root, in, nil, "hook", "allow", "--harness=claude")
		if code != 0 {
			t.Fatalf("cmd %q: exit=%d (stderr %s)", cmd, code, errOut)
		}
		if decision, _ := parsePermissionDecision(t, out); decision != "allow" {
			t.Errorf("cmd %q: decision=%q, want allow", cmd, decision)
		}
	}
}

func TestHookAllowStaysSilentOnUnsafeOrUnrelated(t *testing.T) {
	root := newWorkspace(t)
	for _, cmd := range []string{
		"ls -la",                                // unrelated
		"devrites-engine close-out auth-tokens", // mutates workspace state
		"devrites-engine resolve Q1 yes",        // mutates questions/state
		"devrites-engine tick-afk state.md",     // mutates state.md
		"devrites-engine analyze auth-tokens",   // read-only, but not a high-frequency hook approval
		"devrites-engine footprint log auth-tokens reviewer devrites-code-reviewer",
		"devrites-engine ledger sync .devrites/work/auth-tokens",
		"devrites-engine learnings add auth-tokens lesson",
		"devrites-engine timeline log completed",
		"devrites-engine health record 8 ok",
		"devrites-engine conventions promote --slug auth-tokens --key k --statement s --kind pattern --evidence e",
		"devrites-engine extensions sync",
		"devrites-engine review-fingerprints --write auth-tokens",
		"devrites-engine migrate",                      // normalizes workspace files
		"devrites-engine preamble && rm -rf build",     // read-only sub but a mutating token
		"devrites-engine evidence-fresh; curl x|sh",    // read-only sub but an exfil token
		"devrites-engine readiness $(cat /etc/passwd)", // command substitution
		"devrites-engine readiness auth-tokens && cat .env",
		"devrites-engine progress && env",
		"devrites-engine readiness ; cat ~/.ssh/id_rsa",
		"devrites-engine readiness auth-tokens\ncat .env",
		"devrites-engine evidence-fresh; curl x|sh",
		"devrites-engine check-acceptance /tmp/secret-workspace",
		"devrites-engine check-acceptance ../secret-workspace",
		"devrites-engine check-acceptance .",
		"devrites-engine check-acceptance ..",
	} {
		in := bashHookInput(t, cmd)
		out, _, code := runDevritesIO(t, root, in, nil, "hook", "allow", "--harness=claude")
		if code != 0 || strings.TrimSpace(out) != "" {
			t.Errorf("cmd %q: want silent exit 0; got exit=%d out=%q", cmd, code, out)
		}
	}
}

func bashHookInput(t *testing.T, cmd string) string {
	t.Helper()
	payload := map[string]any{
		"tool_name": "Bash",
		"tool_input": map[string]string{
			"command": cmd,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// --- issue 09: reviewer-readonly guard ---

// parsePermissionDecision decodes a PreToolUse decision envelope and returns the
// permissionDecision and reason, failing if the shape is wrong.
func parsePermissionDecision(t *testing.T, stdout string) (decision, reason string) {
	t.Helper()
	var env struct {
		HookSpecificOutput struct {
			HookEventName            string `json:"hookEventName"`
			PermissionDecision       string `json:"permissionDecision"`
			PermissionDecisionReason string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &env); err != nil {
		t.Fatalf("stdout is not a valid PreToolUse decision: %v\n%s", err, stdout)
	}
	if env.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Errorf("hookEventName = %q, want PreToolUse", env.HookSpecificOutput.HookEventName)
	}
	return env.HookSpecificOutput.PermissionDecision, env.HookSpecificOutput.PermissionDecisionReason
}

func TestHookReviewerReadonlyEnforceDeniesMutatingBash(t *testing.T) {
	for _, h := range []string{"claude", "codex"} {
		h := h
		t.Run(h, func(t *testing.T) {
			root := newWorkspace(t)
			in := `{"tool_name":"Bash","tool_input":{"command":"rm -rf build"}}`
			out, errOut, code := runDevritesIO(t, root, in, []string{"DEVRITES_REVIEWER_RO=enforce"},
				"hook", "reviewer-readonly", "--harness="+h)
			if code != 0 {
				t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
			}
			decision, reason := parsePermissionDecision(t, out)
			if decision != "deny" {
				t.Errorf("permissionDecision = %q, want deny", decision)
			}
			if !strings.Contains(reason, "reviewers are read-only") {
				t.Errorf("reason %q missing the read-only explanation", reason)
			}
		})
	}
}

func TestHookReviewerReadonlyAllowsSafeBash(t *testing.T) {
	root := newWorkspace(t)
	in := `{"tool_name":"Bash","tool_input":{"command":"grep -rn foo ."}}`
	out, _, code := runDevritesIO(t, root, in, []string{"DEVRITES_REVIEWER_RO=enforce"},
		"hook", "reviewer-readonly", "--harness=claude")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("stdout = %q, want silent for a read-only command", out)
	}
}

func TestHookReviewerReadonlyNonBashIsSilent(t *testing.T) {
	root := newWorkspace(t)
	in := `{"tool_name":"Read","tool_input":{"file_path":"secret.txt"}}`
	out, _, code := runDevritesIO(t, root, in, []string{"DEVRITES_REVIEWER_RO=enforce"},
		"hook", "reviewer-readonly", "--harness=claude")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Errorf("want silent exit 0 for a non-Bash tool; got exit=%d out=%q", code, out)
	}
}

// In observe mode (the default) a mutating command is allowed but recorded, so the
// invariant is visible without gating in-progress work.
func TestHookReviewerReadonlyObserveLogsWouldBlock(t *testing.T) {
	root := newWorkspace(t)
	writeActive(t, root, "auth-tokens")
	in := `{"tool_name":"Bash","tool_input":{"command":"git push origin main"}}`
	out, _, code := runDevritesIO(t, root, in, nil, // no enforce env → observe
		"hook", "reviewer-readonly", "--harness=claude")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("stdout = %q, want silent in observe mode", out)
	}
	logPath := filepath.Join(root, "features", "auth-tokens", ".reviewer-ro.log")
	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("observe mode did not write the would-block log: %v", err)
	}
	if !strings.Contains(string(raw), "WOULD-BLOCK") || !strings.Contains(string(raw), "git push") {
		t.Errorf("log = %q, want a WOULD-BLOCK record naming the command", raw)
	}
}

// The bash hook's `read -r tool cmd agent_type` truncates the command at its
// first newline, so it only ever scans line 1: a latent bug. The Go port scans
// the whole command, so a mutating LATER line is denied. This asserts the
// deliberate hardening (documented on reviewerMutateRe); it is a Go-only test, not
// a parity case, because it is where the two are meant to diverge.
func TestHookReviewerReadonlyScansWholeMultilineCommand(t *testing.T) {
	root := newWorkspace(t)
	in := "{\"tool_name\":\"Bash\",\"tool_input\":{\"command\":\"cat notes.txt\\nsed -i s/a/b/ f\"}}"
	out, errOut, code := runDevritesIO(t, root, in, []string{"DEVRITES_REVIEWER_RO=enforce"},
		"hook", "reviewer-readonly", "--harness=claude")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
		t.Errorf("a multi-line command whose 2nd line mutates was not denied; decision = %q", decision)
	}
}

// The agent-required gate reads only agent_type, matching the Bash hook's Node
// parser. A payload that carries the subagent name under subagent_type but not
// agent_type has no identity and is allowed, avoiding a false deny from aliases.
func TestHookReviewerReadonlyAgentRequiredIgnoresSubagentTypeAlias(t *testing.T) {
	root := newWorkspace(t)
	in := `{"tool_name":"Bash","tool_input":{"command":"rm -rf x"},"subagent_type":"devrites-code-reviewer"}`
	out, _, code := runDevritesIO(t, root, in,
		[]string{"DEVRITES_REVIEWER_RO=enforce", "DEVRITES_REVIEWER_AGENT_REQUIRED=1"},
		"hook", "reviewer-readonly", "--harness=claude")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Errorf("want silent allow (no agent_type identity); got exit=%d out=%q", code, out)
	}
}

func TestHookReviewerReadonlyAgentRequiredSkipsNonDevrites(t *testing.T) {
	root := newWorkspace(t)
	// A mutating command from a non-devrites agent is not the reviewer hook's
	// concern when agent identity is required.
	in := `{"tool_name":"Bash","tool_input":{"command":"rm -rf x"},"agent_type":"Explore"}`
	out, _, code := runDevritesIO(t, root, in,
		[]string{"DEVRITES_REVIEWER_RO=enforce", "DEVRITES_REVIEWER_AGENT_REQUIRED=1"},
		"hook", "reviewer-readonly", "--harness=claude")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Errorf("want silent exit 0 for a non-devrites agent; got exit=%d out=%q", code, out)
	}
}

func TestHookReviewerReadonlyBindsCodexExplorerCompatibility(t *testing.T) {
	root := newWorkspace(t)
	in := `{"tool_name":"Edit","tool_input":{"file_path":"src/app.go"},"agent_type":"explorer"}`
	out, errOut, code := runDevritesIO(t, root, in,
		[]string{"DEVRITES_CODEX_GENERIC_AGENT_COMPAT=1"},
		"hook", "reviewer-readonly", "--harness=codex")
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, errOut)
	}
	if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
		t.Fatalf("generic explorer mutation was not denied: %q", out)
	}
}

func TestCodexGenericCompatibilityDoesNotShadowRoot(t *testing.T) {
	root := newWorkspace(t)
	in := `{"tool_name":"Edit","tool_input":{"file_path":"src/app.go"}}`
	env := []string{
		"DEVRITES_CODEX_GENERIC_AGENT_COMPAT=1",
		"DEVRITES_HOOK_PROFILE=strict",
	}
	for _, guard := range []string{"reviewer-readonly", "wright-scope"} {
		out, errOut, code := runDevritesIO(t, root, in, env,
			"hook", guard, "--harness=codex")
		if code != 0 || strings.TrimSpace(out) != "" {
			t.Fatalf("%s shadowed root: exit=%d out=%q stderr=%q", guard, code, out, errOut)
		}
	}
}

func TestHookReviewerReadonlyActiveLeafDeniesEveryMutationSurface(t *testing.T) {
	root := newWorkspace(t)
	env := []string{
		"DEVRITES_AGENT_RUN=1",
		"DEVRITES_ACTIVE_AGENT=devrites-code-reviewer",
	}
	for _, input := range []string{
		`{"tool_name":"Edit","tool_input":{"file_path":"src/app.go"}}`,
		`{"tool_name":"apply_patch","tool_input":{"command":"*** Begin Patch\n*** Add File: src/app.go\n+x\n*** End Patch"}}`,
		`{"tool_name":"Bash","tool_input":{"command":"printf x > src/app.go"}}`,
		`{"tool_name":"spawn_agent","tool_input":{"message":"do work"}}`,
		`{"tool_name":"exec","tool_input":{"code":"writeFile()"}}`,
	} {
		out, errOut, code := runDevritesIO(t, root, input, env,
			"hook", "reviewer-readonly", "--harness=codex")
		if code != 0 {
			t.Fatalf("exit=%d stderr=%q for %s", code, errOut, input)
		}
		if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
			t.Errorf("mutation surface was not denied: %s", input)
		}
	}
}

func TestHookReviewerReadonlyActiveLeafAllowsBoundedProof(t *testing.T) {
	root := newWorkspace(t)
	in := `{"tool_name":"Bash","tool_input":{"command":"go test ./... -count=1"}}`
	out, _, code := runDevritesIO(t, root, in, []string{
		"DEVRITES_AGENT_RUN=1",
		"DEVRITES_ACTIVE_AGENT=devrites-proof-runner",
	}, "hook", "reviewer-readonly", "--harness=claude")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("bounded proof should be allowed: exit=%d out=%q", code, out)
	}
	out, _, code = runDevritesIO(t, root, in, []string{
		"DEVRITES_AGENT_RUN=1",
		"DEVRITES_ACTIVE_AGENT=devrites-proof-runner",
	}, "hook", "wright-scope", "--harness=claude")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("wright hook must not shadow reviewer policy: exit=%d out=%q", code, out)
	}
}

func TestHookWrightScopeUsesExactOrchestratorAllowlist(t *testing.T) {
	root := newWorkspace(t)
	writeActive(t, root, "auth-tokens")
	workspace := filepath.Join(root, "features", "auth-tokens")
	if err := os.WriteFile(filepath.Join(workspace, ".wright-allowlist"), []byte("src/app.go\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := []string{
		"DEVRITES_AGENT_RUN=1",
		"DEVRITES_ACTIVE_AGENT=devrites-slice-wright",
	}
	tests := []struct {
		name  string
		input string
		deny  bool
	}{
		{"listed edit", `{"tool_name":"Edit","tool_input":{"file_path":"src/app.go"}}`, false},
		{"listed freeform patch", `{"tool_name":"apply_patch","tool_input":"*** Begin Patch\n*** Update File: src/app.go\n@@\n-old\n+new\n*** End Patch\n"}`, false},
		{"listed redirect", `{"tool_name":"Bash","tool_input":{"command":"printf x > src/app.go"}}`, false},
		{"unlisted edit", `{"tool_name":"Write","tool_input":{"file_path":"src/other.go"}}`, true},
		{"substring is not scope", `{"tool_name":"Edit","tool_input":{"file_path":"src/app.go.bak"}}`, true},
		{"workspace state forbidden", `{"tool_name":"Edit","tool_input":{"file_path":".devrites/features/auth-tokens/state.md"}}`, true},
		{"nested dispatch forbidden", `{"tool_name":"spawn_agent","tool_input":{"message":"do work"}}`, true},
		{"dependency install forbidden", `{"tool_name":"Bash","tool_input":{"command":"npm install left-pad"}}`, true},
		{"cp target directory forbidden", `{"tool_name":"Bash","tool_input":{"command":"cp --target-directory=/tmp src/app.go"}}`, true},
		{"mv target directory forbidden", `{"tool_name":"Bash","tool_input":{"command":"mv --target-directory=/tmp src/app.go"}}`, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out, errOut, code := runDevritesIO(t, root, test.input, env,
				"hook", "wright-scope", "--harness=codex")
			if code != 0 {
				t.Fatalf("exit=%d stderr=%q", code, errOut)
			}
			if test.deny {
				if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
					t.Fatalf("expected deny, got %q", out)
				}
			} else if strings.TrimSpace(out) != "" {
				t.Fatalf("expected allow, got %q", out)
			}
		})
	}
}

func TestHookWrightScopeBindsCodexWorkerDuringReconcileWindow(t *testing.T) {
	root := newWorkspace(t)
	writeActive(t, root, "auth-tokens")
	workspace := filepath.Join(root, "features", "auth-tokens")
	if err := os.WriteFile(filepath.Join(workspace, ".wright-allowlist"), []byte("src/app.go\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".reconcile-base"), []byte("snapshot\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := []string{"DEVRITES_CODEX_GENERIC_AGENT_COMPAT=1", "DEVRITES_HOOK_PROFILE=strict"}

	for _, test := range []struct {
		name  string
		input string
		deny  bool
	}{
		{"listed edit", `{"tool_name":"Edit","tool_input":{"file_path":"src/app.go"},"agent_type":"worker"}`, false},
		{"unlisted edit", `{"tool_name":"Edit","tool_input":{"file_path":"src/other.go"},"agent_type":"worker"}`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			out, errOut, code := runDevritesIO(t, root, test.input, env,
				"hook", "wright-scope", "--harness=codex")
			if code != 0 {
				t.Fatalf("exit=%d stderr=%q", code, errOut)
			}
			if test.deny {
				if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
					t.Fatalf("expected deny, got %q", out)
				}
			} else if strings.TrimSpace(out) != "" {
				t.Fatalf("expected allow, got %q", out)
			}
		})
	}

	if err := os.Remove(filepath.Join(workspace, ".reconcile-base")); err != nil {
		t.Fatal(err)
	}
	out, _, code := runDevritesIO(t, root,
		`{"tool_name":"Edit","tool_input":{"file_path":"src/other.go"},"agent_type":"worker"}`,
		env, "hook", "wright-scope", "--harness=codex")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("generic worker outside a reconcile window was shadowed: exit=%d out=%q", code, out)
	}
}

func TestHookWrightScopeAcceptsCodexV2NamedRole(t *testing.T) {
	root := newWorkspace(t)
	writeActive(t, root, "auth-tokens")
	workspace := filepath.Join(root, "features", "auth-tokens")
	if err := os.WriteFile(filepath.Join(workspace, ".wright-allowlist"), []byte("src/app.go\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{".reconcile-base", ".reconcile-allowlist", ".reconcile-devrites"} {
		if err := os.WriteFile(filepath.Join(workspace, name), []byte(name+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	env := []string{"DEVRITES_HOOK_PROFILE=strict"}
	for _, test := range []struct {
		path string
		deny bool
	}{
		{"src/app.go", false},
		{"src/other.go", true},
	} {
		payload := hookPayload(t, map[string]any{
			"hook_event_name": "PreToolUse",
			"agent_type":      "devrites-slice-wright",
			"tool_name":       "Edit",
			"tool_input":      map[string]any{"file_path": test.path},
		})
		out, stderr, code := runDevritesIO(t, root, payload, env,
			"hook", "wright-scope", "--harness=codex")
		if code != 0 {
			t.Fatalf("%s exit=%d stderr=%s", test.path, code, stderr)
		}
		if test.deny {
			if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
				t.Fatalf("%s should be denied: %s", test.path, out)
			}
		} else if strings.TrimSpace(out) != "" {
			t.Fatalf("%s should be allowed: %s", test.path, out)
		}
	}
}

func TestHookWrightScopeRequiresSpawnedCodexWorkerForSourceWrites(t *testing.T) {
	root := newWorkspace(t)
	writeActive(t, root, "auth-tokens")
	workspace := filepath.Join(root, "features", "auth-tokens")
	if err := os.WriteFile(filepath.Join(workspace, ".reconcile-base"), []byte("snapshot\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := []string{"DEVRITES_CODEX_GENERIC_AGENT_COMPAT=1"}

	runGuard := func(input string, deny bool) {
		t.Helper()
		out, errOut, code := runDevritesIO(t, root, input, env,
			"hook", "wright-scope", "--harness=codex")
		if code != 0 {
			t.Fatalf("exit=%d out=%q stderr=%q", code, out, errOut)
		}
		if deny {
			if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
				t.Fatalf("expected deny, got %q", out)
			}
		} else if strings.TrimSpace(out) != "" {
			t.Fatalf("expected allow, got %q", out)
		}
	}

	runGuard(`{"tool_name":"spawn_agent","tool_input":{"agent_type":"worker"}}`, false)
	runGuard(`{"tool_name":"spawn_agent","tool_input":{"agent_type":"explorer"}}`, false)
	runGuard(`{"tool_name":"Write","tool_input":{"file_path":".devrites/features/auth-tokens/state.md"}}`, false)
	scratch := t.TempDir()
	packet := filepath.Join(scratch, "agent-packet.yaml")
	runGuard(hookPayload(t, map[string]any{
		"tool_name":  "Write",
		"tool_input": map[string]any{"file_path": packet},
	}), false)
	runGuard(`{"tool_name":"Write","tool_input":{"file_path":"../outside.txt"}}`, true)
	runGuard(`{"tool_name":"Write","tool_input":{"file_path":"src/app.go"}}`, true)
	runGuard(`{"tool_name":"Bash","tool_input":{"command":"printf x > src/app.go"}}`, true)
	runGuard(`{"tool_name":"js","tool_input":{"code":"writeFile()"}}`, true)
	out, errOut, code := runDevritesIO(t, root,
		`{"tool_name":"Bash","tool_input":{"command":"printf x > src/app.go"}}`,
		nil, "hook", "wright-scope", "--harness=claude")
	if code != 0 {
		t.Fatalf("Claude root guard: exit=%d stderr=%q", code, errOut)
	}
	if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
		t.Fatalf("Claude root source write was not denied: %q", out)
	}

	// A legacy marker is inert: it cannot authorize root source writes.
	if err := os.WriteFile(filepath.Join(workspace, ".reconcile-inline"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	runGuard(`{"tool_name":"Bash","tool_input":{"command":"printf x > src/app.go"}}`, true)
	runGuard(`{"tool_name":"js","tool_input":{"code":"writeFile()"}}`, true)

	projectDir := filepath.Dir(root)
	if err := os.MkdirAll(filepath.Join(projectDir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(projectDir, "src", "app.go")
	if err := os.WriteFile(source, []byte("package src\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(workspace, "source-alias")
	if err := os.Symlink(source, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	runGuard(`{"tool_name":"Write","tool_input":{"file_path":".devrites/features/auth-tokens/source-alias"}}`, true)
}

func TestHookA1GuardIgnoresLegacyInlineMarker(t *testing.T) {
	root := newWorkspace(t)
	writeActive(t, root, "auth-tokens")
	workspace := filepath.Join(root, "features", "auth-tokens")
	for _, name := range []string{".reconcile-base", ".reconcile-inline"} {
		if err := os.WriteFile(filepath.Join(workspace, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	input := `{"tool_name":"Edit","tool_input":{"file_path":"src/app.go"}}`
	env := []string{"DEVRITES_A1_HOOK=enforce"}

	for _, host := range []string{"codex", "claude"} {
		t.Run(host, func(t *testing.T) {
			out, errOut, code := runDevritesIO(t, root, input, env,
				"hook", "a1-guard", "--harness="+host)
			if code != 0 {
				t.Fatalf("exit=%d stderr=%q", code, errOut)
			}
			if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
				t.Fatalf("legacy inline marker authorized root source write: %q", out)
			}
		})
	}
}

func TestHookA1GuardAllowsExternalDispatchScratchOnly(t *testing.T) {
	root := newWorkspace(t)
	writeActive(t, root, "auth-tokens")
	workspace := filepath.Join(root, "features", "auth-tokens")
	if err := os.WriteFile(filepath.Join(workspace, ".reconcile-base"), []byte("snapshot\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	scratch := t.TempDir()
	packet := filepath.Join(scratch, "agent-packet.yaml")
	env := []string{"DEVRITES_A1_HOOK=enforce"}
	patch := func(tool, path string, freeform bool) string {
		body := "*** Begin Patch\n*** Add File: " + path + "\n+x\n*** End Patch\n"
		if freeform {
			return hookPayload(t, map[string]any{"tool_name": tool, "tool_input": body})
		}
		return hookPayload(t, map[string]any{"tool_name": tool, "tool_input": map[string]any{"patch": body}})
	}
	shell := func(command string) string {
		return hookPayload(t, map[string]any{
			"tool_name":  "functions.exec_command",
			"tool_input": map[string]any{"cmd": command},
		})
	}

	runGuard := func(input string, deny bool) {
		t.Helper()
		out, errOut, code := runDevritesIO(t, root, input, env,
			"hook", "a1-guard", "--harness=codex")
		if code != 0 {
			t.Fatalf("exit=%d out=%q stderr=%q", code, out, errOut)
		}
		if deny {
			if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
				t.Fatalf("expected deny, got %q", out)
			}
		} else if strings.TrimSpace(out) != "" {
			t.Fatalf("expected allow, got %q", out)
		}
	}

	runGuard(patch("apply_patch", packet, false), false)
	runGuard(patch("functions.apply_patch", packet, true), false)
	runGuard(shell("cat <<'EOF' > "+packet+"\npacket_version: agent-packet/v1\nEOF"), false)
	runGuard(shell("sha256sum "+packet), false)
	runGuard(shell("cp "+packet+" "+filepath.Join(scratch, "agent-packet.copy.yaml")), false)
	runGuard(patch("apply_patch", "src/app.go", false), true)
	runGuard(patch("functions.apply_patch", "src/app.go", true), true)
	runGuard(shell("cat <<'EOF' > src/app.go\npackage src\nEOF"), true)
	runGuard(shell("cat <<'EOF' > "+packet+"\nunterminated"), true)
	runGuard(shell("cat <<EOF > "+packet+"\n$(cp "+packet+" src/app.go)\nEOF"), true)
	runGuard(shell("cp "+packet+" src/app.go"), true)
}

func TestHookWrightScopeFailsClosedWhenWorkspaceResolutionFails(t *testing.T) {
	root := newWorkspace(t)
	if err := os.Remove(filepath.Join(root, "ACTIVE")); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	wrightEnv := []string{
		"DEVRITES_AGENT_RUN=1",
		"DEVRITES_ACTIVE_AGENT=devrites-slice-wright",
	}
	for _, test := range []struct {
		name  string
		input string
		env   []string
		deny  bool
	}{
		{"declared wright edit", `{"tool_name":"Edit","tool_input":{"file_path":"src/app.go"}}`, wrightEnv, true},
		{"declared wright opaque execution", `{"tool_name":"exec","tool_input":{"code":"writeFile()"}}`, wrightEnv, true},
		{"declared wright nested dispatch", `{"tool_name":"spawn_agent","tool_input":{"message":"do work"}}`, wrightEnv, true},
		{"invalid declared identity", `{"tool_name":"Edit","tool_input":{"file_path":"src/app.go"}}`,
			[]string{"DEVRITES_AGENT_RUN=1", "DEVRITES_ACTIVE_AGENT=invalid"}, true},
		{"undeclared root observation", `{"tool_name":"Edit","tool_input":{"file_path":"src/app.go"}}`,
			[]string{"DEVRITES_AGENT_RUN=0", "DEVRITES_ACTIVE_AGENT=", "DEVRITES_WRIGHT_SCOPE="}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			out, errOut, code := runDevritesIO(t, root, test.input, test.env,
				"hook", "wright-scope", "--harness=codex")
			if code != 0 {
				t.Fatalf("exit=%d stderr=%q", code, errOut)
			}
			if test.deny {
				if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
					t.Fatalf("expected deny, got %q", out)
				}
			} else if strings.TrimSpace(out) != "" {
				t.Fatalf("expected non-shadowing allow, got %q", out)
			}
		})
	}
}

func TestHookWrightScopeFailsClosedOnMissingOrInvalidAllowlist(t *testing.T) {
	root := newWorkspace(t)
	writeActive(t, root, "auth-tokens")
	workspace := filepath.Join(root, "features", "auth-tokens")
	env := []string{
		"DEVRITES_AGENT_RUN=1",
		"DEVRITES_ACTIVE_AGENT=devrites-slice-wright",
	}
	in := `{"tool_name":"Edit","tool_input":{"file_path":"src/app.go"}}`

	out, _, _ := runDevritesIO(t, root, in, env, "hook", "wright-scope", "--harness=claude")
	if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
		t.Fatal("missing allowlist did not fail closed")
	}

	if err := os.WriteFile(filepath.Join(workspace, ".wright-allowlist"), []byte("src/app.go\nsrc/app.go\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, _, _ = runDevritesIO(t, root, in, env, "hook", "wright-scope", "--harness=claude")
	if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
		t.Fatal("duplicate allowlist entry did not fail closed")
	}
}

func TestHookWrightScopeBindsForgeCandidateManifestIdentity(t *testing.T) {
	repo := newForgeCLIRepo(t)
	manifest := planForgeCLI(t, repo)
	candidate, err := manifest.Candidate("A")
	if err != nil {
		t.Fatal(err)
	}
	worker := startForgeCLIWorker(t, repo, manifest.RunID, "A", "worker-a")
	baseEnv := []string{
		"DEVRITES_AGENT_RUN=1",
		"DEVRITES_ACTIVE_AGENT=devrites-slice-wright",
		"DEVRITES_FORGE_RUN_ID=" + manifest.RunID,
		"DEVRITES_FORGE_CANDIDATE=A",
		"DEVRITES_FORGE_WORKER_ID=" + worker.id,
		"DEVRITES_FORGE_WORKER_PID=" + strconv.Itoa(worker.cmd.Process.Pid),
		"DEVRITES_FORGE_PROCESS_START=" + worker.token,
	}
	input := `{"tool_name":"Edit","agent_id":"worker-a","tool_input":{"file_path":"tracked.txt"}}`
	out, stderr, code := runDevritesAt(t, repo, candidate.Worktree, input, baseEnv,
		"hook", "wright-scope", "--harness=codex")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("valid Forge candidate was denied: exit=%d stdout=%s stderr=%s", code, out, stderr)
	}
	genericEnv := append([]string{"DEVRITES_CODEX_GENERIC_AGENT_COMPAT=1"}, baseEnv[2:]...)
	genericInput := `{"tool_name":"Edit","agent_id":"worker-a","agent_type":"worker","tool_input":{"file_path":"tracked.txt"}}`
	out, stderr, code = runDevritesAt(t, repo, candidate.Worktree, genericInput, genericEnv,
		"hook", "wright-scope", "--harness=codex")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("valid generic Forge worker was denied: exit=%d stdout=%s stderr=%s", code, out, stderr)
	}
	out, _, code = runDevritesAt(t, repo, candidate.Worktree,
		`{"tool_name":"Edit","agent_id":"worker-a","agent_type":"worker","tool_input":{"file_path":"outside.txt"}}`,
		genericEnv, "hook", "wright-scope", "--harness=codex")
	if code != 0 {
		t.Fatalf("generic Forge worker hook exit=%d", code)
	}
	if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
		t.Fatalf("generic Forge worker escaped its allowlist: %s", out)
	}

	replaceEnv := func(name, value string) []string {
		env := append([]string(nil), baseEnv...)
		prefix := name + "="
		for i := range env {
			if strings.HasPrefix(env[i], prefix) {
				env[i] = prefix + value
			}
		}
		return env
	}
	foreign := newForgeCLIRepo(t)
	for _, test := range []struct {
		name  string
		cwd   string
		input string
		env   []string
	}{
		{"primary root", repo, input, baseEnv},
		{"sibling candidate", manifest.Candidates[1].Worktree, input, baseEnv},
		{"foreign repository", foreign, input, baseEnv},
		{"wrong candidate", candidate.Worktree, input, replaceEnv("DEVRITES_FORGE_CANDIDATE", "B")},
		{"wrong worker", candidate.Worktree, input, replaceEnv("DEVRITES_FORGE_WORKER_ID", "worker-x")},
		{"wrong pid", candidate.Worktree, input, replaceEnv("DEVRITES_FORGE_WORKER_PID", "1")},
		{"wrong token", candidate.Worktree, input, replaceEnv("DEVRITES_FORGE_PROCESS_START", strings.Repeat("0", 64))},
		{"partial binding", candidate.Worktree, input, replaceEnv("DEVRITES_FORGE_PROCESS_START", "")},
		{"missing binding", candidate.Worktree, input, baseEnv[:2]},
		{"wrong hook agent", candidate.Worktree,
			`{"tool_name":"Edit","agent_id":"worker-x","tool_input":{"file_path":"tracked.txt"}}`, baseEnv},
	} {
		t.Run(test.name, func(t *testing.T) {
			out, stderr, code := runDevritesAt(t, repo, test.cwd, test.input, test.env,
				"hook", "wright-scope", "--harness=codex")
			if code != 0 {
				t.Fatalf("exit=%d stderr=%s", code, stderr)
			}
			if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
				t.Fatalf("invalid Forge binding was not denied: %s", out)
			}
		})
	}

	forgeGit(t, candidate.Worktree, "switch", "-c", "wrong-forge-branch")
	out, _, _ = runDevritesAt(t, repo, candidate.Worktree, input, baseEnv,
		"hook", "wright-scope", "--harness=codex")
	if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
		t.Fatal("wrong candidate branch was not denied")
	}
	forgeGit(t, candidate.Worktree, "switch", candidate.Branch)

	manifestPath := filepath.Join(repo, ".devrites", "work", "alpha", ".forge", manifest.RunID, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var tampered map[string]any
	if err := json.Unmarshal(raw, &tampered); err != nil {
		t.Fatal(err)
	}
	candidates := tampered["candidates"].([]any)
	candidates[0].(map[string]any)["worktree"] = foreign
	raw, err = json.MarshalIndent(tampered, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	out, _, _ = runDevritesAt(t, repo, candidate.Worktree, input, baseEnv,
		"hook", "wright-scope", "--harness=codex")
	if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
		t.Fatal("tampered Forge manifest was not denied")
	}
}

// --- issue 09: subagent-orient ---

func TestHookSubagentOrientInjectsDisciplineForDevritesAgent(t *testing.T) {
	for _, h := range []string{"claude", "codex"} {
		h := h
		t.Run(h, func(t *testing.T) {
			root := newWorkspace(t)
			in := `{"agent_type":"devrites-code-reviewer"}`
			out, errOut, code := runDevritesIO(t, root, in, nil, "hook", "subagent-orient", "--harness="+h)
			if code != 0 {
				t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
			}
			var env struct {
				HookSpecificOutput struct {
					HookEventName     string `json:"hookEventName"`
					AdditionalContext string `json:"additionalContext"`
				} `json:"hookSpecificOutput"`
			}
			if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); err != nil {
				t.Fatalf("stdout is not a valid SubagentStart envelope: %v\n%s", err, out)
			}
			if env.HookSpecificOutput.HookEventName != "SubagentStart" {
				t.Errorf("hookEventName = %q, want SubagentStart", env.HookSpecificOutput.HookEventName)
			}
			for _, want := range []string{"DevRites subagent", "Operating rules", "orchestration depth"} {
				if !strings.Contains(env.HookSpecificOutput.AdditionalContext, want) {
					t.Errorf("additionalContext missing %q", want)
				}
			}
		})
	}
}

func TestHookSubagentOrientSilentForNonDevritesAgent(t *testing.T) {
	root := newWorkspace(t)
	for _, in := range []string{`{"agent_type":"Explore"}`, `{"agent_type":""}`, `not json`, ``} {
		out, _, code := runDevritesIO(t, root, in, nil, "hook", "subagent-orient", "--harness=claude")
		if code != 0 || strings.TrimSpace(out) != "" {
			t.Errorf("want silent exit 0 for payload %q; got exit=%d out=%q", in, code, out)
		}
	}
}

func TestHookSubagentOrientCapturesWrightCanonicalBoundary(t *testing.T) {
	root := newWorkspace(t)
	writeActive(t, root, "auth-tokens")
	workspace := filepath.Join(root, "features", "auth-tokens")
	if err := os.MkdirAll(filepath.Join(workspace, ".reconcile-objects"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		".reconcile-base":      "base\n",
		".reconcile-allowlist": "",
		".reconcile-devrites":  "[]\n",
		"state.md":             "Phase: build\n",
		"browser-evidence.md":  "Prior root evidence\n",
	} {
		if err := os.WriteFile(filepath.Join(workspace, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	in := `{"agent_type":"devrites-slice-wright"}`
	out, errOut, code := runDevritesIO(t, root, in, nil, "hook", "subagent-orient", "--harness=claude")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if strings.Contains(out, "Wright boundary unavailable") {
		t.Fatalf("wright start was blocked: %s", out)
	}
	raw, err := os.ReadFile(filepath.Join(workspace, ".reconcile-wright-devrites"))
	if err != nil {
		t.Fatalf("read wright boundary: %v", err)
	}
	for _, path := range []string{"features/auth-tokens/state.md", "features/auth-tokens/browser-evidence.md"} {
		if !strings.Contains(string(raw), path) {
			t.Errorf("wright boundary omitted %s: %s", path, raw)
		}
	}
}

func hookPayload(t *testing.T, value map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func writeCodexAgentContract(t *testing.T, root, role string) {
	t.Helper()
	dir := filepath.Join(filepath.Dir(root), ".codex", "agents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	contract := "name = \"" + role + "\"\ndeveloper_instructions = '''\nROLE-CONTRACT:" + role + "\n'''\n"
	if err := os.WriteFile(filepath.Join(dir, role+".toml"), []byte(contract), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeCodexSkillContract(t *testing.T, root, skill, roles string) {
	t.Helper()
	dir := filepath.Join(filepath.Dir(root), ".agents", "skills", skill)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	contract := "---\nname: " + skill + "\ndescription: Test skill.\nrequired-agent-roles: " + roles + "\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(contract), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeCodexConditionalSkillContract(t *testing.T, root, skill, role string) {
	t.Helper()
	dir := filepath.Join(filepath.Dir(root), ".agents", "skills", skill)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	contract := "---\nname: " + skill + "\ndescription: Test skill.\nrequired-agent-roles: none\n---\n\nConditionally dispatch `" + role + "` when needed.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(contract), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeCodexV2Rollouts(
	t *testing.T,
	codeHome, projectRoot, sessionID, turnID, role, taskName string,
	includeWait, includeInstructions, includeResult bool,
) {
	t.Helper()
	dir := filepath.Join(codeHome, "sessions", "2026", "07", "25")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC()
	parent := []map[string]any{
		{
			"timestamp": at.Format(time.RFC3339Nano),
			"type":      "session_meta",
			"payload": map[string]any{
				"session_id": sessionID,
				"id":         sessionID,
				"cwd":        projectRoot,
			},
		},
		{
			"timestamp": at.Add(time.Millisecond).Format(time.RFC3339Nano),
			"type":      "response_item",
			"payload": map[string]any{
				"type":      "function_call",
				"name":      "spawn_agent",
				"arguments": `{"agent_type":"` + role + `","task_name":"` + taskName + `","fork_turns":"none","message":"encrypted"}`,
				"internal_chat_message_metadata_passthrough": map[string]any{
					"turn_id": turnID,
				},
			},
		},
	}
	if includeWait {
		parent = append(parent, map[string]any{
			"timestamp": at.Add(2 * time.Millisecond).Format(time.RFC3339Nano),
			"type":      "response_item",
			"payload": map[string]any{
				"type":      "function_call",
				"name":      "wait_agent",
				"arguments": `{"timeout_ms":3600000}`,
				"internal_chat_message_metadata_passthrough": map[string]any{
					"turn_id": turnID,
				},
			},
		})
	}
	if includeResult {
		parent = append(parent, map[string]any{
			"timestamp": at.Add(3 * time.Millisecond).Format(time.RFC3339Nano),
			"type":      "response_item",
			"payload": map[string]any{
				"type":      "agent_message",
				"author":    "/root/" + taskName,
				"recipient": "/root",
				"content": []map[string]any{{
					"type": "input_text",
					"text": "Message Type: FINAL_ANSWER\nPayload:\nagent-result/v1 complete",
				}},
				"internal_chat_message_metadata_passthrough": map[string]any{
					"turn_id": turnID,
				},
			},
		})
	}
	writeJSONLines(t, filepath.Join(dir, "rollout-parent-"+sessionID+".jsonl"), parent)

	child := []map[string]any{
		{
			"timestamp": at.Add(time.Millisecond).Format(time.RFC3339Nano),
			"type":      "session_meta",
			"payload": map[string]any{
				"session_id":       sessionID,
				"id":               "child-" + taskName,
				"parent_thread_id": sessionID,
				"cwd":              projectRoot,
				"agent_path":       "/root/" + taskName,
				"agent_role":       role,
			},
		},
	}
	if includeInstructions {
		child = append(child, map[string]any{
			"timestamp": at.Add(2 * time.Millisecond).Format(time.RFC3339Nano),
			"type":      "response_item",
			"payload": map[string]any{
				"type": "message",
				"role": "developer",
				"content": []map[string]any{{
					"type": "input_text",
					"text": "ROLE-CONTRACT:" + role,
				}},
			},
		})
	}
	if includeResult {
		child = append(child, map[string]any{
			"timestamp": at.Add(3 * time.Millisecond).Format(time.RFC3339Nano),
			"type":      "event_msg",
			"payload": map[string]any{
				"type":               "task_complete",
				"last_agent_message": "agent-result/v1 complete",
			},
		})
	}
	writeJSONLines(t, filepath.Join(dir, "rollout-child-"+taskName+".jsonl"), child)
}

func writeJSONLines(t *testing.T, path string, records []map[string]any) {
	t.Helper()
	var lines []string
	for _, record := range records {
		raw, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, string(raw))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCodexAgentDispatchArmsInstalledSkillRequirements(t *testing.T) {
	for _, tc := range []struct {
		name       string
		invocation string
		roles      string
		wantRoles  []string
	}{
		{
			name:       "single-role-dollar-invocation",
			invocation: "$rite-vet",
			roles:      "devrites-plan-reviewer",
			wantRoles:  []string{"devrites-plan-reviewer"},
		},
		{
			name:       "multi-role-slash-invocation",
			invocation: "/rite-review",
			roles:      "devrites-spec-reviewer, devrites-code-reviewer",
			wantRoles:  []string{"devrites-code-reviewer", "devrites-spec-reviewer"},
		},
		{
			name:       "declared-no-agent",
			invocation: "$rite-status",
			roles:      "none",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := newWorkspace(t)
			skill := strings.TrimLeft(tc.invocation, "$/")
			writeCodexSkillContract(t, root, skill, tc.roles)
			for _, role := range tc.wantRoles {
				writeCodexAgentContract(t, root, role)
			}
			sessionID, turnID := "session-"+tc.name, "turn-"+tc.name
			out, stderr, code := runDevritesIO(t, root, hookPayload(t, map[string]any{
				"hook_event_name": "UserPromptSubmit",
				"session_id":      sessionID,
				"turn_id":         turnID,
				"prompt":          tc.invocation,
			}), nil, "hook", "agent-dispatch", "--harness=codex")
			if code != 0 {
				t.Fatalf("arm exit=%d stderr=%s", code, stderr)
			}
			for _, role := range tc.wantRoles {
				if !strings.Contains(out, role) {
					t.Fatalf("arm output omitted %s: %s", role, out)
				}
			}

			stopOut, stopErr, stopCode := runDevritesIO(t, root, hookPayload(t, map[string]any{
				"hook_event_name":  "Stop",
				"session_id":       sessionID,
				"turn_id":          turnID,
				"stop_hook_active": false,
			}), nil, "hook", "agent-dispatch", "--harness=codex")
			if stopCode != 0 {
				t.Fatalf("stop exit=%d stderr=%s", stopCode, stopErr)
			}
			if len(tc.wantRoles) == 0 {
				if strings.TrimSpace(stopOut) != "" {
					t.Fatalf("no-agent skill was blocked: %s", stopOut)
				}
				return
			}
			for _, role := range tc.wantRoles {
				if strings.Contains(stopOut, role) {
					return
				}
			}
			t.Fatalf("required skill roles did not block Stop: %s", stopOut)
		})
	}
}

func TestCodexAgentDispatchPromptGivesExactNamedCallBeforeCompletion(t *testing.T) {
	root := newWorkspace(t)
	role := "devrites-plan-drafter"
	writeCodexAgentContract(t, root, role)
	writeCodexSkillContract(t, root, "rite-plan", role)

	out, stderr, code := runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      "session-plan-drafter-guidance",
		"turn_id":         "turn-plan-drafter-guidance",
		"prompt":          "$rite-plan repair",
	}), nil, "hook", "agent-dispatch", "--harness=codex")
	if code != 0 {
		t.Fatalf("arm exit=%d stderr=%s", code, stderr)
	}
	for _, want := range []string{
		"MANDATORY DISPATCH THIS TURN",
		"spawn_agent",
		"agent_type=" + role,
		"unique task_name",
		`fork_turns="none"`,
		".codex/agents/" + role + ".toml",
		"send agent_type anyway",
		"Wait for the returned child",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("initial dispatch guidance omitted %q:\n%s", want, out)
		}
	}
}

func TestCodexConditionalAgentDispatchRejectsDefaultAndArmsNamedRole(t *testing.T) {
	root := newWorkspace(t)
	role := "devrites-proof-runner"
	writeCodexAgentContract(t, root, role)
	writeCodexConditionalSkillContract(t, root, "rite-polish", role)
	sessionID, turnID := "session-conditional-role", "turn-conditional-role"

	out, stderr, code := runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"prompt":          "$rite-polish",
	}), nil, "hook", "agent-dispatch", "--harness=codex")
	if code != 0 {
		t.Fatalf("arm exit=%d stderr=%s", code, stderr)
	}
	for _, want := range []string{
		"CONDITIONAL DISPATCH RULE",
		"exact named agent_type=devrites-<role>",
		`fork_turns="none"`,
		"send agent_type anyway",
		"never use a default child",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("conditional dispatch guidance omitted %q:\n%s", want, out)
		}
	}

	for _, agentType := range []string{"", "default"} {
		out, stderr, code = runDevritesIO(t, root, hookPayload(t, map[string]any{
			"hook_event_name": "PreToolUse",
			"session_id":      sessionID,
			"turn_id":         turnID,
			"tool_name":       "spawn_agent",
			"tool_use_id":     "default-" + agentType,
			"tool_input": map[string]any{
				"agent_type": agentType,
				"task_name":  "conditional_default",
				"fork_turns": "none",
				"message":    "Run the conditional DevRites check.",
			},
		}), nil, "hook", "agent-dispatch", "--harness=codex")
		if code != 0 {
			t.Fatalf("default spawn exit=%d stderr=%s", code, stderr)
		}
		if decision, reason := parsePermissionDecision(t, out); decision != "deny" ||
			!strings.Contains(reason, "exact named agent_type=devrites-<role>") {
			t.Fatalf("conditional default child not denied: decision=%q reason=%q out=%s", decision, reason, out)
		}
	}

	out, stderr, code = runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "PreToolUse",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"tool_name":       "spawn_agent",
		"tool_use_id":     "missing-named-type",
		"tool_input": map[string]any{
			"task_name":  "conditional_missing_type",
			"fork_turns": "none",
			"message":    "Read .codex/agents/" + role + ".toml and run the check.",
		},
	}), nil, "hook", "agent-dispatch", "--harness=codex")
	if code != 0 {
		t.Fatalf("missing type spawn exit=%d stderr=%s", code, stderr)
	}
	if decision, reason := parsePermissionDecision(t, out); decision != "deny" ||
		!strings.Contains(reason, "agent_type="+role) ||
		!strings.Contains(reason, "send agent_type anyway") {
		t.Fatalf("missing named type did not get exact retry: decision=%q reason=%q out=%s", decision, reason, out)
	}

	out, stderr, code = runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "PreToolUse",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"tool_name":       "spawn_agent",
		"tool_use_id":     "named-conditional",
		"tool_input": map[string]any{
			"agent_type": role,
			"task_name":  "conditional_proof",
			"fork_turns": "none",
			"message":    "Run the conditional DevRites check.",
		},
	}), nil, "hook", "agent-dispatch", "--harness=codex")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("named conditional spawn rejected: exit=%d out=%s stderr=%s", code, out, stderr)
	}

	out, stderr, code = runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "Stop",
		"session_id":      sessionID,
		"turn_id":         turnID,
	}), nil, "hook", "agent-dispatch", "--harness=codex")
	if code != 0 || !strings.Contains(out, "DevRites dispatch for "+role+" is not complete") {
		t.Fatalf("named conditional spawn was not armed: exit=%d out=%s stderr=%s", code, out, stderr)
	}
}

func TestCodexAgentDispatchAllowsDefaultChildOutsideSkillTurn(t *testing.T) {
	root := newWorkspace(t)
	out, stderr, code := runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "PreToolUse",
		"session_id":      "session-unrelated-default",
		"turn_id":         "turn-unrelated-default",
		"tool_name":       "spawn_agent",
		"tool_use_id":     "unrelated-default",
		"tool_input": map[string]any{
			"agent_type": "default",
			"task_name":  "unrelated",
			"fork_turns": "all",
			"message":    "Handle an unrelated bounded task.",
		},
	}), nil, "hook", "agent-dispatch", "--harness=codex")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("unrelated default child was rejected: exit=%d out=%s stderr=%s", code, out, stderr)
	}
}

func TestCodexAgentDispatchRejectsInvalidInstalledSkillRequirements(t *testing.T) {
	root := newWorkspace(t)
	writeCodexSkillContract(t, root, "rite-vet", "root-inline-reviewer")
	out, stderr, code := runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      "session-invalid-skill",
		"turn_id":         "turn-invalid-skill",
		"prompt":          "$rite-vet",
	}), nil, "hook", "agent-dispatch", "--harness=codex")
	if code != 0 {
		t.Fatalf("invalid metadata exit=%d stderr=%s", code, stderr)
	}
	var decision struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &decision); err != nil {
		t.Fatalf("invalid metadata response: %v\n%s", err, out)
	}
	if decision.Decision != "block" || !strings.Contains(decision.Reason, "invalid required agent role") {
		t.Fatalf("invalid skill metadata did not fail closed: %#v", decision)
	}
	stopOut, stopErr, stopCode := runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "Stop",
		"session_id":      "session-invalid-skill",
		"turn_id":         "turn-invalid-skill",
	}), nil, "hook", "agent-dispatch", "--harness=codex")
	if stopCode != 0 || !strings.Contains(stopOut, "required-agent-roles contract is invalid") {
		t.Fatalf("invalid skill metadata did not remain fail closed at Stop: exit=%d out=%s stderr=%s", stopCode, stopOut, stopErr)
	}
}

func TestCodexAgentDispatchDoesNotArmSkillNamesInsideURLs(t *testing.T) {
	root := newWorkspace(t)
	writeCodexSkillContract(t, root, "rite-vet", "devrites-plan-reviewer")
	sessionID, turnID := "session-skill-url", "turn-skill-url"
	runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"prompt":          "Read https://example.test/rite-vet before answering.",
	}), nil, "hook", "agent-dispatch", "--harness=codex")
	out, stderr, code := runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "Stop",
		"session_id":      sessionID,
		"turn_id":         turnID,
	}), nil, "hook", "agent-dispatch", "--harness=codex")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("URL path armed a skill: exit=%d out=%s stderr=%s", code, out, stderr)
	}
}

func TestCodexAgentDispatchDoesNotArmBareAgentContractReference(t *testing.T) {
	root := newWorkspace(t)
	writeCodexAgentContract(t, root, "devrites-security-auditor")
	sessionID, turnID := "session-agent-reference", "turn-agent-reference"
	runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"prompt":          "Explain .codex/agents/devrites-security-auditor.toml without running it.",
	}), nil, "hook", "agent-dispatch", "--harness=codex")
	out, stderr, code := runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "Stop",
		"session_id":      sessionID,
		"turn_id":         turnID,
	}), nil, "hook", "agent-dispatch", "--harness=codex")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("bare agent reference armed dispatch: exit=%d out=%s stderr=%s", code, out, stderr)
	}
}

func TestCodexAgentDispatchBlocksFalseWaitAndStop(t *testing.T) {
	root := newWorkspace(t)
	writeCodexSkillContract(t, root, "devrites-audit", "devrites-security-auditor")
	sessionID, turnID := "session-false-wait", "turn-false-wait"
	prompt := hookPayload(t, map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"prompt":          "$devrites-audit Call spawn_agent and tell it to read .codex/agents/devrites-security-auditor.toml.",
	})
	if _, stderr, code := runDevritesIO(t, root, prompt, nil,
		"hook", "agent-dispatch", "--harness=codex"); code != 0 {
		t.Fatalf("arm exit=%d stderr=%s", code, stderr)
	}

	wait := hookPayload(t, map[string]any{
		"hook_event_name": "PreToolUse",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"tool_name":       "wait",
		"tool_input":      map[string]any{"receiver_thread_ids": []string{}},
	})
	out, stderr, code := runDevritesIO(t, root, wait, nil,
		"hook", "agent-dispatch", "--harness=codex")
	if code != 0 {
		t.Fatalf("wait exit=%d stderr=%s", code, stderr)
	}
	if decision, reason := parsePermissionDecision(t, out); decision != "deny" ||
		!strings.Contains(reason, "spawn_agent") {
		t.Fatalf("false wait not denied: decision=%q reason=%q out=%s", decision, reason, out)
	}

	stop := hookPayload(t, map[string]any{
		"hook_event_name":  "Stop",
		"session_id":       sessionID,
		"turn_id":          turnID,
		"stop_hook_active": false,
	})
	out, stderr, code = runDevritesIO(t, root, stop, nil,
		"hook", "agent-dispatch", "--harness=codex")
	if code != 0 {
		t.Fatalf("stop exit=%d stderr=%s", code, stderr)
	}
	var stopDecision struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &stopDecision); err != nil {
		t.Fatalf("invalid Stop response: %v\n%s", err, out)
	}
	if stopDecision.Decision != "block" ||
		!strings.Contains(stopDecision.Reason, "spawn_agent") ||
		!strings.Contains(stopDecision.Reason, "visible tool schema") ||
		!strings.Contains(stopDecision.Reason, "send agent_type anyway") {
		t.Fatalf("false completion not blocked: %#v", stopDecision)
	}

	stop = hookPayload(t, map[string]any{
		"hook_event_name":  "Stop",
		"session_id":       sessionID,
		"turn_id":          turnID,
		"stop_hook_active": true,
	})
	out, stderr, code = runDevritesIO(t, root, stop, nil,
		"hook", "agent-dispatch", "--harness=codex")
	if code != 0 {
		t.Fatalf("second stop exit=%d stderr=%s", code, stderr)
	}
	var terminal struct {
		Continue   bool   `json:"continue"`
		StopReason string `json:"stopReason"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &terminal); err != nil {
		t.Fatalf("invalid terminal Stop response: %v\n%s", err, out)
	}
	if terminal.Continue || !strings.Contains(terminal.StopReason, "spawn_agent") {
		t.Fatalf("second false completion was not stopped: %#v", terminal)
	}
}

func TestCodexAgentDispatchConfirmsGenericRoleAndResult(t *testing.T) {
	root := newWorkspace(t)
	sessionID, turnID := "session-complete", "turn-complete"
	role := "devrites-security-auditor"
	writeCodexAgentContract(t, root, role)
	writeCodexSkillContract(t, root, "devrites-audit", role)

	prompt := hookPayload(t, map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"prompt":          "$devrites-audit Call spawn_agent and tell it to read .codex/agents/" + role + ".toml.",
	})
	runDevritesIO(t, root, prompt, nil, "hook", "agent-dispatch", "--harness=codex")

	spawn := hookPayload(t, map[string]any{
		"hook_event_name": "PreToolUse",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"tool_name":       "spawn_agent",
		"tool_use_id":     "spawn-1",
		"tool_input": map[string]any{
			"agent_type": "explorer",
			"fork_turns": "none",
			"message":    "Read .codex/agents/" + role + ".toml and return the requested report.",
		},
	})
	out, stderr, code := runDevritesIO(t, root, spawn, nil,
		"hook", "agent-dispatch", "--harness=codex")
	if code != 0 {
		t.Fatalf("spawn exit=%d out=%s stderr=%s", code, out, stderr)
	}
	var rewrite struct {
		HookSpecificOutput struct {
			PermissionDecision string `json:"permissionDecision"`
			UpdatedInput       struct {
				Message string `json:"message"`
			} `json:"updatedInput"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &rewrite); err != nil {
		t.Fatalf("invalid spawn rewrite: %v\n%s", err, out)
	}
	if rewrite.HookSpecificOutput.PermissionDecision != "allow" ||
		!strings.Contains(rewrite.HookSpecificOutput.UpdatedInput.Message, "ROLE-CONTRACT:"+role) ||
		!strings.Contains(rewrite.HookSpecificOutput.UpdatedInput.Message, "return the requested report") {
		t.Fatalf("generic spawn did not receive developer_instructions: %#v", rewrite)
	}

	start := hookPayload(t, map[string]any{
		"hook_event_name": "SubagentStart",
		"session_id":      sessionID,
		"turn_id":         "child-turn",
		"agent_id":        "agent-1",
		"agent_type":      "explorer",
	})
	out, stderr, code = runDevritesIO(t, root, start, nil,
		"hook", "subagent-orient", "--harness=codex")
	if code != 0 {
		t.Fatalf("start exit=%d stderr=%s", code, stderr)
	}
	var startEnvelope struct {
		HookSpecificOutput struct {
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &startEnvelope); err != nil {
		t.Fatalf("invalid start response: %v\n%s", err, out)
	}
	if !strings.Contains(startEnvelope.HookSpecificOutput.AdditionalContext,
		".codex/agents/"+role+".toml") {
		t.Fatalf("generic child did not receive role contract: %s", out)
	}

	stopAgent := hookPayload(t, map[string]any{
		"hook_event_name":        "SubagentStop",
		"session_id":             sessionID,
		"turn_id":                "child-turn",
		"agent_id":               "agent-1",
		"agent_type":             "explorer",
		"last_assistant_message": "agent-result/v1 complete",
	})
	if out, stderr, code := runDevritesIO(t, root, stopAgent, nil,
		"hook", "agent-dispatch", "--harness=codex"); code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("subagent stop exit=%d out=%s stderr=%s", code, out, stderr)
	}

	stop := hookPayload(t, map[string]any{
		"hook_event_name":  "Stop",
		"session_id":       sessionID,
		"turn_id":          turnID,
		"stop_hook_active": false,
	})
	out, stderr, code = runDevritesIO(t, root, stop, nil,
		"hook", "agent-dispatch", "--harness=codex")
	if code != 0 {
		t.Fatalf("unawaited stop exit=%d stderr=%s", code, stderr)
	}
	var blocked struct {
		Decision string `json:"decision"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &blocked); err != nil {
		t.Fatalf("invalid unawaited Stop response: %v\n%s", err, out)
	}
	if blocked.Decision != "block" {
		t.Fatalf("unawaited child result passed Stop: %s", out)
	}

	wait := hookPayload(t, map[string]any{
		"hook_event_name": "PreToolUse",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"tool_name":       "wait",
		"tool_input":      map[string]any{"ids": []string{"agent-1"}},
	})
	if out, stderr, code := runDevritesIO(t, root, wait, nil,
		"hook", "agent-dispatch", "--harness=codex"); code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("wait exit=%d out=%s stderr=%s", code, out, stderr)
	}

	if out, stderr, code := runDevritesIO(t, root, stop, nil,
		"hook", "agent-dispatch", "--harness=codex"); code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("resolved stop exit=%d out=%s stderr=%s", code, out, stderr)
	}
}

func TestCodexAgentDispatchConfirmsDurableV2NamedRole(t *testing.T) {
	root := newWorkspace(t)
	codeHome := t.TempDir()
	sessionID, turnID := "session-v2-named", "turn-v2-named"
	role := "devrites-security-auditor"
	writeCodexAgentContract(t, root, role)
	writeCodexSkillContract(t, root, "devrites-audit", role)

	runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"prompt":          "$devrites-audit Call spawn_agent and tell it to read .codex/agents/" + role + ".toml.",
	}), nil, "hook", "agent-dispatch", "--harness=codex")

	writeCodexV2Rollouts(
		t, codeHome, filepath.Dir(root), sessionID, turnID, role,
		"devrites_security_auditor_named", true, true, true,
	)
	stop := hookPayload(t, map[string]any{
		"hook_event_name":  "Stop",
		"session_id":       sessionID,
		"turn_id":          turnID,
		"stop_hook_active": false,
	})
	if out, stderr, code := runDevritesIO(t, root, stop,
		[]string{"CODEX_HOME=" + codeHome},
		"hook", "agent-dispatch", "--harness=codex"); code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("durable v2 stop exit=%d out=%s stderr=%s", code, out, stderr)
	}
}

func TestCodexAgentDispatchConfirmsDurableV2ChildRoleWhenParentOmitsAgentType(t *testing.T) {
	root := newWorkspace(t)
	codeHome := t.TempDir()
	sessionID, turnID := "session-v2-hidden-role", "turn-v2-hidden-role"
	role := "devrites-plan-drafter"
	writeCodexAgentContract(t, root, role)
	writeCodexSkillContract(t, root, "rite-plan", role)

	runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"prompt":          "$rite-plan unblock",
	}), nil, "hook", "agent-dispatch", "--harness=codex")

	taskName := "devrites_plan_drafter_hidden_role"
	writeCodexV2Rollouts(
		t, codeHome, filepath.Dir(root), sessionID, turnID, role,
		taskName, true, true, true,
	)
	parentPath := filepath.Join(codeHome, "sessions", "2026", "07", "25", "rollout-parent-"+sessionID+".jsonl")
	parent, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	hiddenParentRole := bytes.ReplaceAll(parent, []byte(`\"agent_type\":\"`+role+`\"`), []byte(`\"agent_type\":\"\"`))
	if bytes.Equal(parent, hiddenParentRole) {
		t.Fatal("fixture did not hide the parent agent_type")
	}
	if err := os.WriteFile(parentPath, hiddenParentRole, 0o600); err != nil {
		t.Fatal(err)
	}
	stop := hookPayload(t, map[string]any{
		"hook_event_name": "Stop",
		"session_id":      sessionID,
		"turn_id":         turnID,
	})
	if out, stderr, code := runDevritesIO(t, root, stop,
		[]string{"CODEX_HOME=" + codeHome},
		"hook", "agent-dispatch", "--harness=codex"); code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("hidden parent role stop exit=%d out=%s stderr=%s", code, out, stderr)
	}
}

func TestCodexAgentDispatchConfirmsConditionalDurableV2NamedRole(t *testing.T) {
	root := newWorkspace(t)
	codeHome := t.TempDir()
	sessionID, turnID := "session-v2-conditional", "turn-v2-conditional"
	role := "devrites-security-auditor"
	writeCodexAgentContract(t, root, role)
	writeCodexConditionalSkillContract(t, root, "devrites-audit", role)

	runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"prompt":          "$devrites-audit security",
	}), nil, "hook", "agent-dispatch", "--harness=codex")

	writeCodexV2Rollouts(
		t, codeHome, filepath.Dir(root), sessionID, turnID, role,
		"devrites_security_auditor_conditional", true, true, true,
	)
	stop := hookPayload(t, map[string]any{
		"hook_event_name": "Stop",
		"session_id":      sessionID,
		"turn_id":         turnID,
	})
	if out, stderr, code := runDevritesIO(t, root, stop,
		[]string{"CODEX_HOME=" + codeHome},
		"hook", "agent-dispatch", "--harness=codex"); code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("conditional durable v2 stop exit=%d out=%s stderr=%s", code, out, stderr)
	}
}

func TestCodexAgentDispatchRejectsConditionalDurableV2DefaultChild(t *testing.T) {
	root := newWorkspace(t)
	codeHome := t.TempDir()
	sessionID, turnID := "session-v2-default", "turn-v2-default"
	writeCodexConditionalSkillContract(t, root, "devrites-audit", "devrites-security-auditor")

	runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"prompt":          "$devrites-audit security",
	}), nil, "hook", "agent-dispatch", "--harness=codex")

	writeCodexV2Rollouts(
		t, codeHome, filepath.Dir(root), sessionID, turnID, "default",
		"default_conditional_child", true, true, true,
	)
	stop := hookPayload(t, map[string]any{
		"hook_event_name": "Stop",
		"session_id":      sessionID,
		"turn_id":         turnID,
	})
	out, stderr, code := runDevritesIO(t, root, stop,
		[]string{"CODEX_HOME=" + codeHome},
		"hook", "agent-dispatch", "--harness=codex")
	if code != 0 {
		t.Fatalf("conditional default stop exit=%d stderr=%s", code, stderr)
	}
	var decision struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &decision); err != nil {
		t.Fatalf("invalid conditional default response: %v\n%s", err, out)
	}
	if decision.Decision != "block" ||
		!strings.Contains(decision.Reason, "default or non-DevRites child") ||
		!strings.Contains(decision.Reason, "never use a default child") {
		t.Fatalf("conditional V2 default child passed: %#v", decision)
	}
}

func TestCodexAgentDispatchRejectsIncompleteDurableV2(t *testing.T) {
	for _, tc := range []struct {
		name          string
		role          string
		includeWait   bool
		includeRules  bool
		includeResult bool
	}{
		{name: "generic-role", role: "explorer", includeWait: true, includeRules: true, includeResult: true},
		{name: "missing-wait", role: "devrites-security-auditor", includeRules: true, includeResult: true},
		{name: "missing-rules", role: "devrites-security-auditor", includeWait: true, includeResult: true},
		{name: "missing-result", role: "devrites-security-auditor", includeWait: true, includeRules: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := newWorkspace(t)
			codeHome := t.TempDir()
			sessionID, turnID := "session-"+tc.name, "turn-"+tc.name
			role := "devrites-security-auditor"
			writeCodexAgentContract(t, root, role)
			writeCodexSkillContract(t, root, "devrites-audit", role)
			runDevritesIO(t, root, hookPayload(t, map[string]any{
				"hook_event_name": "UserPromptSubmit",
				"session_id":      sessionID,
				"turn_id":         turnID,
				"prompt":          "$devrites-audit Read .codex/agents/" + role + ".toml.",
			}), nil, "hook", "agent-dispatch", "--harness=codex")
			writeCodexV2Rollouts(
				t, codeHome, filepath.Dir(root), sessionID, turnID, tc.role,
				"devrites_security_auditor_named", tc.includeWait, tc.includeRules, tc.includeResult,
			)
			stop := hookPayload(t, map[string]any{
				"hook_event_name": "Stop",
				"session_id":      sessionID,
				"turn_id":         turnID,
			})
			out, stderr, code := runDevritesIO(t, root, stop,
				[]string{"CODEX_HOME=" + codeHome},
				"hook", "agent-dispatch", "--harness=codex")
			if code != 0 {
				t.Fatalf("stop exit=%d stderr=%s", code, stderr)
			}
			var decision struct {
				Decision string `json:"decision"`
			}
			if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &decision); err != nil {
				t.Fatalf("invalid Stop response: %v\n%s", err, out)
			}
			if decision.Decision != "block" {
				t.Fatalf("incomplete durable V2 dispatch passed: %s", out)
			}
		})
	}
}

func TestCodexAgentDispatchGatesReconcileCloseOnBoundWrightResult(t *testing.T) {
	root := newWorkspace(t)
	writeActive(t, root, "auth-tokens")
	workspace := filepath.Join(root, "features", "auth-tokens")
	for name, body := range map[string]string{
		".reconcile-base":      ".reconcile-base\n",
		".reconcile-allowlist": ".reconcile-allowlist\n",
		".reconcile-devrites":  "[]\n",
	} {
		if err := os.WriteFile(filepath.Join(workspace, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(workspace, ".reconcile-objects"), 0o700); err != nil {
		t.Fatal(err)
	}
	sessionID, turnID := "session-wright", "turn-wright"
	writeCodexAgentContract(t, root, "devrites-slice-wright")
	writeCodexSkillContract(t, root, "rite-build", "devrites-slice-wright")
	runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"prompt":          "$rite-build",
	}), nil, "hook", "agent-dispatch", "--harness=codex")

	check := hookPayload(t, map[string]any{
		"hook_event_name": "PreToolUse",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"tool_name":       "Bash",
		"tool_input":      map[string]any{"command": "rtk devrites-engine reconcile check"},
	})
	if out, stderr, code := runDevritesIO(t, root, check, nil,
		"hook", "agent-dispatch", "--harness=codex"); code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("ordinary reconcile check exit=%d out=%s stderr=%s", code, out, stderr)
	}

	closeWindow := hookPayload(t, map[string]any{
		"hook_event_name": "PreToolUse",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"tool_name":       "Bash",
		"tool_input":      map[string]any{"command": "rtk devrites-engine reconcile close"},
	})
	out, _, _ := runDevritesIO(t, root, closeWindow, nil,
		"hook", "agent-dispatch", "--harness=codex")
	if decision, reason := parsePermissionDecision(t, out); decision != "deny" ||
		!strings.Contains(reason, "confirmed") {
		t.Fatalf("premature reconcile not denied: decision=%q reason=%q", decision, reason)
	}

	for _, payload := range []map[string]any{
		{
			"hook_event_name": "PreToolUse",
			"session_id":      sessionID,
			"turn_id":         turnID,
			"tool_name":       "spawn_agent",
			"tool_use_id":     "spawn-wright",
			"tool_input": map[string]any{
				"agent_type": "devrites-slice-wright",
				"fork_turns": "none",
				"message":    "Read .codex/agents/devrites-slice-wright.toml.",
			},
		},
		{
			"hook_event_name": "SubagentStart",
			"session_id":      sessionID,
			"turn_id":         "child-wright",
			"agent_id":        "agent-wright",
			"agent_type":      "devrites-slice-wright",
		},
		{
			"hook_event_name":        "SubagentStop",
			"session_id":             sessionID,
			"turn_id":                "child-wright",
			"agent_id":               "agent-wright",
			"agent_type":             "devrites-slice-wright",
			"last_assistant_message": "agent-result/v1 complete",
		},
		{
			"hook_event_name": "PreToolUse",
			"session_id":      sessionID,
			"turn_id":         turnID,
			"tool_name":       "wait",
			"tool_input":      map[string]any{"ids": []string{"agent-wright"}},
		},
	} {
		hook := "agent-dispatch"
		if payload["hook_event_name"] == "SubagentStart" {
			hook = "subagent-orient"
		}
		if _, stderr, code := runDevritesIO(t, root, hookPayload(t, payload), nil,
			"hook", hook, "--harness=codex"); code != 0 {
			t.Fatalf("%s exit=%d stderr=%s", hook, code, stderr)
		}
	}
	if _, err := os.Stat(filepath.Join(workspace, ".reconcile-wright-devrites")); err != nil {
		t.Fatalf("wright start did not capture canonical boundary: %v", err)
	}

	if out, stderr, code := runDevritesIO(t, root, closeWindow, nil,
		"hook", "agent-dispatch", "--harness=codex"); code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("completed wright reconcile exit=%d out=%s stderr=%s", code, out, stderr)
	}
}

func TestCodexAgentDispatchCapturesWrightBoundaryBeforeSpawn(t *testing.T) {
	root := newWorkspace(t)
	writeActive(t, root, "auth-tokens")
	workspace := filepath.Join(root, "features", "auth-tokens")
	for name, body := range map[string]string{
		".reconcile-base":      ".reconcile-base\n",
		".reconcile-allowlist": ".reconcile-allowlist\n",
		".reconcile-devrites":  "[]\n",
		"action.log":           "Prior root action\n",
		"browser-evidence.md":  "Prior root evidence\n",
		"decisions.md":         "Prior root decision\n",
		"evidence.md":          "Prior root evidence\n",
		"footprint.log":        "Prior root footprint\n",
		"state.md":             "Phase: build\n",
		"touched-files.md":     "Prior root manifest\n",
	} {
		if err := os.WriteFile(filepath.Join(workspace, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(workspace, ".reconcile-objects"), 0o700); err != nil {
		t.Fatal(err)
	}

	sessionID, turnID := "session-wright-spawn", "turn-wright-spawn"
	writeCodexAgentContract(t, root, "devrites-slice-wright")
	writeCodexSkillContract(t, root, "rite-build", "devrites-slice-wright")
	runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"prompt":          "$rite-build",
	}), nil, "hook", "agent-dispatch", "--harness=codex")

	spawn := hookPayload(t, map[string]any{
		"hook_event_name": "PreToolUse",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"tool_name":       "spawn_agent",
		"tool_use_id":     "spawn-wright",
		"tool_input": map[string]any{
			"agent_type": "devrites-slice-wright",
			"fork_turns": "none",
			"message":    "Read .codex/agents/devrites-slice-wright.toml.",
		},
	})
	if out, stderr, code := runDevritesIO(t, root, spawn, nil,
		"hook", "agent-dispatch", "--harness=codex"); code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("spawn exit=%d out=%s stderr=%s", code, out, stderr)
	}

	raw, err := os.ReadFile(filepath.Join(workspace, ".reconcile-wright-devrites"))
	if err != nil {
		t.Fatalf("read wright boundary: %v", err)
	}
	for _, path := range []string{
		"features/auth-tokens/action.log",
		"features/auth-tokens/browser-evidence.md",
		"features/auth-tokens/decisions.md",
		"features/auth-tokens/evidence.md",
		"features/auth-tokens/footprint.log",
		"features/auth-tokens/state.md",
		"features/auth-tokens/touched-files.md",
	} {
		if !strings.Contains(string(raw), path) {
			t.Errorf("wright boundary omitted %s: %s", path, raw)
		}
	}
}

func TestCodexAgentDispatchRefreshesUnarmedWrightBoundaryUntilDurableSpawn(t *testing.T) {
	root := newWorkspace(t)
	codeHome := t.TempDir()
	writeActive(t, root, "auth-tokens")
	workspace := filepath.Join(root, "features", "auth-tokens")
	for name, body := range map[string]string{
		".reconcile-base":      ".reconcile-base\n",
		".reconcile-allowlist": ".reconcile-allowlist\n",
		".reconcile-devrites":  "[]\n",
		"action.log":           "Prior root action\n",
		"browser-evidence.md":  "Prior root evidence\n",
		"decisions.md":         "Prior root decision\n",
		"evidence.md":          "Prior root evidence\n",
		"footprint.log":        "Prior root footprint\n",
		"state.md":             "Phase: ready\n",
		"touched-files.md":     "Prior root manifest\n",
	} {
		if err := os.WriteFile(filepath.Join(workspace, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(workspace, ".reconcile-objects"), 0o700); err != nil {
		t.Fatal(err)
	}

	sessionID, turnID := "session-wright-durable", "turn-wright-durable"
	role := "devrites-slice-wright"
	writeCodexAgentContract(t, root, role)
	env := []string{"CODEX_HOME=" + codeHome}
	runDevritesIO(t, root, hookPayload(t, map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"prompt":          "Retry the retained reconciliation check.",
	}), env, "hook", "agent-dispatch", "--harness=codex")

	wrightState := filepath.Join(workspace, ".reconcile-wright-devrites")
	initial, err := os.ReadFile(wrightState)
	if err != nil {
		t.Fatalf("read initial wright boundary: %v", err)
	}
	for _, path := range []string{
		"features/auth-tokens/action.log",
		"features/auth-tokens/browser-evidence.md",
		"features/auth-tokens/decisions.md",
		"features/auth-tokens/evidence.md",
		"features/auth-tokens/footprint.log",
		"features/auth-tokens/state.md",
		"features/auth-tokens/touched-files.md",
	} {
		if !strings.Contains(string(initial), path) {
			t.Errorf("pending wright boundary omitted %s: %s", path, initial)
		}
	}
	if err := os.WriteFile(filepath.Join(workspace, "state.md"), []byte("Phase: build\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	postTool := hookPayload(t, map[string]any{
		"hook_event_name": "PostToolUse",
		"session_id":      sessionID,
		"turn_id":         turnID,
		"tool_name":       "apply_patch",
		"tool_use_id":     "root-state-edit",
		"tool_input":      map[string]any{"file_path": "state.md"},
	})
	if out, stderr, code := runDevritesIO(t, root, postTool, env,
		"hook", "agent-dispatch", "--harness=codex"); code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("root post-tool refresh exit=%d out=%s stderr=%s", code, out, stderr)
	}
	beforeSpawn, err := os.ReadFile(wrightState)
	if err != nil {
		t.Fatalf("read refreshed wright boundary: %v", err)
	}
	if bytes.Equal(initial, beforeSpawn) {
		t.Fatal("root PostToolUse did not refresh the pending wright boundary")
	}

	writeCodexV2Rollouts(
		t, codeHome, filepath.Dir(root), sessionID, turnID, role,
		"devrites_slice_wright_unhooked", true, true, true,
	)
	if err := os.WriteFile(filepath.Join(workspace, "state.md"), []byte("Writer mutation\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, stderr, code := runDevritesIO(t, root, postTool, env,
		"hook", "agent-dispatch", "--harness=codex"); code != 0 || strings.TrimSpace(out) != "" {
		t.Fatalf("post-spawn PostToolUse exit=%d out=%s stderr=%s", code, out, stderr)
	}
	afterSpawn, err := os.ReadFile(wrightState)
	if err != nil {
		t.Fatalf("read frozen wright boundary: %v", err)
	}
	if !bytes.Equal(beforeSpawn, afterSpawn) {
		t.Fatal("durable V2 spawn did not freeze the pre-writer canonical boundary")
	}
}

func TestCodexAgentDispatchBindsDurableV2WrightToReconcileWindow(t *testing.T) {
	for _, tc := range []struct {
		name          string
		changeWindow  bool
		wantPermitted bool
	}{
		{name: "same-window", wantPermitted: true},
		{name: "changed-window", changeWindow: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := newWorkspace(t)
			codeHome := t.TempDir()
			writeActive(t, root, "auth-tokens")
			workspace := filepath.Join(root, "features", "auth-tokens")
			for _, name := range []string{".reconcile-base", ".reconcile-allowlist", ".reconcile-devrites"} {
				if err := os.WriteFile(filepath.Join(workspace, name), []byte(name+"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			sessionID, turnID := "session-v2-wright-"+tc.name, "turn-v2-wright-"+tc.name
			role := "devrites-slice-wright"
			writeCodexAgentContract(t, root, role)
			writeCodexSkillContract(t, root, "rite-build", role)
			runDevritesIO(t, root, hookPayload(t, map[string]any{
				"hook_event_name": "UserPromptSubmit",
				"session_id":      sessionID,
				"turn_id":         turnID,
				"prompt":          "$rite-build",
			}), nil, "hook", "agent-dispatch", "--harness=codex")
			writeCodexV2Rollouts(
				t, codeHome, filepath.Dir(root), sessionID, turnID, role,
				"devrites_slice_wright_named", true, true, true,
			)
			if tc.changeWindow {
				future := time.Now().Add(time.Minute)
				if err := os.Chtimes(
					filepath.Join(workspace, ".reconcile-devrites"), future, future,
				); err != nil {
					t.Fatal(err)
				}
			}
			reconcile := hookPayload(t, map[string]any{
				"hook_event_name": "PreToolUse",
				"session_id":      sessionID,
				"turn_id":         turnID,
				"tool_name":       "Bash",
				"tool_input":      map[string]any{"command": "rtk devrites-engine reconcile close"},
			})
			out, stderr, code := runDevritesIO(t, root, reconcile,
				[]string{"CODEX_HOME=" + codeHome},
				"hook", "agent-dispatch", "--harness=codex")
			if code != 0 {
				t.Fatalf("reconcile exit=%d stderr=%s", code, stderr)
			}
			if tc.wantPermitted && strings.TrimSpace(out) != "" {
				t.Fatalf("same-window V2 wright was rejected: %s", out)
			}
			if !tc.wantPermitted {
				if decision, _ := parsePermissionDecision(t, out); decision != "deny" {
					t.Fatalf("changed-window V2 wright was accepted: %s", out)
				}
				return
			}
			for _, name := range []string{".reconcile-base", ".reconcile-allowlist", ".reconcile-devrites"} {
				if err := os.Remove(filepath.Join(workspace, name)); err != nil {
					t.Fatal(err)
				}
			}
			stop := hookPayload(t, map[string]any{
				"hook_event_name": "Stop",
				"session_id":      sessionID,
				"turn_id":         turnID,
			})
			if out, stderr, code := runDevritesIO(t, root, stop, nil,
				"hook", "agent-dispatch", "--harness=codex"); code != 0 || strings.TrimSpace(out) != "" {
				t.Fatalf("persisted V2 wright receipt was lost after close: exit=%d out=%s stderr=%s", code, out, stderr)
			}
		})
	}
}

func TestCodexAgentDispatchRejectsWrongGenericRole(t *testing.T) {
	root := newWorkspace(t)
	writeCodexAgentContract(t, root, "devrites-security-auditor")
	spawn := hookPayload(t, map[string]any{
		"hook_event_name": "PreToolUse",
		"session_id":      "session-role",
		"turn_id":         "turn-role",
		"tool_name":       "spawn_agent",
		"tool_use_id":     "spawn-role",
		"tool_input": map[string]any{
			"agent_type": "worker",
			"fork_turns": "none",
			"message":    "Read .codex/agents/devrites-security-auditor.toml.",
		},
	})
	out, stderr, code := runDevritesIO(t, root, spawn, nil,
		"hook", "agent-dispatch", "--harness=codex")
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr)
	}
	if decision, reason := parsePermissionDecision(t, out); decision != "deny" ||
		!strings.Contains(reason, "devrites-slice-wright") {
		t.Fatalf("worker role mismatch not denied: decision=%q reason=%q", decision, reason)
	}
}
