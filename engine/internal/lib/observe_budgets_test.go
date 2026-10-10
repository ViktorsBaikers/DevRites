package lib

import (
	"bytes"

	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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
		"state.md":                "| phase | build |\n| schema | 4 |\n" + longLine,
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
		"state.md": "| phase | build |\n| schema | 4 |\n| sequence_parent | feat-1 |\n| sequence_position | 2 |\n| sequence_workspaces_remaining | 3 |\n",
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
		"state.md": "| phase | build |\n| schema | 4 |\n",
	})
	summary, err = ObserveSummaryFor(root, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Sequence != nil {
		t.Fatalf("sequence must be absent without cursor fields: %+v", summary.Sequence)
	}
}

// summaryUnreadable decodes the emitted summary JSON and fails when the consumer has
// no way to tell an omitted section from an empty one.
func summaryUnreadable(t *testing.T, out []byte) []string {
	t.Helper()
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("summary is not a JSON object: %v (%s)", err, out)
	}
	raw, ok := doc["unreadable"]
	if !ok {
		t.Fatalf("summary carries no \"unreadable\" key, so an artifact that could not be read is indistinguishable from one that does not exist: %s", out)
	}
	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		t.Fatalf("unreadable is not a list of artifact names: %v (%s)", err, raw)
	}
	return names
}

func summaryFiles(t *testing.T, out []byte, key string) []string {
	t.Helper()
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("summary is not a JSON object: %v (%s)", err, out)
	}
	var rows []struct {
		File string `json:"file"`
	}
	raw, ok := doc[key]
	if !ok {
		t.Fatalf("summary key %q is absent entirely: %s", key, out)
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("%s is not a list: %v (%s)", key, err, raw)
	}
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.File)
	}
	return names
}

// One canonical artifact that exists but cannot be read must not take the
// task graph, the artifact budgets and the bulk-file list down with it.
func TestObserveSummaryNamesUnreadableArtifactAndKeepsReadableBudgets(t *testing.T) {
	if os.Geteuid() == 0 || runtime.GOOS == "windows" {
		t.Skip("file permissions do not bind root or windows")
	}
	root, workspace := writeBudgetWorkspace(t, map[string]string{
		"state.md":                "| phase | build |\n| schema | 4 |\n",
		"tasks.md":                budgetTasks,
		"vet-review-input-1.json": strings.Repeat("{}", 40_000),
	})
	// spec.md is canonical and sorts first; the entries behind it must survive.
	spec := filepath.Join(workspace, "spec.md")
	if err := os.WriteFile(spec, []byte("# Spec\n"), 0o000); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(spec, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(spec, 0o644) })

	var out, errOut bytes.Buffer
	if code := RunObserveSummary(root, "feature", &out, &errOut); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
	names := summaryUnreadable(t, out.Bytes())
	if len(names) != 1 || names[0] != "spec.md" {
		t.Fatalf("unreadable=%v, want exactly [spec.md]", names)
	}
	budgets := summaryFiles(t, out.Bytes(), "artifact_budgets")
	for _, want := range []string{"state.md", "tasks.md"} {
		found := false
		for _, got := range budgets {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("artifact_budgets=%v lost %q to the unreadable spec.md: %s", budgets, want, out.String())
		}
	}
	bulk := summaryFiles(t, out.Bytes(), "bulk_files")
	if len(bulk) != 1 || bulk[0] != "vet-review-input-1.json" {
		t.Fatalf("bulk_files=%v lost the readable 80 KiB file to the unreadable spec.md: %s", bulk, out.String())
	}
}

