package lib

import (
	"fmt"
	"io"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/parallel"
	"github.com/devrites/devrites/internal/state"
)

// RunTaskGraphCheck validates tasks.md for one slug. Shared-path findings are
// advisory: they never change the exit code.
func RunTaskGraphCheck(root, slug string, stdout, stderr io.Writer) int {
	return runTaskGraphCheck(root, slug, false, stdout, stderr)
}

// RunTaskGraphCheckStrict is RunTaskGraphCheck, except a plan whose shared-path
// writer count breaches the hub limit is refused instead of only advised. That is
// the plan-shape repair point: split disjointly rather than widen the cap.
func RunTaskGraphCheckStrict(root, slug string, stdout, stderr io.Writer) int {
	return runTaskGraphCheck(root, slug, true, stdout, stderr)
}

func runTaskGraphCheck(root, slug string, strict bool, stdout, stderr io.Writer) int {
	graph, err := CheckTaskGraph(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "task-graph: %v\n", err)
		return 2
	}
	if len(graph.Problems) > 0 {
		for _, problem := range graph.Problems {
			fmt.Fprintf(stdout, "task-graph: BLOCKED: %s\n", problem)
		}
		return 3
	}
	fmt.Fprintf(stdout, "task-graph: ok (%d slices)\n", len(graph.Slices))
	// Size advisory: never changes the exit code; the metric shows how often
	// it would have fired before anyone makes it blocking.
	if raw, err := readWorkspaceArtifact(root, slug, "tasks.md"); err == nil {
		over := 0
		for _, s := range graph.Slices {
			block, _ := state.ExtractTaskSlice(raw, s.ID)
			if n, ok := sliceSizeAdvisory(block); ok {
				fmt.Fprintf(stdout, "task-graph: ADVISORY size: %s has %d Writer allowlist files (>= %d); split it (acceptance-preserving) or record `Size rationale:`\n", s.ID, n, sliceFileAdvisory)
				over++
			}
		}
		if over > 0 {
			recordMetric(root, slug, "define", "size-advisory", "", int64(over))
		}
		paths, serial := sliceWritePaths(raw, graph)
		if plan, idle, maxReady := schedulePlan(graph, paths, root); len(plan) > 0 {
			fmt.Fprintf(stdout, "task-graph: schedule: %d slices in %d build rounds at cap %d, width %d, idle-eligible %d\n", len(graph.Slices), len(plan), parallel.MaxParallelSlices, scheduleWidth(plan), idle)
			if chain := chainAdvisories(graph, serial, len(plan), maxReady); len(chain) > 0 {
				for _, line := range chain {
					fmt.Fprintf(stdout, "task-graph: %s\n", line)
				}
				recordMetric(root, slug, "define", "chain-advisory", "", int64(len(chain)))
			}
		}
		if hubs := hubAdvisories(graph, paths, serial); len(hubs) > 0 {
			for _, line := range hubs[:min(len(hubs), hubAdvisoryShown)] {
				fmt.Fprintf(stdout, "task-graph: %s\n", line)
			}
			if extra := len(hubs) - hubAdvisoryShown; extra > 0 {
				fmt.Fprintf(stdout, "task-graph: ADVISORY hub: %d more shared paths; fix the ones above first\n", extra)
			}
			recordMetric(root, slug, "define", "hub-advisory", "", int64(len(hubs)))
			if strict {
				fmt.Fprintf(stdout, "task-graph: BLOCKED: %d shared path(s) with %d+ writers; split disjointly (acceptance-preserving) or record `Serial rationale:`\n", len(hubs), hubAdvisoryMin)
				return 3
			}
		}
	}
	if edges := edgeAdvisories(graph); len(edges) > 0 {
		for _, line := range edges {
			fmt.Fprintf(stdout, "task-graph: %s\n", line)
		}
		recordMetric(root, slug, "define", "edge-advisory", "", int64(len(edges)))
	}
	return 0
}

// RunSkillTrustCheck scans one Markdown path.
func RunSkillTrustCheck(path string, stdout, stderr io.Writer) int {
	result, err := ScanSkillTrust(path)
	if err != nil {
		fmt.Fprintf(stderr, "skill-trust: %v\n", err)
		return 2
	}
	fmt.Fprint(stdout, FormatSkillTrust(result))
	if SkillTrustBlocks(result.Findings) {
		return 3
	}
	return 0
}

// RunObserveSummary emits JSON for one slug.
func RunObserveSummary(root, slug string, stdout, stderr io.Writer) int {
	if err := WriteObserveSummaryJSON(root, slug, stdout); err != nil {
		fmt.Fprintf(stderr, "observe: %v\n", err)
		return 2
	}
	return 0
}

// ActiveSlug resolves slug from args or ACTIVE pointer.
func ActiveSlug(root string, args []string) (string, int, error) {
	if len(args) == 1 {
		return args[0], 0, nil
	}
	if len(args) != 0 {
		return "", 2, fmt.Errorf("expected one slug argument")
	}
	slug, err := devritespaths.ActiveSlug(root)
	if err != nil {
		return "", 2, err
	}
	if slug == "" {
		return "", 2, fmt.Errorf("no slug and ACTIVE is empty")
	}
	return slug, 0, nil
}
