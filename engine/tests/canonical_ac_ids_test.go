package main_test

import (
	"strings"
	"testing"
)

func TestCanonicalAcceptanceIDsAcrossGates(t *testing.T) {
	work := t.TempDir()
	writeFeatureFile(t, work, "canonical", "spec.md",
		"## Acceptance criteria\n- [ ] AC-001: alpha\n- [ ] AC-002: beta\n")
	writeFeatureFile(t, work, "canonical", "tasks.md",
		"## SLICE-001 alpha\nSatisfies: AC-001\n## SLICE-002 beta\nSatisfies: AC-002\n## SLICE-003 orphan\n")
	writeFeatureFile(t, work, "canonical", "seal.md",
		"## Acceptance Criteria\n- [x] AC-001: alpha — evidence: t1\n- [x] AC-002: beta — evidence: t2\n")

	t.Run("analyze", func(t *testing.T) {
		stdout, stderr, code := runGo(t, work, "analyze", "canonical")
		if code != 0 || !strings.Contains(stdout, "AC-001 — covered") || !strings.Contains(stdout, "slice 'SLICE-003 orphan'") {
			t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
	})

	t.Run("coverage", func(t *testing.T) {
		stdout, stderr, code := runGo(t, work, "coverage", "canonical")
		if code != 0 || !strings.Contains(stdout, "| AC-001 | SLICE-001 alpha | yes |") {
			t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
	})

	t.Run("check-acceptance", func(t *testing.T) {
		stdout, stderr, code := runGo(t, work, "check-acceptance", ".devrites/features/canonical")
		if code != 0 || !strings.Contains(stdout, "AC-001 AC-002") {
			t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
	})

	t.Run("lanes", func(t *testing.T) {
		stdout, stderr, code := runGo(t, work, "lanes", "plan", "canonical")
		if code != 0 || !strings.Contains(stdout, "- SLICE-001 alpha:") {
			t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
	})
}
