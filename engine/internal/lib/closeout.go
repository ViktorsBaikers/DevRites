package lib

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/notes"
	"github.com/devrites/devrites/internal/parallel"
	"github.com/devrites/devrites/internal/state"
)

// renameWorkspace lets tests observe exclusion at the archive boundary.
var renameWorkspace = os.Rename

// CloseOut physically archives a workspace, including owner-authorized nonfinal
// exits. Seal, Ship, Done, and shipped/sealed/done statuses must first satisfy
// the final gates and retained prerequisites. The workspace is relocated, never
// deleted: the audit trail is preserved under archive/<slug>.
//
// args is `<slug>`; root is the already-resolved .devrites directory. Exit codes:
//
//	0  archived (ACTIVE cleared if it pointed here)
//	3  final prerequisites or a retained parallel lease block archiving
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

	// A broken symlink at the destination still counts as "already there", so use
	// Lstat rather than Stat to avoid silently overwriting it.
	if _, err := os.Lstat(arch); err == nil {
		fmt.Fprintf(stderr, "state close: archive already exists at %s: refusing to clobber\n", arch)
		return 5
	}

	// Hold the stable feature lock through verification, rename, ACTIVE clear,
	// and rollback so queued writers cannot revive an archived workspace.
	code := 1
	if err := parallel.WithControlLock(dv, func() error {
		leased, err := parallel.WorkspaceLeased(dv, slug)
		if err != nil {
			return err
		}
		if leased {
			fmt.Fprintf(stderr, "state close: BLOCKED: %s is named by a retained parallel lease; finish and clean up the controller lease before archiving\n", slug)
			code = 3
			return nil
		}
		return state.WithFeatureLock(dv, slug, func() error {
			return state.WithLock(claimsPath(dv)+".lock", func() error {
				code = closeOutLocked(work, arch, archiveDir, archDisplay, active, slug, stdout, stderr)
				return nil
			})
		})
	}); err != nil {
		fmt.Fprintf(stderr, "state close: %v\n", err)
		return 1
	}
	return code
}

// closeOutLocked performs the rename and the conditional ACTIVE clear. It runs
// under the feature lock held by CloseOut and reports an exit code rather than
// an error, preserving CloseOut's documented exit codes.
func closeOutLocked(work, arch, archiveDir, archDisplay, active, slug string, stdout, stderr io.Writer) int {
	root := filepath.Dir(active)
	ledger, err := readWorkspaceArtifact(root, slug, "state.md")
	if err != nil {
		fmt.Fprintf(stderr, "state close: BLOCKED: %v\n", err)
		return 3
	}
	lines := strings.Split(string(ledger), "\n")
	phase, _ := state.CursorField(lines, "phase")
	status, _ := state.CursorField(lines, "status")
	phase = string(state.CanonicalPhase(phase))
	final := phase == string(state.PhaseSeal) || phase == string(state.PhaseShip) || phase == string(state.PhaseDone) ||
		status == "sealed" || status == "shipped" || status == "done"
	if final {
		result, err := gate.Check(gate.Seal, root, slug)
		if err != nil {
			fmt.Fprintf(stderr, "state close: BLOCKED: %v\n", err)
			return 3
		}
		if result.Blocked {
			fmt.Fprint(stdout, result.Render())
			return 3
		}
		if err := VerifyReleaseUnion(root, slug); err != nil {
			fmt.Fprintf(stderr, "state close: BLOCKED: release-union: %v\n", err)
			return 3
		}
		if err := CheckFinalPrerequisites(root, slug); err != nil {
			fmt.Fprintf(stderr, "state close: BLOCKED: final-prerequisites: %v\n", err)
			return 3
		}
		if EvidenceFresh(root, []string{slug}, stdout, stderr) != 0 || !notes.SealCheck(root, slug, stdout) {
			return 3
		}
	}
	activeSlug, err := devritespaths.PhysicalActiveSlug(filepath.Dir(active))
	if err != nil {
		fmt.Fprintf(stderr, "state close: invalid ACTIVE cursor: %v\n", err)
		return 1
	}
	// Fail closed: if the move does not happen, leave the cursor alone and do not
	// claim success: a lost audit trail must never look like a clean close-out.
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "state close: cannot create archive dir: %v\n", err)
		return 1
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
