package parallel

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	gitOk(t, wt, "commit", "-m", "slice "+marker)
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
	origLease  func(string, *Lease) error
	origRemove func(string, string) error
	origWarn   func(string, ...any)
}

func swapSeams(t *testing.T) {
	t.Helper()
	s := seamRestore{origLease: writeLease, origRemove: removeWorktree, origWarn: warnf}
	t.Cleanup(func() { writeLease, removeWorktree, warnf = s.origLease, s.origRemove, s.origWarn })
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

// TestCleanupWarnsOnFailedBranchCleanup proves Cleanup reports branch cleanup
// failures instead of erasing the lease while orphans remain.
func TestCleanupWarnsOnFailedBranchCleanup(t *testing.T) {
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
	branchA := lease.Slices[0].Branch

	swapSeams(t)
	// Removal "fails" without removing; Cleanup's own RemoveAll then deletes
	// the worktree directory while the stale git registration still lists the
	// branch as checked out, so deleteBranch refuses and must be reported.
	removeWorktree = func(string, string) error {
		return errors.New("simulated stuck worker worktree")
	}
	var warnings []string
	warnf = func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}

	if err := Cleanup(repo, slug, true); err != nil {
		t.Fatalf("cleanup continues after reporting failures: %v", err)
	}
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, branchA) {
		t.Fatalf("warnings should name the stuck branch %s, got:\n%s", branchA, joined)
	}
	leasePath, err := LeasePath(repo, slug)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(leasePath); !os.IsNotExist(err) {
		t.Fatalf("cleanup should still clear the lease, got %v", err)
	}
}
