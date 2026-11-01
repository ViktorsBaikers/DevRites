package parallel

import (
	"errors"
	"fmt"
	"os"
)

func fullyClean(worktree string) error {
	out, err := git(worktree, "status", "--porcelain", "--untracked-files=all", "--ignored=matching")
	if err != nil {
		return err
	}
	if out != "" {
		return fmt.Errorf("worktree has tracked, untracked, or ignored data: %s", worktree)
	}
	return nil
}

// acceptedWorker checks the ref even when its checkout has already disappeared.
// Control rewrites WIP commits, so exact transfer identity replaces ancestry.
func acceptedWorker(root, slug string, lease *Lease, sl LeaseSlice, registered map[string]bool) error {
	if sl.Branch != workerBranch(slug, lease.BatchID, sl.Workspace, sl.ID) || !ownedWorktreePath(root, lease.BatchID, sl) {
		return fmt.Errorf("worker identity is not owned: %s (%s)", sl.WorktreePath, sl.Branch)
	}
	accepted, err := revParse(root, sl.TransferCommit)
	if err != nil || sl.TransferCommit == "" {
		return fmt.Errorf("worker has no usable accepted transfer: %s", sl.Branch)
	}
	tip, err := revParse(root, "refs/heads/"+sl.Branch)
	if err != nil || tip != accepted {
		return fmt.Errorf("worker branch differs from accepted transfer %s: %s", accepted, sl.Branch)
	}
	if _, err := os.Lstat(sl.WorktreePath); os.IsNotExist(err) {
		if registered[resolvedPath(sl.WorktreePath)] {
			return fmt.Errorf("missing worker checkout remains registered; preserve its index and recovery metadata: %s", sl.WorktreePath)
		}
		return nil
	} else if err != nil {
		return err
	}
	if !registered[resolvedPath(sl.WorktreePath)] {
		return fmt.Errorf("worker path is not registered; leaving replacement untouched: %s", sl.WorktreePath)
	}
	ref, err := git(sl.WorktreePath, "symbolic-ref", "--quiet", "HEAD")
	if err != nil || ref != "refs/heads/"+sl.Branch {
		return fmt.Errorf("worker checkout switched branch: %s", sl.WorktreePath)
	}
	head, err := headSHA(sl.WorktreePath)
	if err != nil || head != accepted {
		return fmt.Errorf("worker checkout differs from accepted transfer: %s", sl.WorktreePath)
	}
	return fullyClean(sl.WorktreePath)
}

// Prune removes only unchanged integrated checkouts, not refs or lease rows.
// Worker-bound review/proof and source bookkeeping must finish before callers prune.
func Prune(root, slug string) (removed []string, err error) {
	err = withLeaseLock(root, slug, func() error {
		path, err := LeasePath(root, slug)
		if err != nil {
			return err
		}
		lease, err := ReadLease(path)
		if err != nil {
			return err
		}
		if err := normalizeLease(lease, slug); err != nil {
			return err
		}
		if _, err := controllerLease(root, slug); err != nil {
			return err
		}
		if _, err := recoverLanding(root, slug, path, lease, false, false); err != nil {
			return err
		}
		registered, err := registeredWorktrees(root)
		if err != nil {
			return err
		}
		var failures []error
		for _, sl := range lease.Slices {
			if sl.WrightStatus != WrightIntegrated {
				continue
			}
			if err := acceptedWorker(root, slug, lease, sl, registered); err != nil {
				failures = append(failures, fmt.Errorf("%s: %w", SliceKey(sl.Workspace, sl.ID), err))
				continue
			}
			if _, err := os.Lstat(sl.WorktreePath); os.IsNotExist(err) {
				continue
			}
			if err := removeWorktree(root, sl.WorktreePath); err != nil {
				failures = append(failures, fmt.Errorf("kept recovery path %s and ref %s: %w", sl.WorktreePath, sl.Branch, err))
				continue
			}
			removed = append(removed, sl.WorktreePath)
		}
		return errors.Join(failures...)
	})
	return removed, err
}

func prepareStaging(root, slug string, lease *Lease, path, branch string) error {
	claim := LeaseSlice{ID: "integrate", WorktreePath: path}
	if !ownedWorktreePath(root, lease.BatchID, claim) {
		return fmt.Errorf("staging path is not owned: %s", path)
	}
	registered, err := registeredWorktrees(root)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(path); err == nil {
		if !registered[resolvedPath(path)] {
			return fmt.Errorf("staging path is not registered; leaving replacement untouched: %s", path)
		}
		ref, err := git(path, "symbolic-ref", "--quiet", "HEAD")
		if err != nil || ref != "refs/heads/"+branch {
			return fmt.Errorf("staging checkout changed branch: %s", path)
		}
		if err := fullyClean(path); err != nil {
			return err
		}
		if err := removeWorktree(root, path); err != nil {
			return fmt.Errorf("staging checkout kept at %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	} else if registered[resolvedPath(path)] {
		return fmt.Errorf("missing staging checkout remains registered; preserve its metadata and recover explicitly: %s", path)
	}
	return deleteBranch(root, branch)
}
