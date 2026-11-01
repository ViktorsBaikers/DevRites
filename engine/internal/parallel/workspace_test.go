package parallel

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func liveSource(t *testing.T, repo, workspace string) {
	t.Helper()
	dir := filepath.Join(repo, ".devrites", "work", workspace)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state.md"), []byte("# State\n\nSchema: 4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func sliceJSON(t *testing.T, slices []SlicePaths) string {
	t.Helper()
	data, err := json.Marshal(slices)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func qualifiedSlice(t *testing.T, lease *Lease, key string) LeaseSlice {
	t.Helper()
	for _, sl := range lease.Slices {
		if SliceKey(sl.Workspace, sl.ID) == key {
			return sl
		}
	}
	t.Fatalf("missing slice %s in %+v", key, lease.Slices)
	return LeaseSlice{}
}

func TestMixedWorkspaceRollingCLI(t *testing.T) {
	repo, base := setupRepo(t)
	const child = "next-feature"
	liveSource(t, repo, child)
	local := []SlicePaths{{ID: "SLICE-001", Workspace: rollSlug, Paths: []string{"src/a.go"}}}
	code, _, errOut := runCLI(t, sliceJSON(t, local), "create", "--root", repo, "--slug", rollSlug, "--batch", rollBatch, "--base", base, "--json", "-", "--cap", "2")
	if code != ExitOK {
		t.Fatalf("create: code=%d stderr=%s", code, errOut)
	}
	lease := readLeaseT(t, repo)
	if lease.Capacity != 2 || len(lease.Slices) != 1 || lease.Slices[0].Workspace != "" {
		t.Fatalf("one isolated normalized writer: %+v", lease)
	}
	foreign := []SlicePaths{{ID: "SLICE-001", Workspace: child, Paths: []string{"src/b.go"}}}
	code, out, errOut := runCLI(t, sliceJSON(t, foreign), "select", "--root", repo, "--slug", rollSlug, "--cap", "10", "-")
	if code != ExitOK || !strings.Contains(out, "slices="+child+"/SLICE-001") || !strings.Contains(out, "running=1") {
		t.Fatalf("refill selection: code=%d out=%s stderr=%s", code, out, errOut)
	}
	code, _, errOut = runCLI(t, sliceJSON(t, foreign), "add", "--root", repo, "--slug", rollSlug, "--json", "-")
	if code != ExitOK {
		t.Fatalf("add while first writer pending: code=%d stderr=%s", code, errOut)
	}
	lease = readLeaseT(t, repo)
	owner := qualifiedSlice(t, lease, "SLICE-001")
	worker := qualifiedSlice(t, lease, child+"/SLICE-001")
	wantPath := filepath.Join(ScratchRoot(repo), rollBatch, "_workspaces", child, "SLICE-001")
	wantBranch := "devrites/parallel/" + rollSlug + "/" + rollBatch + "/_workspaces/" + child + "/SLICE-001"
	if worker.WorktreePath != wantPath || worker.Branch != wantBranch || worker.WorktreePath == owner.WorktreePath || lease.Capacity != 2 {
		t.Fatalf("foreign namespace or persisted cap: %+v", lease)
	}
	more := []SlicePaths{{ID: "SLICE-002", Workspace: child, Paths: []string{"src/c.go"}}}
	if _, err := Add(AddOpts{Root: repo, Slug: rollSlug, Slices: more}); err == nil {
		t.Fatal("cold add ignored persisted cap")
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("human work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tc := commitIn(t, worker.WorktreePath, "src/b.go", "Child")
	code, _, errOut = runCLI(t, "", "record-green", "--root", repo, "--slug", rollSlug, "--slice", child+"/SLICE-001", "--commit", tc)
	if code != ExitOK {
		t.Fatalf("foreign green: code=%d stderr=%s", code, errOut)
	}
	lease = readLeaseT(t, repo)
	if qualifiedSlice(t, lease, "SLICE-001").WrightStatus != WrightPending || qualifiedSlice(t, lease, child+"/SLICE-001").TransferCommit != tc {
		t.Fatalf("proof attributed to wrong source: %+v", lease.Slices)
	}
	code, _, errOut = runCLI(t, "", "integrate", "--root", repo, "--slug", rollSlug, "--slice", child+"/SLICE-001", "--apply-to-control")
	if code != ExitOK {
		t.Fatalf("foreign landing: code=%d stderr=%s", code, errOut)
	}
	if subject := gitOk(t, repo, "log", "-1", "--format=%s"); !strings.HasPrefix(subject, "WIP("+child+"):") {
		t.Fatalf("foreign WIP uses controller: %s", subject)
	}
	_, picks, running, err := SelectForSlug(repo, child, 10, []SlicePaths{{ID: "SLICE-001", Paths: []string{"src/b.go"}}, {ID: "SLICE-002", Paths: []string{"src/c.go"}}})
	if err != nil || running != 1 || len(picks) != 1 || SliceKey(picks[0].Workspace, picks[0].ID) != child+"/SLICE-002" {
		t.Fatalf("child-local query lost global identity/seed: picks=%+v running=%d err=%v", picks, running, err)
	}
	lease, err = Add(AddOpts{Root: repo, Slug: rollSlug, Slices: more})
	if err != nil {
		t.Fatal(err)
	}
	refill := qualifiedSlice(t, lease, child+"/SLICE-002")
	if refill.BaseSHA != gitOk(t, repo, "rev-parse", "HEAD") || lease.Capacity != 2 {
		t.Fatalf("refill base/cap: %+v", lease)
	}
	for _, item := range []struct{ key, file, marker string }{{"SLICE-001", "src/a.go", "Owner"}, {child + "/SLICE-002", "src/c.go", "Refill"}} {
		sl := qualifiedSlice(t, lease, item.key)
		commit := commitIn(t, sl.WorktreePath, item.file, item.marker)
		if _, err := RecordGreen(repo, rollSlug, item.key, commit); err != nil {
			t.Fatal(err)
		}
		if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: item.key, ApplyToControl: true}); err != nil {
			t.Fatal(err)
		}
	}
	if data, err := os.ReadFile(filepath.Join(repo, "README.md")); err != nil || string(data) != "human work\n" {
		t.Fatalf("human dirty work changed: %q %v", data, err)
	}
	if salvaged, err := Cleanup(repo, rollSlug, false); err != nil || len(salvaged) != 0 {
		t.Fatalf("complete cleanup: %+v %v", salvaged, err)
	}
	if _, err := os.Stat(filepath.Join(ScratchRoot(repo), rollBatch)); !os.IsNotExist(err) {
		t.Fatalf("owned empty namespace remains: %v", err)
	}
	for _, sl := range []LeaseSlice{owner, worker, refill} {
		if exists, err := branchExists(repo, sl.Branch); err != nil || exists {
			t.Fatalf("owned branch retained: %s %v", sl.Branch, err)
		}
	}
}

func TestRetainedControllerRootAliasesCLI(t *testing.T) {
	for _, tc := range []struct {
		name     string
		relative bool
		symlink  bool
	}{
		{name: "absolute"},
		{name: "relative", relative: true},
		{name: "symlink", symlink: true},
		{name: "relative-symlink", relative: true, symlink: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, base := setupRepo(t)
			liveSource(t, repo, "child")
			lease, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Capacity: 2, Slices: []SlicePaths{{ID: "SLICE-001", Paths: []string{"src/a.go"}}}})
			if err != nil {
				t.Fatal(err)
			}
			root := repo
			if tc.symlink {
				root = filepath.Join(t.TempDir(), "repo-alias")
				if err := os.Symlink(repo, root); err != nil {
					t.Fatal(err)
				}
			}
			if tc.relative {
				cwd, err := os.Getwd()
				if err != nil {
					t.Fatal(err)
				}
				if !strings.EqualFold(filepath.VolumeName(cwd), filepath.VolumeName(root)) {
					t.Skip("relative paths cannot span filesystem volumes")
				}
				root, err = filepath.Rel(cwd, root)
				if err != nil {
					t.Fatal(err)
				}
			}
			code, out, errOut := runCLI(t, "", "status", "--root", root, "--slug", "child")
			if code != ExitOK || !strings.Contains(out, "controller: "+rollSlug) || !strings.Contains(out, "status: running") {
				t.Fatalf("retained status: code=%d out=%s stderr=%s", code, out, errOut)
			}
			candidates := []SlicePaths{{ID: "SLICE-001", Paths: []string{"src/b.go"}}}
			_, picks, running, err := SelectForSlug(root, "child", 10, candidates)
			if err != nil || running != 1 || len(picks) != 1 || SliceKey(picks[0].Workspace, picks[0].ID) != "child/SLICE-001" {
				t.Fatalf("source-local selection: picks=%+v running=%d err=%v", picks, running, err)
			}
			code, out, errOut = runCLI(t, sliceJSON(t, []SlicePaths{{ID: "SLICE-001", Workspace: "child", Paths: []string{"src/b.go"}}}), "select", "--root", root, "--slug", rollSlug, "--cap", "10", "-")
			if code != ExitOK || !strings.Contains(out, "slices=child/SLICE-001") || !strings.Contains(out, "running=1") {
				t.Fatalf("controller selection: code=%d out=%s stderr=%s", code, out, errOut)
			}
			tc := commitIn(t, lease.Slices[0].WorktreePath, "src/a.go", "RelativeRoot")
			if _, err := RecordGreen(repo, rollSlug, "SLICE-001", tc); err != nil {
				t.Fatal(err)
			}
			code, out, errOut = runCLI(t, "", "integrate", "--root", root, "--slug", rollSlug, "--slice", "SLICE-001", "--apply-to-control")
			if code != ExitOK {
				t.Fatalf("relative integration: code=%d out=%s stderr=%s", code, out, errOut)
			}
			retained := readLeaseT(t, repo)
			if retained.Status != StatusComplete || retained.Slices[0].WrightStatus != WrightIntegrated || retained.ControlTip != gitOk(t, repo, "rev-parse", "HEAD") {
				t.Fatalf("integration did not land retained controller: %+v", retained)
			}
			code, out, errOut = runCLI(t, "", "status", "--root", root, "--slug", "child")
			if code != ExitOK || !strings.Contains(out, "controller: "+rollSlug) || !strings.Contains(out, "status: complete") {
				t.Fatalf("completed retained status: code=%d out=%s stderr=%s", code, out, errOut)
			}
			if _, err := Cleanup(repo, rollSlug, false); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMixedTransferCannotProveOtherWorkspace(t *testing.T) {
	repo, base := setupRepo(t)
	liveSource(t, repo, "child")
	lease, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Slices: []SlicePaths{{ID: "SLICE-001", Paths: []string{"src/a.go"}}, {ID: "SLICE-001", Workspace: "child", Paths: []string{"src/b.go"}}}})
	if err != nil {
		t.Fatal(err)
	}
	child := qualifiedSlice(t, lease, "child/SLICE-001")
	tc := commitIn(t, child.WorktreePath, "src/b.go", "Child")
	if _, err := RecordGreen(repo, rollSlug, "SLICE-001", tc); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: "SLICE-001", ApplyToControl: true}); err == nil {
		t.Fatal("foreign transfer passed controller exact-path proof")
	}
	if head := gitOk(t, repo, "rev-parse", "HEAD"); head != base {
		t.Fatalf("failed proof moved control: %s", head)
	}
	owner := qualifiedSlice(t, lease, "SLICE-001")
	ownTC := commitIn(t, owner.WorktreePath, "src/a.go", "Owner")
	if _, err := RecordGreen(repo, rollSlug, rollSlug+"/SLICE-001", ownTC); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: rollSlug + "/SLICE-001", ApplyToControl: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordGreen(repo, rollSlug, "child/SLICE-001", tc); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: "child/SLICE-001", ApplyToControl: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := Cleanup(repo, rollSlug, false); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteLeaseRefillsOnlyUnchangedControl(t *testing.T) {
	for _, external := range []bool{false, true} {
		t.Run(fmt.Sprint(external), func(t *testing.T) {
			repo, base := setupRepo(t)
			lease, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Capacity: 1, Slices: []SlicePaths{{ID: "first", Paths: []string{"src/a.go"}}}})
			if err != nil {
				t.Fatal(err)
			}
			tc := commitIn(t, lease.Slices[0].WorktreePath, "src/a.go", "First")
			if _, err := RecordGreen(repo, rollSlug, "first", tc); err != nil {
				t.Fatal(err)
			}
			tip, lease, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: "first", ApplyToControl: true})
			if err != nil || lease.Status != StatusComplete {
				t.Fatalf("drain: %+v %v", lease, err)
			}
			if external {
				gitOk(t, repo, "commit", "--allow-empty", "-m", "human external commit")
			}
			head := gitOk(t, repo, "rev-parse", "HEAD")
			lease, err = Add(AddOpts{Root: repo, Slug: rollSlug, Slices: []SlicePaths{{ID: "second", Paths: []string{"src/b.go"}}}})
			if external {
				if err == nil || gitOk(t, repo, "rev-parse", "HEAD") != head || readLeaseT(t, repo).Status != StatusComplete {
					t.Fatalf("external control was accepted or changed: %v", err)
				}
				return
			}
			if err != nil || lease.Status != StatusRunning || lease.Capacity != 1 || lease.Slices[1].BaseSHA != tip {
				t.Fatalf("refill drained lease: %+v %v", lease, err)
			}
			if _, err := Cleanup(repo, rollSlug, true); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRepositoryAdmissionSerializesControllers(t *testing.T) {
	repo, base := setupRepo(t)
	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make(chan string, 2)
	for i, slug := range []string{"owner-a", "owner-b"} {
		wg.Add(1)
		go func(index int, owner string) {
			defer wg.Done()
			<-start
			_, err := Create(CreateOpts{Root: repo, Slug: owner, BatchID: "batch", BaseSHA: base, Capacity: 1, Slices: []SlicePaths{{ID: "SLICE-001", Paths: []string{fmt.Sprintf("src/%c.go", 'a'+index)}}}})
			if err == nil {
				results <- owner
			}
		}(i, slug)
	}
	close(start)
	wg.Wait()
	close(results)
	var owners []string
	for owner := range results {
		owners = append(owners, owner)
	}
	if len(owners) != 1 {
		t.Fatalf("competing controllers admitted: %v", owners)
	}
	owner := owners[0]
	otherTree := filepath.Join(t.TempDir(), "alternate")
	gitOk(t, repo, "worktree", "add", "--detach", otherTree, base)
	if _, err := Create(CreateOpts{Root: otherTree, Slug: "third", BatchID: "other", BaseSHA: base, Slices: []SlicePaths{{ID: "next", Paths: []string{"src/c.go"}}}}); err == nil {
		t.Fatal("linked control worktree bypassed repository controller")
	}
	report, err := StatusReport(otherTree, "third")
	if err != nil || !strings.Contains(report, "controller: "+owner) || !strings.Contains(report, filepath.Join(".devrites", "work", owner, "parallel-lease.md")) {
		t.Fatalf("cold status lost controller: %s %v", report, err)
	}
	payload := validTestLease(1)
	payload.BaseSHA = base
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if code, _, _ := runCLI(t, string(raw), "lease-write", "--root", repo, "--slug", "other", "--json", "-"); code != ExitBlocked {
		t.Fatalf("lease-write bypassed repository controller: %d", code)
	}
	if _, err := Cleanup(repo, owner, true); err != nil {
		t.Fatal(err)
	}
}

