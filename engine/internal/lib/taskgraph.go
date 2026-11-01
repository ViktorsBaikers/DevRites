package lib

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/parallel"
	"github.com/devrites/devrites/internal/state"
)

// TaskSlice is one SLICE-### block parsed from tasks.md.
type TaskSlice = state.TaskSlice

// TaskGraphResult is the deterministic outcome of parsing a tasks.md slice graph.
type TaskGraphResult = state.TaskGraphResult

// ParseTaskGraph reads tasks.md content and validates the slice dependency DAG.
func ParseTaskGraph(tasksMarkdown []byte) TaskGraphResult {
	return state.ParseTaskGraph(tasksMarkdown)
}

// CheckTaskGraph validates tasks.md for a feature workspace slug.
func CheckTaskGraph(root, slug string) (TaskGraphResult, error) {
	raw, err := readWorkspaceArtifact(root, slug, "tasks.md")
	if err != nil {
		return TaskGraphResult{}, err
	}
	return ParseTaskGraph(raw), nil
}

// hubAdvisoryMin is how many slices must write one path before it is flagged:
// slices sharing a write path never share a parallel build round.
const hubAdvisoryMin = 3

// hubAdvisoryShown caps printed hub lines; the metric still counts every hub.
const hubAdvisoryShown = 5

// sliceWritePaths returns each slice's write set (Writer allowlist, else Files
// likely touched) and whether it records a `Serial rationale:`.
func sliceWritePaths(tasksMarkdown []byte, graph TaskGraphResult) (map[string][]string, map[string]bool) {
	paths := make(map[string][]string, len(graph.Slices))
	serial := make(map[string]bool)
	for _, s := range graph.Slices {
		block, _ := state.ExtractTaskSlice(tasksMarkdown, s.ID)
		fields := sliceFields(block)
		allow, _ := fieldValue(fields, "writer allowlist")
		paths[s.ID] = pathTokens(allow)
		if len(paths[s.ID]) == 0 {
			touched, _ := fieldValue(fields, "files likely touched")
			paths[s.ID] = pathTokens(touched)
		}
		if v, _ := fieldValue(fields, "serial rationale"); v != "" {
			serial[s.ID] = true
		}
	}
	return paths, serial
}

// schedulePaths is the write set the task-graph schedule and hub check use.
// It adds the manifest withManifests reserves so printed rounds match
// `parallel select --from-plan`, not the allowlist alone. devritesRoot is the
// .devrites directory.
func schedulePaths(tasksMarkdown []byte, graph TaskGraphResult, devritesRoot string) (map[string][]string, map[string]bool) {
	paths, serial := sliceWritePaths(tasksMarkdown, graph)
	project := filepath.Dir(devritesRoot)
	for _, s := range graph.Slices {
		block, _ := state.ExtractTaskSlice(tasksMarkdown, s.ID)
		external, _ := fieldValue(sliceFields(block), "external dependencies")
		paths[s.ID] = withManifests(project, paths[s.ID], external)
	}
	return paths, serial
}

// pathTokens keeps the list entries that can be paths; prose and Markdown links
// (anything with whitespace) are not write paths.
func pathTokens(raw string) []string {
	var out []string
	for _, p := range parsePathList(raw) {
		if !strings.ContainsAny(p, " \t") {
			out = append(out, p)
		}
	}
	return out
}

// schedulePlan replays /rite-build batch selection at the maximum cap: each
// round runs the plan-order path-disjoint subset of dependency-ready slices.
// It returns the picked slice ids per round, the number of dependency-ready
// slices each round left idle (parallelism the cap could not use), and the
// largest dependency-ready set any round saw. That last value separates a
// dependency chain (never more than one slice ready) from write contention
// (many ready, one picked). It returns nil when a slice has no write paths or
// parallel select rejects.
func schedulePlan(graph TaskGraphResult, paths map[string][]string, root string) ([][]string, int, int) {
	done := make(map[string]bool, len(graph.Slices))
	var rounds [][]string
	idle, maxReady := 0, 0
	for len(done) < len(graph.Slices) {
		var ready []parallel.SlicePaths
		for _, s := range graph.Slices {
			if done[s.ID] || !allDone(s.Dependencies, done) {
				continue
			}
			if len(paths[s.ID]) == 0 {
				return nil, 0, 0
			}
			ready = append(ready, parallel.SlicePaths{ID: s.ID, Paths: paths[s.ID]})
		}
		if len(ready) == 0 {
			return nil, 0, 0
		}
		picked, err := parallel.SelectGreedy(parallel.MaxParallelSlices, ready, root)
		if err != nil || len(picked) == 0 {
			return nil, 0, 0
		}
		ids := make([]string, 0, len(picked))
		for _, p := range picked {
			done[p.ID] = true
			ids = append(ids, p.ID)
		}
		idle += len(ready) - len(picked)
		maxReady = max(maxReady, len(ready))
		rounds = append(rounds, ids)
	}
	return rounds, idle, maxReady
}

