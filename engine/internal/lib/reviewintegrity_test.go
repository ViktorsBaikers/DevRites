package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// writeReview drops a review.md into <root>/work/<slug>/ and returns the root.
func writeReview(t *testing.T, root, slug, body string) {
	t.Helper()
	dir := filepath.Join(root, "work", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "review.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write review.md: %v", err)
	}
}

func runReviewIntegrity(t *testing.T, root, slug string) (int, string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := ReviewIntegrity(root, []string{slug}, stdout, stderr)
	return code, stdout.String() + stderr.String()
}

func TestReviewIntegritySilentAxisFails(t *testing.T) {
	root := t.TempDir()
	// Code review axis carries a real finding; Spec axis is an all-zero rubber stamp.
	writeReview(t, root, "feat", `# Review

## Spec
Everything matches the spec. Nothing to report.
Critical 0 / Important 0 / Suggestion 0

## Code review
- **Important** — off-by-one in pagination at api/list.go:42.
`)
	code, out := runReviewIntegrity(t, root, "feat")
	if code != 1 {
		t.Fatalf("silent Spec axis should fail with 1, got %d\n%s", code, out)
	}
	if !contains(out, "Spec: SILENT") {
		t.Fatalf("want Spec flagged silent, got:\n%s", out)
	}
	if !contains(out, "Code review: findings present") {
		t.Fatalf("want Code review recognised as having findings, got:\n%s", out)
	}
}

func TestReviewIntegrityJustifiedCleanPasses(t *testing.T) {
	root := t.TempDir()
	writeReview(t, root, "feat", `# Review

## Spec
No-findings: ran the missing/partial/incorrect passes against each AC; every
criterion is implemented and no scope creep found.

## Code review
- **Nit** — rename tmp to buf at util.go:9.
`)
	code, out := runReviewIntegrity(t, root, "feat")
	if code != 0 {
		t.Fatalf("justified clean axis should pass, got %d\n%s", code, out)
	}
	if !contains(out, "Spec: clean — no-findings justification present") {
		t.Fatalf("want Spec recognised as justified, got:\n%s", out)
	}
}

func TestReviewIntegrityBothWithFindingsPasses(t *testing.T) {
	root := t.TempDir()
	writeReview(t, root, "feat", `## Spec
- **Critical** — AC-003 unimplemented.

## Code review
- **Important** — missing error path.
`)
	if code, out := runReviewIntegrity(t, root, "feat"); code != 0 {
		t.Fatalf("both-with-findings should pass, got %d\n%s", code, out)
	}
}

func TestReviewIntegrityNoReviewPasses(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "work", "feat"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, out := runReviewIntegrity(t, root, "feat"); code != 0 {
		t.Fatalf("absent review.md should pass, got %d\n%s", code, out)
	}
}

func TestReviewIntegrityFreeformPasses(t *testing.T) {
	root := t.TempDir()
	// No per-axis headings — can't assess mechanically, must not false-positive.
	writeReview(t, root, "feat", "# Review\n\nLooks fine overall, shipping.\n")
	if code, out := runReviewIntegrity(t, root, "feat"); code != 0 {
		t.Fatalf("freeform review should pass, got %d\n%s", code, out)
	}
}
