package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	got := res.Render()
	for _, want := range []string{
		"gate: readiness",
		"feature: alpha",
		"phase: build",
		`result: blocked (missing to leave "build": decisions, tasks)`,
		"next: add real content to decisions.md, tasks.md, then re-run: devrites-engine readiness alpha",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Render() missing %q in:\n%s", want, got)
		}
	}
}

func TestStopGateRestPointInvariants(t *testing.T) {
	for _, tc := range []struct {
		name          string
		files         map[string]string
		wantBlocked   bool
		wantReasonSub string
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
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "ACTIVE", "alpha\n")
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
		})
	}
}

func TestOpenBlockingQuestionGates(t *testing.T) {
	got := openBlockingQuestionGates([]byte("## q-1\nstatus: open\ngate: blocking\n\n## Not a question\n\n## q-2\nstatus: open\ngate: validating\n\n## q-3\nstatus: resolved\ngate: blocking\n\n## q-4\nstatus: open\ngate: blocking\n"))
	want := []string{"blocking", "validating"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("openBlockingQuestionGates=%v, want %v", got, want)
	}
}

func writeFeature(t *testing.T, root, slug string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		writeFile(t, root, filepath.Join("features", slug, name), content)
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
