package safepath

import (
	"errors"
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
// A dangling symlink is followed to its target: creating through it writes there.
func ResolveExisting(filename string) (string, error) {
	var tail []string
	for hops := 0; ; {
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
		if info, lerr := os.Lstat(filename); lerr == nil && info.Mode()&os.ModeSymlink != 0 {
			target, rerr := os.Readlink(filename)
			if rerr != nil {
				return "", rerr
			}
			if hops++; hops > 255 {
				return "", &os.PathError{Op: "resolve", Path: filename, Err: errors.New("too many symlinks")}
			}
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(filename), target)
			}
			filename = target
			continue
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
