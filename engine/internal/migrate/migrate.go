// Package migrate normalizes .devrites workspaces in place. work/<slug> is the
// canonical layout; features/<slug> remains readable as a compatibility alias.
// The normalizer adds canonical files without deleting aliases. It is idempotent
// and backs up the pre-migration state before touching anything, so a botched
// upgrade never strands work.
package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/state"
)

// Result reports what a migration did. Skipped is true when there was nothing to
// migrate (a no-op re-run); Migrated lists the slugs upgraded this run; BackupDir
// is the pre-migration snapshot (empty when Skipped).
type Result struct {
	Skipped   bool
	Migrated  []string
	BackupDir string
}

type normalizationTarget struct {
	slug string
	dir  string
}

// Run normalizes the workspace at root. If no feature directory needs additive
// files, it returns Skipped without writing or backing up anything (idempotent).
// Otherwise it snapshots the state, then writes missing canonical files.
func Run(root string) (*Result, error) {
	targets, err := featureDirsNeedingNormalization(root)
	if err != nil {
		return nil, fmt.Errorf("scan workspace: %w", err)
	}
	if len(targets) == 0 {
		return &Result{Skipped: true}, nil
	}

	backup, err := backupWorkspace(root)
	if err != nil {
		return nil, fmt.Errorf("backup workspace: %w", err)
	}
	migrated := make([]string, 0, len(targets))
	for _, target := range targets {
		if err := normalizeFeatureDir(target.dir, target.slug); err != nil {
			return nil, fmt.Errorf("normalize %q: %w", target.slug, err)
		}
		migrated = append(migrated, target.slug)
	}
	sort.Strings(migrated)
	return &Result{Migrated: slices.Compact(migrated), BackupDir: backup}, nil
}

func featureDirsNeedingNormalization(root string) ([]normalizationTarget, error) {
	var targets []normalizationTarget
	for _, layout := range []string{"work", "features"} {
		base := filepath.Join(root, layout)
		entries, err := os.ReadDir(base)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("list feature dirs: %w", err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dir := filepath.Join(base, e.Name())
			if needsNormalizeFeature(dir) {
				targets = append(targets, normalizationTarget{slug: e.Name(), dir: dir})
			}
		}
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].slug != targets[j].slug {
			return targets[i].slug < targets[j].slug
		}
		return targets[i].dir < targets[j].dir
	})
	return targets, nil
}

func needsNormalizeFeature(dir string) bool {
	if !regularFileExists(filepath.Join(dir, state.WorkspaceMapFile)) {
		return true
	}
	if version := declaredSchemaVersion(dir); version < state.SchemaVersion {
		return true
	}
	return regularFileExists(filepath.Join(dir, "proof.md")) && !regularFileExists(filepath.Join(dir, state.EvidenceFile)) ||
		regularFileExists(filepath.Join(dir, "status.md")) && !regularFileExists(filepath.Join(dir, state.LedgerFile))
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func normalizeFeatureDir(dir, slug string) error {
	for _, alias := range state.WorkspaceMapFiles()[1:] {
		if err := copyAliasFile(dir, alias, state.WorkspaceMapFile); err != nil {
			return fmt.Errorf("copy %s to %s: %w", alias, state.WorkspaceMapFile, err)
		}
	}
	if !regularFileExists(filepath.Join(dir, state.WorkspaceMapFile)) {
		phase := derivePhase(filepath.Join(dir, state.LedgerFile))
		if err := state.AtomicWrite(filepath.Join(dir, state.WorkspaceMapFile), []byte(workspaceIndex(slug, phase)), 0o644); err != nil {
			return fmt.Errorf("write workspace index: %w", err)
		}
	}
	phase := derivePhase(filepath.Join(dir, state.LedgerFile))
	if err := upgradeWorkspaceMapSchema(filepath.Join(dir, state.WorkspaceMapFile), slug, phase); err != nil {
		return fmt.Errorf("upgrade workspace schema: %w", err)
	}
	if err := copyAliasFile(dir, "proof.md", state.EvidenceFile); err != nil {
		return fmt.Errorf("copy proof.md to evidence.md: %w", err)
	}
	return copyAliasFile(dir, "status.md", state.LedgerFile)
}

func declaredSchemaVersion(dir string) int {
	for _, name := range state.WorkspaceMapFiles() {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		lines := strings.Split(string(raw), "\n")
		if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
			return 0
		}
		for _, line := range lines[1:] {
			if strings.TrimSpace(line) == "---" {
				break
			}
			key, value, ok := strings.Cut(line, ":")
			if ok && strings.TrimSpace(key) == "schemaVersion" {
				version, _ := strconv.Atoi(strings.TrimSpace(value))
				return version
			}
		}
		return 0
	}
	return 0
}

