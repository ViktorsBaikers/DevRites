package migrate

import (
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
	assertFile(t, root, "work/alpha/proof.md", "proof\n")
	assertFile(t, root, "work/alpha/status.md", "status: done - shipped\n")
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
	testutil.WriteFile(t, filepath.Join(root, "features/beta/state.md"), "- Phase: prove\n")
	testutil.WriteFile(t, filepath.Join(root, "features/beta/evidence.md"), "evidence\n")

	res, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(res.Migrated, ","); got != "beta" {
		t.Fatalf("Run normalized=%q, want beta", got)
	}
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
		{word: "in-progress", want: state.PhaseBuild, ok: true},
		{word: "reviewing", want: state.PhaseReview, ok: true},
		{word: "done - shipped", want: state.PhaseDone, ok: true},
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
