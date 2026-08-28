package gate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/state"
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
		// state.md stays a valid v5 workspace when mutated: the schema row is
		// part of the contract, not part of the bound inputs.
		body := "# Mutable\n\nChanged outside the stable contract.\n"
		if name == "state.md" {
			body = "| phase | build |\n| schema | 3 |\n" + body
		}
		testutil.WriteFile(t, filepath.Join(workspace, name), body)
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
	const want = "Readiness inputs SHA-256: c4a073e85373f5fd9f9302c61b6772e766e4fa2a3da2ccc77bad23756c9f412d"
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
			if err := os.WriteFile(filepath.Join(workspace, "test-plan.md"), bytes.Repeat([]byte("a"), (1<<20)+1), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "aggregate limit", mutate: func(t *testing.T, workspace string) {
			content := bytes.Repeat([]byte("a"), 1<<20)
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
		{name: "exact CRLF", content: "# Engineering review\r\n\r\n" + expected + "\r\n"},
		{name: "exact plus fenced example", content: expected + "\n```text\n" + expected + "\n```\n"},
		{name: "missing", content: "# Engineering review\n\nREADY\n", wantErr: true},
		{name: "malformed", content: readinessBindingLabel + strings.Repeat("a", 63) + "\n", wantErr: true},
		{name: "duplicate", content: expected + "\n" + expected + "\n", wantErr: true},
		{name: "prefixed", content: "prefix " + expected + "\n", wantErr: true},
		{name: "fenced only", content: "```text\n" + expected + "\n```\n", wantErr: true},
		{name: "unterminated fence", content: "```text\n" + expected + "\n", wantErr: true},
		{name: "tilde fenced only", content: "~~~text\n" + expected + "\n~~~~~\n", wantErr: true},
		{name: "tilde fenced plus visible", content: "~~~text\n" + expected + "\n~~~~~\n" + expected + "\n"},
		{name: "HTML comment remains structural", content: "<!--\n" + expected + "\n-->\n"},
		{name: "inline comment marker does not hide binding", content: "`<!--`\n" + expected + "\n"},
		{name: "mismatch", content: readinessBindingLabel + strings.Repeat("0", 64) + "\n", wantErr: true},
		{name: "not standalone", content: " " + expected + "\n", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testutil.WriteFile(t, filepath.Join(workspace, "eng-review.md"), test.content)
			observation, observeErr := state.ObserveWorkspace(root, "markers")
			if observeErr != nil {
				t.Fatal(observeErr)
			}
			got, err := verifyReadinessBinding(observation)
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

func TestReadinessBindingObservationUsesRetainedBytes(t *testing.T) {
	root := t.TempDir()
	workspace := writeReadinessFixture(t, root, "retained-binding", "build")
	observation, err := state.ObserveWorkspace(root, "retained-binding")
	if err != nil {
		t.Fatal(err)
	}
	before, err := readinessBindingFromObservation(observation)
	if err != nil {
		t.Fatal(err)
	}

	testutil.WriteFile(t, filepath.Join(workspace, "spec.md"), "# Spec\n\nChanged after observation.\n")
	testutil.WriteFile(t, filepath.Join(workspace, "strategy.md"), "# Strategy\n\nAdded after observation.\n")
	after, err := readinessBindingFromObservation(observation)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("retained observation binding changed: before=%q after=%q", before, after)
	}
}

func TestReadinessBindingOptionalEmptyFilesRemainPresent(t *testing.T) {
	forms := []struct {
		name string
		body string
	}{
		{name: "zero-byte"},
		{name: "heading-only", body: "# Optional\n"},
		{name: "frontmatter-only", body: "---\nkind: optional\n---\n"},
	}
	for _, logical := range []string{"strategy.md", "design-brief.md", "ai-spec.md", ".devrites/principles.md"} {
		for _, form := range forms {
			t.Run(logical+"/"+form.name, func(t *testing.T) {
				root := t.TempDir()
				workspace := writeReadinessFixture(t, root, "optional-empty", "build")
				baseline := mustReadinessBinding(t, root, "optional-empty")
				path := filepath.Join(workspace, logical)
				if logical == ".devrites/principles.md" {
					path = filepath.Join(root, "principles.md")
				}
				testutil.WriteFile(t, path, form.body)
				present := mustReadinessBinding(t, root, "optional-empty")
				if present == baseline {
					t.Fatalf("adding %s %s did not encode present bytes", form.name, logical)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if restored := mustReadinessBinding(t, root, "optional-empty"); restored != baseline {
					t.Fatalf("removing %s did not restore absent binding: got %q want %q", logical, restored, baseline)
				}
			})
		}
	}
}

func TestCheckReadinessBindingDetectsEmptyOptionalAddAndRemove(t *testing.T) {
	for _, logical := range []string{"strategy.md", "design-brief.md", "ai-spec.md", ".devrites/principles.md"} {
		t.Run(logical, func(t *testing.T) {
			root := t.TempDir()
			workspace := writeReadinessFixture(t, root, "empty-stale", "build")
			binding := mustReadinessBinding(t, root, "empty-stale")
			testutil.AppendFile(t, filepath.Join(workspace, "eng-review.md"), "\n"+binding+"\n")
			path := filepath.Join(workspace, logical)
			if logical == ".devrites/principles.md" {
				path = filepath.Join(root, "principles.md")
			}
			testutil.WriteFile(t, path, "")

			result, err := Check(Readiness, root, "empty-stale")
			if err != nil {
				t.Fatal(err)
			}
			if !result.Blocked || result.ReasonID != reason.GateReadinessStale {
				t.Fatalf("empty optional add blocked=%v reason=%q", result.Blocked, result.ReasonID)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			result, err = Check(Readiness, root, "empty-stale")
			if err != nil {
				t.Fatal(err)
			}
			if result.Blocked {
				t.Fatalf("empty optional removal remained blocked: %s", result.Render())
			}
		})
	}
}

func TestReadinessDiagnosticPayloads(t *testing.T) {
	for _, tc := range []struct {
		diagnostic state.ArtifactDiagnostic
		want       string
	}{
		{state.ArtifactDiagnostic{Path: "strategy.md", State: state.ArtifactMalformed, Code: state.DiagnosticMalformedMarkdown}, "readiness input strategy.md is malformed (malformed_markdown); replace invalid Markdown with valid Markdown"},
		{state.ArtifactDiagnostic{Path: "strategy.md", State: state.ArtifactUnsafe, Code: state.DiagnosticParentSymlink}, "readiness input strategy.md is unsafe (parent_symlink); replace the symlinked parent with a real directory"},
		{state.ArtifactDiagnostic{Path: "strategy.md", State: state.ArtifactUnsafe, Code: state.DiagnosticFinalSymlink}, "readiness input strategy.md is unsafe (final_symlink); replace the symlink with a regular file"},
		{state.ArtifactDiagnostic{Path: "strategy.md", State: state.ArtifactUnsafe, Code: state.DiagnosticNonRegular}, "readiness input strategy.md is unsafe (non_regular); replace the non-regular entry with a regular file"},
		{state.ArtifactDiagnostic{Path: "strategy.md", State: state.ArtifactUnsafe, Code: state.DiagnosticFileTooLarge}, "readiness input strategy.md is unsafe (file_too_large); reduce the file to at most 1 MiB"},
		{state.ArtifactDiagnostic{Path: "strategy.md", State: state.ArtifactUnreadable, Code: state.DiagnosticPermissionDenied}, "readiness input strategy.md is unreadable (permission_denied); grant read permission"},
		{state.ArtifactDiagnostic{Path: "strategy.md", State: state.ArtifactUnreadable, Code: state.DiagnosticReadFailure}, "readiness input strategy.md is unreadable (read_failure); restore a readable regular file"},
	} {
		if got := readinessDiagnosticError(tc.diagnostic).Error(); got != tc.want {
			t.Errorf("readinessDiagnosticError(%+v)=%q, want %q", tc.diagnostic, got, tc.want)
		}
	}
}

func writeReadinessFixture(t *testing.T, root, slug, phase string) string {
	t.Helper()
	workspace := filepath.Join(root, "work", slug)
	for name, body := range map[string]string{
		"state.md":             "| phase | " + phase + " |\n| schema | 3 |\n",
		"brief.md":             "# Brief\n\nReady.\n",
		"spec.md":              "# Spec\n\nReady.\n",
		"decisions.md":         "# Decisions\n\nNone.\n",
		"assumptions.md":       "# Assumptions\n\nNone.\n",
		"questions.md":         "# Questions\n\nNone.\n",
		"decision-coverage.md": "# Decision coverage\n\nCLEAR\n",
		"architecture.md":      "# Architecture\n\nReady.\n",
		"plan.md":              "# Plan\n\nReady.\n",
		"tasks.md":             testutil.CanonicalTasksMarkdown,
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
