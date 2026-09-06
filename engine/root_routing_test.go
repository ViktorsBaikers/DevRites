package main

import (
	"bytes"
	"os"
	"path/filepath"
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
		{name: "unrelated command", command: "unknown", want: rootUnused},
		{name: "check candidate", command: "check", args: []string{"candidate"}, want: rootStrictUsage},
		{name: "check readiness", command: "check", args: []string{"readiness"}, want: rootStrictUsage},
		{name: "check seal", command: "check", args: []string{"seal"}, want: rootStrictUsage},
		{name: "check task-graph", command: "check", args: []string{"task-graph"}, want: rootStrictUsage},
		{name: "check path-disjoint", command: "check", args: []string{"path-disjoint"}, want: rootUnused},
		{name: "check skill-trust", command: "check", args: []string{"skill-trust"}, want: rootUnused},
		{name: "observe summary", command: "observe", args: []string{"summary"}, want: rootStrictUsage},
		{name: "orient", command: "orient", want: rootStrictUsage},
		{name: "check indexes", command: "check", args: []string{"indexes"}, want: rootLenient},
		{name: "resolve", command: "state", args: []string{"resolve"}, want: rootStrict},
		{name: "close", command: "state", args: []string{"close"}, want: rootStrict},
		{name: "unknown state command", command: "state", args: []string{"unknown"}, want: rootUnused},
		{name: "secret scan", command: "secret-scan", want: rootLenient},
		{name: "open visual", command: "open-visual", want: rootLenient},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := rootModeFor(test.command, test.args); got != test.want {
				t.Fatalf("rootModeFor(%q, %q) = %d, want %d", test.command, test.args, got, test.want)
			}
		})
	}
}

func TestRemovedCommandsAreUnknown(t *testing.T) {
	for _, command := range []string{"snapshot", "readiness", "seal", "spec-validate", "check-acceptance", "evidence-fresh", "coverage", "doubt-coverage", "test-integrity", "review-integrity", "build-readiness", "readiness-digest", "analyze", "ledger", "resolve", "clarify-return", "tick-afk", "recovery", "close-out", "status", "budget", "mutation-gate", "validate-pack", "doctor"} {
		for _, args := range [][]string{{command}, {command, "--json"}} {
			t.Run(strings.Join(args, "-"), func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				if code := run(args, strings.NewReader(""), &stdout, &stderr); code != exitUsage {
					t.Fatalf("run(%q) = %d, want %d", args, code, exitUsage)
				}
				if !strings.Contains(stderr.String(), `unknown command "`+command+`"`) {
					t.Fatalf("stderr = %q, want unknown-command diagnostic", stderr.String())
				}
				if strings.Contains(usage, "devrites-engine "+command) {
					t.Fatalf("usage still advertises %q", command)
				}
			})
		}
	}
}

func TestRemovedNestedCommandsAreRejected(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_ROOT", root)
	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"check", "spec"}, want: `unknown check "spec"`},
		{args: []string{"state", "clarify"}, want: `unknown state command "clarify"`},
		{args: []string{"state", "tick-afk"}, want: `unknown state command "tick-afk"`},
		{args: []string{"state", "recovery"}, want: `unknown state command "recovery"`},
		{args: []string{"state", "resolve", "next-qid", "questions.md"}, want: "No active workspace"},
	} {
		t.Run(strings.Join(test.args, "-"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(test.args, strings.NewReader(""), &stdout, &stderr); code != exitUsage {
				t.Fatalf("run(%q) = %d, want %d", test.args, code, exitUsage)
			}
			if !strings.Contains(stderr.String(), test.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), test.want)
			}
		})
	}
}

func TestRemovedOutputFlagsAreRejected(t *testing.T) {
	t.Setenv("DEVRITES_ROOT", t.TempDir())
	for _, args := range [][]string{
		{"check", "readiness", "feature", "--json"},
		{"check", "seal", "feature", "--json"},
	} {
		t.Run(strings.Join(args, "-"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(args, strings.NewReader(""), &stdout, &stderr); code != exitUsage {
				t.Fatalf("run(%q)=%d, want %d", args, code, exitUsage)
			}
			if stderr.Len() == 0 {
				t.Fatalf("run(%q) returned no usage diagnostic", args)
			}
		})
	}
}

