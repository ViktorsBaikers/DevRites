package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
)

// treeBaselineFile is the feature-root map of dirty product paths to their
// pathDigest fingerprint (ADR-0038). `dispatch open` writes it when absent
// (run start: pre-existing dirt is accepted user work), the return that
// completes a wright wave takes that wave's write set, and `dispatch
// attribute` takes the paths it names. Paths no longer dirty keep their
// recorded fingerprint, so a checkpoint commit later collapsed back into the
// work tree (git reset --soft) is not flagged. The engine never reverts a file.
const treeBaselineFile = "tree-baseline.json"

// treeProject returns the project directory and its path inside the git
// repository ("" at the top, else "sub/"). ok is false when root is not a
// project's .devrites tree or the project is not in a git work tree; the
// tripwire is then off.
func treeProject(root string) (project, prefix string, ok bool) {
	if filepath.Base(root) != devritespaths.DevritesRootName {
		return "", "", false
	}
	project = filepath.Dir(root)
	out, err := runGitCommand(project, nil, "rev-parse", "--show-prefix")
	if err != nil {
		return "", "", false
	}
	return project, strings.TrimSpace(string(out)), true
}

// productDirt fingerprints every dirty product path, project-relative: git
// status entries (tracked, staged, untracked; gitignored files never appear)
// outside .devrites/ and .scratch/.
func productDirt(project, prefix string) (map[string]string, error) {
	paths, err := gitStatusPaths(project)
	if err != nil {
		return nil, err
	}
	dirt := make(map[string]string, len(paths))
	for _, path := range paths {
		rel, ok := strings.CutPrefix(path, prefix)
		if !ok || rel == ".devrites" || rel == ".scratch" || strings.HasPrefix(rel, ".devrites/") || strings.HasPrefix(rel, ".scratch/") {
			continue
		}
		dirt[rel] = pathDigest(project, rel)
	}
	return dirt, nil
}

// loadTreeBaseline reads the feature's baseline; ok is false when it is absent.
func loadTreeBaseline(root, slug string) (map[string]string, bool, error) {
	dir, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return nil, false, err
	}
	data, err := os.ReadFile(filepath.Join(dir, treeBaselineFile)) // #nosec G304 -- validated feature workspace artifact
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", treeBaselineFile, err)
	}
	var baseline map[string]string
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, false, fmt.Errorf("%s is not valid JSON: %w", treeBaselineFile, err)
	}
	return baseline, true, nil
}

// recordTreeBaseline rewrites the baseline from the current dirty set and
// carries clean paths' recorded values forward. A path
// whose fingerprint changed is taken when accept covers it (always when no
// baseline exists yet); otherwise it keeps its recorded value, or stays
// unrecorded, and remains flagged. It returns how many paths it took and
// writes nothing outside a git project. Callers hold the feature lock.
func recordTreeBaseline(root, slug string, accept func(string) bool) (int, error) {
	project, prefix, ok := treeProject(root)
	if !ok {
		return 0, nil
	}
	baseline, exists, err := loadTreeBaseline(root, slug)
	if err != nil {
		return 0, err
	}
	dirt, err := productDirt(project, prefix)
	if err != nil {
		return 0, err
	}
	next := make(map[string]string, len(dirt))
	taken := 0
	for path, digest := range dirt {
		was, recorded := baseline[path]
		switch {
		case recorded && was == digest:
			next[path] = was
		case !exists || accept(path):
			next[path] = digest
			taken++
		case recorded:
			next[path] = was
		}
	}
	for path, was := range baseline {
		if _, dirty := dirt[path]; !dirty {
			next[path] = was
		}
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return 0, err
	}
	return taken, state.AtomicWrite(filepath.Join(devritespaths.FeatureDir(root, slug), treeBaselineFile), append(data, '\n'), 0o600)
}

// ensureTreeBaseline records the run-start baseline when none exists.
func ensureTreeBaseline(root, slug string, stdout io.Writer) error {
	if _, exists, err := loadTreeBaseline(root, slug); err != nil || exists {
		return err
	}
	if _, _, ok := treeProject(root); !ok {
		return nil
	}
	n, err := recordTreeBaseline(root, slug, nil)
	if err == nil {
		fmt.Fprintf(stdout, "dispatch: tree baseline recorded (%d dirty product path(s) accepted as user work)\n", n)
	}
	return err
}

// unattributedEdits lists, sorted, the dirty product paths whose fingerprint
// differs from the baseline and that neither a live claim nor an in-flight
// wright wave's slice write set covers. A workspace without a baseline,
// or a project outside git, has none.
func unattributedEdits(root, slug string) ([]string, error) {
	baseline, exists, err := loadTreeBaseline(root, slug)
	if err != nil || !exists {
		return nil, err
	}
	project, prefix, ok := treeProject(root)
	if !ok {
		return nil, nil
	}
	dirt, err := productDirt(project, prefix)
	if err != nil {
		return nil, err
	}
	covered, err := liveAttribution(root, slug)
	if err != nil {
		return nil, err
	}
	var out []string
	for path, digest := range dirt {
		if was, recorded := baseline[path]; (!recorded || was != digest) && !covers(covered, path) {
			out = append(out, path)
		}
	}
	slices.Sort(out)
	return out, nil
}

// liveAttribution is every path a wright may be writing now: live claims in
// the ledger plus the slice write sets of open or sealed wright waves, in
// any phase.
func liveAttribution(root, slug string) ([]string, error) {
	latest, order, _, err := readClaims(claimsPath(root))
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, rec := range liveClaims(latest, order, time.Now().UTC()) {
		paths = append(paths, rec.Paths...)
	}
	f, _, err := loadDispatch(root, slug)
	if err != nil {
		return nil, err
	}
	for _, w := range f.Waves {
		if (w.State == "open" || w.State == "sealed") && isWrightWave(w) {
			paths = append(paths, sliceWriteSet(root, slug, w.Slices)...)
		}
	}
	return paths, nil
}

