package lib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/parallel"
	"github.com/devrites/devrites/internal/testutil"
)

func poolSlice(id, paths, declaration string) string {
	return fmt.Sprintf("\n## %s Ready\nDependencies: none\nWriter allowlist: %s\n%s\n", id, paths, declaration)
}

func poolWorkspace(t *testing.T, root, slug, parent, tasks string) string {
	t.Helper()
	dir := filepath.Join(root, "work", slug)
	for name, content := range map[string]string{
		"state.md": "| phase | build |\n| status | running |\n| schema | 4 |\n| sequence_parent | " + parent + " |\n",
		"brief.md": "# Brief\n\nReady.\n", "spec.md": "# Spec\n\nReady.\n",
		"decisions.md": "# Decisions\n\nNone.\n", "assumptions.md": "# Assumptions\n\nNone.\n",
		"questions.md": "# Questions\n\nNone.\n", "decision-coverage.md": "# Decision coverage\n\nCLEAR\n",
		"architecture.md": "# Architecture\n\nReady.\n", "plan.md": "# Plan\n\nReady.\n",
		"tasks.md": "# Tasks\n" + tasks, "traceability.md": "# Traceability\n\nReady.\n",
		"test-plan.md": "# Test plan\n\nReady.\n", "gates.md": testutil.CanonicalGatesMarkdown,
	} {
		writeFile(t, filepath.Join(dir, name), content)
	}
	poolVet(t, root, slug)
	return dir
}

func poolVet(t *testing.T, root, slug string) {
	t.Helper()
	binding, err := gate.ReadinessBinding(root, slug)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "work", slug, "eng-review.md"), "# Engineering review\n\nREADY\n\n"+binding+"\n")
}

func poolKeys(ready []parallel.SlicePaths) []string {
	var keys []string
	for _, candidate := range ready {
		keys = append(keys, parallel.SliceKey(candidate.Workspace, candidate.ID))
	}
	return keys
}

func TestWorkspaceReadyIndependentFutureAndDurableProducer(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	parent := poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
	poolWorkspace(t, root, "child", "parent",
		poolSlice("SLICE-001", "src/independent.go", "Workspace dependencies: none")+
			poolSlice("SLICE-002", "src/dependent.go", "Workspace dependencies: parent/SLICE-001"))
	local, notices, err := WorkspaceReady(project, "parent", "parent")
	if err != nil || len(notices) != 0 || !slices.Equal(poolKeys(local), []string{"SLICE-001"}) {
		t.Fatalf("controller ready=%v notices=%v err=%v", local, notices, err)
	}
	for _, evidence := range []string{
		"## EVID-001: SLICE-001 writer green\nAll checks passed; branch not landed.\n",
		"## EVID-001: SLICE-001 not checkpointed\nCheckpoint: abc1234\n",
	} {
		writeFile(t, filepath.Join(parent, "evidence.md"), evidence)
		ready, notices, err := WorkspaceReady(project, "parent", "child")
		if err != nil || !slices.Equal(poolKeys(ready), []string{"child/SLICE-001"}) || !strings.Contains(strings.Join(notices, "\n"), "parent/SLICE-001") {
			t.Fatalf("unlanded producer: ready=%v notices=%v err=%v", ready, notices, err)
		}
		mixed := append(append([]parallel.SlicePaths{}, local...), ready...)
		picked, err := parallel.SelectGreedy(2, mixed, project)
		if err != nil || !slices.Equal(poolKeys(picked), []string{"SLICE-001", "child/SLICE-001"}) {
			t.Fatalf("duplicate IDs must stay workspace-distinct: picks=%v err=%v", picked, err)
		}
	}
	writeFile(t, filepath.Join(parent, "evidence.md"), "## EVID-002: SLICE-001 repaired and checkpointed\nCheckpoint: abc1234\n")
	ready, notices, err := WorkspaceReady(project, "parent", "child")
	if err != nil || len(notices) != 0 || !slices.Equal(poolKeys(ready), []string{"child/SLICE-001", "child/SLICE-002"}) {
		t.Fatalf("durable checkpoint must release dependent: ready=%v notices=%v err=%v", ready, notices, err)
	}
}

