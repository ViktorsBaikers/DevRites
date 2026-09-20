package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/rootfacts"
)

// ResolveRoot returns the .devrites directory to operate on. A non-empty
// override (e.g. from DEVRITES_ROOT) may name either the project root or its
// .devrites directory; otherwise the nearest ancestor .devrites is used.
func ResolveRoot(override string) (string, error) {
	facts, err := rootfacts.Resolve(override)
	if err != nil {
		return "", err
	}
	return facts.PhysicalRoot, nil
}

// featureDir is the canonical per-feature state directory under root.
func featureDir(root, slug string) string {
	return devritespaths.FeatureDir(root, slug)
}

// ListFeatures returns sorted slugs for canonical work/ directories that carry
// the state.md ledger. Missing work/ yields an empty list, not an error.
func ListFeatures(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "work"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list features: %w", err)
	}
	var slugs []string
	for _, entry := range entries {
		if entry.IsDir() && isFeatureDir(filepath.Join(root, "work", entry.Name())) {
			slugs = append(slugs, entry.Name())
		}
	}
	sort.Strings(slugs)
	return slugs, nil
}

// isFeatureDir reports whether dir carries the canonical working-state ledger.
func isFeatureDir(dir string) bool {
	return regularFileExists(filepath.Join(dir, LedgerFile))
}

// IsFeatureDir exposes the runtime discovery predicate to repository validators.
func IsFeatureDir(dir string) bool { return isFeatureDir(dir) }

func regularFileExists(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

func firstPhaseWord(value string) string {
	word := strings.ToLower(strings.TrimSpace(value))
	if i := strings.IndexAny(word, " 	"); i > 0 {
		word = word[:i]
	}
	return word
}

// blankOrHash reports whether a trimmed line carries no meaningful content: it
// is empty or starts with '#' (a Markdown ATX heading).
func blankOrHash(trimmed string) bool {
	return trimmed == "" || strings.HasPrefix(trimmed, "#")
}

// stripFrontmatter removes a well-formed leading frontmatter block when
// determining whether an artifact contains substantive body content. The
// closing fence must be a line that is exactly `---`.
func stripFrontmatter(raw []byte) []byte {
	norm := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if !strings.HasPrefix(norm, "---\n") {
		return raw
	}
	lines := strings.Split(norm[len("---\n"):], "\n")
	end := -1
	for i, line := range lines {
		if strings.TrimRight(line, " \t") == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return raw
	}
	return []byte(strings.Join(lines[end+1:], "\n"))
}
