package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTimelineLogAndList(t *testing.T) {
	t.Setenv("DEVRITES_NOW", "2026-07-09T10:11:12Z")
	root := t.TempDir()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	code := Timeline(root, []string{"log", "completed", "--skill", "rite-review", "--slug", "auth", "--outcome", "ok", "--decision", "ship"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("timeline log failed: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(root, "timeline.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{`"ts":"2026-07-09T10:11:12Z"`, `"event":"completed"`, `"skill":"rite-review"`, `"slug":"auth"`, `"decision":"ship"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("timeline entry missing %s: %s", want, got)
		}
	}

	stdout.Reset()
	stderr.Reset()
	if code := Timeline(root, []string{"list", "--limit", "1"}, stdout, stderr); code != 0 {
		t.Fatalf("timeline list failed: %d %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"event":"completed"`) {
		t.Fatalf("timeline list did not print entry: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Timeline(root, []string{"log", "completed", "--skill"}, stdout, stderr); code != 2 {
		t.Fatalf("missing timeline option value should be usage error, got %d", code)
	}
}

func TestHealthRecordValidatesScore(t *testing.T) {
	t.Setenv("DEVRITES_NOW", "2026-07-09")
	root := t.TempDir()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	code := Health(root, []string{"record", "9.5", "ci green", "--note", "staticcheck+tests"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("health record failed: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(root, "health-history.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{`"score":9.5`, `"label":"ci green"`, `"note":"staticcheck+tests"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("health entry missing %s: %s", want, got)
		}
	}

	stdout.Reset()
	stderr.Reset()
	if code := Health(root, []string{"record", "11", "too high"}, stdout, stderr); code != 2 {
		t.Fatalf("out-of-range score should be usage error, got %d", code)
	}
}

func TestReviewFingerprintsStableAndWritable(t *testing.T) {
	root := t.TempDir()
	writeReview(t, root, "feat", `# Review

## Spec
- **Critical** — AC-003 is unproven.

## Code review
- **Important** — missing error handling.`)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := ReviewFingerprints(root, []string{"--write", "feat"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("review-fingerprints failed: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 fingerprints, got %d: %q", len(lines), stdout.String())
	}
	firstID := strings.Fields(lines[0])[0]
	if len(firstID) != 12 {
		t.Fatalf("fingerprint id should be 12 hex chars, got %q", firstID)
	}
	stdout.Reset()
	if code := ReviewFingerprints(root, []string{"feat"}, stdout, stderr); code != 0 {
		t.Fatalf("second fingerprint run failed: %d", code)
	}
	if secondID := strings.Fields(strings.Split(strings.TrimSpace(stdout.String()), "\n")[0])[0]; secondID != firstID {
		t.Fatalf("fingerprint changed across identical input: %s vs %s", firstID, secondID)
	}
	data, err := os.ReadFile(filepath.Join(root, "work", "feat", "review-fingerprints.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"severity":"Critical"`) || !strings.Contains(string(data), `"severity":"Important"`) {
		t.Fatalf("written JSONL missing severities: %s", string(data))
	}
}
