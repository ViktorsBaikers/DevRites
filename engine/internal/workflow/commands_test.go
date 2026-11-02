package workflow

import "testing"

func TestForActionExtractsCanonicalInvocation(t *testing.T) {
	for _, tc := range []struct {
		action string
		verb   string
	}{
		{action: "/rite-define after readiness passes", verb: "define"},
		{action: "resume with $rite-build", verb: "build"},
		{action: "/skill:rite-vet alpha", verb: "vet"},
		{action: "none"},
	} {
		if got := ForAction(tc.action); got.Verb != tc.verb {
			t.Fatalf("ForAction(%q).Verb=%q, want %q", tc.action, got.Verb, tc.verb)
		}
	}
}

func TestSourceInvocationPreservesNativeArgumentContracts(t *testing.T) {
	for _, verb := range []string{"build", "plan", "polish", "review"} {
		if got := ForVerb(verb).SourceInvocation("alpha"); got != "/rite-"+verb {
			t.Errorf("%s: source became phase-local argument: %q", verb, got)
		}
	}
	for _, name := range []string{"vet", "/rite-vet", "$rite-vet", "/skill:rite-vet"} {
		command := ForVerb(name)
		if got := command.SourceInvocation("alpha"); got != "/rite-vet alpha" || command.Skill != "/skill:rite-vet" {
			t.Errorf("%s: command=%+v invocation=%q", name, command, got)
		}
	}
}
