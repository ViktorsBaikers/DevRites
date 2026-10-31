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
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/acceptance"
	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/state"
)

// maxBundleBytes bounds one drafter bundle read. A bundle carries hunks for a
// handful of planning artifacts, each capped at state.MaxArtifactBytes.
const maxBundleBytes = 8 << 20

// trackedIDRE matches the workspace ID families whose silent disappearance is
// a planning defect: an AC, slice, or horizon item never vanishes unrecorded.
var trackedIDRE = regexp.MustCompile(`\b(?:AC|REQ|HZN|SLICE|PF|INV|DEC|PROH|EDGE|DRIFT|ASM|Q|EVID)-\d+(?:-[A-Z][A-Z0-9]*)*\b`)

// datedHeadingRE finds a dated section heading. In the current-state planning
// artifacts it marks an appended repair delta that leaves the superseded clause
// beside its replacement, growing the file and the next review's surface.
var datedHeadingRE = regexp.MustCompile(`(?m)^#{2,4}\s.*\b\d{4}-\d{2}-\d{2}\b`)

var currentStatePlanning = map[string]bool{
	"architecture.md": true, "plan.md": true, "tasks.md": true, "test-plan.md": true, "traceability.md": true,
}

// candidateBundle is the plan-drafter's candidate_files result. Unknown fields
// (acceptance_map, technical_decisions, ...) are the root's to reconcile and
// are ignored here.
type candidateBundle struct {
	BaseHashes     map[string]string `json:"base_hashes"`
	CandidateFiles []candidateFile   `json:"candidate_files"`
	RetiredIDs     []string          `json:"retired_ids"`
}

type candidateFile struct {
	Path      string          `json:"path"`
	Operation string          `json:"operation"`
	Content   *string         `json:"content"`
	Hunks     []candidateHunk `json:"hunks"`
}

type candidateHunk struct {
	ID      string `json:"id"`
	OldText string `json:"old_text"`
	NewText string `json:"new_text"`
}

// bundleTarget is every requested change to one workspace file, merged across
// bundles so anchor-disjoint drafter clusters apply against one shared base.
type bundleTarget struct {
	rel      string
	op       string // create | replace | patch
	content  string
	hunks    []candidateHunk
	pins     []string // base hashes named by each bundle that touches rel
	before   []byte
	existed  bool
	after    []byte
	hunkSeen int
}

// RunApplyBundle implements `devrites-engine state apply-bundle <slug>
// <bundle.json>... [--dry-run]`. It replaces the root's hand fold of drafter
// hunks: every bundle is validated against the current bytes (base hash pin,
// unique non-overlapping old_text, no lost tracked ID, no new task-graph
// problem, no new material budget overshoot) and then all files are written,
// or none are. Several bundles combine when their hunks are disjoint.
//
// Exit codes:
//
//	0  applied (or dry-run clean)
//	2  usage error
//	3  blocked: a bundle failed validation; nothing was written
//	1  internal/IO error
func RunApplyBundle(root string, args []string, stdout, stderr io.Writer) int {
	dryRun := false
	var positional []string
	for _, arg := range args {
		if arg == "--dry-run" {
			dryRun = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			fmt.Fprintln(stderr, "usage: devrites-engine state apply-bundle <slug> <bundle.json>... [--dry-run]")
			return 2
		}
		positional = append(positional, arg)
	}
	if len(positional) < 2 {
		fmt.Fprintln(stderr, "usage: devrites-engine state apply-bundle <slug> <bundle.json>... [--dry-run]")
		return 2
	}
	slug := positional[0]
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "apply-bundle: BLOCKED: workspace for %s: %v\n", slug, err)
		return 3
	}
	if err := state.RequireWorkspaceSchema(root, slug); err != nil {
		fmt.Fprintf(stderr, "apply-bundle: BLOCKED: %v\n", err)
		return 3
	}
	bundles := make([]candidateBundle, 0, len(positional)-1)
	for _, name := range positional[1:] {
		bundle, err := readCandidateBundle(name)
		if err != nil {
			fmt.Fprintf(stderr, "apply-bundle: BLOCKED: bundle %s: %v\n", name, err)
			return 3
		}
		bundles = append(bundles, bundle)
	}

	code := 0
	if err := state.WithFeatureLock(root, slug, func() error {
		code = applyBundlesLocked(slug, workspace, bundles, dryRun, stdout, stderr)
		return nil
	}); err != nil {
		fmt.Fprintf(stderr, "apply-bundle: %v\n", err)
		return 1
	}
	return code
}

