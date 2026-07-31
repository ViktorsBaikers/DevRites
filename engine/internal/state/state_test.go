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

func TestRequiredSectionsIsPhaseRelativeAndOrdered(t *testing.T) {
	got := RequiredSections(PhaseBuild)
	want := []Section{SectionSpec, SectionPlan, SectionDecisions, SectionTasks}
	if len(got) != len(want) {
		t.Fatalf("RequiredSections(build) = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
	if len(RequiredSections(PhaseFrame)) != 0 {
		t.Errorf("frame should require no sections, got %v", RequiredSections(PhaseFrame))
	}
}

func TestLifecycleRegistryOwnsOrderAndResumeCommands(t *testing.T) {
	phases := LifecyclePhases()
	if len(phases) == 0 || phases[0] != PhaseFrame || phases[len(phases)-1] != PhaseDone {
		t.Fatalf("LifecyclePhases()=%v, want frame...done", phases)
	}
	wantPrefix := []Phase{PhaseFrame, PhaseSpec, PhaseClarify, PhaseTemper, PhaseDefine}
	if len(phases) < len(wantPrefix) {
		t.Fatalf("LifecyclePhases()=%v, want prefix %v", phases, wantPrefix)
	}
	for i, want := range wantPrefix {
		if phases[i] != want {
			t.Fatalf("LifecyclePhases()[%d]=%q, want %q (full=%v)", i, phases[i], want, phases)
		}
	}
	if got := ResumeVerb(PhasePlan); got != "vet" {
		t.Fatalf("ResumeVerb(plan)=%q, want vet", got)
	}
	if got := ResumeVerb(PhaseDone); got != "" {
		t.Fatalf("ResumeVerb(done)=%q, want empty", got)
	}

	phases[0] = PhaseDone
	if got := LifecyclePhases()[0]; got != PhaseFrame {
		t.Fatalf("LifecyclePhases exposed mutable registry: first=%q", got)
	}
}

func TestPrebuildWorkspaceRequirementsAreEnforcedByPhase(t *testing.T) {
	requireFiles := func(phase Phase, names ...string) {
		t.Helper()
		definition, ok := definitionFor(phase)
		if !ok {
			t.Fatalf("missing phase definition for %q", phase)
		}
		got := make(map[string]bool, len(definition.workspaceRequired))
		for _, name := range definition.workspaceRequired {
			got[name] = true
		}
		for _, name := range names {
			if !got[name] {
				t.Errorf("phase %q does not require %s: %v", phase, name, definition.workspaceRequired)
			}
		}
	}

	requireFiles(PhaseClarify, "decision-coverage.md")
	requireFiles(PhaseTemper, "decision-coverage.md")
	requireFiles(PhasePlan, "decision-coverage.md")
	requireFiles(PhaseVet, "decision-coverage.md", "eng-review.md", "test-plan.md")
	requireFiles(PhaseBuild, "decision-coverage.md", "eng-review.md", "test-plan.md")
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
	for _, name := range RequiredWorkspaceFiles(PhaseClarify) {
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

func TestLifecycleRegistryInvariants(t *testing.T) {
	phaseNames := map[Phase]bool{}
	transitionRights := map[string]Phase{}
	for i, definition := range phaseDefinitions {
		if definition.phase == "" || phaseNames[definition.phase] {
			t.Fatalf("phase definition %d has empty or duplicate ID %q", i, definition.phase)
		}
		phaseNames[definition.phase] = true
		if definition.transitionRight == "" || transitionRights[definition.transitionRight] != "" {
			t.Fatalf("phase %q has empty or duplicate transition right %q", definition.phase, definition.transitionRight)
		}
		transitionRights[definition.transitionRight] = definition.phase
		if len(definition.workspaceRequired) == 0 {
			t.Fatalf("phase %q has no workspace requirements", definition.phase)
		}
		for _, section := range definition.required {
			known := false
			for _, canonical := range Sections {
				known = known || section == canonical
			}
			if !known {
				t.Fatalf("phase %q requires unknown section %q", definition.phase, section)
			}
		}
		if definition.shippable && !definition.proofRequired {
			t.Fatalf("shippable phase %q does not require proof", definition.phase)
		}
	}
}

func TestStatusFixtureBuildIncomplete(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	for _, name := range RequiredWorkspaceFiles(PhaseBuild) {
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
	for _, name := range RequiredWorkspaceFiles(PhaseSpec) {
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
		required := RequiredWorkspaceFiles(phase)
		for _, name := range []string{"review.md", "seal.md"} {
			if !slices.Contains(required, name) {
				t.Errorf("RequiredWorkspaceFiles(%q) = %v, want %s", phase, required, name)
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
