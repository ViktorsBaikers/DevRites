package lib

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// treeWorkspace is a git project with a committed tracked.go, a .gitignore
// for ignored.out, and .devrites/work/feat in build whose SLICE-001 writes
// src/w.go.
func treeWorkspace(t *testing.T) (root, project string) {
	t.Helper()
	project = initGitRepository(t, t.TempDir())
	root = filepath.Join(project, ".devrites")
	featureDir := filepath.Join(root, "work", "feat")
	writeFile(t, filepath.Join(featureDir, "state.md"), "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | build |\n| status | running |\n")
	writeFile(t, filepath.Join(featureDir, "tasks.md"), "# Tasks\n\n## SLICE-001 Wright\nDependencies: none\nWriter allowlist: src/w.go\n")
	writeFile(t, filepath.Join(project, "tracked.go"), "package a\n")
	writeFile(t, filepath.Join(project, ".gitignore"), "ignored.out\n")
	runDiffScopeGit(t, project, "config", "user.email", "tests@example.invalid")
	runDiffScopeGit(t, project, "config", "user.name", "DevRites Tests")
	runDiffScopeGit(t, project, "add", "tracked.go", ".gitignore")
	runDiffScopeGit(t, project, "commit", "-qm", "base")
	return root, project
}

func runNextT(t *testing.T, root string) (int, string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunNext(root, []string{"feat"}, stdout, stderr)
	return code, stdout.String() + stderr.String()
}

func flaggedPaths(t *testing.T, root string) []string {
	t.Helper()
	flagged, err := unattributedEdits(root, "feat")
	if err != nil {
		t.Fatal(err)
	}
	return flagged
}

