package fsutil

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

// syncDir flushes a directory's entries so a completed rename survives a crash.
// It is a variable so tests can observe the call.
var syncDir = func(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	d, err := os.Open(dir) // #nosec G304 -- dir is the parent of an engine-managed path
	if err != nil {
		return err
	}
	err = d.Sync()
	_ = d.Close()
	if errors.Is(err, syscall.EINVAL) || errors.Is(err, errors.ErrUnsupported) {
		return nil
	}
	return err
}

// WriteFileAtomic writes path by first writing a sibling temp file and then
// renaming it into place.
func WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("atomic write %s: %w", path, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("atomic write %s: %w", path, err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("atomic write %s: %w", path, err)
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("atomic write %s: %w", path, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("atomic write %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("atomic write %s: %w", path, err)
	}
	if err := replaceFile(tmpName, path); err != nil {
		return fmt.Errorf("atomic write %s: %w", path, err)
	}
	cleanup = false
	if err := syncDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("atomic write %s: %w", path, err)
	}
	return nil
}

// WriteFileAtomicIn is WriteFileAtomic confined to root: every directory
// component of rel is resolved relative to the open root, so a component
// swapped for a symlink that leaves the root fails instead of redirecting
// the write.
func WriteFileAtomicIn(root *os.Root, rel string, data []byte, perm fs.FileMode) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("atomic write %s: %w", filepath.Join(root.Name(), rel), err)
		}
	}()
	dir := filepath.Dir(rel)
	if err := root.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var tmpName string
	var tmp *os.File
	for range 100 {
		var nonce [8]byte
		_, _ = rand.Read(nonce[:]) // never fails since go 1.24
		tmpName = filepath.Join(dir, fmt.Sprintf(".%s.tmp-%d", filepath.Base(rel), binary.BigEndian.Uint64(nonce[:])))
		tmp, err = root.OpenFile(tmpName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if !errors.Is(err, fs.ErrExist) {
			break
		}
	}
	if err != nil {
		return err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = root.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := root.Rename(tmpName, rel); err != nil {
		return err
	}
	cleanup = false
	return syncDirIn(root, dir)
}

func syncDirIn(root *os.Root, dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	d, err := root.Open(dir)
	if err != nil {
		return err
	}
	err = d.Sync()
	_ = d.Close()
	if errors.Is(err, syscall.EINVAL) || errors.Is(err, errors.ErrUnsupported) {
		return nil
	}
	return err
}