func readCandidateBundle(name string) (candidateBundle, error) {
	var bundle candidateBundle
	raw, err := readBoundedRegularFile(name, maxBundleBytes)
	if err != nil {
		return bundle, err
	}
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return bundle, fmt.Errorf("not a JSON candidate bundle: %w", err)
	}
	if len(bundle.CandidateFiles) == 0 {
		return bundle, errors.New("candidate_files is empty")
	}
	return bundle, nil
}

func applyBundlesLocked(slug, workspace string, bundles []candidateBundle, dryRun bool, stdout, stderr io.Writer) int {
	ws, err := os.OpenRoot(workspace)
	if err != nil {
		fmt.Fprintf(stderr, "apply-bundle: %v\n", err)
		return 1
	}
	defer func() { _ = ws.Close() }()

	targets, retired, problems := collectBundleTargets(slug, bundles)
	if len(problems) == 0 {
		problems = materializeTargets(ws, targets)
	}
	if len(problems) == 0 {
		problems = checkBundleResult(targets, retired)
	}
	if len(problems) > 0 {
		for _, problem := range problems {
			fmt.Fprintf(stderr, "apply-bundle: BLOCKED: %s\n", problem)
		}
		fmt.Fprintln(stderr, "apply-bundle: nothing written")
		return 3
	}

	hunks := 0
	for _, target := range targets {
		hunks += len(target.hunks)
		lines := state.CountLines(target.after)
		budget := ""
		if limit, ok := state.ArtifactLineBudget(filepath.Base(target.rel)); ok {
			budget = fmt.Sprintf("/%d", limit)
		}
		fmt.Fprintf(stdout, "apply-bundle: %s %s %d hunk(s) sha256 %s -> %s lines %d%s\n",
			target.rel, target.op, len(target.hunks), shortHash(target.before, target.existed), shortHash(target.after, true), lines, budget)
	}
	if dryRun {
		fmt.Fprintf(stdout, "apply-bundle: dry-run clean: %d file(s), %d hunk(s); nothing written\n", len(targets), hunks)
		return 0
	}
	if err := writeBundleTargets(ws, targets); err != nil {
		fmt.Fprintf(stderr, "apply-bundle: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "apply-bundle: applied %d file(s), %d hunk(s)\n", len(targets), hunks)
	return 0
}

// collectBundleTargets merges every bundle's candidate files by workspace path.
// One path takes either one create/replace or any number of patch entries.
func collectBundleTargets(slug string, bundles []candidateBundle) ([]*bundleTarget, map[string]bool, []string) {
	var problems []string
	byRel := map[string]*bundleTarget{}
	retired := map[string]bool{}
	prefix := ".devrites/work/" + slug + "/"
	for bi, bundle := range bundles {
		for _, id := range bundle.RetiredIDs {
			retired[strings.TrimSpace(id)] = true
		}
		pins := map[string]string{}
		for path, hash := range bundle.BaseHashes {
			rel := filepath.ToSlash(filepath.Clean(strings.TrimPrefix(filepath.ToSlash(path), prefix)))
			pin := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(hash), "sha256:"))
			if previous, ok := pins[rel]; ok && previous != pin {
				problems = append(problems, fmt.Sprintf("bundle %d %s: conflicting base hashes for the same file", bi+1, path))
			}
			pins[rel] = pin
		}
		for _, file := range bundle.CandidateFiles {
			rel := strings.TrimPrefix(filepath.ToSlash(file.Path), prefix)
			where := fmt.Sprintf("bundle %d %s", bi+1, file.Path)
			if !filepath.IsLocal(filepath.FromSlash(rel)) || !strings.HasSuffix(rel, ".md") {
				problems = append(problems, where+": path must be a Markdown file inside the "+slug+" workspace")
				continue
			}
			rel = filepath.ToSlash(filepath.Clean(rel))
			op := strings.ToLower(strings.TrimSpace(file.Operation))
			switch op {
			case "create", "replace":
				if file.Content == nil {
					problems = append(problems, where+": "+op+" needs content")
					continue
				}
			case "patch":
				if len(file.Hunks) == 0 {
					problems = append(problems, where+": patch needs hunks")
					continue
				}
			default:
				problems = append(problems, fmt.Sprintf("%s: unknown operation %q", where, file.Operation))
				continue
			}
			target, seen := byRel[rel]
			if !seen {
				target = &bundleTarget{rel: rel, op: op}
				byRel[rel] = target
			} else if op != "patch" || target.op != "patch" {
				problems = append(problems, where+": conflicts with another candidate for the same file (only patch hunks combine)")
				continue
			}
			if op != "create" {
				pin, ok := pins[rel]
				if !ok {
					problems = append(problems, where+": base_hashes has no pin for this file")
					continue
				}
				target.pins = append(target.pins, pin)
			}
			if file.Content != nil {
				target.content = *file.Content
			}
			target.hunks = append(target.hunks, file.Hunks...)
		}
	}
	targets := make([]*bundleTarget, 0, len(byRel))
	for _, target := range byRel {
		targets = append(targets, target)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].rel < targets[j].rel })
	return targets, retired, problems
}

