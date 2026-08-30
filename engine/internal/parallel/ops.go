package parallel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Test seams: indirection over the real primitives so discriminating tests
// can inject transient failures without touching the filesystem.
var (
	writeLease     = WriteLease
	removeWorktree = worktreeRemove
	warnf          = func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, "warning: parallel: "+format+"\n", args...)
	}
)

// CreateOpts configures parallel worktree creation.
type CreateOpts struct {
	Root    string
	Slug    string
	BatchID string
	BaseSHA string
	Session string
	Slices  []SlicePaths
}

// Create path-disjoint-gates, creates <=3 worktrees from base, and writes a running lease.
func Create(opts CreateOpts) (*Lease, error) {
	if err := validateSlug(opts.Slug); err != nil {
		return nil, err
	}
	if err := validateBatchID(opts.BatchID); err != nil {
		return nil, err
	}
	if len(opts.Slices) < 2 || len(opts.Slices) > 3 {
		return nil, fmt.Errorf("create requires 2 or 3 slices (got %d)", len(opts.Slices))
	}
	if _, err := CheckPathDisjoint(opts.Slices, opts.Root); err != nil {
		return nil, err
	}
	base, err := revParse(opts.Root, opts.BaseSHA)
	if err != nil {
		return nil, fmt.Errorf("base sha: %w", err)
	}
	leasePath, err := LeasePath(opts.Root, opts.Slug)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(leasePath); err == nil {
		return nil, fmt.Errorf("lease already exists: %s", leasePath)
	}

	session := opts.Session
	if session == "" {
		session = fmt.Sprintf("pid:%d", os.Getpid())
	}

	createdPaths := make([]string, 0, len(opts.Slices))
	createdBranches := make([]string, 0, len(opts.Slices))
	cleanupPartial := func() {
		for _, p := range createdPaths {
			if err := removeWorktree(opts.Root, p); err != nil {
				warnf("worktree cleanup %s: %v", p, err)
			}
		}
		for _, b := range createdBranches {
			if err := deleteBranch(opts.Root, b); err != nil {
				warnf("branch cleanup %s: %v", b, err)
			}
		}
	}
	slices := make([]LeaseSlice, 0, len(opts.Slices))
	for _, sp := range opts.Slices {
		if err := validateSliceID(sp.ID); err != nil {
			cleanupPartial()
			return nil, err
		}
		paths, err := validateSlicePaths(sp.Paths, fmt.Sprintf("slice %q", sp.ID), opts.Root)
		if err != nil {
			cleanupPartial()
			return nil, err
		}
		branch := WorkerBranch(opts.Slug, opts.BatchID, sp.ID)
		wt := WorkerWorktreePath(opts.Root, opts.BatchID, sp.ID)
		if _, err := os.Stat(wt); err == nil {
			cleanupPartial()
			return nil, fmt.Errorf("worktree path already exists: %s", wt)
		}
		if err := os.MkdirAll(filepath.Dir(wt), 0o755); err != nil {
			cleanupPartial()
			return nil, err
		}
		if err := worktreeAdd(opts.Root, wt, branch, base); err != nil {
			cleanupPartial()
			return nil, fmt.Errorf("worktree add %s: %w", sp.ID, err)
		}
		createdPaths = append(createdPaths, wt)
		createdBranches = append(createdBranches, branch)
		slices = append(slices, LeaseSlice{
			ID:           sp.ID,
			Paths:        paths,
			WorktreePath: wt,
			Branch:       branch,
			WrightStatus: WrightPending,
		})
	}

	lease := &Lease{
		BatchID:             opts.BatchID,
		CreatedAt:           NowUTC(),
		BaseSHA:             base,
		N:                   len(slices),
		Status:              StatusRunning,
		ControlPIDOrSession: session,
		Slices:              slices,
	}
	if err := WriteLease(leasePath, lease); err != nil {
		cleanupPartial()
		return nil, err
	}
	return lease, nil
}

