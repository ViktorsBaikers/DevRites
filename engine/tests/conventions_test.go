package main_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConventionsLedgerCommand(t *testing.T) {
	work := t.TempDir()

	out, _, code := runGo(t, work, "conventions", "band", "--corroborations", "1")
	if code != 0 || strings.TrimSpace(out) != "0.50" {
		t.Fatalf("band c1/k0 = exit %d %q, want 0.50", code, out)
	}
	out, _, code = runGo(t, work, "conventions", "band", "--corroborations", "1", "--contradictions", "1")
	if code != 0 || strings.TrimSpace(out) != "retired" {
		t.Fatalf("band retired = exit %d %q, want retired", code, out)
	}

	out, _, code = runGo(t, work, "conventions", "read", "--root", work)
	if code != 0 || out != "" {
		t.Fatalf("read missing = exit %d %q, want empty success", code, out)
	}

	out, _, code = runGo(t, work, "conventions", "promote",
		"--root", work,
		"--key", "test-runner",
		"--statement", "tests run with vitest, co-located *.test.ts",
		"--kind", "test",
		"--slug", "01-list",
		"--evidence", "evidence.md: 12 passing",
		"--date", "2026-06-20")
	if code != 0 || !strings.Contains(out, "band 0.50") {
		t.Fatalf("promote = exit %d %q, want band 0.50", code, out)
	}

	runGo(t, work, "conventions", "promote",
		"--root", work,
		"--key", "test-runner",
		"--statement", "",
		"--kind", "test",
		"--slug", "02-detail",
		"--evidence", "evidence.md: 8 passing",
		"--date", "2026-06-21")
	runGo(t, work, "conventions", "promote",
		"--root", work,
		"--key", "test-runner",
		"--statement", "",
		"--kind", "test",
		"--slug", "02-detail",
		"--evidence", "re-seal",
		"--date", "2026-06-21")

	ledgerPath := filepath.Join(work, ".devrites", "conventions.md")
	ledger, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	ledgerText := string(ledger)
	for _, want := range []string{
		"## test-runner",
		"band: 0.60",
		"corroborations: 2",
		"01-list 2026-06-20",
	} {
		if !strings.Contains(ledgerText, want) {
			t.Fatalf("ledger missing %q\n%s", want, ledgerText)
		}
	}

	driftPath := filepath.Join(work, ".devrites", "features", "03-api", "drift.md")
	out, _, code = runGo(t, work, "conventions", "contradict",
		"--root", work,
		"--key", "test-runner",
		"--slug", "03-api",
		"--evidence", "now uses jest",
		"--date", "2026-06-22",
		"--drift-file", driftPath)
	if code != 0 || !strings.Contains(out, "DRIFT: convention 'test-runner' contradicted by 03-api") {
		t.Fatalf("contradict = exit %d %q, want drift output", code, out)
	}
	drift, err := os.ReadFile(driftPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(drift), "## convention-drift: test-runner") ||
		!strings.Contains(string(drift), "- now: band 0.40") {
		t.Fatalf("drift file missing expected entry\n%s", string(drift))
	}

	out, _, code = runGo(t, work, "conventions", "orient", "--root", work, "--min-band", "0.4")
	if code != 0 || !strings.Contains(out, "- [0.40] test-runner (test): tests run with vitest") {
		t.Fatalf("orient = exit %d %q, want active convention", code, out)
	}

	_, errOut, code := runGo(t, work, "conventions", "contradict",
		"--root", work,
		"--key", "missing",
		"--slug", "04-missing",
		"--evidence", "none")
	if code != 3 || !strings.Contains(errOut, "no such convention: 'missing'") {
		t.Fatalf("unknown contradict = exit %d stderr %q, want code 3", code, errOut)
	}
}
