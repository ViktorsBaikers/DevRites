package main_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

func TestADR0027ReadinessBindingBlocksPlanDriftWithRestoredMtime(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	workspace := filepath.Join(root, "work", "bound")
	for name, body := range map[string]string{
		"state.md":             "| phase | build |\n",
		"brief.md":             "# Brief\n\nBuild the approved slice.\n",
		"spec.md":              "# Spec\n\n## Acceptance Criteria\n- AC-001: bind readiness inputs.\n",
		"decisions.md":         "# Decisions\n\nNone.\n",
		"assumptions.md":       "# Assumptions\n\nNone.\n",
		"questions.md":         "# Questions\n\nNone.\n",
		"decision-coverage.md": "# Decision coverage\n\nCLEAR\n",
		"architecture.md":      "# Architecture\n\nUse the deterministic gate.\n",
		"plan.md":              "# Plan\n\nBuild slice A.\n",
		"tasks.md":             "# Tasks\n\n- [ ] Build slice A.\n",
		"traceability.md":      "# Traceability\n\nAC-001 -> slice A.\n",
		"eng-review.md":        "# Engineering review\n\nREADY\n",
		"test-plan.md":         "# Test plan\n\nRun focused Go tests.\n",
	} {
		testutil.WriteFile(t, filepath.Join(workspace, name), body)
	}

	binding := emitReadinessBinding(t, root, "bound")
	testutil.AppendFile(t, filepath.Join(workspace, "eng-review.md"), "\n"+binding+"\n")
	out, errOut, code := runDevrites(t, root, "check", "readiness", "bound")
	if code != 0 {
		t.Fatalf("initial readiness exit=%d, want 0\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}

	planPath := filepath.Join(workspace, "plan.md")
	before, err := os.Stat(planPath)
	if err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, planPath, "# Plan\n\nBuild a different slice.\n")
	if err := os.Chtimes(planPath, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	expected := emitReadinessBinding(t, root, "bound")

	out, errOut, code = runDevrites(t, root, "check", "readiness", "bound")
	if code != 3 {
		t.Fatalf("stale readiness exit=%d, want 3\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}
	for _, want := range []string{
		"reason: DRV-GATE-READINESS-STALE",
		"rerun /rite-vet",
		expected,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stale readiness output missing %q:\n%s", want, out)
		}
	}
}

func TestADR0027SealChecksReadinessBindingBeforeEvidenceFreshness(t *testing.T) {
	project, root := newFinalSealRepo(t)
	planPath := filepath.Join(root, "work", "final", "plan.md")
	before, err := os.Stat(planPath)
	if err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, planPath, "# Plan\n\nChanged after the final review.\n")
	if err := os.Chtimes(planPath, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	expected := emitReadinessBinding(t, root, "final")
	chdirForSeal(t, project)

	out, errOut, code := runDevrites(t, root, "check", "seal", "final")
	if code != 3 {
		t.Fatalf("stale seal exit=%d, want 3\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}
	if !strings.Contains(out, "reason: DRV-GATE-READINESS-STALE") || !strings.Contains(out, expected) {
		t.Fatalf("stale seal output lacks reason or expected binding:\n%s", out)
	}
	if strings.Contains(out, "evidence-fresh:") || strings.Contains(errOut, "evidence-fresh:") {
		t.Fatalf("seal ran evidence freshness after stale readiness\nstdout:\n%s\nstderr:\n%s", out, errOut)
	}
}

func emitReadinessBinding(t *testing.T, root, slug string) string {
	t.Helper()
	out, errOut, code := runDevrites(t, root, "check", "readiness", "--emit-binding", slug)
	if code != 0 {
		t.Fatalf("emit readiness binding exit=%d, want 0\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}
	line := strings.TrimSuffix(out, "\n")
	const prefix = "Readiness inputs SHA-256: "
	digest := strings.TrimPrefix(line, prefix)
	if digest == line || len(digest) != 64 || strings.Trim(digest, "0123456789abcdef") != "" {
		t.Fatalf("malformed readiness binding output %q", out)
	}
	return line
}
