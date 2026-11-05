package lib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func canonicalRoadmapFixture(t *testing.T) (string, string, string, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".devrites")
	var current, historical strings.Builder
	for _, table := range []*strings.Builder{&current, &historical} {
		table.WriteString("| Position | Workspace |\n| --- | --- |\n")
	}
	parent, anchor, tip := "", "", ""
	for position := 1; position <= 17; position++ {
		slug := fmt.Sprintf("step-%02d", position)
		if position == 9 {
			slug = "renamed-09"
		}
		fmt.Fprintf(&current, "| %d | %s |\n", position, slug)
		old := slug
		if position == 9 {
			old = "old-09"
		}
		fmt.Fprintf(&historical, "| %d | %s |\n", position, old)
		if position == 17 {
			continue
		}
		location, phase, status := "archive", "done", "shipped"
		if position == 16 {
			location, phase, status = "work", "seal", "ready"
		}
		dir := frontierFixture(t, root, location, slug, parent, phase, status, "", position, "1")
		if position == 1 {
			anchor = dir
		}
		tip, parent = dir, slug
	}
	writeFile(t, filepath.Join(anchor, "decisions.md"), "# Decisions\n\n## DEC-017\n\nApproved current sequence identity.\n\n"+historical.String())
	writeFile(t, filepath.Join(anchor, "spec.md"), "# Historical sequence\n\n"+historical.String())
	writeFile(t, filepath.Join(tip, "decisions.md"), "# Decisions\n\n## DEC-017\n\nApproved current sequence identity.\n")
	return root, anchor, tip, "Sequence root: step-01\nDecision: DEC-017\n\n" + current.String()
}

func TestFrontierCanonicalRoadmapSupersedesHistoryReadOnly(t *testing.T) {
	for _, owner := range []string{"archive", "live"} {
		t.Run(owner, func(t *testing.T) {
			root, anchor, tip, roadmap := canonicalRoadmapFixture(t)
			before := frontierSnapshot(t, root)
			text := frontierText(t, root, "step-16")
			if !strings.Contains(text, "native authority-normalization required") || strings.Contains(text, "frontier-prepare:") {
				t.Fatalf("historical old-09 accepted as bound renamed-09 without authority:\n%s", text)
			}
			if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
				t.Fatal("legacy refusal changed historical bytes")
			}
			dir := anchor
			if owner == "live" {
				dir = tip
			}
			writeFile(t, filepath.Join(dir, "roadmap.md"), roadmap)
			before = frontierSnapshot(t, root)
			var out, errOut bytes.Buffer
			if code := RunFrontier(root, []string{"step-16"}, &out, &errOut); code != 0 {
				t.Fatalf("canonical frontier code=%d: %s", code, &errOut)
			}
			text = out.String()
			authority := "frontier-authority: " + filepath.Join(dir, "roadmap.md") + "," + filepath.Join(dir, "decisions.md")
			authorityFound := false
			for _, line := range strings.Split(text, "\n") {
				authorityFound = authorityFound || line == authority
			}
			if !authorityFound {
				t.Fatalf("missing exact current authority paths %q:\n%s", authority, text)
			}
			for _, want := range []string{"frontier: actionable", "renamed-09 location=archive", "step-17 location=missing action=prepare", "frontier-prepare: /rite-spec 'step-17' source=step-16"} {
				if !strings.Contains(text, want) {
					t.Fatalf("canonical roadmap missing %q:\n%s", want, text)
				}
			}
			if strings.Contains(text, "native authority-normalization") || strings.Contains(text, "old-09 location=") {
				t.Fatalf("historical table still controls scheduling:\n%s", text)
			}
			if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
				t.Fatal("canonical observation rewrote history or created future workspace")
			}
		})
	}
}

func TestFrontierCanonicalRoadmapRetainsLocalHolds(t *testing.T) {
	for _, hold := range []string{"human", "blocked", "question", "dispatch", "invalid cursor"} {
		t.Run(hold, func(t *testing.T) {
			root, anchor, tip, roadmap := canonicalRoadmapFixture(t)
			writeFile(t, filepath.Join(anchor, "roadmap.md"), roadmap)
			switch hold {
			case "human", "blocked":
				status := "awaiting_human"
				if hold == "blocked" {
					status = "blocked"
				}
				frontierFixture(t, root, "work", "step-16", "step-15", "seal", status, "", 16, "1")
			case "question":
				writeFile(t, filepath.Join(tip, "questions.md"), "## Q-001\nstatus: open\ngate: blocking\n")
			case "dispatch":
				writeFile(t, filepath.Join(tip, "dispatch.json"), "{")
			case "invalid cursor":
				frontierFixture(t, root, "work", "step-16", "step-15", "seal", "ready", "/rite-ship other-workspace", 16, "1")
			}
			before := frontierSnapshot(t, root)
			text := frontierText(t, root, "step-16")
			if !strings.Contains(text, "frontier: blocked") || strings.Contains(text, "frontier-prepare:") {
				t.Fatalf("local hold authorized future preparation:\n%s", text)
			}
			if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
				t.Fatal("held observation changed workspace bytes")
			}
		})
	}
}