func TestWorkspaceReadyWithheldFuturePreservesController(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*testing.T, string, string)
		want []string
	}{
		{"missing workspace", func(t *testing.T, root, dir string) {
			if err := os.RemoveAll(dir); err != nil {
				t.Fatal(err)
			}
		}, []string{"unavailable"}},
		{"missing plan", func(t *testing.T, root, dir string) {
			if err := os.Remove(filepath.Join(dir, "plan.md")); err != nil {
				t.Fatal(err)
			}
		}, []string{"plan.md"}},
		{"stale and human blocked", func(t *testing.T, root, dir string) {
			writeFile(t, filepath.Join(dir, "plan.md"), "# Plan\n\nChanged since Vet.\n")
			writeFile(t, filepath.Join(dir, "questions.md"), "## q-1\nstatus: open\ngate: escalating\n")
		}, []string{"stale", "/rite-vet", "human question"}},
		{"awaiting human", func(t *testing.T, root, dir string) {
			writeFile(t, filepath.Join(dir, "state.md"), "| phase | build |\n| status | awaiting_human |\n| sequence_parent | parent |\n")
		}, []string{"awaiting_human"}},
		{"early planning", func(t *testing.T, root, dir string) {
			writeFile(t, filepath.Join(dir, "state.md"), "| phase | plan |\n| sequence_parent | parent |\n")
		}, []string{"not Build entry", "Plan/Vet"}},
		{"finished", func(t *testing.T, root, dir string) {
			writeFile(t, filepath.Join(dir, "state.md"), "| phase | done |\n| sequence_parent | parent |\n")
		}, []string{"not Build entry"}},
		{"shipped", func(t *testing.T, root, dir string) {
			writeFile(t, filepath.Join(dir, "state.md"), "| phase | build |\n| status | shipped |\n| sequence_parent | parent |\n")
		}, []string{"status shipped"}},
		{"unrecorded", func(t *testing.T, root, dir string) {
			writeFile(t, filepath.Join(dir, "state.md"), "| phase | build |\n")
		}, []string{"not a recorded forward continuation"}},
		{"fenced sequence parent", func(t *testing.T, root, dir string) {
			writeFile(t, filepath.Join(dir, "state.md"), "| phase | build |\n```text\n| sequence_parent | parent |\n```\n")
		}, []string{"not a recorded forward continuation"}},
		{"legacy declaration", func(t *testing.T, root, dir string) {
			writeFile(t, filepath.Join(dir, "tasks.md"), "# Tasks\n"+poolSlice("SLICE-001", "src/child.go", ""))
			poolVet(t, root, "child")
		}, []string{"missing Workspace dependencies", "Plan/Vet"}},
		{"fenced declaration", func(t *testing.T, root, dir string) {
			writeFile(t, filepath.Join(dir, "tasks.md"), "# Tasks\n"+poolSlice("SLICE-001", "src/child.go", "```text\nWorkspace dependencies: none\n```"))
			poolVet(t, root, "child")
		}, []string{"missing Workspace dependencies"}},
		{"unbound workspace declaration", func(t *testing.T, root, dir string) {
			writeFile(t, filepath.Join(dir, "tasks.md"), "# Tasks\n"+poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: parent/SLICE-001"))
		}, []string{"stale", "/rite-vet"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := t.TempDir()
			root := filepath.Join(project, ".devrites")
			poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
			dir := poolWorkspace(t, root, "child", "parent", poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: none"))
			tc.edit(t, root, dir)
			ready, notices, err := WorkspaceReady(project, "parent", "child")
			if err != nil || len(ready) != 0 {
				t.Fatalf("unsafe future admitted: ready=%v notices=%v err=%v", ready, notices, err)
			}
			for _, want := range tc.want {
				if !strings.Contains(strings.Join(notices, "\n"), want) {
					t.Errorf("notices=%v missing %q", notices, want)
				}
			}
			ready, notices, err = WorkspaceReady(project, "parent", "parent")
			if err != nil || len(notices) != 0 || !slices.Equal(poolKeys(ready), []string{"SLICE-001"}) {
				t.Fatalf("future hid controller: ready=%v notices=%v err=%v", ready, notices, err)
			}
		})
	}
}

func TestWorkspaceReadyReferenceFailuresAreCandidateLocal(t *testing.T) {
	for _, tc := range []struct{ name, declaration, producer, want string }{
		{"bare reference", "SLICE-001", "", "malformed"},
		{"unsafe slug", "../SLICE-001", "", "malformed"},
		{"empty declaration", "", "", "malformed"},
		{"mixed none", "none, parent/SLICE-001", "", "malformed"},
		{"trailing separator", "parent/SLICE-001,", "", "malformed"},
		{"missing producer", "missing/SLICE-001", "", "producer missing"},
		{"unknown slice", "parent/SLICE-999", "", "unknown"},
		{"invalid producer graph", "producer/SLICE-001", "\n## SLICE-001 Invalid\nDependencies: SLICE-999\n", "tasks.md"},
		{"cross workspace cycle", "producer/SLICE-001", poolSlice("SLICE-001", "src/producer.go", "Workspace dependencies: child/SLICE-002"), "cycle"},
		{"duplicate declaration", "none\nWorkspace dependencies: parent/SLICE-001", "", "duplicate"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := t.TempDir()
			root := filepath.Join(project, ".devrites")
			poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
			if tc.producer != "" {
				poolWorkspace(t, root, "producer", "parent", tc.producer)
			}
			poolWorkspace(t, root, "child", "parent",
				poolSlice("SLICE-001", "src/independent.go", "Workspace dependencies: none")+
					poolSlice("SLICE-002", "src/dependent.go", "Workspace dependencies: "+tc.declaration))
			ready, notices, err := WorkspaceReady(project, "parent", "child")
			if err != nil || !slices.Equal(poolKeys(ready), []string{"child/SLICE-001"}) || !strings.Contains(strings.Join(notices, "\n"), tc.want) {
				t.Fatalf("ready=%v notices=%v err=%v; want candidate-local %q", ready, notices, err, tc.want)
			}
		})
	}
}

