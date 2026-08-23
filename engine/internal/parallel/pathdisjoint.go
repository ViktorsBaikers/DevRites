// Package parallel implements deterministic parallel-worktree control ops:
// path-disjoint checks, advisory leases, and git worktree create/integrate/cleanup.
package parallel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var winAbsRE = regexp.MustCompile(`(?i)^[A-Za-z]:[/\\]`)

// SlicePaths is one path-disjoint eligibility unit.
type SlicePaths struct {
	ID    string   `json:"id"`
	Paths []string `json:"paths"`
}

// NormalizePath mirrors scripts/check-path-disjoint.py: slash-normalize, reject
// empty/absolute/.. paths, and drop empty/"." segments.
func NormalizePath(raw string) (string, error) {
	path := strings.TrimSpace(strings.ReplaceAll(raw, `\`, "/"))
	if path == "" {
		return "", fmt.Errorf("empty path is not allowed")
	}
	if strings.HasPrefix(path, "/") || winAbsRE.MatchString(path) {
		return "", fmt.Errorf("path must be project-relative, not absolute: %q", raw)
	}
	parts := make([]string, 0, strings.Count(path, "/")+1)
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return "", fmt.Errorf("path must not contain '..': %q", raw)
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("empty path is not allowed")
	}
	return strings.Join(parts, "/"), nil
}

func validateSlicePaths(paths []string, label, root string) ([]string, error) {
	if paths == nil {
		return nil, fmt.Errorf("%s: paths must be a list", label)
	}
	normalized := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, raw := range paths {
		path, err := NormalizePath(raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", label, err)
		}
		if _, ok := seen[path]; ok {
			return nil, fmt.Errorf("%s: duplicate path %q", label, path)
		}
		seen[path] = struct{}{}
		normalized = append(normalized, path)
		if root != "" {
			full := filepath.Join(root, filepath.FromSlash(path))
			if info, err := os.Lstat(full); err == nil && info.Mode()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("%s: symlink path is not allowed: %q", label, path)
			}
		}
	}
	return normalized, nil
}

// CheckPathDisjoint returns slice ids when every pair of path sets is disjoint.
func CheckPathDisjoint(slices []SlicePaths, root string) ([]string, error) {
	if len(slices) < 2 {
		return nil, fmt.Errorf("need at least two slices to check path-disjoint eligibility")
	}
	owners := make(map[string][]string)
	ids := make([]string, 0, len(slices))
	for index, item := range slices {
		label := fmt.Sprintf("slice %d", index)
		if item.ID != "" {
			label = fmt.Sprintf("slice %q", item.ID)
			ids = append(ids, item.ID)
		} else {
			ids = append(ids, fmt.Sprintf("%d", index))
		}
		paths, err := validateSlicePaths(item.Paths, label, root)
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			owners[path] = append(owners[path], label)
		}
	}
	var overlaps []string
	for path, labels := range owners {
		if len(labels) > 1 {
			overlaps = append(overlaps, fmt.Sprintf("%q shared by %s", path, strings.Join(labels, ", ")))
		}
	}
	if len(overlaps) > 0 {
		sort.Strings(overlaps)
		return nil, fmt.Errorf("path sets overlap: %s", strings.Join(overlaps, "; "))
	}
	return ids, nil
}

// ParseSlicesJSON accepts {"slices":[...]} or a top-level slices array.
func ParseSlicesJSON(data []byte) ([]SlicePaths, error) {
	var asObject struct {
		Slices []SlicePaths `json:"slices"`
	}
	if err := json.Unmarshal(data, &asObject); err == nil && asObject.Slices != nil {
		return asObject.Slices, nil
	}
	var asList []SlicePaths
	if err := json.Unmarshal(data, &asList); err == nil {
		return asList, nil
	}
	return nil, fmt.Errorf(`input must be {"slices": [...]} or a top-level slices array`)
}
