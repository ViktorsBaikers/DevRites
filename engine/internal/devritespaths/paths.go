package devritespaths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DevritesRootName = ".devrites"

	ClaudeSkillsTarget = ".claude/skills"
	CodexSkillsTarget  = ".agents/skills"
	ClaudeAgentsTarget = ".claude/agents"
	CodexAgentsTarget  = ".codex/agents"

	ManifestName = ".claude/devrites.manifest"
)

// WorkspaceOverride resolves DEVRITES_WORKSPACE relative to the project
// directory that contains the .devrites root.
func WorkspaceOverride(root, slug string) string {
	raw := strings.TrimSpace(os.Getenv("DEVRITES_WORKSPACE"))
	if raw == "" {
		return ""
	}
	path := raw
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(root), path)
	}
	path = filepath.Clean(path)
	if slug == "" || filepath.Base(path) == slug {
		return path
	}
	return ""
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
	if ws := WorkspaceOverride(root, ""); ws != "" {
		return filepath.Base(ws), nil
	}
	raw, err := os.ReadFile(filepath.Join(root, "ACTIVE"))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read active feature: %w", err)
	}
	return strings.TrimSpace(string(raw)), nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
