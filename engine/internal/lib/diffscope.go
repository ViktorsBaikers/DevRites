package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
)

const diffScopeUsage = `usage: devrites-engine check diff-scope <slug> --allow <csv> | --allow-file <path> [--worktree|--staged|--base <ref>] [--allow-empty] [--baseline] [--verbose] [--cwd <dir>]
       devrites-engine check diff-scope <slug> --snapshot [--worktree|--staged|--base <ref>] [--cwd <dir>]

Verifies the changed-path set stays inside the declared allowlist. Modes:
  --worktree  git status --porcelain (modified + staged + untracked) [default]
  --staged    git diff --name-only --cached
  --base ref  git diff --name-only <ref> (tracked only)
Allowlist entries match exactly or as directory prefixes ("dir" covers "dir/...").
A rename counts as both its old and new path. Engine-written .devrites machine
files (claims ledger, dispatch/metrics/baseline/lease/lock files, ctx/ bundles)
are ignored; every other .devrites path is reported.
--snapshot records a fingerprint of every currently changed path in the
feature's diff-scope-baseline.json; run it before dispatching a writer.
--baseline then drops a changed path whose fingerprint still matches that
snapshot ("preexisting (unchanged)"); a baselined path whose bytes or mode
changed is checked like any other, and a run whose every change is
preexisting is an empty diff. More than 20 violations print grouped by
their first two path segments; --verbose lists every path.
Exit 0 clean (or snapshot written), 3 a path escaped the allowlist or the diff
is empty without --allow-empty.`

// diffScopeBaselineFile is the feature-root snapshot --snapshot writes and
// --baseline reads.
const diffScopeBaselineFile = "diff-scope-baseline.json"

// diffScopeListMax is how many paths (and path groups) print one per line
// before output is grouped; --verbose lifts it.
const diffScopeListMax = 20

// RunCheckDiffScope is the mechanical post-return contract gate: a writer's
// touched paths must be a subset of its declared allowlist before any reviewer
// spends context on the diff.
func RunCheckDiffScope(root string, args []string, stdout, stderr io.Writer) int {
	opts, code := parseDiffScopeArgs(root, args, stderr)
	if code != 0 {
		return code
	}
	if opts.snapshot {
		return opts.writeSnapshot(stdout, stderr)
	}
	if len(opts.allow) == 0 {
		fmt.Fprintln(stderr, "diff-scope: an allowlist is required (--allow or --allow-file)")
		return 2
	}
	changed, err := opts.changedPaths()
	if err != nil {
		fmt.Fprintf(stderr, "diff-scope: %v\n", err)
		return 2
	}
	if opts.baseline {
		var preexisting []string
		if changed, preexisting, err = opts.dropUnchanged(changed); err != nil {
			fmt.Fprintf(stderr, "diff-scope: %v\n", err)
			return 2
		}
		if len(preexisting) > diffScopeListMax && !opts.verbose {
			fmt.Fprintf(stdout, "diff-scope: %d preexisting (unchanged) path(s) excluded by baseline (--verbose lists them)\n", len(preexisting))
		} else {
			for _, path := range preexisting {
				fmt.Fprintf(stdout, "diff-scope: preexisting (unchanged): %s\n", path)
			}
		}
	}
	if len(changed) == 0 {
		if opts.allowEmpty {
			fmt.Fprintln(stdout, "diff-scope: empty (0 changed paths; --allow-empty)")
			return 0
		}
		fmt.Fprintln(stdout, "diff-scope: BLOCKED: empty diff; nothing was written (pass --allow-empty to accept)")
		return 3
	}
	var violations []string
	for _, path := range changed {
		if !allowedPath(opts.allow, path) {
			violations = append(violations, path)
		}
	}
	if len(violations) > 0 {
		printViolations(stdout, violations, opts.verbose)
		recordMetric(root, opts.slug, opts.phase, "diff-scope", "", 0)
		return 3
	}
	fmt.Fprintf(stdout, "diff-scope: ok (%d changed path(s) within allowlist)\n", len(changed))
	recordMetric(root, opts.slug, opts.phase, "diff-scope", "", int64(len(changed)))
	return 0
}

