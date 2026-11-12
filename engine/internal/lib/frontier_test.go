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

func TestFrontierEmptyActiveRecoversLiveContinuation(t *testing.T) {
	for _, active := range []string{"missing", "", " \n"} {
		t.Run(fmt.Sprintf("ACTIVE=%q", active), func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			frontierFixture(t, root, "archive", "parent", "", "done", "shipped", "/rite-spec continuation", 1, "1")
			frontierFixture(t, root, "work", "continuation", "parent", "plan", "running", "/rite-vet continuation", 2, "0")
			frontierFixture(t, root, "work", "standalone", "", "plan", "running", "/rite-vet standalone", 0, "")
			frontierFixture(t, root, "work", "position-only", "", "plan", "running", "/rite-vet position-only", 1, "")
			writeFile(t, filepath.Join(root, "work", "loose", "notes.md"), "Not sequence authority.\n")
			if active != "missing" {
				writeFile(t, filepath.Join(root, "ACTIVE"), active)
			}
			before := frontierSnapshot(t, root)
			var out, errOut bytes.Buffer
			if code := RunFrontier(root, nil, &out, &errOut); code != 0 {
				t.Fatalf("code=%d: %s", code, &errOut)
			}
			text := out.String()
			for _, want := range []string{"frontier: actionable", "frontier-source: continuation", "parent location=archive action=complete", "continuation location=live action=prepare"} {
				if !strings.Contains(text, want) {
					t.Fatalf("missing %q:\n%s", want, text)
				}
			}
			for _, unrelated := range []string{"standalone", "position-only", "loose"} {
				if strings.Contains(text, unrelated) {
					t.Fatalf("unrelated workspace %s recovered:\n%s", unrelated, text)
				}
			}
			if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
				t.Fatal("recovery changed ACTIVE or workspace")
			}
		})
	}
}

func TestFrontierEmptyActiveOrdersRelatedLiveOwners(t *testing.T) {
	for _, numbered := range []bool{false, true} {
		t.Run(fmt.Sprintf("numbered=%t", numbered), func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			for i, member := range []struct{ slug, parent, location string }{
				{"root", "", "archive"},
				{"z-owner", "root", "work"},
				{"middle", "z-owner", "archive"},
				{"a-child", "middle", "work"},
			} {
				position, remaining := 0, ""
				if numbered {
					position, remaining = i+1, "0"
				}
				phase, status, next := "done", "shipped", ""
				if member.location == "work" {
					phase, status, next = "plan", "running", "/rite-vet "+member.slug
				}
				frontierFixture(t, root, member.location, member.slug, member.parent, phase, status, next, position, remaining)
			}
			before := frontierSnapshot(t, root)
			text := frontierText(t, root, "")
			if !strings.Contains(text, "frontier-source: z-owner\n") || !strings.Contains(text, "--slug 'z-owner'") || !strings.Contains(text, "--workspaces 'z-owner,a-child'") {
				t.Fatalf("sequence order not retained:\n%s", text)
			}
			if again := frontierText(t, root, ""); again != text {
				t.Fatalf("recovery not deterministic:\n%s\n%s", text, again)
			}
			if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
				t.Fatal("recovery changed workflow bytes")
			}
		})
	}
}

func TestFrontierEmptyActiveRefusesSiblingOwners(t *testing.T) {
	for _, numbered := range []bool{false, true} {
		t.Run(fmt.Sprintf("numbered=%t", numbered), func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			parentPosition, childPosition, remaining := 0, 0, ""
			if numbered {
				parentPosition, childPosition, remaining = 1, 2, "0"
			}
			frontierFixture(t, root, "archive", "parent", "", "done", "shipped", "", parentPosition, remaining)
			for _, slug := range []string{"a-child", "z-child"} {
				frontierFixture(t, root, "work", slug, "parent", "build", "running", "/rite-build SLICE-001", childPosition, remaining)
			}
			before := frontierSnapshot(t, root)
			lines, err := FrontierLines(root, "")
			if err == nil || !strings.Contains(err.Error(), "multiple equally earliest live owners") || len(lines) != 0 {
				t.Fatalf("sibling owner invented: lines=%v err=%v", lines, err)
			}
			var out, errOut bytes.Buffer
			if code := RunFrontier(root, nil, &out, &errOut); code != 2 || out.Len() != 0 || !strings.Contains(errOut.String(), "multiple equally earliest live owners") {
				t.Fatalf("sibling recovery: code=%d stdout=%s stderr=%s", code, &out, &errOut)
			}
			if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
				t.Fatal("sibling recovery changed workflow bytes")
			}
		})
	}
}

