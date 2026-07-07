package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ResolveRoot returns the .devrites directory to operate on. A non-empty
// override (e.g. from DEVRITES_ROOT) is used verbatim; otherwise the nearest
// ancestor .devrites of the working directory is used.
func ResolveRoot(override string) (string, error) {
	if override != "" {
		info, err := os.Stat(override)
		if err != nil {
			return "", fmt.Errorf("DEVRITES_ROOT %q: %w", override, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("DEVRITES_ROOT %q is not a directory", override)
		}
		return override, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; {
		cand := filepath.Join(dir, ".devrites")
		if info, err := os.Stat(cand); err == nil && info.IsDir() {
			return cand, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no .devrites directory found (set DEVRITES_ROOT or run inside a DevRites workspace)")
		}
		dir = parent
	}
}

// Feature is a loaded per-feature workspace: its declared phase plus which
// completeness sections currently have real content.
type Feature struct {
	Slug    string
	Phase   Phase
	Present map[Section]bool
}

// featureDir is where per-feature state lives under the root.
func featureDir(root, slug string) string {
	return filepath.Join(root, "features", slug)
}

// ListFeatures returns the slugs of every feature under root — directories
// under features/ that contain a feature.md — sorted. A root with no features/
// directory yields an empty list, not an error.
func ListFeatures(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "features"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var slugs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, "features", e.Name(), "feature.md")); err == nil {
			slugs = append(slugs, e.Name())
		}
	}
	sort.Strings(slugs)
	return slugs, nil
}

// CandidateFiles returns the paths that make up feature <slug>: its feature.md
// index plus every section file, in a stable order. Not all need exist; callers
// that fingerprint the feature stat and hash the ones that do.
func CandidateFiles(root, slug string) []string {
	dir := featureDir(root, slug)
	files := []string{filepath.Join(dir, "feature.md")}
	for _, s := range Sections {
		files = append(files, filepath.Join(dir, string(s)+".md"))
	}
	return files
}

// LoadFeature reads feature <slug> under root. The feature's index file
// (feature.md) carries the phase in its frontmatter; the six section files
// carry the content. A missing feature.md means the feature does not exist.
func LoadFeature(root, slug string) (*Feature, error) {
	dir := featureDir(root, slug)
	raw, err := os.ReadFile(filepath.Join(dir, "feature.md"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("feature %q not found", slug)
	}
	if err != nil {
		return nil, err
	}

	fm, _ := splitFrontmatter(raw)

	if v := strings.TrimSpace(fm["schemaVersion"]); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("feature %q: invalid schemaVersion %q", slug, v)
		}
		if n > SchemaVersion {
			return nil, fmt.Errorf("feature %q: schemaVersion %d is newer than this engine supports (%d); upgrade devrites", slug, n, SchemaVersion)
		}
	}

	phase := Phase(strings.TrimSpace(fm["phase"]))
	if phase == "" {
		return nil, fmt.Errorf("feature %q: feature.md is missing a phase in its frontmatter", slug)
	}
	if !KnownPhase(phase) {
		return nil, fmt.Errorf("feature %q: unknown phase %q", slug, phase)
	}

	present := make(map[Section]bool, len(Sections))
	for _, s := range Sections {
		present[s] = sectionPresent(filepath.Join(dir, string(s)+".md"))
	}
	return &Feature{Slug: slug, Phase: phase, Present: present}, nil
}

// ReadDeclaredSchemaVersion returns the schemaVersion declared in a feature's
// feature.md frontmatter, or 0 if the file, frontmatter, or field is
// absent/unparseable. Unlike LoadFeature it does NOT refuse a version newer than
// the engine supports — doctor needs to observe a newer version to report skew,
// not fail on it.
func ReadDeclaredSchemaVersion(root, slug string) int {
	raw, err := os.ReadFile(filepath.Join(featureDir(root, slug), "feature.md"))
	if err != nil {
		return 0
	}
	fm, _ := splitFrontmatter(raw)
	n, err := strconv.Atoi(strings.TrimSpace(fm["schemaVersion"]))
	if err != nil {
		return 0
	}
	return n
}

// MaxDeclaredSchemaVersion returns the highest schemaVersion declared by any
// feature under root, or 0 when no feature declares one. It is how doctor reports
// the "state" leg of the version triangle.
func MaxDeclaredSchemaVersion(root string) (int, error) {
	slugs, err := ListFeatures(root)
	if err != nil {
		return 0, err
	}
	max := 0
	for _, slug := range slugs {
		if v := ReadDeclaredSchemaVersion(root, slug); v > max {
			max = v
		}
	}
	return max, nil
}

// blankOrHash reports whether a trimmed line carries no meaningful content —
// it is empty or starts with '#' (a Markdown ATX heading, or a comment inside a
// frontmatter block). Both the section-content scan and the frontmatter parser
// skip such lines.
func blankOrHash(trimmed string) bool {
	return trimmed == "" || strings.HasPrefix(trimmed, "#")
}

// sectionPresent reports whether a section file has real content: it exists
// and, after removing a leading YAML frontmatter block, ATX (`#`) headings, and
// whitespace, some content remains. A stub that is only a heading counts as
// empty, so scaffolding a file never fakes completeness.
func sectionPresent(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	_, body := splitFrontmatter(raw)
	for _, line := range strings.Split(string(body), "\n") {
		if !blankOrHash(strings.TrimSpace(line)) {
			return true
		}
	}
	return false
}

// splitFrontmatter separates a leading `---\n ... \n---\n` block from the body.
// The closing fence must be a line that is exactly `---` (so a `----` rule or a
// `--- note` line in the body is never mistaken for it). It parses only flat
// `key: value` scalars (no third-party YAML dependency). With no well-formed
// block the map is empty and body is the original input.
func splitFrontmatter(raw []byte) (map[string]string, []byte) {
	fm := map[string]string{}
	norm := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if !strings.HasPrefix(norm, "---\n") {
		return fm, raw
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
		return fm, raw // no closing fence — treat as if there were no frontmatter
	}
	for _, line := range lines[:end] {
		t := strings.TrimSpace(line)
		if blankOrHash(t) {
			continue
		}
		if k, v, ok := strings.Cut(t, ":"); ok {
			fm[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return fm, []byte(strings.Join(lines[end+1:], "\n"))
}
