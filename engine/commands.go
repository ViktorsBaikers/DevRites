package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/devrites/devrites/internal/doctor"
	"github.com/devrites/devrites/internal/migrate"
	"github.com/devrites/devrites/internal/state"
)

// resolveRootLenient resolves the .devrites root for the workspace-orientation
// commands (preamble / progress / stuck), which must degrade gracefully outside a
// workspace rather than erroring like the gates do. When no .devrites is found it
// returns a nonexistent <cwd>/.devrites so the command reads an empty workspace
// and no-ops (preamble's "No active workspace", progress's silence).
func resolveRootLenient() string {
	if root, err := state.ResolveRoot(os.Getenv("DEVRITES_ROOT")); err == nil {
		return root
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ".devrites"
	}
	return filepath.Join(cwd, ".devrites")
}

// cmdDoctor reports the binary / pack / state-schema version triangle and its
// skew verdict. It exits non-zero only when the state schema is a newer major
// than the binary can safely parse (a genuine refuse); a mere pack-vs-binary
// skew is a warning that still exits 0.
func cmdDoctor(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: devrites-engine doctor")
		return exitUsage
	}
	root, err := state.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	// The project directory is the parent of the .devrites root — that is where
	// an installed pack's version marker (.claude/devrites.version, package.json)
	// lives.
	projectDir := filepath.Dir(root)
	report, err := doctor.Diagnose(projectDir, root)
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	fmt.Fprint(stdout, report.Render())
	if report.Refuse {
		return exitBlocked
	}
	return exitOK
}

// cmdMigrate normalizes workspaces to the current schema. It is idempotent
// (a second run is a no-op) and backs up the pre-migration state first.
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
