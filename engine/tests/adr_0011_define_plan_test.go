package main_test

import (
	"testing"

	"github.com/devrites/devrites/internal/state"
)

func TestADR0011DefineAndPlanHaveDistinctTransitionRights(t *testing.T) {
	if got := state.ResumeVerb(state.PhaseDefine); got != "define" {
		t.Fatalf("define resumes %q, want define", got)
	}
	if got := state.ResumeVerb(state.PhasePlan); got != "vet" {
		t.Fatalf("plan resumes %q, want vet", got)
	}
	if got, ok := state.PhaseForName("planning"); !ok || got != state.PhasePlan {
		t.Fatalf("planning alias=(%q,%v), want plan compatibility", got, ok)
	}

	rights := map[string]state.Phase{}
	for _, phase := range state.WorkflowPhases() {
		if phase.TransitionRight == "" {
			t.Fatalf("phase %q has no transition right", phase.ID)
		}
		if prior := rights[phase.TransitionRight]; prior != "" {
			t.Fatalf("phases %q and %q share transition right %q", prior, phase.ID, phase.TransitionRight)
		}
		rights[phase.TransitionRight] = phase.ID
	}
}
