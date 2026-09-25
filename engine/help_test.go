package main

import (
	"bytes"
	"strings"
	"testing"
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