type diffScopeOpts struct {
	slug       string
	phase      string
	mode       string // worktree | staged | base
	base       string
	allow      []string
	allowFile  []string
	allowEmpty bool
	snapshot   bool
	baseline   bool
	verbose    bool
	cwd        string
	root       string
	// baselineFile is set when --snapshot or --baseline names a valid feature.
	baselineFile string
}

func parseDiffScopeArgs(root string, args []string, stderr io.Writer) (diffScopeOpts, int) {
	opts := diffScopeOpts{mode: "worktree", phase: "build", root: root}
	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--phase":
			opts.phase = argAt(args, i+1)
			i++
		case "--allow":
			for _, entry := range strings.Split(argAt(args, i+1), ",") {
				if entry = strings.TrimSpace(entry); entry != "" {
					opts.allow = append(opts.allow, entry)
				}
			}
			i++
		case "--allow-file":
			opts.allowFile = append(opts.allowFile, argAt(args, i+1))
			i++
		case "--staged":
			opts.mode = "staged"
		case "--worktree":
			opts.mode = "worktree"
		case "--base":
			opts.mode = "base"
			opts.base = argAt(args, i+1)
			if strings.HasPrefix(opts.base, "-") {
				fmt.Fprintln(stderr, "diff-scope: --base requires a ref (not a flag)")
				return opts, 2
			}
			i++
		case "--allow-empty":
			opts.allowEmpty = true
		case "--snapshot":
			opts.snapshot = true
		case "--baseline":
			opts.baseline = true
		case "--verbose":
			opts.verbose = true
		case "--cwd":
			opts.cwd = argAt(args, i+1)
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "diff-scope: unknown flag %q\n", args[i])
				return opts, 2
			}
			positional = append(positional, args[i])
		}
	}
	if len(positional) != 1 || opts.mode == "base" && opts.base == "" || opts.snapshot && opts.baseline {
		fmt.Fprint(stderr, diffScopeUsage)
		return opts, 2
	}
	opts.slug = positional[0]
	if opts.snapshot || opts.baseline {
		dir, err := devritespaths.ExistingFeatureDirChecked(root, opts.slug)
		if err != nil {
			fmt.Fprintf(stderr, "diff-scope: %v\n", err)
			return opts, 2
		}
		opts.baselineFile = filepath.Join(dir, diffScopeBaselineFile)
	}
	for _, file := range opts.allowFile {
		path := file
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, filepath.FromSlash(file))
		}
		data, err := os.ReadFile(path) // #nosec G304 -- path is rooted under the operator repository
		if err != nil {
			fmt.Fprintf(stderr, "diff-scope: allow-file %s: %v\n", file, err)
			return opts, 2
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			opts.allow = append(opts.allow, line)
		}
	}
	return opts, 0
}

func (o diffScopeOpts) gitDir() string {
	if o.cwd != "" {
		return o.cwd
	}
	return o.root
}

