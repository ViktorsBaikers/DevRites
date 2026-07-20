package workflow

import "testing"

func TestForActionExtractsCanonicalInvocation(t *testing.T) {
	for _, tc := range []struct {
		action string
		verb   string
	}{
		{action: "/rite-define after readiness passes", verb: "define"},
		{action: "resume with $rite-build", verb: "build"},
		{action: "none"},
	} {
		if got := ForAction(tc.action); got.Verb != tc.verb {
			t.Fatalf("ForAction(%q).Verb=%q, want %q", tc.action, got.Verb, tc.verb)
		}
	}
}

func TestForPhaseCoversCurrentLifecycle(t *testing.T) {
	for _, phase := range []string{
		"frame", "spec", "temper", "define", "vet", "build", "converge",
		"prove", "polish", "review", "seal", "ship",
	} {
		if got := ForPhase(phase).Verb; got != phase {
			t.Fatalf("ForPhase(%q).Verb=%q, want %q", phase, got, phase)
		}
	}
	if got := ForPhase("plan").Verb; got != "define" {
		t.Fatalf("ForPhase(plan).Verb=%q, want define compatibility route", got)
	}
	if got := ForPhase("done").Verb; got != "" {
		t.Fatalf("ForPhase(done).Verb=%q, want empty", got)
	}
}