// waveWriteSet is what a completed wright wave owns: its slices' write
// sets plus the paths of its claim, released or not.
func waveWriteSet(root, slug string, w *dispatchWave) []string {
	paths := sliceWriteSet(root, slug, w.Slices)
	if w.Claim != "" {
		if latest, _, _, err := readClaims(claimsPath(root)); err == nil && latest[w.Claim] != nil {
			paths = append(paths, latest[w.Claim].Paths...)
		}
	}
	return paths
}

// sliceWriteSet is the tasks.md write set (Writer allowlist, else Files likely
// touched) of each named slice; an unreadable tasks.md yields none.
func sliceWriteSet(root, slug string, ids []string) []string {
	raw, err := readWorkspaceArtifact(root, slug, "tasks.md")
	if err != nil {
		return nil
	}
	byID, _ := sliceWritePaths(raw, ParseTaskGraph(raw))
	var out []string
	for id, paths := range byID {
		if containsSlice(ids, id) {
			out = append(out, paths...)
		}
	}
	return out
}

// covers reports whether a held path (file or directory) covers path.
func covers(held []string, path string) bool {
	return slices.ContainsFunc(held, func(h string) bool { return claimPathsOverlap(h, path) })
}

// unattributedMessage names the flagged paths and both remedies; neither
// stops for a human and the engine reverts nothing.
func unattributedMessage(slug string, paths []string) string {
	return fmt.Sprintf("unattributed root edit: %s; route each path through a bounded wright (its `dispatch open` names it with --claim <path>, or a slice write set covers it) or record the evidence that it is owner work, gate output, a returned wright outside a dispatch wave, or a protocol-sanctioned root write (Polish ADR promotion, DESIGN.md): devrites-engine dispatch %s attribute --reason <evidence> --path <path> (ADR-0038)",
		strings.Join(paths, ", "), slug)
}

// recordUnattributed logs one tripwire finding to metrics.jsonl.
func recordUnattributed(root, slug, phase string, paths []string) {
	recordMetricNote(root, slug, phase, "unattributed-edit", "", "paths="+strings.Join(paths, ","))
}

// CheckTreeBaseline is the `check candidate` tripwire: it fails while a dirty
// product path changed since the tree baseline with no live wright
// attribution. A workspace without tree-baseline.json always passes.
func CheckTreeBaseline(root, slug string) error {
	flagged, err := unattributedEdits(root, slug)
	if err != nil {
		return fmt.Errorf("tree baseline: %w", err)
	}
	if len(flagged) > 0 {
		recordUnattributed(root, slug, cursorPhase(root, slug), flagged)
		return errors.New(unattributedMessage(slug, flagged))
	}
	return nil
}

// attributeEdits records evidence-backed attribution for flagged paths: each
// must be an unattributed edit now, and its current fingerprint joins the
// baseline. The reason and paths land in metrics.jsonl.
func attributeEdits(root, slug, reason string, paths []string, stdout, stderr io.Writer) int {
	if strings.TrimSpace(reason) == "" || len(paths) == 0 {
		fmt.Fprintln(stderr, "dispatch: attribute needs --reason <evidence> and at least one --path <path>")
		return 2
	}
	code := 0
	err := state.WithFeatureLock(root, slug, func() error {
		flagged, err := unattributedEdits(root, slug)
		if err != nil {
			return err
		}
		for _, path := range paths {
			if !slices.Contains(flagged, path) {
				fmt.Fprintf(stderr, "dispatch: %s is not an unattributed edit (flagged: %s)\n", path, listOrNone(flagged))
				code = 2
				return nil
			}
		}
		if _, err := recordTreeBaseline(root, slug, func(p string) bool { return slices.Contains(paths, p) }); err != nil {
			return err
		}
		recordMetricNote(root, slug, cursorPhase(root, slug), "unattributed-edit", "", "attributed="+strings.Join(paths, ",")+" reason="+reason)
		for _, path := range paths {
			fmt.Fprintf(stdout, "dispatch: attributed %s (%s)\n", path, reason)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(stderr, "dispatch: attribute: %v\n", err)
		return 3
	}
	return code
}

func listOrNone(paths []string) string {
	if len(paths) == 0 {
		return "none"
	}
	return strings.Join(paths, ", ")
}

// openTripwire refuses a wave open while an unattributed edit remains that
// the new wave does not take on: only a wright wave (any phase) takes on
// paths, through its --claim paths and its slices' write sets.
func openTripwire(root, slug string, w *dispatchWave, claimPaths []string) error {
	flagged, err := unattributedEdits(root, slug)
	if err != nil {
		return fmt.Errorf("tree baseline: %w", err)
	}
	if isWrightWave(w) {
		own := append(sliceWriteSet(root, slug, w.Slices), claimPaths...)
		flagged = slices.DeleteFunc(flagged, func(p string) bool { return covers(own, p) })
	}
	if len(flagged) == 0 {
		return nil
	}
	recordUnattributed(root, slug, w.Phase, flagged)
	return errors.New(unattributedMessage(slug, flagged))
}

// absorbWave takes a completed wright wave's write set into the
// baseline. A failure only warns: the return is already recorded.
func absorbWave(root, slug string, w *dispatchWave, stderr io.Writer) {
	if !isWrightWave(w) {
		return
	}
	own := waveWriteSet(root, slug, w)
	if _, err := recordTreeBaseline(root, slug, func(p string) bool { return covers(own, p) }); err != nil {
		fmt.Fprintf(stderr, "dispatch: WARNING: %s not updated: %v\n", treeBaselineFile, err)
	}
}