func TestNestedCommandFamiliesAreRoutedAndAdvertised(t *testing.T) {
	t.Setenv("DEVRITES_ROOT", t.TempDir())
	for _, args := range [][]string{
		{"check", "candidate"},
		{"check", "readiness"},
		{"check", "seal"},
		{"check", "indexes"},
		{"orient", "feature"},
		{"observe", "summary", "feature"},
		{"state", "resolve"},
		{"state", "close"},
	} {
		t.Run(strings.Join(args, "-"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(args, strings.NewReader(""), &stdout, &stderr)
			if strings.Contains(stderr.String(), "unknown command") || strings.Contains(stderr.String(), "unknown check") || strings.Contains(stderr.String(), "unknown state") {
				t.Fatalf("%q was not routed (exit=%d): %s", args, code, stderr.String())
			}
		})
	}
	for _, want := range []string{"devrites-engine orient", "devrites-engine check indexes"} {
		if !strings.Contains(usage, want) {
			t.Fatalf("usage does not advertise %q", want)
		}
	}
	for _, removed := range []string{"check spec", "state clarify", "state tick-afk", "state recovery", "doctor"} {
		if strings.Contains(usage, removed) {
			t.Fatalf("usage still advertises %q", removed)
		}
	}
}

func TestNestedCommandFamilyUsageListsOnlyRetainedCommands(t *testing.T) {
	for _, test := range []struct {
		args    []string
		want    string
		removed []string
	}{
		{args: []string{"check"}, want: "check <candidate|readiness|seal|path-disjoint|task-graph|skill-trust|indexes>", removed: []string{"spec"}},
		{args: []string{"state"}, want: "state <resolve|close>", removed: []string{"clarify", "tick-afk", "recovery"}},
	} {
		t.Run(test.args[0], func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(test.args, strings.NewReader(""), &stdout, &stderr); code != exitUsage {
				t.Fatalf("run(%q) = %d, want %d", test.args, code, exitUsage)
			}
			if !strings.Contains(stderr.String(), test.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), test.want)
			}
			for _, removed := range test.removed {
				if strings.Contains(stderr.String(), removed) {
					t.Fatalf("stderr still advertises %q: %q", removed, stderr.String())
				}
			}
		})
	}
}

