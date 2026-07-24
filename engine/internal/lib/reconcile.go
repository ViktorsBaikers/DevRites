package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/forge"
	"github.com/devrites/devrites/internal/safepath"
)

const (
	reconcileBaseName           = ".reconcile-base"
	reconcileAllowlistName      = ".reconcile-allowlist"
	reconcileCheckedName        = ".reconcile-checked"
	reconcileDevritesName       = ".reconcile-devrites"
	reconcileInlineName         = ".reconcile-inline"
	reconcileObjectsName        = ".reconcile-objects"
	reconcileObjectMarkerName   = ".devrites-baseline"
	reconcileLegacyClaimedName  = ".reconcile-claimed"
	defaultWrightAllowlistName  = ".wright-allowlist"
	wrightAllowlistFileEnv      = "DEVRITES_WRIGHT_ALLOWLIST_FILE"
	reconcileDevritesPathPrefix = ".devrites/"
)

// Reconcile enforces the source-write boundary around one dispatched
// slice-wright. The first `snapshot` captures the dirty worktree, private object
// database, exact orchestrator-authored allowlist, and .devrites state. After a
// clean check, another snapshot re-arms only the dispatch state for a retry while
// retaining the original source baseline.
// `check` compares the captured state with the current state and retains the
// immutable baseline for the later test-integrity and package-existence gates.
// `close` explicitly ends the window and removes its private artifacts.
//
//	0  clean check, snapshot/close completed, or skipped (not a git repo)
//	5  VIOLATION: a path changed outside the orchestrator allowlist
//	6  setup error: bad args or missing/corrupt lifecycle state
func Reconcile(root string, args []string, stdout, stderr io.Writer) int {
	mode := argAt(args, 0)
	slug := argAt(args, 1)
	if slug == "" {
		slug = activeSlug(root)
	}

	d := featureDir(root, slug)
	if slug == "" || !isDir(d) {
		s := slug
		if s == "" {
			s = "<unset>"
		}
		fmt.Fprintf(stderr, "reconcile: no active workspace (slug=%s): nothing to reconcile.\n", s)
		return 6
	}

	base := filepath.Join(d, reconcileBaseName)
	capturedAllowlist := filepath.Join(d, reconcileAllowlistName)
	devritesSnapshot := filepath.Join(d, reconcileDevritesName)
	inline := filepath.Join(d, reconcileInlineName)
	objects := filepath.Join(d, reconcileObjectsName)
	legacyClaimed := filepath.Join(d, reconcileLegacyClaimedName)
	checked := filepath.Join(d, reconcileCheckedName)

	closeWindow := func() error {
		var failures []string
		for _, privateFile := range []string{base, capturedAllowlist, devritesSnapshot, inline, legacyClaimed, checked} {
			if err := os.Remove(privateFile); err != nil && !os.IsNotExist(err) {
				failures = append(failures, fmt.Sprintf("%s: %v", filepath.Base(privateFile), err))
			}
		}
		if err := os.RemoveAll(objects); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", filepath.Base(objects), err))
		}
		if len(failures) > 0 {
			return fmt.Errorf("%s", strings.Join(failures, "; "))
		}
		return nil
	}

	if mode == "close" {
		if err := closeWindow(); err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot close slice window: %v\n", err)
			return 6
		}
		fmt.Fprintf(stdout, "reconcile: closed slice window for %s.\n", slug)
		return 0
	}

	cwd, _ := os.Getwd()
	gitRoot := gitToplevel(cwd)
	if gitRoot == "" {
		fmt.Fprintln(stderr, "reconcile: not a git repo: gate skipped, verify the diff by hand.")
		return 0
	}

	switch mode {
	case "snapshot":
		_, _, activeObjects, active, err := loadReconcileBaseline(gitRoot, d)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot refresh invalid slice window: %v\n", err)
			return 6
		}
		if active {
			checkedTree, err := os.ReadFile(checked)
			if err != nil {
				fmt.Fprintln(stderr, `reconcile: active slice window has no clean check marker; run "reconcile check" before refreshing the dispatch snapshot.`)
				return 6
			}
			nowTree, err := worktreeTree(gitRoot, activeObjects, root)
			if err != nil {
				fmt.Fprintf(stderr, "reconcile: cannot refresh worktree: %v\n", err)
				return 6
			}
			if strings.TrimSpace(string(checkedTree)) != nowTree {
				fmt.Fprintln(stderr, `reconcile: source changed after the last clean check; run "reconcile check" before refreshing the dispatch snapshot.`)
				return 6
			}
			allowlistPath, err := wrightAllowlistPath(gitRoot, d)
			if err != nil {
				fmt.Fprintf(stderr, "reconcile: invalid wright allowlist location: %v\n", err)
				return 6
			}
			allowed, err := readWrightAllowlist(gitRoot, allowlistPath)
			if err != nil {
				fmt.Fprintf(stderr, "reconcile: invalid wright allowlist %s: %v\n", allowlistPath, err)
				return 6
			}
			if err := os.WriteFile(capturedAllowlist, renderWrightAllowlist(allowed), 0o600); err != nil {
				fmt.Fprintf(stderr, "reconcile: cannot refresh wright allowlist: %v\n", err)
				return 6
			}
			state, err := captureReconcileDevritesState(root, devritesSnapshot, activeObjects, checked, inline)
			if err != nil {
				fmt.Fprintf(stderr, "reconcile: cannot refresh .devrites snapshot: %v\n", err)
				return 6
			}
			if err := writeDevritesState(devritesSnapshot, state); err != nil {
				fmt.Fprintf(stderr, "reconcile: cannot write refreshed .devrites snapshot: %v\n", err)
				return 6
			}
			if err := os.Remove(checked); err != nil {
				fmt.Fprintf(stderr, "reconcile: cannot arm refreshed slice window: %v\n", err)
				return 6
			}
			fmt.Fprintf(stdout, "reconcile: dispatch snapshot refreshed for %s; original slice baseline retained.\n", slug)
			return 0
		}

		if err := closeWindow(); err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot clear stale slice window: %v\n", err)
			return 6
		}

		allowlistPath, err := wrightAllowlistPath(gitRoot, d)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: invalid wright allowlist location: %v\n", err)
			return 6
		}
		allowed, err := readWrightAllowlist(gitRoot, allowlistPath)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: invalid wright allowlist %s: %v\n", allowlistPath, err)
			return 6
		}

		tree, err := worktreeTree(gitRoot, objects, root)
		if err != nil {
			_ = closeWindow()
			fmt.Fprintf(stderr, "reconcile: cannot snapshot worktree: %v\n", err)
			return 6
		}
		if err := os.WriteFile(base, []byte(tree+"\n"), 0o644); err != nil {
			_ = closeWindow()
			fmt.Fprintf(stderr, "reconcile: cannot write snapshot: %v\n", err)
			return 6
		}
		if err := os.WriteFile(filepath.Join(objects, reconcileObjectMarkerName), []byte(tree+"\n"), 0o600); err != nil {
			_ = closeWindow()
			fmt.Fprintf(stderr, "reconcile: cannot seal snapshot object database: %v\n", err)
			return 6
		}
		if err := os.WriteFile(capturedAllowlist, renderWrightAllowlist(allowed), 0o600); err != nil {
			_ = closeWindow()
			fmt.Fprintf(stderr, "reconcile: cannot capture wright allowlist: %v\n", err)
			return 6
		}
		state, err := captureReconcileDevritesState(root, devritesSnapshot, objects, checked, inline)
		if err != nil {
			_ = closeWindow()
			fmt.Fprintf(stderr, "reconcile: cannot snapshot .devrites: %v\n", err)
			return 6
		}
		if err := writeDevritesState(devritesSnapshot, state); err != nil {
			_ = closeWindow()
			fmt.Fprintf(stderr, "reconcile: cannot write .devrites snapshot: %v\n", err)
			return 6
		}
		fmt.Fprintf(stdout, "reconcile: snapshot captured for %s.\n", slug)
		return 0

	case "check":
		baseTree, env, _, active, err := loadReconcileBaseline(gitRoot, d)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: invalid snapshot: %v\n", err)
			return 6
		}
		if !active {
			fmt.Fprintln(stderr, `reconcile: no snapshot (.reconcile-base/.reconcile-objects): run "devrites-engine reconcile snapshot" before dispatch.`)
			return 6
		}

		allowed, err := readWrightAllowlist(gitRoot, capturedAllowlist)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: invalid captured allowlist: %v\n", err)
			return 6
		}
		beforeDevrites, err := readDevritesState(devritesSnapshot)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: invalid .devrites snapshot: %v\n", err)
			return 6
		}
		afterDevrites, err := captureReconcileDevritesState(root, devritesSnapshot, objects, checked, inline)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot compare .devrites: %v\n", err)
			return 6
		}

		nowTree, err := worktreeTree(gitRoot, objects, root)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot capture worktree: %v\n", err)
			return 6
		}
		changed, err := changedTreePaths(gitRoot, env, baseTree, nowTree)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot compare worktree: %v\n", err)
			return 6
		}

		violations := map[string]struct{}{}
		for _, changedPath := range changed {
			// Canonical DevRites state is fingerprinted below. It may be tracked
			// or unignored, so do not treat the same path as source as well.
			if changedPath == ".devrites" || strings.HasPrefix(changedPath, reconcileDevritesPathPrefix) {
				continue
			}
			if _, ok := allowed[changedPath]; !ok {
				violations[changedPath] = struct{}{}
			}
		}
		for _, changedPath := range changedDevritesPaths(beforeDevrites, afterDevrites) {
			violations[reconcileDevritesPathPrefix+changedPath] = struct{}{}
		}

		if len(violations) > 0 {
			paths := make([]string, 0, len(violations))
			for changedPath := range violations {
				paths = append(paths, changedPath)
			}
			sort.Strings(paths)
			fmt.Fprintf(stderr, "reconcile: STOP: %d path(s) changed outside the orchestrator-authorized wright allowlist (A1 breach):\n", len(paths))
			for _, changedPath := range paths {
				fmt.Fprintf(stderr, "  - %s\n", changedPath)
			}
			fmt.Fprintln(stderr, "Reject this writer result. Preserve pre-snapshot user changes, restore only the unauthorized slice delta, then re-dispatch with an exact allowlist.")
			return 5
		}
		if err := os.WriteFile(checked, []byte(nowTree+"\n"), 0o600); err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot record clean check: %v\n", err)
			return 6
		}
		fmt.Fprintln(stdout, `reconcile: OK: every slice change matches the orchestrator allowlist; baseline retained for post-slice gates (run "reconcile close" afterward).`)
		return 0

	default:
		fmt.Fprintln(stderr, "reconcile: usage: devrites-engine reconcile <snapshot|check|close> [slug]")
		return 6
	}
}

