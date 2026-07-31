package gate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/testutil"
)

func TestReadinessBindingBindsOnlyStableBuildInputs(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	workspace := writeReadinessFixture(t, root, "stable", "build")
	baseline := mustReadinessBinding(t, root, "stable")

	planPath := filepath.Join(workspace, "plan.md")
	info, err := os.Stat(planPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(planPath, info.ModTime().AddDate(0, 0, 1), info.ModTime().AddDate(0, 0, 1)); err != nil {
		t.Fatal(err)
	}
	if got := mustReadinessBinding(t, root, "stable"); got != baseline {
		t.Fatalf("mtime-only change altered binding: got %q want %q", got, baseline)
	}

	for _, name := range []string{"spec.md", "decision-coverage.md", "architecture.md", "plan.md", "tasks.md", "traceability.md", "test-plan.md"} {
		path := filepath.Join(workspace, name)
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		testutil.WriteFile(t, path, string(body)+"changed\n")
		if got := mustReadinessBinding(t, root, "stable"); got == baseline {
			t.Fatalf("content change to %s did not alter binding", name)
		}
		testutil.WriteFile(t, path, string(body))
	}

	for _, name := range []string{"strategy.md", "design-brief.md", "ai-spec.md"} {
		path := filepath.Join(workspace, name)
		testutil.WriteFile(t, path, "# Optional\n\nPresent.\n")
		present := mustReadinessBinding(t, root, "stable")
		if present == baseline {
			t.Fatalf("adding optional %s did not alter binding", name)
		}
		testutil.WriteFile(t, path, "# Optional\n\nChanged.\n")
		if got := mustReadinessBinding(t, root, "stable"); got == present {
			t.Fatalf("content change to optional %s did not alter binding", name)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if got := mustReadinessBinding(t, root, "stable"); got != baseline {
			t.Fatalf("removing optional %s did not restore binding", name)
		}
	}

	principles := filepath.Join(root, "principles.md")
	testutil.WriteFile(t, principles, "# Principles\n\nKeep proof deterministic.\n")
	presentPrinciples := mustReadinessBinding(t, root, "stable")
	if presentPrinciples == baseline {
		t.Fatal("adding .devrites/principles.md did not alter binding")
	}
	testutil.WriteFile(t, principles, "# Principles\n\nChanged.\n")
	if got := mustReadinessBinding(t, root, "stable"); got == presentPrinciples {
		t.Fatal("content change to .devrites/principles.md did not alter binding")
	}
	if err := os.Remove(principles); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"state.md", "questions.md", "decisions.md", "assumptions.md", "eng-review.md", "evidence.md", "review.md", "seal.md", "handoff.md", "ambient.md"} {
		testutil.WriteFile(t, filepath.Join(workspace, name), "# Mutable\n\nChanged outside the stable contract.\n")
		if got := mustReadinessBinding(t, root, "stable"); got != baseline {
			t.Fatalf("excluded %s altered binding", name)
		}
	}
	testutil.WriteFile(t, filepath.Join(project, ".git", "index"), "ambient git state\n")
	if got := mustReadinessBinding(t, root, "stable"); got != baseline {
		t.Fatal("ambient Git state altered binding")
	}
}

func TestReadinessBindingGoldenDigest(t *testing.T) {
	root := t.TempDir()
	writeReadinessFixture(t, root, "golden", "build")
	const want = "Readiness inputs SHA-256: d84b9050bd8db742c6a379a966bd7457cde04a69d6811130817729776d976ebe"
	if got := mustReadinessBinding(t, root, "golden"); got != want {
		t.Fatalf("ReadinessBinding()=%q, want %q", got, want)
	}
}

func TestReadinessBindingRejectsUnsafeInputsWithoutDisclosure(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{name: "missing required", mutate: func(t *testing.T, workspace string) {
			if err := os.Remove(filepath.Join(workspace, "spec.md")); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "symlink", mutate: func(t *testing.T, workspace string) {
			path := filepath.Join(workspace, "plan.md")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("spec.md", path); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "directory", mutate: func(t *testing.T, workspace string) {
			path := filepath.Join(workspace, "tasks.md")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "malformed utf8", mutate: func(t *testing.T, workspace string) {
			if err := os.WriteFile(filepath.Join(workspace, "architecture.md"), []byte{0xff, 0xfe}, 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "nul", mutate: func(t *testing.T, workspace string) {
			if err := os.WriteFile(filepath.Join(workspace, "traceability.md"), []byte("secret-token\x00hidden"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "per input limit", mutate: func(t *testing.T, workspace string) {
			if err := os.WriteFile(filepath.Join(workspace, "test-plan.md"), bytes.Repeat([]byte("a"), int(maxReadinessInputBytes)+1), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "aggregate limit", mutate: func(t *testing.T, workspace string) {
			content := bytes.Repeat([]byte("a"), int(maxReadinessInputBytes))
			for _, name := range []string{"spec.md", "decision-coverage.md", "architecture.md", "plan.md", "tasks.md", "traceability.md", "test-plan.md", "strategy.md", "design-brief.md"} {
				if err := os.WriteFile(filepath.Join(workspace, name), content, 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			workspace := writeReadinessFixture(t, root, "unsafe", "build")
			test.mutate(t, workspace)
			if _, err := ReadinessBinding(root, "unsafe"); err == nil {
				t.Fatal("ReadinessBinding() should reject unsafe input")
			} else if strings.Contains(err.Error(), workspace) || strings.Contains(err.Error(), "secret-token") {
				t.Fatalf("error disclosed physical path or content: %q", err)
			}
		})
	}
}

func TestVerifyReadinessBindingRequiresOneStructuralStandaloneLine(t *testing.T) {
	root := t.TempDir()
	workspace := writeReadinessFixture(t, root, "markers", "build")
	expected := mustReadinessBinding(t, root, "markers")
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{name: "exact", content: "# Engineering review\n\n" + expected + "\n"},
		{name: "exact plus fenced example", content: expected + "\n```text\n" + expected + "\n```\n"},
		{name: "missing", content: "# Engineering review\n\nREADY\n", wantErr: true},
		{name: "malformed", content: readinessBindingLabel + strings.Repeat("a", 63) + "\n", wantErr: true},
		{name: "duplicate", content: expected + "\n" + expected + "\n", wantErr: true},
		{name: "fenced only", content: "```text\n" + expected + "\n```\n", wantErr: true},
		{name: "mismatch", content: readinessBindingLabel + strings.Repeat("0", 64) + "\n", wantErr: true},
		{name: "not standalone", content: " " + expected + "\n", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testutil.WriteFile(t, filepath.Join(workspace, "eng-review.md"), test.content)
			got, err := verifyReadinessBinding(root, "markers")
			if (err != nil) != test.wantErr {
				t.Fatalf("verifyReadinessBinding() error=%v, wantErr=%v", err, test.wantErr)
			}
			if got != expected {
				t.Fatalf("expected line=%q, want %q", got, expected)
			}
		})
	}
}

func TestCheckUsesReadinessBindingForBuildAndSeal(t *testing.T) {
	for _, phase := range []string{"build", "seal"} {
		t.Run(phase, func(t *testing.T) {
			root := t.TempDir()
			workspace := writeReadinessFixture(t, root, phase, phase)
			if phase == "seal" {
				for _, name := range []string{"evidence.md", "touched-files.md", "review.md", "seal.md"} {
					testutil.WriteFile(t, filepath.Join(workspace, name), "# Final\n\nPresent.\n")
				}
			}
			binding := mustReadinessBinding(t, root, phase)
			testutil.AppendFile(t, filepath.Join(workspace, "eng-review.md"), "\n"+binding+"\n")

			kind := Readiness
			if phase == "seal" {
				kind = Seal
			}
			result, err := Check(kind, root, phase)
			if err != nil {
				t.Fatal(err)
			}
			if result.Blocked {
				t.Fatalf("initial Check() blocked=true: %s", result.Render())
			}

			testutil.WriteFile(t, filepath.Join(workspace, "plan.md"), "# Plan\n\nChanged after Vet.\n")
			expected := mustReadinessBinding(t, root, phase)
			result, err = Check(kind, root, phase)
			if err != nil {
				t.Fatal(err)
			}
			if !result.Blocked || result.ReasonID != reason.GateReadinessStale {
				t.Fatalf("stale Check() blocked=%v reason=%q", result.Blocked, result.ReasonID)
			}
			rendered := result.Render()
			if !strings.Contains(rendered, "rerun /rite-vet") || !strings.Contains(rendered, expected) {
				t.Fatalf("stale output lacks recovery or expected binding:\n%s", rendered)
			}
		})
	}
}

func writeReadinessFixture(t *testing.T, root, slug, phase string) string {
	t.Helper()
	workspace := filepath.Join(root, "work", slug)
	for name, body := range map[string]string{
		"state.md":             "| phase | " + phase + " |\n",
		"brief.md":             "# Brief\n\nReady.\n",
		"spec.md":              "# Spec\n\nReady.\n",
		"decisions.md":         "# Decisions\n\nNone.\n",
		"assumptions.md":       "# Assumptions\n\nNone.\n",
		"questions.md":         "# Questions\n\nNone.\n",
		"decision-coverage.md": "# Decision coverage\n\nCLEAR\n",
		"architecture.md":      "# Architecture\n\nReady.\n",
		"plan.md":              "# Plan\n\nReady.\n",
		"tasks.md":             "# Tasks\n\nReady.\n",
		"traceability.md":      "# Traceability\n\nReady.\n",
		"eng-review.md":        "# Engineering review\n\nREADY\n",
		"test-plan.md":         "# Test plan\n\nReady.\n",
	} {
		testutil.WriteFile(t, filepath.Join(workspace, name), body)
	}
	return workspace
}

func mustReadinessBinding(t *testing.T, root, slug string) string {
	t.Helper()
	binding, err := ReadinessBinding(root, slug)
	if err != nil {
		t.Fatal(err)
	}
	return binding
}
