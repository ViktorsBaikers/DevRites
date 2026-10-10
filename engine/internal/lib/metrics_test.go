package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMetricsSummaryFailsOnOversizedLine(t *testing.T) {
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
	if code == 0 {
		t.Fatalf("truncated ledger reported as complete:\n%s", stdout)
	}
	if !strings.Contains(stderr.String(), "ledger unreadable after 1 events") {
		t.Fatalf("stderr missing truncation report: %q", stderr)
	}
}
