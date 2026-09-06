package state

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestWorkspaceObservationInventoryCanonicalArtifactsOnce(t *testing.T) {
	root, slug := newObservationWorkspace(t)
	wantPaths := []ArtifactPath{
		"state.md",
		"brief.md",
		"spec.md",
		"decisions.md",
		"assumptions.md",
		"questions.md",
		"decision-coverage.md",
		"architecture.md",
		"plan.md",
		"tasks.md",
		"traceability.md",
		"eng-review.md",
		"test-plan.md",
		"evidence.md",
		"touched-files.md",
		"review.md",
		"seal.md",
		"strategy.md",
		"design-brief.md",
		"ai-spec.md",
		".devrites/principles.md",
	}
	for _, path := range wantPaths {
		body := []byte("# Artifact\n\nsubstantive content\n")
		switch path {
		case LedgerFile:
			body = []byte("- Phase: build\n")
		case ".devrites/principles.md":
			body = []byte("root principles\n")
		}
		writeObservationArtifact(t, root, slug, path, body)
	}
	writeFile(t, filepath.Join(root, "work", slug, "principles.md"), []byte("workspace principles decoy\n"))
	writeFile(t, filepath.Join(root, "work", slug, ".devrites", "principles.md"), []byte("workspace .devrites principles decoy\n"))
	physicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_WORKSPACE", filepath.Join(physicalRoot, "work", slug))

	opens := make(map[ArtifactPath]int)
	reads := make(map[ArtifactPath]int)
	observation, err := observeWorkspace(physicalRoot, slug, func(stage observationStage, path ArtifactPath) error {
		switch stage {
		case observationBeforeOpen:
			opens[path]++
		case observationBeforeRead:
			reads[path]++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if observation.Slug() != slug {
		t.Fatalf("Slug() = %q, want %q", observation.Slug(), slug)
	}
	facts := observation.Facts()
	gotPaths := make([]ArtifactPath, len(facts))
	for i, fact := range facts {
		gotPaths[i] = fact.Path()
	}
	if !slices.Equal(gotPaths, wantPaths) {
		t.Fatalf("Facts paths = %v, want %v", gotPaths, wantPaths)
	}
	for _, path := range wantPaths {
		if opens[path] != 1 || reads[path] != 1 {
			t.Errorf("%s opened/read = %d/%d, want 1/1", path, opens[path], reads[path])
		}
	}
	principles, ok := observation.Fact(".devrites/principles.md")
	if !ok || string(principles.Bytes()) != "root principles\n" {
		t.Fatalf("root principles fact = (%q, %v), want selected-root principles", principles.Bytes(), ok)
	}
}

func TestWorkspaceObservationPathAcceptsCanonicalRelativeOverride(t *testing.T) {
	root, slug := newObservationWorkspace(t)
	writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
	physicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_WORKSPACE", filepath.Join(filepath.Base(physicalRoot), "work", slug))

	observation, err := ObserveWorkspace(physicalRoot, slug)
	if err != nil {
		t.Fatal(err)
	}
	if observation.Slug() != slug {
		t.Fatalf("Slug() = %q, want %q", observation.Slug(), slug)
	}
}

func TestWorkspaceObservationPathRejectsWorkspaceOverrideWhitespace(t *testing.T) {
	root, slug := newObservationWorkspace(t)
	writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
	physicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	canonical := filepath.Join(physicalRoot, "work", slug)
	tests := []struct {
		name     string
		override string
	}{
		{name: "leading", override: " " + canonical},
		{name: "trailing", override: canonical + " "},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("DEVRITES_WORKSPACE", test.override)
			observation, err := ObserveWorkspace(physicalRoot, slug)
			if observation != nil {
				t.Fatalf("observation = %+v, want nil", observation)
			}
			assertObservationFailure(t, err, ObservationWorkspaceInvalid, "workspace observation: workspace_invalid: workspace is unavailable; verify the selected logical workspace and canonical workspace override, then retry")
		})
	}
}

func TestWorkspaceObservationPathRejectsLeadingParentRelativeOverride(t *testing.T) {
	root, slug := newObservationWorkspace(t)
	writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
	physicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	rootParent := filepath.Dir(physicalRoot)
	override := filepath.Join("..", filepath.Base(rootParent), filepath.Base(physicalRoot), "work", slug)
	if filepath.IsAbs(override) || !strings.HasPrefix(override, ".."+string(filepath.Separator)) {
		t.Fatalf("override = %q, want leading-parent relative path", override)
	}
	if filepath.Clean(override) != override {
		t.Fatalf("filepath.Clean(%q) changed fixture", override)
	}
	if filepath.IsLocal(override) {
		t.Fatalf("filepath.IsLocal(%q) = true, want false", override)
	}
	want := filepath.Join(physicalRoot, "work", slug)
	if resolved := filepath.Join(rootParent, override); resolved != want {
		t.Fatalf("filepath.Join(%q, %q) = %q, want %q", rootParent, override, resolved, want)
	}
	t.Setenv("DEVRITES_WORKSPACE", override)

	observation, err := ObserveWorkspace(physicalRoot, slug)
	if observation != nil {
		t.Fatalf("observation = %+v, want nil", observation)
	}
	assertObservationFailure(t, err, ObservationWorkspaceInvalid, "workspace observation: workspace_invalid: workspace is unavailable; verify the selected logical workspace and canonical workspace override, then retry")
}

func TestWorkspaceObservationPathRejectsWorkspaceOverrideAliases(t *testing.T) {
	aliases := []struct {
		name string
		path func(root string) string
	}{
		{
			name: "in-root symlink alias",
			path: func(root string) string {
				return filepath.Join(root, "work", "alias")
			},
		},
		{
			name: "outside-root symlink alias",
			path: func(string) string {
				return filepath.Join(t.TempDir(), "alias")
			},
		},
	}
	overrides := []struct {
		name string
		path func(t *testing.T, root, slug, alias string) string
	}{
		{
			name: "direct absolute",
			path: func(_ *testing.T, _, _, alias string) string {
				return alias
			},
		},
		{
			name: "wrapped relative",
			path: func(t *testing.T, root, slug, alias string) string {
				aliasFromRootParent, err := filepath.Rel(filepath.Dir(root), alias)
				if err != nil {
					t.Fatal(err)
				}
				targetFromAliasParent, err := filepath.Rel(filepath.Dir(alias), filepath.Join(root, "work", slug))
				if err != nil {
					t.Fatal(err)
				}
				separator := string(filepath.Separator)
				return aliasFromRootParent + separator + ".." + separator + targetFromAliasParent
			},
		},
		{
			name: "wrapped absolute",
			path: func(t *testing.T, root, slug, alias string) string {
				targetFromAliasParent, err := filepath.Rel(filepath.Dir(alias), filepath.Join(root, "work", slug))
				if err != nil {
					t.Fatal(err)
				}
				separator := string(filepath.Separator)
				return alias + separator + ".." + separator + targetFromAliasParent
			},
		},
	}

	for _, aliasCase := range aliases {
		for _, overrideCase := range overrides {
			t.Run(aliasCase.name+"/"+overrideCase.name, func(t *testing.T) {
				root, slug := newObservationWorkspace(t)
				writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
				physicalRoot, err := filepath.EvalSymlinks(root)
				if err != nil {
					t.Fatal(err)
				}
				alias := aliasCase.path(physicalRoot)
				if err := os.Symlink(filepath.Join(physicalRoot, "work", slug), alias); err != nil {
					t.Fatal(err)
				}
				aliasInfo, err := os.Lstat(alias)
				if err != nil {
					t.Fatal(err)
				}
				if aliasInfo.Mode()&os.ModeSymlink == 0 {
					t.Fatalf("alias mode = %v, want symlink", aliasInfo.Mode())
				}
				t.Setenv("DEVRITES_WORKSPACE", overrideCase.path(t, physicalRoot, slug, alias))

				observation, err := ObserveWorkspace(physicalRoot, slug)
				if observation != nil {
					t.Fatalf("observation = %+v, want nil", observation)
				}
				assertObservationFailure(t, err, ObservationWorkspaceInvalid, "workspace observation: workspace_invalid: workspace is unavailable; verify the selected logical workspace and canonical workspace override, then retry")
			})
		}
	}
}

func TestWorkspaceObservationSymlinkClassifiesRootPrinciplesFinalLink(t *testing.T) {
	root, slug := newObservationWorkspace(t)
	writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
	target := filepath.Join(t.TempDir(), "principles.md")
	writeFile(t, target, []byte("outside principles\n"))
	if err := os.Symlink(target, filepath.Join(root, "principles.md")); err != nil {
		t.Fatal(err)
	}

	observation, err := ObserveWorkspace(root, slug)
	if err != nil {
		t.Fatal(err)
	}
	assertArtifactFact(t, observation, ".devrites/principles.md", ArtifactUnsafe, DiagnosticFinalSymlink, nil)
}

func TestWorkspaceObservationStateClassifiesCodesAndRetention(t *testing.T) {
	root, slug := newObservationWorkspace(t)
	writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
	writeObservationArtifact(t, root, slug, "spec.md", []byte("# Empty\n\n"))
	malformed := []byte("hostile-secret\xff\n")
	writeObservationArtifact(t, root, slug, "decisions.md", malformed)

	target := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(target, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, observationArtifactPath(root, slug, "assumptions.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(observationArtifactPath(root, slug, "questions.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeObservationArtifact(t, root, slug, "decision-coverage.md", sizedMarkdown("coverage\n", (1<<20)+1))
	writeObservationArtifact(t, root, slug, "architecture.md", []byte("architecture\n"))
	writeObservationArtifact(t, root, slug, "plan.md", []byte("plan\n"))
	writeObservationArtifact(t, root, slug, "traceability.md", []byte{})
	present := []byte("# Tasks\n\n- [ ] retained\n")
	writeObservationArtifact(t, root, slug, "tasks.md", present)

	observation, err := observeWorkspace(root, slug, func(stage observationStage, path ArtifactPath) error {
		if stage == observationBeforeOpen && path == "architecture.md" {
			return fs.ErrPermission
		}
		if stage == observationBeforeRead && path == "plan.md" {
			return errors.New("hostile read failure at /physical/path")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	assertArtifactFact(t, observation, "brief.md", ArtifactAbsent, "", nil)
	assertArtifactFact(t, observation, "spec.md", ArtifactEmpty, "", []byte("# Empty\n\n"))
	assertArtifactFact(t, observation, "decisions.md", ArtifactMalformed, DiagnosticMalformedMarkdown, malformed)
	assertArtifactFact(t, observation, "assumptions.md", ArtifactUnsafe, DiagnosticFinalSymlink, nil)
	assertArtifactFact(t, observation, "questions.md", ArtifactUnsafe, DiagnosticNonRegular, nil)
	assertArtifactFact(t, observation, "decision-coverage.md", ArtifactUnsafe, DiagnosticFileTooLarge, nil)
	assertArtifactFact(t, observation, "architecture.md", ArtifactUnreadable, DiagnosticPermissionDenied, nil)
	assertArtifactFact(t, observation, "plan.md", ArtifactUnreadable, DiagnosticReadFailure, nil)
	assertArtifactFact(t, observation, "tasks.md", ArtifactPresent, "", present)
	assertArtifactFact(t, observation, "traceability.md", ArtifactEmpty, "", []byte{})
	zeroByteFact, _ := observation.Fact("traceability.md")
	if zeroByteFact.Bytes() == nil {
		t.Fatal("zero-byte empty artifact did not retain a copied empty byte slice")
	}
}

func TestWorkspaceObservationSymlinkClassifiesParent(t *testing.T) {
	base := t.TempDir()
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "spec.md"), []byte("content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(base, "nested")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()

	fact, _, err := observeArtifact(artifactLocation{
		path: ArtifactPath("spec.md"),
		root: root,
		name: "nested/spec.md",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if fact.State() != ArtifactUnsafe {
		t.Fatalf("State() = %q, want %q", fact.State(), ArtifactUnsafe)
	}
	diagnostic, ok := fact.Diagnostic()
	if !ok || diagnostic.Code != DiagnosticParentSymlink {
		t.Fatalf("Diagnostic() = (%+v, %v), want parent symlink", diagnostic, ok)
	}
	if fact.Bytes() != nil {
		t.Fatalf("Bytes() = %q, want nil", fact.Bytes())
	}
}

func TestWorkspaceObservationDefensiveCopies(t *testing.T) {
	root, slug := newObservationWorkspace(t)
	body := []byte("- Phase: build\n")
	writeObservationArtifact(t, root, slug, LedgerFile, body)
	observation, err := ObserveWorkspace(root, slug)
	if err != nil {
		t.Fatal(err)
	}

	facts := observation.Facts()
	facts[0].path = "mutated.md"
	facts[0].state = ArtifactUnsafe
	facts[0].bytes[0] = 'X'
	facts[0] = ArtifactFact{}

	fact, ok := observation.Fact(LedgerFile)
	if !ok {
		t.Fatal("Fact(state.md) returned ok=false")
	}
	returned := fact.Bytes()
	returned[0] = 'Y'
	fresh, ok := observation.Fact(LedgerFile)
	if !ok || fresh.Path() != LedgerFile || fresh.State() != ArtifactPresent || !slices.Equal(fresh.Bytes(), body) {
		t.Fatalf("fresh fact = (%q, %q, %q, %v), want retained state fact", fresh.Path(), fresh.State(), fresh.Bytes(), ok)
	}
}

func TestWorkspaceObservationUnknownLookupHasNoSideEffects(t *testing.T) {
	root, slug := newObservationWorkspace(t)
	writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
	callbacks := 0
	observation, err := observeWorkspace(root, slug, func(observationStage, ArtifactPath) error {
		callbacks++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	beforeCallbacks := callbacks
	beforeFacts := observation.Facts()

	fact, ok := observation.Fact("not-in-inventory.md")
	if ok || fact.path != "" || fact.state != "" || fact.code != "" || fact.bytes != nil {
		t.Fatalf("Fact(uninventoried) = (%+v, %v), want zero/false", fact, ok)
	}
	if fact.Bytes() != nil {
		t.Fatalf("zero fact Bytes() = %q, want nil", fact.Bytes())
	}
	if diagnostic, diagnosticOK := fact.Diagnostic(); diagnosticOK || diagnostic != (ArtifactDiagnostic{}) {
		t.Fatalf("zero fact Diagnostic() = (%+v, %v), want zero/false", diagnostic, diagnosticOK)
	}
	if callbacks != beforeCallbacks || !slices.EqualFunc(observation.Facts(), beforeFacts, equalArtifactFact) {
		t.Fatal("uninventoried lookup performed I/O or mutated the observation")
	}
}

func TestWorkspaceObservationStateMissingPreservesCallerOrderAndSelectedDiagnostics(t *testing.T) {
	root, slug := newObservationWorkspace(t)
	writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
	writeObservationArtifact(t, root, slug, "brief.md", []byte("# Heading\n"))
	writeObservationArtifact(t, root, slug, "spec.md", []byte("bad\x00markdown"))
	writeObservationArtifact(t, root, slug, "plan.md", []byte("real\n"))
	observation, err := ObserveWorkspace(root, slug)
	if err != nil {
		t.Fatal(err)
	}

	required := []ArtifactPath{"plan.md", "spec.md", "questions.md", "brief.md"}
	missing, diagnostics := observation.Missing(required)
	wantMissing := []ArtifactPath{"spec.md", "questions.md", "brief.md"}
	if !slices.Equal(missing, wantMissing) {
		t.Fatalf("Missing() paths = %v, want %v", missing, wantMissing)
	}
	wantDiagnostics := []ArtifactDiagnostic{{Path: "spec.md", State: ArtifactMalformed, Code: DiagnosticMalformedMarkdown}}
	if !slices.Equal(diagnostics, wantDiagnostics) {
		t.Fatalf("Missing() diagnostics = %+v, want %+v", diagnostics, wantDiagnostics)
	}
}

func TestWorkspaceObservationDisclosureRejectsInvalidWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hostile-root-secret", ".devrites")
	observation, err := ObserveWorkspace(root, "missing")
	if observation != nil {
		t.Fatalf("observation = %+v, want nil", observation)
	}
	assertObservationFailure(t, err, ObservationWorkspaceInvalid, "workspace observation: workspace_invalid: workspace is unavailable; verify the selected logical workspace and canonical workspace override, then retry")
	if strings.Contains(err.Error(), root) || strings.Contains(err.Error(), "hostile-root-secret") {
		t.Fatalf("error disclosed physical input: %v", err)
	}
}

func TestWorkspaceObservationRejectsCanonicalWorkspaceSymlinkWithoutDisclosure(t *testing.T) {
	for _, test := range []struct {
		name   string
		target func(t *testing.T, root string) string
	}{
		{
			name: "inside root",
			target: func(t *testing.T, root string) string {
				t.Helper()
				return filepath.Join(root, "retained-target-secret")
			},
		},
		{
			name: "outside root",
			target: func(t *testing.T, _ string) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "retained-target-secret")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			slug := "linked"
			target := test.target(t, root)
			writeFile(t, filepath.Join(target, LedgerFile), []byte("- Phase: build\ntarget-content-secret\n"))
			if err := os.MkdirAll(filepath.Join(root, "work"), 0o755); err != nil {
				t.Fatal(err)
			}
			workspace := filepath.Join(root, "work", slug)
			if err := os.Symlink(target, workspace); err != nil {
				t.Fatal(err)
			}
			t.Setenv("DEVRITES_WORKSPACE", "")

			observation, err := ObserveWorkspace(root, slug)
			if observation != nil {
				t.Fatalf("observation = %+v, want nil", observation)
			}
			assertObservationFailure(t, err, ObservationWorkspaceInvalid, "workspace observation: workspace_invalid: workspace is unavailable; verify the selected logical workspace and canonical workspace override, then retry")
			if strings.Contains(err.Error(), target) || strings.Contains(err.Error(), "retained-target-secret") || strings.Contains(err.Error(), "target-content-secret") {
				t.Fatalf("error disclosed target path or content: %v", err)
			}
		})
	}
}

func TestWorkspaceObservationRootRejectsIdentityChanges(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(t *testing.T, root, slug string)
	}{
		{
			name: "root replaced",
			mutate: func(t *testing.T, root, slug string) {
				t.Helper()
				if err := os.Rename(root, root+"-moved"); err != nil {
					t.Fatal(err)
				}
				// Replace with a regular file so OpenRoot fails closed on every
				// platform, including Windows directory file-ID reuse across
				// rename/recreate of an empty directory at the same path.
				if err := os.WriteFile(root, []byte("replaced-root\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "workspace replaced",
			mutate: func(t *testing.T, root, slug string) {
				t.Helper()
				workspace := filepath.Join(root, "work", slug)
				if err := os.Rename(workspace, workspace+"-moved"); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(workspace, []byte("replaced-workspace\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, slug := newObservationWorkspace(t)
			writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
			calls := 0
			observation, err := observeWorkspace(root, slug, func(stage observationStage, _ ArtifactPath) error {
				if stage == observationAfterResolve {
					calls++
					tc.mutate(t, root, slug)
				}
				return nil
			})
			if calls != 1 {
				t.Fatalf("after-resolve calls = %d, want 1", calls)
			}
			if observation != nil {
				t.Fatalf("observation = %+v, want nil", observation)
			}
			assertObservationFailure(t, err, ObservationWorkspaceInvalid, "workspace observation: workspace_invalid: workspace is unavailable; verify the selected logical workspace and canonical workspace override, then retry")
		})
	}
}

func TestWorkspaceObservationLimitsPerFile(t *testing.T) {
	for _, tc := range []struct {
		name      string
		size      int
		wantState ArtifactState
		wantCode  DiagnosticCode
		wantBytes int
	}{
		{name: "exactly one MiB", size: 1 << 20, wantState: ArtifactPresent, wantBytes: 1 << 20},
		{name: "one MiB plus one", size: (1 << 20) + 1, wantState: ArtifactUnsafe, wantCode: DiagnosticFileTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, slug := newObservationWorkspace(t)
			writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
			writeObservationArtifact(t, root, slug, "ai-spec.md", sizedMarkdown("optional input\n", tc.size))
			observation, err := ObserveWorkspace(root, slug)
			if err != nil {
				t.Fatal(err)
			}
			fact, ok := observation.Fact("ai-spec.md")
			if !ok || fact.State() != tc.wantState || len(fact.Bytes()) != tc.wantBytes {
				t.Fatalf("ai-spec fact = (%q, %d bytes, %v), want (%q, %d bytes)", fact.State(), len(fact.Bytes()), ok, tc.wantState, tc.wantBytes)
			}
			diagnostic, diagnosticOK := fact.Diagnostic()
			if tc.wantCode == "" {
				if diagnosticOK {
					t.Fatalf("Diagnostic() = %+v, want none", diagnostic)
				}
			} else if !diagnosticOK || diagnostic.Code != tc.wantCode {
				t.Fatalf("Diagnostic() = (%+v, %v), want %q", diagnostic, diagnosticOK, tc.wantCode)
			}
		})
	}
}

func TestWorkspaceObservationLimitsAggregateIncludingLaterOptionalFacts(t *testing.T) {
	for _, tc := range []struct {
		name     string
		overflow bool
	}{
		{name: "exactly eight MiB"},
		{name: "eight MiB plus one in later optional fact", overflow: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, slug := newObservationWorkspace(t)
			paths := []ArtifactPath{LedgerFile, "brief.md", "spec.md", "decisions.md", "assumptions.md", "questions.md", "decision-coverage.md", "strategy.md"}
			for _, path := range paths {
				prefix := "retained content\n"
				if path == LedgerFile {
					prefix = "- Phase: build\n"
				}
				writeObservationArtifact(t, root, slug, path, sizedMarkdown(prefix, 1<<20))
			}
			if tc.overflow {
				writeObservationArtifact(t, root, slug, "ai-spec.md", []byte("x"))
			}

			observation, err := ObserveWorkspace(root, slug)
			if !tc.overflow {
				if err != nil || observation == nil {
					t.Fatalf("ObserveWorkspace() = (%+v, %v), want observation", observation, err)
				}
				return
			}
			if observation != nil {
				t.Fatalf("observation = %+v, want nil", observation)
			}
			assertObservationFailure(t, err, ObservationAggregateTooLarge, "workspace observation: aggregate_too_large: retained content exceeds the 8 MiB aggregate limit; reduce retained Markdown below 8 MiB, then retry")
		})
	}
}

func TestWorkspaceObservationLimitsCountEmptyAndMalformedBytes(t *testing.T) {
	root, slug := newObservationWorkspace(t)
	presentPaths := []ArtifactPath{LedgerFile, "brief.md", "decisions.md", "assumptions.md", "questions.md", "decision-coverage.md", "strategy.md"}
	for _, path := range presentPaths {
		prefix := "retained content\n"
		if path == LedgerFile {
			prefix = "- Phase: build\n"
		}
		writeObservationArtifact(t, root, slug, path, sizedMarkdown(prefix, 1<<20))
	}
	writeObservationArtifact(t, root, slug, "design-brief.md", bytesOf(' ', 1<<20))
	writeObservationArtifact(t, root, slug, "ai-spec.md", []byte{0xff})

	observation, err := ObserveWorkspace(root, slug)
	if observation != nil {
		t.Fatalf("observation = %+v, want nil", observation)
	}
	assertObservationFailure(t, err, ObservationAggregateTooLarge, "workspace observation: aggregate_too_large: retained content exceeds the 8 MiB aggregate limit; reduce retained Markdown below 8 MiB, then retry")
}

func TestWorkspaceObservationConcurrentChangeOutranksCallbackErrorsAfterDetectedMutation(t *testing.T) {
	sentinel := errors.New("sentinel callback failure")
	for _, stage := range []observationStage{
		observationAfterInspect,
		observationBeforeOpen,
		observationBeforeRead,
		observationAfterRead,
	} {
		t.Run(observationStageName(stage), func(t *testing.T) {
			root, slug := newObservationWorkspace(t)
			writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
			writeObservationArtifact(t, root, slug, "brief.md", []byte("before\n"))
			artifactPath := observationArtifactPath(root, slug, "brief.md")
			calls := make(map[observationStage]int)

			observation, err := observeWorkspace(root, slug, func(current observationStage, artifact ArtifactPath) error {
				if artifact != "brief.md" {
					return nil
				}
				calls[current]++
				if current != stage {
					return nil
				}
				writeFile(t, artifactPath, []byte("detectably larger content\n"))
				return sentinel
			})
			if observation != nil {
				t.Fatalf("observation = %+v, want nil", observation)
			}
			assertObservationFailure(t, err, ObservationConcurrentChange, "workspace observation: concurrent_change: workspace changed during acquisition; retry")
			assertSingleObservationAttempt(t, calls, stage)
		})
	}
}

func TestWorkspaceObservationConcurrentChangeRejectsDetectedArtifactChangesWithoutRetry(t *testing.T) {
	for _, tc := range []struct {
		name   string
		stage  observationStage
		setup  func(t *testing.T, path string)
		mutate func(t *testing.T, path string)
	}{
		{
			name:   "appearance",
			stage:  observationAfterInspect,
			setup:  func(*testing.T, string) {},
			mutate: func(t *testing.T, path string) { writeFile(t, path, []byte("appeared\n")) },
		},
		{
			name:  "disappearance",
			stage: observationBeforeOpen,
			setup: func(t *testing.T, path string) { writeFile(t, path, []byte("before\n")) },
			mutate: func(t *testing.T, path string) {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:  "identity replacement",
			stage: observationBeforeOpen,
			setup: func(t *testing.T, path string) { writeFile(t, path, []byte("before\n")) },
			mutate: func(t *testing.T, path string) {
				if err := os.Rename(path, path+".old"); err != nil {
					t.Fatal(err)
				}
				writeFile(t, path, []byte("after!\n"))
			},
		},
		{
			name:  "type change",
			stage: observationBeforeOpen,
			setup: func(t *testing.T, path string) { writeFile(t, path, []byte("before\n")) },
			mutate: func(t *testing.T, path string) {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:   "size change",
			stage:  observationBeforeOpen,
			setup:  func(t *testing.T, path string) { writeFile(t, path, []byte("before\n")) },
			mutate: func(t *testing.T, path string) { writeFile(t, path, []byte("larger content\n")) },
		},
		{
			name:  "replacement after read",
			stage: observationAfterRead,
			setup: func(t *testing.T, path string) { writeFile(t, path, []byte("before\n")) },
			mutate: func(t *testing.T, path string) {
				if err := os.Rename(path, path+".old"); err != nil {
					t.Fatal(err)
				}
				writeFile(t, path, []byte("after!\n"))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, slug := newObservationWorkspace(t)
			writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
			artifactPath := observationArtifactPath(root, slug, "brief.md")
			tc.setup(t, artifactPath)
			stageCalls := make(map[observationStage]int)
			mutated := false
			observation, err := observeWorkspace(root, slug, func(stage observationStage, path ArtifactPath) error {
				if path != "brief.md" {
					return nil
				}
				stageCalls[stage]++
				if !mutated && stage == tc.stage {
					mutated = true
					tc.mutate(t, artifactPath)
				}
				return nil
			})
			if !mutated {
				t.Fatal("target artifact was not mutated")
			}
			assertSingleObservationAttempt(t, stageCalls, tc.stage)
			if observation != nil {
				t.Fatalf("observation = %+v, want nil", observation)
			}
			assertObservationFailure(t, err, ObservationConcurrentChange, "workspace observation: concurrent_change: workspace changed during acquisition; retry")
		})
	}
}

func TestWorkspaceObservationConcurrentChangeDoesNotClaimSameSizeOrTransientDetection(t *testing.T) {
	t.Run("same identity and size", func(t *testing.T) {
		root, slug := newObservationWorkspace(t)
		writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
		before := []byte("before\n")
		after := []byte("after!\n")
		writeObservationArtifact(t, root, slug, "brief.md", before)
		artifactPath := observationArtifactPath(root, slug, "brief.md")
		observation, err := observeWorkspace(root, slug, func(stage observationStage, artifact ArtifactPath) error {
			if stage == observationAfterRead && artifact == "brief.md" {
				writeFile(t, artifactPath, after)
			}
			return nil
		})
		if err != nil {
			if observation != nil {
				t.Fatalf("observation = %+v with error %v, want nil", observation, err)
			}
			assertObservationFailure(t, err, ObservationConcurrentChange, "workspace observation: concurrent_change: workspace changed during acquisition; retry")
			return
		}
		fact, ok := observation.Fact("brief.md")
		if !ok || fact.State() != ArtifactPresent || (!slices.Equal(fact.Bytes(), before) && !slices.Equal(fact.Bytes(), after)) {
			t.Fatalf("successful same-size acquisition returned incoherent fact: state=%q bytes=%q ok=%v", fact.State(), fact.Bytes(), ok)
		}
	})

	t.Run("transient appearance between probes", func(t *testing.T) {
		root, slug := newObservationWorkspace(t)
		writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
		artifactPath := observationArtifactPath(root, slug, "brief.md")
		observation, err := observeWorkspace(root, slug, func(stage observationStage, artifact ArtifactPath) error {
			if stage == observationAfterInspect && artifact == "brief.md" {
				writeFile(t, artifactPath, []byte("transient\n"))
				if removeErr := os.Remove(artifactPath); removeErr != nil {
					t.Fatal(removeErr)
				}
			}
			return nil
		})
		if err != nil {
			if observation != nil {
				t.Fatalf("observation = %+v with error %v, want nil", observation, err)
			}
			assertObservationFailure(t, err, ObservationConcurrentChange, "workspace observation: concurrent_change: workspace changed during acquisition; retry")
			return
		}
		fact, ok := observation.Fact("brief.md")
		if !ok || fact.State() != ArtifactAbsent || fact.Bytes() != nil {
			t.Fatalf("successful transient acquisition returned incoherent fact: state=%q bytes=%q ok=%v", fact.State(), fact.Bytes(), ok)
		}
	})
}

func TestWorkspaceObservationConcurrentChangeDoesNotApplyAfterReturn(t *testing.T) {
	root, slug := newObservationWorkspace(t)
	writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
	before := []byte("retained before return\n")
	writeObservationArtifact(t, root, slug, "brief.md", before)

	observation, err := ObserveWorkspace(root, slug)
	if err != nil {
		t.Fatal(err)
	}
	writeObservationArtifact(t, root, slug, "brief.md", []byte("changed after return\n"))

	fact, ok := observation.Fact("brief.md")
	if !ok || fact.State() != ArtifactPresent || !slices.Equal(fact.Bytes(), before) {
		t.Fatalf("post-return mutation changed retained fact: state=%q bytes=%q ok=%v", fact.State(), fact.Bytes(), ok)
	}
}

func assertSingleObservationAttempt(t *testing.T, calls map[observationStage]int, trigger observationStage) {
	t.Helper()
	if calls[trigger] != 1 {
		t.Fatalf("triggering stage %s calls = %d, want 1", observationStageName(trigger), calls[trigger])
	}
	for stage, count := range calls {
		if count > 1 {
			t.Fatalf("stage %s calls = %d, want at most 1", observationStageName(stage), count)
		}
	}
}

func observationStageName(stage observationStage) string {
	switch stage {
	case observationAfterResolve:
		return "after_resolve"
	case observationAfterInspect:
		return "after_inspect"
	case observationBeforeOpen:
		return "before_open"
	case observationBeforeRead:
		return "before_read"
	case observationAfterRead:
		return "after_read"
	case observationBeforeFinalSweep:
		return "before_final_sweep"
	default:
		return "unknown"
	}
}

func assertArtifactFact(t *testing.T, observation *WorkspaceObservation, path ArtifactPath, state ArtifactState, code DiagnosticCode, wantBytes []byte) {
	t.Helper()
	fact, ok := observation.Fact(path)
	if !ok {
		t.Fatalf("Fact(%q) returned ok=false", path)
	}
	if fact.Path() != path || fact.State() != state || !slices.Equal(fact.Bytes(), wantBytes) {
		t.Fatalf("Fact(%q) = (path %q, state %q, bytes %q), want (%q, %q, %q)", path, fact.Path(), fact.State(), fact.Bytes(), path, state, wantBytes)
	}
	diagnostic, diagnosticOK := fact.Diagnostic()
	if code == "" {
		if diagnosticOK || diagnostic != (ArtifactDiagnostic{}) {
			t.Fatalf("Fact(%q).Diagnostic() = (%+v, %v), want zero/false", path, diagnostic, diagnosticOK)
		}
		return
	}
	wantDiagnostic := ArtifactDiagnostic{Path: path, State: state, Code: code}
	if !diagnosticOK || diagnostic != wantDiagnostic {
		t.Fatalf("Fact(%q).Diagnostic() = (%+v, %v), want (%+v, true)", path, diagnostic, diagnosticOK, wantDiagnostic)
	}
}

func assertObservationFailure(t *testing.T, err error, want ObservationFailure, wantMessage string) {
	t.Helper()
	if err == nil || err.Error() != wantMessage {
		t.Fatalf("error = %v, want %q", err, wantMessage)
	}
	var failure ObservationFailure
	if !errors.As(err, &failure) || failure != want {
		t.Fatalf("errors.As(%v) = %q, want %q", err, failure, want)
	}
}

func equalArtifactFact(left, right ArtifactFact) bool {
	return left.path == right.path && left.state == right.state && left.code == right.code && slices.Equal(left.bytes, right.bytes)
}

func newObservationWorkspace(t *testing.T) (string, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".devrites")
	slug := "observed"
	if err := os.MkdirAll(filepath.Join(root, "work", slug), 0o755); err != nil {
		t.Fatal(err)
	}
	return root, slug
}

func writeObservationArtifact(t *testing.T, root, slug string, path ArtifactPath, body []byte) {
	t.Helper()
	writeFile(t, observationArtifactPath(root, slug, path), body)
}

func observationArtifactPath(root, slug string, path ArtifactPath) string {
	if path == ".devrites/principles.md" {
		return filepath.Join(root, "principles.md")
	}
	return filepath.Join(root, "work", slug, filepath.FromSlash(string(path)))
}

func writeFile(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func sizedMarkdown(prefix string, size int) []byte {
	body := bytesOf('x', size)
	copy(body, prefix)
	return body
}

func bytesOf(value byte, size int) []byte {
	body := make([]byte, size)
	for i := range body {
		body[i] = value
	}
	return body
}
