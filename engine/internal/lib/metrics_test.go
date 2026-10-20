package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An oversized line never truncates the summary: events after it still count
// and the garbage line is reported as unparsed.
func TestMetricsSummaryReadsPastOversizedLine(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	ev := func(e string) string {
		return `{"ts":"2026-01-01T00:00:00Z","phase":"build","event":"` + e + `"}` + "\n"
	}
	ledger := ev("a") + strings.Repeat("x", 1<<20) + "\n" + ev("b") + ev("c")
	if err := os.WriteFile(filepath.Join(featureDir, MetricsFile), []byte(ledger), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunMetrics(root, []string{"summary", "feat"}, stdout, stderr)
	if code != 0 || !strings.Contains(stdout.String(), "events=3") || !strings.Contains(stdout.String(), "unparsed=1") {
		t.Fatalf("code=%d out=%q err=%q", code, stdout, stderr)
	}
}

// A reviewer's Outcome line recorded at return feeds the per-role yield tally;
// an abandon is a metric too, so lost waves show up in the summary.
func TestDispatchOutcomeAndAbandonFeedSummary(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	run := func(want int, args ...string) {
		t.Helper()
		if code := RunDispatch(root, append([]string{"feat"}, args...), out, errOut); code != want {
			t.Fatalf("dispatch %v: code=%d want %d\n%s", args, code, want, errOut)
		}
	}
	run(0, "open", "--phase", "build", "--wave", "r1", "--role", "x", "--role", "y")
	run(0, "start", "--wave", "r1", "--role", "x", "--handle", "h1")
	run(0, "start", "--wave", "r1", "--role", "y", "--handle", "h2")
	run(0, "seal", "--wave", "r1")
	run(3, "return", "--wave", "r1", "--role", "x", "--outcome", "pass")
	run(0, "return", "--wave", "r1", "--role", "x", "--outcome", "findings")
	run(0, "return", "--wave", "r1", "--role", "y", "--outcome", "gap")
	run(0, "open", "--phase", "build", "--wave", "r2", "--role", "x")
	run(3, "abandon", "--wave", "r2", "--reason", "host session died")

	out.Reset()
	if code := RunMetrics(root, []string{"summary", "feat"}, out, errOut); code != 0 {
		t.Fatalf("summary code=%d\n%s", code, errOut)
	}
	got := out.String()
	for _, want := range []string{"outcomes: findings=1", "outcomes: gap=1", "abandon=1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary lacks %q:\n%s", want, got)
		}
	}
}
