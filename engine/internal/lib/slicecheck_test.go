package lib

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func sliceWorkspace(t *testing.T, tasks, spec string) (root string) {
	t.Helper()
	root, featureDir := newWorkspace(t, "build")
	writeFile(t, filepath.Join(featureDir, "tasks.md"), tasks)
	if spec != "" {
		writeFile(t, filepath.Join(featureDir, "spec.md"), spec)
	}
	return root
}

const goodSliceTasks = `# Tasks

## SLICE-001 Wire the thing
Goal: do the thing.
Satisfies: AC-001, AC-002
Acceptance criteria: thing works
Dependencies: none
depends_on: []
Writer allowlist: src/a.go; src/a_test.go
Files likely touched: src/a.go; src/a_test.go
Tests/proof: run go test ./...
Status: pending
`

const sliceSpec = `# Spec

- [ ] AC-001: thing works.
- [ ] AC-002: thing is verified.
`

func TestCheckSlicePassesValidContract(t *testing.T) {
	root := sliceWorkspace(t, goodSliceTasks, sliceSpec)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunCheckSlice(root, []string{"feat", "SLICE-001"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("code=%d want 0\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout.String(), "paths=2") || !strings.Contains(stdout.String(), "acs=2") {
		t.Fatalf("unexpected output:\n%s", stdout)
	}
}

func TestCheckSliceBlocksContractDefects(t *testing.T) {
	tasks := `# Tasks

## SLICE-001 Bad contract
Satisfies: AC-099
Writer allowlist: ../escape.go; src/*.go; .devrites/state.md; src/dup.go; src/dup.go
Files likely touched: src/not-listed.go
`
	root := sliceWorkspace(t, tasks, sliceSpec)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunCheckSlice(root, []string{"feat", "SLICE-001"}, stdout, stderr)
	if code != 3 {
		t.Fatalf("code=%d want 3\n%s%s", code, stdout, stderr)
	}
	out := stdout.String()
	for _, want := range []string{
		"Goal field missing",
		"escapes the project",
		"glob",
		".devrites",
		"duplicated",
		"outside the Writer allowlist",
		"AC-099",
		"Tests/proof field missing",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing problem %q:\n%s", want, out)
		}
	}
}

func TestCheckSliceMissingSliceAndArgs(t *testing.T) {
	root := sliceWorkspace(t, goodSliceTasks, sliceSpec)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunCheckSlice(root, []string{"feat", "SLICE-042"}, stdout, stderr); code != 3 {
		t.Fatalf("unknown slice code=%d want 3", code)
	}
	if code := RunCheckSlice(root, []string{"feat", "not-a-slice"}, stdout, stderr); code != 2 {
		t.Fatalf("bad id code=%d want 2", code)
	}
	if code := RunCheckSlice(root, []string{"feat"}, stdout, stderr); code != 2 {
		t.Fatalf("missing arg code=%d want 2", code)
	}
}

func TestDeferredSliceAfterEmptyFencedFieldIsNeverScheduled(t *testing.T) {
	tasks := poolSlice("SLICE-001", "src/deferred.go", "Size rationale:\n```text\nexample\n```\nSatisfies: none") +
		poolSlice("SLICE-002", "src/required.go", "")
	project := t.TempDir()
	poolWorkspace(t, filepath.Join(project, ".devrites"), "feat", "", tasks)
	ready, err := PlanReady(project, "feat")
	if err != nil || len(ready) != 1 || ready[0].ID != "SLICE-002" {
		t.Fatalf("empty optional field swallowed live deferral: ready=%v err=%v", ready, err)
	}
}

func TestCheckSliceRejectsSliceWithNoAC(t *testing.T) {
	tasks := `# Tasks

## SLICE-001 Chore
Goal: tidy.
Writer allowlist: src/a.go
Tests/proof: none needed
`
	root := sliceWorkspace(t, tasks, sliceSpec)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunCheckSlice(root, []string{"feat", "SLICE-001"}, stdout, stderr)
	if code != 3 || !strings.Contains(stdout.String(), "no AC-### coverage") {
		t.Fatalf("code=%d out:\n%s%s", code, stdout, stderr)
	}
}

func TestCheckAllowlistPathsNormalizesBeforeRules(t *testing.T) {
	if got := checkAllowlistPaths([]string{"./.devrites/state.md"}); len(got) != 1 || !strings.Contains(got[0], ".devrites") {
		t.Fatalf("./.devrites path not rejected as workflow state: %v", got)
	}
	if got := checkAllowlistPaths([]string{"src/a.go", "./src/a.go"}); len(got) != 1 || !strings.Contains(got[0], "duplicated") {
		t.Fatalf("./src/a.go not detected as duplicate of src/a.go: %v", got)
	}
}

func TestCheckSliceTouchedComparesNormalizedPaths(t *testing.T) {
	tasks := strings.Replace(goodSliceTasks, "Files likely touched: src/a.go; src/a_test.go", "Files likely touched: ./src/a.go; src/a_test.go", 1)
	root := sliceWorkspace(t, tasks, sliceSpec)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunCheckSlice(root, []string{"feat", "SLICE-001"}, stdout, stderr); code != 0 {
		t.Fatalf("code=%d want 0\n%s%s", code, stdout, stderr)
	}
}

func bigSliceTasks(rationale string) string {
	var paths []string
	for i := 1; i <= sliceFileAdvisory; i++ {
		paths = append(paths, "`src/f"+strings.Repeat("x", i)+".go`")
	}
	return "# Tasks\n\n## SLICE-001 Big\nGoal: do it.\nSatisfies: AC-001\ndepends_on: []\nWriter allowlist: " +
		strings.Join(paths, ", ") + "\nTests/proof: go test ./...\n" + rationale +
		"\n## SLICE-002 Small\nGoal: tiny.\nSatisfies: AC-001\ndepends_on: [SLICE-001]\nWriter allowlist: `src/a.go`\nTests/proof: go test ./...\n"
}

func TestSliceSizeAdvisoryNeverBlocks(t *testing.T) {
	root := sliceWorkspace(t, bigSliceTasks(""), "## Acceptance criteria\n- AC-001 works\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunCheckSlice(root, []string{"feat", "SLICE-001"}, stdout, stderr); code != 0 {
		t.Fatalf("advisory must not block: code=%d\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout.String(), "ADVISORY size: SLICE-001 has 12 Writer allowlist files") {
		t.Fatalf("advisory missing:\n%s", stdout)
	}
	stdout.Reset()
	if code := RunTaskGraphCheck(root, "feat", stdout, stderr); code != 0 {
		t.Fatalf("task-graph code=%d\n%s%s", code, stdout, stderr)
	}
	out := stdout.String()
	if !strings.Contains(out, "task-graph: ADVISORY size: SLICE-001 has 12") || strings.Contains(out, "SLICE-002 has") {
		t.Fatalf("task-graph advisory wrong:\n%s", out)
	}
	stdout.Reset()
	if code := RunMetrics(root, []string{"summary", "feat"}, stdout, stderr); code != 0 || !strings.Contains(stdout.String(), "size-advisory=1") {
		t.Fatalf("size-advisory metric not recorded: code=%d\n%s", code, stdout)
	}
}

func TestSliceSizeRationaleSilencesAdvisory(t *testing.T) {
	root := sliceWorkspace(t, bigSliceTasks("Size rationale: one generated golden set moves together\n"), "## Acceptance criteria\n- AC-001 works\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunTaskGraphCheck(root, "feat", stdout, stderr); code != 0 || strings.Contains(stdout.String(), "ADVISORY size") {
		t.Fatalf("rationale must silence advisory: code=%d\n%s", code, stdout)
	}
}

func TestCheckSliceIgnoresFencedFields(t *testing.T) {
	tasks := strings.Replace(goodSliceTasks, "Goal: do the thing.", "```text\nSatisfies: none\nWriter allowlist: .devrites/state.md\nTests/proof: example\n```\nGoal: do the thing.", 1)
	root := sliceWorkspace(t, tasks, sliceSpec)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunCheckSlice(root, []string{"feat", "SLICE-001"}, stdout, stderr); code != 0 || !strings.Contains(stdout.String(), "paths=2") {
		t.Fatalf("live fields must win over fenced examples: code=%d\n%s%s", code, stdout, stderr)
	}
	if problems := UnbuiltSliceProblems(root, "feat"); len(problems) != 0 {
		t.Fatalf("live contract rejected: %v", problems)
	}
}
