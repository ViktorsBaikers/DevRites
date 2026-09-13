package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/state"
)

const budgetTasks = `# Tasks

Current checkpoint: intro text.

## SLICE-001 First
Goal: one
Dependencies: none
depends_on: []

### Notes
nested heading stays inside the slice

## SLICE-002 Second
Goal: two
Dependencies: SLICE-001
depends_on: [SLICE-001]

## Trailing section
not a slice
`

func writeBudgetWorkspace(t *testing.T, files map[string]string) (root, workspace string) {
	t.Helper()
	root = filepath.Join(t.TempDir(), ".devrites")
	workspace = filepath.Join(root, "work", "feature")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(workspace, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root, workspace
}

func TestExtractTaskSliceStopsAtNextH2AndKeepsH3(t *testing.T) {
	block, ok := state.ExtractTaskSlice([]byte(budgetTasks), "SLICE-001")
	if !ok {
		t.Fatal("expected SLICE-001")
	}
	if !strings.HasPrefix(block, "## SLICE-001 First\n") || !strings.Contains(block, "### Notes") {
		t.Fatalf("block=%q", block)
	}
	if strings.Contains(block, "SLICE-002") || strings.Contains(block, "Trailing") {
		t.Fatalf("block leaked past the next heading: %q", block)
	}
	last, ok := state.ExtractTaskSlice([]byte(budgetTasks), "SLICE-002")
	if !ok || strings.Contains(last, "Trailing section") {
		t.Fatalf("last=%q ok=%v", last, ok)
	}
	if _, ok := state.ExtractTaskSlice([]byte(budgetTasks), "SLICE-003"); ok {
		t.Fatal("SLICE-003 must not resolve")
	}
}

func TestRunObserveSlicePrintsOneSection(t *testing.T) {
	root, _ := writeBudgetWorkspace(t, map[string]string{"tasks.md": budgetTasks})
	var stdout, stderr bytes.Buffer
	if code := RunObserveSlice(root, "feature", "SLICE-002", &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if got := stdout.String(); !strings.HasPrefix(got, "## SLICE-002 Second\n") || strings.Contains(got, "SLICE-001 First") {
		t.Fatalf("stdout=%q", got)
	}

	stdout.Reset()
	if code := RunObserveSlice(root, "feature", "SLICE-009", &stdout, &stderr); code != 3 || !strings.Contains(stdout.String(), "BLOCKED") {
		t.Fatalf("code=%d stdout=%q", code, stdout.String())
	}
	if code := RunObserveSlice(root, "feature", "slice-1", &stdout, &stderr); code != 2 {
		t.Fatalf("code=%d for malformed id", code)
	}
}

func TestObserveSummaryReportsBudgetsAndBulkFiles(t *testing.T) {
	longLine := strings.Repeat("x", 60_000) + "\n"
	root, _ := writeBudgetWorkspace(t, map[string]string{
		"state.md":                "| phase | build |\n| schema | 3 |\n" + longLine,
		"tasks.md":                budgetTasks,
		"vet-review-input-1.json": strings.Repeat("{}", 40_000),
		"small-note.json":         "{}",
	})
	summary, err := ObserveSummaryFor(root, "feature")
	if err != nil {
		t.Fatal(err)
	}
	byFile := map[string]ArtifactBudget{}
	for _, b := range summary.ArtifactBudgets {
		byFile[b.File] = b
	}
	if st, ok := byFile["state.md"]; !ok || !st.Over || st.Lines != 3 || st.ByteBudget != 120*state.BudgetBytesPerLine {
		t.Fatalf("state.md budget=%+v ok=%v (long line must trip the byte budget)", st, ok)
	}
	if tk, ok := byFile["tasks.md"]; !ok || tk.Over {
		t.Fatalf("tasks.md budget=%+v ok=%v", tk, ok)
	}
	if summary.TaskGraph == nil || len(summary.TaskGraph.Slices) != 2 ||
		summary.TaskGraph.Slices[0].ID != "SLICE-001" || len(summary.TaskGraph.Slices[0].DependsOn) != 0 ||
		summary.TaskGraph.Slices[1].DependsOn[0] != "SLICE-001" {
		t.Fatalf("task_graph.slices=%+v", summary.TaskGraph)
	}
	if len(summary.BulkFiles) != 1 || summary.BulkFiles[0].File != "vet-review-input-1.json" {
		t.Fatalf("bulk_files=%+v", summary.BulkFiles)
	}
}

func TestObserveSummaryReportsSequenceCursor(t *testing.T) {
	root, _ := writeBudgetWorkspace(t, map[string]string{
		"state.md": "| phase | build |\n| schema | 3 |\n| sequence_parent | feat-1 |\n| sequence_position | 2 |\n| sequence_workspaces_remaining | 3 |\n",
	})
	summary, err := ObserveSummaryFor(root, "feature")
	if err != nil {
		t.Fatal(err)
	}
	seq := summary.Sequence
	if seq == nil || seq.Parent != "feat-1" || seq.Position != "2" || seq.WorkspacesRemaining != "3" {
		t.Fatalf("sequence=%+v", seq)
	}
	root, _ = writeBudgetWorkspace(t, map[string]string{
		"state.md": "| phase | build |\n| schema | 3 |\n",
	})
	summary, err = ObserveSummaryFor(root, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Sequence != nil {
		t.Fatalf("sequence must be absent without cursor fields: %+v", summary.Sequence)
	}
}
