package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestReviewersListJSONInstances(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "config.json"), []byte(`{
  "review": {
    "reviewer_instances": {
      "codex-deep": {"cli":"codex", "model":"o3"},
      "claude-fast": {"cli":"claude", "agent":"review"}
    }
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := Reviewers(root, []string{"list"}, stdout, stderr); code != 0 {
		t.Fatalf("reviewers list code=%d\n%s", code, stderr.String())
	}
	for _, want := range []string{"claude-fast -> claude agent=review", "codex-deep -> codex model=o3"} {
		if !contains(stdout.String(), want) {
			t.Fatalf("want %q in stdout:\n%s", want, stdout.String())
		}
	}
}

func TestReviewersRejectsArbitraryCLI(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "config.json"), []byte(`{"reviewer_instances":{"evil":{"cli":"sh"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := Reviewers(root, []string{"list"}, stdout, stderr); code != 1 {
		t.Fatalf("reviewers list should fail, got %d\n%s%s", code, stdout, stderr)
	}
	if !contains(stderr.String(), "not allowed") {
		t.Fatalf("want allowlist error, got:\n%s", stderr.String())
	}
}