// upgradeWorkspaceMapSchema advances only the compatibility declaration. It
// never creates decision coverage, readiness, or proof artifacts: those must be
// earned by their owning phases.
func upgradeWorkspaceMapSchema(path, slug string, phase state.Phase) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	version := declaredSchemaVersion(filepath.Dir(path))
	if version > state.SchemaVersion {
		return fmt.Errorf("schemaVersion %d is newer than this engine supports (%d)", version, state.SchemaVersion)
	}
	if version == state.SchemaVersion {
		return nil
	}

	text := string(raw)
	if !strings.HasPrefix(text, "---\n") {
		prefix := fmt.Sprintf("---\nslug: %s\nphase: %s\nschemaVersion: %d\n---\n\n", slug, phase, state.SchemaVersion)
		return state.AtomicWrite(path, []byte(prefix+text), 0o644)
	}

	lines := strings.Split(text, "\n")
	end := -1
	schemaLine := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
		key, _, ok := strings.Cut(lines[i], ":")
		if ok && strings.TrimSpace(key) == "schemaVersion" {
			schemaLine = i
		}
	}
	if end < 0 {
		return fmt.Errorf("workspace map has an unclosed frontmatter block")
	}
	value := fmt.Sprintf("schemaVersion: %d", state.SchemaVersion)
	if schemaLine >= 0 {
		lines[schemaLine] = value
	} else {
		lines = append(lines[:end], append([]string{value}, lines[end:]...)...)
	}
	return state.AtomicWrite(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func copyAliasFile(dir, alias, canonical string) error {
	src := filepath.Join(dir, alias)
	dst := filepath.Join(dir, canonical)
	if !regularFileExists(src) || regularFileExists(dst) {
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read alias file: %w", err)
	}
	return state.AtomicWrite(dst, data, 0o644)
}

// workspaceIndex renders the compact map used by a normalized workspace.
func workspaceIndex(slug string, phase state.Phase) string {
	return fmt.Sprintf(`---
slug: %s
title: %s
phase: %s
schemaVersion: %d
---

Normalized by DevRites migration.
`, slug, slug, phase, state.SchemaVersion)
}

// derivePhase reads state.md in either canonical cursor-table or legacy form and
// maps its recorded phase/status onto the current lifecycle. An unreadable or
// unrecognised state defaults to build: the safe middle of the arc, from which
// the completeness gates re-derive what's missing.
func derivePhase(statePath string) state.Phase {
	raw, err := os.ReadFile(statePath)
	if err != nil {
		return state.PhaseBuild
	}
	lines := strings.Split(string(raw), "\n")
	for _, key := range []string{state.CursorPhase, state.CursorStatus} {
		if value, found := state.CursorField(lines, key); found {
			if p, ok := mapLegacyPhase(value); ok {
				return p
			}
		}
	}
	return state.PhaseBuild
}

// mapLegacyPhase maps a current or legacy phase/status word onto the lifecycle.
func mapLegacyPhase(word string) (state.Phase, bool) {
	word = strings.ToLower(strings.TrimSpace(word))
	// Take the first token, so a legacy "done" status with trailing notes still maps to "done".
	if i := strings.IndexAny(word, " \t—-"); i > 0 {
		word = word[:i]
	}
	return state.PhaseForName(word)
}

// backupWorkspace snapshots mutable state (work/, features/, and the ACTIVE pointer) into
// a timestamped, gitignored backup directory under root before any change.
func backupWorkspace(root string) (string, error) {
	backup := filepath.Join(root, fmt.Sprintf(".migrate-backup-%d", time.Now().UnixNano()))
	if err := fsutil.CopyTree(filepath.Join(root, "work"), filepath.Join(backup, "work")); err != nil {
		return "", fmt.Errorf("copy work tree: %w", err)
	}
	if err := fsutil.CopyTree(filepath.Join(root, "features"), filepath.Join(backup, "features")); err != nil {
		return "", fmt.Errorf("copy features tree: %w", err)
	}
	// The ACTIVE pointer is a single small file; copy it if present.
	if data, err := os.ReadFile(filepath.Join(root, "ACTIVE")); err == nil {
		if err := os.MkdirAll(backup, 0o755); err != nil {
			return "", fmt.Errorf("create backup dir: %w", err)
		}
		if err := os.WriteFile(filepath.Join(backup, "ACTIVE"), data, 0o644); err != nil {
			return "", fmt.Errorf("copy ACTIVE pointer: %w", err)
		}
	}
	return backup, nil
}
