package lib

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func frontierFixture(t *testing.T, root, location, slug, parent, phase, status, next string, position int, remaining string) string {
	t.Helper()
	dir := filepath.Join(root, location, slug)
	body := cursor(phase, status, next)
	if parent != "" {
		body += "| sequence_parent | " + parent + " |\n"
	}
	if position > 0 {
		body += fmt.Sprintf("| sequence_position | %d |\n", position)
	}
	if remaining != "" {
		body += "| sequence_workspaces_remaining | " + remaining + " |\n"
	}
	writeFile(t, filepath.Join(dir, "state.md"), body)
	return dir
}

func frontierText(t *testing.T, root, slug string) string {
	t.Helper()
	lines, err := FrontierLines(root, slug)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(lines, "\n")
}

func frontierSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			files[path] = string(raw)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestFrontierPreparedPoolReadOnly(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
	poolWorkspace(t, root, "child", "parent", poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: none"))
	poolWorkspace(t, root, "outsider", "", poolSlice("SLICE-001", "src/outside.go", ""))
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 2\n")
	before := frontierSnapshot(t, root)
	text := frontierText(t, root, "parent")
	for _, want := range []string{"frontier: actionable", "frontier-source: parent", "frontier-workspace: child location=live", "--workspaces 'parent,child'", "--slug 'parent'", "frontier-picks: SLICE-001,child/SLICE-001", "--cap 2"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "outsider") {
		t.Fatalf("unrelated writer leaked:\n%s", text)
	}
	if after := frontierSnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("frontier observation mutated workflow bytes")
	}
}

func TestFrontierArchiveBridgeAndExactFuture(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	frontierFixture(t, root, "work", "source", "", "seal", "ready", "/rite-ship source", 1, "3")
	frontierFixture(t, root, "archive", "middle", "source", "done", "shipped", "/rite-spec child", 2, "2")
	frontierFixture(t, root, "work", "child", "middle", "seal", "ready", "/rite-spec missing", 3, "1")
	text := frontierText(t, root, "source")
	for _, want := range []string{"frontier: actionable", "frontier-source: source", "middle location=archive action=complete", "child location=live action=complete", "missing location=missing action=prepare", "frontier-prepare: /rite-spec 'missing' source=child", "--slug 'source'"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "work", "missing")); !os.IsNotExist(err) {
		t.Fatal("future workspace created")
	}
}

func TestFrontierHumanGateKeepsIndependentPlanning(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	frontierFixture(t, root, "work", "parent", "", "vet", "awaiting_human", "/rite-build SLICE-001", 1, "")
	frontierFixture(t, root, "work", "blocked-child", "parent", "plan", "blocked", "/rite-vet blocked-child", 2, "")
	frontierFixture(t, root, "work", "planning-child", "parent", "plan", "running", "/rite-vet planning-child", 2, "")
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 2\n")
	var out, errOut bytes.Buffer
	if code := RunCheckContinuation(root, []string{"parent"}, &out, &errOut); code != 0 {
		t.Fatalf("code=%d: %s", code, &errOut)
	}
	for _, want := range []string{"continuation: required", "invoke: /rite-autocomplete", "source: parent", "parent location=live action=wait reason=human/blocked", "planning-child location=live action=prepare"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q:\n%s", want, &out)
		}
	}
}

func TestFrontierMissingAndStalePlanningDoNotHidePreparedChild(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
	stale := poolWorkspace(t, root, "stale", "parent", poolSlice("SLICE-001", "src/stale.go", "Workspace dependencies: none"))
	writeFile(t, filepath.Join(stale, "plan.md"), "# Plan\nChanged since Vet.\n")
	poolWorkspace(t, root, "ready", "parent", poolSlice("SLICE-001", "src/ready.go", "Workspace dependencies: none"))
	text := frontierText(t, root, "parent")
	if !strings.Contains(text, "stale location=live action=repair") || !strings.Contains(text, "ready/SLICE-001") || strings.Contains(text, "frontier-picks: stale/") {
		t.Fatalf("stale planning poisoned pool:\n%s", text)
	}
}

func TestFrontierRecordedMapsAndAmbiguity(t *testing.T) {
	for _, tc := range []struct {
		name, tables, want string
		prepare            bool
	}{
		{"slug table", "| Position | Slug | Slices |\n| --- | --- | --- |\n| 1 | source | 2 |\n| 2 | future | 3 |\n", "frontier-prepare: /rite-spec 'future'", true},
		{"workspace table", "| Position | Milestone | Workspace | Objective |\n| --- | --- | --- | --- |\n| 1 | First | source | Start |\n| 2 | Next | future | Continue |\n", "frontier-prepare: /rite-spec 'future'", true},
		{"amendment", "| Position | Slug |\n| --- | --- |\n| 1 | source |\n| 2 | old |\n\nAccepted amendment\n\n| Position | Workspace |\n| --- | --- |\n| 1 | source |\n| 2 | replacement |\n", "native authority-normalization required", false},
		{"duplicate", "| Position | Slug |\n| --- | --- |\n| 1 | source |\n| 2 | future |\n| 2 | other |\n", "duplicate sequence position", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			dir := frontierFixture(t, root, "archive", "source", "", "done", "shipped", "", 1, "1")
			writeFile(t, filepath.Join(dir, "decisions.md"), "# Sequence\n\n"+tc.tables)
			text := frontierText(t, root, "source")
			if !strings.Contains(text, tc.want) || strings.Contains(text, "frontier-prepare:") != tc.prepare {
				t.Fatalf("unexpected map outcome:\n%s", text)
			}
		})
	}
}

