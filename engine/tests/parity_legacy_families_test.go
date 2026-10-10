package main_test

import (
	"path/filepath"
	"testing"

	"github.com/devrites/devrites/internal/state"
)

// TestStatusLiveWorkspace renders the completeness report of a canonical
// work/<slug> feature and compares it to its golden.
func TestStatusLiveWorkspace(t *testing.T) {
	work := t.TempDir()
	writeFile(t, work, ".devrites/work/live/state.md", "- Phase: build\n- Status: running\n")
	for name, body := range map[string]string{
		"brief.md":             "# Brief\n\nDo the thing.\n",
		"spec.md":              "# Spec\n\nDo the thing.\n",
		"decisions.md":         "# Decisions\n\nChose X.\n",
		"assumptions.md":       "# Assumptions\n\nNone.\n",
		"questions.md":         "# Questions\n\nNone.\n",
		"decision-coverage.md": "# Decision coverage\n\nCLEAR.\n",
		"architecture.md":      "# Architecture\n\nExisting layer.\n",
		"plan.md":              "# Plan\n\nApproach.\n",
		"tasks.md":             "# Tasks\n\n- [x] slice 1\n",
		"traceability.md":      "# Traceability\n\nMapped.\n",
		"eng-review.md":        "# Engineering review\n\nREADY.\n",
		"test-plan.md":         "# Test plan\n\nRun tests.\n",
		"gates.md":             "# Gates\n\nGreen.\n",
	} {
		writeFile(t, work, ".devrites/work/live/"+name, body)
	}
	report, err := state.Status(filepath.Join(work, ".devrites"), "live")
	if err != nil {
		t.Fatal(err)
	}
	assertGoldenKey(t, "TestStatusLiveWorkspace", "exit 0\n"+report.Render())
}