// scheduleRounds returns the round count alone; see schedulePlan for the shape.
func scheduleRounds(graph TaskGraphResult, paths map[string][]string, root string) int {
	rounds, _, _ := schedulePlan(graph, paths, root)
	return len(rounds)
}

// scheduleWidth is the busiest round: the plan's achieved DAG width.
func scheduleWidth(rounds [][]string) int {
	width := 0
	for _, picked := range rounds {
		width = max(width, len(picked))
	}
	return width
}

// PlanReady is the ready set for `parallel select --from-plan`; repoRoot is
// the project root.
func PlanReady(repoRoot, slug string) ([]parallel.SlicePaths, error) {
	return planReady(filepath.Join(repoRoot, devritespaths.DevritesRootName), slug)
}

// PrecheckSkips maps each gate id `gates precheck --slice <slice>` skips to the
// slices that own it. Gate ids reuse AC-### ids, so a gate's owners are the
// tasks.md slices whose block names its id; it is skipped only when no owner is
// slice or built. A gate no slice names still runs. root is the .devrites
// directory.
func PrecheckSkips(root, slug, slice string, gateIDs []string) (map[string]string, error) {
	raw, err := readWorkspaceArtifact(root, slug, "tasks.md")
	if err != nil {
		return nil, err
	}
	if _, ok := state.ExtractTaskSlice(raw, slice); !ok {
		return nil, fmt.Errorf("%s not found in tasks.md", slice)
	}
	built, err := builtSlices(root, slug)
	if err != nil {
		return nil, err
	}
	live := map[string]bool{}
	owners := map[string][]string{}
	for _, s := range ParseTaskGraph(raw).Slices {
		block, _ := state.ExtractTaskSlice(raw, s.ID)
		for _, ac := range sliceACRe.FindAllString(block, -1) {
			switch {
			case s.ID == slice || built[sliceNumber(s.ID)]:
				live[ac] = true
			case !slices.Contains(owners[ac], s.ID):
				owners[ac] = append(owners[ac], s.ID)
			}
		}
	}
	skips := map[string]string{}
	for _, id := range gateIDs {
		if !live[id] && len(owners[id]) > 0 {
			skips[id] = strings.Join(owners[id], ", ")
		}
	}
	return skips, nil
}

// planReady lists, in plan order, the tasks.md slices that are not built whose
// dependencies are all built, each with its sliceWritePaths plus its dependency
// manifest and lockfile when it declares external dependencies. root is the
// .devrites directory.
func planReady(root, slug string) ([]parallel.SlicePaths, error) {
	ready, _, err := workspacePlanReady(root, slug, false, func(name string) ([]byte, error) {
		return readWorkspaceArtifact(root, slug, name)
	})
	return ready, err
}

// builtWordRe marks a checked state.md item as a built checkpoint.
var builtWordRe = regexp.MustCompile(`(?i)\bbuilt\b`)

// builtSlices returns the slice numbers that are built, outside fenced examples.
// A state.md checkbox (`- [x] SLICE-004 built — <sha>`, or `- [x] Slice 4:
// <name> — built`) counts. So does an evidence.md heading that says that slice
// was checkpointed and whose section has a `Checkpoint: <sha>` line — a missed
// checkbox must not keep dependents out of the ready set.
func builtSlices(root, slug string) (map[string]bool, error) {
	return builtSlicesFrom(func(name string) ([]byte, error) {
		return readWorkspaceArtifact(root, slug, name)
	})
}