func TestFrontierAmbiguousMapPreservesLiveChild(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	parent := poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
	poolWorkspace(t, root, "child", "parent", poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: none"))
	writeFile(t, filepath.Join(parent, "roadmap.md"), "| Position | Slug |\n| --- | --- |\n| 1 | parent |\n| 2 | old |\n\n| Position | Slug |\n| --- | --- |\n| 1 | parent |\n| 2 | revised |\n")
	text := frontierText(t, root, "parent")
	if !strings.Contains(text, "native authority-normalization") || !strings.Contains(text, "child/SLICE-001") {
		t.Fatalf("legacy map hid valid descendant:\n%s", text)
	}
}

func TestFrontierEmptyActiveRecoveryRefusesAmbiguity(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	frontierFixture(t, root, "archive", "tip", "", "done", "shipped", "/rite-spec future", 1, "1")
	writeFile(t, filepath.Join(root, "ACTIVE"), "\n")
	before := frontierSnapshot(t, root)
	text := frontierText(t, root, "")
	if !strings.Contains(text, "frontier-source: tip") || !strings.Contains(text, "frontier-prepare:") {
		t.Fatal(text)
	}
	if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
		t.Fatal("recovery changed ACTIVE or workspace")
	}
	frontierFixture(t, root, "archive", "other-tip", "", "done", "shipped", "/rite-spec other-future", 1, "1")
	if _, err := FrontierLines(root, ""); err == nil || !strings.Contains(err.Error(), "unique archived terminal") {
		t.Fatalf("ambiguous resume accepted: %v", err)
	}
}

func TestFrontierInvalidChainsAndCounters(t *testing.T) {
	for _, tc := range []struct{ name, parent, extra, want string }{
		{"cycle", "source", "", "cycle"},
		{"missing parent", "missing", "", "missing"},
		{"negative remaining", "", "| sequence_workspaces_remaining | -1 |\n", "nonnegative"},
		{"duplicate remaining", "", "| sequence_workspaces_remaining | 1 |\n| sequence_workspaces_remaining | 2 |\n", "duplicate"},
		{"padded position", "", "| sequence_position | 01 |\n", "canonical"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			dir := frontierFixture(t, root, "work", "source", tc.parent, "plan", "running", "/rite-vet source", 0, "")
			raw, err := os.ReadFile(filepath.Join(dir, "state.md"))
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(dir, "state.md"), string(raw)+tc.extra)
			text := frontierText(t, root, "source")
			if !strings.HasPrefix(text, "frontier: blocked") || !strings.Contains(text, tc.want) || strings.Contains(text, "frontier-select-command") {
				t.Fatalf("invalid sequence admitted:\n%s", text)
			}
		})
	}
}

func TestFrontierOptOutAndBudgetsRetainExistingChild(t *testing.T) {
	for _, optout := range []string{"continue_sequence: false\n", "continue_sequence: true\n"} {
		t.Run(strings.TrimSpace(optout), func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			remaining := "2"
			if strings.Contains(optout, "true") {
				remaining = "0"
			}
			frontierFixture(t, root, "work", "parent", "", "seal", "ready", "/rite-ship parent", 1, remaining)
			frontierFixture(t, root, "archive", "tip", "parent", "done", "shipped", "/rite-spec absent", 2, "1")
			frontierFixture(t, root, "work", "child", "tip", "plan", "running", "/rite-vet child", 3, "1")
			writeFile(t, filepath.Join(root, "AFK"), optout)
			text := frontierText(t, root, "parent")
			if strings.Contains(text, "frontier-prepare:") || !strings.Contains(text, "child location=live action=prepare") {
				t.Fatal(text)
			}
		})
	}
}

func TestFrontierWaitingDoesNotForceContinuation(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	frontierFixture(t, root, "work", "parent", "", "seal", "ready", "/rite-ship parent", 1, "0")
	child := frontierFixture(t, root, "work", "child", "parent", "plan", "running", "/rite-vet child", 2, "0")
	writeFile(t, filepath.Join(child, "dispatch.json"), `{"waves":{"plan":{"phase":"plan","state":"sealed","roles":{"planner":{"handle":"retained","started":"2026-11-03T10:00:00Z","returned":false}}}}}`)
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 2\n")
	var out, errOut bytes.Buffer
	if code := RunCheckContinuation(root, []string{"parent"}, &out, &errOut); code != 0 {
		t.Fatalf("%d: %s", code, &errOut)
	}
	if !strings.Contains(out.String(), "continuation: clear") || !strings.Contains(out.String(), "frontier: waiting") || strings.Contains(out.String(), "frontier-picks:") {
		t.Fatal(&out)
	}
}

