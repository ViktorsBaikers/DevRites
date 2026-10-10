package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const regressionTasks = `# Tasks

| Slice | Outcome | Depends on |
| --- | --- | --- |
| SLICE-001 | first | none |
| SLICE-002 | second | SLICE-001 |

## SLICE-001 first
Goal: one.
Status: built

## SLICE-002 second
Goal: two.
Status: pending
`

const regressionSpec = `# Spec

## Acceptance criteria
- [x] AC-001: first thing works
- [ ] AC-002: second thing works
`

const regressionGates = `# Gates

- [x] AC-001: first thing works
  EVIDENCE: attested by reviewer
- [ ] AC-002: second thing works
  CHECK: go test ./...
  EXPECT: ok
  EVIDENCE: pending
`

const regressionQuestions = `# Questions

## q-2026-01-01-001 pick a store
status: answered
answer: sqlite
answered_at: 2026-01-02T00:00:00Z

## q-2026-01-01-002 pick a cache
status: open
`

func regressionWorkspace(t *testing.T, phase string) (root, featureDir string) {
	t.Helper()
	root, featureDir = newWorkspace(t, phase)
	writeFile(t, filepath.Join(featureDir, "spec.md"), regressionSpec)
	writeFile(t, filepath.Join(featureDir, "tasks.md"), regressionTasks)
	writeFile(t, filepath.Join(featureDir, "gates.md"), regressionGates)
	writeFile(t, filepath.Join(featureDir, "questions.md"), regressionQuestions)
	return root, featureDir
}

func checkRegression(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunCheckRegression(root, args, stdout, stderr)
	return code, stdout.String() + stderr.String()
}

func TestRegressionUnprovenWithoutBaseline(t *testing.T) {
	root, _ := regressionWorkspace(t, "build")
	code, out := checkRegression(t, root, "feat")
	if code != 0 || !strings.Contains(out, "unproven") {
		t.Fatalf("code=%d out=%s", code, out)
	}
}

func TestRegressionBaselineUpdateThenCleanPass(t *testing.T) {
	root, _ := regressionWorkspace(t, "build")
	code, out := checkRegression(t, root, "feat", "--update")
	if code != 0 || !strings.Contains(out, "baseline updated") {
		t.Fatalf("update code=%d out=%s", code, out)
	}
	code, out = checkRegression(t, root, "feat")
	if code != 0 || !strings.Contains(out, "PASS") {
		t.Fatalf("check code=%d out=%s", code, out)
	}
}

