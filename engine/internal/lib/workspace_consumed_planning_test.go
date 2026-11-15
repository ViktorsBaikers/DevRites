package lib

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/testutil"
)

func consumedPoolFixture(t *testing.T) (string, string) {
	t.Helper()
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	parent := ""
	for i, slug := range []string{"a", "b", "c"} {
		dir := poolWorkspace(t, root, slug, parent, poolSlice("SLICE-001", "src/"+slug+".go", "Workspace dependencies: none"))
		writeFile(t, filepath.Join(dir, "state.md"), fmt.Sprintf("| phase | build |\n| status | running |\n| schema | 4 |\n| sequence_parent | %s |\n| sequence_position | %d |\n| sequence_workspaces_remaining | %d |\n", parent, i+1, 3-i))
		parent = slug
	}
	writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 2\ncontinue_sequence: true\n")
	writeFile(t, filepath.Join(root, "work", "a", "decisions.md"), "# Decisions\n\n## DEC-001\n\nApproved roadmap identity.\n")
	writeFile(t, filepath.Join(root, "work", "a", "roadmap.md"), "Sequence root: a\nDecision: DEC-001\n\n| Position | Workspace |\n| --- | --- |\n| 1 | a |\n| 2 | b |\n| 3 | c |\n")
	return project, root
}

func TestWorkspaceReadyCanonicalFutureRequiresConsumedInputs(t *testing.T) {
	project, root := consumedPoolFixture(t)
	for _, controller := range []string{"a", "c"} {
		ready, notices, err := WorkspaceReady(project, controller, "c")
		if err != nil || len(ready) != 0 || !strings.Contains(strings.Join(notices, "\n"), "missing Consumed planning inputs") {
			t.Fatalf("controller=%s ready=%v notices=%v err=%v", controller, ready, notices, err)
		}
	}
	ready, notices, err := WorkspaceReady(project, "a", "a")
	if err != nil || len(ready) != 1 || len(notices) != 0 {
		t.Fatalf("root must retain legacy admission: ready=%v notices=%v err=%v", ready, notices, err)
	}
	plan := filepath.Join(root, "work", "c", "plan.md")
	writeFile(t, plan, "# Plan\n\nConsumed planning inputs: none\n")
	poolVet(t, root, "c")
	ready, notices, err = WorkspaceReady(project, "a", "c")
	if err != nil || len(ready) != 0 || !strings.Contains(strings.Join(notices, "\n"), "Predecessor readiness: none") {
		t.Fatalf("missing none binding admitted: ready=%v notices=%v err=%v", ready, notices, err)
	}
	testutil.AppendFile(t, filepath.Join(root, "work", "c", "eng-review.md"), "\nPredecessor readiness: none\n")
	ready, notices, err = WorkspaceReady(project, "a", "c")
	if err != nil || len(ready) != 1 || len(notices) != 0 {
		t.Fatalf("explicit none not admitted: ready=%v notices=%v err=%v", ready, notices, err)
	}
}

func TestWorkspaceReadyConsumedNonadjacentSnapshot(t *testing.T) {
	project, root := consumedPoolFixture(t)
	writeFile(t, filepath.Join(root, "work", "c", "plan.md"), "# Plan\n\nConsumed planning inputs: a\n")
	poolVet(t, root, "c")
	binding, err := gate.ReadinessBinding(root, "a")
	if err != nil {
		t.Fatal(err)
	}
	testutil.AppendFile(t, filepath.Join(root, "work", "c", "eng-review.md"), "\nPredecessor readiness: a "+strings.TrimPrefix(binding, "Readiness inputs SHA-256: ")+"\n")
	for _, slug := range []string{"b", "a"} {
		testutil.AppendFile(t, filepath.Join(root, "work", slug, "plan.md"), "\nChanged design.\n")
		ready, notices, err := WorkspaceReady(project, "a", "c")
		if err != nil {
			t.Fatal(err)
		}
		if slug == "b" && (len(ready) != 1 || len(notices) != 0) {
			t.Fatalf("unconsumed B withheld C: ready=%v notices=%v", ready, notices)
		}
		if slug == "a" && (len(ready) != 0 || !strings.Contains(strings.Join(notices, "\n"), "predecessor a changed")) {
			t.Fatalf("consumed A drift admitted C: ready=%v notices=%v", ready, notices)
		}
	}
}

func TestWorkspaceReadyLegacyDeclarationBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name   string
		afk    string
		legacy bool
	}{
		{"serial", "mode: autocomplete\nmax_parallel: 1\n", false},
		{"standalone", "max_parallel: 2\n", false},
		{"headerless", "mode: autocomplete\nmax_parallel: 2\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project, root := consumedPoolFixture(t)
			writeFile(t, filepath.Join(root, "AFK"), tc.afk)
			if tc.legacy {
				writeFile(t, filepath.Join(root, "work", "a", "roadmap.md"), "| Position | Workspace |\n| --- | --- |\n| 1 | a |\n| 2 | b |\n| 3 | c |\n")
			}
			ready, notices, err := WorkspaceReady(project, "a", "c")
			if err != nil || len(ready) != 1 || len(notices) != 0 {
				t.Fatalf("legacy absent declaration withheld: ready=%v notices=%v err=%v", ready, notices, err)
			}
		})
	}
}
