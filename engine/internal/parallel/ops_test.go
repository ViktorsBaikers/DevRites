package parallel

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// commitIn makes a one-file green commit inside a worker worktree and returns
// its transfer commit SHA.
func commitIn(t *testing.T, wt, file, marker string) string {
	t.Helper()
	body := "package main\n\nfunc " + marker + "() {}\n"
	if err := os.WriteFile(filepath.Join(wt, file), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOk(t, wt, "add", file)
	gitOk(t, wt, "commit", "-m", "WIP(demo-feature): add "+marker+" helper")
	return gitOk(t, wt, "rev-parse", "HEAD")
}

// greenBatch creates a running lease with two recorded-green slices.
func greenBatch(t *testing.T, repo, base, slug, batch string) *Lease {
	t.Helper()
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
	tcA := commitIn(t, lease.Slices[0].WorktreePath, "src/a.go", "A")
	tcB := commitIn(t, lease.Slices[1].WorktreePath, "src/b.go", "B")
	if _, err := RecordGreen(repo, slug, "slice-a", tcA); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordGreen(repo, slug, "slice-b", tcB); err != nil {
		t.Fatal(err)
	}
	return lease
}

type seamRestore struct {
	origLease   func(string, *Lease) error
	origRemove  func(string, string) error
	origSalvage func(LeaseSlice, string) (string, error)
	origWarn    func(string, ...any)
}

func swapSeams(t *testing.T) {
	t.Helper()
	s := seamRestore{origLease: writeLease, origRemove: removeWorktree, origSalvage: salvage, origWarn: warnf}
	t.Cleanup(func() {
		writeLease, removeWorktree, salvage, warnf = s.origLease, s.origRemove, s.origSalvage, s.origWarn
	})
}

// TestIntegrateFailedTransitionSurvivesTransientWriteFailure proves the
// integrate-failed lease transition is retried once when the first write
// fails, so the on-disk lease honestly records integrate-failed instead of a
// stale running status.
func TestIntegrateFailedTransitionSurvivesTransientWriteFailure(t *testing.T) {
	repo, base := setupRepo(t)
	slug, batch := "demo-feature", "batch1"
	greenBatch(t, repo, base, slug, batch)
	leasePath, err := LeasePath(repo, slug)
	if err != nil {
		t.Fatal(err)
	}

	// Corrupt a transfer commit so Integrate fails mid-flight via fail().
	corrupted, err := ReadLease(leasePath)
	if err != nil {
		t.Fatal(err)
	}
	corrupted.Slices[0].TransferCommit = strings.Repeat("deadbeef", 5)
	if err := WriteLease(leasePath, corrupted); err != nil {
		t.Fatal(err)
	}

	swapSeams(t)
	calls := 0
	writeLease = func(path string, l *Lease) error {
		calls++
		if calls == 1 {
			return errors.New("transient lease write failure")
		}
		// delegates to the real WriteLease captured by swapSeams
		return origWriteLease()(path, l)
	}

	_, got, err := Integrate(IntegrateOpts{Root: repo, Slug: slug, ApplyToControl: true})
	if err == nil {
		t.Fatal("expected integrate failure from corrupted transfer commit")
	}
	if calls != 2 {
		t.Fatalf("lease write calls=%d want 2 (initial + one retry)", calls)
	}
	if got == nil || got.Status != StatusIntegrateFailed {
		t.Fatalf("returned lease status=%v want integrate-failed", got)
	}
	onDisk, err := ReadLease(leasePath)
	if err != nil {
		t.Fatal(err)
	}
	if onDisk.Status != StatusIntegrateFailed {
		t.Fatalf("on-disk lease status=%s want integrate-failed", onDisk.Status)
	}
	if head := gitOk(t, repo, "rev-parse", "HEAD"); head != base {
		t.Fatalf("control head %s want base %s", head, base)
	}
}

// origWriteLease returns the real WriteLease bypassing any test override.
func origWriteLease() func(string, *Lease) error {
	return WriteLease
}

// TestIntegrateSurfacesStuckStagingWorktree proves a staging-worktree removal
// failure on the integrate success path is propagated (fail closed, status
// integrate-failed, control untouched) instead of silently compounding into
// un-cleanable branch state.
func TestIntegrateSurfacesStuckStagingWorktree(t *testing.T) {
	repo, base := setupRepo(t)
	slug, batch := "demo-feature", "batch1"
	greenBatch(t, repo, base, slug, batch)
	leasePath, err := LeasePath(repo, slug)
	if err != nil {
		t.Fatal(err)
	}

	swapSeams(t)
	removeWorktree = func(string, string) error {
		return errors.New("simulated locked staging worktree")
	}

	_, got, err := Integrate(IntegrateOpts{Root: repo, Slug: slug, ApplyToControl: true})
	if err == nil {
		t.Fatal("expected integrate to surface the stuck staging worktree")
	}
	if !strings.Contains(err.Error(), "staging worktree") {
		t.Fatalf("error should name the staging worktree, got %v", err)
	}
	if got == nil || got.Status != StatusIntegrateFailed {
		t.Fatalf("returned lease status=%v want integrate-failed", got)
	}
	onDisk, err := ReadLease(leasePath)
	if err != nil {
		t.Fatal(err)
	}
	if onDisk.Status != StatusIntegrateFailed {
		t.Fatalf("on-disk lease status=%s want integrate-failed", onDisk.Status)
	}
	if head := gitOk(t, repo, "rev-parse", "HEAD"); head != base {
		t.Fatalf("control head %s want base %s (merge must not apply when staging cleanup fails)", head, base)
	}
}

// TestCleanupNeverTouchesForeignWorktreePath proves a lease whose
// worktree_path is not the exact directory Create wrote is never salvaged
// into nor deleted — the lease field is advisory, not a RemoveAll license.
// Two spellings of "not ours" are covered: a path outside the batch scratch
// layout, and the deterministic scratch path itself reached through a symlink.
func TestCleanupNeverTouchesForeignWorktreePath(t *testing.T) {
	slug, batch := "demo-feature", "batch1"
	slices := []SlicePaths{
		{ID: "slice-a", Paths: []string{"src/a.go"}},
		{ID: "slice-b", Paths: []string{"src/b.go"}},
	}

	t.Run("worktree_path outside the scratch root", func(t *testing.T) {
		repo, base := setupRepo(t)
		lease, err := Create(CreateOpts{
			Root:    repo,
			Slug:    slug,
			BatchID: batch,
			BaseSHA: base,
			Slices:  slices,
		})
		if err != nil {
			t.Fatal(err)
		}
		foreign := filepath.Join(t.TempDir(), "precious")
		if err := os.MkdirAll(foreign, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(foreign, "keep.txt"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		lease.Slices[0].WorktreePath = foreign
		// A forged branch outside the batch namespace must survive too.
		gitOk(t, repo, "branch", "user/topic", base)
		lease.Slices[0].Branch = "user/topic"
		leasePath, err := LeasePath(repo, slug)
		if err != nil {
			t.Fatal(err)
		}
		if err := WriteLease(leasePath, lease); err != nil {
			t.Fatal(err)
		}

		var warnings []string
		swapSeams(t)
		warnf = func(format string, args ...any) {
			warnings = append(warnings, fmt.Sprintf(format, args...))
		}
		if _, err := Cleanup(repo, slug, true); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(foreign, "keep.txt")); err != nil {
			t.Fatalf("foreign path was touched: %v", err)
		}
		if tip := gitOk(t, repo, "rev-parse", "user/topic"); tip != base {
			t.Fatalf("foreign branch user/topic moved to %s, want %s", tip, base)
		}
		joined := strings.Join(warnings, "\n")
		if !strings.Contains(joined, "outside") {
			t.Fatalf("expected warnings naming foreign path and branch, got:\n%s", warnings)
		}
	})

	// A stale lease still names exactly the path Create wrote; here the batch
	// directory was replaced by a symlink, so the directory a forced cleanup
	// would delete lives outside the scratch root. filepath.Abs matches both sides
	// of the ownership check because neither side resolves symlinks.
	t.Run("scratch worktree_path reached through a symlink", func(t *testing.T) {
		repo, base := setupRepo(t)
		if _, err := Create(CreateOpts{
			Root:    repo,
			Slug:    slug,
			BatchID: batch,
			BaseSHA: base,
			Slices:  slices,
		}); err != nil {
			t.Fatal(err)
		}

		// An unrelated repository outside the workspace, carrying a committed file
		// and a dirty one on the allowlisted path: a salvage would commit inside it
		// and a RemoveAll would delete the tree.
		outside := t.TempDir()
		sliceA := filepath.Join(outside, "slice-a")
		if err := os.MkdirAll(filepath.Join(sliceA, "src"), 0o755); err != nil {
			t.Fatal(err)
		}
		gitOk(t, sliceA, "init")
		gitOk(t, sliceA, "checkout", "-b", "main")
		if err := os.WriteFile(filepath.Join(sliceA, "src", "a.go"), []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitOk(t, sliceA, "add", ".")
		gitOk(t, sliceA, "commit", "-m", "unrelated base")
		unrelatedTip := gitOk(t, sliceA, "rev-parse", "HEAD")
		if err := os.WriteFile(filepath.Join(sliceA, "src", "a.go"), []byte("package main\n\nfunc Unrelated() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		batchDir := filepath.Join(ScratchRoot(repo), batch)
		if err := os.RemoveAll(batchDir); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, batchDir); err != nil {
			t.Fatal(err)
		}

		// The lease on disk is what Create wrote; only the filesystem moved.
		// Assert the path cleanup will use really does resolve through the symlink,
		// so this case cannot pass without traversing it.
		leasePath, err := LeasePath(repo, slug)
		if err != nil {
			t.Fatal(err)
		}
		lease, err := ReadLease(leasePath)
		if err != nil {
			t.Fatal(err)
		}
		claimed := lease.Slices[0].WorktreePath
		if want := WorkerWorktreePath(repo, batch, "slice-a"); claimed != want {
			t.Fatalf("lease worktree_path %q, want the deterministic %q", claimed, want)
		}
		resolved, err := filepath.EvalSymlinks(claimed)
		if err != nil {
			t.Fatalf("claimed worktree_path does not resolve: %v", err)
		}
		realSliceA, err := filepath.EvalSymlinks(sliceA)
		if err != nil {
			t.Fatal(err)
		}
		if resolved != realSliceA {
			t.Fatalf("%q resolved to %q, want the symlink target %q", claimed, resolved, realSliceA)
		}

		var warnings []string
		swapSeams(t)
		warnf = func(format string, args ...any) {
			warnings = append(warnings, fmt.Sprintf(format, args...))
		}
		if _, err := Cleanup(repo, slug, true); err != nil {
			t.Fatal(err)
		}

		if _, err := os.Stat(filepath.Join(sliceA, "src", "a.go")); err != nil {
			t.Fatalf("worktree reached through a symlinked batch directory was deleted: %v", err)
		}
		if tip := gitOk(t, sliceA, "rev-parse", "HEAD"); tip != unrelatedTip {
			t.Fatalf("salvage committed inside an unrelated repository: tip %s want %s", tip, unrelatedTip)
		}
		if joined := strings.Join(warnings, "\n"); !strings.Contains(joined, "worktree_path") {
			t.Fatalf("expected a warning naming the refused worktree_path, got:\n%s", warnings)
		}
	})

	// A symlink that stays inside the scratch root still redirects the delete
	// onto a directory the lease does not name.
	t.Run("scratch worktree_path reached through a symlink inside the scratch root", func(t *testing.T) {
		repo, base := setupRepo(t)
		if _, err := Create(CreateOpts{
			Root:    repo,
			Slug:    slug,
			BatchID: batch,
			BaseSHA: base,
			Slices:  slices,
		}); err != nil {
			t.Fatal(err)
		}
		sibling := filepath.Join(ScratchRoot(repo), "sibling-batch")
		keep := filepath.Join(sibling, "slice-a", "keep.txt")
		if err := os.MkdirAll(filepath.Dir(keep), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(keep, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		batchDir := filepath.Join(ScratchRoot(repo), batch)
		if err := os.RemoveAll(batchDir); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(sibling, batchDir); err != nil {
			t.Fatal(err)
		}

		swapSeams(t)
		warnf = func(string, ...any) {}
		if _, err := Cleanup(repo, slug, true); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(keep); err != nil {
			t.Fatalf("directory under an in-scratch symlink was deleted: %v", err)
		}
	})

	// Ownership is decided by what git registered when Create ran, not by the
	// lease. A lease naming a deterministic in-scratch path that Create never
	// made (and a stray sibling no lease names) must survive a forced cleanup,
	// while the real worktrees are still removed.
	t.Run("in-scratch path Create never made", func(t *testing.T) {
		repo, base := setupRepo(t)
		lease, err := Create(CreateOpts{
			Root:    repo,
			Slug:    slug,
			BatchID: batch,
			BaseSHA: base,
			Slices:  slices,
		})
		if err != nil {
			t.Fatal(err)
		}
		forged := WorkerWorktreePath(repo, batch, "slice-x")
		stray := filepath.Join(ScratchRoot(repo), batch, "stray")
		for _, d := range []string{forged, stray} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(d, "keep.txt"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		lease.Slices = append(lease.Slices, LeaseSlice{
			ID:           "slice-x",
			Paths:        []string{"src/x.go"},
			WorktreePath: forged,
			WrightStatus: WrightPending,
		})
		lease.N = len(lease.Slices)
		leasePath, err := LeasePath(repo, slug)
		if err != nil {
			t.Fatal(err)
		}
		if err := WriteLease(leasePath, lease); err != nil {
			t.Fatal(err)
		}

		var warnings []string
		swapSeams(t)
		warnf = func(format string, args ...any) {
			warnings = append(warnings, fmt.Sprintf(format, args...))
		}
		if _, err := Cleanup(repo, slug, true); err != nil {
			t.Fatal(err)
		}
		for _, d := range []string{forged, stray} {
			if _, err := os.Stat(filepath.Join(d, "keep.txt")); err != nil {
				t.Fatalf("unregistered scratch directory %s was deleted: %v", d, err)
			}
		}
		for _, id := range []string{"slice-a", "slice-b"} {
			if _, err := os.Stat(WorkerWorktreePath(repo, batch, id)); !os.IsNotExist(err) {
				t.Fatalf("registered worktree %s was not removed: %v", id, err)
			}
		}
		if joined := strings.Join(warnings, "\n"); !strings.Contains(joined, "not a worktree") {
			t.Fatalf("expected a warning naming the refused path, got:\n%s", warnings)
		}
	})
}

// TestCleanupFailedSalvageKeepsWorktreeDir proves a salvage error keeps the
// worktree on disk as the only remaining copy — the batch-level scratch
// removal must skip kept slice dirs instead of destroying them.
func TestCleanupFailedSalvageKeepsWorktreeDir(t *testing.T) {
	repo, base := setupRepo(t)
	slug, batch := "demo-feature", "batch1"
	if _, err := Create(CreateOpts{
		Root:    repo,
		Slug:    slug,
		BatchID: batch,
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	wtA := WorkerWorktreePath(repo, batch, "slice-a")
	wtB := WorkerWorktreePath(repo, batch, "slice-b")

	swapSeams(t)
	realSalvage := salvage
	salvage = func(sl LeaseSlice, batchID string) (string, error) {
		if sl.ID == "slice-a" {
			return "", errors.New("injected salvage failure")
		}
		return realSalvage(sl, batchID)
	}
	salvaged, err := Cleanup(repo, slug, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wtA); err != nil {
		t.Fatalf("kept worktree was removed: %v", err)
	}
	if _, err := os.Stat(wtB); !os.IsNotExist(err) {
		t.Fatalf("clean slice-b worktree should be removed, stat err=%v", err)
	}
	if _, err := git(repo, "rev-parse", "--verify", "devrites/parallel/"+slug+"/"+batch+"/slice-a"); err != nil {
		t.Fatalf("slice-a branch should be kept: %v", err)
	}
	found := false
	for _, s := range salvaged {
		if s.SliceID == "slice-a" && s.Worktree == wtA {
			found = true
		}
	}
	if !found {
		t.Fatalf("salvage output should report kept worktree for slice-a, got %+v", salvaged)
	}
}

// TestCleanupNamesDirectoryGitDroppedButNotDeleted proves that when git has
// already dropped a worktree and its directory cannot be deleted, the error
// names the directory and says a rerun will not delete it, and the rerun does
// leave it on disk.
func TestCleanupNamesDirectoryGitDroppedButNotDeleted(t *testing.T) {
	if os.Geteuid() == 0 || runtime.GOOS == "windows" {
		t.Skip("directory permissions do not bind root or windows")
	}
	repo, base := setupRepo(t)
	slug, batch := "demo-feature", "batch1"
	if _, err := Create(CreateOpts{
		Root:    repo,
		Slug:    slug,
		BatchID: batch,
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	wt := WorkerWorktreePath(repo, batch, "slice-a")
	batchDir := filepath.Dir(wt)

	swapSeams(t)
	warnf = func(string, ...any) {}
	removeWorktree = func(repo, path string) error {
		// Like real git: drop the registration, fail to delete the
		// directory, and exit non-zero.
		if err := os.RemoveAll(filepath.Join(repo, ".git", "worktrees", filepath.Base(path))); err != nil {
			return err
		}
		if err := os.Chmod(batchDir, 0o500); err != nil {
			return err
		}
		return errors.New("failed to delete '" + path + "': Permission denied")
	}
	t.Cleanup(func() { _ = os.Chmod(batchDir, 0o700) })

	_, err := Cleanup(repo, slug, true)
	if err == nil || !strings.Contains(err.Error(), wt) || !strings.Contains(err.Error(), "remove it by hand") {
		t.Fatalf("error must name %s and say it needs manual removal, got %v", wt, err)
	}
	if err := os.Chmod(batchDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Cleanup(repo, slug, true); err != nil {
		t.Fatalf("rerun: %v", err)
	}
	if _, err := os.Stat(wt); err != nil {
		t.Fatalf("rerun must leave the unregistered directory in place, stat err=%v", err)
	}
}

// TestCleanupFailedListNamesEachEntryOnce proves a slice whose directory
// removal fails in both the per-slice and the scratch-entry pass is named once,
// in lease order.
func TestCleanupFailedListNamesEachEntryOnce(t *testing.T) {
	if os.Geteuid() == 0 || runtime.GOOS == "windows" {
		t.Skip("directory permissions do not bind root or windows")
	}
	repo, base := setupRepo(t)
	slug, batch := "demo-feature", "batch1"
	if _, err := Create(CreateOpts{
		Root:    repo,
		Slug:    slug,
		BatchID: batch,
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	batchDir := filepath.Dir(WorkerWorktreePath(repo, batch, "slice-a"))

	swapSeams(t)
	warnf = func(string, ...any) {}
	removeWorktree = func(string, string) error {
		if err := os.Chmod(batchDir, 0o500); err != nil {
			return err
		}
		return errors.New("simulated stuck worker worktree")
	}
	t.Cleanup(func() { _ = os.Chmod(batchDir, 0o700) })

	_, err := Cleanup(repo, slug, true)
	const want = "cleanup incomplete for slice-a, slice-b; "
	if err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("failed list must name slice-a and slice-b once each, want prefix %q, got %v", want, err)
	}
}

// TestCleanupKeepsRerunGuidanceWhileStillRegistered proves a failed directory
// delete that leaves git's registration intact is retryable: the error asks for
// a rerun and does not tell the operator to remove the directory by hand.
func TestCleanupKeepsRerunGuidanceWhileStillRegistered(t *testing.T) {
	if os.Geteuid() == 0 || runtime.GOOS == "windows" {
		t.Skip("directory permissions do not bind root or windows")
	}
	repo, base := setupRepo(t)
	slug, batch := "demo-feature", "batch1"
	if _, err := Create(CreateOpts{
		Root:    repo,
		Slug:    slug,
		BatchID: batch,
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	wt := WorkerWorktreePath(repo, batch, "slice-a")
	batchDir := filepath.Dir(wt)

	swapSeams(t)
	warnf = func(string, ...any) {}
	removeWorktree = func(repo, path string) error {
		if err := os.Chmod(batchDir, 0o500); err != nil {
			return err
		}
		return errors.New("failed to delete '" + path + "': Permission denied")
	}
	t.Cleanup(func() { _ = os.Chmod(batchDir, 0o700) })

	_, err := Cleanup(repo, slug, true)
	if err == nil || !strings.Contains(err.Error(), "rerun cleanup") {
		t.Fatalf("error must ask for a rerun, got %v", err)
	}
	if strings.Contains(err.Error(), "remove it by hand") {
		t.Fatalf("a still-registered worktree is retryable, got %v", err)
	}
	leasePath, lerr := LeasePath(repo, slug)
	if lerr != nil {
		t.Fatal(lerr)
	}
	if _, rerr := ReadLease(leasePath); rerr != nil {
		t.Fatalf("lease must be kept for the rerun: %v", rerr)
	}
}

// TestCleanupDoesNotClaimUnregisteredWhenListUnreadable proves that when git's
// registration cannot be re-read after a failed delete, the error says so
// instead of claiming git dropped the worktree.
func TestCleanupDoesNotClaimUnregisteredWhenListUnreadable(t *testing.T) {
	if os.Geteuid() == 0 || runtime.GOOS == "windows" {
		t.Skip("directory permissions do not bind root or windows")
	}
	repo, base := setupRepo(t)
	slug, batch := "demo-feature", "batch1"
	if _, err := Create(CreateOpts{
		Root:    repo,
		Slug:    slug,
		BatchID: batch,
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	wt := WorkerWorktreePath(repo, batch, "slice-a")
	batchDir := filepath.Dir(wt)
	gitDir := filepath.Join(repo, ".git")

	swapSeams(t)
	warnf = func(string, ...any) {}
	removeWorktree = func(repo, path string) error {
		if err := os.Chmod(batchDir, 0o500); err != nil {
			return err
		}
		if err := os.Chmod(gitDir, 0o000); err != nil {
			return err
		}
		return errors.New("failed to delete '" + path + "': Permission denied")
	}
	t.Cleanup(func() {
		_ = os.Chmod(batchDir, 0o700)
		_ = os.Chmod(gitDir, 0o755)
	})

	_, err := Cleanup(repo, slug, true)
	if err == nil || !strings.Contains(err.Error(), wt) {
		t.Fatalf("error must name %s, got %v", wt, err)
	}
	if strings.Contains(err.Error(), "no longer registers") {
		t.Fatalf("registration is unknown, must not be called dropped: %v", err)
	}
	if !strings.Contains(err.Error(), "could not confirm whether git still registers") {
		t.Fatalf("error must say registration could not be confirmed, got %v", err)
	}
}

// TestCleanupStrandedRealGit runs the real git: an undeletable file inside the
// worktree makes `git worktree remove --force` unregister it and still fail,
// so stranding must be decided from git's registration, not the exit code.
func TestCleanupStrandedRealGit(t *testing.T) {
	if os.Geteuid() == 0 || runtime.GOOS == "windows" {
		t.Skip("directory permissions do not bind root or windows")
	}
	repo, base := setupRepo(t)
	slug, batch := "demo-feature", "batch1"
	if _, err := Create(CreateOpts{
		Root:    repo,
		Slug:    slug,
		BatchID: batch,
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	wt := WorkerWorktreePath(repo, batch, "slice-a")
	locked := filepath.Join(wt, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	swapSeams(t)
	warnf = func(string, ...any) {}
	_, err := Cleanup(repo, slug, true)
	if err == nil || !strings.Contains(err.Error(), wt) || !strings.Contains(err.Error(), "remove it by hand") {
		t.Fatalf("error must name %s and say it needs manual removal, got %v", wt, err)
	}
	if err := os.Chmod(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Cleanup(repo, slug, true); err != nil {
		t.Fatalf("rerun: %v", err)
	}
	if _, err := os.Stat(wt); err != nil {
		t.Fatalf("rerun must leave the unregistered directory in place, stat err=%v", err)
	}
}

// TestConcurrentRecordGreenKeepsEverySlice proves the feature lock serializes
// concurrent record-green calls so no sibling's transfer commit is lost to a
// read-modify-write race on the lease.
func TestConcurrentRecordGreenKeepsEverySlice(t *testing.T) {
	repo, base := setupRepo(t)
	slug, batch := "demo-feature", "batch1"
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
	tcA := commitIn(t, lease.Slices[0].WorktreePath, "src/a.go", "A")
	tcB := commitIn(t, lease.Slices[1].WorktreePath, "src/b.go", "B")

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, item := range [][2]string{{"slice-a", tcA}, {"slice-b", tcB}} {
		wg.Add(1)
		go func(sliceID, commit string) {
			defer wg.Done()
			_, err := RecordGreen(repo, slug, sliceID, commit)
			errs <- err
		}(item[0], item[1])
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	leasePath, err := LeasePath(repo, slug)
	if err != nil {
		t.Fatal(err)
	}
	onDisk, err := ReadLease(leasePath)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, sl := range onDisk.Slices {
		got[sl.ID] = sl.TransferCommit
	}
	if got["slice-a"] != tcA || got["slice-b"] != tcB {
		t.Fatalf("lost update: transfer commits %v want a=%s b=%s", got, tcA, tcB)
	}
}

// TestConcurrentCreateSingleWinner proves the feature lock serializes create
// so two racing creates cannot both pass the lease-existence check.
func TestConcurrentCreateSingleWinner(t *testing.T) {
	repo, base := setupRepo(t)
	opts := CreateOpts{
		Root:    repo,
		Slug:    "demo-feature",
		BatchID: "batch1",
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go"}},
		},
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wins := make(chan *Lease, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lease, err := Create(opts)
			if err != nil {
				errs <- err
				return
			}
			wins <- lease
		}()
	}
	wg.Wait()
	close(errs)
	close(wins)
	succeeded := 0
	for range wins {
		succeeded++
	}
	failed := 0
	for range errs {
		failed++
	}
	if succeeded != 1 || failed != 1 {
		t.Fatalf("concurrent create: %d succeeded, %d failed; want exactly 1/1", succeeded, failed)
	}
}

// stuckBranchBatch creates a two-slice batch and stubs worktree removal so
// cleanup leaves a stale registration behind: Cleanup's own RemoveAll deletes
// the worktree directory while git still lists the branch as checked out, so
// deleteBranch refuses. It returns the repo, slug, the stuck branch and the
// captured warnings.
func stuckBranchBatch(t *testing.T) (repo, slug, branch string, warnings *[]string) {
	t.Helper()
	repo, base := setupRepo(t)
	slug = "demo-feature"
	lease, err := Create(CreateOpts{
		Root:    repo,
		Slug:    slug,
		BatchID: "batch1",
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	swapSeams(t)
	removeWorktree = func(string, string) error {
		return errors.New("simulated stuck worker worktree")
	}
	warnings = new([]string)
	warnf = func(format string, args ...any) {
		*warnings = append(*warnings, fmt.Sprintf(format, args...))
	}
	return repo, slug, lease.Slices[0].Branch, warnings
}

// TestCleanupWarnsOnFailedBranchCleanup proves Cleanup reports branch cleanup
// failures and keeps the lease so the orphans stay reachable for a rerun.
func TestCleanupWarnsOnFailedBranchCleanup(t *testing.T) {
	repo, slug, branchA, warnings := stuckBranchBatch(t)

	if _, err := Cleanup(repo, slug, true); err == nil || !strings.Contains(err.Error(), "slice-a") {
		t.Fatalf("cleanup must fail naming slice-a, got %v", err)
	}
	leasePath, err := LeasePath(repo, slug)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(leasePath); err != nil {
		t.Fatalf("lease must survive a failed cleanup, got %v", err)
	}
	// The rerun finds the lease and, with the stale registration pruned by the
	// first pass, completes the removal and clears it.
	if _, err := Cleanup(repo, slug, true); err != nil {
		t.Fatalf("rerun against the kept lease: %v", err)
	}
	if _, err := os.Stat(leasePath); !os.IsNotExist(err) {
		t.Fatalf("rerun should clear the lease, got %v", err)
	}
	if exists, err := branchExists(repo, branchA); err != nil || exists {
		t.Fatalf("rerun should delete %s: exists=%v err=%v", branchA, exists, err)
	}
	if joined := strings.Join(*warnings, "\n"); !strings.Contains(joined, branchA) {
		t.Fatalf("warnings should name the stuck branch %s, got:\n%s", branchA, joined)
	}
}

func TestControlCommitMessage(t *testing.T) {
	t.Parallel()
	got := controlCommitMessage("admin-report", "WIP(admin-report): SLICE-003 add CSV export")
	wantPrefix := "WIP(admin-report): add CSV export\n"
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("subject %q want prefix %q", got, wantPrefix)
	}
	if strings.Contains(got, "SLICE-") || strings.Contains(strings.ToLower(got), "slice-003") {
		t.Fatalf("message still names a slice id: %q", got)
	}
	got = controlCommitMessage("admin-report", "WIP(other): complete slice 3")
	if !strings.HasPrefix(got, "WIP(admin-report): complete\n") {
		t.Fatalf("foreign WIP prefix / slice N: %q", got)
	}
	got = controlCommitMessage("admin-report", "add this slice CSV")
	if !strings.HasPrefix(got, "WIP(admin-report): add CSV\n") {
		t.Fatalf("this slice: %q", got)
	}
	got = controlCommitMessage("admin-report", "keep array slice helper")
	if !strings.HasPrefix(got, "WIP(admin-report): keep array slice helper\n") {
		t.Fatalf("ordinary English stripped: %q", got)
	}
	got = controlCommitMessage("admin-report", "SLICE-001")
	if !strings.HasPrefix(got, "WIP(admin-report): land proven work\n") {
		t.Fatalf("empty summary after strip: %q", got)
	}
}

func TestCreateRefusesAncestorOverlap(t *testing.T) {
	repo, base := setupRepo(t)
	_, err := Create(CreateOpts{
		Root:    repo,
		Slug:    "demo",
		BatchID: "batch1",
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src"}},
			{ID: "slice-b", Paths: []string{"src/a.go"}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "overlap") {
		t.Fatalf("expected overlap refusal, got %v", err)
	}
}