func wrightAllowlistPath(gitRoot, workspace string) (string, error) {
	configured := os.Getenv(wrightAllowlistFileEnv)
	var filename string
	if configured == "" {
		filename = filepath.Join(workspace, defaultWrightAllowlistName)
	} else if filepath.IsAbs(configured) {
		filename = filepath.Clean(configured)
	} else {
		filename = filepath.Join(gitRoot, filepath.FromSlash(configured))
	}
	if !safepath.WithinResolved(filename, workspace) {
		return "", fmt.Errorf("path must remain inside the active workspace")
	}
	return filename, nil
}

func readWrightAllowlist(gitRoot, filename string) (map[string]struct{}, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return parseWrightAllowlist(gitRoot, data)
}

func parseWrightAllowlist(gitRoot string, data []byte) (map[string]struct{}, error) {
	allowed := map[string]struct{}{}
	for lineNumber, candidate := range splitLinesNoTrailing(data) {
		line := lineNumber + 1
		switch {
		case candidate == "":
			return nil, fmt.Errorf("line %d is empty", line)
		case strings.TrimSpace(candidate) != candidate:
			return nil, fmt.Errorf("line %d is not normalized: %q", line, candidate)
		case strings.Contains(candidate, `\`):
			return nil, fmt.Errorf("line %d uses a non-Git path separator: %q", line, candidate)
		case filepath.IsAbs(candidate) || path.IsAbs(candidate) || windowsAbsolutePath(candidate):
			return nil, fmt.Errorf("line %d is absolute: %q", line, candidate)
		case candidate == ".." || strings.HasPrefix(candidate, "../"):
			return nil, fmt.Errorf("line %d traverses outside the repository: %q", line, candidate)
		case candidate == "." || path.Clean(candidate) != candidate:
			return nil, fmt.Errorf("line %d is not normalized: %q", line, candidate)
		case candidate == ".devrites" || strings.HasPrefix(candidate, reconcileDevritesPathPrefix):
			return nil, fmt.Errorf("line %d targets orchestrator-owned .devrites state: %q", line, candidate)
		}
		if _, duplicate := allowed[candidate]; duplicate {
			return nil, fmt.Errorf("line %d duplicates %q", line, candidate)
		}
		fullPath := filepath.Join(gitRoot, filepath.FromSlash(candidate))
		if !safepath.WithinResolved(fullPath, gitRoot) {
			return nil, fmt.Errorf("line %d escapes the repository through a symlink: %q", line, candidate)
		}
		info, err := os.Stat(fullPath)
		switch {
		case err == nil && info.IsDir():
			return nil, fmt.Errorf("line %d names a directory, not a file: %q", line, candidate)
		case err != nil && !os.IsNotExist(err):
			return nil, fmt.Errorf("line %d cannot inspect %q: %w", line, candidate, err)
		}
		allowed[candidate] = struct{}{}
	}
	return allowed, nil
}

func windowsAbsolutePath(candidate string) bool {
	return len(candidate) >= 3 &&
		((candidate[0] >= 'A' && candidate[0] <= 'Z') || (candidate[0] >= 'a' && candidate[0] <= 'z')) &&
		candidate[1] == ':' && (candidate[2] == '/' || candidate[2] == '\\')
}

func renderWrightAllowlist(allowed map[string]struct{}) []byte {
	paths := make([]string, 0, len(allowed))
	for allowedPath := range allowed {
		paths = append(paths, allowedPath)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil
	}
	return []byte(strings.Join(paths, "\n") + "\n")
}

type devritesStateEntry struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

// captureReconcileDevritesState keeps Forge's typed lifecycle manifest private
// while continuing to fingerprint every other .forge path, including the human
// forge-report.md. A manifest is ignored only after the Forge owner validates
// its unique run, derived path, and primary root.
func captureReconcileDevritesState(root string, ignoredPaths ...string) ([]devritesStateEntry, error) {
	manifests, err := validatedForgeManifestPaths(root)
	if err != nil {
		return nil, err
	}
	return captureDevritesState(root, append(ignoredPaths, manifests...)...)
}

func validatedForgeManifestPaths(root string) ([]string, error) {
	work := filepath.Join(root, "work")
	info, err := os.Lstat(work)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, nil
	}
	slugs, err := os.ReadDir(work)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, slug := range slugs {
		if !slug.IsDir() || slug.Type()&os.ModeSymlink != 0 {
			continue
		}
		forgeDir := filepath.Join(work, slug.Name(), ".forge")
		forgeInfo, err := os.Lstat(forgeDir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !forgeInfo.IsDir() || forgeInfo.Mode()&os.ModeSymlink != 0 {
			continue
		}
		runs, err := os.ReadDir(forgeDir)
		if err != nil {
			return nil, err
		}
		for _, run := range runs {
			if !run.IsDir() || run.Type()&os.ModeSymlink != 0 {
				continue
			}
			manifestPath := filepath.Join(forgeDir, run.Name(), "manifest.json")
			manifestInfo, err := os.Lstat(manifestPath)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if !manifestInfo.Mode().IsRegular() {
				continue
			}
			manifest, validatedPath, err := forge.Load(filepath.Dir(root), run.Name())
			if err != nil {
				return nil, fmt.Errorf("reconcile: invalid Forge manifest %s: %w", manifestPath, err)
			}
			if filepath.Clean(validatedPath) != filepath.Clean(manifestPath) || manifest.FeatureSlug != slug.Name() {
				return nil, fmt.Errorf("reconcile: Forge manifest ownership mismatch: %s", manifestPath)
			}
			paths = append(paths, validatedPath)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func captureDevritesState(root string, ignoredPaths ...string) ([]devritesStateEntry, error) {
	ignored := make(map[string]struct{}, len(ignoredPaths))
	for _, ignoredPath := range ignoredPaths {
		rel, err := filepath.Rel(root, ignoredPath)
		if err != nil {
			return nil, err
		}
		ignored[filepath.Clean(rel)] = struct{}{}
	}

	var state []devritesStateEntry
	err := filepath.WalkDir(root, func(filename string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filename == root {
			return nil
		}
		rel, err := filepath.Rel(root, filename)
		if err != nil {
			return err
		}
		rel = filepath.Clean(rel)
		if _, skip := ignored[rel]; skip {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		fingerprint, err := devritesFingerprint(filename, entry)
		if err != nil {
			return err
		}
		state = append(state, devritesStateEntry{
			Path:        filepath.ToSlash(rel),
			Fingerprint: fingerprint,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(state, func(i, j int) bool { return state[i].Path < state[j].Path })
	return state, nil
}

func devritesFingerprint(filename string, entry os.DirEntry) (string, error) {
	info, err := entry.Info()
	if err != nil {
		return "", err
	}
	mode := info.Mode()
	switch {
	case mode.IsRegular():
		data, err := os.ReadFile(filename)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(data)
		return fmt.Sprintf("file:%#o:%s", mode.Perm(), hex.EncodeToString(sum[:])), nil
	case mode.IsDir():
		return fmt.Sprintf("dir:%#o", mode.Perm()), nil
	case mode&os.ModeSymlink != 0:
		target, err := os.Readlink(filename)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256([]byte(target))
		return fmt.Sprintf("symlink:%s", hex.EncodeToString(sum[:])), nil
	default:
		return fmt.Sprintf("other:%s", mode.String()), nil
	}
}

func writeDevritesState(filename string, state []devritesStateEntry) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filename, data, 0o600)
}

func readDevritesState(filename string) ([]devritesStateEntry, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var state []devritesStateEntry
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for i, entry := range state {
		if entry.Path == "" || entry.Fingerprint == "" ||
			path.Clean(entry.Path) != entry.Path || path.IsAbs(entry.Path) ||
			entry.Path == ".." || strings.HasPrefix(entry.Path, "../") || strings.Contains(entry.Path, `\`) {
			return nil, fmt.Errorf("entry %d is invalid", i+1)
		}
		if _, duplicate := seen[entry.Path]; duplicate {
			return nil, fmt.Errorf("entry %d duplicates %q", i+1, entry.Path)
		}
		seen[entry.Path] = struct{}{}
	}
	return state, nil
}

func changedDevritesPaths(before, after []devritesStateEntry) []string {
	beforeMap := make(map[string]string, len(before))
	afterMap := make(map[string]string, len(after))
	for _, entry := range before {
		beforeMap[entry.Path] = entry.Fingerprint
	}
	for _, entry := range after {
		afterMap[entry.Path] = entry.Fingerprint
	}
	changed := map[string]struct{}{}
	for statePath, fingerprint := range beforeMap {
		if afterMap[statePath] != fingerprint {
			changed[statePath] = struct{}{}
		}
	}
	for statePath, fingerprint := range afterMap {
		if beforeMap[statePath] != fingerprint {
			changed[statePath] = struct{}{}
		}
	}
	paths := make([]string, 0, len(changed))
	for statePath := range changed {
		paths = append(paths, statePath)
	}
	sort.Strings(paths)
	return paths
}

// worktreeTree captures the working tree (tracked + untracked, .gitignore
// honored) as a git tree object via a throwaway index, excluding separately
// fingerprinted DevRites roots. Seeding from HEAD keeps committed paths tracked
// even when a later ignore rule matches them. The user's real index is never
// touched.
//
// A failing "git add -A" must not be ignored: write-tree would still succeed
// against the seeded throwaway index. Both the snapshot and the check could then
// record the same stale tree, and the gate would report OK on precisely the A1
// breach it exists to catch. Surface the error and let the caller fail closed.
func worktreeTree(gitRoot, objectDir string, excludedRoots ...string) (string, error) {
	idx := filepath.Join(os.TempDir(), fmt.Sprintf("devrites-reconcile-%d.idx", os.Getpid()))
	_ = os.Remove(idx)
	defer func() { _ = os.Remove(idx) }()
	if err := os.MkdirAll(objectDir, 0o700); err != nil {
		return "", fmt.Errorf("create object directory: %w", err)
	}
	env, err := reconcileGitEnv(gitRoot, objectDir)
	if err != nil {
		return "", err
	}
	env = append(env, "GIT_INDEX_FILE="+idx)

	read := exec.Command("git", "-C", gitRoot, "read-tree", "HEAD")
	read.Env = env
	readOut, readErr := read.CombinedOutput()
	if readErr != nil {
		readFailure := fmt.Errorf("git read-tree HEAD: %w", readErr)
		if detail := strings.TrimSpace(string(readOut)); detail != "" {
			readFailure = fmt.Errorf("git read-tree HEAD: %w: %s", readErr, detail)
		}
		// An unborn branch is the one valid reason HEAD cannot seed the index.
		// Distinguish it from a corrupt or unreadable existing ref so the gate
		// still fails closed on real repository errors.
		symbolic := exec.Command("git", "-C", gitRoot, "symbolic-ref", "--quiet", "HEAD")
		symbolic.Env = env
		refOut, symbolicErr := symbolic.CombinedOutput()
		if symbolicErr != nil {
			return "", readFailure
		}
		ref := strings.TrimSpace(string(refOut))
		exists := exec.Command("git", "-C", gitRoot, "show-ref", "--verify", "--quiet", ref)
		exists.Env = env
		if err := exists.Run(); err == nil {
			return "", readFailure
		} else if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
			return "", fmt.Errorf("inspect HEAD ref %s: %w", ref, err)
		}

		empty := exec.Command("git", "-C", gitRoot, "read-tree", "--empty")
		empty.Env = env
		if out, err := empty.CombinedOutput(); err != nil {
			if detail := strings.TrimSpace(string(out)); detail != "" {
				return "", fmt.Errorf("git read-tree --empty: %w: %s", err, detail)
			}
			return "", fmt.Errorf("git read-tree --empty: %w", err)
		}
	}

	add := exec.Command("git", "-C", gitRoot, "add", "-A")
	add.Env = env
	addOut, err := add.CombinedOutput()
	if err != nil {
		if detail := strings.TrimSpace(string(addOut)); detail != "" {
			return "", fmt.Errorf("git add -A: %w: %s", err, detail)
		}
		return "", fmt.Errorf("git add -A: %w", err)
	}
	for _, excludedRoot := range excludedRoots {
		rel, err := filepath.Rel(gitRoot, excludedRoot)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		rm := exec.Command(
			"git", "-C", gitRoot, "rm", "-r", "--cached", "--ignore-unmatch", "--",
			":(top,literal)"+filepath.ToSlash(rel),
		)
		rm.Env = env
		if out, err := rm.CombinedOutput(); err != nil {
			return "", fmt.Errorf("exclude %s from worktree snapshot: %w: %s", rel, err, strings.TrimSpace(string(out)))
		}
	}

	write := exec.Command("git", "-C", gitRoot, "write-tree")
	write.Env = env
	out, err := write.CombinedOutput()
	if err != nil {
		if detail := strings.TrimSpace(string(out)); detail != "" {
			return "", fmt.Errorf("git write-tree: %w: %s", err, detail)
		}
		return "", fmt.Errorf("git write-tree: %w", err)
	}
	tree := strings.TrimSpace(string(out))
	if tree == "" {
		return "", fmt.Errorf("git write-tree: empty tree sha")
	}
	return tree, nil
}

func reconcileGitEnv(gitRoot, objectDir string) ([]string, error) {
	common := exec.Command("git", "-C", gitRoot, "rev-parse", "--git-common-dir")
	out, err := common.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git rev-parse --git-common-dir: %w: %s", err, strings.TrimSpace(string(out)))
	}
	commonDir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(gitRoot, commonDir)
	}
	alternates := []string{filepath.Join(commonDir, "objects")}
	if inherited := os.Getenv("GIT_OBJECT_DIRECTORY"); inherited != "" {
		alternates = append(alternates, inherited)
	}
	if inherited := os.Getenv("GIT_ALTERNATE_OBJECT_DIRECTORIES"); inherited != "" {
		alternates = append(alternates, inherited)
	}
	return append(os.Environ(),
		"GIT_OBJECT_DIRECTORY="+objectDir,
		"GIT_ALTERNATE_OBJECT_DIRECTORIES="+strings.Join(alternates, string(os.PathListSeparator)),
	), nil
}

// loadReconcileBaseline returns the retained slice baseline and its private Git
// environment. With no lifecycle state at all it reports active=false so gates
// invoked outside a slice can compare against HEAD. A partial or corrupt
// lifecycle never silently falls back to HEAD.
func loadReconcileBaseline(gitRoot, workspace string) (baseTree string, env []string, objectDir string, active bool, err error) {
	basePath := filepath.Join(workspace, reconcileBaseName)
	objectDir = filepath.Join(workspace, reconcileObjectsName)

	baseInfo, baseErr := os.Lstat(basePath)
	objectsInfo, objectsErr := os.Lstat(objectDir)
	baseMissing := os.IsNotExist(baseErr)
	objectsMissing := os.IsNotExist(objectsErr)
	if baseMissing && objectsMissing {
		return "", nil, "", false, nil
	}
	if baseErr != nil && !baseMissing {
		return "", nil, "", false, fmt.Errorf("cannot inspect %s: %w", reconcileBaseName, baseErr)
	}
	if objectsErr != nil && !objectsMissing {
		return "", nil, "", false, fmt.Errorf("cannot inspect %s: %w", reconcileObjectsName, objectsErr)
	}
	if baseMissing || objectsMissing {
		return "", nil, "", false, fmt.Errorf("partial lifecycle: %s and %s must both exist", reconcileBaseName, reconcileObjectsName)
	}
	if !baseInfo.Mode().IsRegular() {
		return "", nil, "", false, fmt.Errorf("%s is not a regular file", reconcileBaseName)
	}
	if !objectsInfo.IsDir() {
		return "", nil, "", false, fmt.Errorf("%s is not a directory", reconcileObjectsName)
	}

	data, err := os.ReadFile(basePath)
	if err != nil {
		return "", nil, "", false, fmt.Errorf("read %s: %w", reconcileBaseName, err)
	}
	baseTree = strings.TrimSpace(string(data))
	if baseTree == "" || strings.ContainsAny(baseTree, " \t\r\n") {
		return "", nil, "", false, fmt.Errorf("%s does not contain one tree id", reconcileBaseName)
	}
	markerPath := filepath.Join(objectDir, reconcileObjectMarkerName)
	markerInfo, err := os.Lstat(markerPath)
	if err != nil {
		return "", nil, "", false, fmt.Errorf("%s is not a sealed object database: %w", reconcileObjectsName, err)
	}
	if !markerInfo.Mode().IsRegular() {
		return "", nil, "", false, fmt.Errorf("%s marker is not a regular file", reconcileObjectsName)
	}
	marker, err := os.ReadFile(markerPath)
	if err != nil {
		return "", nil, "", false, fmt.Errorf("read %s marker: %w", reconcileObjectsName, err)
	}
	if strings.TrimSpace(string(marker)) != baseTree {
		return "", nil, "", false, fmt.Errorf("%s marker does not match %s", reconcileObjectsName, reconcileBaseName)
	}
	env, err = reconcileGitEnv(gitRoot, objectDir)
	if err != nil {
		return "", nil, "", false, err
	}
	if _, err := reconcileGitOutput(gitRoot, env, "cat-file", "-e", baseTree+"^{tree}"); err != nil {
		return "", nil, "", false, fmt.Errorf("%s tree is unavailable: %w", reconcileBaseName, err)
	}
	return baseTree, env, objectDir, true, nil
}

type sliceTreeRange struct {
	base    string
	current string
	env     []string
	cleanup func()
}

// captureSliceTreeRange returns an immutable base→current pair. During an open
// reconcile window it reuses the retained private object database. For a direct
// standalone gate it captures the current dirty tree in a temporary private
// object database and compares it with HEAD.
func captureSliceTreeRange(gitRoot, devritesRoot, workspace string) (sliceTreeRange, error) {
	base, _, objectDir, active, err := loadReconcileBaseline(gitRoot, workspace)
	if err != nil {
		return sliceTreeRange{}, err
	}
	cleanup := func() {}
	if !active {
		objectDir, err = os.MkdirTemp("", "devrites-gate-objects-*")
		if err != nil {
			return sliceTreeRange{}, fmt.Errorf("create temporary object database: %w", err)
		}
		cleanup = func() { _ = os.RemoveAll(objectDir) }
		base = "HEAD"
	}
	current, err := worktreeTree(gitRoot, objectDir, devritesRoot)
	if err != nil {
		cleanup()
		return sliceTreeRange{}, err
	}
	env, err := reconcileGitEnv(gitRoot, objectDir)
	if err != nil {
		cleanup()
		return sliceTreeRange{}, err
	}
	return sliceTreeRange{base: base, current: current, env: env, cleanup: cleanup}, nil
}

func changedTreePaths(gitRoot string, env []string, baseTree, currentTree string) ([]string, error) {
	out, err := reconcileGitOutput(
		gitRoot,
		env,
		"diff", "--name-only", "--no-renames", "-z", baseTree, currentTree, "--",
	)
	if err != nil {
		return nil, err
	}
	var paths []string
	for rawPath := range strings.SplitSeq(string(out), "\x00") {
		if rawPath != "" {
			paths = append(paths, filepath.ToSlash(filepath.Clean(rawPath)))
		}
	}
	return paths, nil
}

func treeFileContent(gitRoot string, env []string, tree, filename string) ([]byte, bool, error) {
	entry, err := reconcileGitOutput(gitRoot, env, "ls-tree", "-z", tree, "--", ":(literal)"+filename)
	if err != nil {
		return nil, false, err
	}
	if len(entry) == 0 {
		return nil, false, nil
	}
	out, err := reconcileGitOutput(gitRoot, env, "show", tree+":"+filename)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func reconcileGitOutput(gitRoot string, env []string, args ...string) ([]byte, error) {
	commandArgs := append([]string{"-C", gitRoot}, args...)
	cmd := exec.Command("git", commandArgs...)
	if env != nil {
		cmd.Env = env
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, detail)
		}
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}
