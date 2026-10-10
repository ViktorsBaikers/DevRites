package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/notes"
)

func TestCommandHelp(t *testing.T) {
	t.Setenv("DEVRITES_ROOT", "")
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"--help"}, want: "Usage:"},
		{args: []string{"-h"}, want: "Usage:"},
		{args: []string{"help"}, want: "Usage:"},
		{args: []string{"check", "--help"}, want: "check candidate <slug>"},
		{args: []string{"check", "help"}, want: "check <candidate|readiness|seal|path-disjoint|task-graph|slice|diff-scope|skill-trust|indexes|regression|drift|windows|dup>"},
		{args: []string{"check", "candidate", "--help"}, want: "check candidate <slug>"},
		{args: []string{"check", "readiness", "-h"}, want: "check readiness --emit-binding <slug>"},
		{args: []string{"check", "seal", "--help"}, want: "check seal <slug>"},
		{args: []string{"check", "path-disjoint", "--help"}, want: "check path-disjoint"},
		{args: []string{"check", "task-graph", "--help"}, want: "check task-graph <slug>"},
		{args: []string{"check", "skill-trust", "--help"}, want: "check skill-trust <path>"},
		{args: []string{"check", "indexes", "--help"}, want: "check indexes [--root <dir>]"},
		{args: []string{"check", "dup", "--help"}, want: "check dup [slug]"},
		{args: []string{"check", "drift", "--help"}, want: "check drift <slug>"},
		{args: []string{"detect", "--help"}, want: "detect commands [--root <dir>]"},
		{args: []string{"state", "--help"}, want: "state <resolve|merge-manifest|close>"},
		{args: []string{"state", "-h"}, want: "state resolve"},
		{args: []string{"state", "help"}, want: "state close <slug>"},
		{args: []string{"state", "resolve", "--help"}, want: `state resolve <qid> "<answer>"`},
		{args: []string{"state", "merge-manifest", "--help"}, want: "state merge-manifest <slug>"},
		{args: []string{"state", "close", "--help"}, want: "state close <slug>"},
		{args: []string{"parallel", "--help"}, want: "usage: parallel <subcommand>"},
		{args: []string{"parallel", "create", "--help"}, want: "usage: parallel create --root --slug --batch --base --json"},
		{args: []string{"parallel", "create", "-h"}, want: "parallel create"},
		{args: []string{"parallel", "select", "--help"}, want: "parallel select --cap"},
		{args: []string{"observe", "--help"}, want: "observe summary"},
		{args: []string{"observe", "summary", "--help"}, want: "observe summary"},
		{args: []string{"observe", "slice", "--help"}, want: "observe slice <slug> <SLICE-ID>"},
		{args: []string{"orient", "--help"}, want: "orient [slug]"},
		{args: []string{"handoff", "--help"}, want: "handoff [slug]"},
		{args: []string{"next", "help"}, want: "devrites-engine next [slug]"},
		{args: []string{"dispatch", "help"}, want: "devrites-engine dispatch <slug>"},
		{args: []string{"secret-scan", "help"}, want: "secret-scan [--staged] [--stdin]"},
		{args: []string{"open-visual", "help"}, want: "open-visual <path-or-name>"},
		{args: []string{"claim", "--help"}, want: "claim <add|release|list|check>"},
		{args: []string{"claim", "help"}, want: "claim <add|release|list|check>"},
		{args: []string{"migrate", "--help"}, want: "migrate <slug>"},
		{args: []string{"secret-scan", "--help"}, want: "secret-scan [--staged] [--stdin]"},
		{args: []string{"open-visual", "--help"}, want: "open-visual <path-or-name>"},
		{args: []string{"overhaul", "--help"}, want: "overhaul records validate <run>"},
		{args: []string{"overhaul", "help"}, want: "overhaul snapshot capture <repo> <out>"},
		{args: []string{"version", "--help"}, want: "devrites-engine version"},
		{args: []string{"install", "--help"}, want: "usage: devrites-engine install"},
		{args: []string{"update", "--help"}, want: "usage: devrites-engine update"},
		{args: []string{"uninstall", "--help"}, want: "usage: devrites-engine uninstall"},
	}
	for _, test := range tests {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(test.args, strings.NewReader(""), &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("run(%q) = %d, want %d; stdout=%q stderr=%q", test.args, code, exitOK, stdout.String(), stderr.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("run(%q) wrote stderr %q", test.args, stderr.String())
			}
			if !strings.Contains(stdout.String(), test.want) {
				t.Fatalf("run(%q) stdout=%q, want %q", test.args, stdout.String(), test.want)
			}
			if strings.Contains(stdout.String(), "unknown") {
				t.Fatalf("run(%q) treated help as unknown: %q", test.args, stdout.String())
			}
		})
	}
}

