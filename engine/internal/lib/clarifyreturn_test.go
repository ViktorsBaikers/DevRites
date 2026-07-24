package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/state"
)

func TestClarifyReturnSurvivesResolveAndRestoresLaterPhase(t *testing.T) {
	root := t.TempDir()
	slug := "retrofit"
	writeReadinessFile(t, root, slug, "state.md", `# State

## Cursor
| Key | Value |
| --- | --- |
| phase | build |
| status | running |
| next_action | /rite-build |
`)
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte(slug+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := ClarifyReturn(root, []string{"enter", slug}, &stdout, &stderr); code != 0 {
		t.Fatalf("enter code=%d stderr=%q", code, stderr.String())
	}
	lines := readStateLines(t, root, slug)
	assertCursor(t, lines, state.CursorPhase, "clarify")
	assertCursor(t, lines, state.CursorReturnPhase, "build")
	assertCursor(t, lines, state.CursorReturnNextAction, "/rite-build")

	writeReadinessFile(t, root, slug, "questions.md", `# Questions

## q-2026-07-23-001
status: open
gate: blocking
question: choose behavior
`)
	statePath := filepath.Join(root, "work", slug, "state.md")
	stateRaw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	waiting := strings.Replace(string(stateRaw), "| status | running |", "| status | awaiting_human |", 1) + `
## Awaiting human
| Key | Value |
| --- | --- |
| question_id | q-2026-07-23-001 |
| gate | blocking |
`
	if err := os.WriteFile(statePath, []byte(waiting), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Resolve(root, []string{"q-2026-07-23-001", "decided"}, &stdout, &stderr); code != 0 {
		t.Fatalf("resolve code=%d stderr=%q", code, stderr.String())
	}
	lines = readStateLines(t, root, slug)
	assertCursor(t, lines, state.CursorPhase, "clarify")
	next, _ := state.CursorField(lines, state.CursorNextAction)
	if !strings.Contains(next, "rite-clarify") || strings.Contains(next, "rite-build") {
		t.Fatalf("resolve next_action=%q, want clarify", next)
	}

	writeReadinessFile(t, root, slug, "brief.md", "# Brief\n\nOutcome.\n")
	writeReadinessFile(t, root, slug, "spec.md", "# Spec\n\n## Acceptance criteria\n- AC-001 succeeds.\n")
	writeReadinessFile(t, root, slug, "decisions.md", "# Decisions\n\nThe resolved answer is reflected in the spec.\n")
	writeReadinessFile(t, root, slug, "assumptions.md", "# Assumptions\n\nNone material.\n")
	digest, err := readinessInputsDigest(root, slug, readinessContract.Coverage.Inputs)
	if err != nil {
		t.Fatal(err)
	}
	writeReadinessFile(t, root, slug, "decision-coverage.md", validCoverage(digest))

	stdout.Reset()
	stderr.Reset()
	if code := ClarifyReturn(root, []string{"restore", slug}, &stdout, &stderr); code != 0 {
		t.Fatalf("restore code=%d stderr=%q", code, stderr.String())
	}
	lines = readStateLines(t, root, slug)
	assertCursor(t, lines, state.CursorPhase, "build")
	assertCursor(t, lines, state.CursorNextAction, "/rite-build")
	if _, ok := state.CursorField(lines, state.CursorReturnPhase); ok {
		t.Fatalf("return phase was not cleared:\n%s", strings.Join(lines, "\n"))
	}
}

func TestClarifyReturnRefusesRestoreBeforeFreshClear(t *testing.T) {
	root := t.TempDir()
	slug := "unclear"
	writeReadinessFile(t, root, slug, "state.md", "- Phase: build\n- Status: running\n- Next step: /rite-build\n")
	if code := ClarifyReturn(root, []string{"enter", slug}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("enter code=%d", code)
	}
	if code := ClarifyReturn(root, []string{"restore", slug}, &bytes.Buffer{}, &bytes.Buffer{}); code != 3 {
		t.Fatalf("restore code=%d, want 3 before CLEAR", code)
	}
}

func readStateLines(t *testing.T, root, slug string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "work", slug, "state.md"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(string(raw), "\n")
}

func assertCursor(t *testing.T, lines []string, key, want string) {
	t.Helper()
	got, ok := state.CursorField(lines, key)
	if !ok || got != want {
		t.Fatalf("%s=(%q,%v), want %q\n%s", key, got, ok, want, strings.Join(lines, "\n"))
	}
}
