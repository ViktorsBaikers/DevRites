package parallel

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitOk(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_AUTHOR_NAME=DevRites Test",
		"GIT_AUTHOR_EMAIL=devrites@example.invalid",
		"GIT_COMMITTER_NAME=DevRites Test",
		"GIT_COMMITTER_EMAIL=devrites@example.invalid",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func setupRepo(t *testing.T) (repo, base string) {
	t.Helper()
	repo = t.TempDir()
	gitOk(t, repo, "init")
	gitOk(t, repo, "checkout", "-b", "main")
	if err := os.MkdirAll(filepath.Join(repo, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "src", "a.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "src", "b.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOk(t, repo, "add", ".")
	gitOk(t, repo, "commit", "-m", "base")
	base = gitOk(t, repo, "rev-parse", "HEAD")
	return repo, base
}

func TestLeaseRoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "parallel-lease.md")
	lease := &Lease{
		BatchID:             "batch1",
		CreatedAt:           "2026-08-23T00:00:00Z",
		BaseSHA:             "abcdef1",
		N:                   2,
		Status:              StatusRunning,
		ControlPIDOrSession: "test",
		Slices: []LeaseSlice{
			{ID: "slice-a", Paths: []string{"src/a.go"}, WorktreePath: "/tmp/a", Branch: "devrites/parallel/x/batch1/slice-a", WrightStatus: WrightPending},
			{ID: "slice-b", Paths: []string{"src/b.go"}, WorktreePath: "/tmp/b", Branch: "devrites/parallel/x/batch1/slice-b", WrightStatus: WrightPending},
		},
	}
	if err := WriteLease(path, lease); err != nil {
		t.Fatal(err)
	}
	got, err := ReadLease(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.BatchID != "batch1" || got.Status != StatusRunning || len(got.Slices) != 2 {
		t.Fatalf("unexpected lease: %#v", got)
	}
}

func TestWriteLeasePreservesFixedTemp(t *testing.T) {
	for _, directory := range []bool{false, true} {
		t.Run(fmt.Sprint(directory), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "lease.md")
			sentinel := path + ".tmp"
			if directory {
				if err := os.Mkdir(sentinel, 0755); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(sentinel, []byte("owned elsewhere"), 0644); err != nil {
				t.Fatal(err)
			}
			lease := &Lease{BatchID: "batch", CreatedAt: "now", BaseSHA: "abcdef1", N: 2, Status: StatusRunning, Slices: []LeaseSlice{{ID: "a", Paths: []string{"a.go"}}, {ID: "b", Paths: []string{"b.go"}}}}
			if err := WriteLease(path, lease); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadLease(path); err != nil {
				t.Fatal(err)
			}
			if directory {
				info, err := os.Stat(sentinel)
				if err != nil || !info.IsDir() {
					t.Fatalf("sentinel changed: %v", err)
				}
			} else {
				raw, err := os.ReadFile(sentinel)
				if err != nil || string(raw) != "owned elsewhere" {
					t.Fatalf("sentinel changed: %q %v", raw, err)
				}
			}
		})
	}
}

func TestCreateAbortCleanup(t *testing.T) {
	repo, base := setupRepo(t)
	slug := "demo-feature"
	batch := "batch1"
	lease, err := Create(CreateOpts{
		Root:    repo,
		Slug:    slug,
		BatchID: batch,
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if lease.Status != StatusRunning || lease.N != 2 {
		t.Fatalf("lease=%#v", lease)
	}
	wtA := lease.Slices[0].WorktreePath
	wtB := lease.Slices[1].WorktreePath
	for _, wt := range []string{wtA, wtB} {
		if _, err := os.Stat(wt); err != nil {
			t.Fatalf("missing worktree %s: %v", wt, err)
		}
		if head := gitOk(t, wt, "rev-parse", "HEAD"); head != base {
			t.Fatalf("worktree head %s want %s", head, base)
		}
	}
	if head := gitOk(t, repo, "rev-parse", "HEAD"); head != base {
		t.Fatalf("control moved after create: %s", head)
	}

	// Drift control tip; abort must never rewind external commits — the drift
	// commit survives and the lease still marks aborted.
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("drift\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOk(t, repo, "add", "README.md")
	gitOk(t, repo, "commit", "-m", "drift")
	drift := gitOk(t, repo, "rev-parse", "HEAD")
	if _, err := Abort(repo, slug); err != nil {
		t.Fatal(err)
	}
	if head := gitOk(t, repo, "rev-parse", "HEAD"); head != drift {
		t.Fatalf("abort rewound control: head %s want drift tip %s", head, drift)
	}
	leasePath, _ := LeasePath(repo, slug)
	lease, err = ReadLease(leasePath)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Status != StatusAborted {
		t.Fatalf("status=%s", lease.Status)
	}
	if _, err := os.Stat(wtA); err != nil {
		t.Fatalf("abort should preserve worktree: %v", err)
	}

	if _, err := Cleanup(repo, slug, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wtA); !os.IsNotExist(err) {
		t.Fatalf("cleanup should remove worktree A")
	}
	if _, err := os.Stat(leasePath); !os.IsNotExist(err) {
		t.Fatalf("cleanup should clear lease")
	}
}

// TestCleanupSalvagesAbortedWork proves a forced cleanup of an aborted batch
// never destroys wright work: committed transfers keep their branches and
// uncommitted allowlisted changes are committed onto the slice branch before
// the worktree is removed.
func TestCleanupSalvagesAbortedWork(t *testing.T) {
	repo, base := setupRepo(t)
	slug, batch := "demo-feature", "batch1"
	lease, err := Create(CreateOpts{
		Root:    repo,
		Slug:    slug,
		BatchID: batch,
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go", "src/b_new.go"}},
			{ID: "slice-c", Paths: []string{"src/c.go"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	wtA := lease.Slices[0].WorktreePath
	wtB := lease.Slices[1].WorktreePath
	wtC := lease.Slices[2].WorktreePath

	// slice-a: committed transfer (red at review — work must survive).
	tcA := commitIn(t, wtA, "src/a.go", "A")
	// slice-b: uncommitted WIP — modified tracked file plus a new untracked
	// file inside the allowlist.
	if err := os.WriteFile(filepath.Join(wtB, "src", "b.go"), []byte("package main\n\nfunc Bwip() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wtB, "src", "b_new.go"), []byte("package main\n\nfunc Bnew() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// slice-c: untouched — nothing to salvage, branch should be deleted.

	if _, err := Abort(repo, slug); err != nil {
		t.Fatal(err)
	}
	salvaged, err := Cleanup(repo, slug, true)
	if err != nil {
		t.Fatal(err)
	}

	bySlice := map[string]Salvage{}
	for _, s := range salvaged {
		bySlice[s.SliceID] = s
	}
	if got := bySlice["slice-a"].Commit; got != tcA {
		t.Fatalf("slice-a salvage commit %s want transfer %s", got, tcA)
	}
	sb := bySlice["slice-b"]
	if sb.Branch == "" || sb.Commit == "" || sb.Commit == base {
		t.Fatalf("slice-b should salvage WIP onto its branch, got %+v", sb)
	}
	if _, ok := bySlice["slice-c"]; ok {
		t.Fatalf("slice-c carried no work; its branch should be deleted, got %+v", bySlice["slice-c"])
	}

	// The salvaged WIP commit actually contains the uncommitted changes.
	if names := gitOk(t, repo, "diff", "--name-only", base, sb.Commit); !strings.Contains(names, "src/b.go") || !strings.Contains(names, "src/b_new.go") {
		t.Fatalf("salvage commit missing WIP paths: %s", names)
	}
	if body := gitOk(t, repo, "show", sb.Commit+":src/b_new.go"); !strings.Contains(body, "Bnew") {
		t.Fatalf("salvaged file content missing: %q", body)
	}
	subj := gitOk(t, repo, "log", "-1", "--format=%s", sb.Commit)
	if strings.Contains(subj, "slice-b") || strings.Contains(subj, "SLICE-") {
		t.Fatalf("salvage subject names a slice id: %q", subj)
	}
	if !strings.HasPrefix(subj, "devrites: salvage WIP (") {
		t.Fatalf("salvage subject %q", subj)
	}

	// Worktrees removed, lease cleared, kept branches still resolve.
	for _, wt := range []string{wtA, wtB, wtC} {
		if _, err := os.Stat(wt); !os.IsNotExist(err) {
			t.Fatalf("cleanup should remove worktree %s", wt)
		}
	}
	leasePath, _ := LeasePath(repo, slug)
	if _, err := os.Stat(leasePath); !os.IsNotExist(err) {
		t.Fatalf("cleanup should clear lease")
	}
	gitOk(t, repo, "rev-parse", "--verify", "refs/heads/"+bySlice["slice-a"].Branch)
	gitOk(t, repo, "rev-parse", "--verify", "refs/heads/"+sb.Branch)
}

func TestCreateAcceptsFourSlices(t *testing.T) {
	t.Parallel()
	repo, _ := setupRepo(t)
	for _, name := range []string{"c.go", "d.go"} {
		if err := os.WriteFile(filepath.Join(repo, "src", name), []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitOk(t, repo, "add", ".")
	gitOk(t, repo, "commit", "-m", "more files")
	base := gitOk(t, repo, "rev-parse", "HEAD")
	lease, err := Create(CreateOpts{
		Root:    repo,
		Slug:    "demo-four",
		BatchID: "batch4",
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go"}},
			{ID: "slice-c", Paths: []string{"src/c.go"}},
			{ID: "slice-d", Paths: []string{"src/d.go"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if lease.N != 4 || len(lease.Slices) != 4 {
		t.Fatalf("want 4 slices, got n=%d slices=%d", lease.N, len(lease.Slices))
	}
}

func TestCreateRefusesCountOutsideBounds(t *testing.T) {
	t.Parallel()
	repo, base := setupRepo(t)
	_, err := Create(CreateOpts{
		Root:    repo,
		Slug:    "demo-one",
		BatchID: "batch1",
		BaseSHA: base,
		Slices:  []SlicePaths{{ID: "slice-a", Paths: []string{"src/a.go"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "2-10 slices") {
		t.Fatalf("expected 2-10 bound for N=1, got %v", err)
	}

	slices := make([]SlicePaths, MaxParallelSlices+1)
	for i := range slices {
		name := fmt.Sprintf("src/x%d.go", i)
		if err := os.WriteFile(filepath.Join(repo, name), []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		slices[i] = SlicePaths{ID: fmt.Sprintf("s%d", i), Paths: []string{name}}
	}
	gitOk(t, repo, "add", ".")
	gitOk(t, repo, "commit", "-m", "eleven")
	base = gitOk(t, repo, "rev-parse", "HEAD")
	_, err = Create(CreateOpts{
		Root:    repo,
		Slug:    "demo-eleven",
		BatchID: "batch11",
		BaseSHA: base,
		Slices:  slices,
	})
	if err == nil || !strings.Contains(err.Error(), "2-10 slices") {
		t.Fatalf("expected 2-10 bound for N=11, got %v", err)
	}
}

func TestValidateLeaseBounds(t *testing.T) {
	t.Parallel()
	ok := validTestLease(MaxParallelSlices)
	if err := ValidateLease(ok); err != nil {
		t.Fatalf("N=%d should validate: %v", MaxParallelSlices, err)
	}
	tooMany := validTestLease(MaxParallelSlices + 1)
	if err := ValidateLease(tooMany); err == nil || !strings.Contains(err.Error(), "2-10 slices") {
		t.Fatalf("N=%d should refuse, got %v", MaxParallelSlices+1, err)
	}
}

func validTestLease(n int) *Lease {
	slices := make([]LeaseSlice, n)
	for i := range slices {
		id := fmt.Sprintf("s%d", i)
		slices[i] = LeaseSlice{
			ID:           id,
			Paths:        []string{fmt.Sprintf("src/%s.go", id)},
			WorktreePath: "/tmp/" + id,
			Branch:       "devrites/parallel/x/batch/" + id,
			WrightStatus: WrightPending,
		}
	}
	return &Lease{
		BatchID:             "batch1",
		CreatedAt:           "2026-08-23T00:00:00Z",
		BaseSHA:             "abcdef1",
		N:                   n,
		Status:              StatusRunning,
		ControlPIDOrSession: "test",
		Slices:              slices,
	}
}

func TestCreateRefusesOverlap(t *testing.T) {
	repo, base := setupRepo(t)
	_, err := Create(CreateOpts{
		Root:    repo,
		Slug:    "demo",
		BatchID: "batch1",
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/a.go"}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "overlap") {
		t.Fatalf("expected overlap refusal, got %v", err)
	}
}

func TestIntegrateDivergentSiblings(t *testing.T) {
	repo, base := setupRepo(t)
	slug := "demo-feature"
	batch := "batch1"
	lease, err := Create(CreateOpts{
		Root:    repo,
		Slug:    slug,
		BatchID: batch,
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	wtA := lease.Slices[0].WorktreePath
	wtB := lease.Slices[1].WorktreePath

	if err := os.WriteFile(filepath.Join(wtA, "src", "a.go"), []byte("package main\n\nfunc A() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOk(t, wtA, "add", "src/a.go")
	gitOk(t, wtA, "commit", "-m", "WIP(demo-feature): add A helper")
	tcA := gitOk(t, wtA, "rev-parse", "HEAD")

	if err := os.WriteFile(filepath.Join(wtB, "src", "b.go"), []byte("package main\n\nfunc B() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOk(t, wtB, "add", "src/b.go")
	gitOk(t, wtB, "commit", "-m", "WIP(demo-feature): add B helper")
	tcB := gitOk(t, wtB, "rev-parse", "HEAD")

	if _, err := RecordGreen(repo, slug, "slice-a", tcA); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordGreen(repo, slug, "slice-b", tcB); err != nil {
		t.Fatal(err)
	}

	tip, lease, err := Integrate(IntegrateOpts{Root: repo, Slug: slug, ApplyToControl: true})
	if err != nil {
		t.Fatal(err)
	}
	if lease.Status != StatusComplete {
		t.Fatalf("status=%s", lease.Status)
	}
	if tip == "" || tip == base {
		t.Fatalf("expected advanced tip, got %s", tip)
	}
	head := gitOk(t, repo, "rev-parse", "HEAD")
	if head != tip {
		t.Fatalf("control head %s want tip %s", head, tip)
	}
	if n := gitOk(t, repo, "rev-list", "--count", base+"..HEAD"); n != "2" {
		t.Fatalf("control should gain one commit per sibling, got %s", n)
	}
	log := gitOk(t, repo, "log", "--format=%s", base+"..HEAD")
	for _, subj := range strings.Split(log, "\n") {
		if !strings.HasPrefix(subj, "WIP(demo-feature):") {
			t.Fatalf("control subject %q missing WIP(demo-feature): prefix", subj)
		}
		if strings.Contains(strings.ToUpper(subj), "SLICE-") {
			t.Fatalf("control subject %q still names a slice id", subj)
		}
	}
	a := mustRead(t, filepath.Join(repo, "src", "a.go"))
	b := mustRead(t, filepath.Join(repo, "src", "b.go"))
	if !strings.Contains(a, "func A") || !strings.Contains(b, "func B") {
		t.Fatalf("integrated contents missing: a=%q b=%q", a, b)
	}

	if _, err := Cleanup(repo, slug, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wtA); !os.IsNotExist(err) {
		t.Fatalf("cleanup should remove workers")
	}
}

// TestIntegrateWithoutApplyKeepsLeaseRunning proves a flag-less integrate
// stages the per-sibling commits on the integrate branch but leaves the lease
// running and control at base, so a later non-forced cleanup cannot discard
// the only refs holding the integrated tree.
func TestIntegrateWithoutApplyKeepsLeaseRunning(t *testing.T) {
	repo, base := setupRepo(t)
	slug, batch := "demo-feature", "batch1"
	greenBatch(t, repo, base, slug, batch)

	tip, lease, err := Integrate(IntegrateOpts{Root: repo, Slug: slug})
	if err != nil {
		t.Fatal(err)
	}
	if tip == "" || tip == base {
		t.Fatalf("expected staged integrate tip, got %s", tip)
	}
	if lease.Status != StatusRunning {
		t.Fatalf("status=%s want running without --apply-to-control", lease.Status)
	}
	if head := gitOk(t, repo, "rev-parse", "HEAD"); head != base {
		t.Fatalf("control moved without apply: %s", head)
	}
	ibranch := IntegrateBranchName(slug, batch)
	if got := gitOk(t, repo, "rev-parse", ibranch); got != tip {
		t.Fatalf("integrate branch %s want integrate tip %s", got, tip)
	}
	if _, err := Cleanup(repo, slug, false); err == nil {
		t.Fatal("cleanup on running lease must refuse without --force")
	}
	// Forced cleanup salvages then removes; the integrate branch holds the
	// integrate tip, so it is kept and reported rather than deleted.
	if _, err := Cleanup(repo, slug, true); err != nil {
		t.Fatal(err)
	}
	if got := gitOk(t, repo, "rev-parse", ibranch); got != tip {
		t.Fatalf("forced cleanup dropped integrate branch: %s want %s", got, tip)
	}
}

// TestIntegrateFailedLeaseRepairsAndRetries proves integrate-failed is not a
// dead end: a repaired transfer re-records on the failed lease and a retried
// integrate completes into one control commit per sibling.
func TestIntegrateFailedLeaseRepairsAndRetries(t *testing.T) {
	repo, base := setupRepo(t)
	slug, batch := "demo-feature", "batch1"
	greenBatch(t, repo, base, slug, batch)
	realTC := gitOk(t, repo, "rev-parse", WorkerBranch(slug, batch, "slice-a"))

	leasePath, err := LeasePath(repo, slug)
	if err != nil {
		t.Fatal(err)
	}
	bad, err := ReadLease(leasePath)
	if err != nil {
		t.Fatal(err)
	}
	bad.Slices[0].TransferCommit = strings.Repeat("deadbeef", 5)
	if err := WriteLease(leasePath, bad); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: slug, ApplyToControl: true}); err == nil {
		t.Fatal("expected integrate failure")
	}
	onDisk, err := ReadLease(leasePath)
	if err != nil {
		t.Fatal(err)
	}
	if onDisk.Status != StatusIntegrateFailed {
		t.Fatalf("status=%s want integrate-failed", onDisk.Status)
	}

	if _, err := RecordGreen(repo, slug, "slice-a", realTC); err != nil {
		t.Fatalf("record-green on integrate-failed lease: %v", err)
	}
	if _, lease, err := Integrate(IntegrateOpts{Root: repo, Slug: slug, ApplyToControl: true}); err != nil {
		t.Fatalf("integrate retry: %v", err)
	} else if lease.Status != StatusComplete {
		t.Fatalf("status=%s want complete", lease.Status)
	}
	if n := gitOk(t, repo, "rev-list", "--count", base+"..HEAD"); n != "2" {
		t.Fatalf("expected one control commit per sibling, got %s", n)
	}
}

// TestIntegrateRefusesDirtyControlOnSlicePaths proves --apply-to-control
// neither absorbs nor clobbers uncommitted user work on slice paths: overlap
// refuses with the lease still running, while unrelated dirty files are left
// alone and the FF proceeds.
func TestIntegrateRefusesDirtyControlOnSlicePaths(t *testing.T) {
	repo, base := setupRepo(t)
	slug, batch := "demo-feature", "batch1"
	greenBatch(t, repo, base, slug, batch)

	if err := os.MkdirAll(filepath.Join(repo, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	userWIP := filepath.Join(repo, "src", "a.go")
	if err := os.WriteFile(userWIP, []byte("user wip\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: slug, ApplyToControl: true}); err == nil {
		t.Fatal("expected dirty-control refusal")
	}
	leasePath, err := LeasePath(repo, slug)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := ReadLease(leasePath)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Status != StatusRunning {
		t.Fatalf("lease status=%s want running after precondition refusal", lease.Status)
	}
	if head := gitOk(t, repo, "rev-parse", "HEAD"); head != base {
		t.Fatalf("control moved despite refusal: %s", head)
	}

	// Unrelated uncommitted work is not part of the batch and must survive.
	gitOk(t, repo, "checkout", "--", "src/a.go")
	keep := filepath.Join(repo, "unrelated.txt")
	if err := os.WriteFile(keep, []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: slug, ApplyToControl: true}); err != nil {
		t.Fatalf("integrate with non-overlapping dirty file: %v", err)
	}
	if b, err := os.ReadFile(keep); err != nil || string(b) != "keep me\n" {
		t.Fatalf("unrelated dirty file lost: err=%v body=%q", err, b)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
