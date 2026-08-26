package gate

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/testutil"
)

func TestCheckBlocksCyclicTaskGraphWhenTasksAreRequired(t *testing.T) {
	root := t.TempDir()
	workspace := writeReadinessFixture(t, root, "cyclic", "build")
	testutil.WriteFile(t, filepath.Join(workspace, "tasks.md"), `# Tasks

## SLICE-001 A
Dependencies: SLICE-002

## SLICE-002 B
Dependencies: SLICE-001
`)
	binding := mustReadinessBinding(t, root, "cyclic")
	testutil.AppendFile(t, filepath.Join(workspace, "eng-review.md"), "\n"+binding+"\n")

	res, err := Check(Readiness, root, "cyclic")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Blocked || res.ReasonID != reason.GateReadinessMissing {
		t.Fatalf("blocked=%v reason=%q, want blocked missing", res.Blocked, res.ReasonID)
	}
	joined := strings.Join(res.StateProblems, "\n")
	if !strings.Contains(joined, "task-graph: dependency cycle:") {
		t.Fatalf("StateProblems=%q", joined)
	}
	if !strings.Contains(res.Render(), "result: blocked (state invariant)") {
		t.Fatalf("Render()=\n%s", res.Render())
	}
}

func TestCheckBlocksCyclicTaskGraphAtSeal(t *testing.T) {
	root := t.TempDir()
	writeCompleteGateFeature(t, root, "cyclic-seal", state.PhaseSeal, state.PhaseSeal, "none\n")
	workspace := filepath.Join(root, "work", "cyclic-seal")
	testutil.WriteFile(t, filepath.Join(workspace, "tasks.md"), `# Tasks

## SLICE-001 A
Dependencies: SLICE-002

## SLICE-002 B
Dependencies: SLICE-001
`)
	binding := mustReadinessBinding(t, root, "cyclic-seal")
	testutil.AppendFile(t, filepath.Join(workspace, "eng-review.md"), "\n"+binding+"\n")

	res, err := Check(Seal, root, "cyclic-seal")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Blocked || res.ReasonID != reason.GateSealMissing {
		t.Fatalf("blocked=%v reason=%q, want blocked %s", res.Blocked, res.ReasonID, reason.GateSealMissing)
	}
	joined := strings.Join(res.StateProblems, "\n")
	if !strings.Contains(joined, "task-graph: dependency cycle:") {
		t.Fatalf("StateProblems=%q", joined)
	}
	if !strings.Contains(res.Render(), "result: blocked (state invariant)") {
		t.Fatalf("Render()=\n%s", res.Render())
	}
}

func TestCheckBlocksMalformedTaskGraphInsteadOfDroppingTokens(t *testing.T) {
	root := t.TempDir()
	workspace := writeReadinessFixture(t, root, "malformed", "define")
	testutil.WriteFile(t, filepath.Join(workspace, "tasks.md"), `# Tasks

## SLICE-001 Ready
Dependencies: none

## SLICE-002 Next
Dependencies: SLICE-001 and later
`)

	res, err := Check(Readiness, root, "malformed")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Blocked {
		t.Fatal("expected malformed dependency to block readiness")
	}
	joined := strings.Join(res.StateProblems, "\n")
	if !strings.Contains(joined, `malformed dependency "and"`) || !strings.Contains(joined, `malformed dependency "later"`) {
		t.Fatalf("StateProblems=%q", joined)
	}
}

func TestCheckBlocksMissingDependenciesInsteadOfTreatingSliceAsIndependent(t *testing.T) {
	root := t.TempDir()
	workspace := writeReadinessFixture(t, root, "nodeps", "define")
	testutil.WriteFile(t, filepath.Join(workspace, "tasks.md"), `# Tasks

## SLICE-001 Ready
Goal: looks complete without an ordering field
`)

	res, err := Check(Readiness, root, "nodeps")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Blocked {
		t.Fatal("expected missing Dependencies to block readiness")
	}
	joined := strings.Join(res.StateProblems, "\n")
	if !strings.Contains(joined, "SLICE-001 is missing Dependencies") {
		t.Fatalf("StateProblems=%q", joined)
	}
}

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