func TestCommandHelpDoesNotRequireWorkspace(t *testing.T) {
	t.Setenv("DEVRITES_ROOT", "")
	for _, args := range [][]string{
		{"state", "--help"},
		{"check", "candidate", "--help"},
		{"orient", "--help"},
		{"migrate", "--help"},
		{"parallel", "create", "--help"},
	} {
		var stdout, stderr bytes.Buffer
		code := run(args, strings.NewReader(""), &stdout, &stderr)
		if code != exitOK || stderr.Len() != 0 || !strings.Contains(stdout.String(), "usage:") {
			t.Fatalf("run(%q) code=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
		if strings.Contains(stdout.String()+stderr.String(), "root selection") {
			t.Fatalf("run(%q) required a workspace for help", args)
		}
	}
}

func TestUnknownCommandHelpStillUnknown(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"frobnicate", "--help"}
	if code := run(args, strings.NewReader(""), &stdout, &stderr); code != exitUsage {
		t.Fatalf("run(%q) = %d, want %d", args, code, exitUsage)
	}
	if !strings.Contains(stderr.String(), `unknown command "frobnicate"`) {
		t.Fatalf("stderr = %q, want unknown-command diagnostic", stderr.String())
	}
}

func TestHelpTokenInValuePositionDoesNotShortCircuit(t *testing.T) {
	for _, quote := range []string{"maxConns = 4", "--help", "-h", "-help"} {
		t.Run(quote, func(t *testing.T) {
			project, _ := noteWorkspace(t)
			writeBasenameFile(t, project, "pool.go", "package src\n\n// flags: --help -h -help\nconst maxConns = 4\n")
			var stdout, stderr bytes.Buffer
			args := []string{"note", "add", "feature", "pool.go", quote, "some title"}
			code := run(args, strings.NewReader(""), &stdout, &stderr)
			notes := filepath.Join(project, ".devrites", "work", "feature", "notes.md")
			if _, err := os.Stat(notes); err != nil {
				t.Fatalf("notes.md absent (code=%d stdout=%.80q stderr=%q)", code, stdout.String(), stderr.String())
			}
			if code != exitOK || !strings.Contains(stdout.String(), "added: NOTE-001") {
				t.Fatalf("code=%d stdout=%q", code, stdout.String())
			}
		})
	}
	for _, args := range [][]string{{"note", "add", "--help"}, {"note", "--help"}, {"check", "seal", "--help"}} {
		var stdout, stderr bytes.Buffer
		if code := run(args, strings.NewReader(""), &stdout, &stderr); code != exitOK || !strings.Contains(stdout.String(), "usage") && !strings.Contains(stdout.String(), "Usage") {
			t.Fatalf("%v code=%d stdout=%.80q", args, code, stdout.String())
		}
	}
}

func TestHelpTokenAsFlagValueOrOperandIsNotHelp(t *testing.T) {
	for _, tok := range []string{"--help", "-h", "-help"} {
		_, root := noteWorkspace(t)
		var stdout, stderr bytes.Buffer
		code := run([]string{"claim", "add", "--reason", tok, "--session", "s1", "src/a.go"}, strings.NewReader(""), &stdout, &stderr)
		if _, err := os.Stat(filepath.Join(root, "claims.jsonl")); err != nil || code != exitOK {
			t.Fatalf("reason=%s: code=%d claims absent (%v) stdout=%.60q", tok, code, err, stdout.String())
		}
		_, root = noteWorkspace(t)
		stdout.Reset()
		code = run([]string{"claim", "add", "--session", "s1", "src/a.go", tok}, strings.NewReader(""), &stdout, &stderr)
		if _, err := os.Stat(filepath.Join(root, "claims.jsonl")); err == nil {
			t.Fatalf("trailing %s wrote a claim", tok)
		}
		if code != exitOK || !strings.Contains(stdout.String(), "claim") {
			t.Fatalf("trailing %s code=%d stdout=%.60q", tok, code, stdout.String())
		}
	}
	_, root := noteWorkspace(t)
	var stdout, stderr bytes.Buffer
	if code := run([]string{"claim", "add", "--reason", "ordinary", "--session", "s1", "src/a.go"}, strings.NewReader(""), &stdout, &stderr); code != exitOK {
		t.Fatalf("control code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(root, "claims.jsonl")); err != nil {
		t.Fatal(err)
	}
}

func TestHelpFlagAfterOtherFlagsPrintsHelp(t *testing.T) {
	t.Setenv("DEVRITES_ROOT", "")
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"secret-scan", "--staged", "--help"}, "secret-scan [--staged] [--stdin]"},
		{[]string{"secret-scan", "--staged", "--stdin", "-h"}, "secret-scan [--staged] [--stdin]"},
		{[]string{"check", "indexes", "--json", "-help"}, "check indexes [--root <dir>]"},
		{[]string{"parallel", "create", "--json", "--help"}, "parallel create"},
	} {
		var stdout, stderr bytes.Buffer
		code := run(test.args, strings.NewReader(""), &stdout, &stderr)
		if code != exitOK || stderr.Len() != 0 || !strings.Contains(stdout.String(), test.want) {
			t.Fatalf("run(%q) code=%d stdout=%.80q stderr=%q", test.args, code, stdout.String(), stderr.String())
		}
	}
}

