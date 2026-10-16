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
	normalized := make([]SlicePaths, 0, len(ready))
	seen := make(map[string]struct{}, len(ready))
	for i, item := range ready {
		label := fmt.Sprintf("slice %d", i)
		if item.ID != "" {
			if err := validateSliceID(item.ID); err != nil {
				return nil, fmt.Errorf("%s: %w", label, err)
			}
			label = fmt.Sprintf("slice %q", item.ID)
		} else {
			return nil, fmt.Errorf("%s: missing id", label)
		}
		if _, ok := seen[item.ID]; ok {
			return nil, fmt.Errorf("duplicate slice id %q", item.ID)
		}
		seen[item.ID] = struct{}{}
		paths, err := validateSlicePaths(item.Paths, label, root)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, SlicePaths{ID: item.ID, Paths: paths})
	}

	running := make(map[string]bool, len(seed))
	for _, s := range seed {
		running[s.ID] = true
	}
	selected := append([]SlicePaths{}, seed...)
	for _, cand := range normalized {
		if len(selected) >= cap {
			break
		}
		if running[cand.ID] {
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
		ids[i] = item.ID
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
