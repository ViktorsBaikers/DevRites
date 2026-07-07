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

// cmdDoctor reports the binary / pack / state-schema version triangle and its
// skew verdict. It exits non-zero only when the state schema is a newer major
// than the binary can safely parse (a genuine refuse); a mere pack-vs-binary
// skew is a warning that still exits 0.
func cmdDoctor(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: devrites doctor")
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

// cmdMigrate upgrades an old-layout workspace to the current schema. It is
// idempotent (a second run is a no-op) and backs up the pre-migration state
// first.
func cmdMigrate(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: devrites migrate")
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
		fmt.Fprintln(stdout, "migrate: already up to date (no old-layout features found)")
		return exitOK
	}
	fmt.Fprintf(stdout, "migrated %d feature(s): %v\n", len(result.Migrated), result.Migrated)
	fmt.Fprintf(stdout, "backup: %s\n", result.BackupDir)
	return exitOK
}
