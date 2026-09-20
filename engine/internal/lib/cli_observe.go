package lib

import (
	"fmt"
	"io"

	"github.com/devrites/devrites/internal/devritespaths"
)

// RunTaskGraphCheck validates tasks.md for one slug.
func RunTaskGraphCheck(root, slug string, stdout, stderr io.Writer) int {
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