func TestMixedCapacityUpdatesAreDurable(t *testing.T) {
	repo, base := setupRepo(t)
	liveSource(t, repo, "child")
	_, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Capacity: 2, Slices: []SlicePaths{{ID: "same", Paths: []string{"src/a.go"}}}})
	if err != nil {
		t.Fatal(err)
	}
	newSlice := []SlicePaths{{ID: "same", Workspace: "child", Paths: []string{"src/b.go"}}}
	if _, err := Add(AddOpts{Root: repo, Slug: rollSlug, Capacity: 1, Slices: newSlice}); err == nil {
		t.Fatal("lower cap discarded active slot")
	}
	if lease := readLeaseT(t, repo); lease.Capacity != 2 || len(lease.Slices) != 1 {
		t.Fatalf("refused lower cap mutated lease: %+v", lease)
	}
	lease, err := Add(AddOpts{Root: repo, Slug: rollSlug, Capacity: 3, Slices: newSlice})
	if err != nil || lease.Capacity != 3 {
		t.Fatalf("raise cap: %+v %v", lease, err)
	}
	if _, err := Add(AddOpts{Root: repo, Slug: rollSlug, Capacity: 11, Slices: []SlicePaths{{ID: "third", Paths: []string{"src/c.go"}}}}); err == nil {
		t.Fatal("capacity exceeded hard ten ceiling")
	}
	lease, err = Add(AddOpts{Root: repo, Slug: rollSlug, Slices: []SlicePaths{{ID: "third", Workspace: "child", Paths: []string{"src/c.go"}}}})
	if err != nil || lease.Capacity != 3 || len(activeSlices(lease)) != 3 {
		t.Fatalf("inherit raised cap: %+v %v", lease, err)
	}
	for _, slug := range []string{rollSlug, "child", ""} {
		_, picks, running, err := SelectForSlug(repo, slug, 10, []SlicePaths{{ID: "fourth", Paths: []string{"src/d.go"}}})
		if err != nil || len(picks) != 0 || running != 3 {
			t.Fatalf("selection %q overbooked: picks=%+v running=%d err=%v", slug, picks, running, err)
		}
	}
	if _, err := Cleanup(repo, rollSlug, true); err != nil {
		t.Fatal(err)
	}
}

