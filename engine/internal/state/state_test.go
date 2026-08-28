package state

// Focused unit tests on the deterministic core: phase→required mapping,
// present/empty detection, and status computation. These assert behavior
// (observable results), not internal structure.

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func expectedPhasePolicies() []PhasePolicy {
	sectionsSpec := []Section{SectionSpec}
	sectionsPlan := []Section{SectionSpec, SectionPlan}
	sectionsBuild := []Section{SectionSpec, SectionPlan, SectionDecisions, SectionTasks}
	sectionsProof := []Section{SectionSpec, SectionPlan, SectionDecisions, SectionTasks, SectionProof}
	sectionsComplete := []Section{SectionSpec, SectionPlan, SectionDecisions, SectionTasks, SectionProof, SectionStatus}

	artifactsFrame := []ArtifactPath{"state.md"}
	artifactsSpec := []ArtifactPath{"brief.md", "spec.md", "state.md", "decisions.md", "assumptions.md", "questions.md"}
	artifactsClarify := append(append([]ArtifactPath(nil), artifactsSpec...), "decision-coverage.md")
	artifactsPlan := append(append([]ArtifactPath(nil), artifactsClarify...), "architecture.md", "plan.md", "tasks.md", "traceability.md")
	artifactsVetted := append(append([]ArtifactPath(nil), artifactsPlan...), "eng-review.md", "test-plan.md")
	artifactsProof := append(append([]ArtifactPath(nil), artifactsVetted...), "evidence.md", "touched-files.md")
	artifactsFinal := append(append([]ArtifactPath(nil), artifactsProof...), "review.md", "seal.md")

	return []PhasePolicy{
		{Target: PhaseFrame, ResumeVerb: "frame", TransitionRight: "Frame an unstructured request before lifecycle work.", RequiredArtifacts: artifactsFrame},
		{Target: PhaseSpec, ResumeVerb: "spec", TransitionRight: "Author the product specification.", RequiredSections: sectionsSpec, RequiredArtifacts: artifactsSpec},
		{Target: PhaseClarify, ResumeVerb: "clarify", TransitionRight: "Close decision coverage in the written specification.", RequiredSections: sectionsSpec, RequiredArtifacts: artifactsClarify, BlocksOpenQuestions: true},
		{Target: PhaseTemper, ResumeVerb: "temper", TransitionRight: "Optionally challenge the clarified specification strategy.", RequiredSections: sectionsSpec, RequiredArtifacts: artifactsClarify, BlocksOpenQuestions: true},
		{Target: PhaseDefine, ResumeVerb: "define", TransitionRight: "Author and approve the initial implementation plan.", RequiredSections: sectionsPlan, RequiredArtifacts: artifactsPlan, BlocksOpenQuestions: true},
		{Target: PhasePlan, ResumeVerb: "vet", TransitionRight: "Hold the approved or repaired plan checkpoint for engineering review.", RequiredSections: sectionsPlan, RequiredArtifacts: artifactsPlan, BlocksOpenQuestions: true},
		{Target: PhaseVet, ResumeVerb: "vet", TransitionRight: "Review implementation readiness before build.", RequiredSections: sectionsBuild, RequiredArtifacts: artifactsVetted, BlocksOpenQuestions: true},
		{Target: PhaseBuild, ResumeVerb: "build", TransitionRight: "Implement the next approved vertical slice.", RequiredSections: sectionsBuild, RequiredArtifacts: artifactsVetted, BlocksOpenQuestions: true},
		{Target: PhaseConverge, ResumeVerb: "converge", TransitionRight: "Recover unmet clarified intent into new slices.", RequiredSections: sectionsBuild, RequiredArtifacts: artifactsVetted, BlocksOpenQuestions: true},
		{Target: PhaseProve, ResumeVerb: "prove", TransitionRight: "Produce acceptance evidence for the implementation.", RequiredSections: sectionsProof, RequiredArtifacts: artifactsProof, ProofRequired: true, BlocksOpenQuestions: true},
		{Target: PhasePolish, ResumeVerb: "polish", TransitionRight: "Apply the bounded quality pass.", RequiredSections: sectionsProof, RequiredArtifacts: artifactsProof, ProofRequired: true, BlocksOpenQuestions: true},
		{Target: PhaseReview, ResumeVerb: "review", TransitionRight: "Review the proven implementation.", RequiredSections: sectionsProof, RequiredArtifacts: artifactsProof, ProofRequired: true, BlocksOpenQuestions: true},
		{Target: PhaseSeal, ResumeVerb: "seal", TransitionRight: "Decide the final GO or NO-GO.", RequiredSections: sectionsComplete, RequiredArtifacts: artifactsFinal, ProofRequired: true, BlocksOpenQuestions: true, Shippable: true},
		{Target: PhaseShip, ResumeVerb: "ship", TransitionRight: "Perform authorized release and close-out mutations.", RequiredSections: sectionsComplete, RequiredArtifacts: artifactsFinal, ProofRequired: true, BlocksOpenQuestions: true, Shippable: true},
		{Target: PhaseDone, TransitionRight: "Represent archived completion with no resume command.", RequiredSections: sectionsComplete, RequiredArtifacts: artifactsFinal, ProofRequired: true, BlocksOpenQuestions: true, Shippable: true},
	}
}

