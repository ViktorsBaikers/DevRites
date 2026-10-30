package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/parallel"
)

// planTasks: SLICE-001 is built; 002-004 depend only on it; 003 adds an
// external dependency (so it claims go.mod/go.sum) and 004 writes go.sum;
// 005 waits on unbuilt 002.
const planTasks = `# Tasks

## SLICE-001 Store
Dependencies: none
Writer allowlist: src/a.go

## SLICE-002 Read
Dependencies: SLICE-001
Writer allowlist: src/b.go
External dependencies: none

## SLICE-003 Fetch
Dependencies: SLICE-001
Writer allowlist: src/c.go
External dependencies: github.com/example/fetch

## SLICE-004 Pin
Dependencies: SLICE-001
Files likely touched: go.sum; src/d.go

## SLICE-005 Later
Dependencies: SLICE-002
Writer allowlist: src/e.go
`

// planWorkspace lays out <project>/.devrites/work/feat in build with
// planTasks, SLICE-001 checked built, and a go.mod at the project root.
func planWorkspace(t *testing.T) (root, featureDir string) {
	t.Helper()
	project := t.TempDir()
	root = filepath.Join(project, ".devrites")
	featureDir = filepath.Join(root, "work", "feat")
	writeFile(t, filepath.Join(featureDir, "state.md"), "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | build |\n| status | running |\n\n## Slice checkpoints\n- [x] SLICE-001 built — abc1234\n- [ ] SLICE-002 pending\n\n```markdown\n- [x] SLICE-002 built — example only\n```\n")
	writeFile(t, filepath.Join(featureDir, "tasks.md"), planTasks)
	writeFile(t, filepath.Join(project, "go.mod"), "module example\n")
	return root, featureDir
}

func writePlanLease(t *testing.T, root string) {
	t.Helper()
	path, err := parallel.LeasePath(filepath.Dir(root), "feat")
	if err != nil {
		t.Fatal(err)
	}
	lease := &parallel.Lease{
		BatchID: "b1", CreatedAt: "2026-10-20T00:00:00Z", BaseSHA: "abcdef1", N: 2,
		Status: parallel.StatusRunning, ControlPIDOrSession: "test",
		Slices: []parallel.LeaseSlice{
			{ID: "SLICE-002", Paths: []string{"src/b.go"}, WorktreePath: "/tmp/w2", Branch: "devrites/parallel/feat/b1/SLICE-002", WrightStatus: parallel.WrightPending},
			{ID: "SLICE-009", Paths: []string{"src/z.go"}, WorktreePath: "/tmp/w9", Branch: "devrites/parallel/feat/b1/SLICE-009", WrightStatus: parallel.WrightIntegrated},
		},
	}
	if err := parallel.WriteLease(path, lease); err != nil {
		t.Fatal(err)
	}
}

