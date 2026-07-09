package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/devrites/devrites/internal/workflow"
)

// touchedPathRe captures each `backtick-quoted` path listed in touched-files.md.
var touchedPathRe = regexp.MustCompile("`([^`]+)`")

// EvidenceFresh reports whether a feature's proof still post-dates the code it
// proves. It compares the newest modification time among the touched files (the
// backtick-quoted paths in touched-files.md that still exist) against the proof
// (evidence.md / proof.md / browser-evidence.md); if any touched file is newer, the proof
// is stale and must be regenerated before a GO decision.
//
// base is the project directory that contains the .devrites tree. slug defaults
// to the active feature named in .devrites/ACTIVE.
//
//	0  fresh — the proof post-dates every touched file, or there is nothing to compare
//	3  stale — a touched file is newer than the proof
//	5  no workspace, or no proof to check
func EvidenceFresh(root string, args []string, stdout, stderr io.Writer) int {
	slug := ""
	if len(args) > 0 {
		slug = args[0]
	}
	if slug == "" {
		slug = activeSlug(root)
	}

	workDir := featureDir(root, slug)
	if slug == "" || !isDir(workDir) {
		name := slug
		if name == "" {
			name = "<unset>"
		}
		fmt.Fprintf(stderr, "evidence-fresh: no active workspace (slug=%s)\n", name)
		return 5
	}

	newestProof, haveProof := newestModTime(
		filepath.Join(workDir, "evidence.md"),
		filepath.Join(workDir, "proof.md"),
		filepath.Join(workDir, "browser-evidence.md"),
	)
	if !haveProof {
		fmt.Fprintf(stderr, "evidence-fresh: no evidence.md / proof.md / browser-evidence.md in %s — run %s\n", workDir, workflow.ForVerb("prove").Both())
		return 5
	}

	touched, err := os.ReadFile(filepath.Join(workDir, "touched-files.md"))
	if err != nil {
		fmt.Fprintln(stdout, "evidence-fresh: no touched-files.md — nothing to compare (treating as fresh).")
		return 0
	}

	var newestCode int64
	var newestPath string
	for _, match := range touchedPathRe.FindAllStringSubmatch(string(touched), -1) {
		listed := match[1]
		// touched-files.md lists PROJECT-relative paths; the project directory is
		// the parent of the .devrites root. An absolute path is taken as given.
		path := listed
		if !filepath.IsAbs(path) {
			path = filepath.Join(filepath.Dir(root), path)
		}
		if m, ok := modTime(path); ok && m > newestCode {
			newestCode, newestPath = m, listed
		}
	}

	if newestCode > newestProof {
		fmt.Fprintf(stderr, "evidence-fresh: STALE — %s is newer than the proof. Re-run %s before GO.\n", newestPath, workflow.ForVerb("prove").Both())
		return 3
	}
	fmt.Fprintln(stdout, "evidence-fresh: OK — evidence post-dates every touched file.")
	return 0
}

// newestModTime returns the most recent modification time among the given paths
// that exist as regular files, and whether any qualified.
func newestModTime(paths ...string) (newest int64, ok bool) {
	for _, p := range paths {
		if !isFile(p) {
			continue
		}
		if m, mok := modTime(p); mok && (!ok || m > newest) {
			newest, ok = m, true
		}
	}
	return newest, ok
}

// modTime returns a file's modification time in whole seconds since the epoch.
func modTime(path string) (secs int64, ok bool) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, false
	}
	return fi.ModTime().Unix(), true
}