func TestPlanReadyWorkspaceDependenciesLegacyAndArchivedProducer(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	producer := poolWorkspace(t, root, "producer", "", poolSlice("SLICE-001", "src/producer.go", ""))
	consumer := poolWorkspace(t, root, "local", "",
		poolSlice("SLICE-001", "src/legacy.go", "")+
			poolSlice("SLICE-002", "src/dependent.go", "Workspace dependencies: producer/SLICE-001")+
			poolSlice("SLICE-003", "src/missing.go", "Workspace dependencies: missing/SLICE-001"))
	t.Setenv("DEVRITES_WORKSPACE", consumer)
	ready, err := PlanReady(project, "local")
	if err != nil || !slices.Equal(poolKeys(ready), []string{"SLICE-001"}) {
		t.Fatalf("legacy local must run while producer unfinished/missing: ready=%v err=%v", ready, err)
	}
	writeFile(t, filepath.Join(producer, "state.md"), "| phase | done |\n- [x] SLICE-001 built — abc1234\n")
	ready, err = PlanReady(project, "local")
	if err != nil || !slices.Equal(poolKeys(ready), []string{"SLICE-001", "SLICE-002"}) {
		t.Fatalf("scoped consumer must resolve live producer: ready=%v err=%v", ready, err)
	}
	if os.Getenv("DEVRITES_WORKSPACE") != consumer {
		t.Fatal("producer resolution changed consumer override")
	}
	if err := os.MkdirAll(filepath.Join(root, "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(producer, filepath.Join(root, "archive", "producer")); err != nil {
		t.Fatal(err)
	}
	ready, err = PlanReady(project, "local")
	if err != nil || !slices.Equal(poolKeys(ready), []string{"SLICE-001", "SLICE-002"}) {
		t.Fatalf("archived durable producer must satisfy declared dependency: ready=%v err=%v", ready, err)
	}
	archived := filepath.Join(root, "archive", "producer", "state.md")
	if err := os.Remove(archived); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "state.md")
	writeFile(t, outside, "| phase | done |\n- [x] SLICE-001 built — abc1234\n")
	if err := os.Symlink(outside, archived); err != nil {
		t.Fatal(err)
	}
	ready, err = PlanReady(project, "local")
	if err != nil || !slices.Equal(poolKeys(ready), []string{"SLICE-001"}) {
		t.Fatalf("unsafe archived checkpoint must not satisfy dependency: ready=%v err=%v", ready, err)
	}
}

