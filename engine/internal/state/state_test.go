package state

// Focused unit tests on the deterministic core: phase→required mapping,
// present/empty detection, and status computation. These assert behavior
// (observable results), not internal structure.

import (
	"os"
	"os/exec"
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

func TestLoadFeatureObservesEveryLifecycleArtifact(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeWorkSection(t, root, "all-artifacts", "state.md", "- Phase: frame\n")
	writeWorkSection(t, root, "all-artifacts", "seal.md", "# Seal\n\nGO\n")

	feature, err := LoadFeature(root, "all-artifacts")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]bool{}
	for _, policy := range PhasePolicies() {
		for _, artifact := range policy.RequiredArtifacts {
			expected[string(artifact)] = true
		}
	}
	if len(feature.PresentFiles) != len(expected) {
		t.Fatalf("PresentFiles has %d artifacts, want %d: %v", len(feature.PresentFiles), len(expected), feature.PresentFiles)
	}
	for name := range expected {
		if _, observed := feature.PresentFiles[name]; !observed {
			t.Errorf("PresentFiles does not observe lifecycle artifact %q", name)
		}
	}
	if !feature.PresentFiles["seal.md"] {
		t.Error("PresentFiles did not inspect a later-phase artifact for a frame workspace")
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
			if artifact == "" || seenArtifacts[artifact] {
				t.Fatalf("phase %q has empty or duplicate required artifact %q", policy.Target, artifact)
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
func TestLoadFeatureFromOfficialBulletLedger(t *testing.T) {
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
	if !rep.Present[SectionProof] {
		t.Error("proof section should be present via canonical evidence.md")
	}
	if !rep.Present[SectionStatus] {
		t.Error("status section should be present via canonical state.md")
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

func TestLoadFeatureRejectsCorruptStateMarkdown(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	dir := filepath.Join(root, "work", "corrupt")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state.md"), []byte("| phase | build |\x00\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFeature(root, "corrupt"); err == nil || !strings.Contains(err.Error(), "NUL") {
		t.Fatalf("LoadFeature() error = %v, want NUL rejection", err)
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
	if err := os.WriteFile(filepath.Join(legacy, "state.md"), []byte("- Phase: spec\n"), 0o644); err != nil {
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

	if _, err := LoadFeature(root, "aliases"); err == nil {
		t.Fatal("feature.md/index.md/status.md aliases created a feature without state.md")
	}
	if slugs, err := ListFeatures(root); err != nil {
		t.Fatal(err)
	} else if len(slugs) != 0 {
		t.Fatalf("alias-only workspace was discovered: %v", slugs)
	}

	writeWorkSection(t, root, "aliases", "state.md", "- Phase: prove\n")
	writeWorkSection(t, root, "aliases", "proof.md", "# Proof\n\nOld alias content.\n")
	feature, err := LoadFeature(root, "aliases")
	if err != nil {
		t.Fatal(err)
	}
	if feature.Present[SectionProof] || feature.PresentFiles[EvidenceFile] {
		t.Fatal("proof.md alias satisfied canonical evidence.md presence")
	}
}

func TestStatusCursorCannotStandInForPhase(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeWorkSection(t, root, "status-only", "state.md", "- Status: build\n")
	if _, err := LoadFeature(root, "status-only"); err == nil || !strings.Contains(err.Error(), "no phase in state.md") {
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

func TestSectionPresentDistinguishesContentFromStubs(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"missing", "", false}, // no file written for this case
		{"empty", "", false},
		{"heading-only stub", "# Tasks\n", false},
		{"frontmatter-only", "---\nphase: build\n---\n", false},
		{"real content", "# Spec\n\nDo the thing.\n", true},
		{"content after frontmatter", "---\nk: v\n---\n\nreal words\n", true},
		{"fenced content", "# Notes\n\n```\nexample\n```\n", true},
		{"NUL content", "# Notes\n\nbad\x00\n", false},
		{"malformed UTF-8", "# Notes\n\nbad\xff\n", false},
	}
	for i, c := range cases {
		path := filepath.Join(dir, "s.md")
		_ = os.Remove(path)
		if c.name != "missing" {
			if err := os.WriteFile(path, []byte(c.body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if got := sectionPresent(path); got != c.want {
			t.Errorf("case %d (%s): sectionPresent = %v, want %v", i, c.name, got, c.want)
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
