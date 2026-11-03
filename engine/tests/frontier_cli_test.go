package main_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/testutil"
)

func frontierCLIWorkspace(t *testing.T, root, slug, cursor string, ready bool) string {
	t.Helper()
	dir := filepath.Join(root, "work", slug)
	testutil.WriteFile(t, filepath.Join(dir, "state.md"), "# State\n\n| Key | Value |\n| --- | --- |\n| schema | 4 |\n"+cursor)
	if ready {
		for name, content := range map[string]string{
			"brief.md": "# Brief\n\nReady.\n", "spec.md": "# Spec\n\nReady.\n",
			"decisions.md": "# Decisions\n\nNone.\n", "assumptions.md": "# Assumptions\n\nNone.\n",
			"questions.md": "# Questions\n\nNone.\n", "decision-coverage.md": "# Decision coverage\n\nCLEAR\n",
			"architecture.md": "# Architecture\n\nReady.\n", "plan.md": "# Plan\n\nReady.\n",
			"tasks.md":        "# Tasks\n\n## SLICE-001 Ready\nDependencies: none\nWriter allowlist: src/" + slug + ".go\nWorkspace dependencies: none\n",
			"traceability.md": "# Traceability\n\nReady.\n", "test-plan.md": "# Test plan\n\nReady.\n",
			"gates.md": testutil.CanonicalGatesMarkdown,
		} {
			testutil.WriteFile(t, filepath.Join(dir, name), content)
		}
		binding, err := gate.ReadinessBinding(root, slug)
		if err != nil {
			t.Fatal(err)
		}
		testutil.WriteFile(t, filepath.Join(dir, "eng-review.md"), "# Engineering review\n\nREADY\n\n"+binding+"\n")
	}
	return dir
}

func frontierCLISnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			files[path] = "directory"
			return nil
		}
		data, err := os.ReadFile(path)
		files[path] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestFrontierCLILocalSelectionReportsPreparationWithoutBroadening(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	testutil.WriteFile(t, filepath.Join(root, "ACTIVE"), "parent\n")
	frontierCLIWorkspace(t, root, "parent", "| phase | build |\n| status | running |\n| sequence_position | 1 |\n| sequence_workspaces_remaining | 2 |\n| next_action | /rite-build SLICE-001 |\n", true)
	frontierCLIWorkspace(t, root, "child", "| phase | build |\n| status | running |\n| sequence_parent | parent |\n| sequence_position | 2 |\n| sequence_workspaces_remaining | 1 |\n| next_action | /rite-spec future |\n", true)
	frontierCLIWorkspace(t, root, "unrelated", "| phase | build |\n| status | running |\n", true)
	before := frontierCLISnapshot(t, root)
	out, errOut, code := runDevrites(t, root, "parallel", "select", "--root", filepath.Dir(root), "--slug", "parent", "--from-plan", "--cap", "5")
	if code != 0 || out != "select: n_eff=1 mode=serial slices=SLICE-001\n" {
		t.Fatalf("local selection: code=%d out=%q err=%q", code, out, errOut)
	}
	if !strings.Contains(errOut, "frontier-prepare: /rite-spec 'future'") || !strings.Contains(errOut, "frontier-select-command:") || strings.Contains(errOut, "unrelated") {
		t.Fatalf("missing or broadened frontier: %s", errOut)
	}
	out, errOut, code = runDevrites(t, root, "frontier", "parent")
	if code != 0 || !strings.Contains(out, "frontier-source: parent") || !strings.Contains(out, "frontier-workspace: child") || strings.Contains(out, "unrelated") {
		t.Fatalf("frontier: code=%d out=%q err=%q", code, out, errOut)
	}
	out, errOut, code = runDevrites(t, root, "parallel", "select", "--root", filepath.Dir(root), "--slug", "parent", "--from-plan", "--workspaces", "parent,child", "--cap", "5")
	if code != 0 || out != "select: n_eff=2 mode=parallel slices=SLICE-001,child/SLICE-001\n" || strings.Contains(errOut, "frontier:") {
		t.Fatalf("explicit mixed selection: code=%d out=%q err=%q", code, out, errOut)
	}
	if after := frontierCLISnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("frontier or local selection mutated workflow files")
	}
}

func TestFrontierCLIExplicitPoolExcludesPendingControllerWriter(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	testutil.WriteFile(t, filepath.Join(root, "ACTIVE"), "parent\n")
	parent := frontierCLIWorkspace(t, root, "parent", "| phase | build |\n| status | running |\n| sequence_position | 1 |\n| sequence_workspaces_remaining | 2 |\n| next_action | /rite-build SLICE-001 |\n", true)
	frontierCLIWorkspace(t, root, "child", "| phase | build |\n| status | running |\n| sequence_parent | parent |\n| sequence_position | 2 |\n| sequence_workspaces_remaining | 1 |\n| next_action | /rite-build SLICE-001 |\n", true)
	testutil.WriteFile(t, filepath.Join(parent, "dispatch.json"), `{"waves":{"writer":{"phase":"build","state":"sealed","slices":["SLICE-001"],"roles":{"slice-wright":{"handle":"retained","started":"2026-01-01T00:00:00Z","returned":false}}}}}`)
	before := frontierCLISnapshot(t, root)
	out, errOut, code := runDevrites(t, root, "frontier", "parent")
	if code != 0 || !strings.Contains(out, "parent location=live action=wait") || !strings.Contains(out, "--slug 'parent'") || !strings.Contains(out, "--workspaces 'child'") || !strings.Contains(out, "frontier-picks: child/SLICE-001") {
		t.Fatalf("frontier failed to withhold pending controller: code=%d out=%q err=%q", code, out, errOut)
	}
	out, errOut, code = runDevrites(t, root, "parallel", "select", "--root", filepath.Dir(root), "--slug", "parent", "--from-plan", "--workspaces", "child", "--cap", "5")
	if code != 0 || out != "select: n_eff=1 mode=serial slices=child/SLICE-001\n" {
		t.Fatalf("explicit frontier pool reintroduced pending controller: code=%d out=%q err=%q", code, out, errOut)
	}
	if after := frontierCLISnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("frontier selection mutated workflow files")
	}
}

