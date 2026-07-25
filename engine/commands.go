package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/doctor"
	"github.com/devrites/devrites/internal/migrate"
	"github.com/devrites/devrites/internal/rootfacts"
	"github.com/devrites/devrites/internal/state"
)

type rootMode uint8

const (
	rootUnused rootMode = iota
	rootLenient
	rootStrict
)

// rootModeFor is the single policy boundary between diagnostic/read-only
// commands, which may degrade outside a workspace, and commands that can write
// workspace or Git state, which must never fall back after an unsafe root
// selection.
func rootModeFor(command string, args []string) rootMode {
	subcommand := firstRootOperand(args)
	switch command {
	case "first-task", "spec-dedupe", "evidence-fresh", "coverage",
		"doubt-coverage", "budget", "preamble", "progress",
		"build-readiness", "readiness-digest", "analyze", "mutation-gate",
		"test-integrity", "review-integrity",
		"archive-search", "config", "reviewers", "outside-voice", "docs-stale",
		"secret-scan", "lanes", "overrides":
		return rootLenient
	case "clarify-return", "reconcile", "close-out", "forge":
		return rootStrict
	case "resolve":
		if subcommand == "next-qid" {
			return rootLenient
		}
		return rootStrict
	case "footprint", "stuck":
		if subcommand == "log" {
			return rootStrict
		}
		return rootLenient
	case "recovery":
		if subcommand == "record" || subcommand == "clear" {
			return rootStrict
		}
		return rootLenient
	case "decisions":
		if subcommand == "index" {
			return rootStrict
		}
		return rootLenient
	case "ledger":
		if subcommand == "sync" {
			return rootStrict
		}
		return rootLenient
	case "learnings":
		if subcommand == "add" {
			return rootStrict
		}
		return rootLenient
	case "timeline":
		if subcommand == "log" || subcommand == "purge" {
			return rootStrict
		}
		return rootLenient
	case "health":
		if subcommand == "" || subcommand == "run" || subcommand == "check" || subcommand == "record" {
			return rootStrict
		}
		return rootLenient
	case "review-fingerprints":
		if hasRootFlag(args, "--write") {
			return rootStrict
		}
		return rootLenient
	case "reviewer-stats":
		if subcommand == "record" {
			return rootStrict
		}
		return rootLenient
	case "extensions":
		if subcommand == "sync" {
			return rootStrict
		}
		return rootLenient
	case "context":
		if subcommand == "sync" {
			return rootStrict
		}
		return rootLenient
	case "runbook":
		if subcommand == "run" || subcommand == "resume" {
			return rootStrict
		}
		return rootLenient
	default:
		return rootUnused
	}
}

func firstRootOperand(args []string) string {
	for _, arg := range args {
		if arg == "--json" {
			continue
		}
		return strings.ToLower(strings.TrimSpace(arg))
	}
	return ""
}

func hasRootFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

// resolveRootFor resolves exactly once per command invocation. Strict commands
// retain root-safety refusals; lenient readers preserve their historical
// diagnostic/no-workspace behavior.
func resolveRootFor(command string, args []string) (string, int, error) {
	switch rootModeFor(command, args) {
	case rootUnused:
		return "", exitOK, nil
	case rootLenient:
		return resolveRootLenient(), exitOK, nil
	case rootStrict:
		root, err := state.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
		if err == nil {
			return root, exitOK, nil
		}
		if errors.Is(err, rootfacts.ErrUnsafeRoot) {
			return "", exitBlocked, err
		}
		if errors.Is(err, rootfacts.ErrNoRoot) {
			return fallbackRoot(), exitOK, nil
		}
		return "", exitUsage, err
	default:
		panic("unknown root resolution mode")
	}
}

// resolveRootLenient resolves the .devrites root for read-only and diagnostic
// commands that must degrade cleanly outside a workspace. When no root is found,
// it returns a nonexistent <cwd>/.devrites so those commands see an empty
// workspace and keep their established no-workspace behavior.
func resolveRootLenient() string {
	if root, err := state.ResolveRoot(os.Getenv("DEVRITES_ROOT")); err == nil {
		return root
	}
	return fallbackRoot()
}

func fallbackRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ".devrites"
	}
	return filepath.Join(cwd, devritespaths.DevritesRootName)
}

func readFile(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(raw)
}

func gitDir(projectDir string) string {
	path := filepath.Join(projectDir, ".git")
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return path
	}
	line := strings.TrimSpace(readFile(path))
	if target, ok := strings.CutPrefix(line, "gitdir:"); ok {
		target = strings.TrimSpace(target)
		if filepath.IsAbs(target) {
			return target
		}
		return filepath.Clean(filepath.Join(projectDir, target))
	}
	return path
}

