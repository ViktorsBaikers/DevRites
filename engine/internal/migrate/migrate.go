// Package migrate performs the one-shot upgrade of a pre-v1 .devrites workspace
// (the flat .devrites/work/<slug>/ layout the bash runtime used) to the current
// per-feature schema (.devrites/features/<slug>/ with a fixed section set). It is
// idempotent — re-running is a no-op — and always backs up the pre-migration
// state before touching anything, so a botched upgrade never strands work.
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

// oldToNew maps a legacy work/<slug> file to its new section file. Legacy files
// with no mapping (review.md, seal.md, drift.md, questions.md, …) are copied
// verbatim into the new feature dir so nothing is lost, but only these six carry
// completeness meaning.
var oldToNew = map[string]string{
	"spec.md":      "spec.md",
	"plan.md":      "plan.md",
	"decisions.md": "decisions.md",
	"tasks.md":     "tasks.md",
	"evidence.md":  "proof.md",
	"state.md":     "status.md",
}

// Result reports what a migration did. Skipped is true when there was nothing to
// migrate (a no-op re-run); Migrated lists the slugs upgraded this run; BackupDir
// is the pre-migration snapshot (empty when Skipped).
type Result struct {
	Skipped   bool
	Migrated  []string
	BackupDir string
}

// Run migrates the workspace at root. It first determines which slugs still live
// only in the old layout; if none do, it returns Skipped without writing or
// backing up anything (idempotent). Otherwise it snapshots the old state, then
// writes each feature's new-schema files.
func Run(root string) (*Result, error) {
	workDir := filepath.Join(root, "work")
	slugs, err := oldSlugs(workDir)
	if err != nil {
		return nil, err
	}

	var todo []string
	for _, slug := range slugs {
		if !newFeatureExists(root, slug) {
			todo = append(todo, slug)
		}
	}
	if len(todo) == 0 {
		return &Result{Skipped: true}, nil
	}

	backup, err := backupWorkspace(root)
	if err != nil {
		return nil, err
	}
	for _, slug := range todo {
		if err := migrateFeature(root, workDir, slug); err != nil {
			return nil, fmt.Errorf("migrate %q: %w", slug, err)
		}
	}
	return &Result{Migrated: todo, BackupDir: backup}, nil
}

// oldSlugs lists the feature slugs in the legacy work/ directory. A missing work/
// directory yields no slugs (nothing old to migrate), not an error.
func oldSlugs(workDir string) ([]string, error) {
	entries, err := os.ReadDir(workDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var slugs []string
	for _, e := range entries {
		if e.IsDir() {
			slugs = append(slugs, e.Name())
		}
	}
	sort.Strings(slugs)
	return slugs, nil
}

func newFeatureExists(root, slug string) bool {
	_, err := os.Stat(filepath.Join(root, "features", slug, "feature.md"))
	return err == nil
}

// migrateFeature writes the new-schema feature dir from one legacy work/<slug>.
func migrateFeature(root, workDir, slug string) error {
	src := filepath.Join(workDir, slug)
	dst := filepath.Join(root, "features", slug)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			return err
		}
		target := e.Name()
		if mapped, ok := oldToNew[e.Name()]; ok {
			target = mapped
		}
		if err := state.AtomicWrite(filepath.Join(dst, target), data, 0o644); err != nil {
			return err
		}
	}

	phase := derivePhase(filepath.Join(src, "state.md"))
	return state.AtomicWrite(filepath.Join(dst, "feature.md"), []byte(featureIndex(slug, phase)), 0o644)
}

func featureIndex(slug string, phase state.Phase) string {
	return fmt.Sprintf(`---
slug: %s
title: %s
phase: %s
schemaVersion: %d
---

Migrated from the pre-v1 .devrites/work/ layout.
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
		t := strings.ToLower(strings.TrimSpace(line))
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

// backupWorkspace snapshots the legacy state (work/ and the ACTIVE pointer) into
// a timestamped, gitignored backup directory under root before any change.
func backupWorkspace(root string) (string, error) {
	backup := filepath.Join(root, fmt.Sprintf(".migrate-backup-%d", time.Now().UnixNano()))
	if err := copyTree(filepath.Join(root, "work"), filepath.Join(backup, "work")); err != nil {
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