func builtSlicesFrom(read func(string) ([]byte, error)) (map[string]bool, error) {
	raw, err := read("state.md")
	if err != nil {
		return nil, err
	}
	masked, err := markdowntext.Structural(raw)
	if err != nil {
		return nil, fmt.Errorf("state.md: %w", err)
	}
	built := map[string]bool{}
	for _, item := range regressBuiltRe.FindAllString(string(masked), -1) {
		if !builtWordRe.MatchString(item) {
			continue
		}
		for _, m := range regressSliceIDRe.FindAllStringSubmatch(item, -1) {
			built[strings.TrimLeft(m[1], "0")] = true
		}
	}
	extra, err := evidenceBuiltFrom(read)
	if err != nil {
		return nil, err
	}
	for id := range extra {
		built[id] = true
	}
	return built, nil
}

var (
	checkpointedHeadingRe = regexp.MustCompile(`(?i)\bcheckpointed\b`)
	notCheckpointedRe     = regexp.MustCompile(`(?i)\bnot checkpointed\b|\bcheckpoint (?:refused|reverted|failed)\b`)
	checkpointSHARe       = regexp.MustCompile(`(?m)^[ \t]*Checkpoint: [0-9a-fA-F]{7,40}\b`)
)

// evidenceBuiltFrom reads evidence.md checkpoint sections. A missing file is none.
func evidenceBuiltFrom(read func(string) ([]byte, error)) (map[string]bool, error) {
	raw, err := read("evidence.md")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	masked, err := markdowntext.Structural(raw)
	if err != nil {
		return nil, fmt.Errorf("evidence.md: %w", err)
	}
	text := string(masked)
	if strings.HasPrefix(text, "## ") {
		text = "\n" + text
	}
	built := map[string]bool{}
	for part := range strings.SplitSeq(text, "\n## ") {
		heading, body, _ := strings.Cut(part, "\n")
		var ids []string
		for _, id := range sliceTokenRe.FindAllString(heading, -1) {
			ids = append(ids, sliceNumber(id))
		}
		if len(ids) == 0 {
			continue
		}
		if notCheckpointedRe.MatchString(heading) {
			for _, id := range ids {
				delete(built, id)
			}
			continue
		}
		if checkpointedHeadingRe.MatchString(heading) && checkpointSHARe.MatchString(body) {
			for _, id := range ids {
				built[id] = true
			}
		}
	}
	return built, nil
}

// A backtick name, module path, npm scope, or x.y.z pin. go.mod/go.sum is not a module path.
// A toolchain version (Go 1.22.0) is not a package add.
var (
	backtickPkgRe      = regexp.MustCompile("`([^`]+)`")
	modulePathRe       = regexp.MustCompile(`(?i)(?:[a-z0-9-]+\.)+[a-z]{2,}/[a-z0-9._@~+-]+`)
	npmScopeRe         = regexp.MustCompile(`(?i)@[a-z0-9._-]+/[a-z0-9._-]+`)
	versionPinRe       = regexp.MustCompile(`(?i)\bv?\d+\.\d+\.\d+\b`)
	toolchainVersionRe = regexp.MustCompile(`(?i)\b(?:go|golang|node|bun|npm|pnpm|yarn|rustc|python|ruby|java)\s+v?\d+\.\d+\.\d+\b`)
	manifestHostRe     = regexp.MustCompile(`(?i)\.(?:mod|sum|json|toml|yaml|yml|lock|lockb)$`)
)

// externalClaimsManifest reports whether External dependencies names a package
// add, so parallel select reserves the nearest manifest. A none prefix does
// not. Prose with no module path, npm scope, `package`, or non-toolchain
// version pin does not either. shortcut: name the package when the slice
// edits a manifest; "Go 1.22.0" or "standard library" is not a package add.
func externalClaimsManifest(external string) bool {
	ext := strings.ToLower(strings.Trim(strings.TrimSpace(external), "`"))
	if ext == "" || ext == "-" || ext == "n/a" || strings.HasPrefix(ext, "none") {
		return false
	}
	return namesPackage(strings.TrimSpace(external))
}

