package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecisionsSearchFindsArchivedDecision(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "archive", "auth-refresh", "decisions.md")
	mustWrite(t, p, "# Decisions\n- Use short-lived refresh tokens for auth sessions.\n")
	var out, err bytes.Buffer
	if code := Decisions(root, []string{"search", "refresh", "tokens"}, &out, &err); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, err.String())
	}
	if !strings.Contains(out.String(), "auth-refresh") || !strings.Contains(out.String(), "refresh tokens") {
		t.Fatalf("missing decision hit:\n%s", out.String())
	}
}

func TestSecretScanBlocksHighSeverityTouchedFile(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	mustWrite(t, filepath.Join(root, "ACTIVE"), "leak\n")
	mustWrite(t, filepath.Join(root, "work", "leak", "touched-files.md"), "- secrets.txt\n")
	mustWrite(t, filepath.Join(project, "secrets.txt"), "token=ghp_abcdefghijklmnopqrstuvwxyzABCDE12345\n")
	var out, err bytes.Buffer
	if code := SecretScan(root, nil, &out, &err); code != 3 {
		t.Fatalf("want block rc=3 got %d stdout=%s stderr=%s", code, out.String(), err.String())
	}
}

func TestSpecDedupeSearchesScratchPRDs(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	mustWrite(t, filepath.Join(project, ".scratch", "webhooks", "PRD.md"), "# Webhook retries\nAdd retry backoff for failed webhooks.\n")
	var out, err bytes.Buffer
	if code := SpecDedupe(root, []string{"webhook", "retry", "backoff"}, &out, &err); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, err.String())
	}
	if !strings.Contains(out.String(), ".scratch/webhooks/PRD.md") {
		t.Fatalf("missing dedupe hit:\n%s", out.String())
	}
}

func mustWrite(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
