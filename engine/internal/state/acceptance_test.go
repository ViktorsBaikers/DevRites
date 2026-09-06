package state

import (
	"slices"
	"testing"
)

func TestParseAcceptanceMapEmptySpecPasses(t *testing.T) {
	got := ParseAcceptanceMap([]byte("# Spec\n\nReady.\n"), []byte("tasks"), []byte("plan"), true, true)
	if len(got.SpecIDs) != 0 || len(got.Problems) != 0 {
		t.Fatalf("ids=%v problems=%v", got.SpecIDs, got.Problems)
	}
}

func TestParseAcceptanceMapRequiresTasksAndTestPlan(t *testing.T) {
	spec := []byte("# Spec\n\n## Acceptance criteria\n- [ ] AC-001: export CSV\n- [ ] AC-002: reject cross-user\n")
	tasks := []byte("# Tasks\nSatisfies: AC-001\n")
	plan := []byte("# Test plan\nAC-001 → T1\n")
	got := ParseAcceptanceMap(spec, tasks, plan, true, true)
	if !slices.Equal(got.SpecIDs, []string{"AC-001", "AC-002"}) {
		t.Fatalf("ids=%v", got.SpecIDs)
	}
	if !slices.Equal(got.Problems, []string{
		"acceptance AC-002 is not referenced in tasks.md",
		"acceptance AC-002 is not referenced in test-plan.md",
	}) {
		t.Fatalf("problems=%v", got.Problems)
	}
}

func TestParseAcceptanceMapSkipsUnrequiredArtifacts(t *testing.T) {
	spec := []byte("## Acceptance Criteria\n- AC-001: bind readiness inputs.\n")
	got := ParseAcceptanceMap(spec, nil, nil, false, false)
	if !slices.Equal(got.SpecIDs, []string{"AC-001"}) || len(got.Problems) != 0 {
		t.Fatalf("ids=%v problems=%v", got.SpecIDs, got.Problems)
	}
}

func TestParseAcceptanceMapIgnoresNoncanonicalAndDuplicates(t *testing.T) {
	spec := []byte("## Acceptance criteria\n- AC-001: one\n- AC1: ignored\n- AC-001: again\n")
	tasks := []byte("Satisfies: AC-001\n")
	plan := []byte("AC-001\n")
	got := ParseAcceptanceMap(spec, tasks, plan, true, true)
	if !slices.Equal(got.SpecIDs, []string{"AC-001"}) || len(got.Problems) != 0 {
		t.Fatalf("ids=%v problems=%v", got.SpecIDs, got.Problems)
	}
}

func TestParseAcceptanceMapStopsAtNextH2(t *testing.T) {
	spec := []byte("## Acceptance criteria\n- AC-001: in section\n\n## Non-goals\n- AC-999: outside\n")
	tasks := []byte("AC-001\n")
	plan := []byte("AC-001\n")
	got := ParseAcceptanceMap(spec, tasks, plan, true, true)
	if !slices.Equal(got.SpecIDs, []string{"AC-001"}) || len(got.Problems) != 0 {
		t.Fatalf("ids=%v problems=%v", got.SpecIDs, got.Problems)
	}
}