func TestFrontierCanonicalRoadmapRetainsWorkspaceScope(t *testing.T) {
	for _, next := range []string{"", "/rite-ship step-16"} {
		t.Run(next, func(t *testing.T) {
			root, anchor, tip, roadmap := canonicalRoadmapFixture(t)
			writeFile(t, filepath.Join(anchor, "roadmap.md"), roadmap)
			frontierFixture(t, root, "work", "step-16", "step-15", "seal", "ready", next, 16, "1")
			t.Setenv("DEVRITES_WORKSPACE", tip)
			before := frontierSnapshot(t, root)
			text := frontierText(t, root, "step-16")
			if strings.Contains(text, "frontier-prepare:") || strings.Contains(text, "step-17 location=missing action=prepare") {
				t.Fatalf("matching workspace scope authorized future preparation:\n%s", text)
			}
			if !strings.Contains(text, "step-16 location=live action=wait") {
				t.Fatalf("scope changed local cursor disposition:\n%s", text)
			}
			if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
				t.Fatal("scoped observation changed workspace bytes")
			}
		})
	}
}

func TestFrontierCanonicalRoadmapMasksUnclosedFence(t *testing.T) {
	root, anchor, _, roadmap := canonicalRoadmapFixture(t)
	// CommonMark fences may run to EOF; fenced authority must stay inert.
	writeFile(t, filepath.Join(anchor, "roadmap.md"), roadmap+"\n```markdown\nSequence root: other\nDecision: DEC-999\n| Position | Workspace |\n| --- | --- |\n| 1 | other |\n")
	text := frontierText(t, root, "step-16")
	if !strings.Contains(text, "frontier-prepare: /rite-spec 'step-17' source=step-16") || strings.Contains(text, "native authority-normalization") {
		t.Fatalf("fenced tail changed canonical authority:\n%s", text)
	}
}

func TestFrontierCanonicalRoadmapInvalidAuthorityNeverFallsBack(t *testing.T) {
	for _, name := range []string{
		"wrong root", "missing root", "missing decision", "duplicate decision", "decision in other owner",
		"multiple owners", "position gap", "duplicate position", "duplicate slug", "invalid slug",
		"conflicting bound node", "unenrolled bound node", "conflicting next Spec", "second table",
		"missing table", "invalid UTF-8", "NUL byte", "symlink", "unreadable", "nonregular",
		"unrelated second table", "duplicate root header", "decision prefix only", "fenced decision only",
	} {
		t.Run(name, func(t *testing.T) {
			root, anchor, tip, roadmap := canonicalRoadmapFixture(t)
			// Legacy authority agrees with cursors, so fallback would visibly prepare step-17.
			table := roadmap[strings.Index(roadmap, "| Position |"):]
			writeFile(t, filepath.Join(anchor, "decisions.md"), "# Decisions\n\n## DEC-017\n\n"+table)
			writeFile(t, filepath.Join(anchor, "spec.md"), "# Spec\n\n"+table)
			path := filepath.Join(anchor, "roadmap.md")
			switch name {
			case "wrong root":
				roadmap = strings.Replace(roadmap, "Sequence root: step-01", "Sequence root: step-16", 1)
			case "missing root":
				roadmap = strings.Replace(roadmap, "Sequence root: step-01\n", "", 1)
			case "missing decision":
				roadmap = strings.Replace(roadmap, "Decision: DEC-017\n", "", 1)
			case "duplicate decision":
				writeFile(t, filepath.Join(anchor, "decisions.md"), "## DEC-017\n\n## DEC-017\n\n"+table)
			case "decision in other owner":
				writeFile(t, filepath.Join(anchor, "decisions.md"), "# No decision here\n\n"+table)
			case "multiple owners":
				writeFile(t, filepath.Join(tip, "roadmap.md"), roadmap)
			case "position gap":
				roadmap = strings.Replace(roadmap, "| 17 | step-17 |", "| 18 | step-17 |", 1)
			case "duplicate position":
				roadmap += "| 17 | other-future |\n"
			case "duplicate slug":
				roadmap = strings.Replace(roadmap, "| 17 | step-17 |", "| 17 | step-16 |", 1)
			case "invalid slug":
				roadmap = strings.Replace(roadmap, "| 17 | step-17 |", "| 17 | ../escape |", 1)
			case "conflicting bound node":
				roadmap = strings.Replace(roadmap, "| 9 | renamed-09 |", "| 9 | old-09 |", 1)
			case "unenrolled bound node":
				frontierFixture(t, root, "work", "unlisted", "step-16", "plan", "running", "/rite-vet unlisted", 0, "1")
			case "conflicting next Spec":
				frontierFixture(t, root, "work", "step-16", "step-15", "seal", "ready", "/rite-spec other-future", 16, "1")
			case "second table":
				roadmap += "\n" + table
			case "missing table":
				roadmap = "Sequence root: step-01\nDecision: DEC-017\n"
			case "invalid UTF-8":
				roadmap += "\xff"
			case "NUL byte":
				roadmap += "\x00"
			case "unrelated second table":
				roadmap += "\n| Note | Value |\n| --- | --- |\n| extra | table |\n"
			case "duplicate root header":
				roadmap = "Sequence root: step-01\n" + roadmap
			case "decision prefix only":
				writeFile(t, filepath.Join(anchor, "decisions.md"), "## DEC-0170\n\n"+table)
			case "fenced decision only":
				writeFile(t, filepath.Join(anchor, "decisions.md"), "```markdown\n## DEC-017\n```\n\n"+table)
			}
			writeFile(t, path, roadmap)
			switch name {
			case "symlink":
				target := filepath.Join(t.TempDir(), "roadmap.md")
				writeFile(t, target, roadmap)
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			case "unreadable":
				if runtime.GOOS == "windows" {
					t.Skip("Windows does not enforce Unix file read permissions")
				}
				if os.Geteuid() == 0 {
					t.Skip("root bypasses file read permissions")
				}
				if err := os.Chmod(path, 0); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(path, 0600) })
			case "nonregular":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			lines, err := FrontierLines(root, "step-16")
			text := strings.Join(lines, "\n")
			if err == nil && !strings.Contains(text, "native authority-normalization required") {
				t.Fatalf("invalid canonical authority accepted:\n%s", text)
			}
			if strings.Contains(text, "frontier-prepare:") {
				t.Fatalf("invalid canonical authority fell back to legacy scheduling:\n%s", text)
			}
			if _, err := os.Stat(filepath.Join(root, "work", "step-17")); !os.IsNotExist(err) {
				t.Fatalf("invalid authority created future workspace: %v", err)
			}
		})
	}
}