func TestMixedIdentityAndOneWriterValidation(t *testing.T) {
	cases := []struct {
		name   string
		slices []SlicePaths
	}{
		{"workspace traversal", []SlicePaths{{ID: "one", Workspace: "../outside", Paths: []string{"src/a.go"}}}},
		{"reserved namespace", []SlicePaths{{ID: "_workspaces", Paths: []string{"src/a.go"}}}},
		{"unsafe path", []SlicePaths{{ID: "one", Paths: []string{".devrites/work/child/state.md"}}}},
		{"controller alias duplicate", []SlicePaths{{ID: "one", Paths: []string{"src/a.go"}}, {ID: "one", Workspace: rollSlug, Paths: []string{"src/b.go"}}}},
		{"staging collision", []SlicePaths{{ID: "integrate", Paths: []string{"src/a.go"}}}},
		{"missing source", []SlicePaths{{ID: "one", Workspace: "missing", Paths: []string{"src/a.go"}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, base := setupRepo(t)
			if _, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Slices: tc.slices}); err == nil {
				t.Fatal("unsafe candidate admitted")
			}
			if gitOk(t, repo, "rev-parse", "HEAD") != base {
				t.Fatal("refusal changed control")
			}
		})
	}
	picks, err := SelectGreedy(2, []SlicePaths{{ID: "same", Paths: []string{"src/a.go"}}, {ID: "same", Workspace: "child", Paths: []string{"src/b.go"}}}, "")
	if err != nil || len(picks) != 2 || picks[1].Workspace != "child" {
		t.Fatalf("source lost during normalization: %+v %v", picks, err)
	}
}

