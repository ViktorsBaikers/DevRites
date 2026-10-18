package acceptance

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// maxInputFiles bounds one INPUTS expansion so a broad glob cannot turn a
// gate's reuse check into a repository-wide hash.
const maxInputFiles = 20000

// InputsDigest hashes every regular file an INPUTS spec names under project:
// whitespace-separated project-relative paths, directories (walked
// recursively, .git skipped) or globs. Each pattern must match at least one
// file, so a typo fails closed (the gate re-executes) instead of hashing
// nothing. The digest binds each sorted relative path to its content.
func InputsDigest(project, spec string) (string, error) {
	files := map[string]bool{}
	for _, pattern := range strings.Fields(spec) {
		clean := path.Clean(pattern)
		if path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(pattern, `\`) {
			return "", fmt.Errorf("INPUTS %q must be a project-relative slash path", pattern)
		}
		matches, err := filepath.Glob(filepath.Join(project, filepath.FromSlash(clean)))
		if err != nil {
			return "", fmt.Errorf("INPUTS %q: %w", pattern, err)
		}
		before := len(files)
		for _, match := range matches {
			if err := addInputFiles(project, match, files); err != nil {
				return "", err
			}
		}
		if len(files) == before {
			return "", fmt.Errorf("INPUTS %q matches no file", pattern)
		}
		if len(files) > maxInputFiles {
			return "", fmt.Errorf("INPUTS names more than %d files", maxInputFiles)
		}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	digest := sha256.New()
	for _, name := range names {
		file, err := os.Open(filepath.Join(project, filepath.FromSlash(name))) // #nosec G304 -- path enumerated under the project root
		if err != nil {
			return "", err
		}
		content := sha256.New()
		_, err = io.Copy(content, file)
		_ = file.Close()
		if err != nil {
			return "", err
		}
		fmt.Fprintf(digest, "%d:%s\x00%x\n", len(name), name, content.Sum(nil))
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func addInputFiles(project, start string, files map[string]bool) error {
	return filepath.WalkDir(start, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(project, name)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return errors.New("INPUTS path escapes the project")
		}
		files[filepath.ToSlash(rel)] = true
		return nil
	})
}
