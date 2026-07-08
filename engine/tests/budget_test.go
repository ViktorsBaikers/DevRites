package main_test

import (
	"strings"
	"testing"
)

// TestStatusLiveWorkspace is the P2 acceptance check: `devrites status <slug>`
// must report a canonical work/<slug> feature the live pack created without a
// feature.md manifest — phase from the state.md ledger, proof/status via their
// evidence.md/state.md aliases. Before the schema unification this returned
// "feature not found".
func TestStatusLiveWorkspace(t *testing.T) {
	work := t.TempDir()
	writeFile(t, work, ".devrites/work/live/state.md", "- Phase: build\n- Status: running\n")
	writeFile(t, work, ".devrites/work/live/spec.md", "# Spec\n\nDo the thing.\n")
	writeFile(t, work, ".devrites/work/live/plan.md", "# Plan\n\nApproach.\n")
	writeFile(t, work, ".devrites/work/live/decisions.md", "# Decisions\n\nChose X.\n")
	writeFile(t, work, ".devrites/work/live/tasks.md", "# Tasks\n\n- [x] slice 1\n")
	(parityCase{
		workdir: work,
		env:     libRootEnv(work),
		goArgs:  []string{"status", "live"},
	}).assertEqual(t)
}

// TestParityBudget checks `budget` against golden snapshots: the within-budget
// report, an over-budget advisory (exit 0) and its --strict block (exit 3), the
// active-slug default, a stray *.md measured at the default ceiling, and the two
// usage errors (missing feature, no active feature).
func TestParityBudget(t *testing.T) {
	nLines := func(n int) string { return strings.Repeat("x\n", n) }
	work := t.TempDir()

	// within: every file under its ceiling.
	writeFeatureFile(t, work, "small", "spec.md", nLines(10))
	writeFeatureFile(t, work, "small", "state.md", nLines(5))
	writeFeatureFile(t, work, "small", "tasks.md", nLines(5))

	// over: spec.md and tasks.md exceed their ceilings; state.md is fine.
	writeFeatureFile(t, work, "big", "spec.md", nLines(300))
	writeFeatureFile(t, work, "big", "tasks.md", nLines(160))
	writeFeatureFile(t, work, "big", "state.md", nLines(5))

	// extra: an unknown notes.md is measured at the default ceiling.
	writeFeatureFile(t, work, "extra", "spec.md", nLines(5))
	writeFeatureFile(t, work, "extra", "notes.md", nLines(210))

	// active: default slug comes from ACTIVE.
	writeFeatureFile(t, work, "act", "spec.md", nLines(5))

	// ghost is never created (missing-feature error path).

	for _, tc := range []struct {
		name   string
		active string // ACTIVE pointer to set ("" removes it)
		args   []string
	}{
		{"within", "", []string{"budget", "small"}},
		{"over-advisory", "", []string{"budget", "big"}},
		{"over-strict", "", []string{"budget", "big", "--strict"}},
		{"extra-md", "", []string{"budget", "extra"}},
		{"default-active", "act", []string{"budget"}},
		{"no-feature", "", []string{"budget", "ghost"}},
		{"no-active", "", []string{"budget"}},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			setActiveAFK(t, work, tc.active, false)
			(parityCase{
				workdir: work,
				env:     libRootEnv(work),
				goArgs:  tc.args,
			}).assertEqual(t)
		})
	}
}
