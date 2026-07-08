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

// phaseArc is the canonical order the lifecycle advances through (ADR-0004).
var phaseArc = []state.Phase{
	state.PhaseFrame,
	state.PhaseSpec,
	state.PhasePlan,
	state.PhaseBuild,
	state.PhaseProve,
	state.PhaseVet,
	state.PhaseSeal,
	state.PhaseShip,
}

func TestADR0004RequiredSectionsAreAdditiveDownTheArc(t *testing.T) {
	for _, p := range phaseArc {
		if !state.KnownPhase(p) {
			t.Fatalf("phase %q in the arc is not a known phase", p)
		}
	}

	// Each phase's required set must contain everything the previous phase
	// required — completeness only accumulates, never regresses.
	for i := 1; i < len(phaseArc); i++ {
		prev := asSet(state.RequiredSections(phaseArc[i-1]))
		cur := asSet(state.RequiredSections(phaseArc[i]))
		for s := range prev {
			if !cur[s] {
				t.Errorf("phase %q dropped section %q that %q required — arc must be additive",
					phaseArc[i], s, phaseArc[i-1])
			}
		}
	}

	// Anchors: framing requires nothing; seal requires the full section set.
	if got := state.RequiredSections(state.PhaseFrame); len(got) != 0 {
		t.Errorf("frame requires %v, want none", got)
	}
	if got, want := len(state.RequiredSections(state.PhaseSeal)), len(state.Sections); got != want {
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
