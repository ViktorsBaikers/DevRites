package gate

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/testutil"
)

func TestTestDriverREMatchesOnlyWholeWordDriversUnderScripts(t *testing.T) {
	for path, want := range map[string]bool{
		"scripts/dev-run-test.sh":     true,
		"scripts/run-tests.mjs":       true,
		"scripts/test.sh":             true,
		"tools/scripts/smoke_test.py": true,
		"scripts/e2e/test.unit.bash":  true,
		"scripts/latest-release.sh":   false,
		"scripts/attest.sh":           false,
		"scripts/contest.sh":          false,
		"scripts/testdata.sh":         false,
		"scripts/dev-run-test.go":     false,
		"src/export.test.js":          false,
		"tests/phase-gate-test.sh":    false,
		"myscripts/run-test.sh":       false,
	} {
		if got := testDriverRE.MatchString(path); got != want {
			t.Errorf("testDriverRE.MatchString(%q)=%v, want %v", path, got, want)
		}
	}
}

func TestTestDriverProblems(t *testing.T) {
	const allowlist = testutil.CanonicalTasksMarkdown + "Writer allowlist: src/app.go, `scripts/dev-run-test.sh`, scripts/latest-release.sh\n"
	const touched = "# Touched files\n\n## Touched files\n\nx\n\n## Candidate manifest\n\n| State | File | Slice | Reason |\n| --- | --- | --- | --- |\n"
	const decisions = "# Decisions\n\n## DEC-030 — Defer the driver\n\nOwner verbatim.\n"
	runGate := testutil.CanonicalGatesMarkdown + "- [ ] G2: driver green\n  CHECK: bash scripts/dev-run-test.sh --all\n  EXPECT: DEV-RUN-TEST-PASS\n"
	cwdGate := testutil.CanonicalGatesMarkdown + "- [ ] G2: driver green\n  CHECK: bash ./dev-run-test.sh\n  EXPECT: PASS\n  CWD: scripts\n"
	waiver := func(id string) string {
		return testutil.CanonicalGatesMarkdown + "\nDriver waiver: `scripts/dev-run-test.sh` — " + id + " owner deferred the mutation machinery\n"
	}
	for _, tc := range []struct {
		name         string
		phase        state.Phase
		files        map[string]string
		wantProblem  string
		wantWaived   string
		wantNoIssues bool
	}{
		{name: "allowlisted driver with no CHECK blocks at readiness", phase: state.PhaseVet,
			files:       map[string]string{"tasks.md": allowlist},
			wantProblem: "test driver scripts/dev-run-test.sh is in tasks.md but no gates.md CHECK executes it"},
		{name: "CHECK naming the driver passes", phase: state.PhaseVet,
			files:        map[string]string{"tasks.md": allowlist, "gates.md": runGate},
			wantNoIssues: true},
		{name: "CHECK naming the driver relative to CWD passes", phase: state.PhaseVet,
			files:        map[string]string{"tasks.md": allowlist, "gates.md": cwdGate},
			wantNoIssues: true},
		{name: "waiver citing a recorded DEC passes and is reported", phase: state.PhaseVet,
			files:      map[string]string{"tasks.md": allowlist, "gates.md": waiver("DEC-030"), "decisions.md": decisions},
			wantWaived: "unexecuted driver (waived by DEC-030): scripts/dev-run-test.sh"},
		{name: "CHECK running a different scripts/ tree does not count", phase: state.PhaseVet,
			files:       map[string]string{"tasks.md": allowlist, "gates.md": testutil.CanonicalGatesMarkdown + "- [ ] G2: other driver\n  CHECK: bash tools/scripts/dev-run-test.sh\n  EXPECT: PASS\n"},
			wantProblem: "test driver scripts/dev-run-test.sh is in tasks.md"},
		{name: "waiver with a colon after the DEC id passes", phase: state.PhaseVet,
			files:      map[string]string{"tasks.md": allowlist, "gates.md": waiver("DEC-030:"), "decisions.md": decisions},
			wantWaived: "unexecuted driver (waived by DEC-030): scripts/dev-run-test.sh"},
		{name: "waiver citing a missing DEC blocks", phase: state.PhaseVet,
			files:       map[string]string{"tasks.md": allowlist, "gates.md": waiver("DEC-031"), "decisions.md": decisions},
			wantProblem: "waiver cites DEC-031, which has no decisions.md heading"},
		{name: "fenced waiver example does not count", phase: state.PhaseVet,
			files:       map[string]string{"tasks.md": allowlist, "gates.md": testutil.CanonicalGatesMarkdown + "\n```text\nDriver waiver: scripts/dev-run-test.sh — DEC-030 example\n```\n", "decisions.md": decisions},
			wantProblem: "test driver scripts/dev-run-test.sh is in tasks.md"},
		{name: "ordinary scripts are not drivers", phase: state.PhaseVet,
			files:        map[string]string{"tasks.md": testutil.CanonicalTasksMarkdown + "Writer allowlist: scripts/latest-release.sh, scripts/attest.sh\n"},
			wantNoIssues: true},
		{name: "touched present driver blocks once proof is required", phase: state.PhaseSeal,
			files:       map[string]string{"touched-files.md": touched + "| present | `scripts/dev-run-test.sh` | SLICE-001 | repair |\n"},
			wantProblem: "test driver scripts/dev-run-test.sh is in touched-files.md"},
		{name: "touched deleted driver is not checked", phase: state.PhaseSeal,
			files:        map[string]string{"tasks.md": allowlist, "touched-files.md": touched + "| deleted | `scripts/dev-run-test.sh` | SLICE-001 | removed |\n"},
			wantNoIssues: true},
		{name: "phases without gates.md never check drivers", phase: state.PhasePlan,
			files:        map[string]string{"tasks.md": allowlist},
			wantNoIssues: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeCompleteGateFeature(t, root, "drivers", tc.phase, tc.phase, "none\n")
			writeWorkFeature(t, root, "drivers", tc.files)
			observation, err := state.ObserveWorkspace(root, "drivers")
			if err != nil {
				t.Fatal(err)
			}
			policy, _ := state.PolicyFor(tc.phase)
			problems, waived := testDriverProblems(observation, policy)
			joined := strings.Join(problems, "\n")
			switch {
			case tc.wantNoIssues && (len(problems) > 0 || len(waived) > 0):
				t.Fatalf("problems=%q waived=%q, want none", problems, waived)
			case tc.wantProblem != "" && (len(problems) != 1 || !strings.Contains(joined, tc.wantProblem)):
				t.Fatalf("problems=%q, want one containing %q", problems, tc.wantProblem)
			case tc.wantWaived != "" && (len(problems) > 0 || len(waived) != 1 || waived[0] != tc.wantWaived):
				t.Fatalf("problems=%q waived=%q, want waived %q", problems, waived, tc.wantWaived)
			}
		})
	}
}