func gitOperation(projectDir, resolvedGitDir string) string {
	dir := resolvedGitDir
	if dir == "" {
		dir = gitDir(projectDir)
	}
	if _, err := os.Stat(filepath.Join(dir, "MERGE_HEAD")); err == nil {
		return "merge"
	}
	for _, name := range []string{"rebase-merge", "rebase-apply"} {
		if info, err := os.Stat(filepath.Join(dir, name)); err == nil && info.IsDir() {
			return "rebase"
		}
	}
	return ""
}

// cmdDoctor reports canonical root facts and the binary, pack, and state schema
// versions. It exits nonzero for an unsafe root or newer state schema. Missing
// optional data and warnings remain diagnostic.
func cmdDoctor(args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 || len(args) == 1 && args[0] != "--verbose" {
		fmt.Fprintln(stderr, "usage: devrites-engine doctor [--verbose]")
		return exitUsage
	}
	facts, resolveErr := rootfacts.Resolve(os.Getenv("DEVRITES_ROOT"))
	if resolveErr != nil && len(facts.Hazards) == 0 {
		fmt.Fprintf(stderr, "devrites: %v\n", resolveErr)
		return exitUsage
	}
	root := facts.PhysicalRoot
	projectDir := facts.PhysicalProject
	report, err := doctor.DiagnoseFacts(facts)
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	fmt.Fprint(stdout, report.Render())
	if op := gitOperation(projectDir, facts.Git.Dir); op != "" {
		fmt.Fprintf(stdout, "git-state: %s in progress: resolve with .claude/skills/devrites-lib/reference/standards/git-workflow.md#merge-conflict-recovery\n", op)
	}
	if root != "" {
		for _, warning := range extensionProvenanceWarnings(root) {
			fmt.Fprintf(stdout, "warning: %s\n", warning)
		}
	}
	if root != "" {
		slug := strings.TrimSpace(readFile(filepath.Join(root, "ACTIVE")))
		if snap, err := state.Snapshot(root, slug); err == nil {
			fmt.Fprintln(stdout)
			fmt.Fprintln(stdout, "readiness-dashboard:")
			fmt.Fprintf(stdout, "  active: %s (%s, %s)\n", snap.Slug, snap.Phase, snap.RunMode)
			fmt.Fprintf(stdout, "  evidence: %s\n", snap.Evidence.Status)
			fmt.Fprintf(stdout, "  drift: %s\n", snap.Drift.Status)
			fmt.Fprintf(stdout, "  review: %s\n", snap.Review.Status)
			fmt.Fprintf(stdout, "  harness: %s: %s\n", snap.Harness.Status, snap.Harness.Detail)
			fmt.Fprintf(stdout, "  extensions: %s (%d)\n", snap.Extensions.Status, snap.Extensions.Count)
			fmt.Fprintf(stdout, "  worktree: %s (%d changed)\n", snap.DirtyWorkspace.Status, snap.DirtyWorkspace.Changed)
			fmt.Fprintln(stdout, "  capabilities:")
			for _, cap := range snap.Capabilities {
				fmt.Fprintf(stdout, "    - %s: %s · used by %s · fallback: %s · risk: %s\n", cap.Name, cap.Status, cap.UsedBy, cap.Fallback, cap.Risk)
			}
		}
	}
	if report.Refuse {
		return exitBlocked
	}
	return exitOK
}

// cmdMigrate normalizes workspaces to the current schema. It is idempotent
// (a second run is a no-op) and backs up the pre-migration state first.
func extensionProvenanceWarnings(root string) []string {
	dir := filepath.Join(root, "extensions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var warnings []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		edir := filepath.Join(dir, entry.Name())
		hasArtifact := readFile(filepath.Join(edir, "skill", "SKILL.md")) != "" || readFile(filepath.Join(edir, "agent.md")) != "" || readFile(filepath.Join(edir, "component.yaml")) != ""
		if hasArtifact && readFile(filepath.Join(edir, "provenance.json")) == "" {
			warnings = append(warnings, fmt.Sprintf("extension %s has no provenance.json", entry.Name()))
		}
	}
	return warnings
}

func cmdMigrate(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: devrites-engine migrate")
		return exitUsage
	}
	root, err := state.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	result, err := migrate.Run(root)
	if err != nil {
		fmt.Fprintf(stderr, "devrites: migrate failed: %v\n", err)
		return 1
	}
	if result.Skipped {
		fmt.Fprintln(stdout, "migrate: already up to date (no workspace normalization needed)")
		return exitOK
	}
	fmt.Fprintf(stdout, "migrated %d feature(s): %v\n", len(result.Migrated), result.Migrated)
	fmt.Fprintf(stdout, "backup: %s\n", result.BackupDir)
	return exitOK
}

func cmdSnapshot(args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: devrites-engine snapshot [slug]")
		return exitUsage
	}
	root, err := state.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	slug := ""
	if len(args) == 1 {
		slug = args[0]
	}
	snapshot, err := state.Snapshot(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "devrites: cannot render snapshot JSON: %v\n", err)
		return exitUsage
	}
	_, _ = stdout.Write(append(data, '\n'))
	return exitOK
}
