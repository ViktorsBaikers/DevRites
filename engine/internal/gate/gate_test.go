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
		"feature.md": "---\nphase: build\nschemaVersion: 1\n---\n",
		"spec.md":    "real spec\n",
		"plan.md":    "real plan\n",
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
		`result: blocked (missing to leave "build": decisions, tasks)`,
		"then re-run: devrites-engine readiness alpha",
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

func TestStopGateRestPointInvariants(t *testing.T) {
	for _, tc := range []struct {
		name          string
		files         map[string]string
		wantBlocked   bool
		wantReasonSub string
		wantReasonID  reason.ID
	}{
		{
			name: "red blocks",
			files: map[string]string{
				"feature.md": "---\nphase: build\nschemaVersion: 1\n---\n",
				"state.md":   "- Phase: build\n",
				".red":       "go test ./...\n",
			},
			wantBlocked:   true,
			wantReasonSub: "tests/build RED",
			wantReasonID:  reason.HookStopRed,
		},
		{
			name: "unsurfaced human gate blocks",
			files: map[string]string{
				"feature.md":   "---\nphase: build\nschemaVersion: 1\n---\n",
				"state.md":     "- Phase: build\n- Status: running\n",
				"questions.md": "## q-1\nstatus: open\ngate: blocking\n\n## q-2\nstatus: open\ngate: validating\n",
			},
			wantBlocked:   true,
			wantReasonSub: "open blocking/validating human question",
			wantReasonID:  reason.HookStopUnsurfacedHumanGate,
		},
		{
			name: "awaiting human allows surfaced gate",
			files: map[string]string{
				"feature.md":   "---\nphase: build\nschemaVersion: 1\n---\n",
				"state.md":     "- Phase: build\n- Status: awaiting_human\n",
				"questions.md": "## q-1\nstatus: open\ngate: blocking\n",
			},
		},
		{
			name: "seal without proof blocks",
			files: map[string]string{
				"feature.md": "---\nphase: seal\nschemaVersion: 1\n---\n",
				"state.md":   "- Phase: seal\n",
			},
			wantBlocked:   true,
			wantReasonSub: "proof.md is empty",
			wantReasonID:  reason.HookStopMissingProof,
		},
		{
			name: "done without proof blocks",
			files: map[string]string{
				"feature.md": "---\nphase: ship\nschemaVersion: 1\n---\n",
				"state.md":   "| Key | Value |\n| --- | --- |\n| phase | done |\n",
			},
			wantBlocked:   true,
			wantReasonSub: "proof.md is empty",
			wantReasonID:  reason.HookStopMissingProof,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			testutil.WriteFile(t, filepath.Join(root, "ACTIVE"), "alpha\n")
			writeFeature(t, root, "alpha", tc.files)

			got, err := StopGate(root)
			if err != nil {
				t.Fatal(err)
			}
			if got.Slug != "alpha" {
				t.Fatalf("StopGate slug=%q, want alpha", got.Slug)
			}
			if got.Blocked != tc.wantBlocked {
				t.Fatalf("StopGate blocked=%v, want %v; reason=%q", got.Blocked, tc.wantBlocked, got.Reason)
			}
			if tc.wantReasonSub != "" && !strings.Contains(got.Reason, tc.wantReasonSub) {
				t.Fatalf("StopGate reason=%q, want substring %q", got.Reason, tc.wantReasonSub)
			}
			wantReasonID := tc.wantReasonID
			if wantReasonID == "" {
				wantReasonID = reason.HookStopClear
			}
			if got.ReasonID != wantReasonID {
				t.Fatalf("StopGate reason_id=%q, want %q", got.ReasonID, wantReasonID)
			}
		})
	}
}

func TestStopGateUsesWorkspaceOverride(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	override := filepath.Join(root, "work", "alpha")
	t.Setenv("DEVRITES_WORKSPACE", override)
	testutil.WriteFile(t, filepath.Join(override, "feature.md"), "---\nphase: build\nschemaVersion: 1\n---\n")
	testutil.WriteFile(t, filepath.Join(override, "state.md"), "- Phase: build\n")
	testutil.WriteFile(t, filepath.Join(override, ".red"), "go test ./...\n")

	got, err := StopGate(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Slug != "alpha" {
		t.Fatalf("StopGate slug=%q, want alpha", got.Slug)
	}
	if !got.Blocked || !strings.Contains(got.Reason, "tests/build RED") {
		t.Fatalf("StopGate did not block on override .red: blocked=%v reason=%q", got.Blocked, got.Reason)
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
		testutil.WriteFile(t, filepath.Join(root, "features", slug, name), content)
	}
}

func writeWorkFeature(t *testing.T, root, slug string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		testutil.WriteFile(t, filepath.Join(root, "work", slug, name), content)
	}
}