func TestFrontierEmptyActiveWithoutSequence(t *testing.T) {
	for _, workspace := range []string{"none", "standalone", "empty parent"} {
		t.Run(workspace, func(t *testing.T) {
			root := t.TempDir()
			if workspace != "none" {
				dir := frontierFixture(t, root, "work", "standalone", "", "plan", "running", "/rite-vet standalone", 0, "")
				if workspace == "empty parent" {
					writeFile(t, filepath.Join(dir, "state.md"), cursor("plan", "running", "/rite-vet standalone")+"| sequence_parent | |\n")
				}
			}
			before := frontierSnapshot(t, root)
			if text := frontierText(t, root, ""); text != "frontier: none" {
				t.Fatalf("fresh idea blocked or standalone adopted:\n%s", text)
			}
			var out, errOut bytes.Buffer
			if code := RunFrontier(root, nil, &out, &errOut); code != 0 || out.String() != "frontier: none\n" || errOut.Len() != 0 {
				t.Fatalf("fresh recovery: code=%d stdout=%s stderr=%s", code, &out, &errOut)
			}
			if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
				t.Fatal("fresh recovery changed workflow bytes")
			}
		})
	}
}

func TestFrontierEmptyActiveRecoversNumberedRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	frontierFixture(t, root, "work", "root", "", "plan", "running", "/rite-vet root", 1, "0")
	text := frontierText(t, root, "")
	if !strings.Contains(text, "frontier-source: root\n") || !strings.Contains(text, "root location=live action=prepare") {
		t.Fatal(text)
	}
}

func TestFrontierEmptyActiveRefusesIndependentSequences(t *testing.T) {
	for _, otherLocation := range []string{"work", "archive"} {
		t.Run(otherLocation, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			frontierFixture(t, root, "archive", "parent", "", "done", "shipped", "/rite-spec child", 1, "1")
			frontierFixture(t, root, "work", "child", "parent", "plan", "running", "/rite-vet child", 2, "0")
			if otherLocation == "work" {
				frontierFixture(t, root, "work", "other", "", "plan", "running", "/rite-vet other", 1, "0")
			} else {
				frontierFixture(t, root, "archive", "other", "", "done", "shipped", "/rite-spec future", 1, "1")
			}
			before := frontierSnapshot(t, root)
			var out, errOut bytes.Buffer
			if code := RunFrontier(root, nil, &out, &errOut); code != 2 || out.Len() != 0 || !strings.Contains(errOut.String(), "found 2") {
				t.Fatalf("ambiguous recovery: code=%d stdout=%s stderr=%s", code, &out, &errOut)
			}
			if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
				t.Fatal("ambiguous recovery changed workflow bytes")
			}
		})
	}
}

