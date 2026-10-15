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
