package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/forge"
	"github.com/devrites/devrites/internal/safepath"
)

const (
	reconcileBaseName           = ".reconcile-base"
	reconcileAllowlistName      = ".reconcile-allowlist"
	reconcileCheckedName        = ".reconcile-checked"
	reconcileDevritesName       = ".reconcile-devrites"
	reconcileObjectsName        = ".reconcile-objects"
	reconcileWrightStateName    = ".reconcile-wright-devrites"
	defaultWrightAllowlistName  = ".wright-allowlist"
	wrightAllowlistFileEnv      = "DEVRITES_WRIGHT_ALLOWLIST_FILE"
	reconcileDevritesPathPrefix = ".devrites/"
	reconcileAbortSchema        = "devrites.reconcile-abort-receipt.v1"
)

var (
	reconcileAbortReceiptName = regexp.MustCompile(`^\.reconcile-abort-([0-9a-f]{64})\.json$`)
	reconcileObjectID         = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)
	reconcileSHA256           = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type reconcileAbortReceipt struct {
	SchemaVersion           string   `json:"schemaVersion"`
	Slug                    string   `json:"slug"`
	BaselineTree            string   `json:"baselineTree"`
	SourceManifestSHA256    string   `json:"sourceManifestSHA256"`
	SourceEntryCount        int      `json:"sourceEntryCount"`
	CapturedAllowlistSHA256 string   `json:"capturedAllowlistSHA256"`
	RestoredPaths           []string `json:"restoredPaths"`
	WindowClosed            bool     `json:"windowClosed"`
}

