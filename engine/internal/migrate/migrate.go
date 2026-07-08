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
	"sort"
	"strings"
	"time"

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
		return nil, err
	}
	if len(targets) == 0 {
		return &Result{Skipped: true}, nil
	}

	backup, err := backupWorkspace(root)
	if err != nil {
		return nil, err
	}
	migrated := make([]string, 0, len(targets))
	for _, target := range targets {
		if err := normalizeFeatureDir(target.dir, target.slug); err != nil {
			return nil, fmt.Errorf("normalize %q: %w", target.slug, err)
		}
		migrated = append(migrated, target.slug)
	}
	return &Result{Migrated: uniqueSorted(migrated), BackupDir: backup}, nil
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
			return nil, err
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
	if !regularFileExists(filepath.Join(dir, "feature.md")) {
		return true
	}
	return regularFileExists(filepath.Join(dir, "evidence.md")) && !regularFileExists(filepath.Join(dir, "proof.md")) ||
		regularFileExists(filepath.Join(dir, state.LedgerFile)) && !regularFileExists(filepath.Join(dir, "status.md"))
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func normalizeFeatureDir(dir, slug string) error {
	if !regularFileExists(filepath.Join(dir, "feature.md")) {
		phase := derivePhase(filepath.Join(dir, state.LedgerFile))
		if err := state.AtomicWrite(filepath.Join(dir, "feature.md"), []byte(featureIndex(slug, phase)), 0o644); err != nil {
			return err
		}
	}
	if err := copyAliasFile(dir, "evidence.md", "proof.md"); err != nil {
		return err
	}
	return copyAliasFile(dir, state.LedgerFile, "status.md")
}

func uniqueSorted(values []string) []string {
	sort.Strings(values)
	out := values[:0]
	var prev string
	for i, v := range values {
		if i == 0 || v != prev {
			out = append(out, v)
			prev = v
		}
	}
	return out
}

func copyAliasFile(dir, alias, canonical string) error {
	src := filepath.Join(dir, alias)
	dst := filepath.Join(dir, canonical)
	if !regularFileExists(src) || regularFileExists(dst) {
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return state.AtomicWrite(dst, data, 0o644)
}

// featureIndex renders the manifest added to a normalized workspace.
func featureIndex(slug string, phase state.Phase) string {
	return fmt.Sprintf(`---
slug: %s
title: %s
phase: %s
schemaVersion: %d
---

Normalized by DevRites migration.
`, slug, slug, phase, state.SchemaVersion)
}

// derivePhase reads the legacy state.md and maps its recorded phase/status onto a
// v1 phase. An unreadable or unrecognised state defaults to build — the safe
// middle of the arc, from which the completeness gates re-derive what's missing.
func derivePhase(statePath string) state.Phase {
	raw, err := os.ReadFile(statePath)
	if err != nil {
		return state.PhaseBuild
	}
	for _, line := range strings.Split(string(raw), "\n") {
		t := strings.TrimLeft(strings.ToLower(strings.TrimSpace(line)), "-*+ \t")
		key, val, ok := strings.Cut(t, ":")
		if !ok || (strings.TrimSpace(key) != "phase" && strings.TrimSpace(key) != "status") {
			continue
		}
		if p, ok := mapLegacyPhase(strings.TrimSpace(val)); ok {
			return p
		}
	}
	return state.PhaseBuild
}

// mapLegacyPhase maps a legacy phase/status word onto a v1 phase.
func mapLegacyPhase(word string) (state.Phase, bool) {
	// Take the first token, so "done — shipped 2024" maps on "done".
	if i := strings.IndexAny(word, " \t—-"); i > 0 {
		word = word[:i]
	}
	switch word {
	case "frame":
		return state.PhaseFrame, true
	case "spec", "specced", "specifying":
		return state.PhaseSpec, true
	case "plan", "planned", "define", "defined":
		return state.PhasePlan, true
	case "build", "building", "wip", "in", "in-progress":
		return state.PhaseBuild, true
	case "prove", "proving", "proven", "testing":
		return state.PhaseProve, true
	case "vet", "review", "reviewing", "vetting":
		return state.PhaseVet, true
	case "seal", "sealed", "sealing":
		return state.PhaseSeal, true
	case "ship", "shipped", "done", "closed", "complete", "completed":
		return state.PhaseShip, true
	default:
		return "", false
	}
}

// backupWorkspace snapshots mutable state (work/, features/, and the ACTIVE pointer) into
// a timestamped, gitignored backup directory under root before any change.
func backupWorkspace(root string) (string, error) {
	backup := filepath.Join(root, fmt.Sprintf(".migrate-backup-%d", time.Now().UnixNano()))
	if err := copyTree(filepath.Join(root, "work"), filepath.Join(backup, "work")); err != nil {
		return "", err
	}
	if err := copyTree(filepath.Join(root, "features"), filepath.Join(backup, "features")); err != nil {
		return "", err
	}
	// The ACTIVE pointer is a single small file; copy it if present.
	if data, err := os.ReadFile(filepath.Join(root, "ACTIVE")); err == nil {
		if err := os.MkdirAll(backup, 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(backup, "ACTIVE"), data, 0o644); err != nil {
			return "", err
		}
	}
	return backup, nil
}

// copyTree recursively copies src into dst. A missing src is not an error.
func copyTree(src, dst string) error {
	info, err := os.Stat(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyTree(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
