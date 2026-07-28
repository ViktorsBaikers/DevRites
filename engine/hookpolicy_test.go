package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestHookActiveProfileTiers(t *testing.T) {
	cases := []struct {
		profile string
		hook    string
		want    bool
	}{
		{"", "orient", true},        // default=standard, minimal hook on
		{"", "stop-gate", true},     // default=standard, standard hook on
		{"minimal", "orient", true}, // minimal keeps orientation
		{"minimal", "stop-gate", false},
		{"minimal", "a1-guard", false},
		{"minimal", "redwatch", false},
		{"standard", "stop-gate", true},
		{"strict", "stop-gate", true},
		{"strict", "orient", true},
		{"MiNiMaL", "stop-gate", false},      // case-insensitive
		{"bogus", "stop-gate", true},         // unknown profile → standard
		{"minimal", "not-a-real-hook", true}, // unknown hook never disabled by profile
	}
	for _, c := range cases {
		t.Setenv("DEVRITES_HOOK_PROFILE", c.profile)
		t.Setenv("DEVRITES_DISABLED_HOOKS", "")
		if got := hookActive(c.hook); got != c.want {
			t.Errorf("profile=%q hook=%q: hookActive=%v want %v", c.profile, c.hook, got, c.want)
		}
	}
}

func TestHookDisabledKillList(t *testing.T) {
	t.Setenv("DEVRITES_HOOK_PROFILE", "standard")
	t.Setenv("DEVRITES_DISABLED_HOOKS", "stop-gate, redwatch ")
	if hookActive("stop-gate") {
		t.Error("stop-gate in kill list should be inactive")
	}
	if hookActive("redwatch") {
		t.Error("redwatch in kill list (whitespace-padded) should be inactive")
	}
	if !hookActive("orient") {
		t.Error("orient not in kill list should stay active")
	}
}

func TestHookEnforce(t *testing.T) {
	// Specific var wins regardless of profile.
	t.Setenv("DEVRITES_HOOK_PROFILE", "standard")
	t.Setenv("DEVRITES_STOP_GATE", "enforce")
	if !hookEnforce("DEVRITES_STOP_GATE") {
		t.Error("explicit enforce var should enforce")
	}
	// Observe by default under standard.
	t.Setenv("DEVRITES_STOP_GATE", "")
	if hookEnforce("DEVRITES_STOP_GATE") {
		t.Error("standard profile without var should observe")
	}
	// strict flips every guard on.
	t.Setenv("DEVRITES_HOOK_PROFILE", "strict")
	if !hookEnforce("DEVRITES_STOP_GATE") {
		t.Error("strict profile should enforce every guard")
	}
}

func TestBlockingHookExitAudit(t *testing.T) {
	type exitContract struct {
		reentry   string
		malformed string
		bound     string
		killPath  string
	}
	audit := map[string]exitContract{
		"stop-gate": {
			reentry:   "explicit stop_hook_active=true lets the re-entered Stop complete",
			malformed: "empty, malformed, missing, null, or non-boolean stop_hook_active fails open",
			bound:     "one block before the host re-enters with stop_hook_active=true",
			killPath:  "DEVRITES_DISABLED_HOOKS=stop-gate or DEVRITES_HOOK_PROFILE=minimal",
		},
		"agent-dispatch": {
			reentry:   "explicit stop_hook_active=true terminates the re-entered Stop with the same failure",
			malformed: "missing or malformed event, session, or turn identity fails open",
			bound:     "one block before the host re-enters with stop_hook_active=true",
			killPath:  "DEVRITES_DISABLED_HOOKS=agent-dispatch or DEVRITES_HOOK_PROFILE=minimal",
		},
		"reviewer-readonly": {
			reentry:   "not applicable: each PreToolUse decision applies to one tool request",
			malformed: "missing or malformed tool identity fails open",
			bound:     "one synchronous deny; no retry or subprocess",
			killPath:  "DEVRITES_DISABLED_HOOKS=reviewer-readonly or DEVRITES_HOOK_PROFILE=minimal",
		},
		"a1-guard": {
			reentry:   "not applicable: each PreToolUse decision applies to one tool request",
			malformed: "missing or malformed tool identity fails open",
			bound:     "one synchronous deny; no retry or subprocess",
			killPath:  "DEVRITES_DISABLED_HOOKS=a1-guard or DEVRITES_HOOK_PROFILE=minimal",
		},
		"wright-scope": {
			reentry:   "not applicable: each PreToolUse decision applies to one tool request",
			malformed: "missing or malformed tool identity fails open",
			bound:     "one synchronous deny; no retry or subprocess",
			killPath:  "DEVRITES_DISABLED_HOOKS=wright-scope or DEVRITES_HOOK_PROFILE=minimal",
		},
		"git-guard": {
			reentry:   "not applicable: each PreToolUse decision applies to one tool request",
			malformed: "missing or malformed structured tool input fails open",
			bound:     "one synchronous deny or one consumed one-shot allow; no retry or subprocess",
			killPath:  "DEVRITES_DISABLED_HOOKS=git-guard or DEVRITES_HOOK_PROFILE=minimal",
		},
	}

	for name, def := range hookRegistry {
		if !def.canBlock {
			continue
		}
		contract, ok := audit[name]
		if !ok {
			t.Errorf("production blocker %q has no exit audit row", name)
			continue
		}
		if contract.reentry == "" || contract.malformed == "" || contract.bound == "" || contract.killPath == "" {
			t.Errorf("blocker %q has an incomplete exit audit row: %#v", name, contract)
		}
	}
	for name := range audit {
		def, ok := hookRegistry[name]
		if !ok || !def.canBlock {
			t.Errorf("exit audit row %q is not a production blocker", name)
		}
	}

	for name := range audit {
		t.Run(name, func(t *testing.T) {
			t.Setenv("DEVRITES_HOOK_PROFILE", "standard")
			t.Setenv("DEVRITES_DISABLED_HOOKS", name)
			if hookActive(name) {
				t.Fatal("per-hook kill list did not disable blocker")
			}

			t.Setenv("DEVRITES_DISABLED_HOOKS", "")
			t.Setenv("DEVRITES_HOOK_PROFILE", "minimal")
			if hookActive(name) {
				t.Fatal("minimal profile did not disable blocker")
			}

			t.Setenv("DEVRITES_HOOK_PROFILE", "standard")
			var stdout, stderr bytes.Buffer
			if code := run([]string{"hook", name, "--harness=claude"}, strings.NewReader("not-json"), &stdout, &stderr); code != exitOK {
				t.Fatalf("malformed input exit = %d, want 0 (stderr %q)", code, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("malformed input emitted a blocking decision: %q", stdout.String())
			}
		})
	}
}