func TestCheckReturnsMissingSectionsInSelectedPolicyOrder(t *testing.T) {
	tests := []struct {
		name           string
		kind           Kind
		current        state.Phase
		required       state.Phase
		emptyArtifacts []string
		wantMissing    []state.Section
	}{
		{
			name:           "readiness",
			kind:           Readiness,
			current:        state.PhaseBuild,
			required:       state.PhaseBuild,
			emptyArtifacts: []string{"tasks.md", "spec.md"},
			wantMissing:    []state.Section{state.SectionSpec, state.SectionTasks},
		},
		{
			name:           "seal",
			kind:           Seal,
			current:        state.PhaseSpec,
			required:       state.PhaseSeal,
			emptyArtifacts: []string{state.EvidenceFile, "tasks.md"},
			wantMissing:    []state.Section{state.SectionTasks, state.SectionProof},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeCompleteGateFeature(t, root, test.name, test.current, test.required, "none\n")
			for _, name := range test.emptyArtifacts {
				testutil.WriteFile(t, filepath.Join(root, "work", test.name, name), "# Empty\n")
			}

			result, err := Check(test.kind, root, test.name)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(result.Missing, test.wantMissing) {
				t.Fatalf("Missing=%v, want %v", result.Missing, test.wantMissing)
			}
		})
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
	physicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	override := filepath.Join(physicalRoot, "work", "alpha")
	t.Setenv("DEVRITES_WORKSPACE", override)
	testutil.WriteFile(t, filepath.Join(override, "README.md"), "---\nphase: spec\nschemaVersion: 1\n---\n")
	testutil.WriteFile(t, filepath.Join(override, "state.md"), "- Phase: build\n- Status: running\n")
	testutil.WriteFile(t, filepath.Join(override, "spec.md"), "spec\n")
	testutil.WriteFile(t, filepath.Join(override, "questions.md"), "## q-1\nstatus: open\ngate: blocking\n")

	got, err := Check(Readiness, physicalRoot, "alpha")
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

func TestCheckObservationUsesRetainedPhaseQuestionsReadinessAndReview(t *testing.T) {
	root := t.TempDir()
	workspace := writeReadinessFixture(t, root, "retained", "build")
	binding := mustReadinessBinding(t, root, "retained")
	testutil.AppendFile(t, filepath.Join(workspace, "eng-review.md"), "\n"+binding+"\n")

	observation, err := state.ObserveWorkspace(root, "retained")
	if err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, filepath.Join(workspace, "state.md"), "- Phase: spec\n- Status: running\n")
	testutil.WriteFile(t, filepath.Join(workspace, "questions.md"), "## q-1\nstatus: open\ngate: blocking\n")
	testutil.WriteFile(t, filepath.Join(workspace, "plan.md"), "# Plan\n\nChanged after observation.\n")
	testutil.WriteFile(t, filepath.Join(workspace, "eng-review.md"), "# Engineering review\n\nNo binding.\n")

	result, err := checkObservation(Readiness, observation)
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != state.PhaseBuild || result.Blocked || len(result.StateProblems) != 0 {
		t.Fatalf("retained result phase=%q blocked=%v problems=%v", result.Phase, result.Blocked, result.StateProblems)
	}
}

func TestGateDiagnosticRecoveryLines(t *testing.T) {
	for _, tc := range []struct {
		diagnostic state.ArtifactDiagnostic
		required   string
		optional   string
	}{
		{
			state.ArtifactDiagnostic{Path: "spec.md", State: state.ArtifactMalformed, Code: state.DiagnosticMalformedMarkdown},
			"next: repair spec.md: replace invalid Markdown with valid Markdown; required artifacts need substantive content",
			"next: repair spec.md: replace invalid Markdown with valid Markdown; optional readiness input may instead be removed",
		},
		{
			state.ArtifactDiagnostic{Path: "spec.md", State: state.ArtifactUnsafe, Code: state.DiagnosticParentSymlink},
			"next: repair spec.md: replace the symlinked parent with a real directory",
			"next: repair spec.md: replace the symlinked parent with a real directory; optional readiness input may instead be removed",
		},
		{
			state.ArtifactDiagnostic{Path: "spec.md", State: state.ArtifactUnsafe, Code: state.DiagnosticFinalSymlink},
			"next: repair spec.md: replace the symlink with a regular file",
			"next: repair spec.md: replace the symlink with a regular file; optional readiness input may instead be removed",
		},
		{
			state.ArtifactDiagnostic{Path: "spec.md", State: state.ArtifactUnsafe, Code: state.DiagnosticNonRegular},
			"next: repair spec.md: replace the non-regular entry with a regular file",
			"next: repair spec.md: replace the non-regular entry with a regular file; optional readiness input may instead be removed",
		},
		{
			state.ArtifactDiagnostic{Path: "spec.md", State: state.ArtifactUnsafe, Code: state.DiagnosticFileTooLarge},
			"next: repair spec.md: reduce the file to at most 1 MiB",
			"next: repair spec.md: reduce the file to at most 1 MiB; optional readiness input may instead be removed",
		},
		{
			state.ArtifactDiagnostic{Path: "spec.md", State: state.ArtifactUnreadable, Code: state.DiagnosticPermissionDenied},
			"next: repair spec.md: grant read permission",
			"next: repair spec.md: grant read permission; optional readiness input may instead be removed",
		},
		{
			state.ArtifactDiagnostic{Path: "spec.md", State: state.ArtifactUnreadable, Code: state.DiagnosticReadFailure},
			"next: repair spec.md: restore a readable regular file",
			"next: repair spec.md: restore a readable regular file; optional readiness input may instead be removed",
		},
	} {
		if got := diagnosticRecovery(tc.diagnostic, true); got != tc.required {
			t.Errorf("required diagnosticRecovery(%+v)=%q, want %q", tc.diagnostic, got, tc.required)
		}
		if got := diagnosticRecovery(tc.diagnostic, false); got != tc.optional {
			t.Errorf("optional diagnosticRecovery(%+v)=%q, want %q", tc.diagnostic, got, tc.optional)
		}
	}

	unknown := state.ArtifactDiagnostic{Path: "strategy.md", State: state.ArtifactUnsafe, Code: "unknown"}
	if got := diagnosticRecovery(unknown, true); got != "" {
		t.Fatalf("required unknown diagnosticRecovery()=%q, want empty", got)
	}
	if got := diagnosticRecovery(unknown, false); got != "" {
		t.Fatalf("optional unknown diagnosticRecovery()=%q, want empty", got)
	}
}