func TestPooledSelectKeepsReadyWorkAndDiagnostics(t *testing.T) {
	saved := WorkspaceReady
	t.Cleanup(func() { WorkspaceReady = saved })
	var calls []string
	WorkspaceReady = func(root, controller, target string) ([]SlicePaths, []string, error) {
		calls = append(calls, target)
		switch target {
		case "owner":
			return []SlicePaths{{ID: "SLICE-001", Paths: []string{"src/a.go"}}}, nil, nil
		case "missing":
			return nil, []string{"native planning required"}, errors.New("workspace absent")
		case "stale":
			return nil, []string{"human gate still blocked"}, errors.New("native re-Vet required")
		case "wrong":
			return []SlicePaths{{ID: "SLICE-001", Workspace: controller, Paths: []string{"src/c.go"}}}, nil, nil
		case "child":
			return []SlicePaths{{ID: "SLICE-001", Workspace: target, Paths: []string{"src/b.go"}}}, []string{"native ready"}, nil
		default:
			return nil, nil, errors.New("unexpected target")
		}
	}
	root := t.TempDir()
	code, out, errOut := runCLI(t, "", "select", "--root", root, "--slug", "owner", "--cap", "2", "--from-plan", "--workspaces", "missing,wrong,child,owner,stale")
	if code != ExitOK || !strings.Contains(out, "slices=SLICE-001,child/SLICE-001") {
		t.Fatalf("unavailable candidate erased ready pool: code=%d out=%s stderr=%s", code, out, errOut)
	}
	if strings.Join(calls, ",") != "owner,missing,wrong,child,stale" {
		t.Fatalf("pool order: %v", calls)
	}
	for _, reason := range []string{"workspace absent", "native planning required", "human gate still blocked", "native re-Vet required", "native ready", "workspace wrong"} {
		if !strings.Contains(errOut, reason) {
			t.Fatalf("lost workspace diagnostic %q: %s", reason, errOut)
		}
	}
	for _, args := range [][]string{
		{"--workspaces", "owner,child"},
		{"--from-plan", "--workspaces", "child,child"},
		{"--from-plan", "--workspaces", ""},
		{"--from-plan", "--workspaces", "../bad"},
		{"--from-plan", "--workspaces", "child", "-"},
		{"--from-plan", "--workspaces", "child", "--workspaces", "owner"},
	} {
		baseArgs := []string{"select", "--root", root, "--slug", "owner", "--cap", "2"}
		if code, _, _ := runCLI(t, "[]", append(baseArgs, args...)...); code != ExitUsage {
			t.Fatalf("invalid pool arguments admitted: %v code=%d", args, code)
		}
	}
}

