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

func mustObserve(t *testing.T, root, slug string) *state.WorkspaceObservation {
	t.Helper()
	observation, err := state.ObserveWorkspace(root, slug)
	if err != nil {
		t.Fatal(err)
	}
	return observation
}

func TestReadinessBlocksWhenLookaheadPredecessorReplans(t *testing.T) {
	root := t.TempDir()
	parent := writeReadinessFixture(t, root, "parent", "build")
	child := writeReadinessFixture(t, root, "child", "build")
	recorded := strings.TrimPrefix(mustReadinessBinding(t, root, "parent"), readinessBindingLabel)
	testutil.AppendFile(t, filepath.Join(child, "eng-review.md"), "\n"+predecessorLabel+"parent "+recorded+"\n")
	testutil.AppendFile(t, filepath.Join(child, "eng-review.md"), "\n"+mustReadinessBinding(t, root, "child")+"\n")

	result, err := Check(Readiness, root, "child")
	if err != nil {
		t.Fatal(err)
	}
	if result.Blocked {
		t.Fatalf("unchanged predecessor must not block:\n%s", result.Render())
	}

	testutil.WriteFile(t, filepath.Join(parent, "plan.md"), "# Plan\n\nRepaired after the child was vetted.\n")
	result, err = Check(Readiness, root, "child")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Blocked || result.ReasonID != reason.GateReadinessStale {
		t.Fatalf("replanned predecessor: blocked=%v reason=%q", result.Blocked, result.ReasonID)
	}
	if rendered := result.Render(); !strings.Contains(rendered, "predecessor parent changed") || !strings.Contains(rendered, "rerun /rite-vet") {
		t.Fatalf("output lacks the predecessor and the re-vet route:\n%s", rendered)
	}
}

func TestReadinessAcceptsArchivedPredecessorAndDetectsReplan(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	parent := writeReadinessFixture(t, root, "parent", "build")
	child := writeReadinessFixture(t, root, "child", "build")
	t.Setenv("DEVRITES_WORKSPACE", child)
	binding := mustReadinessBinding(t, root, "parent")
	recorded := strings.TrimPrefix(binding, readinessBindingLabel)
	testutil.AppendFile(t, filepath.Join(child, "eng-review.md"), "\n"+predecessorLabel+"parent "+recorded+"\n")
	testutil.AppendFile(t, filepath.Join(child, "eng-review.md"), "\n"+mustReadinessBinding(t, root, "child")+"\n")
	archive := filepath.Join(root, "archive", "parent")
	if err := os.MkdirAll(filepath.Dir(archive), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(parent, archive); err != nil {
		t.Fatal(err)
	}
	if got := mustReadinessBinding(t, root, "parent"); got != binding {
		t.Fatalf("archive changed predecessor identity: got %q want %q", got, binding)
	}
	if _, got, err := ReadinessInputDigests(root, "parent"); err != nil || got != binding {
		t.Fatalf("source-bound archive introspection = (%q, %v), want %q", got, err, binding)
	}
	result, err := Check(Readiness, root, "child")
	if err != nil {
		t.Fatal(err)
	}
	if result.Blocked {
		t.Fatalf("archived predecessor blocked:\n%s", result.Render())
	}
	if _, err := os.Lstat(parent); !os.IsNotExist(err) {
		t.Fatalf("predecessor restored to live workspace: %v", err)
	}

	testutil.WriteFile(t, filepath.Join(archive, "plan.md"), "# Plan\n\nRepaired after archival.\n")
	result, err = Check(Readiness, root, "child")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Blocked || result.ReasonID != reason.GateReadinessStale {
		t.Fatalf("changed archive: blocked=%v reason=%q", result.Blocked, result.ReasonID)
	}
	if rendered := result.Render(); !strings.Contains(rendered, "predecessor parent changed") || strings.Contains(rendered, "cannot be computed") {
		t.Fatalf("changed archive reported unavailable instead of stale:\n%s", rendered)
	}
}

func TestReadinessIgnoresWorkspaceWithoutPredecessorLine(t *testing.T) {
	root := t.TempDir()
	child := writeReadinessFixture(t, root, "child", "build")
	testutil.AppendFile(t, filepath.Join(child, "eng-review.md"), "\n"+mustReadinessBinding(t, root, "child")+"\n")
	if got := predecessorProblems(root, mustObserve(t, root, "child")); got != nil {
		t.Fatalf("no predecessor line must mean no check, got %v", got)
	}
}

func TestReadinessRefusesMalformedPredecessorLine(t *testing.T) {
	root := t.TempDir()
	child := writeReadinessFixture(t, root, "child", "build")
	testutil.AppendFile(t, filepath.Join(child, "eng-review.md"), "\n"+predecessorLabel+"../escape deadbeef\n")
	got := predecessorProblems(root, mustObserve(t, root, "child"))
	if len(got) != 1 || !strings.Contains(got[0], "malformed") {
		t.Fatalf("malformed line: %v", got)
	}
}
