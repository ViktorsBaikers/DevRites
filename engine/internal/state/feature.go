package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/markdowntext"
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

// Feature is a loaded per-feature workspace: its declared phase plus which
// completeness sections currently have real content.
type Feature struct {
	Slug         string
	Phase        Phase
	Present      map[Section]bool
	PresentFiles map[string]bool
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

// LoadFeature reads a canonical feature workspace. state.md is the sole
// lifecycle authority; optional README.md metadata is not runtime state.
func LoadFeature(root, slug string) (*Feature, error) {
	dir := featureDir(root, slug)
	if !regularFileExists(filepath.Join(dir, LedgerFile)) {
		return nil, fmt.Errorf("feature %q not found", slug)
	}

	phase, ledgerDeclared, err := declaredPhaseFromLedger(filepath.Join(dir, LedgerFile))
	if err != nil {
		return nil, fmt.Errorf("feature %q: %w", slug, err)
	}
	if !ledgerDeclared {
		return nil, fmt.Errorf("feature %q: no phase in %s ledger", slug, LedgerFile)
	}
	policy, ok := PolicyFor(phase)
	if !ok {
		return nil, fmt.Errorf("feature %q: unknown phase %q", slug, phase)
	}
	phase = policy.Target

	present := make(map[Section]bool, len(Sections))
	for _, section := range Sections {
		present[section] = sectionPresentAny(dir, section)
	}
	presentFiles := make(map[string]bool)
	for _, lifecyclePolicy := range PhasePolicies() {
		for _, artifact := range lifecyclePolicy.RequiredArtifacts {
			name := string(artifact)
			if _, observed := presentFiles[name]; observed {
				continue
			}
			presentFiles[name] = sectionPresent(filepath.Join(dir, name))
		}
	}
	return &Feature{
		Slug:         slug,
		Phase:        phase,
		Present:      present,
		PresentFiles: presentFiles,
	}, nil
}

func sectionPresentAny(dir string, s Section) bool {
	for _, name := range sectionFiles[s] {
		if sectionPresent(filepath.Join(dir, name)) {
			return true
		}
	}
	return false
}

func declaredPhaseFromLedger(path string) (Phase, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read %s: %w", LedgerFile, err)
	}
	structural, err := markdowntext.Structural(raw)
	if err != nil {
		return "", false, fmt.Errorf("read %s: %w", LedgerFile, err)
	}
	lines := strings.Split(string(structural), "\n")
	value, ok := CursorField(lines, CursorPhase)
	if !ok {
		return "", false, nil
	}
	word := firstPhaseWord(value)
	if policy, known := PolicyFor(Phase(word)); known {
		return policy.Target, true, nil
	}
	return Phase(word), true, nil
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

// sectionPresent reports whether a section file has real content: it exists
// and, after removing a leading YAML frontmatter block, ATX (`#`) headings, and
// whitespace, some content remains. A stub that is only a heading counts as
// empty, so scaffolding a file never fakes completeness.
func sectionPresent(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	if _, err := markdowntext.Structural(raw); err != nil {
		return false
	}
	body := stripFrontmatter(raw)
	for _, line := range strings.Split(string(body), "\n") {
		if !blankOrHash(strings.TrimSpace(line)) {
			return true
		}
	}
	return false
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
		return raw // no closing fence: treat as if there were no frontmatter
	}
	return []byte(strings.Join(lines[end+1:], "\n"))
}
