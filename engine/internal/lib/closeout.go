package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CloseOut retires a shipped feature: it moves the feature directory into the
// archive and, when the feature is still the active one, clears the ACTIVE cursor
// so the next /rite-spec starts clean. The workspace is relocated, never deleted —
// the audit trail is preserved under archive/<slug>.
//
// args is `<slug> [devrites-dir]`. The optional second argument overrides the
// .devrites directory (default: the resolved root), which lets a caller archive a
// workspace outside the current one. Exit codes:
//
//	0  archived (ACTIVE cleared if it pointed here)
//	4  no slug given, or the feature has no workspace
//	5  an archive already exists at the destination — refuse to clobber it
func CloseOut(root string, args []string, stdout, stderr io.Writer) int {
	slug := argAt(args, 0)
	dv := argAt(args, 1)
	if dv == "" {
		dv = root
	}
	if slug == "" {
		fmt.Fprintln(stderr, "close-out: usage: devrites-engine close-out <slug> [devrites-dir]")
		return 4
	}

	work := featureDir(dv, slug)
	arch := filepath.Join(dv, "archive", slug)
	active := filepath.Join(dv, "ACTIVE")

	if !isDir(work) {
		fmt.Fprintf(stderr, "close-out: no workspace at %s\n", work)
		return 4
	}
	// A broken symlink at the destination still counts as "already there", so use
	// Lstat rather than Stat to avoid silently overwriting it.
	if _, err := os.Lstat(arch); err == nil {
		fmt.Fprintf(stderr, "close-out: archive already exists at %s — refusing to clobber\n", arch)
		return 5
	}

	// Fail closed: if the move does not happen, leave the cursor alone and do not
	// claim success — a lost audit trail must never look like a clean close-out.
	if err := os.MkdirAll(filepath.Join(dv, "archive"), 0o755); err != nil {
		fmt.Fprintf(stderr, "close-out: cannot create archive dir: %v\n", err)
		return 1
	}
	if err := os.Rename(work, arch); err != nil {
		fmt.Fprintf(stderr, "close-out: cannot archive %s -> %s: %v\n", work, arch, err)
		return 1
	}

	// Clear ACTIVE only if it still points at the slug we just archived.
	if isFile(active) {
		if b, err := os.ReadFile(active); err == nil && stripASCIISpace(string(b)) == slug {
			_ = os.WriteFile(active, nil, 0o644)
			fmt.Fprintf(stdout, "close-out: archived %s -> %s and cleared ACTIVE\n", slug, arch)
			return 0
		}
	}
	fmt.Fprintf(stdout, "close-out: archived %s -> %s (ACTIVE pointed elsewhere — left as-is)\n", slug, arch)
	return 0
}