// namesPackage reports a module path, version pin, npm scope, or `package`
// name. Manifest filenames mentioned only to deny an edit do not count.
func namesPackage(raw string) bool {
	for _, m := range backtickPkgRe.FindAllStringSubmatch(raw, -1) {
		base := path.Base(strings.TrimSpace(m[1]))
		if base != "" && base != "." && !manifestHostRe.MatchString(base) {
			return true
		}
	}
	if npmScopeRe.MatchString(raw) || versionPinRe.MatchString(toolchainVersionRe.ReplaceAllString(raw, " ")) {
		return true
	}
	for _, loc := range modulePathRe.FindAllStringIndex(raw, -1) {
		token := raw[loc[0]:loc[1]]
		host := token[:strings.IndexByte(token, '/')]
		if !manifestHostRe.MatchString(host) {
			return true
		}
	}
	return false
}

// withManifests adds, when external names a package add, the manifest and
// lockfiles of the nearest directory above each write path that holds one,
// so two dependency-adding slices never share a round. A lone lockfile is
// always claimed (adding a dependency writes it); of several, only those
// present are.
func withManifests(project string, paths []string, external string) []string {
	if !externalClaimsManifest(external) {
		return paths
	}
	isFile := func(rel string) bool {
		info, err := os.Stat(filepath.Join(project, filepath.FromSlash(rel)))
		return err == nil && info.Mode().IsRegular()
	}
	out := append([]string{}, paths...)
	seen := map[string]bool{}
	for _, p := range paths {
		seen[strings.ToLower(path.Clean(p))] = true
	}
	add := func(p string) {
		if key := strings.ToLower(p); !seen[key] {
			seen[key] = true
			out = append(out, p)
		}
	}
	for _, p := range paths {
		for dir := path.Dir(path.Clean(p)); ; dir = path.Dir(dir) {
			found := false
			for _, m := range gate.DependencyManifests {
				if !isFile(path.Join(dir, m.Manifest)) {
					continue
				}
				found = true
				add(path.Join(dir, m.Manifest))
				for _, lock := range m.Locks {
					if len(m.Locks) == 1 || isFile(path.Join(dir, lock)) {
						add(path.Join(dir, lock))
					}
				}
			}
			if found || dir == "." || dir == "/" {
				break
			}
		}
	}
	return out
}

// planBatch returns the eligible slice ids and the new picks of `parallel
// select --from-plan` at capN, seeded from slug's running lease. root is the
// .devrites directory.
func planBatch(root, slug string, capN int) (eligible, batch []string, err error) {
	ready, err := planReady(root, slug)
	if err != nil {
		return nil, nil, err
	}
	elig, picks, _, err := parallel.SelectForSlug(filepath.Dir(root), slug, capN, ready)
	if err != nil {
		return nil, nil, err
	}
	ids := func(in []parallel.SlicePaths) []string {
		var out []string
		for _, s := range in {
			if s.Workspace == "" || s.Workspace == slug {
				out = append(out, s.ID)
			}
		}
		return out
	}
	return ids(elig), ids(picks), nil
}

// planSelectCommand is the exact command that reproduces planBatch from any
// directory; project is the directory holding .devrites.
func planSelectCommand(project, slug string, capN int) string {
	return fmt.Sprintf("devrites-engine parallel select --root %s --slug %s --cap %d --from-plan", project, slug, capN)
}

