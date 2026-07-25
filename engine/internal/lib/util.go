package lib

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// gitToplevel returns the absolute path of the git working tree containing dir,
// or "" when dir is not inside a repository. The git-backed gates
// (test-integrity and reconcile) skip rather than block when it is empty.
func gitToplevel(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return filepath.Clean(strings.TrimSpace(string(out)))
}

// gitDiffNames lists the paths that differ, relative to gitRoot. With one ref it
// diffs the working tree against that ref; with two it diffs the two tree objects.
func gitDiffNames(gitRoot string, refs ...string) []string {
	args := append([]string{"-C", gitRoot, "diff", "--name-only"}, refs...)
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil
	}
	return splitLinesNoTrailing(out)
}

// stripASCIISpace removes every whitespace character from s (used to normalise an
// ACTIVE pointer before comparing it to a slug).
func stripASCIISpace(s string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(spaceChars, r) {
			return -1
		}
		return r
	}, s)
}

// isAllDigits reports whether s is a non-empty run of ASCII digits.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// featureFile is the path to a leaf file inside a feature's directory:
// <root>/features/<slug>/<name>.
func featureFile(root, slug, name string) string {
	return filepath.Join(featureDir(root, slug), name)
}

// slugOrActive returns the slug named in args[0], falling back to the active
// feature when none is given.
func slugOrActive(root string, args []string) string {
	if slug := argAt(args, 0); slug != "" {
		return slug
	}
	return activeSlug(root)
}
