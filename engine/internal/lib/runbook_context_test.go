package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

func TestContextSyncUpsertsManagedBlockOnly(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte("demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "principles.md"), []byte("# Principles\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(project, "AGENTS.md")
	if err := os.WriteFile(path, []byte("# Team rules\n\nKeep this.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := Context(root, []string{"sync", "AGENTS.md"}, stdout, stderr); code != 0 {
		t.Fatalf("context sync = %d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}
	got := testutil.ReadFile(t, path)
	for _, want := range []string{"# Team rules", "<!-- DEVRITES START -->", "Active workspace: `.devrites/work/demo/`", "Project principles"} {
		if !strings.Contains(got, want) {
			t.Fatalf("synced AGENTS.md missing %q:\n%s", want, got)
		}
	}
	if strings.Count(got, "<!-- DEVRITES START -->") != 1 {
		t.Fatalf("expected one managed block, got:\n%s", got)
	}

	if code := Context(root, []string{"sync", "AGENTS.md"}, stdout, stderr); code != 0 {
		t.Fatalf("second context sync = %d", code)
	}
	got = testutil.ReadFile(t, path)
	if strings.Count(got, "<!-- DEVRITES START -->") != 1 {
		t.Fatalf("second sync duplicated block:\n%s", got)
	}
}

func TestContextSyncRejectsUnsafePath(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if code := Context(root, []string{"sync", "../AGENTS.md"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 2 {
		t.Fatalf("unsafe path code = %d, want 2", code)
	}
}

func TestContextSyncDoesNotOverwriteUnreadableTarget(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(project, "AGENTS.md")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	stderr := &bytes.Buffer{}
	if code := Context(root, []string{"sync", "AGENTS.md"}, &bytes.Buffer{}, stderr); code != 1 {
		t.Fatalf("unreadable target code = %d, want 1; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "read existing context file") {
		t.Fatalf("stderr=%q, want target read error", stderr.String())
	}
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		t.Fatalf("target was replaced: info=%v err=%v", info, err)
	}
}

func TestContextSyncRefusesExternalWorkspaceOverride(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(project, "AGENTS.md")
	if err := os.WriteFile(target, []byte("unchanged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_WORKSPACE", filepath.Join(t.TempDir(), "feature"))
	stderr := &bytes.Buffer{}
	if code := Context(root, []string{"sync", "AGENTS.md"}, &bytes.Buffer{}, stderr); code != 3 {
		t.Fatalf("external workspace context sync = %d, want refusal\n%s", code, stderr)
	}
	if got := testutil.ReadFile(t, target); got != "unchanged\n" {
		t.Fatalf("refused context sync mutated target:\n%s", got)
	}
}

func TestRunbookValidateAndGateResume(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	rb := filepath.Join(root, "runbooks")
	if err := os.MkdirAll(rb, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rb, "demo.yaml"), []byte("steps:\n  - shell: echo before\n  - gate: review\n  - shell: echo after\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := Runbook(root, []string{"validate", "demo"}, stdout, stderr); code != 0 {
		t.Fatalf("validate = %d\n%s%s", code, stdout, stderr)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Runbook(root, []string{"run", "demo"}, stdout, stderr); code != 3 {
		t.Fatalf("run should pause at gate, got %d\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout.String(), "before") || strings.Contains(stdout.String(), "after") {
		t.Fatalf("gate should stop before final shell:\n%s", stdout.String())
	}
	runsDir := filepath.Join(root, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected one run state, entries=%v err=%v", entries, err)
	}
	id := entries[0].Name()
	stdout.Reset()
	stderr.Reset()
	if code := Runbook(root, []string{"resume", id}, stdout, stderr); code != 0 {
		t.Fatalf("resume = %d\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout.String(), "after") || !strings.Contains(stdout.String(), "completed") {
		t.Fatalf("resume did not finish runbook:\n%s", stdout.String())
	}
}

func TestRunbookDryRunDoesNotExecuteShell(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	rb := filepath.Join(root, "runbooks")
	if err := os.MkdirAll(rb, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(project, "marker")
	if err := os.WriteFile(filepath.Join(rb, "demo.yaml"), []byte("steps:\n  - shell: touch marker\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := Runbook(root, []string{"run", "demo", "--dry-run"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("dry run code = %d", code)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("dry run executed shell step")
	}
}

func TestRunbookReportsCompletionStateWriteFailure(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "demo.yaml")
	if err := os.WriteFile(path, []byte("steps:\n  - rite: build\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "runs"), []byte("blocks state directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := executeRunbook(root, path, 0, "test-run", false, stdout, stderr); code != 1 {
		t.Fatalf("executeRunbook code = %d, want 1; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if strings.Contains(stdout.String(), "completed") || !strings.Contains(stderr.String(), "runbook:") {
		t.Fatalf("completion write failure was misreported; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
