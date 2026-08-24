package lib

import (
	"encoding/json"
	"io"

	"github.com/devrites/devrites/internal/state"
)

// ObserveSummary is a sanitized, machine-readable workspace snapshot.
type ObserveSummary struct {
	Slug              string   `json:"slug"`
	Phase             string   `json:"phase,omitempty"`
	Status            string   `json:"status,omitempty"`
	NextAction        string   `json:"next_action,omitempty"`
	MissingSections   []string `json:"missing_sections,omitempty"`
	MissingFiles      []string `json:"missing_files,omitempty"`
	PrinciplesPresent bool     `json:"principles_present"`
	TaskGraph         *struct {
		SliceCount int      `json:"slice_count"`
		Cycle      []string `json:"cycle,omitempty"`
		Unknown    []string `json:"unknown_dependencies,omitempty"`
		OK         bool     `json:"ok"`
	} `json:"task_graph,omitempty"`
}

// ObserveSummaryFor builds a summary for one feature slug.
func ObserveSummaryFor(root, slug string) (ObserveSummary, error) {
	report, err := state.Status(root, slug)
	if err != nil {
		return ObserveSummary{}, err
	}
	summary := ObserveSummary{
		Slug:              report.Slug,
		Phase:             string(report.Phase),
		Status:            report.Status,
		NextAction:        report.NextAction,
		PrinciplesPresent: report.PrinciplesPresent,
		MissingFiles:      append([]string(nil), report.MissingFiles...),
		MissingSections:   missingSectionNames(report.Missing),
	}

	if graph, graphErr := CheckTaskGraph(root, slug); graphErr == nil && len(graph.Slices) > 0 {
		summary.TaskGraph = &struct {
			SliceCount int      `json:"slice_count"`
			Cycle      []string `json:"cycle,omitempty"`
			Unknown    []string `json:"unknown_dependencies,omitempty"`
			OK         bool     `json:"ok"`
		}{
			SliceCount: len(graph.Slices),
			Cycle:      graph.Cycle,
			Unknown:    graph.Unknown,
			OK:         len(graph.Problems) == 0,
		}
	}
	return summary, nil
}

func missingSectionNames(sections []state.Section) []string {
	if len(sections) == 0 {
		return nil
	}
	out := make([]string, len(sections))
	for i, section := range sections {
		out[i] = string(section)
	}
	return out
}

// WriteObserveSummaryJSON prints one JSON object to stdout.
func WriteObserveSummaryJSON(root, slug string, stdout io.Writer) error {
	summary, err := ObserveSummaryFor(root, slug)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(summary)
}
