package main_test

// Guard test for ADR-0004 (phase-relative section completeness). Locks the
// invariant that required sections are ADDITIVE down the rite-* arc: every phase
// requires a superset of the phase before it. If someone reorders the arc or
// drops a section from a later phase's requirements, this fails loudly with the
// ADR number attached.

import (
	"testing"

	"github.com/devrites/devrites/internal/state"
)

func TestADR0004RequiredSectionsAreAdditiveDownTheArc(t *testing.T) {
	policies := state.PhasePolicies()
	for _, policy := range policies {
		lookedUp, ok := state.PolicyFor(policy.Target)
		if !ok || lookedUp.Target != policy.Target {
			t.Fatalf("phase %q in the arc is not a known phase", policy.Target)
		}
	}

	// Each phase's required set must contain everything the previous phase
	// required: completeness only accumulates, never regresses.
	for i := 1; i < len(policies); i++ {
		previous := asSet(policies[i-1].RequiredSections)
		current := asSet(policies[i].RequiredSections)
		for section := range previous {
			if !current[section] {
				t.Errorf("phase %q dropped section %q that %q required: arc must be additive",
					policies[i].Target, section, policies[i-1].Target)
			}
		}
	}

	// Anchors: framing requires nothing; seal requires the full section set.
	frame, _ := state.PolicyFor(state.PhaseFrame)
	if len(frame.RequiredSections) != 0 {
		t.Errorf("frame requires %v, want none", frame.RequiredSections)
	}
	seal, _ := state.PolicyFor(state.PhaseSeal)
	if got, want := len(seal.RequiredSections), len(state.Sections); got != want {
		t.Errorf("seal requires %d sections, want the full set of %d", got, want)
	}
}

func asSet(ss []state.Section) map[state.Section]bool {
	m := make(map[state.Section]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}
