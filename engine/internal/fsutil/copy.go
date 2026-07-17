package fsutil

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

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
	return nil
}

// FileModTime returns a regular file's modification time in whole seconds.
func FileModTime(path string) (int64, bool) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return 0, false
	}
	return info.ModTime().Unix(), true
}

// NewestModTime returns the newest modification time among regular files.
func NewestModTime(paths ...string) (newest int64, ok bool) {
	for _, path := range paths {
		if modified, exists := FileModTime(path); exists && (!ok || modified > newest) {
			newest, ok = modified, true
		}
	}
	return newest, ok
}

// CopyTree recursively copies src to dst, creating parent directories and
// overwriting files. A missing src is not an error.
func CopyTree(src, dst string) error {
	info, err := os.Stat(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("copy %s: %w", src, err)
	}
	if !info.IsDir() {
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("copy %s: %w", src, err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("copy to %s: %w", dst, err)
		}
		return os.WriteFile(dst, data, 0o644)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("copy %s: %w", src, err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("copy to %s: %w", dst, err)
	}
	for _, e := range entries {
		if err := CopyTree(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
