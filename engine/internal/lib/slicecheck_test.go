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
