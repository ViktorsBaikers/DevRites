package lib

import (
	"fmt"
	"path"
	"sort"
	"strings"

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

// scheduleRounds replays /rite-build batch selection at the maximum cap: each
// round runs the plan-order path-disjoint subset of dependency-ready slices.
// It returns 0 when a slice has no write paths or parallel select rejects them.
func scheduleRounds(graph TaskGraphResult, paths map[string][]string, root string) int {
	done := make(map[string]bool, len(graph.Slices))
	rounds := 0
	for len(done) < len(graph.Slices) {
		var ready []parallel.SlicePaths
		for _, s := range graph.Slices {
			if done[s.ID] || !allDone(s.Dependencies, done) {
				continue
			}
			if len(paths[s.ID]) == 0 {
				return 0
			}
			ready = append(ready, parallel.SlicePaths{ID: s.ID, Paths: paths[s.ID]})
		}
		if len(ready) == 0 {
			return 0
		}
		picked, err := parallel.SelectGreedy(parallel.MaxParallelSlices, ready, root)
		if err != nil || len(picked) == 0 {
			return 0
		}
		for _, p := range picked {
			done[p.ID] = true
		}
		rounds++
	}
	return rounds
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
	writers := map[string][]string{}
	shown := map[string]string{} // first spelling seen, for display
	for _, s := range graph.Slices {
		if serial[s.ID] {
			continue
		}
		seen := map[string]bool{}
		for _, p := range paths[s.ID] {
			key := strings.ToLower(path.Clean(strings.ReplaceAll(p, `\`, "/")))
			if !seen[key] {
				seen[key] = true
				writers[key] = append(writers[key], s.ID)
				if _, ok := shown[key]; !ok {
					shown[key] = p
				}
			}
		}
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
