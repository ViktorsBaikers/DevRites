package lib

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
)

// maxSequenceChain bounds the sequence_parent walk so a malformed cycle cannot
// loop forever; real chains are far smaller.
const maxSequenceChain = 64

// RunMergeManifest folds predecessor candidate manifests into the release
// workspace's `## Candidate manifest` so a release milestone's prove/review/seal
// binds the union surface instead of only its own slices. On a path collision
// the later sequence position's row wins (it records the most recent touch).
// Everything outside the manifest section of the release workspace's
// touched-files.md is preserved.
//
// args is `<slug>` to walk the recorded `sequence_parent` chain, or
// `<slug> <predecessor> [<predecessor>...]` for an explicit ordered list —
// required for chains recorded before the sequence cursor fields existed.
// Exit codes:
//
//	0  merged and written
//	2  usage error
//	3  blocked: a workspace, manifest, or chain check failed
//	1  internal/IO error
func RunMergeManifest(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: devrites-engine state merge-manifest <slug> [<predecessor>...]")
		return 2
	}
	slug := args[0]
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "merge-manifest: BLOCKED: workspace for %s: %v\n", slug, err)
		return 3
	}
	if err := state.RequireWorkspaceSchema(root, slug); err != nil {
		fmt.Fprintf(stderr, "merge-manifest: BLOCKED: %v\n", err)
		return 3
	}
	code := 0
	if err := state.WithFeatureLock(root, slug, func() error {
		code = mergeManifestLocked(root, slug, workspace, args, stdout, stderr)
		return nil
	}); err != nil {
		fmt.Fprintf(stderr, "merge-manifest: %v\n", err)
		return 1
	}
	return code
}

// mergeManifestLocked reads, unions, and rewrites touched-files.md under the
// feature lock so a concurrent writer cannot interleave with the rewrite.
func mergeManifestLocked(root, slug, workspace string, args []string, stdout, stderr io.Writer) int {
	manifestPath := filepath.Join(workspace, "touched-files.md")
	releaseRaw, err := readBoundedRegularFile(manifestPath, maxCandidateManifestBytes)
	if err != nil {
		fmt.Fprintf(stderr, "merge-manifest: BLOCKED: candidate manifest %s: %v\n", manifestPath, err)
		return 3
	}

	var manifests [][]candidateRow
	if len(args) > 1 {
		manifests, err = explicitManifests(root, slug, args[1:])
	} else {
		// The walked chain is validated inside walkSequenceChain; only its
		// manifests feed the union.
		_, manifests, err = walkSequenceChain(root, slug, workspace)
	}
	if err != nil {
		fmt.Fprintf(stderr, "merge-manifest: BLOCKED: %v\n", err)
		return 3
	}

	merged := make(map[string]candidateRow)
	collisions := 0
	fold := func(rows []candidateRow) {
		for _, row := range rows {
			if _, dup := merged[row.path]; dup {
				collisions++
			}
			merged[row.path] = row
		}
	}
	for _, rows := range manifests {
		fold(rows)
	}
	releaseRows, err := parseCandidateManifest(releaseRaw)
	if err != nil {
		fmt.Fprintf(stderr, "merge-manifest: BLOCKED: candidate manifest %s: %v\n", manifestPath, err)
		return 3
	}
	fold(releaseRows)

	paths := make([]string, 0, len(merged))
	for path := range merged {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	body := make([]string, 0, len(paths)+2)
	if len(paths) == 0 {
		body = append(body, "No project files.")
	} else {
		body = append(body, "| State | File | Slice | Reason |", "| --- | --- | --- | --- |")
		for _, path := range paths {
			row := merged[path]
			body = append(body, fmt.Sprintf("| %s | `%s` | %s | %s |", row.state, row.path, row.slice, row.reason))
		}
	}
	updated, err := replaceManifestSectionBody(releaseRaw, body)
	if err != nil {
		fmt.Fprintf(stderr, "merge-manifest: BLOCKED: candidate manifest %s: %v\n", manifestPath, err)
		return 3
	}
	if err := state.AtomicWrite(manifestPath, updated, 0o644); err != nil {
		fmt.Fprintf(stderr, "merge-manifest: cannot write %s: %v\n", manifestPath, err)
		return 1
	}
	fmt.Fprintf(stdout, "merge-manifest: merged %d manifest(s) into %d row(s)", len(manifests)+1, len(paths))
	if collisions > 0 {
		fmt.Fprintf(stdout, " (%d path collision(s), later position kept)", collisions)
	}
	fmt.Fprintln(stdout)
	return 0
}

// explicitManifests parses the caller-ordered predecessor list. A numbered
// workspace or one recording a parent requires the complete recorded chain;
// legacy workspaces without cursors retain caller-supplied ordering.
func explicitManifests(root, slug string, predecessors []string) ([][]candidateRow, error) {
	seen := make(map[string]struct{}, len(predecessors))
	for _, pred := range predecessors {
		if pred == slug {
			return nil, fmt.Errorf("predecessor %q names the target workspace", pred)
		}
		if _, dup := seen[pred]; dup {
			return nil, fmt.Errorf("duplicate predecessor %q", pred)
		}
		seen[pred] = struct{}{}
	}
	dir, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return nil, err
	}
	raw, err := readBoundedRegularFile(filepath.Join(dir, "state.md"), maxCandidateManifestBytes)
	if err != nil {
		return nil, fmt.Errorf("%s state.md: %w", slug, err)
	}
	lines := strings.Split(string(raw), "\n")
	parent, _ := state.CursorField(lines, state.CursorSequenceParent)
	_, numbered := state.CursorField(lines, state.CursorSequencePosition)
	if parent != "" || numbered {
		chain, manifests, err := walkSequenceChain(root, slug, dir)
		if err != nil {
			return nil, err
		}
		if len(chain) != len(predecessors) {
			return nil, fmt.Errorf("%s requires the complete ordered sequence_parent chain", slug)
		}
		for i, pred := range predecessors {
			if pred != chain[i] {
				return nil, fmt.Errorf("%s requires the complete ordered sequence_parent chain", slug)
			}
		}
		return manifests, nil
	}
	manifests := make([][]candidateRow, 0, len(predecessors))
	for _, pred := range predecessors {
		rows, err := predecessorManifestRows(root, pred)
		if err != nil {
			return nil, fmt.Errorf("predecessor %s: %w", pred, err)
		}
		manifests = append(manifests, rows)
	}
	return manifests, nil
}