func TestUsageListsRuntimeFailureExit(t *testing.T) {
	if !strings.Contains(usage, "\n  1  runtime/I/O failure\n") {
		t.Fatalf("usage Exit codes block omits exit 1:\n%s", usage)
	}
}

func TestCLIDocsStateCloseExitCodes(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "docs", "cli.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	for _, want := range []string{
		"`state close` exits `4` for usage or a missing workspace",
		"`3` for a schema refusal",
		"`5` when an archive already exists",
		"`1` for an invalid workspace, archive directory, or ACTIVE cursor",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("docs/cli.md omits %q", want)
		}
	}
}

func TestNoteHelpAndErrorPathShareLegendAndExitCodes(t *testing.T) {
	const grades = "Grades: exact | moved | stale | ambiguous | lost"
	const exits = "Exit codes: 0 ok, 2 usage, 3 blocked"
	for _, args := range [][]string{
		{"note", "--help"},
		{"note", "help"},
		{"note", "add", "--help"},
	} {
		var stdout, stderr bytes.Buffer
		code := run(args, strings.NewReader(""), &stdout, &stderr)
		if code != exitOK || stderr.Len() != 0 {
			t.Fatalf("run(%q) code=%d stderr=%q", args, code, stderr.String())
		}
		for _, want := range []string{grades, exits} {
			if !strings.Contains(stdout.String(), want) {
				t.Errorf("run(%q) help omits %q:\n%s", args, want, stdout.String())
			}
		}
	}
	var stdout, stderr bytes.Buffer
	if code := notes.Run(t.TempDir(), nil, &stdout, &stderr); code != notes.ExitUsage {
		t.Fatalf("notes.Run with no args = %d, want usage", code)
	}
	for _, want := range []string{grades, exits} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("note usage error omits %q:\n%s", want, stderr.String())
		}
	}
}