func TestFrontierCLINoticesPrecedeLocalReadinessFailure(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	frontierCLIWorkspace(t, root, "parent", "| phase | done |\n| status | shipped |\n| sequence_position | 1 |\n| sequence_workspaces_remaining | 1 |\n| next_action | /rite-spec future |\n", false)
	out, errOut, code := runDevrites(t, root, "parallel", "select", "--root", filepath.Dir(root), "--slug", "parent", "--from-plan", "--cap", "5")
	notice, failure := strings.Index(errOut, "frontier-prepare:"), strings.LastIndex(errOut, "parallel select:")
	if code != 3 || out != "" || notice < 0 || failure <= notice || !strings.Contains(errOut, "/rite-spec 'future'") {
		t.Fatalf("notice must survive local failure: code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestFrontierCLIArchivedResumeAndAmbiguousSource(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	testutil.WriteFile(t, filepath.Join(root, "ACTIVE"), "")
	archive := func(slug string) {
		t.Helper()
		dir := frontierCLIWorkspace(t, root, slug, "| phase | done |\n| status | shipped |\n| sequence_position | 1 |\n| sequence_workspaces_remaining | 1 |\n| next_action | /rite-spec "+slug+"-next |\n", false)
		if err := os.MkdirAll(filepath.Join(root, "archive"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(dir, filepath.Join(root, "archive", slug)); err != nil {
			t.Fatal(err)
		}
	}
	archive("old")
	before := frontierCLISnapshot(t, root)
	for _, args := range [][]string{{"frontier"}, {"frontier", "old"}} {
		out, errOut, code := runDevrites(t, root, args...)
		if code != 0 || !strings.Contains(out, "frontier-source: old") || !strings.Contains(out, "/rite-spec 'old-next'") {
			t.Fatalf("archived resume %v: code=%d out=%q err=%q", args, code, out, errOut)
		}
	}
	if !reflect.DeepEqual(before, frontierCLISnapshot(t, root)) {
		t.Fatal("archived discovery mutated workflow files")
	}
	if out, errOut, code := runDevrites(t, root, "frontier", "missing"); code == 0 || errOut == "" {
		t.Fatalf("wrong source accepted: code=%d out=%q err=%q", code, out, errOut)
	}
	archive("other")
	if out, errOut, code := runDevrites(t, root, "frontier"); code == 0 || !strings.Contains(errOut, "unique archived terminal sequence tip") {
		t.Fatalf("ambiguous source accepted: code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestFrontierCLIAFKLifecycleKeepsLocalCursor(t *testing.T) {
	for _, phase := range []string{"spec", "vet", "build", "seal", "done"} {
		t.Run(phase, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			testutil.WriteFile(t, filepath.Join(root, "AFK"), "max_parallel: 5\n")
			frontierCLIWorkspace(t, root, "parent", "| phase | "+phase+" |\n| status | running |\n| sequence_position | 1 |\n| sequence_workspaces_remaining | 2 |\n| next_action | /rite-vet parent |\n", false)
			frontierCLIWorkspace(t, root, "child", "| phase | done |\n| status | shipped |\n| sequence_parent | parent |\n| sequence_position | 2 |\n| sequence_workspaces_remaining | 1 |\n| next_action | /rite-spec future |\n", false)
			for _, command := range []string{"next", "handoff"} {
				out, errOut, code := runDevrites(t, root, command, "parent")
				if code != 0 || strings.Count(out, "frontier-source: parent\n") != 1 || !strings.Contains(out, "/rite-spec 'future'") {
					t.Fatalf("%s %s: code=%d out=%q err=%q", command, phase, code, out, errOut)
				}
				local := "source: parent\n"
				if command == "handoff" {
					local = "next_action: /rite-vet parent\n"
					continuation, continuationErr, continuationCode := runDevrites(t, root, "check", "continuation", "parent")
					if continuationCode != 0 || !strings.HasSuffix(out, continuation) {
						t.Fatalf("handoff differs from check continuation: code=%d err=%q continuation=%q handoff=%q", continuationCode, continuationErr, continuation, out)
					}
				}
				if !strings.Contains(out, local) {
					t.Fatalf("%s lost local cursor: %s", command, out)
				}
			}
		})
	}
}
