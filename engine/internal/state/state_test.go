package state

// Focused unit tests on the deterministic core: phase→required mapping,
// present/empty detection, and status computation. These assert behavior
// (observable results), not internal structure.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureRoot = "../../testdata/fixtures/basic/devrites-root"

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
	if got := ResumeVerb(PhasePlan); got != "define" {
		t.Fatalf("ResumeVerb(plan)=%q, want define", got)
	}
	if got := ResumeVerb(PhaseDone); got != "" {
		t.Fatalf("ResumeVerb(done)=%q, want empty", got)
	}

	phases[0] = PhaseDone
	if got := LifecyclePhases()[0]; got != PhaseFrame {
		t.Fatalf("LifecyclePhases exposed mutable registry: first=%q", got)
	}
}

func TestLifecycleRegistryInvariants(t *testing.T) {
	phaseNames := map[Phase]bool{}
	aliases := map[string]Phase{}
	for i, definition := range phaseDefinitions {
		if definition.phase == "" || phaseNames[definition.phase] {
			t.Fatalf("phase definition %d has empty or duplicate ID %q", i, definition.phase)
		}
		phaseNames[definition.phase] = true
		if len(definition.workspaceRequired) == 0 {
			t.Fatalf("phase %q has no workspace requirements", definition.phase)
		}
		for _, alias := range definition.aliases {
			if alias == "" || aliases[alias] != "" || KnownPhase(Phase(alias)) {
				t.Fatalf("phase %q has empty or duplicate alias %q", definition.phase, alias)
			}
			aliases[alias] = definition.phase
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
	rep, err := Status(fixtureRoot, "auth-tokens")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Phase != PhaseBuild {
		t.Errorf("phase = %q, want build", rep.Phase)
	}
	// tasks.md is a heading-only stub → empty → the one missing required section.
	if len(rep.Missing) != 1 || rep.Missing[0] != SectionTasks {
		t.Errorf("Missing = %v, want [tasks]", rep.Missing)
	}
	if rep.Complete() {
		t.Error("Complete() = true, want false")
	}
	// proof is empty but not required at build, so it must not count as missing.
	if rep.Required[SectionProof] {
		t.Error("proof should not be required during the build phase")
	}
}

func TestStatusFixtureSpecComplete(t *testing.T) {
	rep, err := Status(fixtureRoot, "search-ranking")
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Complete() {
		t.Errorf("Complete() = false, want true (missing: %v)", rep.Missing)
	}
}

func TestStatusUnknownSlugErrors(t *testing.T) {
	if _, err := Status(fixtureRoot, "nope"); err == nil {
		t.Fatal("Status on unknown slug returned nil error, want an error")
	}
}

func writeSection(t *testing.T, root, slug, name, body string) {
	t.Helper()
	dir := filepath.Join(root, "features", slug)
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

// A live workspace the pack creates has no feature.md manifest: the phase lives in
// the state.md ledger and the proof/status sections are satisfied by their aliases
// (evidence.md / state.md). The engine must load, list, and report it anyway.
func TestLoadFeatureFromLedgerAndAliases(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeSection(t, root, "live", "state.md", "- Phase: prove\n- Status: running\n")
	writeSection(t, root, "live", "spec.md", "# Spec\n\nDo the thing.\n")
	writeSection(t, root, "live", "plan.md", "# Plan\n\nApproach.\n")
	writeSection(t, root, "live", "decisions.md", "# Decisions\n\nChose X.\n")
	writeSection(t, root, "live", "tasks.md", "# Tasks\n\n- [x] slice 1\n")
	writeSection(t, root, "live", "evidence.md", "# Evidence\n\nTests pass.\n") // alias for proof

	rep, err := Status(root, "live")
	if err != nil {
		t.Fatalf("Status on a manifest-less live workspace = %v, want nil", err)
	}
	if rep.Phase != PhaseProve {
		t.Errorf("phase = %q, want prove (from the state.md ledger)", rep.Phase)
	}
	if !rep.Present[SectionProof] {
		t.Error("proof section should be present via its evidence.md alias")
	}
	if !rep.Present[SectionStatus] {
		t.Error("status section should be present via its state.md alias")
	}
	if !rep.Complete() {
		t.Errorf("prove-phase feature should be complete, missing: %v", rep.Missing)
	}

	slugs, err := ListFeatures(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(slugs) != 1 || slugs[0] != "live" {
		t.Errorf("ListFeatures = %v, want [live] (a ledger-only dir must list)", slugs)
	}
}

func TestLedgerPhaseOverridesStaleManifestPhase(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeWorkSection(t, root, "live", "feature.md", "---\nphase: spec\nschemaVersion: 1\n---\n")
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

func TestSnapshotUsesCanonicalNextActionAndWarnsWhenRequiredProofMissing(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeWorkSection(t, root, "live", "state.md", `# State

| Key | Value |
| --- | --- |
| phase | review |
| status | running |
| next_action | /rite-seal after review is clean |
`)
	for name, body := range map[string]string{
		"spec.md":      "# Spec\n\nReady.\n",
		"plan.md":      "# Plan\n\nReady.\n",
		"decisions.md": "# Decisions\n\nReady.\n",
		"tasks.md":     "# Tasks\n\nReady.\n",
	} {
		writeWorkSection(t, root, "live", name, body)
	}

	snap, err := Snapshot(root, "live")
	if err != nil {
		t.Fatal(err)
	}
	if snap.NextCommands.Verb != "seal" || snap.NextCommand != "/rite-seal" {
		t.Fatalf("next commands=%+v legacy=%q, want canonical next_action seal", snap.NextCommands, snap.NextCommand)
	}
	if got := strings.Join(snap.Warnings, "\n"); !strings.Contains(got, "requires fresh evidence") {
		t.Fatalf("warnings=%v, want missing required-proof warning", snap.Warnings)
	}
}

func TestSnapshotReadsCanonicalActiveSliceAndCountsQuestionsByRecord(t *testing.T) {
	workDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workDir, "state.md"), []byte("| phase | build |\n| active_slice | SLICE-002 |\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "tasks.md"), []byte("## SLICE-001 First\n\n## SLICE-002 Second\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "questions.md"), []byte("## Q-001\nstatus: open\ngate: blocking\n\n## Q-002\nstatus: answered\ngate: blocking\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	slice := currentSlice(workDir)
	if slice == nil || slice.Name != "SLICE-002" || slice.Index != 2 || slice.Total != 2 {
		t.Fatalf("currentSlice=%+v, want canonical SLICE-002 at 2/2", slice)
	}
	if drift := driftSummary(workDir); drift.Status != "open" || drift.Open != 1 {
		t.Fatalf("driftSummary=%+v, want one open question record", drift)
	}
}

func TestWorkLayoutIsCanonicalAndFeaturesIsAlias(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeWorkSection(t, root, "live", "state.md", "- Phase: build\n")
	writeWorkSection(t, root, "live", "spec.md", "# Spec\n\nDo the thing.\n")
	writeSection(t, root, "alias", "state.md", "- Phase: spec\n")
	writeSection(t, root, "alias", "spec.md", "# Spec\n\nAlias.\n")

	for _, slug := range []string{"live", "alias"} {
		if _, err := Status(root, slug); err != nil {
			t.Fatalf("Status(%q) = %v, want nil", slug, err)
		}
	}

	slugs, err := ListFeatures(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(slugs, ","); got != "alias,live" {
		t.Fatalf("ListFeatures = %v, want [alias live]", slugs)
	}
}

// An unknown phase word in the ledger is rejected, so a ledger-only feature is a
// clear error rather than silently falling back or mis-loading.
func TestLedgerPhaseRejectsUnknownWord(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeSection(t, root, "bogus", "state.md", "- Phase: banana\n")
	if _, err := Status(root, "bogus"); err == nil {
		t.Error("Status on a ledger with an unknown phase word = nil error, want an error")
	}
}

func writeFeatureMD(t *testing.T, root, slug, featureMD string) {
	t.Helper()
	dir := filepath.Join(root, "features", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "feature.md"), []byte(featureMD), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadFeatureRejectsBadFrontmatter(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	cases := map[string]string{
		"no-phase":  "---\ntitle: x\n---\n\nbody\n",             // documented: missing phase is an error
		"bad-phase": "---\nphase: banana\n---\n",                // documented: unknown phase is an error
		"future":    "---\nphase: spec\nschemaVersion: 99\n---", // schemaVersion newer than the engine
		"bad-ver":   "---\nphase: spec\nschemaVersion: x\n---",  // non-numeric schemaVersion
	}
	for slug, md := range cases {
		writeFeatureMD(t, root, slug, md)
	}
	for slug := range cases {
		if _, err := Status(root, slug); err == nil {
			t.Errorf("Status(%q) = nil error, want an error", slug)
		}
	}
}

func TestLoadFeatureAcceptsSupportedSchemaVersion(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeFeatureMD(t, root, "ok", "---\nphase: spec\nschemaVersion: 1\n---\n")
	if _, err := Status(root, "ok"); err != nil {
		t.Errorf("Status on schemaVersion 1 = %v, want nil", err)
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
		if got != root {
			t.Fatalf("ResolveRoot(%q) = %q, want %q", in, got, root)
		}
	}
}

func TestDevritesWorkspaceOverridesActiveFeature(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeWorkSection(t, root, "explicit", "state.md", "- Phase: spec\n")
	writeWorkSection(t, root, "explicit", "spec.md", "# Spec\n\nBody\n")
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_WORKSPACE", filepath.Join(root, "work", "explicit"))

	snap, err := Snapshot(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if snap.Slug != "explicit" {
		t.Fatalf("Snapshot slug = %q, want explicit", snap.Slug)
	}
}
