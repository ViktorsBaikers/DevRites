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
				"host-specific DevRites command",
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

// --- issue 09: allow (migrated to engine subcommands) ---

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
		"devrites-engine reindex",                      // writes the SQLite cache
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
// permissionDecision + reason, failing if the shape is wrong.
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
// first newline, so it only ever scans line 1 — a latent bug. The Go port scans
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

// The agent-required gate reads agent_type ONLY (matching the bash node parse), so
// a payload that carries the subagent name under subagent_type but not agent_type
// has no identity and is allowed — no false deny from the aliases.
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