// RecordGreen marks a slice green with its transfer commit.
func RecordGreen(repoRoot, slug, sliceID, commit string) (*Lease, error) {
	leasePath, err := LeasePath(repoRoot, slug)
	if err != nil {
		return nil, err
	}
	lease, err := ReadLease(leasePath)
	if err != nil {
		return nil, err
	}
	if lease.Status != StatusRunning {
		return nil, fmt.Errorf("record-green requires status=running (have %s)", lease.Status)
	}
	sha, err := revParse(repoRoot, commit)
	if err != nil {
		return nil, fmt.Errorf("transfer commit: %w", err)
	}
	found := false
	for i := range lease.Slices {
		if lease.Slices[i].ID == sliceID {
			lease.Slices[i].WrightStatus = WrightGreen
			lease.Slices[i].TransferCommit = sha
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("slice not in lease: %s", sliceID)
	}
	if err := WriteLease(leasePath, lease); err != nil {
		return nil, err
	}
	return lease, nil
}

// Abort marks the lease aborted and restores control tip to base_sha.
func Abort(repoRoot, slug string, force bool) (*Lease, error) {
	leasePath, err := LeasePath(repoRoot, slug)
	if err != nil {
		return nil, err
	}
	lease, err := ReadLease(leasePath)
	if err != nil {
		return nil, err
	}
	base, err := revParse(repoRoot, lease.BaseSHA)
	if err != nil {
		return nil, err
	}
	if err := ensureControlAtBase(repoRoot, base, force); err != nil {
		return nil, err
	}
	lease.Status = StatusAborted
	if err := WriteLease(leasePath, lease); err != nil {
		return nil, err
	}
	return lease, nil
}

func ensureControlAtBase(repoRoot, base string, force bool) error {
	head, err := headSHA(repoRoot)
	if err != nil {
		return err
	}
	if head == base {
		return nil
	}
	dirty, err := porcelainDirty(repoRoot)
	if err != nil {
		return err
	}
	if dirty && !force {
		return fmt.Errorf("control tree dirty; refuse reset to base (pass --force to override)")
	}
	if err := resetHard(repoRoot, base); err != nil {
		return err
	}
	head, err = headSHA(repoRoot)
	if err != nil {
		return err
	}
	if head != base {
		return fmt.Errorf("failed to leave control at base %s", base)
	}
	return nil
}

// IntegrateOpts configures staging integrate.
type IntegrateOpts struct {
	Root           string
	Slug           string
	ApplyToControl bool
	Force          bool
}

// Integrate all-or-nothing applies sibling transfer commits onto a staging branch.
func Integrate(opts IntegrateOpts) (tip string, lease *Lease, err error) {
	leasePath, err := LeasePath(opts.Root, opts.Slug)
	if err != nil {
		return "", nil, err
	}
	lease, err = ReadLease(leasePath)
	if err != nil {
		return "", nil, err
	}
	if lease.Status != StatusRunning {
		return "", nil, fmt.Errorf("integrate requires status=running (have %s)", lease.Status)
	}
	base, err := revParse(opts.Root, lease.BaseSHA)
	if err != nil {
		return "", nil, err
	}
	if err := ensureControlAtBase(opts.Root, base, opts.Force); err != nil {
		return "", nil, err
	}

	for _, sl := range lease.Slices {
		if sl.WrightStatus != WrightGreen {
			return "", nil, fmt.Errorf("slice %s is not green", sl.ID)
		}
		if sl.TransferCommit == "" {
			return "", nil, fmt.Errorf("slice %s missing transfer_commit", sl.ID)
		}
	}

	ibranch := IntegrateBranchName(opts.Slug, lease.BatchID)
	stageWT := filepath.Join(ScratchRoot(opts.Root), lease.BatchID, "integrate")
	_ = os.RemoveAll(stageWT)
	if exists, _ := branchExists(opts.Root, ibranch); exists {
		_ = deleteBranch(opts.Root, ibranch)
	}
	if err := os.MkdirAll(filepath.Dir(stageWT), 0o755); err != nil {
		return "", nil, err
	}
	if err := worktreeAdd(opts.Root, stageWT, ibranch, base); err != nil {
		return "", nil, fmt.Errorf("staging worktree: %w", err)
	}

	fail := func(reason error) (string, *Lease, error) {
		if err := removeWorktree(opts.Root, stageWT); err != nil {
			warnf("staging worktree cleanup: %v", err)
		}
		if err := deleteBranch(opts.Root, ibranch); err != nil {
			warnf("integrate branch cleanup: %v", err)
		}
		if _, err := git(opts.Root, "worktree", "prune"); err != nil {
			warnf("worktree prune: %v", err)
		}
		lease.Status = StatusIntegrateFailed
		if err := writeLease(leasePath, lease); err != nil {
			// The integrate-failed transition must survive a transient write
			// failure: a stale running lease still blocks create and stays
			// retryable, but the honest state must be recorded when possible.
			if retryErr := writeLease(leasePath, lease); retryErr != nil {
				warnf("lease %s still status=%s (write failed twice: %v)",
					leasePath, StatusRunning, retryErr)
			}
		}
		if err := ensureControlAtBase(opts.Root, base, true); err != nil {
			warnf("control reset to base: %v", err)
		}
		return "", lease, reason
	}

	for _, sl := range lease.Slices {
		tc, err := revParse(opts.Root, sl.TransferCommit)
		if err != nil {
			return fail(fmt.Errorf("slice %s transfer: %w", sl.ID, err))
		}
		ok, err := isAncestor(opts.Root, base, tc)
		if err != nil {
			return fail(err)
		}
		if !ok {
			return fail(fmt.Errorf("transfer_commit for %s is not a descendant of base", sl.ID))
		}
		got, err := diffNames(opts.Root, base, tc)
		if err != nil {
			return fail(err)
		}
		want := append([]string(nil), sl.Paths...)
		sort.Strings(want)
		sort.Strings(got)
		if !pathListsEqual(want, got) {
			return fail(fmt.Errorf("exact-path proof failed for slice %s: want=%v got=%v", sl.ID, want, got))
		}
		for _, p := range got {
			if p == ".devrites" || strings.HasPrefix(p, ".devrites/") {
				return fail(fmt.Errorf("path proof failed: .devrites path in transfer: %s", p))
			}
		}

		stageHead, err := headSHA(stageWT)
		if err != nil {
			return fail(err)
		}
		canFF, err := isAncestor(opts.Root, stageHead, tc)
		if err != nil {
			return fail(err)
		}
		if canFF {
			if err := mergeFFOnly(stageWT, tc); err != nil {
				return fail(fmt.Errorf("ff-only integrate failed for slice %s: %w", sl.ID, err))
			}
		} else {
			if err := cherryPickRange(stageWT, base, tc); err != nil {
				cherryPickAbort(stageWT)
				return fail(fmt.Errorf("cherry-pick replay failed for slice %s: %w", sl.ID, err))
			}
		}
	}

	tip, err = headSHA(stageWT)
	if err != nil {
		return fail(err)
	}
	if err := removeWorktree(opts.Root, stageWT); err != nil {
		if _, statErr := os.Stat(stageWT); statErr == nil {
			// The directory is genuinely stuck; failing now keeps the lease in
			// integrate-failed and avoids compounding into un-cleanable branch
			// state. Retrying integrate re-attempts the same removals.
			return fail(fmt.Errorf("staging worktree could not be removed; retry integrate after cleanup: %w", err))
		}
		warnf("staging worktree cleanup: %v", err)
	}
	if _, err := git(opts.Root, "worktree", "prune"); err != nil {
		warnf("worktree prune: %v", err)
	}

	if opts.ApplyToControl {
		if err := mergeFFOnly(opts.Root, tip); err != nil {
			return fail(fmt.Errorf("control fast-forward to integrate tip failed: %w", err))
		}
	} else if err := ensureControlAtBase(opts.Root, base, opts.Force); err != nil {
		return fail(err)
	}

	lease.Status = StatusComplete
	if err := WriteLease(leasePath, lease); err != nil {
		return "", nil, err
	}
	return tip, lease, nil
}

func pathListsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Cleanup removes worker worktrees/branches and clears the lease (complete or --force).
func Cleanup(repoRoot, slug string, force bool) error {
	leasePath, err := LeasePath(repoRoot, slug)
	if err != nil {
		return err
	}
	if _, err := os.Stat(leasePath); os.IsNotExist(err) {
		return nil
	}
	lease, err := ReadLease(leasePath)
	if err != nil {
		return err
	}
	if lease.Status != StatusComplete && !force {
		return fmt.Errorf("cleanup refuses status=%s without --force (abort keeps workers for diagnosis)", lease.Status)
	}
	for _, sl := range lease.Slices {
		if sl.WorktreePath != "" {
			if err := removeWorktree(repoRoot, sl.WorktreePath); err != nil {
				warnf("worktree cleanup %s: %v", sl.WorktreePath, err)
			}
			if err := os.RemoveAll(sl.WorktreePath); err != nil {
				warnf("worktree dir cleanup %s: %v", sl.WorktreePath, err)
			}
		}
		if sl.Branch != "" {
			if err := deleteBranch(repoRoot, sl.Branch); err != nil {
				warnf("branch cleanup %s: %v", sl.Branch, err)
			}
		}
	}
	ibranch := IntegrateBranchName(slug, lease.BatchID)
	if err := deleteBranch(repoRoot, ibranch); err != nil {
		warnf("integrate branch cleanup %s: %v", ibranch, err)
	}
	if err := os.RemoveAll(filepath.Join(ScratchRoot(repoRoot), lease.BatchID)); err != nil {
		warnf("scratch cleanup: %v", err)
	}
	if _, err := git(repoRoot, "worktree", "prune"); err != nil {
		warnf("worktree prune: %v", err)
	}
	return ClearLease(leasePath)
}

// StatusReport returns a human-readable lease status.
func StatusReport(repoRoot, slug string) (string, error) {
	leasePath, err := LeasePath(repoRoot, slug)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(leasePath); os.IsNotExist(err) {
		return fmt.Sprintf("status: no lease at %s\n", leasePath), nil
	}
	lease, err := ReadLease(leasePath)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "lease: %s\n", leasePath)
	fmt.Fprintf(&b, "status: %s\n", lease.Status)
	fmt.Fprintf(&b, "base_sha: %s\n", lease.BaseSHA)
	for _, sl := range lease.Slices {
		tc := sl.TransferCommit
		if tc == "" {
			tc = "-"
		}
		fmt.Fprintf(&b, "slice %s: wright=%s wt=%s branch=%s transfer=%s\n",
			sl.ID, sl.WrightStatus, sl.WorktreePath, sl.Branch, tc)
	}
	if head, err := headSHA(repoRoot); err == nil {
		fmt.Fprintf(&b, "control_HEAD: %s\n", head)
	}
	return b.String(), nil
}

// LeaseJSON returns indented lease JSON.
func LeaseJSON(lease *Lease) (string, error) {
	b, err := json.MarshalIndent(lease, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
