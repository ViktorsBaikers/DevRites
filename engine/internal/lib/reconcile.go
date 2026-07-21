package lib

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Reconcile enforces that the orchestrator never edits source itself — only the
// dispatched slice-wright writes code and tests, while the orchestrator writes
// bookkeeping under .devrites/. `snapshot` records the working tree just before
// the wright runs; `check` diffs it afterward and flags any changed path that is
// neither under .devrites/ nor in the wright's claimed set. The workspace is
// <root>/features/<slug>.
//
//	0  clean check, snapshot written, inline fallback, or skipped (not a git repo)
//	5  VIOLATION — source changed outside the wright's claimed set
//	6  setup error — bad args, missing snapshot, or missing claimed manifest
func Reconcile(root string, args []string, stdout, stderr io.Writer) int {
	mode := argAt(args, 0)
	slug := argAt(args, 1)
	if slug == "" {
		slug = activeSlug(root)
	}

	cwd, _ := os.Getwd()
	gitRoot := gitToplevel(cwd)
	if gitRoot == "" {
		fmt.Fprintln(stderr, "reconcile: not a git repo — gate skipped, verify the diff by hand.")
		return 0
	}

	d := featureDir(root, slug)
	base := filepath.Join(d, ".reconcile-base")
	claimed := filepath.Join(d, ".reconcile-claimed")
	inline := filepath.Join(d, ".reconcile-inline")

	if slug == "" || !isDir(d) {
		s := slug
		if s == "" {
			s = "<unset>"
		}
		fmt.Fprintf(stderr, "reconcile: no active workspace (slug=%s) — nothing to reconcile.\n", s)
		return 6
	}

	closeWindow := func() { _ = os.Remove(base); _ = os.Remove(claimed); _ = os.Remove(inline) }

	switch mode {
	case "snapshot":
		_ = os.Remove(inline) // clear any stale sentinel / prior-slice manifest
		_ = os.Remove(claimed)
		tree, err := worktreeTree(gitRoot)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot snapshot worktree: %v\n", err)
			return 6
		}
		if err := os.WriteFile(base, []byte(tree+"\n"), 0o644); err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot write snapshot: %v\n", err)
			return 6
		}
		fmt.Fprintf(stdout, "reconcile: snapshot captured for %s.\n", slug)
		return 0

	case "check":
		if isFile(inline) {
			fmt.Fprintln(stderr, "reconcile: inline-fallback run (no wright dispatched) — gate skipped.")
			closeWindow()
			return 0
		}
		if !isFile(base) {
			fmt.Fprintln(stderr, `reconcile: no snapshot (.reconcile-base) — run "devrites-engine reconcile snapshot" before dispatch.`)
			return 6
		}
		if !isFile(claimed) {
			fmt.Fprintln(stderr, "reconcile: no claimed manifest (.reconcile-claimed) — write the wright Files-changed paths first.")
			return 6
		}
		baseBytes, err := os.ReadFile(base)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot read snapshot: %v\n", err)
			return 6
		}
		baseTree := strings.TrimSpace(string(baseBytes))
		if baseTree == "" {
			fmt.Fprintln(stderr, "reconcile: empty snapshot (.reconcile-base) — re-run \"devrites-engine reconcile snapshot\" before dispatch.")
			return 6
		}
		nowTree, err := worktreeTree(gitRoot)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot capture worktree: %v\n", err)
			return 6
		}

		claimedSet := claimedPaths(claimed)
		viol := 0
		var violist strings.Builder
		for _, f := range gitDiffNames(gitRoot, baseTree, nowTree) {
			if f == "" || strings.HasPrefix(f, ".devrites/") {
				continue
			}
			if _, ok := claimedSet[f]; !ok {
				viol++
				fmt.Fprintf(&violist, "  - %s\n", f)
			}
		}
		if viol > 0 {
			fmt.Fprintf(stderr, "reconcile: STOP — %d source file(s) changed OUTSIDE the wright (A1 breach):\n", viol)
			fmt.Fprint(stderr, violist.String())
			fmt.Fprintln(stderr, "The orchestrator must NOT edit source. Re-dispatch the wright (continue it once) or revert these.")
			return 5
		}
		closeWindow()
		fmt.Fprintln(stdout, "reconcile: OK — every source change is the wright's; orchestrator touched only bookkeeping.")
		return 0

	default:
		fmt.Fprintln(stderr, "reconcile: usage: devrites-engine reconcile <snapshot|check> [slug]")
		return 6
	}
}

// worktreeTree captures the whole working tree (tracked + untracked, .gitignore
// honored) as a git tree object via a throwaway index, so the user's real index
// is never touched.
//
// A failing "git add -A" must not be ignored: write-tree would still succeed
// against the empty throwaway index and return the empty-tree sha. Both the
// snapshot and the check would then record that same sha, the diff would come
// back empty, and the gate would report OK on precisely the A1 breach it exists
// to catch. Surface the error and let the caller fail closed.
func worktreeTree(gitRoot string) (string, error) {
	idx := filepath.Join(os.TempDir(), fmt.Sprintf("devrites-reconcile-%d.idx", os.Getpid()))
	_ = os.Remove(idx)
	defer func() { _ = os.Remove(idx) }()
	env := append(os.Environ(), "GIT_INDEX_FILE="+idx)

	add := exec.Command("git", "-C", gitRoot, "add", "-A")
	add.Env = env
	add.Stdout, add.Stderr = io.Discard, io.Discard
	if err := add.Run(); err != nil {
		return "", fmt.Errorf("git add -A: %w", err)
	}

	write := exec.Command("git", "-C", gitRoot, "write-tree")
	write.Env = env
	out, err := write.Output()
	if err != nil {
		return "", fmt.Errorf("git write-tree: %w", err)
	}
	tree := strings.TrimSpace(string(out))
	if tree == "" {
		return "", fmt.Errorf("git write-tree: empty tree sha")
	}
	return tree, nil
}

// claimedPaths reads the .reconcile-claimed manifest into a set of the exact
// paths the wright reported changing.
func claimedPaths(path string) map[string]struct{} {
	set := map[string]struct{}{}
	data, err := os.ReadFile(path)
	if err != nil {
		return set
	}
	for _, line := range splitLinesNoTrailing(data) {
		set[line] = struct{}{}
	}
	return set
}