// walkSequenceChain follows sequence_parent links, returning predecessors
// oldest-first. Numbered chains must descend consecutively to position 1;
// fully unnumbered legacy chains may end at any parentless workspace. Broken
// links or mixed numbering refuse the merge instead of producing a partial union.
func walkSequenceChain(root, slug, workspace string) ([]string, [][]candidateRow, error) {
	visited := map[string]struct{}{slug: {}}
	chain := []string{}
	manifests := [][]candidateRow{}
	dir := workspace
	numbered := false
	var expectedPosition uint64
	for {
		raw, err := readBoundedRegularFile(filepath.Join(dir, "state.md"), maxCandidateManifestBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("%s state.md: %w", filepath.Base(dir), err)
		}
		lines := strings.Split(string(raw), "\n")
		parent, _ := state.CursorField(lines, state.CursorSequenceParent)
		positionText, hasPosition := state.CursorField(lines, state.CursorSequencePosition)
		if len(chain) == 0 {
			numbered = hasPosition
		}
		if hasPosition != numbered {
			return nil, nil, fmt.Errorf("%s mixes numbered and unnumbered sequence positions", filepath.Base(dir))
		}
		if numbered {
			position, err := strconv.ParseUint(positionText, 10, 64)
			if err != nil || position == 0 || strconv.FormatUint(position, 10) != positionText {
				return nil, nil, fmt.Errorf("%s sequence_position %q must be a positive canonical decimal", filepath.Base(dir), positionText)
			}
			if len(chain) > 0 && position != expectedPosition {
				return nil, nil, fmt.Errorf("%s sequence_position %d; expected %d", filepath.Base(dir), position, expectedPosition)
			}
			if parent == "" && position != 1 {
				return nil, nil, fmt.Errorf("%s sequence_position %d has no sequence_parent; only position 1 may end the chain", filepath.Base(dir), position)
			}
			if parent != "" && position == 1 {
				return nil, nil, fmt.Errorf("%s sequence_position 1 must not have a sequence_parent", filepath.Base(dir))
			}
			expectedPosition = position - 1
		}
		if parent == "" {
			break
		}
		if len(chain) == maxSequenceChain {
			return nil, nil, fmt.Errorf("sequence chain longer than %d", maxSequenceChain)
		}
		if _, dup := visited[parent]; dup {
			return nil, nil, fmt.Errorf("sequence_parent cycle at %q", parent)
		}
		visited[parent] = struct{}{}
		rows, err := predecessorManifestRows(root, parent)
		if err != nil {
			return nil, nil, fmt.Errorf("sequence_parent %s: %w", parent, err)
		}
		dir, err = devritespaths.ExistingFeatureDirChecked(root, parent)
		if errors.Is(err, os.ErrNotExist) {
			dir, err = devritespaths.ExistingArchivedFeatureDirChecked(root, parent)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("sequence_parent %s: %w", parent, err)
		}
		chain = append(chain, parent)
		manifests = append(manifests, rows)
	}
	if len(chain) == 0 && !numbered {
		return nil, nil, fmt.Errorf("%s has no sequence_parent; pass an explicit ordered predecessor list", slug)
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
		manifests[i], manifests[j] = manifests[j], manifests[i]
	}
	return chain, manifests, nil
}