func TestRegressionDetectsProgressLoss(t *testing.T) {
	root, dir := regressionWorkspace(t, "prove")
	if code, out := checkRegression(t, root, "feat", "--update"); code != 0 {
		t.Fatalf("update code=%d out=%s", code, out)
	}

	writeFile(t, filepath.Join(dir, "spec.md"), `# Spec

## Acceptance criteria
- [ ] AC-001: first thing works
- [ ] AC-002: second thing works
`)
	writeFile(t, filepath.Join(dir, "tasks.md"), strings.Replace(regressionTasks, "Status: built", "Status: in-progress", 1))
	writeFile(t, filepath.Join(dir, "gates.md"), strings.Replace(regressionGates, "- [x] AC-001", "- [ ] AC-001", 1))
	writeFile(t, filepath.Join(dir, "questions.md"), strings.Replace(regressionQuestions, "status: answered", "status: open", 1))
	writeFile(t, filepath.Join(dir, "state.md"), "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | build |\n| status | running |\n")

	code, out := checkRegression(t, root, "feat")
	if code != 3 {
		t.Fatalf("expected blocked, code=%d out=%s", code, out)
	}
	for _, want := range []string{"AC-001", "gate AC-001", "SLICE-001", "q-2026-01-01-001", "phase moved backward"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestRegressionDetectsRemovedArtifact(t *testing.T) {
	root, dir := regressionWorkspace(t, "build")
	if code, _ := checkRegression(t, root, "feat", "--update"); code != 0 {
		t.Fatalf("update failed")
	}
	if err := os.Remove(filepath.Join(dir, "gates.md")); err != nil {
		t.Fatal(err)
	}
	code, out := checkRegression(t, root, "feat")
	if code != 3 || !strings.Contains(out, "gates.md") {
		t.Fatalf("code=%d out=%s", code, out)
	}
}

func TestRegressionIgnoresFencedDecoys(t *testing.T) {
	root, dir := regressionWorkspace(t, "build")
	writeFile(t, filepath.Join(dir, "spec.md"), "# Spec\n\n```\n- [x] AC-099: fenced decoy\n```\n\n## Acceptance criteria\n- [ ] AC-001: real\n")
	if code, out := checkRegression(t, root, "feat", "--update"); code != 0 {
		t.Fatalf("update code=%d out=%s", code, out)
	}
	writeFile(t, filepath.Join(dir, "spec.md"), "# Spec\n\n## Acceptance criteria\n- [ ] AC-001: real\n")
	code, out := checkRegression(t, root, "feat")
	if code != 0 {
		t.Fatalf("fenced decoy was recorded: code=%d out=%s", code, out)
	}
}

func TestRegressionUpdateBlessesAndReports(t *testing.T) {
	root, dir := regressionWorkspace(t, "build")
	if code, _ := checkRegression(t, root, "feat", "--update"); code != 0 {
		t.Fatalf("update failed")
	}
	writeFile(t, filepath.Join(dir, "spec.md"), strings.Replace(regressionSpec, "- [x] AC-001", "- [ ] AC-001", 1))
	code, out := checkRegression(t, root, "feat", "--update")
	if code != 0 || !strings.Contains(out, "REGRESSION-BLESSED") {
		t.Fatalf("update code=%d out=%s", code, out)
	}
	if code, out := checkRegression(t, root, "feat"); code != 0 {
		t.Fatalf("post-bless check should pass: code=%d out=%s", code, out)
	}
}

func TestRegressionCorruptBaselineBlocks(t *testing.T) {
	root, dir := regressionWorkspace(t, "build")
	writeFile(t, filepath.Join(dir, regressionBaselineFile), "{not json")
	code, out := checkRegression(t, root, "feat")
	if code != 3 || !strings.Contains(out, "unreadable") {
		t.Fatalf("code=%d out=%s", code, out)
	}
}

func TestRegressionUsageErrors(t *testing.T) {
	root, _ := regressionWorkspace(t, "build")
	if code, _ := checkRegression(t, root); code != 2 {
		t.Fatalf("no args code=%d", code)
	}
	if code, _ := checkRegression(t, root, "feat", "--bogus"); code != 2 {
		t.Fatalf("bad flag code=%d", code)
	}
}

func TestRegressionDegradedArtifactRefusesRatchet(t *testing.T) {
	root, dir := regressionWorkspace(t, "build")
	if code, out := checkRegression(t, root, "feat", "--update"); code != 0 || !strings.Contains(out, "baseline updated") {
		t.Fatalf("healthy update code=%d out=%s", code, out)
	}
	baselinePath := filepath.Join(dir, regressionBaselineFile)
	before, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatal(err)
	}
	healthy, err := fingerprintWorkspace(root, "feat", dir)
	if err != nil || len(healthy.CheckedAC) != 1 || healthy.CheckedAC[0] != "AC-001" {
		t.Fatalf("healthy fingerprint err=%v checked=%v", err, healthy.CheckedAC)
	}

	writeFile(t, filepath.Join(dir, "spec.md"), "# Spec\n\n## Acceptance criteria\n- [x] AC-001: a\x00\n")
	if _, err := fingerprintWorkspace(root, "feat", dir); err == nil || !strings.Contains(err.Error(), "spec.md") {
		t.Fatalf("degraded fingerprint must name spec.md, err=%v", err)
	}
	code, out := checkRegression(t, root, "feat", "--update")
	if code != 2 || strings.Contains(out, "baseline updated") || !strings.Contains(out, "spec.md") {
		t.Fatalf("degraded ratchet code=%d out=%s", code, out)
	}
	after, err := os.ReadFile(baselinePath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("baseline rewritten from degraded fingerprint: err=%v", err)
	}
}

func TestRegressionUpdateRefusesToWipeAcceptanceFacts(t *testing.T) {
	root, dir := regressionWorkspace(t, "build")
	if code, out := checkRegression(t, root, "feat", "--update"); code != 0 || !strings.Contains(out, "baseline updated") {
		t.Fatalf("healthy update code=%d out=%s", code, out)
	}
	baselinePath := filepath.Join(dir, regressionBaselineFile)
	before, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(dir, "spec.md"), strings.Replace(regressionSpec, "- [x] AC-001", "- [ ] AC-001", 1))
	writeFile(t, filepath.Join(dir, "gates.md"), strings.Replace(regressionGates, "- [x] AC-001", "- [ ] AC-001", 1))

	code, out := checkRegression(t, root, "feat", "--update")
	if code != 3 || strings.Contains(out, "baseline updated") || !strings.Contains(out, "--allow-empty") {
		t.Fatalf("wiping ratchet code=%d out=%s", code, out)
	}
	after, err := os.ReadFile(baselinePath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("baseline rewritten by refused wipe: err=%v", err)
	}

	code, out = checkRegression(t, root, "feat", "--update", "--allow-empty")
	if code != 0 || !strings.Contains(out, "REGRESSION-BLESSED") || !strings.Contains(out, "baseline updated") {
		t.Fatalf("explicit wipe code=%d out=%s", code, out)
	}
}

