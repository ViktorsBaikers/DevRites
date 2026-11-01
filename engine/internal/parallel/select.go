package parallel

import (
	"errors"
	"fmt"
	"strings"
)

// checkPathDisjoint is a seam so tests can inject trial-check outcomes.
var checkPathDisjoint = CheckPathDisjoint

// SelectGreedy returns the largest plan-order pairwise-disjoint subset of ready
// whose size is ≤ cap. ready must already be dependency-satisfied and in plan
// order. cap is 1–MaxParallelSlices; 1 is a legal one-slice round, not a latch.
func SelectGreedy(cap int, ready []SlicePaths, root string) ([]SlicePaths, error) {
	return SelectGreedySeeded(cap, nil, ready, root)
}

// SelectGreedySeeded is SelectGreedy with seed slices already running: they
// occupy cap slots and claim their paths, a ready candidate with a seed's id
// is skipped, and only the new picks are returned.
func SelectGreedySeeded(cap int, seed, ready []SlicePaths, root string) ([]SlicePaths, error) {
	if cap < 1 || cap > MaxParallelSlices {
		return nil, fmt.Errorf("cap must be 1-%d (got %d)", MaxParallelSlices, cap)
	}
	normalized, err := normalizeSlices("", ready, root)
	if err != nil {
		return nil, err
	}
	seed, err = normalizeSlices("", seed, root)
	if err != nil {
		return nil, err
	}

	running := make(map[string]bool, len(seed))
	for _, s := range seed {
		running[SliceKey(s.Workspace, s.ID)] = true
	}
	selected := append([]SlicePaths{}, seed...)
	for _, cand := range normalized {
		if len(selected) >= cap {
			break
		}
		if running[SliceKey(cand.Workspace, cand.ID)] {
			continue
		}
		if len(selected) == 0 {
			selected = append(selected, cand)
			continue
		}
		trial := append(append([]SlicePaths{}, selected...), cand)
		if _, err := checkPathDisjoint(trial, root); err != nil {
			if errors.Is(err, ErrPathOverlap) {
				continue
			}
			return nil, err
		}
		selected = trial
	}
	return selected[len(seed):], nil
}

// PlanReady returns the plan-order ready set for `select --from-plan` (root is
// the project root). main sets it: the task-graph reader lives in lib, which
// imports this package.
var PlanReady func(root, slug string) ([]SlicePaths, error)

// WorkspaceReady supplies native-authorized candidates and per-workspace notices.
// main wires this hook from lib, which owns planning and dependency readiness.
var WorkspaceReady func(root, controller, target string) ([]SlicePaths, []string, error)

// SelectForSlug seeds every retained repository slot, regardless of source.
// Bare candidates belong to slug; returned foreign identities stay qualified
// relative to the retained controller. An omitted slug means controller-local.
func SelectForSlug(root, slug string, capN int, ready []SlicePaths) (eligible, picks []SlicePaths, running int, err error) {
	if capN < 1 || capN > MaxParallelSlices {
		return nil, nil, -1, fmt.Errorf("cap must be 1-%d (got %d)", MaxParallelSlices, capN)
	}
	if slug != "" {
		if err := validateSlug(slug); err != nil {
			return nil, nil, -1, err
		}
	}
	running = -1
	var seed []SlicePaths
	controller := slug
	var found *repositoryLease
	if root != "" || slug != "" {
		found, err = controllerLease(root, "")
		if err != nil {
			return nil, nil, -1, err
		}
	}
	candidates := make([]SlicePaths, len(ready))
	copy(candidates, ready)
	if found != nil {
		if found.lease.PendingLanding != nil {
			return nil, nil, -1, fmt.Errorf("pending landing blocks admission; recover controller %s at %s with integrate --apply-to-control or abort", found.controller, found.root)
		}
		controller = found.controller
		if slug != "" && slug != controller {
			for i := range candidates {
				if candidates[i].Workspace == "" {
					candidates[i].Workspace = slug
				}
			}
		}
		seed = activeSlices(found.lease)
		running = len(seed)
		if capacity := leaseCapacity(found.lease); capN > capacity {
			capN = capacity
		}
	}
	eligible, err = normalizeSlices(controller, candidates, root)
	if err != nil {
		return nil, nil, -1, err
	}
	if found != nil {
		named := make(map[string]bool, len(found.lease.Slices))
		for _, sl := range found.lease.Slices {
			named[SliceKey(sl.Workspace, sl.ID)] = true
		}
		filtered := eligible[:0]
		for _, sp := range eligible {
			if !named[SliceKey(sp.Workspace, sp.ID)] {
				filtered = append(filtered, sp)
			}
		}
		eligible = filtered
	}
	picks, err = SelectGreedySeeded(capN, seed, eligible, root)
	return eligible, picks, running, err
}

func selectMode(n int) string {
	switch {
	case n <= 0:
		return "none"
	case n == 1:
		return "serial"
	default:
		return "parallel"
	}
}

// formatSelect prints the select line. running < 0 means no lease seeded the
// selection; otherwise new picks join that many running slices, so the mode
// counts them too and the line reports running=<k>.
func formatSelect(picks []SlicePaths, running int) string {
	ids := make([]string, len(picks))
	for i, item := range picks {
		ids[i] = SliceKey(item.Workspace, item.ID)
	}
	mode := selectMode(len(picks))
	if running > 0 && len(picks) > 0 {
		mode = selectMode(len(picks) + running)
	}
	line := fmt.Sprintf("select: n_eff=%d mode=%s slices=%s", len(picks), mode, strings.Join(ids, ","))
	if running >= 0 {
		line += fmt.Sprintf(" running=%d", running)
	}
	return line
}