// VerifyReleaseUnion enforces the release-milestone contract at the candidate
// and seal gates: a workspace whose cursor declares `sequence_role: release`
// must carry a candidate manifest covering every predecessor in its recorded
// `sequence_parent` chain. Workspaces without the role are unaffected.
func VerifyReleaseUnion(root, slug string) error {
	dir, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return fmt.Errorf("workspace for %s: %w", slug, err)
	}
	raw, err := readBoundedRegularFile(filepath.Join(dir, "state.md"), maxCandidateManifestBytes)
	if errors.Is(err, os.ErrNotExist) {
		return nil // no state.md: nothing declares the release role
	}
	if err != nil {
		return fmt.Errorf("%s state.md: %w", slug, err)
	}
	role, _ := state.CursorField(strings.Split(string(raw), "\n"), state.CursorSequenceRole)
	if role != "release" {
		return nil
	}
	_, manifests, err := walkSequenceChain(root, slug, dir)
	if err != nil {
		return err
	}
	manifestRaw, err := readBoundedRegularFile(filepath.Join(dir, "touched-files.md"), maxCandidateManifestBytes)
	if err != nil {
		return fmt.Errorf("candidate manifest: %w", err)
	}
	rows, err := parseCandidateManifest(manifestRaw)
	if err != nil {
		return fmt.Errorf("candidate manifest: %w", err)
	}
	declared := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		declared[row.path] = struct{}{}
	}
	var missing []string
	for _, ancestor := range manifests {
		for _, row := range ancestor {
			if _, ok := declared[row.path]; !ok {
				missing = append(missing, row.path)
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		const show = 8
		list := missing
		if len(list) > show {
			list = append(append([]string(nil), missing[:show]...), fmt.Sprintf("…and %d more", len(missing)-show))
		}
		return fmt.Errorf("release manifest does not cover the recorded sequence chain; run `devrites-engine state merge-manifest %s`. Missing: %s", slug, strings.Join(list, ", "))
	}
	return nil
}

// predecessorManifestRows loads the candidate manifest of a sealed or archived
// predecessor workspace. Live workspaces win over archive entries so an
// unshipped sequence member is always found first.
func predecessorManifestRows(root, slug string) ([]candidateRow, error) {
	dir, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if errors.Is(err, os.ErrNotExist) {
		dir, err = devritespaths.ExistingArchivedFeatureDirChecked(root, slug)
	}
	if err != nil {
		return nil, fmt.Errorf("no live or archived workspace: %w", err)
	}
	raw, err := readBoundedRegularFile(filepath.Join(dir, "touched-files.md"), maxCandidateManifestBytes)
	if err != nil {
		return nil, fmt.Errorf("candidate manifest: %w", err)
	}
	rows, err := parseCandidateManifest(raw)
	if err != nil {
		return nil, fmt.Errorf("candidate manifest: %w", err)
	}
	return rows, nil
}

// replaceManifestSectionBody rewrites the body of the single
// `## Candidate manifest` section, keeping every other byte of the file —
// headings, `## Touched files` narrative, and `## Review trail` — intact.
func replaceManifestSectionBody(raw []byte, body []string) ([]byte, error) {
	text := string(raw)
	crlf := strings.Contains(text, "\r\n")
	lines := strings.Split(text, "\n")
	start, count := -1, 0
	for i, line := range lines {
		if strings.TrimSpace(line) == "## Candidate manifest" {
			start = i + 1
			count++
		}
	}
	if count != 1 {
		return nil, errors.New("requires exactly one ## Candidate manifest section")
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "## ") {
			end = i
			break
		}
	}
	emitted := append([]string(nil), body...)
	if crlf {
		for i := range emitted {
			emitted[i] += "\r"
		}
	}
	if end < len(lines) {
		emitted = append(emitted, "")
		if crlf {
			emitted[len(emitted)-1] = "\r"
		}
	}
	lines = append(lines[:start], append(emitted, lines[end:]...)...)
	return []byte(strings.Join(lines, "\n")), nil
}
