package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/testutil"
)

func TestRunNormalizesCanonicalWorkLayout(t *testing.T) {
	root := t.TempDir()
	testutil.WriteFile(t, filepath.Join(root, "ACTIVE"), "alpha\n")
	testutil.WriteFile(t, filepath.Join(root, "work/alpha/spec.md"), "spec\n")
	testutil.WriteFile(t, filepath.Join(root, "work/alpha/plan.md"), "plan\n")
	testutil.WriteFile(t, filepath.Join(root, "work/alpha/evidence.md"), "proof\n")
	testutil.WriteFile(t, filepath.Join(root, "work/alpha/state.md"), "status: done - shipped\n")
	testutil.WriteFile(t, filepath.Join(root, "work/alpha/review.md"), "review\n")

	res, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped {
		t.Fatalf("Run skipped=true, want migration")
	}
	if got := strings.Join(res.Migrated, ","); got != "alpha" {
		t.Fatalf("Run migrated=%q, want alpha", got)
	}
	if res.BackupDir == "" {
		t.Fatalf("Run backup dir empty")
	}

	assertFile(t, root, "work/alpha/spec.md", "spec\n")
	assertFile(t, root, "work/alpha/evidence.md", "proof\n")
	assertFile(t, root, "work/alpha/state.md", "status: done - shipped\n")
	assertFileContains(t, root, "work/alpha/README.md", "phase: done")
	assertFileContains(t, root, "work/alpha/README.md", "schemaVersion: 2")
	assertMissing(t, root, "work/alpha/proof.md")
	assertMissing(t, root, "work/alpha/status.md")
	assertFile(t, root, "work/alpha/review.md", "review\n")

	f, err := state.LoadFeature(root, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if f.Phase != state.PhaseDone {
		t.Fatalf("migrated phase=%q, want %q", f.Phase, state.PhaseDone)
	}
	assertFile(t, res.BackupDir, "ACTIVE", "alpha\n")

	again, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if !again.Skipped {
		t.Fatalf("second Run skipped=false, migrated=%v", again.Migrated)
	}
}

func TestRunNormalizesLiveFeatureAliases(t *testing.T) {
	root := t.TempDir()
	testutil.WriteFile(t, filepath.Join(root, "features/beta/feature.md"), "# Beta\n")
	testutil.WriteFile(t, filepath.Join(root, "features/beta/status.md"), "- Phase: prove\n")
	testutil.WriteFile(t, filepath.Join(root, "features/beta/proof.md"), "evidence\n")

	res, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(res.Migrated, ","); got != "beta" {
		t.Fatalf("Run normalized=%q, want beta", got)
	}
	assertFileContains(t, root, "features/beta/README.md", "schemaVersion: 2")
	assertFileContains(t, root, "features/beta/README.md", "# Beta")
	assertFile(t, root, "features/beta/state.md", "- Phase: prove\n")
	assertFile(t, root, "features/beta/evidence.md", "evidence\n")
	assertFile(t, root, "features/beta/status.md", "- Phase: prove\n")
	assertFile(t, root, "features/beta/proof.md", "evidence\n")

	f, err := state.LoadFeature(root, "beta")
	if err != nil {
		t.Fatal(err)
	}
	if f.Phase != state.PhaseProve {
		t.Fatalf("normalized phase=%q, want %q", f.Phase, state.PhaseProve)
	}
}

func TestRunUpgradesV1DeclarationWithoutFabricatingReadiness(t *testing.T) {
	root := t.TempDir()
	testutil.WriteFile(t, filepath.Join(root, "work/old/README.md"), "---\nphase: vet\nschemaVersion: 1\n---\n\nOld workspace.\n")
	testutil.WriteFile(t, filepath.Join(root, "work/old/state.md"), "| phase | vet |\n")

	result, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Skipped || strings.Join(result.Migrated, ",") != "old" {
		t.Fatalf("result=%+v, want old migrated", result)
	}
	assertFileContains(t, root, "work/old/README.md", "schemaVersion: 2")
	for _, name := range []string{"decision-coverage.md", "eng-review.md", "test-plan.md"} {
		assertMissing(t, root, filepath.Join("work/old", name))
	}
}

func TestMapLegacyPhase(t *testing.T) {
	for _, phase := range state.LifecyclePhases() {
		got, ok := mapLegacyPhase(string(phase))
		if !ok || got != phase {
			t.Fatalf("mapLegacyPhase(%q)=(%q,%v), want canonical phase", phase, got, ok)
		}
	}

	for _, tc := range []struct {
		word string
		want state.Phase
		ok   bool
	}{
		{word: "specced", want: state.PhaseSpec, ok: true},
		{word: "clarified", want: state.PhaseClarify, ok: true},
		{word: "in-progress", want: state.PhaseBuild, ok: true},
		{word: "reviewing", want: state.PhaseReview, ok: true},
		{word: "done - shipped", want: state.PhaseDone, ok: true},
		{word: "done — shipped", want: state.PhaseDone, ok: true},
		{word: "mystery"},
	} {
		got, ok := mapLegacyPhase(tc.word)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("mapLegacyPhase(%q)=(%q,%v), want (%q,%v)", tc.word, got, ok, tc.want, tc.ok)
		}
	}
}

func assertFile(t *testing.T, root, rel, want string) {
	t.Helper()
	if got := testutil.ReadFile(t, filepath.Join(root, rel)); got != want {
		t.Fatalf("%s=%q, want %q", rel, got, want)
	}
}

func assertFileContains(t *testing.T, root, rel, want string) {
	t.Helper()
	if got := testutil.ReadFile(t, filepath.Join(root, rel)); !strings.Contains(got, want) {
		t.Fatalf("%s=%q, want it to contain %q", rel, got, want)
	}
}

func assertMissing(t *testing.T, root, rel string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
		t.Fatalf("%s exists or stat failed with %v, want missing", rel, err)
	}
}
