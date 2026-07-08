package main

import "testing"

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
