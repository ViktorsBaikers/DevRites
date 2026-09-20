package lib

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
)

// ObserveSlice is one slice's identity and ordering edges, enough for a root to pick
// the next buildable slice without loading tasks.md.
type ObserveSlice struct {
	ID        string   `json:"id"`
	DependsOn []string `json:"depends_on"`
}

// ObserveTaskGraph is the slice-graph subset of ObserveSummary.
type ObserveTaskGraph struct {
	SliceCount int            `json:"slice_count"`
	Slices     []ObserveSlice `json:"slices,omitempty"`
	Cycle      []string       `json:"cycle,omitempty"`
	Unknown    []string       `json:"unknown_dependencies,omitempty"`
	Problems   []string       `json:"problems,omitempty"`
	OK         bool           `json:"ok"`
}

// ObserveSummary is a sanitized, machine-readable workspace snapshot.
type ObserveSequenceCursor struct {
	Parent              string `json:"parent"`
	Position            string `json:"position"`
	WorkspacesRemaining string `json:"workspaces_remaining"`
	Role                string `json:"role,omitempty"`
}

type ObserveSummary struct {
	Slug              string                 `json:"slug"`
	Phase             string                 `json:"phase,omitempty"`
	Status            string                 `json:"status,omitempty"`
	NextAction        string                 `json:"next_action,omitempty"`
	MissingSections   []string               `json:"missing_sections,omitempty"`
	MissingFiles      []string               `json:"missing_files,omitempty"`
	PrinciplesPresent bool                   `json:"principles_present"`
	TaskGraph         *ObserveTaskGraph      `json:"task_graph,omitempty"`
	ArtifactBudgets   []ArtifactBudget       `json:"artifact_budgets,omitempty"`
	BulkFiles         []BulkFile             `json:"bulk_files,omitempty"`
	Sequence          *ObserveSequenceCursor `json:"sequence,omitempty"`
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

	if graph, graphErr := CheckTaskGraph(root, slug); graphErr == nil && (len(graph.Slices) > 0 || len(graph.Problems) > 0) {
		slices := make([]ObserveSlice, 0, len(graph.Slices))
		for _, slice := range graph.Slices {
			deps := make([]string, 0, len(slice.Dependencies))
			deps = append(deps, slice.Dependencies...)
			slices = append(slices, ObserveSlice{ID: slice.ID, DependsOn: deps})
		}
		summary.TaskGraph = &ObserveTaskGraph{
			SliceCount: len(graph.Slices),
			Slices:     slices,
			Cycle:      append([]string(nil), graph.Cycle...),
			Unknown:    append([]string(nil), graph.Unknown...),
			Problems:   append([]string(nil), graph.Problems...),
			OK:         len(graph.Problems) == 0,
		}
	}
	if report.SequenceParent != "" || report.SequencePosition != "" || report.SequenceWorkspacesRemaining != "" || report.SequenceRole != "" {
		summary.Sequence = &ObserveSequenceCursor{
			Parent:              report.SequenceParent,
			Position:            report.SequencePosition,
			WorkspacesRemaining: report.SequenceWorkspacesRemaining,
			Role:                report.SequenceRole,
		}
	}
	if workspace, wsErr := devritespaths.ExistingFeatureDirChecked(root, slug); wsErr == nil {
		if budgets, bulk, budgetErr := observeArtifactBudgets(workspace); budgetErr == nil {
			summary.ArtifactBudgets = budgets
			summary.BulkFiles = bulk
		}
	}
	return summary, nil
}

// RunObserveSlice prints one SLICE-### section of tasks.md for a slug.
func RunObserveSlice(root, slug, id string, stdout, stderr io.Writer) int {
	if !state.ValidSliceID(id) {
		fmt.Fprintf(stderr, "observe slice: %q is not a SLICE-### id\n", id)
		return 2
	}
	raw, err := readWorkspaceArtifact(root, slug, "tasks.md")
	if err != nil {
		fmt.Fprintf(stderr, "observe slice: %v\n", err)
		return 2
	}
	block, ok := state.ExtractTaskSlice(raw, id)
	if !ok {
		fmt.Fprintf(stdout, "observe slice: BLOCKED: %s not found in tasks.md\n", id)
		return 3
	}
	fmt.Fprint(stdout, block)
	return 0
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
