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
		{name: "check family", command: "check", args: []string{"readiness"}, want: rootUnused},
		{name: "resolve", command: "state", args: []string{"resolve"}, want: rootStrict},
		{name: "close", command: "state", args: []string{"close"}, want: rootStrict},
		{name: "unknown state command", command: "state", args: []string{"unknown"}, want: rootUnused},
		{name: "secret scan", command: "secret-scan", want: rootLenient},
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
	for _, command := range []string{"snapshot", "readiness", "seal", "spec-validate", "check-acceptance", "evidence-fresh", "coverage", "doubt-coverage", "test-integrity", "review-integrity", "build-readiness", "readiness-digest", "analyze", "ledger", "resolve", "clarify-return", "tick-afk", "recovery", "close-out", "migrate", "status", "budget", "mutation-gate", "validate-pack", "doctor"} {
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
	for _, args := range [][]string{{"check", "candidate"}, {"check", "readiness"}, {"check", "seal"}, {"state", "resolve"}, {"state", "close"}} {
		t.Run(strings.Join(args, "-"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			_ = run(args, strings.NewReader(""), &stdout, &stderr)
			if strings.Contains(stderr.String(), "unknown command") || strings.Contains(stderr.String(), "unknown check") || strings.Contains(stderr.String(), "unknown state") {
				t.Fatalf("%q was not routed: %s", args, stderr.String())
			}
		})
	}
	for _, want := range []string{"devrites-engine check candidate", "devrites-engine check readiness", "devrites-engine check seal", "devrites-engine state resolve", "devrites-engine state close"} {
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
		{args: []string{"check"}, want: "check <candidate|readiness|seal>", removed: []string{"spec"}},
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
	if err := os.WriteFile(filepath.Join(project, "source.go"), []byte("package source\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := "# Touched files\n\n## Touched files\nCandidate paths are declared below.\n\n## Candidate manifest\n| State | File | Slice | Reason |\n| --- | --- | --- | --- |\n| present | `source.go` | S-1 | Implementation. |\n"
	if err := os.WriteFile(filepath.Join(workspace, "touched-files.md"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
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
		if err := os.WriteFile(filepath.Join(workspace, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
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

	if err := os.Remove(filepath.Join(workspace, "plan.md")); err != nil {
		t.Fatal(err)
	}
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