func TestFrontierPreservesWorkspaceOverride(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	parent := frontierFixture(t, root, "work", "parent", "", "seal", "ready", "/rite-ship parent", 1, "0")
	frontierFixture(t, root, "work", "child", "parent", "plan", "running", "/rite-vet child", 2, "0")
	t.Setenv("DEVRITES_WORKSPACE", parent)
	text := frontierText(t, root, "parent")
	if !strings.Contains(text, "DEVRITES_WORKSPACE scope retained") || strings.HasPrefix(text, "frontier: actionable") {
		t.Fatal(text)
	}
}

func TestFrontierArchivedBridgeStillSelectsLiveDescendant(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
	frontierFixture(t, root, "archive", "middle", "parent", "done", "shipped", "/rite-spec child", 0, "")
	poolWorkspace(t, root, "child", "middle", poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: none"))
	text := frontierText(t, root, "parent")
	if !strings.Contains(text, "middle location=archive action=complete") || !strings.Contains(text, "frontier-picks: SLICE-001,child/SLICE-001") || strings.Contains(text, "--workspaces 'parent,middle") {
		t.Fatal(text)
	}
}

func TestFrontierRelatedMalformedRefusedWithoutHidingValidChild(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
	poolWorkspace(t, root, "child", "parent", poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: none"))
	writeFile(t, filepath.Join(root, "work", "bad", "state.md"), "| sequence_parent | parent |\n| sequence_workspaces_remaining | unknown |\n")
	writeFile(t, filepath.Join(root, "work", "unrelated", "state.md"), "```broken\n")
	text := frontierText(t, root, "parent")
	if !strings.Contains(text, "bad location=live action=repair") || !strings.Contains(text, "child/SLICE-001") || strings.Contains(text, "unrelated") {
		t.Fatal(text)
	}
}

func TestFrontierMissingPlanAndPartialNoticesPreserveReadySlices(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
	missing := poolWorkspace(t, root, "missing-plan", "parent", poolSlice("SLICE-001", "src/missing.go", "Workspace dependencies: none"))
	if err := os.Remove(filepath.Join(missing, "plan.md")); err != nil {
		t.Fatal(err)
	}
	poolWorkspace(t, root, "child", "parent", poolSlice("SLICE-001", "src/approved.go", "Workspace dependencies: none")+poolSlice("SLICE-002", "src/legacy.go", ""))
	text := frontierText(t, root, "parent")
	if !strings.Contains(text, "missing-plan location=live action=repair") || !strings.Contains(text, "child/SLICE-001") || strings.Contains(text, "frontier-picks: SLICE-001,child/SLICE-001,child/SLICE-002") {
		t.Fatal(text)
	}
}

func TestFrontierNeverOffersHeldWriterInSelectCommand(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
	child := poolWorkspace(t, root, "child", "parent", poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: none"))
	writeFile(t, filepath.Join(child, "dispatch.json"), `{"waves":{"writer":{"phase":"build","state":"sealed","slices":["SLICE-001"],"roles":{"slice-wright":{"handle":"retained","started":"2026-11-03T10:00:00Z","returned":false}}}}}`)
	text := frontierText(t, root, "parent")
	if !strings.Contains(text, "child location=live action=wait") || !strings.Contains(text, "--workspaces 'parent'") || strings.Contains(text, "frontier-picks: SLICE-001,child/") {
		t.Fatal(text)
	}
}

func TestFrontierTerminalPoolDoesNotRequireContinuation(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	frontierFixture(t, root, "work", "parent", "", "done", "shipped", "", 1, "0")
	frontierFixture(t, root, "archive", "child", "parent", "done", "shipped", "", 2, "0")
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 2\n")
	var out, errOut bytes.Buffer
	if code := RunCheckContinuation(root, []string{"parent"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "continuation: clear") || !strings.Contains(out.String(), "frontier: clear") {
		t.Fatalf("%d: %s %s", code, &out, &errOut)
	}
}

func TestFrontierUnknownStatusAndInvalidPolicyFailClosed(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	frontierFixture(t, root, "work", "parent", "", "build", "paused", "/rite-build SLICE-001", 1, "1")
	text := frontierText(t, root, "parent")
	if !strings.HasPrefix(text, "frontier: blocked") || strings.Contains(text, "frontier-select-command") {
		t.Fatal(text)
	}
	for _, policy := range []string{"max_parallel: 0\n", "max_parallel: unknown\n", "continue_sequence: unknown\n", "max_parallel: 2\nmax_parallel: 3\n"} {
		writeFile(t, filepath.Join(root, "AFK"), policy)
		if _, err := FrontierLines(root, "parent"); err == nil {
			t.Fatalf("invalid AFK policy admitted: %q", policy)
		}
	}
}
