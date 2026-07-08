package main_test

// Guard test for ADR-0003 (a blocked gate is a HITL pause, never a crash).
// Exit code 3 is a reserved, load-bearing contract the harness and hooks branch
// on: an unmet gate exits 3 with an actionable "blocked" line, and a satisfied
// gate exits 0. This test names the contract against its ADR so a change to the
// reserved code fails with the decision attached. (Broader gate behavior lives
// in gate_test.go; this pins the exit-code contract itself.)

import (
	"strings"
	"testing"
)

func TestADR0003BlockedGateExitsThreeNotCrash(t *testing.T) {
	root := newWorkspace(t)

	// auth-tokens is in the build phase with tasks empty -> readiness blocks.
	out, _, code := runDevrites(t, root, "readiness", "auth-tokens")
	if code != 3 {
		t.Fatalf("blocked readiness exit = %d, want the reserved 3 (HITL pause)", code)
	}
	if !strings.Contains(out, "blocked") {
		t.Errorf("a blocked gate must state it is blocked on stdout\n%s", out)
	}

	// A satisfied gate is a clean pass, never a spurious non-zero.
	_, _, ok := runDevrites(t, root, "readiness", "search-ranking") // spec phase, spec present
	if ok != 0 {
		t.Fatalf("satisfied readiness exit = %d, want 0", ok)
	}
}
