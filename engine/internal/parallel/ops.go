package parallel

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	Root     string
	Slug     string
	BatchID  string
	BaseSHA  string
	Session  string
	Slices   []SlicePaths
	Capacity int
}

// withLeaseLock serializes lease read-modify-write across engine processes:
// the host may issue record-green/integrate calls concurrently, and a
// lock-free pair of them would lose one update.
func withLeaseLock(repoRoot, slug string, fn func() error) error {
	if err := validateSlug(slug); err != nil {
		return err
	}
	return WithControlLock(filepath.Join(repoRoot, devritespaths.DevritesRootName), func() error {
		return state.WithFeatureLock(
			filepath.Join(repoRoot, devritespaths.DevritesRootName), slug, fn)
	})
}

// Create validates and creates 1-10 isolated worktrees with a durable writer cap.
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
	if err := validateCapacity(opts.Capacity); err != nil {
		return nil, err
	}
	if opts.Capacity == 0 {
		opts.Capacity = MaxParallelSlices
	}
	if len(opts.Slices) > opts.Capacity {
		return nil, fmt.Errorf("create would make %d slices active; parallel allows at most %d", len(opts.Slices), opts.Capacity)
	}
	normalized, err := normalizeSlices(opts.Slug, opts.Slices, opts.Root)
	if err != nil {
		return nil, err
	}
	opts.Slices = normalized
	if err := validateSources(filepath.Join(opts.Root, devritespaths.DevritesRootName), opts.Slices); err != nil {
		return nil, err
	}
	if len(opts.Slices) >= 2 {
		if _, err := CheckPathDisjoint(opts.Slices, opts.Root); err != nil {
			return nil, err
		}
	}
	if _, err := controllerLease(opts.Root, opts.Slug); err != nil {
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

	slices, cleanupPartial, err := createWorktrees(opts.Root, opts.Slug, opts.BatchID, base, opts.Slices)
	if err != nil {
		return nil, err
	}

	lease := &Lease{
		BatchID:             opts.BatchID,
		CreatedAt:           NowUTC(),
		BaseSHA:             base,
		N:                   len(slices),
		Capacity:            opts.Capacity,
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

// createWorktrees makes one pending worker worktree and branch per slice from
// base, all or nothing: a failure removes what this call already made. The
// returned rollback undoes a success when the caller's lease write fails.
func createWorktrees(root, slug, batchID, base string, specs []SlicePaths) ([]LeaseSlice, func(), error) {
	made := make([]LeaseSlice, 0, len(specs))
	rollback := func() {
		// Worktrees first: git refuses to delete a branch still checked out.
		for _, sl := range made {
			if err := removeWorktree(root, sl.WorktreePath); err != nil {
				warnf("worktree cleanup %s: %v", sl.WorktreePath, err)
			}
		}
		for _, sl := range made {
			if err := deleteBranch(root, sl.Branch); err != nil {
				warnf("branch cleanup %s: %v", sl.Branch, err)
			}
			if ownedWorktreePath(root, batchID, sl) {
				if sl.Workspace != "" {
					_ = os.Remove(filepath.Dir(sl.WorktreePath))
					_ = os.Remove(filepath.Join(ScratchRoot(root), batchID, "_workspaces"))
				}
				_ = os.Remove(filepath.Join(ScratchRoot(root), batchID))
			}
		}
	}
	for _, sp := range specs {
		sl, err := createWorktree(root, slug, batchID, base, sp)
		if err != nil {
			rollback()
			return nil, nil, err
		}
		made = append(made, sl)
	}
	return made, rollback, nil
}

func createWorktree(root, slug, batchID, base string, sp SlicePaths) (LeaseSlice, error) {
	if sp.Workspace == "" && sp.ID == "integrate" {
		return LeaseSlice{}, fmt.Errorf("slice id %q is reserved for staging integration", sp.ID)
	}
	branch := workerBranch(slug, batchID, sp.Workspace, sp.ID)
	wt := workerWorktreePath(root, batchID, sp.Workspace, sp.ID)
	if _, err := os.Stat(wt); err == nil {
		return LeaseSlice{}, fmt.Errorf("worktree path already exists: %s", wt)
	}
	claim := LeaseSlice{ID: sp.ID, Workspace: sp.Workspace, WorktreePath: wt}
	if !ownedWorktreePath(root, batchID, claim) {
		return LeaseSlice{}, fmt.Errorf("worktree path is not owned: %s", wt)
	}
	if err := os.MkdirAll(filepath.Dir(wt), 0o755); err != nil {
		return LeaseSlice{}, err
	}
	if err := worktreeAdd(root, wt, branch, base); err != nil {
		return LeaseSlice{}, fmt.Errorf("worktree add %s: %w", SliceKey(sp.Workspace, sp.ID), err)
	}
	return LeaseSlice{
		ID:           sp.ID,
		Workspace:    sp.Workspace,
		Paths:        sp.Paths,
		WorktreePath: wt,
		Branch:       branch,
		WrightStatus: WrightPending,
	}, nil
}

// AddOpts configures worker refill or an explicit metadata-only capacity update.
type AddOpts struct {
	Root     string
	Slug     string
	Slices   []SlicePaths
	Capacity int
}

// Add refills freed slots of a running lease: each new slice gets a worktree
// from the current control tip, under the same path-disjoint and cap rules as
// the slices still active. Landed (integrated) slices no longer claim paths.
// An explicit Capacity with no Slices changes only the bound, not lease status.
func Add(opts AddOpts) (lease *Lease, err error) {
	err = withLeaseLock(opts.Root, opts.Slug, func() error {
		lease, err = addLocked(opts)
		return err
	})
	return lease, err
}

func addLocked(opts AddOpts) (*Lease, error) {
	if abs, err := filepath.Abs(opts.Root); err == nil {
		opts.Root = abs
	}
	if len(opts.Slices) == 0 && opts.Capacity == 0 {
		return nil, fmt.Errorf("add requires at least one slice or an explicit cap")
	}
	leasePath, err := LeasePath(opts.Root, opts.Slug)
	if err != nil {
		return nil, err
	}
	lease, err := ReadLease(leasePath)
	if err != nil {
		return nil, err
	}
	if err := normalizeLease(lease, opts.Slug); err != nil {
		return nil, err
	}
	if _, err := controllerLease(opts.Root, opts.Slug); err != nil {
		return nil, err
	}
	if err := validateLeaseSources(filepath.Join(opts.Root, devritespaths.DevritesRootName), lease); err != nil {
		return nil, err
	}
	if err := validateCapacity(opts.Capacity); err != nil {
		return nil, err
	}
	if opts.Capacity == 0 {
		opts.Capacity = leaseCapacity(lease)
	}
	if lease.Status != StatusRunning && lease.Status != StatusComplete {
		return nil, fmt.Errorf("add requires status=running|complete (have %s)", lease.Status)
	}
	head, err := revParse(opts.Root, expectedControl(lease))
	if err != nil {
		return nil, err
	}
	if err := ensureControlAtBase(opts.Root, head); err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(lease.Slices)+len(opts.Slices))
	for _, sl := range lease.Slices {
		seen[SliceKey(sl.Workspace, sl.ID)] = true
	}
	added, err := normalizeSlices(opts.Slug, opts.Slices, opts.Root)
	if err != nil {
		return nil, err
	}
	if err := validateSources(filepath.Join(opts.Root, devritespaths.DevritesRootName), added); err != nil {
		return nil, err
	}
	var addedPaths []string
	for _, sp := range added {
		key := SliceKey(sp.Workspace, sp.ID)
		if seen[key] {
			return nil, fmt.Errorf("slice id %q already in lease", key)
		}
		seen[key] = true
		addedPaths = append(addedPaths, sp.Paths...)
	}
	all := append(activeSlices(lease), added...)
	if len(all) > opts.Capacity {
		return nil, fmt.Errorf("add would make %d slices active; parallel allows at most %d", len(all), opts.Capacity)
	}
	if len(all) >= 2 {
		if _, err := CheckPathDisjoint(all, opts.Root); err != nil {
			return nil, err
		}
	}
	if len(added) == 0 {
		lease.Capacity = opts.Capacity
		if err := WriteLease(leasePath, lease); err != nil {
			return nil, err
		}
		return lease, nil
	}
	// The new worktree starts from control HEAD, so uncommitted user work on
	// its paths would be neither in the slice nor safe from its landing.
	dirty, err := porcelainDirtyPaths(opts.Root, addedPaths)
	if err != nil {
		return nil, err
	}
	if dirty {
		return nil, fmt.Errorf("control has uncommitted changes on new slice paths; stop for the human to resolve them (never stash or commit them), then rerun add")
	}

	made, rollback, err := createWorktrees(opts.Root, opts.Slug, lease.BatchID, head, added)
	if err != nil {
		return nil, err
	}
	for i := range made {
		made[i].BaseSHA = head
	}
	lease.Slices = append(lease.Slices, made...)
	lease.N = len(lease.Slices)
	lease.Capacity = opts.Capacity
	lease.Status = StatusRunning
	if err := WriteLease(leasePath, lease); err != nil {
		rollback()
		return nil, err
	}
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
	if err := normalizeLease(lease, slug); err != nil {
		return nil, err
	}
	sliceID, err = normalizeSliceRef(slug, sliceID)
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
		if SliceKey(lease.Slices[i].Workspace, lease.Slices[i].ID) == sliceID {
			if lease.Slices[i].WrightStatus == WrightIntegrated {
				// Re-greening a landed slice would land its work a second time.
				return nil, fmt.Errorf("slice %s already integrated", sliceID)
			}
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
	if err := normalizeLease(lease, slug); err != nil {
		return nil, err
	}
	if lease.Status == StatusComplete {
		return nil, fmt.Errorf("abort refused: lease already complete — the batch landed in control; nothing to discard")
	}
	expected, err := revParse(repoRoot, expectedControl(lease))
	if err != nil {
		return nil, err
	}
	if err := ensureControlAtBase(repoRoot, expected); err != nil {
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
	Root string
	Slug string
	// Slice, when set, lands only that slice (requires ApplyToControl);
	// otherwise every slice not yet integrated lands together.
	Slice          string
	ApplyToControl bool
}

// Integrate all-or-nothing applies transfer commits onto a staging branch
// cut from the expected control tip — every not-yet-integrated slice, or just
// opts.Slice — and with ApplyToControl fast-forwards control to it.
func Integrate(opts IntegrateOpts) (tip string, lease *Lease, err error) {
	err = withLeaseLock(opts.Root, opts.Slug, func() error {
		tip, lease, err = integrateLocked(opts)
		return err
	})
	return tip, lease, err
}

func integrateLocked(opts IntegrateOpts) (tip string, lease *Lease, err error) {
	if opts.Slice != "" && !opts.ApplyToControl {
		return "", nil, fmt.Errorf("integrate --slice requires --apply-to-control")
	}
	leasePath, err := LeasePath(opts.Root, opts.Slug)
	if err != nil {
		return "", nil, err
	}
	lease, err = ReadLease(leasePath)
	if err != nil {
		return "", nil, err
	}
	if err := normalizeLease(lease, opts.Slug); err != nil {
		return "", nil, err
	}
	if _, err := controllerLease(opts.Root, opts.Slug); err != nil {
		return "", nil, err
	}
	if opts.Slice != "" {
		opts.Slice, err = normalizeSliceRef(opts.Slug, opts.Slice)
		if err != nil {
			return "", nil, err
		}
	}
	if lease.Status != StatusRunning && lease.Status != StatusIntegrateFailed {
		return "", nil, fmt.Errorf("integrate requires status=running|integrate-failed (have %s)", lease.Status)
	}
	// Control may only have moved by this lease's own landings; anything else
	// is external work that integrate must neither rewind nor build on.
	expected, err := revParse(opts.Root, expectedControl(lease))
	if err != nil {
		return "", nil, err
	}
	if err := ensureControlAtBase(opts.Root, expected); err != nil {
		return "", nil, err
	}

	var targets []int
	for i, sl := range lease.Slices {
		switch {
		case opts.Slice == "" && sl.WrightStatus != WrightIntegrated,
			opts.Slice != "" && SliceKey(sl.Workspace, sl.ID) == opts.Slice:
			targets = append(targets, i)
		}
	}
	switch {
	case opts.Slice != "" && len(targets) == 0:
		return "", nil, fmt.Errorf("slice not in lease: %s", opts.Slice)
	case opts.Slice != "" && lease.Slices[targets[0]].WrightStatus == WrightIntegrated:
		return "", nil, fmt.Errorf("slice %s already integrated", opts.Slice)
	case len(targets) == 0:
		return "", nil, fmt.Errorf("no slices left to integrate")
	}

	union := make([]string, 0, len(targets))
	for _, i := range targets {
		union = append(union, lease.Slices[i].Paths...)
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
			return "", nil, fmt.Errorf("control has uncommitted changes on slice paths; stop for the human to resolve them (never stash or commit them), then rerun integrate --apply-to-control")
		}
	}

	for _, i := range targets {
		sl := lease.Slices[i]
		if sl.WrightStatus != WrightGreen {
			return "", nil, fmt.Errorf("slice %s is not green", SliceKey(sl.Workspace, sl.ID))
		}
		if sl.TransferCommit == "" {
			return "", nil, fmt.Errorf("slice %s missing transfer_commit", SliceKey(sl.Workspace, sl.ID))
		}
	}

	ibranch := IntegrateBranchName(opts.Slug, lease.BatchID)
	stageWT := filepath.Join(ScratchRoot(opts.Root), lease.BatchID, "integrate")
	_ = os.RemoveAll(stageWT)
	// A crashed prior attempt leaves the worktree registered; prune it so the
	// branch delete and worktree add below can succeed on retry.
	if _, err := git(opts.Root, "worktree", "prune"); err != nil {
		warnf("worktree prune: %v", err)
	}
	if exists, _ := branchExists(opts.Root, ibranch); exists {
		_ = deleteBranch(opts.Root, ibranch)
	}
	if err := os.MkdirAll(filepath.Dir(stageWT), 0o755); err != nil {
		return "", nil, err
	}
	if err := worktreeAdd(opts.Root, stageWT, ibranch, expected); err != nil {
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
		if err := ensureControlAtBase(opts.Root, expected); err != nil {
			warnf("control diverged from base during integrate: %v", err)
		}
		return "", lease, reason
	}

	for _, i := range targets {
		sl := lease.Slices[i]
		key := SliceKey(sl.Workspace, sl.ID)
		// Each slice is proven against the commit its worktree was cut from,
		// which for a refill is a later control tip than the batch base.
		base, err := revParse(opts.Root, sliceBase(lease, sl))
		if err != nil {
			return fail(fmt.Errorf("slice %s base: %w", key, err))
		}
		tc, err := revParse(opts.Root, sl.TransferCommit)
		if err != nil {
			return fail(fmt.Errorf("slice %s transfer: %w", key, err))
		}
		ok, err := isAncestor(opts.Root, base, tc)
		if err != nil {
			return fail(err)
		}
		if !ok {
			return fail(fmt.Errorf("transfer_commit for %s is not a descendant of base", key))
		}
		got, err := diffNames(opts.Root, base, tc)
		if err != nil {
			return fail(err)
		}
		want := append([]string(nil), sl.Paths...)
		sort.Strings(want)
		sort.Strings(got)
		if !pathListsEqual(want, got) {
			return fail(fmt.Errorf("exact-path proof failed for slice %s: want=%v got=%v", key, want, got))
		}
		for _, p := range got {
			if p == ".devrites" || strings.HasPrefix(p, ".devrites/") {
				return fail(fmt.Errorf("path proof failed: .devrites path in transfer: %s", p))
			}
		}

		if err := cherryPickNoCommit(stageWT, base, tc); err != nil {
			cherryPickAbort(stageWT)
			return fail(fmt.Errorf("apply failed for slice %s: %w", key, err))
		}
		subj, err := commitSubject(opts.Root, tc)
		if err != nil {
			return fail(fmt.Errorf("transfer subject for %s: %w", key, err))
		}
		source := sl.Workspace
		if source == "" {
			source = opts.Slug
		}
		if _, err = stagePathsAndCommit(stageWT, controlCommitMessage(source, subj), sl.Paths); err != nil {
			return fail(fmt.Errorf("control commit for %s: %w", key, err))
		}
		cherryPickQuit(stageWT)
	}
	if tip, err = headSHA(stageWT); err != nil {
		return fail(fmt.Errorf("integrate tip: %w", err))
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
		for _, i := range targets {
			lease.Slices[i].WrightStatus = WrightIntegrated
		}
		lease.ControlTip = tip
		// A per-slice landing also clears an earlier integrate-failed: the
		// lease is healthy again while other slices are still in flight.
		lease.Status = StatusRunning
		if len(activeSlices(lease)) == 0 {
			lease.Status = StatusComplete
		}
		if err := WriteLease(leasePath, lease); err != nil {
			return "", nil, err
		}
	} else if err := ensureControlAtBase(opts.Root, expected); err != nil {
		return fail(err)
	}
	// Without --apply-to-control the lease stays running: the integrate
	// commits live only on the integrate branch, and a running lease keeps a
	// non-forced cleanup from discarding the refs that hold them.
	return tip, lease, nil
}

var (
	sliceTaskID       = regexp.MustCompile(`(?i)\bSLICE-\d+\b`)
	sliceProcessMark  = regexp.MustCompile(`(?i)\bthis slice\b|\bslice\s+\d+\b`)
	existingWIPPrefix = regexp.MustCompile(`(?i)^WIP\([^)]+\):\s*`)
)

// controlCommitMessage is the local unpushed subject/body landed on control.
// Ship collapse still keys off the WIP(<slug>): prefix; the rest is a human
// summary with DevRites process marks stripped (SLICE-###, "this slice",
// "slice N"). Ordinary English such as "array slice" is left intact.
func controlCommitMessage(slug, transferSubject string) string {
	prefix := fmt.Sprintf("WIP(%s): ", slug)
	line := strings.TrimSpace(strings.SplitN(transferSubject, "\n", 2)[0])
	line = existingWIPPrefix.ReplaceAllString(line, "")
	line = sliceTaskID.ReplaceAllString(line, "")
	line = sliceProcessMark.ReplaceAllString(line, "")
	line = strings.TrimSpace(strings.Trim(line, ":-"))
	line = strings.Join(strings.Fields(line), " ")
	if line == "" {
		line = "land proven work"
	}
	return prefix + line + "\n\n[devrites-context]\n"
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
//
// Lexical equality is necessary but not sufficient. filepath.Abs cleans a
// path without resolving it, so when a component under the repository is a
// symlink both sides carry the identical spelling while the directory a
// cleanup would delete lives somewhere else. No component from the repository
// root down to the worktree directory may be a symlink.
func ownedWorktreePath(repoRoot, batchID string, sl LeaseSlice) bool {
	got, err1 := filepath.Abs(sl.WorktreePath)
	want, err2 := filepath.Abs(workerWorktreePath(repoRoot, batchID, sl.Workspace, sl.ID))
	root, err3 := filepath.Abs(repoRoot)
	if err1 != nil || err2 != nil || err3 != nil || got != want {
		return false
	}
	for p := got; p != root && p != filepath.Dir(p); p = filepath.Dir(p) {
		fi, err := os.Lstat(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return false
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return false
		}
	}
	return true
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
		fmt.Sprintf("devrites: salvage WIP (%s)", batchID), sl.Paths)
}

// Cleanup removes worker worktrees and clears the lease. A complete batch, and
// any slice already integrated, deletes the merged slice branches; a forced
// non-complete cleanup first salvages each other sibling — uncommitted
// allowlisted changes become a commit on the slice branch — and keeps every
// branch whose tip moved past that slice's base as the durable rejected-work
// ref.
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
	if err := normalizeLease(lease, slug); err != nil {
		return nil, err
	}
	if lease.Status != StatusComplete && !force {
		return nil, fmt.Errorf("cleanup refuses status=%s without --force (non-complete cleanup salvages work onto slice branches)", lease.Status)
	}
	complete := lease.Status == StatusComplete
	// Ownership comes from what git registered, read before anything is
	// removed, never from the lease's own claim.
	registered, err := registeredWorktrees(repoRoot)
	if err != nil {
		return nil, err
	}
	var salvaged []Salvage
	var failed, stranded, unconfirmed []string
	for _, sl := range lease.Slices {
		key := SliceKey(sl.Workspace, sl.ID)
		// A slice integrate already landed is done like a complete batch: its
		// work is on control, so there is nothing to salvage or keep.
		landed := complete || sl.WrightStatus == WrightIntegrated
		salvageFailed := false
		// Only the deterministic scratch layout is ours to touch. A forged or
		// stale lease worktree_path is never salvaged into (git ops would run
		// in an arbitrary repo) and never removed (RemoveAll would delete an
		// arbitrary directory).
		inScratch := sl.WorktreePath != "" && ownedWorktreePath(repoRoot, lease.BatchID, sl)
		owned := inScratch && sl.Branch == workerBranch(slug, lease.BatchID, sl.Workspace, sl.ID) && createdWorktree(registered, sl.WorktreePath)
		if owned {
			if _, err := os.Stat(sl.WorktreePath); err == nil {
				branch, err := git(sl.WorktreePath, "symbolic-ref", "--quiet", "--short", "HEAD")
				if err != nil || branch != sl.Branch {
					owned = false
					warnf("slice %s no longer has its registered worker branch; leaving %s untouched", key, sl.WorktreePath)
				}
			}
		}
		if sl.WorktreePath != "" && !owned {
			if inScratch {
				warnf("slice %s worktree_path %q is not a worktree this lease owns and git registered; leaving it untouched",
					key, sl.WorktreePath)
			} else {
				warnf("slice %s worktree_path %q is outside %s; leaving it untouched",
					key, sl.WorktreePath, ScratchRoot(repoRoot))
			}
		}
		if !landed && owned {
			if _, err := salvage(sl, lease.BatchID); err != nil {
				warnf("salvage %s: %v", key, err)
				salvageFailed = true
			}
		}
		if owned && !salvageFailed {
			if err := removeWorktree(repoRoot, sl.WorktreePath); err != nil {
				warnf("worktree cleanup %s: %v", sl.WorktreePath, err)
			}
			if err := os.RemoveAll(sl.WorktreePath); err != nil {
				warnf("worktree dir cleanup %s: %v", sl.WorktreePath, err)
				failed = append(failed, key)
				// git can unregister a worktree and still exit non-zero, so the
				// exit code says nothing about registration; ask git again.
				if reg, rerr := registeredWorktrees(repoRoot); rerr != nil {
					unconfirmed = append(unconfirmed, sl.WorktreePath)
				} else if !createdWorktree(reg, sl.WorktreePath) {
					// A rerun refuses an unregistered directory as not
					// created by us, so it will not delete it.
					stranded = append(stranded, sl.WorktreePath)
				}
			}
		}
		if sl.Branch != "" && sl.Branch != workerBranch(slug, lease.BatchID, sl.Workspace, sl.ID) {
			warnf("slice %s branch %q does not match its worker namespace; leaving it untouched",
				key, sl.Branch)
			if salvageFailed {
				salvaged = append(salvaged, Salvage{SliceID: key, Worktree: sl.WorktreePath})
			}
			continue
		}
		if sl.Branch != "" {
			keep := salvageFailed
			tip := ""
			if !landed {
				base := ""
				if b, err := revParse(repoRoot, sliceBase(lease, sl)); err != nil {
					warnf("base sha %s: %v", sliceBase(lease, sl), err)
				} else {
					base = b
				}
				if t, err := revParse(repoRoot, sl.Branch); err != nil {
					warnf("branch tip %s: %v", sl.Branch, err)
					keep = true // fail toward preservation
				} else {
					tip = t
					keep = keep || base == "" || tip != base
				}
			}
			if keep {
				s := Salvage{SliceID: key, Branch: sl.Branch, Commit: tip}
				if salvageFailed {
					s.Worktree = sl.WorktreePath
				}
				salvaged = append(salvaged, s)
			} else if err := deleteBranch(repoRoot, sl.Branch); err != nil {
				warnf("branch cleanup %s: %v", sl.Branch, err)
				// A refused worktree is not removed and the lease is cleared once
				// nothing else failed, so the branch stays behind; the warning
				// above names it.
				if owned || sl.WorktreePath == "" {
					failed = append(failed, key)
				}
			}
		} else if salvageFailed {
			salvaged = append(salvaged, Salvage{SliceID: key, Worktree: sl.WorktreePath})
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
			failed = append(failed, "integrate")
		}
	}
	batchDir := filepath.Join(ScratchRoot(repoRoot), lease.BatchID)
	claim := LeaseSlice{ID: "integrate", WorktreePath: WorkerWorktreePath(repoRoot, lease.BatchID, "integrate")}
	if ownedWorktreePath(repoRoot, lease.BatchID, claim) {
		// Remove only empty source namespaces; unknown or unsalvaged files stay.
		for _, sl := range lease.Slices {
			if sl.Workspace != "" && ownedWorktreePath(repoRoot, lease.BatchID, sl) {
				_ = os.Remove(filepath.Join(batchDir, "_workspaces", sl.Workspace))
			}
		}
		namespace := filepath.Join(batchDir, "_workspaces")
		if info, err := os.Lstat(namespace); err == nil && info.IsDir() {
			_ = os.Remove(namespace)
		}
		// Fails harmlessly while anything was left behind.
		_ = os.Remove(batchDir)
	}
	if _, err := git(repoRoot, "worktree", "prune"); err != nil {
		warnf("worktree prune: %v", err)
	}
	if len(failed) > 0 {
		// The lease is the only index of what is left on disk; keep it so a
		// rerun can retry the removals, except for directories git no longer
		// registers: a rerun leaves those in place.
		msg := fmt.Sprintf("cleanup incomplete for %s; lease kept, rerun cleanup after fixing the warnings above", strings.Join(uniqueInOrder(failed), ", "))
		if len(stranded) > 0 {
			msg += fmt.Sprintf("; git no longer registers %s, a rerun will not delete it, remove it by hand", strings.Join(stranded, ", "))
		}
		if len(unconfirmed) > 0 {
			msg += fmt.Sprintf("; could not confirm whether git still registers %s, remove it by hand if a rerun leaves it", strings.Join(unconfirmed, ", "))
		}
		return salvaged, errors.New(msg)
	}
	return salvaged, ClearLease(leasePath)
}

// StatusReport returns a human-readable lease status.
func StatusReport(repoRoot, slug string) (string, error) {
	leasePath, err := LeasePath(repoRoot, slug)
	if err != nil {
		return "", err
	}
	found, err := controllerLease(repoRoot, "")
	if err != nil {
		return "", err
	}
	if found == nil {
		return fmt.Sprintf("status: no lease at %s\n", leasePath), nil
	}
	lease := found.lease
	leasePath, err = LeasePath(found.root, found.controller)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "lease: %s\n", leasePath)
	fmt.Fprintf(&b, "controller: %s\n", found.controller)
	fmt.Fprintf(&b, "capacity: %d\n", leaseCapacity(lease))
	fmt.Fprintf(&b, "status: %s\n", lease.Status)
	fmt.Fprintf(&b, "base_sha: %s\n", lease.BaseSHA)
	if lease.ControlTip != "" {
		fmt.Fprintf(&b, "control_tip: %s\n", lease.ControlTip)
	}
	for _, sl := range lease.Slices {
		tc := sl.TransferCommit
		if tc == "" {
			tc = "-"
		}
		fmt.Fprintf(&b, "slice %s: wright=%s wt=%s branch=%s transfer=%s",
			SliceKey(sl.Workspace, sl.ID), sl.WrightStatus, sl.WorktreePath, sl.Branch, tc)
		if sl.BaseSHA != "" {
			fmt.Fprintf(&b, " base=%s", sl.BaseSHA)
		}
		b.WriteString("\n")
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

// uniqueInOrder drops repeats, keeping each entry's first position.
func uniqueInOrder(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
