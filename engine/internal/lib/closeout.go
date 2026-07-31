package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
)

// CloseOut retires a shipped feature: it moves the feature directory into the
// archive and, when the feature is still the active one, clears the ACTIVE cursor
// so the next /rite-spec starts clean. The workspace is relocated, never deleted:
// the audit trail is preserved under archive/<slug>.
//
// args is `<slug>`; root is the already-resolved .devrites directory. Exit codes:
//
//	0  archived (ACTIVE cleared if it pointed here)
//	4  no slug given, or the feature has no workspace
//	5  an archive already exists at the destination: refuse to clobber it
func CloseOut(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || args[0] == "" {
		fmt.Fprintln(stderr, "usage: devrites-engine state close <slug>")
		return 4
	}
	slug := args[0]
	dv := root

	work, err := devritespaths.ExistingFeatureDirChecked(dv, slug)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(stderr, "state close: no workspace for %s\n", slug)
			return 4
		}
		fmt.Fprintf(stderr, "state close: invalid workspace for %s: %v\n", slug, err)
		return 1
	}
	archiveDir, err := devritespaths.ArchiveDirChecked(dv)
	if err != nil {
		fmt.Fprintf(stderr, "state close: invalid archive directory: %v\n", err)
		return 1
	}
	arch := filepath.Join(archiveDir, slug)
	archDisplay := filepath.ToSlash(filepath.Join("archive", slug))
	active := filepath.Join(dv, "ACTIVE")

	activeSlug, err := devritespaths.ActiveSlug(dv)
	if err != nil {
		fmt.Fprintf(stderr, "state close: invalid ACTIVE cursor: %v\n", err)
		return 1
	}
	// A broken symlink at the destination still counts as "already there", so use
	// Lstat rather than Stat to avoid silently overwriting it.
	if _, err := os.Lstat(arch); err == nil {
		fmt.Fprintf(stderr, "state close: archive already exists at %s: refusing to clobber\n", arch)
		return 5
	}

	// Fail closed: if the move does not happen, leave the cursor alone and do not
	// claim success: a lost audit trail must never look like a clean close-out.
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "state close: cannot create archive dir: %v\n", err)
		return 1
	}
	if err := os.Rename(work, arch); err != nil {
		fmt.Fprintf(stderr, "state close: cannot archive %s -> %s: %v\n", work, arch, err)
		return 1
	}

	// Clear ACTIVE only if it still points at the slug we just archived.
	if activeSlug == slug {
		if err := state.AtomicWrite(active, nil, 0o644); err != nil {
			rollbackErr := os.Rename(arch, work)
			if rollbackErr != nil {
				fmt.Fprintf(stderr, "state close: cannot clear ACTIVE after archiving: %v; rollback %s -> %s also failed: %v\n", err, arch, work, rollbackErr)
			} else {
				fmt.Fprintf(stderr, "state close: cannot clear ACTIVE; archive move rolled back: %v\n", err)
			}
			return 1
		}
		fmt.Fprintf(stdout, "state close: archived %s -> %s and cleared ACTIVE\n", slug, archDisplay)
		return 0
	}
	fmt.Fprintf(stdout, "state close: archived %s -> %s (ACTIVE pointed elsewhere: left as-is)\n", slug, archDisplay)
	return 0
}