// writeSnapshot fingerprints every currently changed path so a later
// --baseline run can tell pre-existing dirt from a writer's edits.
func (o diffScopeOpts) writeSnapshot(stdout, stderr io.Writer) int {
	changed, err := o.changedPaths()
	var top string
	if err == nil {
		top, err = o.repoTop()
	}
	digests := make(map[string]string, len(changed))
	if err == nil {
		for _, path := range changed {
			digests[path] = pathDigest(top, path)
		}
		var data []byte
		if data, err = json.Marshal(digests); err == nil {
			err = os.WriteFile(o.baselineFile, append(data, '\n'), 0o600)
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "diff-scope: snapshot: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "diff-scope: snapshot (%d changed path(s)) -> %s\n", len(digests), diffScopeBaselineFile)
	return 0
}

// repoTop is the directory git reports changed paths relative to.
func (o diffScopeOpts) repoTop() (string, error) {
	out, err := runGitCommand(o.gitDir(), nil, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// dropUnchanged splits changed into paths still to check and paths whose
// fingerprint matches the snapshot. A missing snapshot fails closed.
func (o diffScopeOpts) dropUnchanged(changed []string) (kept, preexisting []string, err error) {
	data, err := os.ReadFile(o.baselineFile) // #nosec G304 -- validated feature workspace artifact
	if err != nil {
		return nil, nil, fmt.Errorf("baseline: %w (run --snapshot before dispatch)", err)
	}
	var baseline map[string]string
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, nil, fmt.Errorf("baseline %s is not valid JSON: %w", diffScopeBaselineFile, err)
	}
	top, err := o.repoTop()
	if err != nil {
		return nil, nil, err
	}
	for _, path := range changed {
		if was, ok := baseline[path]; ok && was != "" && was == pathDigest(top, path) {
			preexisting = append(preexisting, path)
			continue
		}
		kept = append(kept, path)
	}
	return kept, preexisting, nil
}

// pathDigest fingerprints one path under top by mode plus content (or link
// target). A deleted path is "absent"; an unreadable one is "", which never
// matches a baseline.
func pathDigest(top, rel string) string {
	full := filepath.Join(top, filepath.FromSlash(rel))
	info, err := os.Lstat(full)
	if errors.Is(err, fs.ErrNotExist) {
		return "absent"
	}
	if err != nil {
		return ""
	}
	var data []byte
	switch {
	case info.Mode()&fs.ModeSymlink != 0:
		target, err := os.Readlink(full)
		if err != nil {
			return ""
		}
		data = []byte(target)
	case info.Mode().IsRegular():
		if data, err = os.ReadFile(full); err != nil { // #nosec G304 -- path is a git-reported repository path
			return ""
		}
	default:
		return ""
	}
	sum := sha256.Sum256(data)
	return info.Mode().String() + ":" + hex.EncodeToString(sum[:])
}

// printViolations lists each violation, or above diffScopeListMax groups them
// by their first two path segments, largest group first, with a count and
// three examples.
func printViolations(w io.Writer, violations []string, verbose bool) {
	if verbose || len(violations) <= diffScopeListMax {
		for _, path := range violations {
			fmt.Fprintf(w, "diff-scope: BLOCKED: %s outside allowlist\n", path)
		}
		return
	}
	var keys []string
	groups := map[string][]string{}
	for _, path := range violations {
		parts := strings.SplitN(path, "/", 3)
		key := path
		if len(parts) == 3 {
			key = parts[0] + "/" + parts[1] + "/"
		}
		if _, seen := groups[key]; !seen {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], path)
	}
	sort.SliceStable(keys, func(i, j int) bool { return len(groups[keys[i]]) > len(groups[keys[j]]) })
	fmt.Fprintf(w, "diff-scope: BLOCKED: %d path(s) outside allowlist in %d group(s) (--verbose lists every path)\n", len(violations), len(keys))
	for i, key := range keys {
		if i == diffScopeListMax {
			fmt.Fprintf(w, "diff-scope: BLOCKED: %d more group(s)\n", len(keys)-i)
			break
		}
		paths := groups[key]
		fmt.Fprintf(w, "diff-scope: BLOCKED: %s %d path(s), e.g. %s\n", key, len(paths), strings.Join(paths[:min(3, len(paths))], ", "))
	}
}

func (o diffScopeOpts) changedPaths() ([]string, error) {
	dir := o.gitDir()
	var paths []string
	var err error
	switch o.mode {
	case "staged":
		paths, err = gitDiffNames(dir, "--cached")
	case "base":
		paths, err = gitDiffNames(dir, o.base)
	default:
		paths, err = gitStatusPaths(dir)
	}
	if err != nil {
		return nil, err
	}
	// The engine's own machine files are never task paths.
	control := devritesRepoPrefix(dir, o.root)
	kept := paths[:0]
	for _, path := range paths {
		if rel, ok := strings.CutPrefix(path, control+"/"); !ok || !engineMachineFile(rel) {
			kept = append(kept, path)
		}
	}
	return kept, nil
}

// engineMachineFiles are the feature-root files the engine itself writes:
// the dispatch ledger, metrics ledger, drift, regression, diff-scope, and tree
// baselines, the parallel lease, the feature lock, and the gate-execution lock.
var engineMachineFiles = map[string]bool{
	"dispatch.json":        true,
	diffScopeBaselineFile:  true,
	treeBaselineFile:       true,
	MetricsFile:            true,
	driftBaselineFile:      true,
	regressionBaselineFile: true,
	"parallel-lease.md":    true,
	".lock":                true,
	"gates.md.lock":        true,
}

// engineMachineFile reports whether rel, relative to the .devrites tree, is
// engine-written: the claims ledger and its lock, a feature's machine files,
// or its ctx/ context bundles. Every other .devrites path stays reported.
func engineMachineFile(rel string) bool {
	if rel == ClaimsFile || rel == ClaimsFile+".lock" {
		return true
	}
	rest, ok := strings.CutPrefix(rel, "work/")
	if !ok {
		return false
	}
	slug, name, ok := strings.Cut(rest, "/")
	return ok && devritespaths.ValidSlug(slug) && (engineMachineFiles[name] || strings.HasPrefix(name, "ctx/"))
}

// devritesRepoPrefix returns the project's control tree as a path relative to
// dir's repository root: normally ".devrites", or "sub/.devrites" when the
// project lives in a subdirectory of the repository.
func devritesRepoPrefix(dir, root string) string {
	if filepath.Base(root) != ".devrites" {
		return ".devrites"
	}
	top, err := runGitCommand(dir, nil, "rev-parse", "--show-toplevel")
	if err != nil {
		return ".devrites"
	}
	physicalTop, err := filepath.EvalSymlinks(strings.TrimSpace(string(top)))
	if err != nil {
		return ".devrites"
	}
	physicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return ".devrites"
	}
	rel, err := filepath.Rel(physicalTop, physicalRoot)
	if err != nil || !filepath.IsLocal(rel) {
		return ".devrites"
	}
	return filepath.ToSlash(rel)
}

// gitStatusPaths lists every path the writer touched: tracked modifications,
// staged entries, renames (new name, then the old name it replaced), and
// untracked files.
func gitStatusPaths(dir string) ([]string, error) {
	out, err := runGitCommand(dir, nil, "status", "--porcelain", "-uall", "-z")
	if err != nil {
		return nil, err
	}
	var paths []string
	// -z emits one NUL-terminated record per entry with paths unquoted; a
	// rename or copy carries a second record, new name first, then the old.
	records := strings.Split(string(out), "\x00")
	for i := 0; i < len(records); i++ {
		record := records[i]
		if len(record) < 4 {
			continue
		}
		path := record[3:]
		if path == "" {
			continue
		}
		if strings.ContainsAny(record[:2], "RC") {
			paths = append(paths, path)
			if i+1 < len(records) && records[i+1] != "" {
				i++
				paths = append(paths, records[i])
			}
			continue
		}
		paths = append(paths, path)
	}
	return paths, nil
}

// allowedPath reports whether path is inside the allowlist. An entry matches
// the exact path or, as a directory prefix, everything beneath it.
func allowedPath(allow []string, path string) bool {
	for _, entry := range allow {
		entry = strings.TrimSuffix(entry, "/")
		if path == entry || strings.HasPrefix(path, entry+"/") {
			return true
		}
	}
	return false
}