func TestWorkspaceLeasedUsesExplicitControlRoot(t *testing.T) {
	parent := t.TempDir()
	control := filepath.Join(parent, "custom-control")
	lease := validTestLease(1)
	lease.Slices[0].Workspace = "child"
	lease.Status = StatusComplete
	lease.Slices[0].WrightStatus = WrightIntegrated
	path := filepath.Join(control, "work", "owner", "parallel-lease.md")
	if err := WriteLease(path, lease); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_WORKSPACE", filepath.Join(t.TempDir(), "unrelated"))
	for workspace, want := range map[string]bool{"owner": true, "child": true, "other": false} {
		got, err := WorkspaceLeased(control, workspace)
		if err != nil || got != want {
			t.Fatalf("leased %s=%v want %v err=%v", workspace, got, want, err)
		}
	}
	if _, err := os.Stat(filepath.Join(control, "parallel.lock")); !os.IsNotExist(err) {
		t.Fatalf("read-only lease query created lock: %v", err)
	}
	if err := WithControlLock(control, func() error {
		_, err := os.Stat(filepath.Join(control, "parallel.lock"))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(parent, ".devrites")); !os.IsNotExist(err) {
		t.Fatalf("custom root redirected or wrote outside control: %v", err)
	}
	lease.Status = StatusAborted
	lease.Slices[0].WrightStatus = WrightPending
	if err := WriteLease(path, lease); err != nil {
		t.Fatal(err)
	}
	if got, err := WorkspaceLeased(control, "child"); err != nil || !got {
		t.Fatalf("aborted source lost retention: %v %v", got, err)
	}
	if err := ClearLease(path); err != nil {
		t.Fatal(err)
	}
	if got, err := WorkspaceLeased(control, "child"); err != nil || got {
		t.Fatalf("removed lease retained source: %v %v", got, err)
	}
}

func TestForeignNamespacePreservesLegacyWorkerAndUnknownContent(t *testing.T) {
	repo, base := setupRepo(t)
	liveSource(t, repo, "child")
	lease, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Slices: []SlicePaths{{ID: "child", Paths: []string{"src/a.go"}}, {ID: "SLICE-001", Workspace: "child", Paths: []string{"src/b.go"}}}})
	if err != nil {
		t.Fatal(err)
	}
	local := qualifiedSlice(t, lease, "child")
	foreign := qualifiedSlice(t, lease, "child/SLICE-001")
	if local.WorktreePath == foreign.WorktreePath || local.Branch == foreign.Branch {
		t.Fatal("foreign namespace collided with allowed legacy slice id")
	}
	for _, sl := range lease.Slices {
		if err := os.WriteFile(filepath.Join(sl.WorktreePath, sl.Paths[0]), []byte("retained writer work\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	unknown := filepath.Join(ScratchRoot(repo), rollBatch, "_workspaces", "unknown", "precious")
	if err := os.MkdirAll(filepath.Dir(unknown), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unknown, []byte("human data"), 0o644); err != nil {
		t.Fatal(err)
	}
	salvaged, err := Cleanup(repo, rollSlug, true)
	if err != nil || len(salvaged) != 2 {
		t.Fatalf("mixed salvage: %+v %v", salvaged, err)
	}
	for _, sl := range []LeaseSlice{local, foreign} {
		key := SliceKey(sl.Workspace, sl.ID)
		found := false
		for _, result := range salvaged {
			if result.SliceID == key && result.Branch == sl.Branch && result.Commit != base {
				found = true
			}
		}
		if !found {
			t.Fatalf("salvage identity lost for %s: %+v", key, salvaged)
		}
		if _, err := os.Stat(sl.WorktreePath); !os.IsNotExist(err) {
			t.Fatalf("salvaged worker remains: %s %v", key, err)
		}
		if data := gitOk(t, repo, "show", sl.Branch+":"+sl.Paths[0]); data != "retained writer work" {
			t.Fatalf("salvage lost source content: %s", data)
		}
	}
	if data, err := os.ReadFile(unknown); err != nil || string(data) != "human data" {
		t.Fatalf("unknown namespace content removed: %q %v", data, err)
	}
}

func TestForeignFailedSalvageKeepsOnlyCopy(t *testing.T) {
	repo, base := setupRepo(t)
	liveSource(t, repo, "child")
	lease, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Slices: []SlicePaths{{ID: "SLICE-001", Paths: []string{"src/a.go"}}, {ID: "SLICE-001", Workspace: "child", Paths: []string{"src/b.go"}}}})
	if err != nil {
		t.Fatal(err)
	}
	foreign := qualifiedSlice(t, lease, "child/SLICE-001")
	dirtyFile := filepath.Join(foreign.WorktreePath, "src/b.go")
	if err := os.WriteFile(dirtyFile, []byte("only remaining copy"), 0o644); err != nil {
		t.Fatal(err)
	}
	swapSeams(t)
	realSalvage := salvage
	salvage = func(sl LeaseSlice, batch string) (string, error) {
		if sl.Workspace == "child" {
			return "", errors.New("salvage unavailable")
		}
		return realSalvage(sl, batch)
	}
	results, err := Cleanup(repo, rollSlug, true)
	if err != nil || len(results) != 1 || results[0].SliceID != "child/SLICE-001" || results[0].Worktree != foreign.WorktreePath {
		t.Fatalf("failed salvage lost qualified recovery path: %+v %v", results, err)
	}
	if data, err := os.ReadFile(dirtyFile); err != nil || string(data) != "only remaining copy" {
		t.Fatalf("failed foreign salvage deleted only copy: %q %v", data, err)
	}
}