// Reconcile enforces the source-write boundary around one dispatched
// slice-wright. The first `snapshot` captures the dirty worktree, private object
// database, exact orchestrator-authored allowlist, and .devrites state. A
// confirmed wright start captures the current canonical state separately, so
// retained-window recovery records are not attributed to the writer. After a
// clean check, another snapshot re-arms only the dispatch state for a retry
// while retaining the original source baseline.
// `check` compares the captured state with the current state and retains the
// immutable baseline for the later test-integrity gate.
// `restore-check` rolls back only source drift introduced after the last clean
// check. `abort` restores the original source baseline, records a
// content-addressed receipt, and closes a rejected writer window. `close`
// revalidates and then removes the private window artifacts.
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

	if slug == "" {
		s := slug
		if s == "" {
			s = "<unset>"
		}
		fmt.Fprintf(stderr, "reconcile: no active workspace (slug=%s): nothing to reconcile.\n", s)
		return 6
	}
	d, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(stderr, "reconcile: no active workspace (slug=%s): nothing to reconcile.\n", slug)
		} else {
			fmt.Fprintf(stderr, "reconcile: invalid workspace for %s: %v\n", slug, err)
		}
		return 6
	}

	base := filepath.Join(d, reconcileBaseName)
	capturedAllowlist := filepath.Join(d, reconcileAllowlistName)
	devritesSnapshot := filepath.Join(d, reconcileDevritesName)
	objects := filepath.Join(d, reconcileObjectsName)
	checked := filepath.Join(d, reconcileCheckedName)
	wrightState := filepath.Join(d, reconcileWrightStateName)

	closeWindow := func() error {
		var failures []string
		for _, privateFile := range []string{base, capturedAllowlist, devritesSnapshot, checked, wrightState} {
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
		info, err := os.Lstat(checked)
		if err != nil || !info.Mode().IsRegular() {
			fmt.Fprintln(stderr, `reconcile: active slice window has no clean check marker; run "reconcile check" before closing it.`)
			return 6
		}
		// Defense in depth: a prior marker proves the explicit check occurred;
		// rerunning it here binds close to the current source and canonical
		// workspace state rather than trusting a stale marker.
		if code := Reconcile(root, []string{"check", slug}, stdout, stderr); code != 0 {
			return code
		}
		if err := closeWindow(); err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot close slice window: %v\n", err)
			return 6
		}
		fmt.Fprintf(stdout, "reconcile: closed slice window for %s.\n", slug)
		return 0
	}

	cwd, _ := os.Getwd()
	gitRoot, err := gitToplevel(cwd)
	if err != nil {
		fmt.Fprintf(stderr, "reconcile: cannot resolve git worktree: %v\n", err)
		return 6
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
			state, err := captureReconcileDevritesState(root, devritesSnapshot, activeObjects, checked, wrightState)
			if err != nil {
				fmt.Fprintf(stderr, "reconcile: cannot refresh .devrites snapshot: %v\n", err)
				return 6
			}
			if err := writeDevritesState(devritesSnapshot, state); err != nil {
				fmt.Fprintf(stderr, "reconcile: cannot write refreshed .devrites snapshot: %v\n", err)
				return 6
			}
			for _, privateFile := range []string{checked, wrightState} {
				if err := os.Remove(privateFile); err != nil && !os.IsNotExist(err) {
					fmt.Fprintf(stderr, "reconcile: cannot arm refreshed slice window: %v\n", err)
					return 6
				}
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
		if err := os.WriteFile(capturedAllowlist, renderWrightAllowlist(allowed), 0o600); err != nil {
			_ = closeWindow()
			fmt.Fprintf(stderr, "reconcile: cannot capture wright allowlist: %v\n", err)
			return 6
		}
		state, err := captureReconcileDevritesState(root, devritesSnapshot, objects, checked, wrightState)
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

	case "abort":
		baseTree, env, activeObjects, active, err := loadReconcileBaseline(gitRoot, d)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: invalid snapshot: %v\n", err)
			return 6
		}
		if !active {
			fmt.Fprintln(stderr, `reconcile: no active snapshot to abort.`)
			return 6
		}
		allowed, err := readWrightAllowlist(gitRoot, capturedAllowlist)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: invalid captured allowlist: %v\n", err)
			return 6
		}
		allowlistBytes := renderWrightAllowlist(allowed)
		allowlistDigest := sha256.Sum256(allowlistBytes)

		nowTree, err := worktreeTree(gitRoot, activeObjects, root)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot capture worktree for abort: %v\n", err)
			return 6
		}
		changed, err := changedTreePaths(gitRoot, env, baseTree, nowTree)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot compare abort baseline: %v\n", err)
			return 6
		}
		changed = sourceTreePaths(changed)
		if err := restoreTreePaths(gitRoot, env, baseTree, changed); err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot restore rejected writer source delta: %v\n", err)
			return 6
		}

		restoredTree, err := worktreeTree(gitRoot, activeObjects, root)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot verify aborted worktree: %v\n", err)
			return 6
		}
		remaining, err := changedTreePaths(gitRoot, env, baseTree, restoredTree)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot verify abort baseline: %v\n", err)
			return 6
		}
		if remaining = sourceTreePaths(remaining); len(remaining) != 0 {
			fmt.Fprintln(stderr, "reconcile: rejected writer source delta was not fully restored; retained window left open.")
			return 6
		}
		manifestDigest, entryCount, err := sourceTreeManifest(gitRoot, env, baseTree)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot fingerprint restored source baseline: %v\n", err)
			return 6
		}
		restoredDigest, restoredEntryCount, err := sourceTreeManifest(gitRoot, env, restoredTree)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot fingerprint restored worktree: %v\n", err)
			return 6
		}
		if restoredDigest != manifestDigest || restoredEntryCount != entryCount {
			fmt.Fprintln(stderr, "reconcile: restored source manifest does not match the original baseline; retained window left open.")
			return 6
		}
		receipt := reconcileAbortReceipt{
			SchemaVersion:           reconcileAbortSchema,
			Slug:                    slug,
			BaselineTree:            baseTree,
			SourceManifestSHA256:    manifestDigest,
			SourceEntryCount:        entryCount,
			CapturedAllowlistSHA256: hex.EncodeToString(allowlistDigest[:]),
			RestoredPaths:           changed,
			WindowClosed:            true,
		}
		receiptData, receiptName, err := renderReconcileAbortReceipt(receipt)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot render abort receipt: %v\n", err)
			return 6
		}
		receiptPath := filepath.Join(d, receiptName)
		pendingReceipt, err := stageContentAddressedReceipt(receiptPath, receiptData)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot stage abort receipt: %v\n", err)
			return 6
		}
		if pendingReceipt != "" {
			defer func() { _ = os.Remove(pendingReceipt) }()
		}
		if err := closeWindow(); err != nil {
			fmt.Fprintf(stderr, "reconcile: source baseline restored, but cannot close rejected slice window: %v\n", err)
			return 6
		}
		if err := commitContentAddressedReceipt(receiptPath, pendingReceipt, receiptData); err != nil {
			fmt.Fprintf(stderr, "reconcile: source baseline restored and window closed, but cannot persist abort receipt: %v\n", err)
			return 6
		}
		fmt.Fprintf(stdout, "reconcile: aborted rejected slice window for %s; restored %d source path(s); receipt %s.\n", slug, len(changed), receiptName)
		return 0

	case "restore-check":
		_, env, activeObjects, active, err := loadReconcileBaseline(gitRoot, d)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: invalid snapshot: %v\n", err)
			return 6
		}
		if !active {
			fmt.Fprintln(stderr, `reconcile: no snapshot (.reconcile-base/.reconcile-objects): nothing can be restored.`)
			return 6
		}
		checkedData, err := os.ReadFile(checked)
		if err != nil {
			fmt.Fprintln(stderr, `reconcile: active slice window has no clean check marker; nothing can be restored.`)
			return 6
		}
		checkedTree := strings.TrimSpace(string(checkedData))
		if checkedTree == "" || strings.ContainsAny(checkedTree, " \t\r\n") {
			fmt.Fprintf(stderr, "reconcile: invalid %s: expected one tree id\n", reconcileCheckedName)
			return 6
		}
		nowTree, err := worktreeTree(gitRoot, activeObjects, root)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot capture worktree for restore: %v\n", err)
			return 6
		}
		changed, err := restoreTreeDelta(gitRoot, env, checkedTree, nowTree)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot restore post-check source drift: %v\n", err)
			return 6
		}
		restoredTree, err := worktreeTree(gitRoot, activeObjects, root)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot verify restored worktree: %v\n", err)
			return 6
		}
		if restoredTree != checkedTree {
			fmt.Fprintln(stderr, "reconcile: restored worktree does not match the last clean check; retained window left open.")
			return 6
		}
		if len(changed) == 0 {
			fmt.Fprintln(stdout, "reconcile: no post-check source drift to restore.")
			return 0
		}
		fmt.Fprintln(stdout, "reconcile: restored only the post-check source delta:")
		for _, changedPath := range changed {
			fmt.Fprintf(stdout, "  - %s\n", changedPath)
		}
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
		beforeDevritesPath := devritesSnapshot
		if _, err := os.Lstat(wrightState); err == nil {
			beforeDevritesPath = wrightState
		} else if !os.IsNotExist(err) {
			fmt.Fprintf(stderr, "reconcile: invalid wright-start .devrites snapshot: %v\n", err)
			return 6
		}
		beforeDevrites, err := readDevritesState(beforeDevritesPath)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: invalid .devrites snapshot: %v\n", err)
			return 6
		}
		beforeDevrites = withoutRootOwnedOperationalState(beforeDevrites)
		afterDevrites, err := captureReconcileDevritesState(root, devritesSnapshot, objects, checked, wrightState)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot compare .devrites: %v\n", err)
			return 6
		}

		nowTree, err := worktreeTree(gitRoot, objects, root)
		if err != nil {
			fmt.Fprintf(stderr, "reconcile: cannot capture worktree: %v\n", err)
			return 6
		}
		if checkedData, checkedErr := os.ReadFile(checked); checkedErr == nil {
			checkedTree := strings.TrimSpace(string(checkedData))
			if checkedTree == "" || strings.ContainsAny(checkedTree, " \t\r\n") {
				fmt.Fprintf(stderr, "reconcile: invalid %s: expected one tree id\n", reconcileCheckedName)
				return 6
			}
			if checkedTree != nowTree {
				changedAfterCheck, err := changedTreePaths(gitRoot, env, checkedTree, nowTree)
				if err != nil {
					fmt.Fprintf(stderr, "reconcile: cannot compare with the last clean check: %v\n", err)
					return 6
				}
				sort.Strings(changedAfterCheck)
				fmt.Fprintln(stderr, "reconcile: STOP: source changed after the last clean check; root-owned artifact gates must not edit source:")
				for _, changedPath := range changedAfterCheck {
					fmt.Fprintf(stderr, "  - %s\n", changedPath)
				}
				return 5
			}
		} else if !os.IsNotExist(checkedErr) {
			fmt.Fprintf(stderr, "reconcile: cannot read %s: %v\n", reconcileCheckedName, checkedErr)
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
		fmt.Fprintln(stderr, "reconcile: usage: devrites-engine reconcile <snapshot|check|restore-check|abort|close> [slug]")
		return 6
	}
}

