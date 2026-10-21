package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devrites/devrites/internal/gate"
)

// driftWorkspace builds a workspace with every required readiness input.
func driftWorkspace(t *testing.T) (root, featureDir string) {
	t.Helper()
	root, featureDir = newWorkspace(t, "vet")
	for _, name := range []string{
		"spec.md", "decision-coverage.md", "architecture.md",
		"plan.md", "tasks.md", "traceability.md", "test-plan.md",
	} {
		writeFile(t, filepath.Join(featureDir, name), "# "+name+"\n\ncontent\n")
	}
	return root, featureDir
}

func checkDrift(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunCheckDrift(root, args, stdout, stderr)
	return code, stdout.String() + stderr.String()
}

func TestDriftUnprovenWithoutBaselineOrBinding(t *testing.T) {
	root, _ := driftWorkspace(t)
	code, out := checkDrift(t, root, "feat")
	if code != 0 || !strings.Contains(out, "unproven") {
		t.Fatalf("code=%d out=%s", code, out)
	}
}

func TestDriftRecordThenCleanPass(t *testing.T) {
	root, _ := driftWorkspace(t)
	code, out := checkDrift(t, root, "feat", "--record")
	if code != 0 || !strings.Contains(out, "baseline recorded") {
		t.Fatalf("record code=%d out=%s", code, out)
	}
	code, out = checkDrift(t, root, "feat")
	if code != 0 || !strings.Contains(out, "drift: none") {
		t.Fatalf("compare code=%d out=%s", code, out)
	}
}

func TestDriftAttributesChangedFile(t *testing.T) {
	root, featureDir := driftWorkspace(t)
	if code, out := checkDrift(t, root, "feat", "--record"); code != 0 {
		t.Fatalf("record code=%d out=%s", code, out)
	}
	writeFile(t, filepath.Join(featureDir, "plan.md"), "# plan.md\n\nrewritten\n")
	writeFile(t, filepath.Join(featureDir, "strategy.md"), "# strategy\n\nnew optional input\n")
	code, out := checkDrift(t, root, "feat")
	if code != 0 {
		t.Fatalf("code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "plan.md changed") || !strings.Contains(out, "strategy.md added") {
		t.Fatalf("expected per-file attribution:\n%s", out)
	}
	if strings.Contains(out, "spec.md changed") {
		t.Fatalf("spec.md did not change:\n%s", out)
	}
}

func TestDriftFallbackEngReviewBinding(t *testing.T) {
	root, featureDir := driftWorkspace(t)
	_, binding, err := gate.ReadinessInputDigests(root, "feat")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(featureDir, "eng-review.md"), "# Eng review\n\n"+binding+"\n")
	code, out := checkDrift(t, root, "feat")
	if code != 0 || !strings.Contains(out, "drift: none") {
		t.Fatalf("code=%d out=%s", code, out)
	}
	writeFile(t, filepath.Join(featureDir, "tasks.md"), "# tasks\n\ndrifted\n")
	// eng-review must predate the drifted file for the mtime heuristic.
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(featureDir, "eng-review.md"), old, old); err != nil {
		t.Fatal(err)
	}
	code, out = checkDrift(t, root, "feat")
	if code != 0 {
		t.Fatalf("code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "DRIFT: readiness inputs changed") || !strings.Contains(out, "tasks.md") {
		t.Fatalf("expected aggregate drift + candidate:\n%s", out)
	}
}

func TestDriftCorruptBaselineBlocks(t *testing.T) {
	root, featureDir := driftWorkspace(t)
	writeFile(t, filepath.Join(featureDir, driftBaselineFile), "{not json")
	code, out := checkDrift(t, root, "feat")
	if code != 3 || !strings.Contains(out, "BLOCKED") {
		t.Fatalf("code=%d out=%s", code, out)
	}
}

func TestDriftClassifiesReviewedIdentityOnlyChange(t *testing.T) {
	root, featureDir := driftWorkspace(t)
	if code, out := checkDrift(t, root, "feat", "--record"); code != 0 {
		t.Fatalf("record code=%d out=%s", code, out)
	}
	plan := filepath.Join(featureDir, "test-plan.md")
	line := "Workflow Artifact reviewed identity: " + strings.Repeat("ab", 32) + "\n"
	writeFile(t, plan, "# test-plan.md\n\ncontent\n"+line)
	code, out := checkDrift(t, root, "feat")
	if code != 0 || !strings.Contains(out, "test-plan.md reviewed-identity-only") || !strings.Contains(out, "bytes-only recheck") {
		t.Fatalf("identity-only change not classified: code=%d\n%s", code, out)
	}
	// Any other edit, including a malformed identity line, is ordinary drift.
	for _, body := range []string{
		"# test-plan.md\n\nchanged\n" + line,
		"# test-plan.md\n\ncontent\nWorkflow Artifact reviewed identity: not-a-digest\n",
	} {
		writeFile(t, plan, body)
		if _, out := checkDrift(t, root, "feat"); !strings.Contains(out, "test-plan.md changed") || strings.Contains(out, "bytes-only") {
			t.Fatalf("non-identity edit hidden:\n%s", out)
		}
	}
	writeFile(t, plan, "# test-plan.md\n\ncontent\n"+line)
	writeFile(t, filepath.Join(featureDir, "plan.md"), "# plan.md\n\nrewritten\n")
	if _, out := checkDrift(t, root, "feat"); strings.Contains(out, "bytes-only") {
		t.Fatalf("a second changed input must void the bytes-only route:\n%s", out)
	}
}

// A fenced example binding in eng-review.md is documentation, not the record.
func TestDriftFallbackIgnoresFencedExampleBinding(t *testing.T) {
	root, featureDir := driftWorkspace(t)
	_, binding, err := gate.ReadinessInputDigests(root, "feat")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(featureDir, "eng-review.md"), "# Eng review\n\n"+binding+"\n\nFormat:\n\n```text\nReadiness inputs SHA-256: 0000\n```\n")
	if code, out := checkDrift(t, root, "feat"); code != 0 || !strings.Contains(out, "drift: none") {
		t.Fatalf("code=%d out=%s", code, out)
	}
}

func TestDriftRecordLabelsCursorPhase(t *testing.T) {
	root, featureDir := driftWorkspace(t)
	if code, out := checkDrift(t, root, "feat", "--record"); code != 0 {
		t.Fatalf("record code=%d out=%s", code, out)
	}
	raw, err := os.ReadFile(filepath.Join(featureDir, MetricsFile))
	if err != nil || !strings.Contains(string(raw), `"phase":"vet","event":"drift-baseline"`) {
		t.Fatalf("metrics=%q err=%v", raw, err)
	}
}

func TestDiffInputsClassifiesTraceabilityProgressOnly(t *testing.T) {
	base := []gate.ReadinessInputDigest{{Path: "traceability.md", Present: true, SHA256: "a", Normalized: "n"}}
	cur := []gate.ReadinessInputDigest{{Path: "traceability.md", Present: true, SHA256: "b", Normalized: "n"}}
	if diffs := diffInputs(base, cur); len(diffs) != 1 || diffs[0].Kind != "progress-only" {
		t.Fatalf("diffs = %+v, want one progress-only", diffs)
	}
	cur[0].Normalized = "m"
	if diffs := diffInputs(base, cur); len(diffs) != 1 || diffs[0].Kind != "changed" {
		t.Fatalf("diffs = %+v, want one changed", diffs)
	}
}
