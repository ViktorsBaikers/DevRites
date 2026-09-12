package parallel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
)

// Test seams: indirection over the real primitives so discriminating tests
// can inject transient failures without touching the filesystem.
var (
	writeLease     = WriteLease
	removeWorktree = worktreeRemove
	salvage        = salvageSlice
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

// withLeaseLock serializes lease read-modify-write across engine processes:
// the host may issue record-green/integrate calls concurrently, and a
// lock-free pair of them would lose one update.
func withLeaseLock(repoRoot, slug string, fn func() error) error {
	if err := validateSlug(slug); err != nil {
		return err
	}
	return state.WithFeatureLock(
		filepath.Join(repoRoot, devritespaths.DevritesRootName), slug, fn)
}

// Create path-disjoint-gates, creates 2-10 worktrees from base, and writes a running lease.
func Create(opts CreateOpts) (lease *Lease, err error) {
	err = withLeaseLock(opts.Root, opts.Slug, func() error {
		lease, err = createLocked(opts)
		return err
	})
	return lease, err
}

func createLocked(opts CreateOpts) (*Lease, error) {
	if err := validateSlug(opts.Slug); err != nil {
		return nil, err
	}
	// Store absolute worktree paths in the lease so cleanup's destructive ops
	// never depend on the caller's working directory.
	if abs, err := filepath.Abs(opts.Root); err == nil {
		opts.Root = abs
	}
	if err := validateBatchID(opts.BatchID); err != nil {
		return nil, err
	}
	if err := checkParallelSliceCount(len(opts.Slices)); err != nil {
		return nil, err
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
	ensureScratchExcluded(opts.Root)
	return lease, nil
}

// RecordGreen marks a slice green with its transfer commit.
func RecordGreen(repoRoot, slug, sliceID, commit string) (lease *Lease, err error) {
	err = withLeaseLock(repoRoot, slug, func() error {
		lease, err = recordGreenLocked(repoRoot, slug, sliceID, commit)
		return err
	})
	return lease, err
}

func recordGreenLocked(repoRoot, slug, sliceID, commit string) (*Lease, error) {
	leasePath, err := LeasePath(repoRoot, slug)
	if err != nil {
		return nil, err
	}
	lease, err := ReadLease(leasePath)
	if err != nil {
		return nil, err
	}
	if lease.Status != StatusRunning && lease.Status != StatusIntegrateFailed {
		return nil, fmt.Errorf("record-green requires status=running|integrate-failed (have %s)", lease.Status)
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

// Abort marks the lease aborted. Control is never rewound: a moved HEAD is
// external work, so divergence is reported, not reset.
func Abort(repoRoot, slug string) (lease *Lease, err error) {
	err = withLeaseLock(repoRoot, slug, func() error {
		lease, err = abortLocked(repoRoot, slug)
		return err
	})
	return lease, err
}

func abortLocked(repoRoot, slug string) (*Lease, error) {
	leasePath, err := LeasePath(repoRoot, slug)
	if err != nil {
		return nil, err
	}
	lease, err := ReadLease(leasePath)
	if err != nil {
		return nil, err
	}
	if lease.Status == StatusComplete {
		return nil, fmt.Errorf("abort refused: lease already complete — the batch landed in control; nothing to discard")
	}
	base, err := revParse(repoRoot, lease.BaseSHA)
	if err != nil {
		return nil, err
	}
	if err := ensureControlAtBase(repoRoot, base); err != nil {
		warnf("control not at base: %v", err)
	}
	lease.Status = StatusAborted
	if err := WriteLease(leasePath, lease); err != nil {
		return nil, err
	}
	return lease, nil
}

// ensureControlAtBase verifies the control tip still equals the batch base.
// It never mutates: a moved or dirty control belongs to the user, so callers
// only learn about divergence — they must not repair it by rewinding.
func ensureControlAtBase(repoRoot, base string) error {
	head, err := headSHA(repoRoot)
	if err != nil {
		return err
	}
	if head == base {
		return nil
	}
	return fmt.Errorf("control head %s moved past base %s; refusing to rewind external commits", head, base)
}

// IntegrateOpts configures staging integrate.
type IntegrateOpts struct {
	Root           string
	Slug           string
	ApplyToControl bool
}

// Integrate all-or-nothing applies sibling transfer commits onto a staging branch.
func Integrate(opts IntegrateOpts) (tip string, lease *Lease, err error) {
	err = withLeaseLock(opts.Root, opts.Slug, func() error {
		tip, lease, err = integrateLocked(opts)
		return err
	})
	return tip, lease, err
}

func integrateLocked(opts IntegrateOpts) (tip string, lease *Lease, err error) {
	leasePath, err := LeasePath(opts.Root, opts.Slug)
	if err != nil {
		return "", nil, err
	}
	lease, err = ReadLease(leasePath)
	if err != nil {
		return "", nil, err
	}
	if lease.Status != StatusRunning && lease.Status != StatusIntegrateFailed {
		return "", nil, fmt.Errorf("integrate requires status=running|integrate-failed (have %s)", lease.Status)
	}
	base, err := revParse(opts.Root, lease.BaseSHA)
	if err != nil {
		return "", nil, err
	}
	if err := ensureControlAtBase(opts.Root, base); err != nil {
		return "", nil, err
	}

	union := make([]string, 0, len(lease.Slices))
	ids := make([]string, 0, len(lease.Slices))
	for _, sl := range lease.Slices {
		union = append(union, sl.Paths...)
		ids = append(ids, sl.ID)
	}
	if opts.ApplyToControl {
		// FF into control must not silently absorb or clobber the user's
		// uncommitted work on slice paths. Non-overlapping dirty files are
		// left alone — they are not part of the batch.
		dirty, err := porcelainDirtyPaths(opts.Root, union)
		if err != nil {
			return "", nil, err
		}
		if dirty {
			return "", nil, fmt.Errorf("control has uncommitted changes on slice paths; commit or stash them before integrate --apply-to-control")
		}
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
		if err := ensureControlAtBase(opts.Root, base); err != nil {
			warnf("control diverged from base during integrate: %v", err)
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

		if err := cherryPickNoCommit(stageWT, base, tc); err != nil {
			cherryPickAbort(stageWT)
			return fail(fmt.Errorf("squash apply failed for slice %s: %w", sl.ID, err))
		}
	}

	msg := fmt.Sprintf("WIP(%s): parallel batch %s (%s)\n\n[devrites-context]\nslices: %s\nbase: %s",
		opts.Slug, lease.BatchID, strings.Join(ids, ", "), strings.Join(ids, ", "), base)
	if tip, err = stagePathsAndCommit(stageWT, msg, union); err != nil {
		return fail(fmt.Errorf("squash commit: %w", err))
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
		lease.Status = StatusComplete
		if err := WriteLease(leasePath, lease); err != nil {
			return "", nil, err
		}
	} else if err := ensureControlAtBase(opts.Root, base); err != nil {
		return fail(err)
	}
	// Without --apply-to-control the lease stays running: the squash commit
	// lives only on the integrate branch, and a running lease keeps a
	// non-forced cleanup from discarding the refs that hold it.
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

// Salvage records one worker branch kept by a non-complete cleanup: the
// rejected work stays reachable for mining instead of being discarded.
// Worktree is set when salvage itself failed and the worktree was left on
// disk as the only remaining copy.
type Salvage struct {
	SliceID  string `json:"slice_id"`
	Branch   string `json:"branch"`
	Commit   string `json:"commit"`
	Worktree string `json:"worktree,omitempty"`
}

// ownedWorktreePath reports whether the lease's worktree path is exactly the
// deterministic <scratch>/<batch>/<slice> location Create wrote. Anything else
// is not ours: cleanup must neither salvage into it nor delete it.
func ownedWorktreePath(repoRoot, batchID string, sl LeaseSlice) bool {
	got, err1 := filepath.Abs(sl.WorktreePath)
	want, err2 := filepath.Abs(WorkerWorktreePath(repoRoot, batchID, sl.ID))
	return err1 == nil && err2 == nil && got == want
}

// salvageSlice commits any uncommitted allowlisted changes in a worker
// worktree onto its slice branch so forced cleanup loses no wright work.
func salvageSlice(sl LeaseSlice, batchID string) (string, error) {
	if sl.WorktreePath == "" || len(sl.Paths) == 0 {
		return "", nil
	}
	if _, err := os.Stat(sl.WorktreePath); err != nil {
		// Worktree already gone; committed work stays reachable on the branch.
		return "", nil
	}
	dirty, err := porcelainDirtyPaths(sl.WorktreePath, sl.Paths)
	if err != nil {
		return "", err
	}
	if !dirty {
		return "", nil
	}
	return stagePathsAndCommit(sl.WorktreePath,
		fmt.Sprintf("devrites: salvage %s WIP (%s)", sl.ID, batchID), sl.Paths)
}

// Cleanup removes worker worktrees and clears the lease. A complete batch
// deletes the merged slice branches; a forced non-complete cleanup first
// salvages each sibling — uncommitted allowlisted changes become a commit on
// the slice branch — and keeps every branch whose tip moved past base as the
// durable rejected-work ref.
func Cleanup(repoRoot, slug string, force bool) (salvaged []Salvage, err error) {
	err = withLeaseLock(repoRoot, slug, func() error {
		salvaged, err = cleanupLocked(repoRoot, slug, force)
		return err
	})
	return salvaged, err
}

func cleanupLocked(repoRoot, slug string, force bool) ([]Salvage, error) {
	leasePath, err := LeasePath(repoRoot, slug)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(leasePath); os.IsNotExist(err) {
		return nil, nil
	}
	lease, err := ReadLease(leasePath)
	if err != nil {
		return nil, err
	}
	if lease.Status != StatusComplete && !force {
		return nil, fmt.Errorf("cleanup refuses status=%s without --force (non-complete cleanup salvages work onto slice branches)", lease.Status)
	}
	complete := lease.Status == StatusComplete
	base := ""
	if !complete {
		if b, err := revParse(repoRoot, lease.BaseSHA); err != nil {
			warnf("base sha %s: %v", lease.BaseSHA, err)
		} else {
			base = b
		}
	}
	var salvaged []Salvage
	keptWorktrees := map[string]bool{}
	for _, sl := range lease.Slices {
		salvageFailed := false
		// Only the deterministic scratch layout is ours to touch. A forged or
		// stale lease worktree_path is never salvaged into (git ops would run
		// in an arbitrary repo) and never removed (RemoveAll would delete an
		// arbitrary directory).
		owned := sl.WorktreePath != "" && ownedWorktreePath(repoRoot, lease.BatchID, sl)
		if sl.WorktreePath != "" && !owned {
			warnf("slice %s worktree_path %q is outside %s; leaving it untouched",
				sl.ID, sl.WorktreePath, ScratchRoot(repoRoot))
		}
		if !complete && owned {
			if _, err := salvage(sl, lease.BatchID); err != nil {
				warnf("salvage %s: %v", sl.ID, err)
				salvageFailed = true
				keptWorktrees[sl.ID] = true
			}
		}
		if owned && !salvageFailed {
			if err := removeWorktree(repoRoot, sl.WorktreePath); err != nil {
				warnf("worktree cleanup %s: %v", sl.WorktreePath, err)
			}
			if err := os.RemoveAll(sl.WorktreePath); err != nil {
				warnf("worktree dir cleanup %s: %v", sl.WorktreePath, err)
			}
		}
		if sl.Branch != "" && !ownedBranchName(slug, lease.BatchID, sl.Branch) {
			warnf("slice %s branch %q is outside devrites/parallel/%s/%s/; leaving it untouched",
				sl.ID, sl.Branch, slug, lease.BatchID)
			if salvageFailed {
				salvaged = append(salvaged, Salvage{SliceID: sl.ID, Worktree: sl.WorktreePath})
			}
			continue
		}
		if sl.Branch != "" {
			keep := salvageFailed
			tip := ""
			if !complete {
				if t, err := revParse(repoRoot, sl.Branch); err != nil {
					warnf("branch tip %s: %v", sl.Branch, err)
					keep = true // fail toward preservation
				} else {
					tip = t
					keep = keep || base == "" || tip != base
				}
			}
			if keep {
				s := Salvage{SliceID: sl.ID, Branch: sl.Branch, Commit: tip}
				if salvageFailed {
					s.Worktree = sl.WorktreePath
				}
				salvaged = append(salvaged, s)
			} else if err := deleteBranch(repoRoot, sl.Branch); err != nil {
				warnf("branch cleanup %s: %v", sl.Branch, err)
			}
		} else if salvageFailed {
			salvaged = append(salvaged, Salvage{SliceID: sl.ID, Worktree: sl.WorktreePath})
		}
	}
	ibranch := IntegrateBranchName(slug, lease.BatchID)
	if itip, err := revParse(repoRoot, ibranch); err == nil {
		keep := true
		if head, herr := headSHA(repoRoot); herr == nil {
			if anc, aerr := isAncestor(repoRoot, itip, head); aerr == nil && anc {
				keep = false
			}
		}
		if keep {
			salvaged = append(salvaged, Salvage{SliceID: "integrate", Branch: ibranch, Commit: itip})
		} else if err := deleteBranch(repoRoot, ibranch); err != nil {
			warnf("integrate branch cleanup %s: %v", ibranch, err)
		}
	}
	batchDir := filepath.Join(ScratchRoot(repoRoot), lease.BatchID)
	if len(keptWorktrees) == 0 {
		if err := os.RemoveAll(batchDir); err != nil {
			warnf("scratch cleanup: %v", err)
		}
	} else if entries, err := os.ReadDir(batchDir); err == nil {
		// A failed salvage keeps its worktree as the only copy — remove the
		// batch dir's other entries but never a kept slice dir.
		for _, e := range entries {
			if keptWorktrees[e.Name()] {
				continue
			}
			if err := os.RemoveAll(filepath.Join(batchDir, e.Name())); err != nil {
				warnf("scratch entry cleanup %s: %v", e.Name(), err)
			}
		}
	} else {
		warnf("scratch cleanup %s: %v", batchDir, err)
	}
	if _, err := git(repoRoot, "worktree", "prune"); err != nil {
		warnf("worktree prune: %v", err)
	}
	return salvaged, ClearLease(leasePath)
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
