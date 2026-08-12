package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/state"
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

func TestCheckAppliesOpenQuestionsByTargetPolicy(t *testing.T) {
	const openQuestion = "## q-1\nstatus: open\ngate: blocking\n"
	cases := []struct {
		name          string
		kind          Kind
		current       state.Phase
		required      state.Phase
		wantTarget    state.Phase
		wantBlocked   bool
		wantReason    reason.ID
		wantInvariant string
	}{
		{name: "frame readiness", kind: Readiness, current: state.PhaseFrame, required: state.PhaseFrame, wantTarget: state.PhaseFrame, wantReason: reason.GateReadinessPassed},
		{name: "spec readiness", kind: Readiness, current: state.PhaseSpec, required: state.PhaseSpec, wantTarget: state.PhaseSpec, wantReason: reason.GateReadinessPassed},
		{name: "clarify readiness", kind: Readiness, current: state.PhaseClarify, required: state.PhaseClarify, wantTarget: state.PhaseClarify, wantBlocked: true, wantReason: reason.GateReadinessMissing, wantInvariant: "open blocking human question(s) remain in questions.md but state.md is not awaiting_human"},
		{name: "seal from spec", kind: Seal, current: state.PhaseSpec, required: state.PhaseSeal, wantTarget: state.PhaseSeal, wantBlocked: true, wantReason: reason.GateSealMissing, wantInvariant: "open blocking human question(s) remain in questions.md but state.md is not awaiting_human"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeCompleteGateFeature(t, root, "phase-policy", tc.current, tc.required, openQuestion)
			if tc.kind == Seal {
				binding, err := ReadinessBinding(root, "phase-policy")
				if err != nil {
					t.Fatal(err)
				}
				testutil.AppendFile(t, filepath.Join(root, "work", "phase-policy", "eng-review.md"), "\n"+binding+"\n")
			}

			got, err := Check(tc.kind, root, "phase-policy")
			if err != nil {
				t.Fatal(err)
			}
			if got.Target != tc.wantTarget || got.Blocked != tc.wantBlocked || got.ReasonID != tc.wantReason {
				t.Fatalf("target=%q blocked=%v reason=%q, want target=%q blocked=%v reason=%q", got.Target, got.Blocked, got.ReasonID, tc.wantTarget, tc.wantBlocked, tc.wantReason)
			}
			if tc.wantInvariant == "" {
				if len(got.StateProblems) != 0 {
					t.Fatalf("StateProblems=%v, want none", got.StateProblems)
				}
			} else if strings.Join(got.StateProblems, "\n") != tc.wantInvariant {
				t.Fatalf("StateProblems=%q, want %q", got.StateProblems, tc.wantInvariant)
			}
		})
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

func writeCompleteGateFeature(t *testing.T, root, slug string, current, required state.Phase, questions string) {
	t.Helper()
	policy, ok := state.PolicyFor(required)
	if !ok {
		t.Fatalf("PolicyFor(%q) returned unknown", required)
	}
	questionsRequired := false
	for _, artifact := range policy.RequiredArtifacts {
		name := string(artifact)
		content := "# " + name + "\n\nreal\n"
		switch name {
		case "state.md":
			content = "- Phase: " + string(current) + "\n- Status: running\n"
		case "questions.md":
			questionsRequired = true
			content = questions
		}
		testutil.WriteFile(t, filepath.Join(root, "work", slug, name), content)
	}
	if !questionsRequired {
		testutil.WriteFile(t, filepath.Join(root, "work", slug, "questions.md"), questions)
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
