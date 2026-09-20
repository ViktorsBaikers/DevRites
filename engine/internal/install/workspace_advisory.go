package install

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/state"
)

// reportStaleWorkspaces prints a post-install/update advisory naming every
// live workspace whose state.md lacks the current schema row. Without it a
// pack update that bumps the engine schema strands existing workspaces until
// the first gated command refuses mid-workflow. Advisory only: it reads
// workspace state.md files and writes nothing.
func reportStaleWorkspaces(stdout io.Writer, target string) {
	devritesRoot := filepath.Join(target, ".devrites")
	workDir := filepath.Join(devritesRoot, "work")
	entries, err := os.ReadDir(workDir)
	if err != nil {
		return // no workspaces — nothing to report
	}
	var stale []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		ledger := filepath.Join(workDir, e.Name(), state.LedgerFile)
		if info, statErr := os.Stat(ledger); statErr != nil || !info.Mode().IsRegular() {
			continue // operational remnant, not a live workspace
		}
		version, err := state.WorkspaceSchema(devritesRoot, e.Name())
		if err != nil || version < state.SchemaVersion {
			stale = append(stale, e.Name())
		}
	}
	if len(stale) == 0 {
		return
	}
	sort.Strings(stale)
	fmt.Fprintf(stdout, "workspaces predating schema %d: %s\n", state.SchemaVersion, strings.Join(stale, ", "))
	fmt.Fprintln(stdout, "  run `devrites-engine migrate <slug>` per workspace before resuming them")
}