func TestForeignOnlyLeaseNeverMatchesBareSlice(t *testing.T) {
	repo, base := setupRepo(t)
	liveSource(t, repo, "child")
	lease, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Capacity: 1, Slices: []SlicePaths{{ID: "SLICE-001", Workspace: "child", Paths: []string{"src/b.go"}}}})
	if err != nil {
		t.Fatal(err)
	}
	tc := commitIn(t, lease.Slices[0].WorktreePath, "src/b.go", "Foreign")
	if _, err := RecordGreen(repo, rollSlug, "SLICE-001", tc); err == nil {
		t.Fatal("bare controller id matched foreign-only slice")
	}
	if _, err := RecordGreen(repo, rollSlug, "child/SLICE-001", tc); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: "SLICE-001", ApplyToControl: true}); err == nil {
		t.Fatal("bare integration matched foreign-only slice")
	}
	if gitOk(t, repo, "rev-parse", "HEAD") != base {
		t.Fatal("bare foreign lookup moved control")
	}
	if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: "child/SLICE-001", ApplyToControl: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := Cleanup(repo, rollSlug, false); err != nil {
		t.Fatal(err)
	}
}

func TestArchivedSourceCannotEnterRetainedLease(t *testing.T) {
	repo, base := setupRepo(t)
	liveSource(t, repo, "child")
	lease, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Slices: []SlicePaths{{ID: "SLICE-001", Paths: []string{"src/a.go"}}}})
	if err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(repo, ".devrites", "archive")
	if err := os.MkdirAll(archive, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(repo, ".devrites", "work", "child"), filepath.Join(archive, "child")); err != nil {
		t.Fatal(err)
	}
	foreign := SlicePaths{ID: "SLICE-001", Workspace: "child", Paths: []string{"src/b.go"}}
	if _, err := Add(AddOpts{Root: repo, Slug: rollSlug, Slices: []SlicePaths{foreign}}); err == nil {
		t.Fatal("add admitted archived source after closeout won control lock")
	}
	lease.Slices = append(lease.Slices, LeaseSlice{ID: foreign.ID, Workspace: foreign.Workspace, Paths: foreign.Paths, WrightStatus: WrightPending})
	lease.N++
	payload, err := json.Marshal(lease)
	if err != nil {
		t.Fatal(err)
	}
	if code, _, _ := runCLI(t, string(payload), "lease-write", "--root", repo, "--slug", rollSlug, "--json", "-"); code != ExitBlocked {
		t.Fatalf("lease-write admitted archived source: %d", code)
	}
	if got := readLeaseT(t, repo); len(got.Slices) != 1 || got.ControlTip != "" || gitOk(t, repo, "rev-parse", "HEAD") != base {
		t.Fatalf("refused source mutated lease/control: %+v", got)
	}
	if _, err := Cleanup(repo, rollSlug, true); err != nil {
		t.Fatal(err)
	}
}

func TestMixedLeaseCapacitySchemaAndLegacyDefault(t *testing.T) {
	lease := validTestLease(10)
	for i := range lease.Slices {
		lease.Slices[i].ID = "SLICE-001"
		lease.Slices[i].Workspace = fmt.Sprintf("workspace-%d", i)
	}
	path := filepath.Join(t.TempDir(), "lease.md")
	if err := WriteLease(path, lease); err != nil {
		t.Fatal(err)
	}
	got, err := ReadLease(path)
	if err != nil || len(got.Slices) != 10 || got.Capacity != 0 || got.Slices[9].Workspace != "workspace-9" {
		t.Fatalf("mixed identity or legacy default lost: %+v %v", got, err)
	}
	lease.Capacity = 9
	if err := ValidateLease(lease); err == nil {
		t.Fatal("mixed active count bypassed explicit cap")
	}
	lease.Slices[0].WrightStatus = WrightIntegrated
	if err := ValidateLease(lease); err != nil {
		t.Fatalf("landed source still consumes cap: %v", err)
	}
	lease.Capacity = 11
	if err := ValidateLease(lease); err == nil {
		t.Fatal("schema accepted capacity over hard maximum")
	}
}

func TestLeaseWriteInheritsCapacityButNeverRebasesRetainedTip(t *testing.T) {
	repo, base := setupRepo(t)
	lease, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Capacity: 1, Slices: []SlicePaths{{ID: "first", Paths: []string{"src/a.go"}}}})
	if err != nil {
		t.Fatal(err)
	}
	lease.Capacity = 0
	payload, err := json.Marshal(lease)
	if err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := runCLI(t, string(payload), "lease-write", "--root", repo, "--slug", rollSlug, "--json", "-"); code != ExitOK {
		t.Fatalf("same-controller old field set refused: code=%d stderr=%s", code, errOut)
	}
	if got := readLeaseT(t, repo); got.Capacity != 1 {
		t.Fatalf("lease-write reset durable cap: %+v", got)
	}
	gitOk(t, repo, "commit", "--allow-empty", "-m", "external human commit")
	head := gitOk(t, repo, "rev-parse", "HEAD")
	lease.ControlTip = head
	payload, err = json.Marshal(lease)
	if err != nil {
		t.Fatal(err)
	}
	if code, _, _ := runCLI(t, string(payload), "lease-write", "--root", repo, "--slug", rollSlug, "--json", "-"); code != ExitBlocked {
		t.Fatalf("lease-write silently accepted external tip: %d", code)
	}
	if got := readLeaseT(t, repo); got.ControlTip != "" || gitOk(t, repo, "rev-parse", "HEAD") != head {
		t.Fatalf("lease-write changed expected or actual tip: %+v", got)
	}
	if _, err := Cleanup(repo, rollSlug, true); err != nil {
		t.Fatal(err)
	}
}