func assertPhasePolicyEqual(t *testing.T, label string, got, want PhasePolicy) {
	t.Helper()
	if got.Target != want.Target ||
		got.ResumeVerb != want.ResumeVerb ||
		got.TransitionRight != want.TransitionRight ||
		!slices.Equal(got.RequiredSections, want.RequiredSections) ||
		!slices.Equal(got.RequiredArtifacts, want.RequiredArtifacts) ||
		got.ProofRequired != want.ProofRequired ||
		got.BlocksOpenQuestions != want.BlocksOpenQuestions ||
		got.Shippable != want.Shippable {
		t.Errorf("%s = %+v, want %+v", label, got, want)
	}
}

func policyForTest(t *testing.T, target Phase) PhasePolicy {
	t.Helper()
	policy, ok := PolicyFor(target)
	if !ok {
		t.Fatalf("PolicyFor(%q) returned unknown", target)
	}
	return policy
}

func TestPhasePoliciesExposeCanonicalLifecycle(t *testing.T) {
	want := expectedPhasePolicies()
	got := PhasePolicies()
	if len(got) != len(want) {
		t.Fatalf("len(PhasePolicies()) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		assertPhasePolicyEqual(t, "PhasePolicies", got[i], want[i])
		lookedUp, ok := PolicyFor(want[i].Target)
		if !ok {
			t.Fatalf("PolicyFor(%q) returned unknown", want[i].Target)
		}
		assertPhasePolicyEqual(t, "PolicyFor", lookedUp, want[i])
	}
	for _, unknown := range []Phase{"", "planning"} {
		policy, ok := PolicyFor(unknown)
		if ok {
			t.Errorf("PolicyFor(%q) = (%+v, true), want unknown lookup rejected", unknown, policy)
		}
		assertPhasePolicyEqual(t, "unknown PolicyFor result", policy, PhasePolicy{})
	}
}

func TestPolicyForReturnsDefensiveNestedCopies(t *testing.T) {
	want := expectedPhasePolicies()[7]
	policy := policyForTest(t, PhaseBuild)
	policy.RequiredSections[0] = SectionStatus
	policy.RequiredArtifacts[0] = "mutated.md"

	assertPhasePolicyEqual(t, "PolicyFor after mutation", policyForTest(t, PhaseBuild), want)
}

func TestPhasePoliciesReturnsDefensiveNestedCopies(t *testing.T) {
	want := expectedPhasePolicies()
	policies := PhasePolicies()
	policies[7].Target = PhaseDone
	policies[7].RequiredSections[0] = SectionStatus
	policies[7].RequiredArtifacts[0] = "mutated.md"

	assertPhasePolicyEqual(t, "sibling policy after mutation", policies[8], want[8])
	fresh := PhasePolicies()
	assertPhasePolicyEqual(t, "PhasePolicies after mutation", fresh[7], want[7])
}

func TestPrebuildWorkspaceRequirementsAreEnforcedByPhase(t *testing.T) {
	requireFiles := func(phase Phase, names ...string) {
		t.Helper()
		policy := policyForTest(t, phase)
		got := make(map[string]bool, len(policy.RequiredArtifacts))
		for _, artifact := range policy.RequiredArtifacts {
			got[string(artifact)] = true
		}
		for _, name := range names {
			if !got[name] {
				t.Errorf("phase %q does not require %s: %v", phase, name, policy.RequiredArtifacts)
			}
		}
	}

	requireFiles(PhaseClarify, "decision-coverage.md")
	requireFiles(PhaseTemper, "decision-coverage.md")
	requireFiles(PhasePlan, "decision-coverage.md")
	requireFiles(PhaseVet, "decision-coverage.md", "eng-review.md", "test-plan.md")
	requireFiles(PhaseBuild, "decision-coverage.md", "eng-review.md", "test-plan.md")
}

func TestWorkspaceObservationIncludesEveryLifecycleArtifact(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeWorkSection(t, root, "all-artifacts", "state.md", "- Phase: frame\n")
	writeWorkSection(t, root, "all-artifacts", "seal.md", "# Seal\n\nGO\n")

	observation, err := ObserveWorkspace(root, "all-artifacts")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[ArtifactPath]bool{}
	for _, policy := range PhasePolicies() {
		for _, artifact := range policy.RequiredArtifacts {
			expected[artifact] = true
		}
	}
	for artifact := range expected {
		if _, observed := observation.Fact(artifact); !observed {
			t.Errorf("observation does not include lifecycle artifact %q", artifact)
		}
	}
	seal, ok := observation.Fact("seal.md")
	if !ok || seal.State() != ArtifactPresent {
		t.Errorf("later-phase seal fact = (%q, %v), want present", seal.State(), ok)
	}
}

func TestRuntimeCompletenessUsesWorkspaceRequiredFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	for name, body := range map[string]string{
		"state.md":     "| Key | Value |\n| --- | --- |\n| phase | vet |\n",
		"spec.md":      "# Spec\n\nreal\n",
		"plan.md":      "# Plan\n\nreal\n",
		"decisions.md": "# Decisions\n\nreal\n",
		"tasks.md":     "# Tasks\n\nreal\n",
	} {
		writeWorkSection(t, root, "missing-vet", name, body)
	}

	report, err := Status(root, "missing-vet")
	if err != nil {
		t.Fatal(err)
	}
	if report.Complete() {
		t.Fatal("legacy sections made a vet workspace complete without workspaceRequired artifacts")
	}
	got := strings.Join(report.MissingFiles, ",")
	for _, want := range []string{"brief.md", "decision-coverage.md", "eng-review.md", "test-plan.md"} {
		if !strings.Contains(got, want) {
			t.Errorf("MissingFiles=%v, want %s", report.MissingFiles, want)
		}
	}
}

