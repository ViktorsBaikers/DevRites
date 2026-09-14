package parallel

import (
	"fmt"
	"strings"
)

// SelectGreedy returns the largest plan-order pairwise-disjoint subset of ready
// whose size is ≤ cap. ready must already be dependency-satisfied and in plan
// order. cap is 1–MaxParallelSlices; 1 is a legal one-slice round, not a latch.
func SelectGreedy(cap int, ready []SlicePaths, root string) ([]SlicePaths, error) {
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

	selected := make([]SlicePaths, 0, min(cap, len(normalized)))
	for _, cand := range normalized {
		if len(selected) >= cap {
			break
		}
		if len(selected) == 0 {
			selected = append(selected, cand)
			continue
		}
		trial := append(append([]SlicePaths{}, selected...), cand)
		if _, err := CheckPathDisjoint(trial, root); err != nil {
			if strings.Contains(err.Error(), "overlap") {
				continue
			}
			return nil, err
		}
		selected = trial
	}
	return selected, nil
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

func formatSelect(selected []SlicePaths) string {
	ids := make([]string, len(selected))
	for i, item := range selected {
		ids[i] = item.ID
	}
	return fmt.Sprintf("select: n_eff=%d mode=%s slices=%s", len(selected), selectMode(len(selected)), strings.Join(ids, ","))
}
