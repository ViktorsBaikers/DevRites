package main_test

import (
	"testing"

	"github.com/devrites/devrites/internal/state"
)

func TestADR0011DefineAndPlanHaveDistinctTransitionRights(t *testing.T) {
	define, ok := state.PolicyFor(state.PhaseDefine)
	if !ok || define.ResumeVerb != "define" {
		t.Fatalf("define policy=(%+v,%v), want resume verb define", define, ok)
	}
	plan, ok := state.PolicyFor(state.PhasePlan)
	if !ok || plan.ResumeVerb != "vet" {
		t.Fatalf("plan policy=(%+v,%v), want resume verb vet", plan, ok)
	}
	if policy, ok := state.PolicyFor(state.Phase("planning")); ok {
		t.Fatalf("planning alias=(%+v,true), want speculative alias rejected", policy)
	}

	rights := map[string]state.Phase{}
	for _, policy := range state.PhasePolicies() {
		if policy.TransitionRight == "" {
			t.Fatalf("phase %q has no transition right", policy.Target)
		}
		if prior := rights[policy.TransitionRight]; prior != "" {
			t.Fatalf("phases %q and %q share transition right %q", prior, policy.Target, policy.TransitionRight)
		}
		rights[policy.TransitionRight] = policy.Target
	}
}
