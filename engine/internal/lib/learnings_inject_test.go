package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func writeLedger(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "learnings.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestTopLearningsThresholdAndCap(t *testing.T) {
	root := writeLedger(t, learningsHeader+
		"- [2026-07-01] (rule · slug · c=0.9) always pin actions\n"+
		"- [2026-07-02] (note · slug) unmarked entry defaults to 0.5\n"+
		"- [2026-07-03] (rule · slug · c=0.75) prefer table tests\n"+
		"- [2026-07-04] (rule · slug · c=0.6) below the floor\n")

	got := TopLearnings(root, 5, 0.7)
	if len(got) != 2 {
		t.Fatalf("expected 2 entries >= 0.7, got %d: %v", len(got), got)
	}
	// Highest confidence first.
	if want := "always pin actions"; !contains(got[0], want) {
		t.Errorf("expected highest-confidence first; got[0]=%q", got[0])
	}
	if contains(got[0]+got[1], "below the floor") || contains(got[0]+got[1], "defaults to 0.5") {
		t.Error("sub-threshold entries must not be injected")
	}

	// Cap wins.
	if capped := TopLearnings(root, 1, 0.7); len(capped) != 1 {
		t.Errorf("cap=1 should yield 1 entry, got %d", len(capped))
	}
	// max<=0 disables entirely.
	if off := TopLearnings(root, 0, 0.7); off != nil {
		t.Errorf("max=0 should disable injection, got %v", off)
	}
}

func TestTopLearningsMissingLedger(t *testing.T) {
	if got := TopLearnings(t.TempDir(), 5, 0.7); got != nil {
		t.Errorf("missing ledger should yield nil, got %v", got)
	}
}

func TestParseConfidence(t *testing.T) {
	for _, s := range []string{"0", "1", "0.5", "0.73"} {
		if _, ok := parseConfidence(s); !ok {
			t.Errorf("%q should parse", s)
		}
	}
	for _, s := range []string{"-0.1", "1.5", "high", ""} {
		if _, ok := parseConfidence(s); ok {
			t.Errorf("%q should be rejected", s)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
