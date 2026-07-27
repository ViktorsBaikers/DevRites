package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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

func TestDecisionsIndexAcceptsLongDecisionLine(t *testing.T) {
	root := t.TempDir()
	decision := "- " + strings.Repeat("a", 128*1024) + " searchable-tail\n"
	mustWrite(t, filepath.Join(root, "archive", "large", "decisions.md"), decision)
	var out, err bytes.Buffer

	if code := Decisions(root, []string{"index"}, &out, &err); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, err.String())
	}
	indexed, readErr := os.ReadFile(filepath.Join(root, "decisions-index.jsonl"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Contains(indexed, []byte("searchable-tail")) {
		t.Fatal("long decision was silently omitted from the index")
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

func TestSecretScanFailsClosedWhenGitIsUnavailable(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	t.Chdir(project)
	t.Setenv("PATH", t.TempDir())

	var out, err bytes.Buffer
	if code := SecretScan(root, nil, &out, &err); code != 2 {
		t.Fatalf("want rc=2 got %d stdout=%s stderr=%s", code, out.String(), err.String())
	}
	if !strings.Contains(err.String(), "cannot inspect changed paths") {
		t.Fatalf("missing fail-closed diagnostic: %s", err.String())
	}
}

func TestDocsStaleFailsClosedWhenGitIsUnavailable(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	t.Chdir(project)
	t.Setenv("PATH", t.TempDir())

	var out, err bytes.Buffer
	if code := DocsStale(root, nil, &out, &err); code != 2 {
		t.Fatalf("want rc=2 got %d stdout=%s stderr=%s", code, out.String(), err.String())
	}
	if !strings.Contains(err.String(), "cannot inspect changed paths") {
		t.Fatalf("missing fail-closed diagnostic: %s", err.String())
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

func TestBoundedCommandOutputStopsLongRunningProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test command uses POSIX sh")
	}

	started := time.Now()
	if _, err := boundedCommandOutput(50*time.Millisecond, "", "sh", "-c", "sleep 2"); err == nil {
		t.Fatal("expected timed-out command to return an error")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("timed-out command took %s", elapsed)
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
