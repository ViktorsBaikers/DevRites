package devritespaths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/safepath"
)

const (
	DevritesRootName = ".devrites"

	ClaudeSkillsTarget = ".claude/skills"
	CodexSkillsTarget  = ".agents/skills"
	ClaudeAgentsTarget = ".claude/agents"
	CodexAgentsTarget  = ".codex/agents"

	ManifestName = ".claude/devrites.manifest"
)

// WorkspaceOverride resolves a valid DEVRITES_WORKSPACE. Invalid overrides are
// ignored here so path construction stays inside root; callers that surface
// diagnostics use WorkspaceOverrideChecked.
func WorkspaceOverride(root, slug string) string {
	path, _ := WorkspaceOverrideChecked(root, slug)
	return path
}

// WorkspaceOverrideChecked resolves DEVRITES_WORKSPACE relative to the project
// and proves its physical path remains inside the selected .devrites root.
func WorkspaceOverrideChecked(root, slug string) (string, error) {
	raw := strings.TrimSpace(os.Getenv("DEVRITES_WORKSPACE"))
	if raw == "" {
		return "", nil
	}
	path := raw
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(root), path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("DRV-WORKSPACE-INVALID: resolve DEVRITES_WORKSPACE: %w", err)
	}
	path, err = safepath.ResolveExisting(abs)
	if err != nil {
		return "", fmt.Errorf("DRV-WORKSPACE-INVALID: resolve DEVRITES_WORKSPACE %q: %w", raw, err)
	}
	if !safepath.WithinResolved(path, root) {
		return "", fmt.Errorf("DRV-WORKSPACE-OUTSIDE-ROOT: DEVRITES_WORKSPACE %q is outside %q; run `unset DEVRITES_WORKSPACE`", raw, root)
	}
	resolvedRoot, err := safepath.ResolveExisting(root)
	if err != nil {
		return "", fmt.Errorf("DRV-WORKSPACE-INVALID: resolve DevRites root %q: %w", root, err)
	}
	sameRoot := path == resolvedRoot
	if pathInfo, pathErr := os.Stat(path); pathErr == nil {
		if rootInfo, rootErr := os.Stat(resolvedRoot); rootErr == nil {
			sameRoot = os.SameFile(pathInfo, rootInfo)
		}
	}
	rel, relErr := filepath.Rel(resolvedRoot, path)
	parts := strings.FieldsFunc(rel, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	canonicalWorkspace := relErr == nil && len(parts) == 2 &&
		(parts[0] == "work" || parts[0] == "features")
	if sameRoot || !canonicalWorkspace || !validSlug(filepath.Base(path)) {
		return "", fmt.Errorf("DRV-WORKSPACE-INVALID: DEVRITES_WORKSPACE %q is not a feature workspace; run `unset DEVRITES_WORKSPACE`", raw)
	}
	if slug != "" && filepath.Base(path) != slug {
		return "", fmt.Errorf("DRV-WORKSPACE-SLUG-MISMATCH: DEVRITES_WORKSPACE basename %q does not match feature %q; run `unset DEVRITES_WORKSPACE`", filepath.Base(path), slug)
	}
	return path, nil
}

// FeatureDir is the per-feature workspace directory under root. work/ is
// canonical; features/ remains readable as a compatibility alias.
func FeatureDir(root, slug string) string {
	if ws := WorkspaceOverride(root, slug); ws != "" {
		return ws
	}
	work := filepath.Join(root, "work", slug)
	if isDir(work) {
		return work
	}
	features := filepath.Join(root, "features", slug)
	if isDir(features) {
		return features
	}
	return work
}

// ActiveSlug reads and trims the active-feature pointer. A missing pointer file
// is not an error; it yields the empty slug.
func ActiveSlug(root string) (string, error) {
	if ws, err := WorkspaceOverrideChecked(root, ""); err != nil {
		return "", err
	} else if ws != "" {
		return filepath.Base(ws), nil
	}
	activePath := filepath.Join(root, "ACTIVE")
	if info, err := os.Lstat(activePath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("DRV-ACTIVE-SYMLINK: ACTIVE is a symlink; run `rm -f %q`", activePath)
	}
	raw, err := os.ReadFile(activePath)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read active feature: %w", err)
	}
	slug := strings.TrimSpace(string(raw))
	if slug != "" && !validSlug(slug) {
		return "", fmt.Errorf("DRV-ACTIVE-INVALID: ACTIVE value %q is not a feature slug; run `rm -f %q`", slug, activePath)
	}
	return slug, nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func validSlug(slug string) bool {
	return slug != "" && slug != "." && slug != ".." &&
		filepath.Base(slug) == slug &&
		!strings.ContainsAny(slug, `/\`)
}