func TestCheckRendersRequiredDiagnosticBeforeRecoveryWithoutGenericRecovery(t *testing.T) {
	root := t.TempDir()
	writeCompleteGateFeature(t, root, "diagnostic", state.PhaseSpec, state.PhaseSpec, "none\n")
	testutil.WriteFile(t, filepath.Join(root, "work", "diagnostic", "spec.md"), "hostile-secret\x00")

	result, err := Check(Readiness, root, "diagnostic")
	if err != nil {
		t.Fatal(err)
	}
	want := "gate: readiness\n" +
		"feature: diagnostic\n" +
		"phase: spec\n" +
		"result: blocked (missing to leave \"spec\": spec.md)\n" +
		"reason: DRV-GATE-READINESS-MISSING\n" +
		"artifact: spec.md: malformed (malformed_markdown)\n" +
		"next: repair spec.md: replace invalid Markdown with valid Markdown; required artifacts need substantive content\n" +
		"retry: devrites-engine check readiness diagnostic\n"
	if got := result.Render(); got != want {
		t.Fatalf("Render()=\n%q\nwant\n%q", got, want)
	}
	if strings.Contains(result.Render(), "next: add real content") || strings.Contains(result.Render(), "hostile-secret") {
		t.Fatalf("diagnostic output used generic recovery or disclosed content:\n%s", result.Render())
	}
}

func TestCheckOmitsUnselectedEvidenceDiagnostic(t *testing.T) {
	root := t.TempDir()
	const slug = "unselected-evidence"
	writeCompleteGateFeature(t, root, slug, state.PhaseSpec, state.PhaseSpec, "none\n")
	workspace := filepath.Join(root, "work", slug)
	if err := os.Symlink("brief.md", filepath.Join(workspace, state.EvidenceFile)); err != nil {
		t.Fatalf("create unselected evidence symlink: %v", err)
	}

	result, err := Check(Readiness, root, slug)
	if err != nil {
		t.Fatal(err)
	}
	want := "gate: readiness\n" +
		"feature: unselected-evidence\n" +
		"phase: spec\n" +
		"result: pass\n" +
		"reason: DRV-GATE-READINESS-PASSED\n"
	if got := result.Render(); got != want {
		t.Fatalf("Render()=\n%q\nwant\n%q", got, want)
	}
	if strings.Contains(result.Render(), "artifact:") || strings.Contains(result.Render(), "next:") {
		t.Fatalf("unselected evidence emitted diagnostic or recovery:\n%s", result.Render())
	}
}

func TestCheckSelectsOptionalReadinessDiagnosticInBindingOrder(t *testing.T) {
	root := t.TempDir()
	workspace := writeReadinessFixture(t, root, "optional-diagnostic", "build")
	testutil.WriteFile(t, filepath.Join(workspace, "strategy.md"), "bad\x00strategy")

	result, err := Check(Readiness, root, "optional-diagnostic")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Blocked || result.ReasonID != reason.GateReadinessStale || len(result.Diagnostics) != 1 {
		t.Fatalf("blocked=%v reason=%q diagnostics=%+v", result.Blocked, result.ReasonID, result.Diagnostics)
	}
	rendered := result.Render()
	ordered := []string{
		"reason: DRV-GATE-READINESS-STALE\n",
		"artifact: strategy.md: malformed (malformed_markdown)\n",
		"next: repair strategy.md: replace invalid Markdown with valid Markdown; optional readiness input may instead be removed\n",
		"invariant: readiness input strategy.md is malformed (malformed_markdown); replace invalid Markdown with valid Markdown; repair the input and rerun /rite-vet\n",
		"retry: devrites-engine check readiness optional-diagnostic\n",
	}
	position := -1
	for _, line := range ordered {
		next := strings.Index(rendered, line)
		if next <= position {
			t.Fatalf("Render() missing ordered line %q:\n%s", line, rendered)
		}
		position = next
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
		case "tasks.md":
			content = testutil.CanonicalTasksMarkdown
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