// CaptureReconcileWrightBoundary records canonical state at the confirmed
// wright start. The original source baseline remains untouched; reconciliation
// can therefore distinguish root recovery records that predate the writer from
// canonical mutations made while the writer is active.
func CaptureReconcileWrightBoundary(root, slug string) error {
	d := featureDir(root, slug)
	if slug == "" || !isDir(d) {
		return fmt.Errorf("no active workspace")
	}
	base := filepath.Join(d, reconcileBaseName)
	capturedAllowlist := filepath.Join(d, reconcileAllowlistName)
	devritesSnapshot := filepath.Join(d, reconcileDevritesName)
	objects := filepath.Join(d, reconcileObjectsName)
	checked := filepath.Join(d, reconcileCheckedName)
	wrightState := filepath.Join(d, reconcileWrightStateName)

	for _, filename := range []string{base, capturedAllowlist} {
		info, err := os.Lstat(filename)
		if err != nil || !info.Mode().IsRegular() {
			if err == nil {
				err = fmt.Errorf("not a regular file")
			}
			return fmt.Errorf("invalid reconcile window %s: %w", filepath.Base(filename), err)
		}
	}
	if !isDir(objects) {
		return fmt.Errorf("invalid reconcile window %s: not a directory", filepath.Base(objects))
	}
	if _, err := readDevritesState(devritesSnapshot); err != nil {
		return fmt.Errorf("invalid reconcile window %s: %w", filepath.Base(devritesSnapshot), err)
	}
	state, err := captureReconcileDevritesState(
		root,
		devritesSnapshot,
		objects,
		checked,
		wrightState,
	)
	if err != nil {
		return err
	}
	return writeDevritesState(wrightState, state)
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
		operational, err := reconcileRootOwnedOperationalFile(filename, rel, entry)
		if err != nil {
			return err
		}
		if operational {
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

// reconcileRootOwnedOperationalFile excludes engine/root-owned operational
// records from the writer delta. Their owning hooks/gates validate them where
// needed; root activity must not be attributed to the slice writer.
func reconcileRootOwnedOperationalFile(filename, rel string, entry os.DirEntry) (bool, error) {
	if !entry.Type().IsRegular() || !reconcileRootOwnedOperationalPath(rel) {
		return false, nil
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	parts := strings.Split(rel, "/")
	if len(parts) == 3 && parts[2] == recoveryAttemptsFile {
		if _, err := readRecoveryAttempts(filename); err != nil {
			return false, fmt.Errorf("validate root-owned recovery ledger %s: %w", rel, err)
		}
	}
	if len(parts) == 3 && reconcileAbortReceiptName.MatchString(parts[2]) {
		data, err := os.ReadFile(filename)
		if err != nil {
			return false, fmt.Errorf("read reconcile abort receipt %s: %w", rel, err)
		}
		match := reconcileAbortReceiptName.FindStringSubmatch(parts[2])
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != match[1] {
			return false, fmt.Errorf("validate reconcile abort receipt %s: content digest does not match filename", rel)
		}
		var receipt reconcileAbortReceipt
		if err := json.Unmarshal(data, &receipt); err != nil {
			return false, fmt.Errorf("validate reconcile abort receipt %s: %w", rel, err)
		}
		if receipt.SchemaVersion != reconcileAbortSchema || !receipt.WindowClosed ||
			receipt.Slug != parts[1] || !reconcileObjectID.MatchString(receipt.BaselineTree) ||
			!reconcileSHA256.MatchString(receipt.SourceManifestSHA256) ||
			!reconcileSHA256.MatchString(receipt.CapturedAllowlistSHA256) ||
			receipt.SourceEntryCount < 0 || !validReceiptPaths(receipt.RestoredPaths) {
			return false, fmt.Errorf("validate reconcile abort receipt %s: invalid receipt fields", rel)
		}
	}
	return true, nil
}

func validReceiptPaths(paths []string) bool {
	previous := ""
	for _, receiptPath := range paths {
		if receiptPath == "" || receiptPath == "." || receiptPath == ".devrites" ||
			strings.HasPrefix(receiptPath, reconcileDevritesPathPrefix) ||
			path.Clean(receiptPath) != receiptPath ||
			(previous != "" && receiptPath <= previous) {
			return false
		}
		previous = receiptPath
	}
	return true
}

func reconcileRootOwnedOperationalPath(rel string) bool {
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "timeline.jsonl" {
		return true
	}
	parts := strings.Split(rel, "/")
	if len(parts) != 3 ||
		(parts[0] != "work" && parts[0] != "features") ||
		!slugToken.MatchString(parts[1]) {
		return false
	}
	switch parts[2] {
	case "events.jsonl",
		recoveryAttemptsFile,
		".red",
		"handoff.md",
		".a1-guard.log",
		".reviewer-ro.log",
		".stop-gate.log",
		".wright-scope.log":
		return true
	default:
		return reconcileAbortReceiptName.MatchString(parts[2])
	}
}

func withoutRootOwnedOperationalState(state []devritesStateEntry) []devritesStateEntry {
	filtered := state[:0]
	for _, item := range state {
		if strings.HasPrefix(item.Fingerprint, "file:") &&
			reconcileRootOwnedOperationalPath(item.Path) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
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

// restoreTreeDelta materializes the last clean tree in a private directory and
// copies back only paths that changed afterward. It never touches the real Git
// index and rejects parent symlink escapes before deleting or writing.
func restoreTreeDelta(gitRoot string, env []string, checkedTree, currentTree string) ([]string, error) {
	changed, err := changedTreePaths(gitRoot, env, checkedTree, currentTree)
	if err != nil {
		return nil, err
	}
	sort.Strings(changed)
	if len(changed) == 0 {
		return nil, nil
	}
	if err := restoreTreePaths(gitRoot, env, checkedTree, changed); err != nil {
		return nil, err
	}
	return changed, nil
}

func restoreTreePaths(gitRoot string, env []string, targetTree string, changed []string) error {
	materialized, err := os.MkdirTemp("", "devrites-reconcile-restore-*")
	if err != nil {
		return fmt.Errorf("create restore directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(materialized) }()
	index := filepath.Join(materialized, "index")
	restoreEnv := append(append([]string{}, env...), "GIT_INDEX_FILE="+index)
	if _, err := reconcileGitOutput(gitRoot, restoreEnv, "read-tree", targetTree); err != nil {
		return err
	}
	materializedTree := filepath.Join(materialized, "tree")
	if err := os.MkdirAll(materializedTree, 0o700); err != nil {
		return fmt.Errorf("create restore tree: %w", err)
	}
	gitPrefix := filepath.ToSlash(materializedTree) + "/"
	if _, err := reconcileGitOutput(
		gitRoot,
		restoreEnv,
		"-c", "core.autocrlf=false",
		"checkout-index", "--all", "--force", "--prefix="+gitPrefix,
	); err != nil {
		return err
	}

	for _, changedPath := range changed {
		if changedPath == "." || changedPath == ".devrites" ||
			strings.HasPrefix(changedPath, reconcileDevritesPathPrefix) ||
			path.Clean(changedPath) != changedPath {
			return fmt.Errorf("unsafe restore path %q", changedPath)
		}
		target := filepath.Join(gitRoot, filepath.FromSlash(changedPath))
		targetParent := filepath.Dir(target)
		if !safepath.WithinResolved(targetParent, gitRoot) {
			return fmt.Errorf("restore parent escapes repository through a symlink: %s", changedPath)
		}
		source := filepath.Join(materializedTree, filepath.FromSlash(changedPath))
		sourceInfo, sourceErr := os.Lstat(source)
		if sourceErr != nil && !os.IsNotExist(sourceErr) {
			return fmt.Errorf("inspect clean source %s: %w", changedPath, sourceErr)
		}
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("remove source path %s: %w", changedPath, err)
		}
		if os.IsNotExist(sourceErr) {
			continue
		}
		if err := os.MkdirAll(targetParent, 0o755); err != nil {
			return fmt.Errorf("create restore parent for %s: %w", changedPath, err)
		}
		switch {
		case sourceInfo.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(source)
			if err != nil {
				return fmt.Errorf("read clean symlink %s: %w", changedPath, err)
			}
			if err := os.Symlink(link, target); err != nil {
				return fmt.Errorf("restore symlink %s: %w", changedPath, err)
			}
		case sourceInfo.Mode().IsRegular():
			data, err := os.ReadFile(source)
			if err != nil {
				return fmt.Errorf("read clean file %s: %w", changedPath, err)
			}
			if err := os.WriteFile(target, data, sourceInfo.Mode().Perm()); err != nil {
				return fmt.Errorf("restore file %s: %w", changedPath, err)
			}
		default:
			return fmt.Errorf("unsupported clean tree entry for %s: %s", changedPath, sourceInfo.Mode())
		}
	}
	return nil
}

func sourceTreePaths(paths []string) []string {
	source := make([]string, 0, len(paths))
	for _, changedPath := range paths {
		if changedPath == ".devrites" || strings.HasPrefix(changedPath, reconcileDevritesPathPrefix) {
			continue
		}
		source = append(source, changedPath)
	}
	sort.Strings(source)
	return source
}

func sourceTreeManifest(gitRoot string, env []string, tree string) (string, int, error) {
	out, err := reconcileGitOutput(gitRoot, env, "ls-tree", "-r", "-z", "--full-tree", tree)
	if err != nil {
		return "", 0, err
	}
	hash := sha256.New()
	count := 0
	for _, entry := range strings.Split(string(out), "\x00") {
		if entry == "" {
			continue
		}
		tab := strings.IndexByte(entry, '\t')
		if tab < 0 {
			return "", 0, fmt.Errorf("invalid ls-tree entry")
		}
		entryPath := entry[tab+1:]
		if entryPath == ".devrites" || strings.HasPrefix(entryPath, reconcileDevritesPathPrefix) {
			continue
		}
		_, _ = hash.Write([]byte(entry))
		_, _ = hash.Write([]byte{0})
		count++
	}
	return hex.EncodeToString(hash.Sum(nil)), count, nil
}

func renderReconcileAbortReceipt(receipt reconcileAbortReceipt) ([]byte, string, error) {
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return nil, "", err
	}
	data = append(data, '\n')
	digest := sha256.Sum256(data)
	name := fmt.Sprintf(".reconcile-abort-%x.json", digest)
	return data, name, nil
}

func stageContentAddressedReceipt(filename string, data []byte) (string, error) {
	existing, readErr := os.ReadFile(filename)
	if readErr == nil {
		if string(existing) != string(data) {
			return "", fmt.Errorf("existing content-addressed receipt does not match %s", filepath.Base(filename))
		}
		return "", nil
	}
	if !os.IsNotExist(readErr) {
		return "", readErr
	}
	pending := strings.TrimSuffix(filename, ".json") + ".pending"
	file, err := os.OpenFile(pending, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(pending)
		return "", err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(pending)
		return "", err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(pending)
		return "", err
	}
	if err := os.Chmod(pending, 0o444); err != nil {
		_ = os.Remove(pending)
		return "", err
	}
	return pending, nil
}

func commitContentAddressedReceipt(filename, pending string, data []byte) error {
	if pending == "" {
		return nil
	}
	if err := os.Rename(pending, filename); err != nil {
		return err
	}
	persisted, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	if string(persisted) != string(data) {
		return fmt.Errorf("persisted abort receipt does not match staged content")
	}
	return nil
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

	_, readErr := runGitCommand(gitRoot, env, "read-tree", "HEAD")
	if readErr != nil {
		readFailure := fmt.Errorf("git read-tree HEAD: %w", readErr)
		// An unborn branch is the one valid reason HEAD cannot seed the index.
		// Distinguish it from a corrupt or unreadable existing ref so the gate
		// still fails closed on real repository errors.
		refOut, symbolicErr := runGitCommand(gitRoot, env, "symbolic-ref", "--quiet", "HEAD")
		if symbolicErr != nil {
			return "", readFailure
		}
		ref := strings.TrimSpace(string(refOut))
		if _, err := runGitCommand(gitRoot, env, "show-ref", "--verify", "--quiet", ref); err == nil {
			return "", readFailure
		} else if code, ok := gitErrorExitCode(err); !ok || code != 1 {
			return "", fmt.Errorf("inspect HEAD ref %s: %w", ref, err)
		}

		if _, err := runGitCommand(gitRoot, env, "read-tree", "--empty"); err != nil {
			return "", fmt.Errorf("git read-tree --empty: %w", err)
		}
	}

	if _, err := runGitCommand(gitRoot, env, "add", "-A"); err != nil {
		return "", fmt.Errorf("git add -A: %w", err)
	}
	for _, excludedRoot := range excludedRoots {
		rel, err := filepath.Rel(gitRoot, excludedRoot)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if _, err := runGitCommand(
			gitRoot, env, "rm", "-r", "--cached", "--ignore-unmatch", "--",
			":(top,literal)"+filepath.ToSlash(rel),
		); err != nil {
			return "", fmt.Errorf("exclude %s from worktree snapshot: %w", rel, err)
		}
	}

	out, err := runGitCommand(gitRoot, env, "write-tree")
	if err != nil {
		return "", fmt.Errorf("git write-tree: %w", err)
	}
	tree := strings.TrimSpace(string(out))
	if tree == "" {
		return "", fmt.Errorf("git write-tree: empty tree sha")
	}
	return tree, nil
}

func reconcileGitEnv(gitRoot, objectDir string) ([]string, error) {
	out, err := runGitCommand(gitRoot, nil, "rev-parse", "--git-common-dir")
	if err != nil {
		return nil, fmt.Errorf("git rev-parse --git-common-dir: %w", err)
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
	return runGitCommand(gitRoot, env, args...)
}
