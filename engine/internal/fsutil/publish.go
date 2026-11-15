package fsutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PublishDirNoReplace atomically publishes an exclusively owned staging tree.
// Both paths must be clean, relative paths under root with existing parents.
// Managed components must not be symlinks or reparse points. Callers must keep
// the stage and parent identities stable (including against directory moves)
// until this returns; native parent handles prevent path-symlink redirection.
//
// Native no-replace support is required: Linux renameat2, Darwin renameatx_np,
// or Windows FileRenameInformation, on amd64/arm64. Unsupported kernels and
// filesystems fail closed. No ordinary-rename fallback is used. Sync failures
// are never ignored. Windows requires NTFS (other filesystems can acknowledge
// directory flushes without persisting entries) and successful directory flushes.
//
// published is true once the rename succeeds, even if durability fails. Never
// delete the destination on that error: recover its identity and retry syncing.
// This function neither creates parents nor removes staging data on failure.
func PublishDirNoReplace(root *os.Root, from, to string) (published bool, err error) {
	return publishDirNoReplace(root, from, to, nativePublishRename, nativePublishSync)
}

type publishRenameFunc func(sourceParent, stage, targetParent *os.File, from, to string) error

func publishDirNoReplace(root *os.Root, from, to string, rename publishRenameFunc, sync func(*os.File) error) (published bool, err error) {
	defer func() {
		if err != nil {
			if published {
				err = fmt.Errorf("published %q but durability is uncertain; preserve target and retry directory/parent sync: %w", to, err)
			} else {
				err = fmt.Errorf("publish %q to %q: %w", from, to, err)
			}
		}
	}()
	if err := validatePublishPath(from); err != nil {
		return false, err
	}
	if err := validatePublishPath(to); err != nil {
		return false, err
	}
	if from == to {
		return false, fmt.Errorf("source and destination are identical: %w", os.ErrExist)
	}
	sourceParent, err := openPublishParent(root, filepath.Dir(from))
	if err != nil {
		return false, err
	}
	defer sourceParent.Close()
	targetParent, err := openPublishParent(root, filepath.Dir(to))
	if err != nil {
		return false, err
	}
	defer targetParent.Close()
	stage, err := nativePublishOpen(sourceParent, filepath.Base(from), true, true)
	if err != nil {
		return false, err
	}
	defer stage.Close()
	if err := syncPublishTree(stage, sync); err != nil {
		return false, err
	}
	// Preflight parent flush support before making anything discoverable.
	if err := syncPublishParents(sourceParent, targetParent, sync); err != nil {
		return false, err
	}
	if err := rename(sourceParent, stage, targetParent, filepath.Base(from), filepath.Base(to)); err != nil {
		return false, err
	}
	return true, syncPublishParents(sourceParent, targetParent, sync)
}

// SyncPublishedDir confirms durability of the existing tree and its parent.
// A retry must propagate errors; existence alone is not proof of durability.
// This does not certify removal from a former staging parent, whose identity
// is unknown here. Callers recovering that failure must sync that parent too.
func SyncPublishedDir(root *os.Root, rel string) error {
	return syncPublishedDir(root, rel, nativePublishSync)
}

func syncPublishedDir(root *os.Root, rel string, sync func(*os.File) error) error {
	if err := validatePublishPath(rel); err != nil {
		return err
	}
	parent, err := openPublishParent(root, filepath.Dir(rel))
	if err != nil {
		return err
	}
	defer parent.Close()
	dir, err := nativePublishOpen(parent, filepath.Base(rel), true, false)
	if err != nil {
		return err
	}
	defer dir.Close()
	if err := syncPublishTree(dir, sync); err != nil {
		return fmt.Errorf("sync published %q: %w", rel, err)
	}
	if err := sync(parent); err != nil {
		return fmt.Errorf("sync published parent of %q: %w", rel, err)
	}
	return nil
}

func validatePublishPath(rel string) error {
	if !filepath.IsLocal(rel) || rel == "." || filepath.Clean(rel) != rel {
		return fmt.Errorf("invalid publication path %q", rel)
	}
	for _, name := range strings.Split(filepath.ToSlash(rel), "/") {
		if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "\\:\x00") || strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
			return fmt.Errorf("invalid publication component %q", name)
		}
	}
	return nil
}

func openPublishParent(root *os.Root, rel string) (*os.File, error) {
	anchor, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	// Reopen through the handle: Windows flushes require write access.
	parent, err := nativePublishOpen(anchor, ".", true, false)
	_ = anchor.Close()
	if err != nil {
		return nil, err
	}
	if rel == "." {
		return parent, nil
	}
	for _, name := range strings.Split(filepath.ToSlash(rel), "/") {
		next, err := nativePublishOpen(parent, name, true, false)
		_ = parent.Close()
		if err != nil {
			return nil, err
		}
		parent = next
	}
	return parent, nil
}

func syncPublishTree(dir *os.File, sync func(*os.File) error) error {
	entries, err := dir.ReadDir(-1)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := validatePublishPath(entry.Name()); err != nil {
			return err
		}
		if entry.Type() & ^os.ModeDir != 0 {
			return fmt.Errorf("publication entry %q is not a regular file or directory", entry.Name())
		}
		child, err := nativePublishOpen(dir, entry.Name(), entry.IsDir(), false)
		if err != nil {
			return err
		}
		info, err := child.Stat()
		if err == nil {
			switch {
			case info.IsDir():
				err = syncPublishTree(child, sync)
			case info.Mode().IsRegular():
				err = sync(child)
			default:
				err = fmt.Errorf("publication entry %q is not regular", entry.Name())
			}
		}
		closeErr := child.Close()
		if err != nil || closeErr != nil {
			return errors.Join(err, closeErr)
		}
	}
	return sync(dir)
}

func syncPublishParents(source, target *os.File, sync func(*os.File) error) error {
	// Attempt both even if one fails; neither failure may be hidden by a retry.
	sourceErr := sync(source)
	targetErr := sync(target)
	if sourceErr != nil {
		sourceErr = fmt.Errorf("sync staging parent: %w", sourceErr)
	}
	if targetErr != nil {
		targetErr = fmt.Errorf("sync destination parent: %w", targetErr)
	}
	return errors.Join(sourceErr, targetErr)
}