func TestRegressionMalformedGatesRefusesRatchet(t *testing.T) {
	root, dir := regressionWorkspace(t, "build")
	if code, out := checkRegression(t, root, "feat", "--update"); code != 0 || !strings.Contains(out, "baseline updated") {
		t.Fatalf("healthy update code=%d out=%s", code, out)
	}
	baselinePath := filepath.Join(dir, regressionBaselineFile)
	before, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(dir, "gates.md"), strings.Replace(regressionGates, "first thing works", "first thing works\x00", 1))
	if _, err := fingerprintWorkspace(root, "feat", dir); err == nil || !strings.Contains(err.Error(), "gates.md") {
		t.Fatalf("malformed gates fingerprint must name gates.md, err=%v", err)
	}
	code, out := checkRegression(t, root, "feat", "--update", "--allow-empty")
	if code != 2 || strings.Contains(out, "baseline updated") || !strings.Contains(out, "gates.md") {
		t.Fatalf("malformed gates ratchet code=%d out=%s", code, out)
	}
	after, err := os.ReadFile(baselinePath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("baseline rewritten from malformed gates: err=%v", err)
	}
}

func TestRegressionUnreadablePhaseIsNotPass(t *testing.T) {
	root, dir := regressionWorkspace(t, "prove")
	if code, out := checkRegression(t, root, "feat", "--update"); code != 0 || !strings.Contains(out, "baseline updated") {
		t.Fatalf("healthy update code=%d out=%s", code, out)
	}
	if code, out := checkRegression(t, root, "feat"); code != 0 || !strings.Contains(out, "PASS") {
		t.Fatalf("healthy control code=%d out=%s", code, out)
	}

	writeFile(t, filepath.Join(dir, "state.md"), "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| status | running |\n")
	// Premise: the corrupted state.md must leave the phase unobservable.
	if fp, err := fingerprintWorkspace(root, "feat", dir); err == nil && fp.PhaseOrdinal >= 0 {
		t.Fatalf("premise failed: phase still observable, ordinal=%d", fp.PhaseOrdinal)
	}
	code, out := checkRegression(t, root, "feat")
	if code == 0 || strings.Contains(out, "PASS") {
		t.Fatalf("unreadable phase must not pass: code=%d out=%s", code, out)
	}
}