func TestForeignRefillRollsBackNewWorkersOnly(t *testing.T) {
	repo, base := setupRepo(t)
	liveSource(t, repo, "child")
	lease, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Capacity: 3, Slices: []SlicePaths{{ID: "first", Paths: []string{"src/a.go"}}}})
	if err != nil {
		t.Fatal(err)
	}
	blockedBranch := "devrites/parallel/" + rollSlug + "/" + rollBatch + "/_workspaces/child/blocked"
	gitOk(t, repo, "branch", blockedBranch, base)
	added := []SlicePaths{{ID: "new", Workspace: "child", Paths: []string{"src/b.go"}}, {ID: "blocked", Workspace: "child", Paths: []string{"src/c.go"}}}
	if _, err := Add(AddOpts{Root: repo, Slug: rollSlug, Slices: added}); err == nil {
		t.Fatal("pre-existing foreign branch was overwritten")
	}
	if got := readLeaseT(t, repo); len(got.Slices) != 1 || got.Capacity != 3 {
		t.Fatalf("failed refill changed lease: %+v", got)
	}
	if _, err := os.Stat(lease.Slices[0].WorktreePath); err != nil {
		t.Fatalf("rollback removed existing writer: %v", err)
	}
	newBranch := "devrites/parallel/" + rollSlug + "/" + rollBatch + "/_workspaces/child/new"
	if exists, err := branchExists(repo, newBranch); err != nil || exists {
		t.Fatalf("rolled back foreign branch remains: %v %v", exists, err)
	}
	if _, err := os.Stat(filepath.Join(ScratchRoot(repo), rollBatch, "_workspaces", "child", "new")); !os.IsNotExist(err) {
		t.Fatalf("rolled back foreign worktree remains: %v", err)
	}
	if gitOk(t, repo, "rev-parse", blockedBranch) != base || gitOk(t, repo, "rev-parse", "HEAD") != base {
		t.Fatal("rollback changed pre-existing branch or control")
	}
	if _, err := Cleanup(repo, rollSlug, true); err != nil {
		t.Fatal(err)
	}
}

