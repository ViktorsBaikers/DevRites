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

	ClaudeSkillsTarget    = ".claude/skills"
	CodexSkillsTarget     = ".agents/skills"
	OmpSkillsTarget       = ".omp/skills"
	ClaudeAgentsTarget    = ".claude/agents"
	CodexAgentsTarget     = ".codex/agents"
	OmpAgentsTarget       = ".omp/agents"
	ClaudeWorkflowsTarget = ".claude/workflows"

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
		parts[0] == "work"
	if sameRoot || !canonicalWorkspace || !validSlug(filepath.Base(path)) {
		return "", fmt.Errorf("DRV-WORKSPACE-INVALID: DEVRITES_WORKSPACE %q is not a feature workspace; run `unset DEVRITES_WORKSPACE`", raw)
	}
	if slug != "" && filepath.Base(path) != slug {
		return "", fmt.Errorf("DRV-WORKSPACE-SLUG-MISMATCH: DEVRITES_WORKSPACE basename %q does not match feature %q; run `unset DEVRITES_WORKSPACE`", filepath.Base(path), slug)
	}
	return path, nil
}

// FeatureDir is the canonical per-feature workspace directory under root.
func FeatureDir(root, slug string) string {
	if ws := WorkspaceOverride(root, slug); ws != "" {
		return ws
	}
	return filepath.Join(root, "work", slug)
}

// ExistingFeatureDirChecked resolves an existing canonical feature workspace
// without following a work/<slug> symlink. Mutating
// control-plane commands use this instead of FeatureDir so an attacker cannot
// redirect archive/removal operations outside the selected DevRites root.
func ExistingFeatureDirChecked(root, slug string) (string, error) {
	if !validSlug(slug) {
		return "", fmt.Errorf("DRV-WORKSPACE-INVALID: %q is not a feature slug", slug)
	}
	if ws, err := WorkspaceOverrideChecked(root, slug); err != nil {
		return "", err
	} else if ws != "" {
		return ws, nil
	}
	candidate := filepath.Join(root, "work", slug)
	resolved, err := checkedCanonicalDir(root, candidate, "work", slug)
	if err == nil {
		return resolved, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return "", os.ErrNotExist
	}
	return "", err
}

// ArchiveDirChecked resolves (or safely prepares) the canonical archive
// directory. Existing symlinked archive components are rejected before a
// close-out can move a workspace through them.
func ArchiveDirChecked(root string) (string, error) {
	return checkedCanonicalDir(root, filepath.Join(root, "archive"), "archive", "")
}

func checkedCanonicalDir(root, candidate, parent, slug string) (string, error) {
	resolvedRoot, err := safepath.ResolveExisting(root)
	if err != nil {
		return "", fmt.Errorf("DRV-WORKSPACE-INVALID: resolve DevRites root %q: %w", root, err)
	}
	info, err := os.Lstat(candidate)
	if err != nil {
		if !os.IsNotExist(err) || parent != "archive" {
			return "", err
		}
		resolved, resolveErr := safepath.ResolveExisting(candidate)
		if resolveErr != nil {
			return "", fmt.Errorf("DRV-ARCHIVE-INVALID: resolve archive %q: %w", candidate, resolveErr)
		}
		if !safepath.WithinResolved(resolved, resolvedRoot) {
			return "", fmt.Errorf("DRV-ARCHIVE-OUTSIDE-ROOT: archive %q escapes %q", candidate, root)
		}
		rel, relErr := filepath.Rel(resolvedRoot, resolved)
		if relErr != nil || filepath.Clean(rel) != "archive" {
			return "", fmt.Errorf("DRV-ARCHIVE-INVALID: archive %q is not the canonical archive directory", candidate)
		}
		return filepath.Clean(candidate), nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("DRV-WORKSPACE-SYMLINK: canonical directory %q must not be a symlink", candidate)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("DRV-WORKSPACE-INVALID: canonical directory %q is not a directory", candidate)
	}
	resolved, err := safepath.ResolveExisting(candidate)
	if err != nil {
		return "", fmt.Errorf("DRV-WORKSPACE-INVALID: resolve canonical directory %q: %w", candidate, err)
	}
	if !safepath.WithinResolved(resolved, resolvedRoot) {
		return "", fmt.Errorf("DRV-WORKSPACE-OUTSIDE-ROOT: canonical directory %q escapes %q", candidate, root)
	}
	want := parent
	if slug != "" {
		want = filepath.Join(parent, slug)
	}
	rel, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || filepath.Clean(rel) != filepath.Clean(want) {
		return "", fmt.Errorf("DRV-WORKSPACE-INVALID: canonical directory %q resolves to unexpected path %q", candidate, resolved)
	}
	return filepath.Clean(candidate), nil
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
	// #nosec G304 -- operator-supplied workspace root; ACTIVE symlink refused above
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

func validSlug(slug string) bool {
	return slug != "" && slug != "." && slug != ".." &&
		filepath.Base(slug) == slug &&
		!strings.ContainsAny(slug, `/\`)
}