// Check wires the driver rule into readiness: a block names the driver, and a
// waived driver is printed as a note so seal reviewers see the deferral.
func TestCheckReportsTestDriverBlocksAndWaivers(t *testing.T) {
	root := t.TempDir()
	for slug, gates := range map[string]string{
		"unexecuted": testutil.CanonicalGatesMarkdown,
		"waived":     testutil.CanonicalGatesMarkdown + "\nDriver waiver: scripts/dev-run-test.sh — DEC-030 owner deferred it\n",
	} {
		writeCompleteGateFeature(t, root, slug, state.PhaseBuild, state.PhaseBuild, "none\n")
		workspace := filepath.Join(root, "work", slug)
		testutil.WriteFile(t, filepath.Join(workspace, "tasks.md"), testutil.CanonicalTasksMarkdown+"Writer allowlist: scripts/dev-run-test.sh\n")
		testutil.WriteFile(t, filepath.Join(workspace, "gates.md"), gates)
		testutil.WriteFile(t, filepath.Join(workspace, "decisions.md"), "# Decisions\n\n## DEC-030 — Defer the driver\n\nOwner verbatim.\n")
		testutil.AppendFile(t, filepath.Join(workspace, "eng-review.md"), "\n"+mustReadinessBinding(t, root, slug)+"\n")
	}

	res, err := Check(Readiness, root, "unexecuted")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Blocked || !strings.Contains(res.Render(), "invariant: gates: test driver scripts/dev-run-test.sh is in tasks.md") {
		t.Fatalf("blocked=%v render=\n%s", res.Blocked, res.Render())
	}

	res, err = Check(Readiness, root, "waived")
	if err != nil {
		t.Fatal(err)
	}
	rendered := res.Render()
	if strings.Contains(rendered, "test driver") || !strings.Contains(rendered, "note: unexecuted driver (waived by DEC-030): scripts/dev-run-test.sh\n") {
		t.Fatalf("render=\n%s", rendered)
	}
}

func TestRenderPrintsNotesOnPass(t *testing.T) {
	res := &Result{Kind: Seal, Slug: "s", Phase: state.PhaseSeal, ReasonID: ResultReasonID(Seal, false),
		Notes: []string{"unexecuted driver (waived by DEC-030): scripts/dev-run-test.sh"}}
	want := "gate: seal\nfeature: s\nphase: seal\nresult: pass\nreason: DRV-GATE-SEAL-PASSED\n" +
		"note: unexecuted driver (waived by DEC-030): scripts/dev-run-test.sh\n"
	if got := res.Render(); got != want {
		t.Fatalf("Render()=\n%q\nwant\n%q", got, want)
	}
}