func TestFrontierEmptyActiveRefusesInvalidCompetingSequence(t *testing.T) {
	for _, tc := range []struct {
		name, parent, remaining, extra, want string
		position                             int
	}{
		{name: "missing parent", parent: "missing", want: "sequence_parent missing missing"},
		{name: "invalid parent", parent: "../outside", want: "invalid sequence_parent"},
		{name: "cycle", parent: "invalid", want: "cycle"},
		{name: "position gap", parent: "valid", position: 3, want: "does not follow valid"},
		{name: "orphan position", position: 2, remaining: "0", want: "has no sequence_parent"},
		{name: "negative counter", position: 1, remaining: "-1", want: "nonnegative"},
		{name: "padded counter", position: 1, remaining: "01", want: "canonical"},
		{name: "duplicate counter", position: 1, remaining: "1", extra: "| sequence_workspaces_remaining | 2 |\n", want: "duplicate"},
		{name: "padded position", remaining: "0", extra: "| sequence_position | 01 |\n", want: "canonical"},
		{name: "duplicate empty parent", extra: "| sequence_parent | |\n| sequence_parent | |\n", want: "duplicate sequence_parent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			frontierFixture(t, root, "work", "valid", "", "plan", "running", "/rite-vet valid", 1, "0")
			dir := frontierFixture(t, root, "work", "invalid", tc.parent, "build", "running", "/rite-build SLICE-001", tc.position, tc.remaining)
			if tc.extra != "" {
				raw, err := os.ReadFile(filepath.Join(dir, "state.md"))
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(dir, "state.md"), string(raw)+tc.extra)
			}
			before := frontierSnapshot(t, root)
			lines, err := FrontierLines(root, "")
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "repair native sequence metadata") || len(lines) != 0 {
				t.Fatalf("invalid recovery admitted: lines=%v err=%v", lines, err)
			}
			var out, errOut bytes.Buffer
			if code := RunFrontier(root, nil, &out, &errOut); code != 2 || out.Len() != 0 || !strings.Contains(errOut.String(), tc.want) {
				t.Fatalf("invalid recovery: code=%d stdout=%s stderr=%s", code, &out, &errOut)
			}
			if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
				t.Fatal("invalid recovery changed workflow bytes")
			}
		})
	}
}

func TestFrontierEmptyActiveRecoveryRetainsHumanGate(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	frontierFixture(t, root, "archive", "parent", "", "done", "shipped", "/rite-spec held", 1, "1")
	frontierFixture(t, root, "work", "held", "parent", "vet", "awaiting_human", "/rite-build SLICE-001", 2, "0")
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 2\n")
	before := frontierSnapshot(t, root)
	text := frontierText(t, root, "")
	for _, want := range []string{"frontier: blocked", "frontier-source: held", "held location=live action=wait reason=human/blocked cursor; retain native gate"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "frontier-picks:") || strings.Contains(text, "frontier-select-command:") {
		t.Fatalf("human-held source offered to writer:\n%s", text)
	}
	var out, errOut bytes.Buffer
	if code := RunCheckContinuation(root, []string{"held"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "continuation: clear") || strings.Contains(out.String(), "continuation: required") {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, &out, &errOut)
	}
	if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
		t.Fatal("human-held recovery changed workflow bytes")
	}
}

func TestFrontierEmptyActiveRecoveryPreservesWorkspaceOverride(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	frontierFixture(t, root, "work", "sequence", "", "plan", "running", "/rite-vet sequence", 1, "0")
	standalone := frontierFixture(t, root, "work", "standalone", "", "plan", "running", "/rite-vet standalone", 0, "")
	before := frontierSnapshot(t, root)
	t.Setenv("DEVRITES_WORKSPACE", standalone)
	if text := frontierText(t, root, ""); text != "frontier: none" {
		t.Fatalf("override replaced by recovered sequence:\n%s", text)
	}
	if _, err := FrontierLines(root, "sequence"); err == nil || !strings.Contains(err.Error(), "DRV-WORKSPACE-SLUG-MISMATCH") {
		t.Fatalf("explicit source bypassed override: %v", err)
	}
	t.Setenv("DEVRITES_WORKSPACE", t.TempDir())
	if _, err := FrontierLines(root, ""); err == nil || !strings.Contains(err.Error(), "DRV-WORKSPACE-OUTSIDE-ROOT") {
		t.Fatalf("invalid override bypassed by recovery: %v", err)
	}
	if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
		t.Fatal("override recovery changed workflow bytes")
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
