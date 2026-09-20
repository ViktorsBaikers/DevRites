package lib

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandoffEmitsResumeRecord(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	writeFile(t, filepath.Join(featureDir, "questions.md"), "# Questions\n\n## Q-1\nstatus: open\ngate: blocking\n")
	writeFile(t, filepath.Join(featureDir, "decisions.md"), "# Decisions\n\n## Dead ends\n\n- cached lookup: races the writer\n- retry loop: masks the root cause\n")
	writeFile(t, filepath.Join(featureDir, "gates.md"), "# Gates\n\n- [x] G1: builds\n  CHECK: go build ./...\n  EXPECT: ok\n  EVIDENCE: ran\n- [ ] G2: tests green\n  CHECK: go test ./...\n  EXPECT: ok\n")
	writeFile(t, filepath.Join(featureDir, "spec.md"), "# Spec\n")

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunHandoff(root, []string{"feat"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"feature: feat",
		"phase: build",
		"status: running",
		"awaiting_human: no",
		"open_questions: 1",
		"blocking_question_gates: blocking",
		"gates_total: 2",
		"gates_met: 0",
		"gates_unmet: 2",
		"gates_stale: 1",
		"unmet_id: G1",
		"unmet_id: G2",
		"stale_id: G1",
		"dead_end: cached lookup: races the writer",
		"dead_end: retry loop: masks the root cause",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "read_next:") || !strings.Contains(out, "state.md") || !strings.Contains(out, "gates.md") {
		t.Fatalf("read_next missing present artifacts:\n%s", out)
	}
}

func TestHandoffMissingLedgerAndNoQuestions(t *testing.T) {
	root, _ := newWorkspace(t, "spec")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunHandoff(root, []string{"feat"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"gates_ledger: absent",
		"open_questions: 0",
		"blocking_question_gates: none",
		"dead_ends: none",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestHandoffRejectsMissingFeature(t *testing.T) {
	root, _ := newWorkspace(t, "spec")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunHandoff(root, []string{"nope"}, stdout, stderr); code == 0 {
		t.Fatalf("expected non-zero for missing feature, out=%s", stdout.String())
	}
}

func TestDeadEndsBoundsAndSectionScope(t *testing.T) {
	body := "# D\n\n## Dead ends\n\n- one\n- two\n\n## DEC-001 other\n- not a dead end\n"
	ends := deadEnds([]byte(body))
	if len(ends) != 2 || ends[0] != "one" || ends[1] != "two" {
		t.Fatalf("ends=%v", ends)
	}
}
