package parallel

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/devrites/devrites/internal/state"
)

// WithControlLock serializes controller admission and control-tip mutations.
// devritesRoot is explicit; callers take this lock before any feature lock.
func WithControlLock(devritesRoot string, fn func() error) error {
	devritesRoot = filepath.Clean(devritesRoot)
	if err := validateControlRoot(devritesRoot); err != nil {
		return err
	}
	dir := devritesRoot
	root := filepath.Dir(devritesRoot)
	common, err := commonGitDir(root)
	if err != nil {
		return err
	}
	if common != "" {
		dir = common
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return state.WithLock(filepath.Join(dir, "parallel.lock"), fn)
}

func commonGitDir(root string) (string, error) {
	common, err := git(root, "rev-parse", "--git-common-dir")
	if err == nil {
		if !filepath.IsAbs(common) {
			common = filepath.Join(root, common)
		}
		return resolvedPath(common), nil
	}
	dir, absErr := filepath.Abs(root)
	if absErr != nil {
		return "", absErr
	}
	for {
		if _, statErr := os.Lstat(filepath.Join(dir, ".git")); !os.IsNotExist(statErr) {
			return "", fmt.Errorf("parallel repository discovery: %w", err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

func validateControlRoot(root string) error {
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("parallel control root is not a safe directory: %s", root)
	}
	return nil
}

type repositoryLease struct {
	root       string
	controller string
	lease      *Lease
}

// controlLeases reads explicit paths, never the ambient workspace override.
func controlLeases(devritesRoot string) ([]repositoryLease, error) {
	devritesRoot = filepath.Clean(devritesRoot)
	if err := validateControlRoot(devritesRoot); err != nil {
		return nil, err
	}
	root := filepath.Dir(devritesRoot)
	roots := map[string]bool{resolvedPath(devritesRoot): true}
	common, err := commonGitDir(root)
	if err != nil {
		return nil, err
	}
	if common != "" {
		registered, err := registeredWorktrees(root)
		if err != nil {
			return nil, err
		}
		for path := range registered {
			roots[filepath.Join(path, ".devrites")] = true
		}
	}
	ordered := make([]string, 0, len(roots))
	for path := range roots {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	var leases []repositoryLease
	for _, control := range ordered {
		work := filepath.Join(control, "work")
		for _, path := range []string{control, work} {
			if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("parallel workspace path is a symlink: %s", path)
			} else if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
		}
		entries, err := os.ReadDir(work)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
				continue
			}
			path := filepath.Join(work, entry.Name(), "parallel-lease.md")
			info, err := os.Lstat(path)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if err := validateSlug(entry.Name()); err != nil {
				return nil, err
			}
			if entry.Type()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				return nil, fmt.Errorf("parallel lease path is not an owned regular file: %s", path)
			}
			lease, err := ReadLease(path)
			if err != nil {
				return nil, fmt.Errorf("controller %s: %w", entry.Name(), err)
			}
			if err := normalizeLease(lease, entry.Name()); err != nil {
				return nil, err
			}
			leases = append(leases, repositoryLease{filepath.Dir(control), entry.Name(), lease})
		}
	}
	return leases, nil
}

func controllerLease(root, controller string) (*repositoryLease, error) {
	leases, err := controlLeases(filepath.Join(root, ".devrites"))
	if err != nil {
		return nil, err
	}
	if len(leases) > 1 {
		return nil, fmt.Errorf("multiple retained repository controller leases; cleanup explicitly before admission")
	}
	if len(leases) == 0 {
		return nil, nil
	}
	found := &leases[0]
	if controller != "" && (found.controller != controller || found.root != resolvedPath(root)) {
		return nil, fmt.Errorf("repository controller lease retained by %s at %s; use its explicit root and --slug for mutations", found.controller, found.root)
	}
	return found, nil
}

// WorkspaceLeased protects both controller and source workspaces until cleanup.
// devritesRoot is explicit; closeout holds WithControlLock through its rename.
func WorkspaceLeased(devritesRoot, workspace string) (bool, error) {
	if err := validateSlug(workspace); err != nil {
		return false, err
	}
	leases, err := controlLeases(devritesRoot)
	if err != nil {
		return false, err
	}
	for _, found := range leases {
		if found.controller == workspace {
			return true, nil
		}
		for _, sl := range found.lease.Slices {
			if sl.Workspace == workspace {
				return true, nil
			}
		}
	}
	return false, nil
}

func validateSources(devritesRoot string, slices []SlicePaths) error {
	seen := make(map[string]bool)
	for _, sp := range slices {
		if sp.Workspace == "" || seen[sp.Workspace] {
			continue
		}
		seen[sp.Workspace] = true
		if err := validateSource(devritesRoot, sp.Workspace); err != nil {
			return err
		}
	}
	return nil
}

func validateSource(devritesRoot, workspace string) error {
	if workspace == "" {
		return nil
	}
	path := devritesRoot
	for index, component := range []string{"", "work", workspace, state.LedgerFile} {
		if component != "" {
			path = filepath.Join(path, component)
		}
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("source workspace %s is not live: %w", workspace, err)
		}
		if (index == 3 && !info.Mode().IsRegular()) || (index != 3 && !info.IsDir()) {
			return fmt.Errorf("source workspace %s has an unsafe live path: %s", workspace, path)
		}
	}
	return nil
}

func validateLeaseSources(devritesRoot string, lease *Lease) error {
	seen := make(map[string]bool)
	for _, sl := range lease.Slices {
		if seen[sl.Workspace] {
			continue
		}
		if err := validateSource(devritesRoot, sl.Workspace); err != nil {
			return err
		}
		seen[sl.Workspace] = true
	}
	return nil
}
