package state

// Focused unit tests on the deterministic core: phase→required mapping,
// present/empty detection, and status computation. These assert behavior
// (observable results), not internal structure.

import (
	"os"
	"path/filepath"
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
