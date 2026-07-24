package main_test

// CLI coverage for Issue 03's dual-harness `hook orient` behavior and Issue 04's
// `hook stop-gate`. These tests run the binary against a fixture workspace and
// check stdout and exit status. parity_test.go owns the parity oracle.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
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
