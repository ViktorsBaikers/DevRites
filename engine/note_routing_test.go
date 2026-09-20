package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func noteWorkspace(t *testing.T) (project, root string) {
	t.Helper()
	project = t.TempDir()
	root = filepath.Join(project, ".devrites")
	workspace := filepath.Join(root, "work", "feature")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	writeBasenameFile(t, workspace, "state.md", "| schema | 4 |\n")
	t.Setenv("DEVRITES_ROOT", root)
	return project, root
}

func TestNoteLifecycleAddCheckRepairRm(t *testing.T) {
	project, _ := noteWorkspace(t)
	writeBasenameFile(t, project, "pool.go", "package src\n\nconst maxConns = 4\n")

	var stdout, stderr bytes.Buffer
	args := []string{"note", "add", "feature", "pool.go", "maxConns = 4", "pool cap rationale", "fairness not memory"}
	if code := run(args, strings.NewReader(""), &stdout, &stderr); code != exitOK {
		t.Fatalf("add code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "added: NOTE-001") {
		t.Fatalf("add stdout=%q", stdout.String())
	}

	// A quote that does not resolve is refused at write time.
	stdout.Reset()
	stderr.Reset()
	args = []string{"note", "add", "feature", "pool.go", "nope = 1", "bad note"}
	if code := run(args, strings.NewReader(""), &stdout, &stderr); code != exitBlocked {
		t.Fatalf("add bad quote code=%d", code)
	}

	stdout.Reset()
	if code := run([]string{"note", "check", "feature"}, strings.NewReader(""), &stdout, &stderr); code != exitOK {
		t.Fatalf("check code=%d", code)
	}
	if !strings.Contains(stdout.String(), "result: exact") {
		t.Fatalf("check stdout=%q", stdout.String())
	}

	// Relocate the anchor to another file: check reports moved, --repair rewrites.
	writeBasenameFile(t, project, "pool.go", "package src\n\nconst maxConns = 8\n")
	if err := os.MkdirAll(filepath.Join(project, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeBasenameFile(t, filepath.Join(project, "config"), "limits.go", "package config\nconst maxConns = 4\n")
	stdout.Reset()
	if code := run([]string{"note", "check", "feature"}, strings.NewReader(""), &stdout, &stderr); code != exitBlocked {
		t.Fatalf("moved check code=%d", code)
	}
	if !strings.Contains(stdout.String(), "NOTE-001 moved -> config/limits.go") {
		t.Fatalf("moved stdout=%q", stdout.String())
	}
	stdout.Reset()
	if code := run([]string{"note", "check", "feature", "--repair"}, strings.NewReader(""), &stdout, &stderr); code != exitOK {
		t.Fatalf("repair code=%d stdout=%q", code, stdout.String())
	}
	raw, err := os.ReadFile(filepath.Join(project, ".devrites", "work", "feature", "notes.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "SUBJECT: config/limits.go") {
		t.Fatalf("repair did not rewrite subject: %q", raw)
	}

	stdout.Reset()
	if code := run([]string{"note", "rm", "feature", "NOTE-001"}, strings.NewReader(""), &stdout, &stderr); code != exitOK {
		t.Fatalf("rm code=%d stderr=%q", code, stderr.String())
	}
	raw, _ = os.ReadFile(filepath.Join(project, ".devrites", "work", "feature", "notes.md"))
	if strings.Contains(string(raw), "NOTE-001") {
		t.Fatal("rm left the note in place")
	}
}

func TestNoteHelpRoutes(t *testing.T) {
	for _, args := range [][]string{{"note", "help"}, {"note", "--help"}} {
		var stdout, stderr bytes.Buffer
		if code := run(args, strings.NewReader(""), &stdout, &stderr); code != exitOK || !strings.Contains(stdout.String(), "note <add|list|check|rm>") {
			t.Fatalf("%v code=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
	}
}