func TestWorkspaceReadyBoundPlanChangeRequiresNativeRevet(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	parent := poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
	child := poolWorkspace(t, root, "child", "parent", poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: none"))
	binding, err := gate.ReadinessBinding(root, "parent")
	if err != nil {
		t.Fatal(err)
	}
	bound := "Predecessor readiness: parent " + strings.TrimPrefix(binding, "Readiness inputs SHA-256: ") + "\n"
	eng, err := os.ReadFile(filepath.Join(child, "eng-review.md"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(child, "eng-review.md"), string(eng)+bound)
	ready, notices, err := WorkspaceReady(project, "parent", "child")
	if err != nil || len(ready) != 1 || len(notices) != 0 {
		t.Fatalf("initial ready=%v notices=%v err=%v", ready, notices, err)
	}
	writeFile(t, filepath.Join(parent, "plan.md"), "# Plan\n\nChanged interface.\n")
	ready, notices, err = WorkspaceReady(project, "parent", "child")
	if err != nil || len(ready) != 0 || !strings.Contains(strings.Join(notices, "\n"), "changed its planning inputs") || !strings.Contains(strings.Join(notices, "\n"), "/rite-vet") {
		t.Fatalf("source drift must route native re-Vet: ready=%v notices=%v err=%v", ready, notices, err)
	}
	after, err := os.ReadFile(filepath.Join(child, "eng-review.md"))
	if err != nil || string(after) != string(eng)+bound {
		t.Fatalf("admission auto-renewed binding: err=%v", err)
	}
}

func TestWorkspaceReadySequenceTraversalAndOverrideFailClosed(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	parent := poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
	middle := poolWorkspace(t, root, "middle", "parent", poolSlice("SLICE-001", "src/middle.go", ""))
	child := poolWorkspace(t, root, "child", "middle", poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: none"))
	if err := os.MkdirAll(filepath.Join(root, "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(middle, filepath.Join(root, "archive", "middle")); err != nil {
		t.Fatal(err)
	}
	ready, notices, err := WorkspaceReady(project, "parent", "child")
	if err != nil || len(ready) != 1 || len(notices) != 0 {
		t.Fatalf("archived link: ready=%v notices=%v err=%v", ready, notices, err)
	}
	t.Setenv("DEVRITES_WORKSPACE", parent)
	ready, notices, err = WorkspaceReady(project, "parent", "child")
	if err != nil || len(ready) != 0 || !strings.Contains(strings.Join(notices, "\n"), "unset DEVRITES_WORKSPACE") || os.Getenv("DEVRITES_WORKSPACE") != parent {
		t.Fatalf("override must stay scoped and unchanged: ready=%v notices=%v err=%v", ready, notices, err)
	}
	t.Setenv("DEVRITES_WORKSPACE", "")
	writeFile(t, filepath.Join(child, "state.md"), "| phase | build |\n| sequence_parent | middle |\n")
	writeFile(t, filepath.Join(root, "archive", "middle", "state.md"), "| phase | done |\n| sequence_parent | child |\n")
	ready, notices, err = WorkspaceReady(project, "parent", "child")
	if err != nil || len(ready) != 0 || !strings.Contains(strings.Join(notices, "\n"), "sequence_parent cycle") {
		t.Fatalf("cyclic authorization: ready=%v notices=%v err=%v", ready, notices, err)
	}
}

func TestWorkspaceReadyCLIInvalidFutureKeepsCurrentFrontier(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
	child := poolWorkspace(t, root, "child", "parent", poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: none"))
	writeFile(t, filepath.Join(child, "plan.md"), "# Plan\n\nUnvetted change.\n")
	saved := parallel.WorkspaceReady
	parallel.WorkspaceReady = WorkspaceReady
	t.Cleanup(func() { parallel.WorkspaceReady = saved })
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := parallel.Run("parallel", []string{"select", "--root", project, "--slug", "parent", "--cap", "2", "--from-plan", "--workspaces", "parent,child"}, strings.NewReader(""), stdout, stderr)
	if code != 0 || !strings.Contains(stdout.String(), "slices=SLICE-001") || !strings.Contains(stdout.String()+stderr.String(), "/rite-vet") {
		t.Fatalf("invalid future erased global frontier: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
}

func TestWorkspaceReadyVetEntryAndScopedDependencyNotices(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
	child := poolWorkspace(t, root, "child", "parent",
		poolSlice("SLICE-001", "src/approved.go", "Workspace dependencies: none")+
			poolSlice("SLICE-002", "src/legacy.go", ""))
	for _, tc := range []struct{ status, prefix string }{
		{"", ""},
		{"running", ""},
		{"running", "```text\n| status | ready |\n```\n"},
		{"ready", ""},
	} {
		writeFile(t, filepath.Join(child, "state.md"), tc.prefix+"| phase | vet |\n| status | "+tc.status+" |\n| sequence_parent | parent |\n")
		ready, notices, err := WorkspaceReady(project, "parent", "child")
		if tc.status != "ready" {
			if err != nil || len(ready) != 0 || !strings.Contains(strings.Join(notices, "\n"), "Vet is not ready") {
				t.Fatalf("unfinished Vet status %q prefix %q: ready=%v notices=%v err=%v", tc.status, tc.prefix, ready, notices, err)
			}
			continue
		}
		if err != nil || !slices.Equal(poolKeys(ready), []string{"child/SLICE-001"}) || !strings.Contains(strings.Join(notices, "\n"), "child/SLICE-002: missing Workspace dependencies") {
			t.Fatalf("Vet entry must admit approved slice only: ready=%v notices=%v err=%v", ready, notices, err)
		}
	}
}

func TestDispatchMixedLeaseWidthUsesLocalIDsAndGlobalCapacity(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
	poolWorkspace(t, root, "child", "parent",
		poolSlice("SLICE-001", "src/child-one.go", "Workspace dependencies: none")+
			poolSlice("SLICE-002", "src/child-two.go", "Workspace dependencies: none"))
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 3\n")
	path, err := parallel.LeasePath(project, "parent")
	if err != nil {
		t.Fatal(err)
	}
	lease := &parallel.Lease{
		BatchID: "b1", CreatedAt: "2026-10-20T00:00:00Z", BaseSHA: "abcdef1",
		N: 1, Capacity: 2, Status: parallel.StatusRunning, ControlPIDOrSession: "test",
		Slices: []parallel.LeaseSlice{{
			ID: "SLICE-001", Paths: []string{"src/parent.go"}, WorktreePath: "/tmp/parent-w1",
			Branch: "devrites/parallel/parent/b1/SLICE-001", WrightStatus: parallel.WrightPending,
		}},
	}
	if err := parallel.WriteLease(path, lease); err != nil {
		t.Fatal(err)
	}
	open := func(wave, id string) (int, string) {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := RunDispatch(root, []string{"child", "open", "--phase", "build", "--wave", wave,
			"--role", "slice-wright", "--slice", id, "--contract", wrightContractPath(t)}, stdout, stderr)
		return code, stdout.String() + stderr.String()
	}
	if code, out := open("narrow-wrong-lane", "SLICE-002"); code != 3 || !strings.Contains(out, "picks SLICE-001") {
		t.Fatalf("qualified child pick must still prevent local underdispatch: code=%d\n%s", code, out)
	}
	if code, out := open("fills-capacity", "SLICE-001"); code != 0 {
		t.Fatalf("one globally free slot must not demand two local writers: code=%d\n%s", code, out)
	}
	lease.Capacity = 3
	lease.N = 2
	lease.Slices = append(lease.Slices, parallel.LeaseSlice{
		Workspace: "child", ID: "SLICE-001", Paths: []string{"src/child-one.go"},
		WorktreePath: "/tmp/child-w1", Branch: "devrites/parallel/parent/b1/_workspaces/child/SLICE-001",
		WrightStatus: parallel.WrightPending,
	})
	if err := parallel.WriteLease(path, lease); err != nil {
		t.Fatal(err)
	}
	if code, out := open("repair-only", "SLICE-001"); code != 3 || !strings.Contains(out, "picks SLICE-002") {
		t.Fatalf("repair must not waive newly available child lane: code=%d\n%s", code, out)
	}
	if code, out := open("remaining-lane", "SLICE-002"); code != 0 {
		t.Fatalf("remaining local pick must not require foreign lane: code=%d\n%s", code, out)
	}
}

func TestWorkspaceReadySequenceTraversalBound(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
	poolWorkspace(t, root, "child", "link-0", poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: none"))
	for i := range maxSequenceChain {
		parent := fmt.Sprintf("link-%d", i+1)
		if i == maxSequenceChain-1 {
			parent = "parent"
		}
		writeFile(t, filepath.Join(root, "work", fmt.Sprintf("link-%d", i), "state.md"), "| phase | done |\n| sequence_parent | "+parent+" |\n")
	}
	ready, notices, err := WorkspaceReady(project, "parent", "child")
	if err != nil || len(ready) != 0 || !strings.Contains(strings.Join(notices, "\n"), "chain exceeds") {
		t.Fatalf("unbounded continuation traversal: ready=%v notices=%v err=%v", ready, notices, err)
	}
}

func TestWorkspaceReadySkipsDeferredBeforeAdmission(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		for _, paths := range []string{"src/required.go", "src/deferred.go", ""} {
			t.Run(fmt.Sprintf("foreign=%t/paths=%s", foreign, paths), func(t *testing.T) {
				project := t.TempDir()
				root := filepath.Join(project, ".devrites")
				target := "parent"
				if foreign {
					poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
					target = "child"
				}
				parent := ""
				if foreign {
					parent = "parent"
				}
				poolWorkspace(t, root, target, parent,
					poolSlice("SLICE-001", paths, "Satisfies: NoNe\nWorkspace dependencies: malformed")+
						poolSlice("SLICE-002", "src/required.go", "Workspace dependencies: none")+
						poolSlice("SLICE-003", "src/other.go", "```text\nSatisfies: none\nWriter allowlist: src/example.go\n```\nSatisfies: AC-001\nWorkspace dependencies: none"))
				ready, notices, err := WorkspaceReady(project, "parent", target)
				prefix := ""
				if foreign {
					prefix = "child/"
				}
				if err != nil || len(notices) != 0 || !slices.Equal(poolKeys(ready), []string{prefix + "SLICE-002", prefix + "SLICE-003"}) {
					t.Fatalf("ready=%v notices=%v err=%v", ready, notices, err)
				}
				picked, err := parallel.SelectGreedy(1, ready, project)
				if err != nil || !slices.Equal(poolKeys(picked), []string{prefix + "SLICE-002"}) {
					t.Fatalf("deferred slice starved required work: picks=%v err=%v", picked, err)
				}
			})
		}
	}
}
