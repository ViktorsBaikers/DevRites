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