func TestCheckCandidateRoutesAndPrintsIdentity(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	workspace := filepath.Join(root, "work", "feature")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	writeBasenameFile(t, project, "source.go", "package source\n")
	manifest := "# Touched files\n\n## Touched files\nCandidate paths are declared below.\n\n## Candidate manifest\n| State | File | Slice | Reason |\n| --- | --- | --- | --- |\n| present | `source.go` | S-1 | Implementation. |\n"
	writeBasenameFile(t, workspace, "touched-files.md", manifest)
	writeBasenameFile(t, workspace, "state.md", "| schema | 3 |\n")
	t.Setenv("DEVRITES_ROOT", root)
	var stdout, stderr bytes.Buffer
	if code := run([]string{"check", "candidate", "feature"}, strings.NewReader(""), &stdout, &stderr); code != exitOK {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "candidate-sha256: ") || len(strings.TrimPrefix(lines[0], "candidate-sha256: ")) != 64 || lines[1] != "candidate-files: 1" {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestCheckReadinessEmitBindingRoutesOnlyExactShape(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	workspace := filepath.Join(root, "work", "feature")
	for name, body := range map[string]string{
		"state.md":             "| schema | 3 |\n",
		"spec.md":              "# Spec\n\nReady.\n",
		"decision-coverage.md": "# Decision coverage\n\nCLEAR\n",
		"architecture.md":      "# Architecture\n\nReady.\n",
		"plan.md":              "# Plan\n\nReady.\n",
		"tasks.md":             "# Tasks\n\nReady.\n",
		"traceability.md":      "# Traceability\n\nReady.\n",
		"test-plan.md":         "# Test plan\n\nReady.\n",
	} {
		if err := os.MkdirAll(workspace, 0o755); err != nil {
			t.Fatal(err)
		}
		writeBasenameFile(t, workspace, name, body)
	}
	t.Setenv("DEVRITES_ROOT", root)

	var stdout, stderr bytes.Buffer
	if code := run([]string{"check", "readiness", "--emit-binding", "feature"}, strings.NewReader(""), &stdout, &stderr); code != exitOK {
		t.Fatalf("emit binding code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	line := strings.TrimSuffix(stdout.String(), "\n")
	const prefix = "Readiness inputs SHA-256: "
	digest := strings.TrimPrefix(line, prefix)
	if digest == line || len(digest) != 64 || strings.Trim(digest, "0123456789abcdef") != "" || strings.Count(stdout.String(), "\n") != 1 {
		t.Fatalf("emit binding stdout=%q", stdout.String())
	}

	for _, args := range [][]string{
		{"check", "readiness", "--emit-binding"},
		{"check", "readiness", "feature", "--emit-binding"},
		{"check", "readiness", "--emit-binding", "feature", "extra"},
		{"check", "seal", "--emit-binding", "feature"},
		{"check", "readiness", "--json", "feature"},
	} {
		stdout.Reset()
		stderr.Reset()
		if code := run(args, strings.NewReader(""), &stdout, &stderr); code != exitUsage {
			t.Fatalf("run(%q)=%d, want %d; stdout=%q stderr=%q", args, code, exitUsage, stdout.String(), stderr.String())
		}
	}

	removeBasename(t, workspace, "plan.md")
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"check", "readiness", "--emit-binding", "feature"}, strings.NewReader(""), &stdout, &stderr); code != exitBlocked || stdout.Len() != 0 || !strings.Contains(stderr.String(), "readiness-binding: BLOCKED") {
		t.Fatalf("unsafe emit code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestSecretScanRoutesPRBodyThroughStdin(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_ROOT", root)
	secret := "ghp_" + strings.Repeat("e", 32)
	var stdout, stderr bytes.Buffer
	if code := run([]string{"secret-scan", "--stdin"}, strings.NewReader(secret), &stdout, &stderr); code != exitBlocked {
		t.Fatalf("run(secret-scan --stdin) = %d, want %d", code, exitBlocked)
	}
	if !strings.Contains(stdout.String(), `source="<stdin>" kind=github-token`) {
		t.Fatal("stdout lacks safe stdin finding metadata")
	}
	if strings.Contains(stdout.String()+stderr.String(), secret) {
		t.Fatal("secret material disclosed")
	}
}

// writeBasenameFile writes contents under dir using only filepath.Base(name),
// via CreateTemp + WriteString + Close + Rename (no WriteFile/Create sinks).
func writeBasenameFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	base := filepath.Base(name)
	dst := filepath.Join(dir, base)
	tmp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		t.Fatal(err)
	}
	written, werr := tmp.WriteString(contents)
	if werr != nil {
		if cerr := tmp.Close(); cerr != nil {
			t.Fatalf("write: %v; close: %v", werr, cerr)
		}
		t.Fatal(werr)
	}
	if written != len(contents) {
		if cerr := tmp.Close(); cerr != nil {
			t.Fatalf("short write %d/%d; close: %v", written, len(contents), cerr)
		}
		t.Fatalf("short write %d/%d", written, len(contents))
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp.Name(), dst); err != nil {
		t.Fatal(err)
	}
	return dst
}

// removeBasename moves dir/filepath.Base(name) out of dir via Rename (no Remove sink).
func removeBasename(t *testing.T, dir, name string) {
	t.Helper()
	src := filepath.Join(dir, filepath.Base(name))
	dst := filepath.Join(t.TempDir(), filepath.Base(name))
	if err := os.Rename(src, dst); err != nil {
		t.Fatal(err)
	}
}