func TestFrontierHeaderlessRoadmapRemainsLegacy(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	dir := frontierFixture(t, root, "archive", "source", "", "done", "shipped", "", 1, "1")
	writeFile(t, filepath.Join(dir, "roadmap.md"), "| Position | Workspace |\n| --- | --- |\n| 1 | source |\n| 2 | future |\n")
	text := frontierText(t, root, "source")
	if !strings.Contains(text, "frontier-prepare: /rite-spec 'future'") || strings.Contains(text, "native authority-normalization") {
		t.Fatalf("headerless legacy roadmap rejected:\n%s", text)
	}
}

func TestContinuationCanonicalOwnerDiscoveredBeyondSourceAncestors(t *testing.T) {
	for _, location := range []string{"work", "archive"} {
		for _, duplicate := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/duplicate=%t", location, duplicate), func(t *testing.T) {
				root := filepath.Join(t.TempDir(), ".devrites")
				frontierFixture(t, root, "work", "source", "", "plan", "running", "/rite-spec child", 1, "2")
				child := frontierFixture(t, root, location, "child", "source", "done", "shipped", "/rite-spec future", 2, "1")
				roadmap := "Sequence root: source\nDecision: DEC-001\n\n| Position | Workspace |\n| --- | --- |\n| 1 | source |\n| 2 | child |\n| 3 | future |\n"
				writeFile(t, filepath.Join(child, "decisions.md"), "## DEC-001\n\nApproved sequence.\n")
				writeFile(t, filepath.Join(child, "roadmap.md"), roadmap)
				if duplicate {
					// Both owners lie outside source's ancestor chain.
					future := frontierFixture(t, root, "work", "future", "child", "plan", "running", "/rite-vet future", 3, "0")
					writeFile(t, filepath.Join(future, "decisions.md"), "## DEC-001\n\nApproved sequence.\n")
					writeFile(t, filepath.Join(future, "roadmap.md"), roadmap)
				}
				writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 2\n")
				before := frontierSnapshot(t, root)
				var out, errOut bytes.Buffer
				code := RunCheckContinuation(root, []string{"source"}, &out, &errOut)
				text := out.String()
				if duplicate {
					if code == 0 && !strings.Contains(text, "native authority-normalization") {
						t.Fatalf("existing child bypassed duplicate descendant owners:\n%s\n%s", text, &errOut)
					}
					if strings.Contains(text, "frontier-prepare:") || strings.Contains(text, "invoke: /rite-spec") {
						t.Fatalf("duplicate authority authorized successor:\n%s", text)
					}
				} else {
					if code != 0 {
						t.Fatalf("descendant canonical owner ignored: code=%d %s", code, &errOut)
					}
					for _, want := range []string{"continuation: required", "invoke: /rite-autocomplete", "frontier-prepare: /rite-spec 'future' source=child"} {
						if !strings.Contains(text, want) {
							t.Fatalf("missing %q:\n%s", want, text)
						}
					}
					if strings.Contains(text, "native authority-normalization") {
						t.Fatalf("valid descendant owner refused:\n%s", text)
					}
				}
				if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
					t.Fatal("continuation changed sequence artifacts")
				}
			})
		}
	}
}