func TestRuntimeCompletenessRejectsEmptyRequiredFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	for _, artifact := range policyForTest(t, PhaseClarify).RequiredArtifacts {
		name := string(artifact)
		body := "# " + name + "\n\nreal\n"
		if name == "decision-coverage.md" {
			body = ""
		}
		writeWorkSection(t, root, "empty-coverage", name, body)
	}
	writeWorkSection(t, root, "empty-coverage", "state.md", "| phase | clarify |\n")

	report, err := Status(root, "empty-coverage")
	if err != nil {
		t.Fatal(err)
	}
	if report.Complete() || !slices.Contains(report.MissingFiles, "decision-coverage.md") {
		t.Fatalf("complete=%v missing=%v, want empty coverage missing", report.Complete(), report.MissingFiles)
	}
}

func TestPhasePolicyInvariants(t *testing.T) {
	policies := PhasePolicies()
	phaseNames := map[Phase]bool{}
	transitionRights := map[string]Phase{}
	sectionOrder := make(map[Section]int, len(Sections))
	for i, section := range Sections {
		sectionOrder[section] = i
	}

	for i, policy := range policies {
		if policy.Target == "" || phaseNames[policy.Target] {
			t.Fatalf("policy %d has empty or duplicate target %q", i, policy.Target)
		}
		phaseNames[policy.Target] = true
		if policy.TransitionRight == "" || transitionRights[policy.TransitionRight] != "" {
			t.Fatalf("phase %q has empty or duplicate transition right %q", policy.Target, policy.TransitionRight)
		}
		transitionRights[policy.TransitionRight] = policy.Target
		if len(policy.RequiredArtifacts) == 0 {
			t.Fatalf("phase %q has no required artifacts", policy.Target)
		}

		seenSections := map[Section]bool{}
		previousSectionIndex := -1
		for _, section := range policy.RequiredSections {
			index, known := sectionOrder[section]
			if !known || seenSections[section] || index <= previousSectionIndex {
				t.Fatalf("phase %q has non-canonical required sections %v", policy.Target, policy.RequiredSections)
			}
			seenSections[section] = true
			previousSectionIndex = index
		}
		seenArtifacts := map[ArtifactPath]bool{}
		for _, artifact := range policy.RequiredArtifacts {
			name := string(artifact)
			if artifact == "" || seenArtifacts[artifact] {
				t.Fatalf("phase %q has empty or duplicate required artifact %q", policy.Target, artifact)
			}
			if path.IsAbs(name) || path.Clean(name) != name || name == "." || strings.HasPrefix(name, "../") || strings.Contains(name, `\`) {
				t.Fatalf("phase %q has non-feature-relative required artifact %q", policy.Target, artifact)
			}
			if artifact == ".devrites/principles.md" {
				t.Fatalf("phase %q includes root principles in feature-relative requirements", policy.Target)
			}
			seenArtifacts[artifact] = true
		}

		if i > 0 {
			for _, section := range policies[i-1].RequiredSections {
				if !seenSections[section] {
					t.Errorf("phase %q dropped section %q required by %q", policy.Target, section, policies[i-1].Target)
				}
			}
			for _, artifact := range policies[i-1].RequiredArtifacts {
				if !seenArtifacts[artifact] {
					t.Errorf("phase %q dropped artifact %q required by %q", policy.Target, artifact, policies[i-1].Target)
				}
			}
		}
		if policy.Target == PhaseDone && policy.ResumeVerb != "" {
			t.Errorf("done resume verb = %q, want empty", policy.ResumeVerb)
		}
		if policy.Shippable && !policy.ProofRequired {
			t.Errorf("shippable phase %q does not require proof", policy.Target)
		}
	}
}

func TestStatusFixtureBuildIncomplete(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	for _, artifact := range policyForTest(t, PhaseBuild).RequiredArtifacts {
		name := string(artifact)
		body := "# " + name + "\n\nreal\n"
		if name == "state.md" {
			body = "- Phase: build\n"
		} else if name == "tasks.md" {
			body = "# Tasks\n"
		}
		writeWorkSection(t, root, "auth-tokens", name, body)
	}
	rep, err := Status(root, "auth-tokens")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Phase != PhaseBuild || rep.Complete() || !slices.Contains(rep.MissingFiles, "tasks.md") {
		t.Fatalf("phase=%q complete=%v missing=%v", rep.Phase, rep.Complete(), rep.MissingFiles)
	}
}

func TestStatusFixtureSpecComplete(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	for _, artifact := range policyForTest(t, PhaseSpec).RequiredArtifacts {
		name := string(artifact)
		body := "# " + name + "\n\nreal\n"
		if name == "state.md" {
			body = "- Phase: spec\n"
		}
		writeWorkSection(t, root, "search-ranking", name, body)
	}
	rep, err := Status(root, "search-ranking")
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Complete() {
		t.Fatalf("complete=false, missing=%v", rep.MissingFiles)
	}
}

func TestStatusRenderRemainsExactForCompleteAndOrdinaryMissingArtifacts(t *testing.T) {
	completeRoot := filepath.Join(t.TempDir(), ".devrites")
	writeStatusRequiredArtifacts(t, completeRoot, "complete", PhaseSpec)
	complete, err := Status(completeRoot, "complete")
	if err != nil {
		t.Fatal(err)
	}
	wantComplete := "feature: complete\n" +
		"phase: spec\n" +
		"  spec       present  required\n" +
		"  plan       empty\n" +
		"  decisions  present\n" +
		"  tasks      empty\n" +
		"  proof      empty\n" +
		"  status     present\n" +
		"result: complete\n"
	if got := complete.Render(); got != wantComplete {
		t.Fatalf("complete Render() =\n%q\nwant\n%q", got, wantComplete)
	}

	for _, tc := range []struct {
		name      string
		briefBody *string
	}{
		{name: "absent"},
		{name: "empty", briefBody: new(string)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			writeStatusRequiredArtifacts(t, root, tc.name, PhaseSpec)
			brief := filepath.Join(root, "work", tc.name, "brief.md")
			if tc.briefBody == nil {
				if err := os.Remove(brief); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(brief, []byte(*tc.briefBody), 0o644); err != nil {
				t.Fatal(err)
			}
			report, statusErr := Status(root, tc.name)
			if statusErr != nil {
				t.Fatal(statusErr)
			}
			want := "feature: " + tc.name + "\n" +
				"phase: spec\n" +
				"  spec       present  required\n" +
				"  plan       empty\n" +
				"  decisions  present\n" +
				"  tasks      empty\n" +
				"  proof      empty\n" +
				"  status     present\n" +
				"result: incomplete (missing files: brief.md)\n"
			if got := report.Render(); got != want {
				t.Fatalf("Render() =\n%q\nwant\n%q", got, want)
			}
		})
	}
}

func TestStatusRendersSelectedDiagnosticsBeforeResultWithoutRecovery(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeStatusRequiredArtifacts(t, root, "diagnostic", PhaseSpec)
	writeWorkSection(t, root, "diagnostic", "spec.md", "hostile-secret\x00")

	report, err := Status(root, "diagnostic")
	if err != nil {
		t.Fatal(err)
	}
	want := "feature: diagnostic\n" +
		"phase: spec\n" +
		"  spec       empty    required\n" +
		"  plan       empty\n" +
		"  decisions  present\n" +
		"  tasks      empty\n" +
		"  proof      empty\n" +
		"  status     present\n" +
		"artifact: spec.md: malformed (malformed_markdown)\n" +
		"result: incomplete (missing files: spec.md)\n"
	if got := report.Render(); got != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", got, want)
	}
	if strings.Contains(report.Render(), "next:") || strings.Contains(report.Render(), "hostile-secret") {
		t.Fatalf("Status disclosed content or emitted recovery:\n%s", report.Render())
	}
}

func TestStatusOmitsUnselectedEvidenceDiagnostic(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeStatusRequiredArtifacts(t, root, "unselected-evidence", PhaseSpec)
	writeWorkSection(t, root, "unselected-evidence", EvidenceFile, "hostile-secret\x00")

	report, err := Status(root, "unselected-evidence")
	if err != nil {
		t.Fatal(err)
	}
	want := "feature: unselected-evidence\n" +
		"phase: spec\n" +
		"  spec       present  required\n" +
		"  plan       empty\n" +
		"  decisions  present\n" +
		"  tasks      empty\n" +
		"  proof      empty\n" +
		"  status     present\n" +
		"result: complete\n"
	if got := report.Render(); got != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", got, want)
	}
	if strings.Contains(report.Render(), "artifact:") || strings.Contains(report.Render(), "next:") {
		t.Fatalf("unselected evidence emitted diagnostic or recovery:\n%s", report.Render())
	}
}

func TestStatusReturnsExactLogicalErrorsForUnusableState(t *testing.T) {
	for _, tc := range []struct {
		name     string
		prepare  func(t *testing.T, root, slug string)
		callback observationCallback
		want     string
	}{
		{
			name:    "absent",
			prepare: func(*testing.T, string, string) {},
			want:    `compute status: feature "broken": state.md is absent; add real content to state.md and retry`,
		},
		{
			name: "empty",
			prepare: func(t *testing.T, root, slug string) {
				writeWorkSection(t, root, slug, LedgerFile, "")
			},
			want: `compute status: feature "broken": state.md is empty; add real content to state.md and retry`,
		},
		{
			name: "malformed",
			prepare: func(t *testing.T, root, slug string) {
				writeWorkSection(t, root, slug, LedgerFile, "bad\x00state")
			},
			want: `compute status: feature "broken": state.md is malformed (malformed_markdown); repair state.md and retry`,
		},
		{
			name: "final symlink",
			prepare: func(t *testing.T, root, slug string) {
				target := filepath.Join(t.TempDir(), "state.md")
				if err := os.WriteFile(target, []byte("- Phase: build\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, filepath.Join(root, "work", slug, LedgerFile)); err != nil {
					t.Fatal(err)
				}
			},
			want: `compute status: feature "broken": state.md is unsafe (final_symlink); repair state.md and retry`,
		},
		{
			name: "nonregular",
			prepare: func(t *testing.T, root, slug string) {
				if err := os.Mkdir(filepath.Join(root, "work", slug, LedgerFile), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			want: `compute status: feature "broken": state.md is unsafe (non_regular); repair state.md and retry`,
		},
		{
			name: "too large",
			prepare: func(t *testing.T, root, slug string) {
				writeFile(t, filepath.Join(root, "work", slug, LedgerFile), sizedMarkdown("- Phase: build\n", (1<<20)+1))
			},
			want: `compute status: feature "broken": state.md is unsafe (file_too_large); repair state.md and retry`,
		},
		{
			name: "permission denied",
			prepare: func(t *testing.T, root, slug string) {
				writeWorkSection(t, root, slug, LedgerFile, "- Phase: build\n")
			},
			callback: func(stage observationStage, path ArtifactPath) error {
				if stage == observationBeforeOpen && path == LedgerFile {
					return fs.ErrPermission
				}
				return nil
			},
			want: `compute status: feature "broken": state.md is unreadable (permission_denied); repair state.md and retry`,
		},
		{
			name: "read failure",
			prepare: func(t *testing.T, root, slug string) {
				writeWorkSection(t, root, slug, LedgerFile, "- Phase: build\n")
			},
			callback: func(stage observationStage, path ArtifactPath) error {
				if stage == observationBeforeRead && path == LedgerFile {
					return errors.New("hostile read failure")
				}
				return nil
			},
			want: `compute status: feature "broken": state.md is unreadable (read_failure); repair state.md and retry`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			slug := "broken"
			if err := os.MkdirAll(filepath.Join(root, "work", slug), 0o755); err != nil {
				t.Fatal(err)
			}
			tc.prepare(t, root, slug)
			report, err := statusWithCallback(root, slug, tc.callback)
			if report != nil || err == nil || err.Error() != tc.want {
				t.Fatalf("statusWithCallback() = (%+v, %v), want nil and %q", report, err, tc.want)
			}
			if strings.Contains(err.Error(), root) || strings.Contains(err.Error(), "hostile") {
				t.Fatalf("error disclosed physical path or content: %v", err)
			}
		})
	}
}

func TestStatusUsesRetainedStateWithoutConsumerReopen(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeWorkSection(t, root, "retained", LedgerFile, "- Phase: build\n")
	path := filepath.Join(root, "work", "retained", LedgerFile)
	opens := 0
	reads := 0
	report, err := statusWithCallback(root, "retained", func(stage observationStage, artifact ArtifactPath) error {
		if artifact != LedgerFile {
			return nil
		}
		switch stage {
		case observationBeforeOpen:
			opens++
		case observationBeforeRead:
			reads++
		case observationAfterRead:
			if writeErr := os.WriteFile(path, []byte("- Phase: prove\n"), 0o644); writeErr != nil {
				return writeErr
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Phase != PhaseBuild || opens != 1 || reads != 1 {
		t.Fatalf("phase=%q opens=%d reads=%d, want retained build and 1/1", report.Phase, opens, reads)
	}
}

func TestStatusPreservesMissingAndUnknownPhaseErrors(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeWorkSection(t, root, "missing-phase", LedgerFile, "- Status: build\n")
	if _, err := Status(root, "missing-phase"); err == nil || err.Error() != `compute status: feature "missing-phase": no phase in state.md ledger; record phase in state.md and retry` {
		t.Fatalf("missing-phase error = %v", err)
	}
	writeWorkSection(t, root, "unknown-phase", LedgerFile, "- Phase: building\n")
	if _, err := Status(root, "unknown-phase"); err == nil || err.Error() != `compute status: feature "unknown-phase": unknown phase "building"; record a known phase in state.md and retry` {
		t.Fatalf("unknown-phase error = %v", err)
	}
}

func TestCanonicalWorkspaceCompletenessUsesConcretePhaseFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeWorkSection(t, root, "live", "state.md", "| Key | Value |\n| --- | --- |\n| phase | vet |\n")
	writeWorkSection(t, root, "live", "spec.md", "# Spec\n\nReady.\n")
	writeWorkSection(t, root, "live", "plan.md", "# Plan\n\nReady.\n")
	writeWorkSection(t, root, "live", "decisions.md", "# Decisions\n\nReady.\n")
	writeWorkSection(t, root, "live", "tasks.md", "# Tasks\n\nReady.\n")

	rep, err := Status(root, "live")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Complete() {
		t.Fatal("canonical vet workspace without decision/readiness artifacts reported complete")
	}
	for _, name := range []string{"decision-coverage.md", "eng-review.md", "test-plan.md"} {
		if !slices.Contains(rep.MissingFiles, name) {
			t.Errorf("MissingFiles = %v, want %s", rep.MissingFiles, name)
		}
	}
}

func TestFinalPhasesRequireReviewAndSealArtifacts(t *testing.T) {
	for _, phase := range []Phase{PhaseSeal, PhaseShip, PhaseDone} {
		required := policyForTest(t, phase).RequiredArtifacts
		for _, name := range []ArtifactPath{"review.md", "seal.md"} {
			if !slices.Contains(required, name) {
				t.Errorf("PolicyFor(%q).RequiredArtifacts = %v, want %s", phase, required, name)
			}
		}
	}
}

func TestStatusUnknownSlugErrors(t *testing.T) {
	if _, err := Status(filepath.Join(t.TempDir(), ".devrites"), "nope"); err == nil {
		t.Fatal("Status on unknown slug returned nil error, want an error")
	}
}

func writeStatusRequiredArtifacts(t *testing.T, root, slug string, phase Phase) {
	t.Helper()
	for _, artifact := range policyForTest(t, phase).RequiredArtifacts {
		body := "# Artifact\n\nreal content\n"
		if artifact == LedgerFile {
			body = "- Phase: " + string(phase) + "\n"
		}
		writeWorkSection(t, root, slug, string(artifact), body)
	}
}

func writeSection(t *testing.T, root, slug, name, body string) {
	t.Helper()
	dir := filepath.Join(root, "work", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeWorkSection(t *testing.T, root, slug, name, body string) {
	t.Helper()
	dir := filepath.Join(root, "work", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A live workspace map need not carry frontmatter: the phase lives in
// the canonical state.md ledger and the proof/status concepts are satisfied by
// evidence.md/state.md. The engine must load, list, and report it anyway.
func TestStatusFromOfficialBulletLedger(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeSection(t, root, "live", "state.md", "- Phase: prove\n- Status: running\n")
	writeSection(t, root, "live", "spec.md", "# Spec\n\nDo the thing.\n")
	writeSection(t, root, "live", "plan.md", "# Plan\n\nApproach.\n")
	writeSection(t, root, "live", "decisions.md", "# Decisions\n\nChose X.\n")
	writeSection(t, root, "live", "tasks.md", "# Tasks\n\n- [x] slice 1\n")
	writeSection(t, root, "live", "evidence.md", "# Evidence\n\nTests pass.\n")

	rep, err := Status(root, "live")
	if err != nil {
		t.Fatalf("Status on a manifest-less live workspace = %v, want nil", err)
	}
	if rep.Phase != PhaseProve {
		t.Errorf("phase = %q, want prove (from the state.md ledger)", rep.Phase)
	}
	rendered := rep.Render()
	if !strings.Contains(rendered, "  proof      present  required\n") {
		t.Errorf("proof section should be present via canonical evidence.md:\n%s", rendered)
	}
	if !strings.Contains(rendered, "  status     present\n") {
		t.Errorf("status section should be present via canonical state.md:\n%s", rendered)
	}

	slugs, err := ListFeatures(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(slugs) != 1 || slugs[0] != "live" {
		t.Errorf("ListFeatures = %v, want [live] (a ledger-only dir must list)", slugs)
	}
}

func TestStateLedgerIgnoresOptionalREADMEFrontmatter(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeWorkSection(t, root, "live", "README.md", "---\nphase: unsupported\nschemaVersion: 999\ninvalid: [\n---\noptional notes\xff\n")
	writeWorkSection(t, root, "live", "state.md", "| Key | Value |\n| --- | --- |\n| phase | temper |\n")
	writeWorkSection(t, root, "live", "spec.md", "# Spec\n\nReady.\n")

	rep, err := Status(root, "live")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Phase != PhaseTemper {
		t.Fatalf("phase=%q, want current ledger phase %q", rep.Phase, PhaseTemper)
	}
}

func TestStatusRejectsCorruptStateMarkdownWithoutContentDisclosure(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	dir := filepath.Join(root, "work", "corrupt")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state.md"), []byte("| phase | build |\x00\n| schema | 3 |\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	want := `compute status: feature "corrupt": state.md is malformed (malformed_markdown); repair state.md and retry`
	if _, err := Status(root, "corrupt"); err == nil || err.Error() != want {
		t.Fatalf("Status() error = %v, want %q", err, want)
	}
}

func TestOnlyCanonicalWorkLayoutIsDiscovered(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeWorkSection(t, root, "live", "state.md", "- Phase: build\n")
	writeWorkSection(t, root, "live", "spec.md", "# Spec\n\nDo the thing.\n")
	legacy := filepath.Join(root, "features", "alias")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "state.md"), []byte("- Phase: spec\n- Schema: 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(root, "alias"); err == nil {
		t.Fatal("features/<slug> compatibility layout was accepted")
	}

	slugs, err := ListFeatures(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(slugs, ","); got != "live" {
		t.Fatalf("ListFeatures = %v, want [live]", slugs)
	}
}

func TestSpeculativeWorkspaceAliasesAreRejected(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	aliasDir := filepath.Join(root, "work", "aliases")
	if err := os.MkdirAll(aliasDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"feature.md", "index.md"} {
		if err := os.WriteFile(filepath.Join(aliasDir, name), []byte("---\nphase: spec\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(aliasDir, "status.md"), []byte("- Status: spec\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Status(root, "aliases"); err == nil {
		t.Fatal("feature.md/index.md/status.md aliases created a feature without state.md")
	}
	if slugs, err := ListFeatures(root); err != nil {
		t.Fatal(err)
	} else if len(slugs) != 0 {
		t.Fatalf("alias-only workspace was discovered: %v", slugs)
	}

	writeWorkSection(t, root, "aliases", "state.md", "- Phase: prove\n")
	writeWorkSection(t, root, "aliases", "proof.md", "# Proof\n\nOld alias content.\n")
	report, err := Status(root, "aliases")
	if err != nil {
		t.Fatal(err)
	}
	observation, err := ObserveWorkspace(root, "aliases")
	if err != nil {
		t.Fatal(err)
	}
	evidence, ok := observation.Fact(EvidenceFile)
	if report.present[SectionProof] || !ok || evidence.State() != ArtifactAbsent {
		t.Fatal("proof.md alias satisfied canonical evidence.md presence")
	}
}

func TestStatusCursorCannotStandInForPhase(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeWorkSection(t, root, "status-only", "state.md", "- Status: build\n")
	if _, err := Status(root, "status-only"); err == nil || !strings.Contains(err.Error(), "no phase in state.md") {
		t.Fatalf("status-as-phase error = %v, want explicit missing phase", err)
	}
}

// An unknown phase word in the ledger is rejected, so a ledger-only feature is a
// clear error rather than silently falling back or mis-loading.
func TestLedgerPhaseRejectsUnknownWord(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeSection(t, root, "bogus", "state.md", "- Phase: building\n")
	if _, err := Status(root, "bogus"); err == nil {
		t.Error("Status on a ledger with an unknown phase word = nil error, want an error")
	}
}

func TestArtifactClassificationDistinguishesContentFromStubs(t *testing.T) {
	cases := []struct {
		name string
		body string
		want ArtifactState
	}{
		{"empty", "", ArtifactEmpty},
		{"heading-only stub", "# Tasks\n", ArtifactEmpty},
		{"frontmatter-only", "---\nphase: build\n---\n", ArtifactEmpty},
		{"real content", "# Spec\n\nDo the thing.\n", ArtifactPresent},
		{"content after frontmatter", "---\nk: v\n---\n\nreal words\n", ArtifactPresent},
		{"fenced content", "# Notes\n\n```\nexample\n```\n", ArtifactPresent},
		{"NUL content", "# Notes\n\nbad\x00\n", ArtifactMalformed},
		{"malformed UTF-8", "# Notes\n\nbad\xff\n", ArtifactMalformed},
	}
	for _, tc := range cases {
		state, _ := classifyArtifact([]byte(tc.body))
		if state != tc.want {
			t.Errorf("%s: classifyArtifact state=%q, want %q", tc.name, state, tc.want)
		}
	}
}

func TestResolveRootAcceptsProjectRootOrDevritesRoot(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, in := range []string{project, root} {
		got, err := ResolveRoot(in)
		if err != nil {
			t.Fatalf("ResolveRoot(%q): %v", in, err)
		}
		want, err := filepath.EvalSymlinks(root)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("ResolveRoot(%q) = %q, want canonical %q", in, got, want)
		}
	}
}

func TestResolveRootUsesExplicitOverrideBeforeGitBoundedImplicitRoot(t *testing.T) {
	base := t.TempDir()
	implicitProject := filepath.Join(base, "implicit")
	explicitProject := filepath.Join(base, "explicit")
	for _, project := range []string{implicitProject, explicitProject} {
		if err := os.MkdirAll(filepath.Join(project, ".devrites"), 0o755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("git", "-C", project, "init", "-q")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git init: %v\n%s", err, out)
		}
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(implicitProject); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	implicit, err := ResolveRoot("")
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := ResolveRoot(explicitProject)
	if err != nil {
		t.Fatal(err)
	}
	if implicit == explicit {
		t.Fatalf("explicit override %q did not replace implicit root %q", explicit, implicit)
	}
	if want, _ := filepath.EvalSymlinks(filepath.Join(explicitProject, ".devrites")); explicit != want {
		t.Fatalf("explicit root = %q, want %q", explicit, want)
	}
}

func TestResolveRootRejectsExternalWorkspaceOverride(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_WORKSPACE", filepath.Join(t.TempDir(), "feature"))
	if _, err := ResolveRoot(project); err == nil || !strings.Contains(err.Error(), "unsafe DevRites root") {
		t.Fatalf("ResolveRoot error = %v, want unsafe external workspace refusal", err)
	}
}

func TestListFeaturesIgnoresAllOperationalRemnants(t *testing.T) {
	root := t.TempDir()
	names := []string{
		"native-engine-cleanup",
		"native-engine-cleanup-s1",
		"native-engine-cleanup-s10",
		"native-engine-cleanup-s11",
		"native-engine-cleanup-s12",
		"native-engine-cleanup-s13",
		"native-engine-cleanup-s14",
		"native-engine-cleanup-s15",
		"native-engine-cleanup-s16",
		"native-engine-cleanup-s16b",
		"native-engine-cleanup-s17",
		"native-engine-cleanup-s18",
		"native-engine-cleanup-s19",
		"native-engine-cleanup-s2",
		"native-engine-cleanup-s20",
		"native-engine-cleanup-s21",
		"native-engine-cleanup-s22",
		"native-engine-cleanup-s23",
		"native-engine-cleanup-s24",
		"native-engine-cleanup-s3",
		"native-engine-cleanup-s3b",
		"native-engine-cleanup-s4",
		"native-engine-cleanup-s5a",
		"native-engine-cleanup-s5b",
		"native-engine-cleanup-s6a",
		"native-engine-cleanup-s6b",
		"native-engine-cleanup-s7",
		"native-engine-cleanup-s8",
		"native-engine-cleanup-s9",
	}
	for i, name := range names {
		dir := filepath.Join(root, "work", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		switch {
		case i == 0:
			if err := os.WriteFile(filepath.Join(dir, ".wright-allowlist"), []byte("bounded paths\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		case i == 14:
			if err := os.WriteFile(filepath.Join(dir, "recovery-attempts.jsonl"), []byte("{}\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	live := filepath.Join(root, "work", "live")
	if err := os.MkdirAll(live, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(live, LedgerFile), []byte("| phase | frame |\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ListFeatures(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "live" {
		t.Fatalf("ListFeatures() = %v, want [live]; operational remnants became workspaces", got)
	}
}
