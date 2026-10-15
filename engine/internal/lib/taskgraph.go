package lib

import (
	"fmt"
	"strings"

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
