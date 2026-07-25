package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestRootModeForCoversReadAndWriteSurfaces(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
		want    rootMode
	}{
		{name: "unrelated command", command: "doctor", want: rootUnused},
		{name: "first task", command: "first-task", want: rootLenient},
		{name: "spec dedupe", command: "spec-dedupe", want: rootLenient},
		{name: "footprint render", command: "footprint", args: []string{"render"}, want: rootLenient},
		{name: "footprint roster", command: "footprint", args: []string{"roster"}, want: rootLenient},
		{name: "footprint log", command: "footprint", args: []string{"log"}, want: rootStrict},
		{name: "evidence fresh", command: "evidence-fresh", want: rootLenient},
		{name: "coverage", command: "coverage", want: rootLenient},
		{name: "doubt coverage", command: "doubt-coverage", want: rootLenient},
		{name: "budget", command: "budget", want: rootLenient},
		{name: "preamble", command: "preamble", want: rootLenient},
		{name: "progress", command: "progress", want: rootLenient},
		{name: "stuck check", command: "stuck", args: []string{"check"}, want: rootLenient},
		{name: "stuck log", command: "stuck", args: []string{"log"}, want: rootStrict},
		{name: "recovery route", command: "recovery", args: []string{"route"}, want: rootLenient},
		{name: "recovery check", command: "recovery", args: []string{"check"}, want: rootLenient},
		{name: "recovery fingerprint", command: "recovery", args: []string{"fingerprint"}, want: rootLenient},
		{name: "recovery record", command: "recovery", args: []string{"record"}, want: rootStrict},
		{name: "recovery clear", command: "recovery", args: []string{"clear"}, want: rootStrict},
		{name: "build readiness", command: "build-readiness", want: rootLenient},
		{name: "readiness digest", command: "readiness-digest", want: rootLenient},
		{name: "clarify enter", command: "clarify-return", args: []string{"enter"}, want: rootStrict},
		{name: "clarify restore", command: "clarify-return", args: []string{"restore"}, want: rootStrict},
		{name: "analyze", command: "analyze", want: rootLenient},
		{name: "mutation gate", command: "mutation-gate", want: rootLenient},
		{name: "test integrity", command: "test-integrity", want: rootLenient},
		{name: "review integrity", command: "review-integrity", want: rootLenient},
		{name: "reconcile snapshot", command: "reconcile", args: []string{"snapshot"}, want: rootStrict},
		{name: "reconcile check", command: "reconcile", args: []string{"check"}, want: rootStrict},
		{name: "reconcile close", command: "reconcile", args: []string{"close"}, want: rootStrict},
		{name: "resolve", command: "resolve", want: rootStrict},
		{name: "resolve next qid", command: "resolve", args: []string{"next-qid"}, want: rootLenient},
		{name: "close out", command: "close-out", want: rootStrict},
		{name: "archive search", command: "archive-search", want: rootLenient},
		{name: "decisions search", command: "decisions", args: []string{"search"}, want: rootLenient},
		{name: "decisions index", command: "decisions", args: []string{"index"}, want: rootStrict},
		{name: "ledger diff", command: "ledger", args: []string{"diff"}, want: rootLenient},
		{name: "ledger validate", command: "ledger", args: []string{"validate"}, want: rootLenient},
		{name: "ledger list", command: "ledger", args: []string{"list"}, want: rootLenient},
		{name: "ledger show", command: "ledger", args: []string{"show"}, want: rootLenient},
		{name: "ledger sync", command: "ledger", args: []string{"sync"}, want: rootStrict},
		{name: "ledger json before sync", command: "ledger", args: []string{"--json", "sync"}, want: rootStrict},
		{name: "learnings list", command: "learnings", args: []string{"list"}, want: rootLenient},
		{name: "learnings top", command: "learnings", args: []string{"top"}, want: rootLenient},
		{name: "learnings mine", command: "learnings", args: []string{"mine"}, want: rootLenient},
		{name: "learnings nudge", command: "learnings", args: []string{"nudge"}, want: rootLenient},
		{name: "learnings add", command: "learnings", args: []string{"add"}, want: rootStrict},
		{name: "timeline list", command: "timeline", args: []string{"list"}, want: rootLenient},
		{name: "timeline report", command: "timeline", args: []string{"report"}, want: rootLenient},
		{name: "timeline log", command: "timeline", args: []string{"log"}, want: rootStrict},
		{name: "timeline purge", command: "timeline", args: []string{"purge"}, want: rootStrict},
		{name: "health list", command: "health", args: []string{"list"}, want: rootLenient},
		{name: "health default", command: "health", want: rootStrict},
		{name: "health run", command: "health", args: []string{"run"}, want: rootStrict},
		{name: "health check", command: "health", args: []string{"check"}, want: rootStrict},
		{name: "health record", command: "health", args: []string{"record"}, want: rootStrict},
		{name: "config", command: "config", args: []string{"get"}, want: rootLenient},
		{name: "reviewers", command: "reviewers", args: []string{"list"}, want: rootLenient},
		{name: "outside voice", command: "outside-voice", want: rootLenient},
		{name: "docs stale", command: "docs-stale", want: rootLenient},
		{name: "secret scan", command: "secret-scan", want: rootLenient},
		{name: "review fingerprints read", command: "review-fingerprints", want: rootLenient},
		{name: "review fingerprints write", command: "review-fingerprints", args: []string{"demo", "--write"}, want: rootStrict},
		{name: "reviewer stats report", command: "reviewer-stats", args: []string{"report"}, want: rootLenient},
		{name: "reviewer stats record", command: "reviewer-stats", args: []string{"record"}, want: rootStrict},
		{name: "lanes", command: "lanes", args: []string{"plan"}, want: rootLenient},
		{name: "forge missing verb", command: "forge", want: rootStrict},
		{name: "forge plan", command: "forge", args: []string{"plan"}, want: rootStrict},
		{name: "forge record", command: "forge", args: []string{"record"}, want: rootStrict},
		{name: "forge extract", command: "forge", args: []string{"extract"}, want: rootStrict},
		{name: "forge merge", command: "forge", args: []string{"merge"}, want: rootStrict},
		{name: "forge cleanup", command: "forge", args: []string{"cleanup"}, want: rootStrict},
		{name: "forge reap", command: "forge", args: []string{"reap"}, want: rootStrict},
		{name: "extensions list", command: "extensions", args: []string{"list"}, want: rootLenient},
		{name: "extensions validate", command: "extensions", args: []string{"validate"}, want: rootLenient},
		{name: "extensions sync", command: "extensions", args: []string{"sync"}, want: rootStrict},
		{name: "overrides", command: "overrides", args: []string{"validate"}, want: rootLenient},
		{name: "context show", command: "context", args: []string{"show"}, want: rootLenient},
		{name: "context sync", command: "context", args: []string{"sync"}, want: rootStrict},
		{name: "runbook list", command: "runbook", args: []string{"list"}, want: rootLenient},
		{name: "runbook validate", command: "runbook", args: []string{"validate"}, want: rootLenient},
		{name: "runbook run", command: "runbook", args: []string{"run"}, want: rootStrict},
		{name: "runbook resume", command: "runbook", args: []string{"resume"}, want: rootStrict},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := rootModeFor(test.command, test.args); got != test.want {
				t.Fatalf("rootModeFor(%q, %q) = %d, want %d", test.command, test.args, got, test.want)
			}
		})
	}
}