// ADR-0038: the tripwire flags root product edits with no live wright
// attribution, never blocks a workspace without a baseline, accepts run-start
// dirt, and takes a completed wright wave's write set.
func TestTreeTripwire(t *testing.T) {
	root, project := treeWorkspace(t)
	featureDir := filepath.Join(root, "work", "feat")

	// Pre-existing dirt: user work plus paths the tripwire never tracks.
	writeFile(t, filepath.Join(project, "tracked.go"), "package a // user\n")
	writeFile(t, filepath.Join(project, "notes.txt"), "user notes\n")
	writeFile(t, filepath.Join(project, "ignored.out"), "gate output\n")
	writeFile(t, filepath.Join(project, ".scratch", "plan.md"), "scratch\n")

	// No baseline yet: nothing is flagged and next does not block.
	if got := flaggedPaths(t, root); len(got) != 0 {
		t.Fatalf("no baseline must flag nothing, got %v", got)
	}
	if code, out := runNextT(t, root); code != 0 || strings.Contains(out, "unattributed") {
		t.Fatalf("next without a baseline: code=%d\n%s", code, out)
	}
	if err := CheckTreeBaseline(root, "feat"); err != nil {
		t.Fatalf("check candidate without a baseline: %v", err)
	}

	// Run start: the first open accepts current dirt as user work.
	if code, out := runDispatchT(t, root, "open", "--phase", "vet", "--wave", "v1", "--role", "a"); code != 0 || !strings.Contains(out, "tree baseline recorded (2 dirty product path(s)") {
		t.Fatalf("first open: code=%d\n%s", code, out)
	}
	raw, err := os.ReadFile(filepath.Join(featureDir, treeBaselineFile))
	if err != nil {
		t.Fatal(err)
	}
	var baseline map[string]string
	if err := json.Unmarshal(raw, &baseline); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for k := range baseline {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	if !slices.Equal(keys, []string{"notes.txt", "tracked.go"}) {
		t.Fatalf("baseline keys = %v; want only dirty product paths", keys)
	}
	if got := flaggedPaths(t, root); len(got) != 0 {
		t.Fatalf("run-start dirt must not be flagged, got %v", got)
	}

	// Root edits: a new file and an amended user-dirty file.
	writeFile(t, filepath.Join(project, "src", "root.go"), "package src\n")
	writeFile(t, filepath.Join(project, "tracked.go"), "package a // root\n")
	if got := flaggedPaths(t, root); !slices.Equal(got, []string{"src/root.go", "tracked.go"}) {
		t.Fatalf("flagged = %v", got)
	}
	if code, out := runNextT(t, root); code != 3 || !strings.Contains(out, "blocked: unattributed root edit: src/root.go, tracked.go") || !strings.Contains(out, "next-phase:") {
		t.Fatalf("next must block and still print the path: code=%d\n%s", code, out)
	}
	if err := CheckTreeBaseline(root, "feat"); err == nil || !strings.Contains(err.Error(), "dispatch feat attribute --reason") {
		t.Fatalf("check candidate must block with the remedy, got %v", err)
	}
	if code, out := runDispatchT(t, root, "open", "--phase", "vet", "--wave", "v2", "--role", "a", "--claim-session", "me", "--claim", "src/root.go"); code != 3 || !strings.Contains(out, "BLOCKED: unattributed root edit: src/root.go, tracked.go") {
		t.Fatalf("a non-wright wave never takes paths on: code=%d\n%s", code, out)
	}
	metrics, _ := os.ReadFile(filepath.Join(featureDir, MetricsFile))
	if !strings.Contains(string(metrics), `"event":"unattributed-edit"`) {
		t.Fatalf("findings must reach metrics.jsonl:\n%s", metrics)
	}

	// A wright wave that does not take every flagged path is still refused.
	wright := []string{"open", "--phase", "build", "--wave", "SLICE-001", "--role", "slice-wright", "--slice", "SLICE-001", "--contract", wrightContractPath(t), "--claim-session", "me", "--claim", "src/root.go"}
	if code, out := runDispatchT(t, root, wright...); code != 3 || !strings.Contains(out, "BLOCKED: unattributed root edit: tracked.go;") {
		t.Fatalf("wright wave leaving tracked.go flagged: code=%d\n%s", code, out)
	}

	// Attribution names only flagged paths, then clears them.
	if code, out := runDispatchT(t, root, "attribute", "--reason", "owner edit", "--path", "src/nope.go"); code != 2 || !strings.Contains(out, "not an unattributed edit") {
		t.Fatalf("attribute of an unflagged path: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "attribute", "--path", "tracked.go"); code != 2 {
		t.Fatalf("attribute without --reason: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "attribute", "--reason", "owner edit DEC-1", "--path", "tracked.go"); code != 0 || !strings.Contains(out, "attributed tracked.go (owner edit DEC-1)") {
		t.Fatalf("attribute: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "status", "--path", "x"); code != 2 {
		t.Fatalf("--path outside attribute: code=%d\n%s", code, out)
	}

	// The wright wave takes src/root.go on through its claim; its in-flight
	// slice write set covers the wright's own edit.
	if code, out := runDispatchT(t, root, wright...); code != 0 {
		t.Fatalf("wright open taking src/root.go: code=%d\n%s", code, out)
	}
	writeFile(t, filepath.Join(project, "src", "w.go"), "package src // wright\n")
	if got := flaggedPaths(t, root); len(got) != 0 {
		t.Fatalf("in-flight wright paths must not be flagged, got %v", got)
	}
	runDispatchT(t, root, "start", "--wave", "SLICE-001", "--role", "slice-wright", "--handle", "h1")
	if code, out := runDispatchT(t, root, "return", "--wave", "SLICE-001", "--role", "slice-wright"); code != 0 {
		t.Fatalf("return: code=%d\n%s", code, out)
	}
	if got := flaggedPaths(t, root); len(got) != 0 {
		t.Fatalf("the completed wave's write set must join the baseline, got %v", got)
	}
	if code, out := runNextT(t, root); code != 0 {
		t.Fatalf("next after the wave: code=%d\n%s", code, out)
	}

	// A root amendment after the wave returned is flagged again; a live
	// claim covers it.
	writeFile(t, filepath.Join(project, "src", "w.go"), "package src // root amendment\n")
	if got := flaggedPaths(t, root); !slices.Equal(got, []string{"src/w.go"}) {
		t.Fatalf("post-return root amendment: flagged = %v", got)
	}
	if code, _, e := runClaim(t, root, "add", "--session", "other", "src"); code != 0 {
		t.Fatalf("claim add: %d %s", code, e)
	}
	if got := flaggedPaths(t, root); len(got) != 0 {
		t.Fatalf("a live claim must cover the path, got %v", got)
	}
}

// A wright wave outside Build (Polish corrections) takes its --claim paths on
// at open and into the baseline at its completing return.
func TestTreeTripwirePolishWrightWave(t *testing.T) {
	root, project := treeWorkspace(t)
	if code, out := runDispatchT(t, root, "open", "--phase", "vet", "--wave", "v1", "--role", "a"); code != 0 {
		t.Fatalf("first open: code=%d\n%s", code, out)
	}
	writeFile(t, filepath.Join(project, "tracked.go"), "package a // polish\n")
	if code, out := runDispatchT(t, root, "open", "--phase", "polish", "--wave", "p1", "--role", "slice-wright", "--claim-session", "me", "--claim", "tracked.go"); code != 0 {
		t.Fatalf("polish wright open: code=%d\n%s", code, out)
	}
	if got := flaggedPaths(t, root); len(got) != 0 {
		t.Fatalf("in-flight polish wright claim must cover its path, got %v", got)
	}
	runDispatchT(t, root, "start", "--wave", "p1", "--role", "slice-wright", "--handle", "h1")
	if code, out := runDispatchT(t, root, "return", "--wave", "p1", "--role", "slice-wright"); code != 0 {
		t.Fatalf("return: code=%d\n%s", code, out)
	}
	if got := flaggedPaths(t, root); len(got) != 0 {
		t.Fatalf("the returned polish wave's claim paths must join the baseline, got %v", got)
	}
}

// A WIP checkpoint commit followed by a later baseline write keeps the
// committed path recorded, so collapsing the checkpoint back into the work
// tree (git reset --soft) flags nothing.
func TestTreeTripwireCheckpointCollapse(t *testing.T) {
	root, project := treeWorkspace(t)
	if code, out := runDispatchT(t, root, "open", "--phase", "vet", "--wave", "v1", "--role", "a"); code != 0 {
		t.Fatalf("first open: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "open", "--phase", "build", "--wave", "SLICE-001", "--role", "slice-wright", "--slice", "SLICE-001", "--contract", wrightContractPath(t)); code != 0 {
		t.Fatalf("wright open: code=%d\n%s", code, out)
	}
	writeFile(t, filepath.Join(project, "src", "w.go"), "package src // wright\n")
	runDispatchT(t, root, "start", "--wave", "SLICE-001", "--role", "slice-wright", "--handle", "h1")
	if code, out := runDispatchT(t, root, "return", "--wave", "SLICE-001", "--role", "slice-wright"); code != 0 {
		t.Fatalf("return: code=%d\n%s", code, out)
	}
	runDiffScopeGit(t, project, "add", "src/w.go")
	runDiffScopeGit(t, project, "commit", "-qm", "WIP(feat): SLICE-001")

	// A second wright wave rewrites the baseline while src/w.go is clean.
	writeFile(t, filepath.Join(project, "tracked.go"), "package a // polish\n")
	if code, out := runDispatchT(t, root, "open", "--phase", "polish", "--wave", "p1", "--role", "slice-wright", "--claim-session", "me", "--claim", "tracked.go"); code != 0 {
		t.Fatalf("second wright open: code=%d\n%s", code, out)
	}
	runDispatchT(t, root, "start", "--wave", "p1", "--role", "slice-wright", "--handle", "h2")
	if code, out := runDispatchT(t, root, "return", "--wave", "p1", "--role", "slice-wright"); code != 0 {
		t.Fatalf("second return: code=%d\n%s", code, out)
	}

	runDiffScopeGit(t, project, "reset", "--soft", "HEAD~1")
	if got := flaggedPaths(t, root); len(got) != 0 {
		t.Fatalf("collapsed checkpoint must not be flagged, got %v", got)
	}
}

// A root that is not a project's .devrites tree, or a project outside git,
// never records a baseline.
func TestTreeBaselineOffOutsideGitProject(t *testing.T) {
	for name, root := range map[string]string{
		"bare root":   t.TempDir(),
		"no git repo": filepath.Join(t.TempDir(), ".devrites"),
	} {
		t.Run(name, func(t *testing.T) {
			writeFile(t, filepath.Join(root, "work", "feat", "state.md"), "# State\n")
			if code, out := runDispatchT(t, root, "open", "--phase", "vet", "--wave", "v1", "--role", "a"); code != 0 || strings.Contains(out, "tree baseline") {
				t.Fatalf("open: code=%d\n%s", code, out)
			}
			if _, err := os.Stat(filepath.Join(root, "work", "feat", treeBaselineFile)); !os.IsNotExist(err) {
				t.Fatalf("baseline written outside a git project: %v", err)
			}
		})
	}
}
