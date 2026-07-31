package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/testutil"
)

func TestCheckAndRenderReadiness(t *testing.T) {
	root := t.TempDir()
	writeFeature(t, root, "alpha", map[string]string{
		"state.md":             "- Phase: build\n",
		"brief.md":             "brief\n",
		"spec.md":              "real spec\n",
		"assumptions.md":       "none\n",
		"questions.md":         "none\n",
		"decision-coverage.md": "clear\n",
		"architecture.md":      "architecture\n",
		"plan.md":              "real plan\n",
		"traceability.md":      "traceability\n",
		"eng-review.md":        "ready\n",
		"test-plan.md":         "tests\n",
	})

	res, err := Check(Readiness, root, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Blocked {
		t.Fatalf("Check blocked=false, want true")
	}
	if res.ReasonID != reason.GateReadinessMissing {
		t.Fatalf("Check reason=%q, want %q", res.ReasonID, reason.GateReadinessMissing)
	}
	got := res.Render()
	for _, want := range []string{
		"gate: readiness",
		"feature: alpha",
		"phase: build",
		"reason: DRV-GATE-READINESS-MISSING",
		`result: blocked (missing to leave "build": decisions.md, tasks.md)`,
		"retry: devrites-engine check readiness alpha",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Render() missing %q in:\n%s", want, got)
		}
	}
}

func TestCheckUsesConcreteWorkspaceRequirements(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"state.md":             "| phase | vet |\n",
		"brief.md":             "brief\n",
		"spec.md":              "spec\n",
		"decisions.md":         "decisions\n",
		"assumptions.md":       "assumptions\n",
		"questions.md":         "none\n",
		"decision-coverage.md": "clear\n",
		"architecture.md":      "architecture\n",
		"plan.md":              "plan\n",
		"tasks.md":             "tasks\n",
		"traceability.md":      "traceability\n",
		"eng-review.md":        "ready\n",
		"test-plan.md":         "", // an empty required file must not satisfy the gate
	}
	writeWorkFeature(t, root, "concrete", files)

	res, err := Check(Readiness, root, "concrete")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Blocked || strings.Join(res.MissingFiles, ",") != "test-plan.md" {
		t.Fatalf("blocked=%v missing=%v, want only test-plan.md", res.Blocked, res.MissingFiles)
	}
}

func TestSealRequiresDurableReviewAndSealArtifacts(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"state.md":             "| phase | seal |\n",
		"brief.md":             "brief\n",
		"spec.md":              "spec\n",
		"decisions.md":         "decisions\n",
		"assumptions.md":       "assumptions\n",
		"questions.md":         "none\n",
		"decision-coverage.md": "clear\n",
		"architecture.md":      "architecture\n",
		"plan.md":              "plan\n",
		"tasks.md":             "tasks\n",
		"traceability.md":      "traceability\n",
		"eng-review.md":        "ready\n",
		"test-plan.md":         "tests\n",
		"evidence.md":          "evidence\n",
		"touched-files.md":     "none\n",
	}
	writeWorkFeature(t, root, "final", files)

	res, err := Check(Seal, root, "final")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Blocked || strings.Join(res.MissingFiles, ",") != "review.md,seal.md" {
		t.Fatalf("blocked=%v missing=%v, want review.md and seal.md", res.Blocked, res.MissingFiles)
	}
}

func TestRenderIncludesReasonForPassedGate(t *testing.T) {
	result := Result{
		Kind:     Seal,
		Slug:     "alpha",
		Phase:    "seal",
		ReasonID: reason.GateSealPassed,
	}
	got := result.Render()
	for _, want := range []string{"result: pass", "reason: DRV-GATE-SEAL-PASSED"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Render() missing %q in:\n%s", want, got)
		}
	}
}