// afkMaxParallel returns the AFK sentinel's max_parallel, or 0 when the
// sentinel, the line, or a readable number is absent.
func afkMaxParallel(root string) int {
	data, err := os.ReadFile(filepath.Join(root, "AFK")) // #nosec G304 -- .devrites/AFK sentinel
	if err != nil {
		return 0
	}
	m := afkParallelRe.FindStringSubmatch(string(data))
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// parallelPlanVisible reports that the root must see every slice. An
// unattended cap above 1, or a running parallel lease, is a batch: scoping
// the bundle to active_slice hides the siblings that batch is supposed to start.
func parallelPlanVisible(devritesRoot, slug string) bool {
	if afkMaxParallel(devritesRoot) >= 2 {
		return true
	}
	if slug == "" {
		return false
	}
	path, err := parallel.LeasePath(filepath.Dir(devritesRoot), slug)
	if err != nil {
		return false
	}
	lease, err := parallel.ReadLease(path)
	return err == nil && lease.Status == parallel.StatusRunning
}

func allDone(ids []string, done map[string]bool) bool {
	for _, id := range ids {
		if !done[id] {
			return false
		}
	}
	return true
}

// hubAdvisories names each path written by hubAdvisoryMin+ slices that carry
// no `Serial rationale:`, most-shared first.
func hubAdvisories(graph TaskGraphResult, paths map[string][]string, serial map[string]bool) []string {
	claimed := map[string][]string{} // key -> slices claiming exactly it
	shown := map[string]string{}     // first spelling seen, for display
	for _, s := range graph.Slices {
		if serial[s.ID] {
			continue
		}
		seen := map[string]bool{}
		for _, p := range paths[s.ID] {
			key := strings.ToLower(path.Clean(strings.ReplaceAll(p, `\`, "/")))
			if !seen[key] {
				seen[key] = true
				claimed[key] = append(claimed[key], s.ID)
				if _, ok := shown[key]; !ok {
					shown[key] = p
				}
			}
		}
	}
	// A directory claim contains every path beneath it, so writers of a path
	// include slices claiming its ancestors and its descendants; counting exact
	// spellings alone would report one writer for a hub every slice touches.
	writers := map[string][]string{}
	for key := range claimed {
		set := map[string]bool{}
		for _, id := range claimed[key] {
			set[id] = true
		}
		for other, ids := range claimed {
			if other == key {
				continue
			}
			if strings.HasPrefix(other, key+"/") || strings.HasPrefix(key, other+"/") {
				for _, id := range ids {
					set[id] = true
				}
			}
		}
		ids := make([]string, 0, len(set))
		for id := range set {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		writers[key] = ids
	}
	hubs := make([]string, 0, len(writers))
	for p, ids := range writers {
		if len(ids) >= hubAdvisoryMin {
			hubs = append(hubs, p)
		}
	}
	sort.Slice(hubs, func(i, j int) bool {
		if len(writers[hubs[i]]) != len(writers[hubs[j]]) {
			return len(writers[hubs[i]]) > len(writers[hubs[j]])
		}
		return hubs[i] < hubs[j]
	})
	lines := make([]string, len(hubs))
	for i, p := range hubs {
		lines[i] = fmt.Sprintf("ADVISORY hub: %s is written by %d slices (%s), so they build one at a time; move the shared edit into one seam slice they depend on, or record `Serial rationale:`", shown[p], len(writers[p]), strings.Join(writers[p], ", "))
	}
	return lines
}

// chainAdvisories names a plan whose schedule never overlaps because only one
// slice was ever dependency-ready per round, so no parallel cap can help. A plan
// serialized by write contention instead is a hub problem, not a chain, and a
// plan whose slices all record `Serial rationale:` is deliberate; both stay quiet.
func chainAdvisories(graph TaskGraphResult, serial map[string]bool, rounds, maxReady int) []string {
	if len(graph.Slices) < 2 || rounds != len(graph.Slices) || maxReady != 1 {
		return nil
	}
	for _, s := range graph.Slices {
		if !serial[s.ID] {
			return []string{fmt.Sprintf("ADVISORY chain: %d slices run in %d build rounds, so no two slices overlap and a parallel cap cannot help; cut a slice that stands alone, or record `Serial rationale:`", len(graph.Slices), rounds)}
		}
	}
	return nil
}

// edgeAdvisories names each dependency edge that does not say what crosses it:
// no `(via <interface>)`, or a via the dependency's Consumes / Produces lacks.
// An edge the dependent reads nothing across only serializes work.
func edgeAdvisories(graph TaskGraphResult) []string {
	produces := make(map[string]string, len(graph.Slices))
	for _, s := range graph.Slices {
		produces[s.ID] = s.ConsumesProduces
	}
	var lines []string
	for _, s := range graph.Slices {
		for _, dep := range s.Dependencies {
			via := s.Via[dep]
			switch {
			case via == "":
				lines = append(lines, fmt.Sprintf(`ADVISORY edge: %s -> %s names no crossing interface (add "(via <interface>)" or drop the edge)`, s.ID, dep))
			case produces[dep] == "":
				lines = append(lines, fmt.Sprintf("ADVISORY edge: %s -> %s via %q but %s declares no Consumes / Produces", s.ID, dep, via, dep))
			case !strings.Contains(strings.ToLower(produces[dep]), strings.ToLower(via)):
				lines = append(lines, fmt.Sprintf("ADVISORY edge: %s -> %s via %q is not in %s Consumes / Produces", s.ID, dep, via, dep))
			}
		}
	}
	return lines
}