func TestPlanReadyFromTasksAndCheckpoints(t *testing.T) {
	root, _ := planWorkspace(t)
	ready, err := planReady(root, "feat")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	var order []string
	for _, s := range ready {
		got[s.ID] = s.Paths
		order = append(order, s.ID)
	}
	// Built SLICE-001 and SLICE-005 (its dependency is checked only inside a
	// fence) are not ready; plan order holds.
	if !slices.Equal(order, []string{"SLICE-002", "SLICE-003", "SLICE-004"}) {
		t.Fatalf("ready order %v", order)
	}
	if !slices.Equal(got["SLICE-002"], []string{"src/b.go"}) {
		t.Fatalf("`External dependencies: none` must add no manifest: %v", got["SLICE-002"])
	}
	if !slices.Equal(got["SLICE-003"], []string{"src/c.go", "go.mod", "go.sum"}) {
		t.Fatalf("external dependency must claim go.mod and go.sum: %v", got["SLICE-003"])
	}

	// A disjoint ready pair is batched; SLICE-004's go.sum collides with
	// SLICE-003's dependency surface, so it waits.
	eligible, batch, err := planBatch(root, "feat", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(eligible, []string{"SLICE-002", "SLICE-003", "SLICE-004"}) || !slices.Equal(batch, []string{"SLICE-002", "SLICE-003"}) {
		t.Fatalf("eligible=%v batch=%v", eligible, batch)
	}

	// A running lease's slices are neither eligible nor picked again; its
	// active sibling still holds its paths and slot.
	writePlanLease(t, root)
	eligible, batch, err = planBatch(root, "feat", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(eligible, []string{"SLICE-003", "SLICE-004"}) || !slices.Equal(batch, []string{"SLICE-003"}) {
		t.Fatalf("with lease: eligible=%v batch=%v", eligible, batch)
	}
}

func TestPlanBatchExistingDepsStayParallel(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	feature := filepath.Join(root, "work", "feat")
	writeFile(t, filepath.Join(project, "go.mod"), "module example\n")
	writeFile(t, filepath.Join(project, "go.sum"), "\n")
	writeFile(t, filepath.Join(feature, "state.md"), "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | build |\n| status | running |\n")
	writeFile(t, filepath.Join(feature, "tasks.md"), `# Tasks

## SLICE-001 List
Dependencies: none
Writer allowlist: internal/library/list.go
External dependencies: existing SQLite/goose only; no new module.

## SLICE-002 Tool
Dependencies: none
Writer allowlist: scripts/s5-toolchain.sh
External dependencies: exact isolated x/perf tool and its verified compiled closure; no product go.mod/go.sum or web lock edits.

## SLICE-003 Redact
Dependencies: none
Writer allowlist: scripts/dev-run.sh
External dependencies: stdlib and existing host/redactor tools only.
`)
	paths := []string{"internal/library/list.go"}
	for _, ext := range []string{
		"already-admitted benchstat; existing modules only.",
		"existing make/Docker/Bun/Go tools only.",
		"none new (stdlib crypto/aes, crypto/cipher)",
	} {
		if got := withManifests(project, paths, ext); !slices.Equal(got, paths) {
			t.Fatalf("%q reserved a manifest: %v", ext, got)
		}
	}
	if got := withManifests(project, paths, "existing modules plus github.com/foo/bar v1.2.3"); !slices.Equal(got, []string{"internal/library/list.go", "go.mod", "go.sum"}) {
		t.Fatalf("named module must still reserve go.mod: %v", got)
	}

	eligible, batch, err := planBatch(root, "feat", 5)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"SLICE-001", "SLICE-002", "SLICE-003"}
	if !slices.Equal(eligible, want) || !slices.Equal(batch, want) {
		t.Fatalf("eligible=%v batch=%v", eligible, batch)
	}
}

func TestPlanReadyNearestManifestAndMissingPaths(t *testing.T) {
	root, featureDir := planWorkspace(t)
	project := filepath.Dir(root)
	if err := os.Remove(filepath.Join(project, "go.mod")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(project, "web", "package.json"), "{}\n")
	writeFile(t, filepath.Join(project, "web", "pnpm-lock.yaml"), "x\n")
	if got := withManifests(project, []string{"web/src/app.ts"}, "`left-pad`"); !slices.Equal(got, []string{"web/src/app.ts", "web/package.json", "web/pnpm-lock.yaml"}) {
		t.Fatalf("nearest manifest with its present lockfile: %v", got)
	}
	if got := withManifests(project, []string{"src/c.go"}, "github.com/example/fetch"); !slices.Equal(got, []string{"src/c.go"}) {
		t.Fatalf("no manifest above the path: %v", got)
	}

	// The golden-workspace checkpoint form `Slice N: <name> — built` counts too.
	writeFile(t, filepath.Join(featureDir, "state.md"), "# State\n\n## Slice progress\n- [x] Slice 1: store — built\n- [ ] Slice 2: read\n")
	ready, err := planReady(root, "feat")
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, s := range ready {
		order = append(order, s.ID)
	}
	if !slices.Equal(order, []string{"SLICE-002", "SLICE-003", "SLICE-004"}) {
		t.Fatalf("`Slice 1: store — built` checkpoint: ready order %v", order)
	}

	writeFile(t, filepath.Join(featureDir, "tasks.md"), strings.Replace(planTasks, "Writer allowlist: src/b.go", "Goal: read", 1))
	if _, err := planReady(root, "feat"); err == nil || !strings.Contains(err.Error(), "SLICE-002 has no Writer allowlist") {
		t.Fatalf("a ready slice without write paths must be named: %v", err)
	}
}

func TestNextPrintsEligibleBatchInBuild(t *testing.T) {
	root, _ := planWorkspace(t)
	writeFile(t, filepath.Join(root, "AFK"), "max_slices: 4\nmax_parallel: 3\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunNext(root, []string{"feat"}, stdout, stderr); code != 0 {
		t.Fatalf("code=%d err=%s", code, stderr)
	}
	want := "eligible_now: SLICE-002,SLICE-003,SLICE-004\nbatch: SLICE-002,SLICE-003\nselect-command: devrites-engine parallel select --root " + filepath.Dir(root) + " --slug feat --cap 3 --from-plan\n"
	if out := stdout.String(); !strings.Contains(out, want) {
		t.Fatalf("want %q in:\n%s", want, out)
	}

	// The printed command reproduces the batch through the parallel CLI.
	saved := parallel.PlanReady
	parallel.PlanReady = PlanReady
	t.Cleanup(func() { parallel.PlanReady = saved })
	stdout.Reset()
	code := parallel.Run("parallel", []string{"select", "--root", filepath.Dir(root), "--slug", "feat", "--cap", "3", "--from-plan"}, strings.NewReader(""), stdout, stderr)
	if code != 0 || stdout.String() != "select: n_eff=2 mode=parallel slices=SLICE-002,SLICE-003\n" {
		t.Fatalf("select --from-plan code=%d out=%q err=%s", code, stdout, stderr)
	}
}

func TestNextBatchUnavailableIsAdvisory(t *testing.T) {
	root, featureDir := planWorkspace(t)
	if err := os.Remove(filepath.Join(featureDir, "tasks.md")); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunNext(root, []string{"feat"}, stdout, stderr); code != 0 {
		t.Fatalf("code=%d err=%s", code, stderr)
	}
	if out := stdout.String(); !strings.Contains(out, "batch: unavailable (") || strings.Contains(out, "eligible_now:") {
		t.Fatalf("want an advisory unavailable line:\n%s", out)
	}
}