func TestResultReasonIDsAreStableForEveryGateOutcome(t *testing.T) {
	cases := []struct {
		kind    Kind
		blocked bool
		want    reason.ID
	}{
		{Readiness, false, reason.GateReadinessPassed},
		{Readiness, true, reason.GateReadinessMissing},
		{Seal, false, reason.GateSealPassed},
		{Seal, true, reason.GateSealMissing},
	}
	for _, tc := range cases {
		if got := ResultReasonID(tc.kind, tc.blocked); got != tc.want {
			t.Fatalf("ResultReasonID(%q, %v)=%q, want %q", tc.kind, tc.blocked, got, tc.want)
		}
	}
}

func TestStateAwaitingHumanReadsCanonicalCursor(t *testing.T) {
	data := []byte("| Key | Value |\n| --- | --- |\n| status | awaiting_human |\n")
	if !stateAwaitingHuman(data) {
		t.Fatal("stateAwaitingHuman ignored canonical cursor table")
	}
}

func TestCheckBlocksOpenHumanQuestions(t *testing.T) {
	for _, tc := range []struct {
		name             string
		status           string
		wantInvariantSub string
	}{
		{
			name:             "unsurfaced gate",
			status:           "running",
			wantInvariantSub: "state.md is not awaiting_human",
		},
		{
			name:             "surfaced gate",
			status:           "awaiting_human",
			wantInvariantSub: "open blocking/validating human question(s) remain",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFeature(t, root, "alpha", map[string]string{
				"state.md":     "- Phase: build\n- Status: " + tc.status + "\n",
				"spec.md":      "spec\n",
				"plan.md":      "plan\n",
				"decisions.md": "decisions\n",
				"tasks.md":     "tasks\n",
				"questions.md": "## q-1\nstatus: open\ngate: blocking\n\n## q-2\nstatus: open\ngate: validating\n",
			})

			got, err := Check(Readiness, root, "alpha")
			if err != nil {
				t.Fatal(err)
			}
			if !got.Blocked {
				t.Fatal("Check blocked=false, want true")
			}
			invariants := strings.Join(got.StateProblems, "\n")
			if !strings.Contains(invariants, tc.wantInvariantSub) {
				t.Fatalf("StateProblems=%q, want substring %q", invariants, tc.wantInvariantSub)
			}
			if got.ReasonID != reason.GateReadinessMissing {
				t.Fatalf("reason_id=%q, want %q", got.ReasonID, reason.GateReadinessMissing)
			}
		})
	}
}

func TestCheckUsesWorkspaceOverrideForStateInvariants(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	override := filepath.Join(root, "work", "alpha")
	t.Setenv("DEVRITES_WORKSPACE", override)
	testutil.WriteFile(t, filepath.Join(override, "README.md"), "---\nphase: spec\nschemaVersion: 1\n---\n")
	testutil.WriteFile(t, filepath.Join(override, "state.md"), "- Phase: build\n- Status: running\n")
	testutil.WriteFile(t, filepath.Join(override, "spec.md"), "spec\n")
	testutil.WriteFile(t, filepath.Join(override, "questions.md"), "## q-1\nstatus: open\ngate: blocking\n")

	got, err := Check(Readiness, root, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Blocked || len(got.StateProblems) == 0 {
		t.Fatalf("Check did not inspect override workspace: blocked=%v problems=%v", got.Blocked, got.StateProblems)
	}
}

func TestOpenBlockingQuestionGates(t *testing.T) {
	got := openBlockingQuestionGates([]byte("## Q-1\nstatus: open\ngate: blocking\n\n## Not a question\n\n## q-2\nstatus: open\ngate: validating\n\n## q-3\nstatus: resolved\ngate: blocking\n\n## q-4\nstatus: open\ngate: blocking\n\n## Q-5\nstatus: open\ngate: escalating\n"))
	want := []string{"blocking", "validating", "escalating"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("openBlockingQuestionGates=%v, want %v", got, want)
	}
}

func writeFeature(t *testing.T, root, slug string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		testutil.WriteFile(t, filepath.Join(root, "work", slug, name), content)
	}
}

func writeWorkFeature(t *testing.T, root, slug string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		testutil.WriteFile(t, filepath.Join(root, "work", slug, name), content)
	}
}