// A tasks.md that is not a readable regular file must be named, not dropped.
func TestObserveSummaryNamesUnreadableTasksInsteadOfOmittingTaskGraph(t *testing.T) {
	root, workspace := writeBudgetWorkspace(t, map[string]string{
		"state.md": "| phase | build |\n| schema | 4 |\n",
		"tasks.md": budgetTasks,
	})
	if err := os.Remove(filepath.Join(workspace, "tasks.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(workspace, "tasks.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if code := RunObserveSummary(root, "feature", &out, &errOut); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
	names := summaryUnreadable(t, out.Bytes())
	if len(names) != 1 || names[0] != "tasks.md" {
		t.Fatalf("unreadable=%v, want exactly [tasks.md]", names)
	}
	if _, ok := map[string]json.RawMessage(mustDecode(t, out.Bytes()))["task_graph"]; ok {
		t.Fatalf("task_graph must stay absent when tasks.md cannot be read: %s", out.String())
	}
	if budgets := summaryFiles(t, out.Bytes(), "artifact_budgets"); len(budgets) != 1 || budgets[0] != "state.md" {
		t.Fatalf("artifact_budgets=%v, want [state.md]", budgets)
	}
}

func mustDecode(t *testing.T, out []byte) map[string]json.RawMessage {
	t.Helper()
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("summary is not a JSON object: %v (%s)", err, out)
	}
	return doc
}

// A workspace that has not reached define has no tasks.md; that is absence, not a read failure.
func TestObserveSummaryOmitsUnreadableWhenTasksAbsent(t *testing.T) {
	root, _ := writeBudgetWorkspace(t, map[string]string{
		"state.md": "| phase | spec |\n| schema | 4 |\n",
	})
	var out, errOut bytes.Buffer
	if code := RunObserveSummary(root, "feature", &out, &errOut); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
	if _, ok := mustDecode(t, out.Bytes())["unreadable"]; ok {
		t.Fatalf("a missing tasks.md must not be reported as unreadable: %s", out.String())
	}
}

// A tasks.md that fails both the task-graph read and the budget read is named once.
func TestObserveSummaryNamesUnreadableTasksOnce(t *testing.T) {
	if os.Geteuid() == 0 || runtime.GOOS == "windows" {
		t.Skip("file permissions do not bind root or windows")
	}
	root, workspace := writeBudgetWorkspace(t, map[string]string{
		"state.md": "| phase | build |\n| schema | 4 |\n",
		"tasks.md": budgetTasks,
	})
	tasks := filepath.Join(workspace, "tasks.md")
	if err := os.Chmod(tasks, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(tasks, 0o644) })

	var out, errOut bytes.Buffer
	if code := RunObserveSummary(root, "feature", &out, &errOut); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
	if names := summaryUnreadable(t, out.Bytes()); len(names) != 1 || names[0] != "tasks.md" {
		t.Fatalf("unreadable=%v, want exactly [tasks.md]", names)
	}
}

const (
	artifactSchemaDocPath  = "../../../pack/.claude/skills/devrites-lib/reference/workspace-artifact-schema.md"
	outlineTemplateDocPath = "../../../pack/.claude/skills/devrites-lib/reference/visual-playbooks/outline-template.md"
)

// Every artifact the schema's Budget column names must either be measured by the
// engine or say in its Budget cell that the budget is advisory.
func TestSchemaBudgetRowsAreEnforcedOrMarkedAdvisory(t *testing.T) {
	raw, err := os.ReadFile(artifactSchemaDocPath)
	if err != nil {
		t.Fatal(err)
	}
	rows := 0
	for _, line := range strings.Split(string(raw), "\n") {
		cells := strings.Split(line, "|")
		if len(cells) != 5 || !strings.Contains(cells[3], "lines") {
			continue
		}
		first, budget := strings.TrimSpace(cells[1]), cells[3]
		if !strings.HasPrefix(first, "`") || !strings.HasSuffix(first, "`") {
			continue
		}
		rows++
		name := strings.Trim(first, "`")
		if _, canonical := state.ArtifactLineBudget(name); canonical {
			continue
		}
		if !strings.Contains(strings.ToLower(budget), "advisory") {
			t.Errorf("%s has a budget %q but is neither in state.artifactLineBudgets nor marked advisory", name, strings.TrimSpace(budget))
		}
	}
	if rows < 30 {
		t.Fatalf("parsed %d budget rows from the schema table, want at least 30", rows)
	}
}

func TestOutlineTemplateBudgetIsMarkedAdvisory(t *testing.T) {
	raw, err := os.ReadFile(outlineTemplateDocPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "- Budget:") {
			if !strings.Contains(strings.ToLower(line), "advisory") {
				t.Fatalf("outline budget line %q does not say the budget is advisory", line)
			}
			return
		}
	}
	t.Fatal("outline template has no \"- Budget:\" line")
}

// A canonical artifact larger than the read cap is reported over budget from its
// size alone, without being read whole.
func TestObserveArtifactBudgetsOversizedArtifactIsOverWithoutFullRead(t *testing.T) {
	const size = 32 << 20
	_, workspace := writeBudgetWorkspace(t, map[string]string{"state.md": "| phase | build |\n"})
	file, err := os.OpenFile(filepath.Join(workspace, "state.md"), os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(size); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	budgets, _, unreadable, err := observeArtifactBudgets(workspace)
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatal(err)
	}
	if len(unreadable) != 0 {
		t.Fatalf("unreadable=%v, an oversized artifact is over budget, not unreadable", unreadable)
	}
	if len(budgets) != 1 || !budgets[0].Over || budgets[0].Bytes != size {
		t.Fatalf("budgets=%+v, want state.md over budget with Bytes=%d", budgets, size)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > size/4 {
		t.Fatalf("allocated %d bytes for a %d byte artifact; the read must be bounded", allocated, size)
	}
}
