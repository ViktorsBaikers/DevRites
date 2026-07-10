package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func recordStat(t *testing.T, root, agent string, findings int) {
	t.Helper()
	var out, errOut strings.Builder
	if rc := ReviewerStats(root, []string{"record", agent, fmt.Sprint(findings)}, &out, &errOut); rc != 0 {
		t.Fatalf("record %s %d: rc=%d stderr=%s", agent, findings, rc, errOut.String())
	}
}

func reportStats(t *testing.T, root string) string {
	t.Helper()
	var out, errOut strings.Builder
	if rc := ReviewerStats(root, []string{"report"}, &out, &errOut); rc != 0 {
		t.Fatalf("report: rc=%d stderr=%s", rc, errOut.String())
	}
	return out.String()
}

func TestReviewerStatsGateCandidateAfterDryStreak(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < gateStreak; i++ {
		recordStat(t, root, "devrites-performance-reviewer", 0)
	}
	got := reportStats(t, root)
	if !strings.Contains(got, "devrites-performance-reviewer: gate-candidate") {
		t.Fatalf("want gate-candidate after %d dry dispatches, got %q", gateStreak, got)
	}
}

func TestReviewerStatsFindingResetsStreak(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < gateStreak; i++ {
		recordStat(t, root, "devrites-devex-reviewer", 0)
	}
	recordStat(t, root, "devrites-devex-reviewer", 2)
	got := reportStats(t, root)
	if !strings.Contains(got, "devrites-devex-reviewer: run (") {
		t.Fatalf("a finding must reset the streak to run, got %q", got)
	}
}

func TestReviewerStatsInsuranceAndAlwaysOnNeverGate(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < gateStreak+5; i++ {
		recordStat(t, root, "devrites-security-auditor", 0)
		recordStat(t, root, "devrites-spec-reviewer", 0)
	}
	got := reportStats(t, root)
	if strings.Contains(got, "gate-candidate") {
		t.Fatalf("insurance/always-on reviewers must never be gate-candidates, got %q", got)
	}
	if !strings.Contains(got, "devrites-security-auditor: run (insurance") {
		t.Fatalf("want insurance verdict for security auditor, got %q", got)
	}
	if !strings.Contains(got, "devrites-spec-reviewer: run (always-on)") {
		t.Fatalf("want always-on verdict for spec reviewer, got %q", got)
	}
}

func TestReviewerStatsRejectsBadInput(t *testing.T) {
	root := t.TempDir()
	var out, errOut strings.Builder
	if rc := ReviewerStats(root, []string{"record", "not a valid agent", "0"}, &out, &errOut); rc != 2 {
		t.Fatalf("invalid agent name: want rc=2, got %d", rc)
	}
	if rc := ReviewerStats(root, []string{"record", "devrites-code-reviewer", "-1"}, &out, &errOut); rc != 2 {
		t.Fatalf("negative findings: want rc=2, got %d", rc)
	}
}

func TestReviewerStatsCorruptLineDegradesNotBreaks(t *testing.T) {
	root := t.TempDir()
	recordStat(t, root, "devrites-frontend-reviewer", 1)
	f, err := os.OpenFile(filepath.Join(root, reviewerStatsFile), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{not json\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	got := reportStats(t, root)
	if !strings.Contains(got, "devrites-frontend-reviewer: run (1 finding / 1 dispatch, zero-streak 0)") {
		t.Fatalf("corrupt line must not break the report, got %q", got)
	}
}

func TestReviewerStatsEmptyReport(t *testing.T) {
	got := reportStats(t, t.TempDir())
	if !strings.Contains(got, "no dispatches recorded") {
		t.Fatalf("empty ledger: got %q", got)
	}
}
