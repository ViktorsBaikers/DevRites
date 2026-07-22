package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
)

// ResolveRoot returns the .devrites directory to operate on. A non-empty
// override (e.g. from DEVRITES_ROOT) may name either the project root or its
// .devrites directory; otherwise the nearest ancestor .devrites is used.
func ResolveRoot(override string) (string, error) {
	if override != "" {
		info, err := os.Stat(override)
		if err != nil {
			return "", fmt.Errorf("DEVRITES_ROOT %q: %w", override, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("DEVRITES_ROOT %q is not a directory", override)
		}
		if filepath.Base(filepath.Clean(override)) == ".devrites" {
			return override, nil
		}
		child := filepath.Join(override, ".devrites")
		if info, err := os.Stat(child); err == nil && info.IsDir() {
			return child, nil
		}
		return "", fmt.Errorf("DEVRITES_ROOT %q is not a project root or .devrites directory", override)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
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

// featureDir is where per-feature state lives under the root. work/ is the
// canonical layout; features/ remains readable as a compatibility alias.
func featureDir(root, slug string) string {
	return devritespaths.FeatureDir(root, slug)
}

// ListFeatures returns the slugs of every feature under root: directories under
// canonical work/ plus compatibility features/ recognized as a feature: sorted.
// A directory is a feature if it has a workspace map OR the working-state
// ledger (state.md), so a live workspace the pack created without a map
// still lists. Missing layout directories yield an empty list, not an error.
func ListFeatures(root string) ([]string, error) {
	seen := map[string]bool{}
	for _, layout := range []string{"work", "features"} {
		entries, err := os.ReadDir(filepath.Join(root, layout))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("list features: %w", err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if isFeatureDir(filepath.Join(root, layout, e.Name())) {
				seen[e.Name()] = true
			}
		}
	}
	slugs := make([]string, 0, len(seen))
	for slug := range seen {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	return slugs, nil
}

// isFeatureDir reports whether dir carries a workspace map or the working-state
// ledger. Either is a sufficient phase source for LoadFeature.
func isFeatureDir(dir string) bool {
	if regularFileExists(filepath.Join(dir, LedgerFile)) {
		return true
	}
	for _, name := range workspaceMapFiles {
		if regularFileExists(filepath.Join(dir, name)) {
			return true
		}
	}
	return false
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// LoadFeature reads feature <slug> under root. The phase comes from the live
// working-state ledger when it declares one, otherwise from workspace-map
// frontmatter. A feature with neither a map nor a ledger does not exist.
// Section content is read from the canonical section files or their
// aliases (see sectionFiles).
func LoadFeature(root, slug string) (*Feature, error) {
	dir := featureDir(root, slug)

	var manifest []byte
	for _, name := range workspaceMapFiles {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			manifest = data
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("feature %q: read %s: %w", slug, name, err)
		}
	}
	hasManifest := manifest != nil
	hasLedger := regularFileExists(filepath.Join(dir, LedgerFile))
	if !hasManifest && !hasLedger {
		return nil, fmt.Errorf("feature %q not found", slug)
	}

	var manifestPhase Phase
	if hasManifest {
		fm, _ := splitFrontmatter(manifest)
		if v := strings.TrimSpace(fm["schemaVersion"]); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("feature %q: invalid schemaVersion %q", slug, v)
			}
			if n > SchemaVersion {
				return nil, fmt.Errorf("feature %q: schemaVersion %d is newer than this engine supports (%d); upgrade devrites", slug, n, SchemaVersion)
			}
		}
		manifestPhase = Phase(strings.TrimSpace(fm["phase"]))
	}

	// state.md is the mutable runtime cursor, so it must win over the manifest,
	// which migration and compatibility flows may leave stale. Preserve an
	// explicitly unknown ledger value so the validation below reports it rather
	// than silently falling back to feature.md.
	phase, ledgerDeclared := declaredPhaseFromLedger(filepath.Join(dir, LedgerFile))
	if !ledgerDeclared {
		phase = manifestPhase
	}
	if phase == "" {
		return nil, fmt.Errorf("feature %q: no phase in workspace-map frontmatter or %s ledger", slug, LedgerFile)
	}
	if !KnownPhase(phase) {
		return nil, fmt.Errorf("feature %q: unknown phase %q", slug, phase)
	}

	present := make(map[Section]bool, len(Sections))
	for _, s := range Sections {
		present[s] = sectionPresentAny(dir, s)
	}
	return &Feature{Slug: slug, Phase: phase, Present: present}, nil
}

// sectionPresentAny reports whether any file that can satisfy section s has real
// content: the canonical name or a transitional alias.
func sectionPresentAny(dir string, s Section) bool {
	for _, name := range sectionFiles[s] {
		if sectionPresent(filepath.Join(dir, name)) {
			return true
		}
	}
	return false
}

func declaredPhaseFromLedger(path string) (Phase, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	value, ok := CursorField(strings.Split(string(raw), "\n"), CursorPhase)
	if !ok {
		return "", false
	}
	word := strings.ToLower(strings.TrimSpace(value))
	if i := strings.IndexAny(word, " \t"); i > 0 {
		word = word[:i]
	}
	return Phase(word), true
}

// ReadDeclaredSchemaVersion returns the schemaVersion declared in a feature's
// feature.md frontmatter, or 0 if the file, frontmatter, or field is
// absent/unparseable. Unlike LoadFeature it does NOT refuse a version newer than
// the engine supports: doctor needs to observe a newer version to report skew,
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
		return 0, fmt.Errorf("resolve max schema version: %w", err)
	}
	max := 0
	for _, slug := range slugs {
		if v := ReadDeclaredSchemaVersion(root, slug); v > max {
			max = v
		}
	}
	return max, nil
}

// blankOrHash reports whether a trimmed line carries no meaningful content:
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
		return fm, raw // no closing fence: treat as if there were no frontmatter
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
