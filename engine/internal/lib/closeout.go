package lib

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
)

// releaseLockBeforeRename drops the feature lock before the workspace rename on
// Windows, which refuses to rename a directory while the lock file inside it is
// open. renameWorkspace is the rename; both are variables so tests can inject the
// platform decision and observe the lock at the moment of the rename.
var (
	releaseLockBeforeRename = runtime.GOOS == "windows"
	renameWorkspace         = os.Rename
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
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(stderr, "state close: no workspace for %s at %s\n", slug, filepath.Join(dv, "work", slug))
			return 4
		}
		fmt.Fprintf(stderr, "state close: invalid workspace for %s: %v\n", slug, err)
		return 1
	}
	if err := state.RequireWorkspaceSchema(dv, slug); err != nil {
		fmt.Fprintf(stderr, "state close: %v\n", err)
		return 3
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

	// Hold the per-feature lock across the rename and the ACTIVE clear, as
	// dispatch and the parallel lease ops do across their load-modify-save. A
	// writer already holding the lock finishes its save before the rename, so
	// that save is archived. On unix a writer queued behind this close-out
	// acquires a lock file that now lives in the archive; state.WithFeatureLock
	// reports state.ErrWorkspaceMoved and the writer does not run, so it cannot
	// re-create work/<slug>. Windows cannot rename a directory that holds an
	// open file, so there the lock is released just before the rename.
	code := 1
	if err := state.WithFeatureLockRelease(dv, slug, func(release func()) error {
		code = closeOutLocked(release, work, arch, archiveDir, archDisplay, active, activeSlug, slug, stdout, stderr)
		return nil
	}); err != nil {
		fmt.Fprintf(stderr, "state close: cannot lock feature %s: %v\n", slug, err)
		return 1
	}
	return code
}

// closeOutLocked performs the rename and the conditional ACTIVE clear. It runs
// under the feature lock held by CloseOut and reports an exit code rather than
// an error, preserving CloseOut's documented exit codes.
func closeOutLocked(release func(), work, arch, archiveDir, archDisplay, active, activeSlug, slug string, stdout, stderr io.Writer) int {
	// Fail closed: if the move does not happen, leave the cursor alone and do not
	// claim success: a lost audit trail must never look like a clean close-out.
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "state close: cannot create archive dir: %v\n", err)
		return 1
	}
	if releaseLockBeforeRename {
		release()
	}
	if err := renameWorkspace(work, arch); err != nil {
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