// materializeTargets reads each file's current bytes, verifies its pins, and
// computes the after bytes.
func materializeTargets(ws *os.Root, targets []*bundleTarget) []string {
	var problems []string
	for _, target := range targets {
		before, existed, err := readBundleFile(ws, target.rel)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", target.rel, err))
			continue
		}
		target.before, target.existed = before, existed
		if target.op == "create" {
			if existed {
				problems = append(problems, target.rel+": create target already exists")
				continue
			}
			target.after = []byte(target.content)
			continue
		}
		if !existed {
			problems = append(problems, target.rel+": "+target.op+" target does not exist")
			continue
		}
		current := sha256Hex(before)
		for _, pin := range target.pins {
			if pin != current {
				problems = append(problems, fmt.Sprintf("%s: stale base: pinned %s, current %s", target.rel, abbrev(pin), abbrev(current)))
				break
			}
		}
		if target.op == "replace" {
			target.after = []byte(target.content)
			continue
		}
		after, hunkProblems := applyHunks(string(before), target.hunks)
		for _, problem := range hunkProblems {
			problems = append(problems, target.rel+": "+problem)
		}
		target.after = []byte(after)
	}
	return problems
}

// applyHunks replaces each hunk's old_text, which must occur exactly once in
// base and must not overlap another hunk's span.
func applyHunks(base string, hunks []candidateHunk) (string, []string) {
	type span struct {
		start, end int
		hunk       candidateHunk
	}
	var problems []string
	spans := make([]span, 0, len(hunks))
	for i, hunk := range hunks {
		name := hunk.ID
		if name == "" {
			name = fmt.Sprintf("#%d", i+1)
		}
		hunk.ID = name
		if hunk.OldText == "" {
			problems = append(problems, "hunk "+name+": empty old_text")
			continue
		}
		start := strings.Index(base, hunk.OldText)
		if start < 0 {
			problems = append(problems, "hunk "+name+": old_text not found (stale anchor)")
			continue
		}
		// Search from start+1 so an overlapping second occurrence also counts.
		if strings.Contains(base[start+1:], hunk.OldText) {
			problems = append(problems, "hunk "+name+": old_text occurs more than once (non-unique anchor)")
			continue
		}
		spans = append(spans, span{start, start + len(hunk.OldText), hunk})
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	for i := 1; i < len(spans); i++ {
		if spans[i].start < spans[i-1].end {
			problems = append(problems, fmt.Sprintf("hunks %s and %s overlap", spans[i-1].hunk.ID, spans[i].hunk.ID))
		}
	}
	if len(problems) > 0 {
		return base, problems
	}
	var out strings.Builder
	last := 0
	for _, s := range spans {
		out.WriteString(base[last:s.start])
		out.WriteString(s.hunk.NewText)
		last = s.end
	}
	out.WriteString(base[last:])
	return out.String(), nil
}

// checkBundleResult rejects an outcome the root would otherwise discover only
// at Vet: a tracked ID that vanished from every touched file without being
// listed in retired_ids, a new tasks.md graph problem, a new test-plan.md
// preflight row that cannot run as written, or a budgeted artifact
// that grew into material overshoot without a covering
// `Budget override: <=N lines; <reason>` line.
func checkBundleResult(targets []*bundleTarget, retired map[string]bool) []string {
	var problems []string
	beforeIDs, afterIDs := map[string]bool{}, map[string]bool{}
	for _, target := range targets {
		for _, id := range trackedIDRE.FindAllString(string(target.before), -1) {
			beforeIDs[id] = true
		}
		for _, id := range trackedIDRE.FindAllString(string(target.after), -1) {
			afterIDs[id] = true
		}
	}
	var lost []string
	for id := range beforeIDs {
		if !afterIDs[id] && !retired[id] {
			lost = append(lost, id)
		}
	}
	if len(lost) > 0 {
		slices.Sort(lost)
		problems = append(problems, "IDs vanished from every touched file (list them in retired_ids if intended): "+strings.Join(lost, ", "))
	}

	for _, target := range targets {
		name := filepath.Base(target.rel)
		if name == "tasks.md" && filepath.Dir(target.rel) == "." {
			old := map[string]bool{}
			if target.existed {
				for _, p := range state.ParseTaskGraph(target.before).Problems {
					old[p] = true
				}
			}
			for _, p := range state.ParseTaskGraph(target.after).Problems {
				if !old[p] {
					problems = append(problems, "tasks.md: "+p)
				}
			}
		}
		if name == "test-plan.md" && filepath.Dir(target.rel) == "." {
			old := map[string]bool{}
			for _, p := range acceptance.PreflightShapeProblems(target.before) {
				old[p] = true
			}
			for _, p := range acceptance.PreflightShapeProblems(target.after) {
				if !old[p] {
					problems = append(problems, "test-plan.md: "+p)
				}
			}
		}
		if filepath.Dir(target.rel) == "." && currentStatePlanning[name] &&
			len(datedHeadingRE.FindAllIndex(target.after, -1)) > len(datedHeadingRE.FindAllIndex(target.before, -1)) {
			problems = append(problems, target.rel+": adds a dated delta section; rewrite the owning clause in place (rationale to decisions.md, closed narrative to history/)")
		}
		if filepath.Dir(target.rel) != "." || !state.ArtifactBudgetGateApplies(name) {
			continue
		}
		after, _ := state.ArtifactBudget(name, target.after)
		before, _ := state.ArtifactBudget(name, target.before)
		grew := !target.existed || after.Lines > before.Lines || after.Bytes > before.Bytes
		if after.Material && !after.Override && grew {
			problems = append(problems, fmt.Sprintf("%s: %d lines / %d bytes is a material overshoot of its %d-line budget; split or relocate instead of growing it",
				target.rel, after.Lines, after.Bytes, after.LineBudget))
		}
	}
	return problems
}

// writeBundleTargets writes every target atomically and, if one write fails,
// restores the files already written so the workspace never holds half a
// bundle.
func writeBundleTargets(ws *os.Root, targets []*bundleTarget) error {
	for i, target := range targets {
		if err := fsutil.WriteFileAtomicIn(ws, filepath.FromSlash(target.rel), target.after, 0o644); err != nil {
			for _, done := range targets[:i] {
				if done.existed {
					_ = fsutil.WriteFileAtomicIn(ws, filepath.FromSlash(done.rel), done.before, 0o644)
				} else {
					_ = ws.Remove(filepath.FromSlash(done.rel))
				}
			}
			return fmt.Errorf("write %s: %w (earlier files restored)", target.rel, err)
		}
	}
	return nil
}

func readBundleFile(ws *os.Root, rel string) ([]byte, bool, error) {
	info, err := ws.Lstat(filepath.FromSlash(rel))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, errors.New("not a regular file")
	}
	if info.Size() > state.MaxArtifactBytes {
		return nil, false, fmt.Errorf("exceeds the %d-byte artifact cap", state.MaxArtifactBytes)
	}
	raw, err := ws.ReadFile(filepath.FromSlash(rel))
	return raw, true, err
}

func sha256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func shortHash(raw []byte, exists bool) string {
	if !exists {
		return "(new)"
	}
	return abbrev(sha256Hex(raw))
}

func abbrev(hash string) string {
	if len(hash) > 12 {
		return hash[:12]
	}
	return hash
}
