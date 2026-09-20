package safepath

import (
	"os"
	"path/filepath"
)

// WithinResolved reports whether candidate remains within parent after resolving
// symlinks in every existing path prefix. It also handles not-yet-created files.
func WithinResolved(candidate, parent string) bool {
	if candidateInfo, err := os.Stat(candidate); err == nil {
		if parentInfo, err := os.Stat(parent); err == nil && os.SameFile(candidateInfo, parentInfo) {
			return true
		}
	}
	resolvedCandidate, err := ResolveExisting(candidate)
	if err != nil {
		return false
	}
	resolvedParent, err := ResolveExisting(parent)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(resolvedParent, resolvedCandidate)
	return err == nil && rel != ".." &&
		!filepath.IsAbs(rel) && !hasParentPrefix(rel)
}

// ResolveExisting resolves symlinks in the longest existing path prefix, then
// appends any missing tail without allowing it to change the resolved parent.
func ResolveExisting(filename string) (string, error) {
	var tail []string
	for {
		resolved, err := filepath.EvalSymlinks(filename)
		if err == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return filepath.Clean(resolved), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(filename)
		if parent == filename {
			return "", err
		}
		tail = append(tail, filepath.Base(filename))
		filename = parent
	}
}

func hasParentPrefix(rel string) bool {
	return rel == ".." || len(rel) > 3 && rel[:3] == ".."+string(filepath.Separator)
}