func TestStrictForgeRefusesParentWorkspaceAcrossNestedRepository(t *testing.T) {
	parent := t.TempDir()
	if err := os.Mkdir(filepath.Join(parent, ".devrites"), 0o755); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(parent, "nested")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	initRootRoutingGitRepo(t, child)
	withRootRoutingCWD(t, child)
	t.Setenv("DEVRITES_ROOT", "")
	t.Setenv("DEVRITES_WORKSPACE", "")

	var stdout, stderr bytes.Buffer
	code := run([]string{"forge", "plan"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitBlocked {
		t.Fatalf("forge plan exit = %d, want %d; stdout=%q stderr=%q", code, exitBlocked, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "unsafe DevRites root") {
		t.Fatalf("stderr = %q, want unsafe-root refusal", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(child, ".devrites")); !os.IsNotExist(err) {
		t.Fatalf("strict refusal must not create nested workspace; stat err=%v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"context", "show"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("context show exit = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "DRV-ROOT-OUTSIDE-GIT") {
		t.Fatalf("context show should diagnose the unsafe parent root; stdout=%q", stdout.String())
	}
}

func TestStrictWriterRefusesExternalWorkspaceBeforeMutation(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	if err := os.MkdirAll(filepath.Join(root, "work", "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	initRootRoutingGitRepo(t, project)
	withRootRoutingCWD(t, project)
	t.Setenv("DEVRITES_ROOT", "")
	t.Setenv("DEVRITES_WORKSPACE", t.TempDir())
	before := rootRoutingTree(t, root)

	var stdout, stderr bytes.Buffer
	code := run([]string{"timeline", "log", "rite-build"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitBlocked {
		t.Fatalf("timeline log exit = %d, want %d; stdout=%q stderr=%q", code, exitBlocked, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "unsafe DevRites root") {
		t.Fatalf("stderr = %q, want unsafe-root refusal", stderr.String())
	}
	if after := rootRoutingTree(t, root); !reflect.DeepEqual(after, before) {
		t.Fatalf("strict refusal mutated workspace tree: before=%q after=%q", before, after)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(
		[]string{"timeline", "purge", "--run", "drv-run-v1:0123456789abcdef0123456789abcdef"},
		strings.NewReader(""),
		&stdout,
		&stderr,
	)
	if code != exitBlocked {
		t.Fatalf("timeline purge exit = %d, want %d; stdout=%q stderr=%q", code, exitBlocked, stdout.String(), stderr.String())
	}
	if after := rootRoutingTree(t, root); !reflect.DeepEqual(after, before) {
		t.Fatalf("unsafe timeline purge mutated workspace tree: before=%q after=%q", before, after)
	}
}

func TestStrictWriterRefusesWorkspaceOverrideWithoutRoot(t *testing.T) {
	project := t.TempDir()
	initRootRoutingGitRepo(t, project)
	withRootRoutingCWD(t, project)
	t.Setenv("DEVRITES_ROOT", "")
	t.Setenv("DEVRITES_WORKSPACE", t.TempDir())

	var stdout, stderr bytes.Buffer
	code := run([]string{"learnings", "add", "demo", "must not escape"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitBlocked {
		t.Fatalf("learnings add exit = %d, want %d; stdout=%q stderr=%q", code, exitBlocked, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "DEVRITES_WORKSPACE requires a selected DevRites root") {
		t.Fatalf("stderr = %q, want workspace-without-root refusal", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(project, ".devrites")); !os.IsNotExist(err) {
		t.Fatalf("strict refusal must not create a root; stat err=%v", err)
	}
}

func TestStrictWriterRefusesMissingExplicitRoot(t *testing.T) {
	project := t.TempDir()
	initRootRoutingGitRepo(t, project)
	withRootRoutingCWD(t, project)
	root := filepath.Join(project, ".devrites")
	t.Setenv("DEVRITES_ROOT", root)
	t.Setenv("DEVRITES_WORKSPACE", "")

	var stdout, stderr bytes.Buffer
	code := run([]string{"learnings", "add", "demo", "must not bootstrap"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitBlocked {
		t.Fatalf("learnings add exit = %d, want %d; stdout=%q stderr=%q", code, exitBlocked, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "unsafe DevRites root") {
		t.Fatalf("stderr = %q, want invalid explicit-root refusal", stderr.String())
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("strict refusal must not create explicit missing root; stat err=%v", err)
	}
}

func initRootRoutingGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", "-q", dir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init %s: %v\n%s", dir, err, output)
	}
}

func withRootRoutingCWD(t *testing.T, dir string) {
	t.Helper()
	before, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(before); err != nil {
			t.Errorf("restore cwd: %v", err)
		}
	})
}

func rootRoutingTree(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	return paths
}