func TestForeignCleanupPreservesUserBranchCheckout(t *testing.T) {
	repo, base := setupRepo(t)
	liveSource(t, repo, "child")
	lease, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Slices: []SlicePaths{{ID: "SLICE-001", Workspace: "child", Paths: []string{"src/b.go"}}}})
	if err != nil {
		t.Fatal(err)
	}
	worker := lease.Slices[0]
	gitOk(t, worker.WorktreePath, "checkout", "-b", "user/topic")
	path := filepath.Join(worker.WorktreePath, "src/b.go")
	if err := os.WriteFile(path, []byte("human branch work"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Cleanup(repo, rollSlug, true); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "human branch work" {
		t.Fatalf("cleanup changed switched user worktree: %q %v", data, err)
	}
	if gitOk(t, worker.WorktreePath, "rev-parse", "HEAD") != base || gitOk(t, worker.WorktreePath, "symbolic-ref", "--short", "HEAD") != "user/topic" {
		t.Fatal("cleanup salvaged or switched user branch")
	}
}

func TestForeignSourceStateRejectsSymlink(t *testing.T) {
	repo, base := setupRepo(t)
	liveSource(t, repo, "child")
	statePath := filepath.Join(repo, ".devrites", "work", "child", "state.md")
	outside := filepath.Join(t.TempDir(), "human-state")
	if err := os.WriteFile(outside, []byte("not engine-owned"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(statePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, statePath); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Slices: []SlicePaths{{ID: "SLICE-001", Workspace: "child", Paths: []string{"src/b.go"}}}}); err == nil {
		t.Fatal("symlinked source state admitted")
	}
	if data, err := os.ReadFile(outside); err != nil || string(data) != "not engine-owned" {
		t.Fatalf("foreign source validation touched user state: %q %v", data, err)
	}
}

func TestCapacityOnlyAddReconcilesWithoutDrainingOrReopening(t *testing.T) {
	repo, base := setupRepo(t)
	liveSource(t, repo, "child")
	_, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Capacity: 1, Slices: []SlicePaths{{ID: "SLICE-001", Paths: []string{"src/a.go"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Add(AddOpts{Root: repo, Slug: rollSlug}); err == nil {
		t.Fatal("empty add without explicit capacity admitted")
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("unrelated human work"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := runCLI(t, "[]", "add", "--root", repo, "--slug", rollSlug, "--cap", "2", "--json", "-"); code != ExitOK {
		t.Fatalf("raise cap without draining: code=%d stderr=%s", code, errOut)
	}
	if got := readLeaseT(t, repo); got.Capacity != 2 || len(got.Slices) != 1 || got.Status != StatusRunning || got.ControlTip != "" {
		t.Fatalf("capacity-only raise changed writers/status/tip: %+v", got)
	}
	candidate := []SlicePaths{{ID: "SLICE-001", Workspace: "child", Paths: []string{"src/b.go"}}}
	_, picks, running, err := SelectForSlug(repo, rollSlug, 10, candidate)
	if err != nil || len(picks) != 1 || running != 1 {
		t.Fatalf("reconciled cap not visible to pool: %+v %d %v", picks, running, err)
	}
	lease, err := Add(AddOpts{Root: repo, Slug: rollSlug, Slices: picks})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Add(AddOpts{Root: repo, Slug: rollSlug, Capacity: 1}); err == nil {
		t.Fatal("capacity-only lower cancelled active writer")
	}
	if got := readLeaseT(t, repo); got.Capacity != 2 || len(got.Slices) != 2 {
		t.Fatalf("refused lower changed durable lease: %+v", got)
	}
	for _, item := range []struct{ key, file, marker string }{{"SLICE-001", "src/a.go", "Owner"}, {"child/SLICE-001", "src/b.go", "Child"}} {
		worker := qualifiedSlice(t, lease, item.key)
		tc := commitIn(t, worker.WorktreePath, item.file, item.marker)
		if _, err := RecordGreen(repo, rollSlug, item.key, tc); err != nil {
			t.Fatal(err)
		}
		if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: item.key, ApplyToControl: true}); err != nil {
			t.Fatal(err)
		}
		if item.key == "SLICE-001" {
			got, err := Add(AddOpts{Root: repo, Slug: rollSlug, Capacity: 1})
			if err != nil || got.Capacity != 1 || got.Status != StatusRunning || len(got.Slices) != 2 {
				t.Fatalf("capacity-only lower after one landing: %+v %v", got, err)
			}
		}
	}
	tip := gitOk(t, repo, "rev-parse", "HEAD")
	lease, err = Add(AddOpts{Root: repo, Slug: rollSlug, Capacity: 3})
	if err != nil || lease.Capacity != 3 || lease.Status != StatusComplete || lease.ControlTip != tip || len(lease.Slices) != 2 {
		t.Fatalf("capacity-only update reopened complete lease: %+v %v", lease, err)
	}
	gitOk(t, repo, "commit", "--allow-empty", "-m", "human external commit")
	head := gitOk(t, repo, "rev-parse", "HEAD")
	if _, err := Add(AddOpts{Root: repo, Slug: rollSlug, Capacity: 2}); err == nil {
		t.Fatal("capacity-only update accepted external control movement")
	}
	if got := readLeaseT(t, repo); got.Capacity != 3 || got.ControlTip != tip || gitOk(t, repo, "rev-parse", "HEAD") != head {
		t.Fatalf("refused capacity-only update changed state/control: %+v", got)
	}
	if data, err := os.ReadFile(filepath.Join(repo, "README.md")); err != nil || string(data) != "unrelated human work" {
		t.Fatalf("capacity-only update touched human work: %q %v", data, err)
	}
	if _, err := Cleanup(repo, rollSlug, false); err != nil {
		t.Fatal(err)
	}
}

func TestExplicitControlRootNeverWritesThroughSymlink(t *testing.T) {
	outside := t.TempDir()
	link := filepath.Join(t.TempDir(), "control")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if err := WithControlLock(link, func() error {
		t.Fatal("unsafe control lock callback executed")
		return nil
	}); err == nil {
		t.Fatal("symlink control root admitted")
	}
	if _, err := WorkspaceLeased(link, "owner"); err == nil {
		t.Fatal("lease scan followed unsafe control root")
	}
	if _, err := os.Stat(filepath.Join(outside, "parallel.lock")); !os.IsNotExist(err) {
		t.Fatalf("lock escaped explicit control root: %v", err)
	}
}

func TestBrokenRepositoryCannotFallBackToUncoordinatedControl(t *testing.T) {
	repo, _ := setupRepo(t)
	control := filepath.Join(repo, ".devrites")
	if err := os.Rename(filepath.Join(repo, ".git", "HEAD"), filepath.Join(repo, ".git", "saved-HEAD")); err != nil {
		t.Fatal(err)
	}
	if err := WithControlLock(control, func() error {
		t.Fatal("broken Git discovery executed an uncoordinated callback")
		return nil
	}); err == nil {
		t.Fatal("broken repository selected fixture-only fallback lock")
	}
	if _, err := WorkspaceLeased(control, "owner"); err == nil {
		t.Fatal("broken repository was reported as unleased")
	}
	if _, err := os.Stat(filepath.Join(control, "parallel.lock")); !os.IsNotExist(err) {
		t.Fatalf("broken repository created fallback lock: %v", err)
	}
}

func TestRepositoryDiscoveryPreservesNewlineWorktreePaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not permit newline directory names")
	}
	repo, base := setupRepo(t)
	other := filepath.Join(t.TempDir(), "linked-new\nline")
	gitOk(t, repo, "worktree", "add", "--detach", other, base)
	lease, err := Create(CreateOpts{Root: other, Slug: "foreign", BatchID: "b1", BaseSHA: base, Capacity: 1,
		Slices: []SlicePaths{{ID: "SLICE-001", Paths: []string{"src/a.go"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Create(CreateOpts{Root: repo, Slug: "owner", BatchID: "b2", BaseSHA: base, Capacity: 1,
		Slices: []SlicePaths{{ID: "SLICE-002", Paths: []string{"src/b.go"}}}}); err == nil {
		t.Fatal("newline-bearing worktree hid retained repository controller")
	}
	_, picks, running, err := SelectForSlug(repo, "owner", 1, []SlicePaths{{ID: "SLICE-002", Paths: []string{"src/b.go"}}})
	if err != nil || running != 1 || len(picks) != 0 {
		t.Fatalf("newline-bearing controller lost global capacity: picks=%v running=%d err=%v", picks, running, err)
	}
	salvaged, err := Cleanup(other, "foreign", true)
	if err != nil || len(salvaged) != 0 {
		t.Fatalf("registered newline-bearing worker could not clean up: %v %v", salvaged, err)
	}
	if _, err := os.Stat(lease.Slices[0].WorktreePath); !os.IsNotExist(err) {
		t.Fatalf("owned newline-bearing worker remains: %v", err)
	}
}
