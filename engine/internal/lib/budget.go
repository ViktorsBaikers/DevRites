package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// The context-budget lint. A feature is deliberately a directory of small,
// single-concern files rather than one long document: each file stays
// context-cheap to read, and completeness is self-evident from which files
// exist. That discipline only holds if the files stay small — a spec.md that
// grows to 800 lines is a "sausage" that an agent must load whole, spending the
// context window it was meant to conserve. Budget measures every workspace file
// against a per-file line ceiling and flags the overgrown ones.
//
// It is advisory by default (exit 0): the ceilings are guidance, and a passing
// build is never blocked on a long file. Pass --strict to turn an overflow into
// the standard block exit (3), so a gate or hook can hold on it.

// budgetLimits is the per-file line ceiling for the known workspace files, in the
// order they are reported. The order is the workflow's reading order (manifest
// first, then the spec-through-proof arc, then the working and cross-cutting
// notes), not alphabetical, so the report reads top-to-bottom like the workspace.
var budgetLimits = []struct {
	name  string
	limit int
}{
	{"README.md", 120},           // workspace map and read-next table
	{"index.md", 120},            // workspace map alias
	{"feature.md", 120},          // legacy workspace map alias
	{"brief.md", 80},             // objective and bounds
	{"spec.md", 260},             // product WHAT/WHY and acceptance contract
	{"ai-spec.md", 160},          // optional AI/LLM annex
	{"architecture.md", 180},     // technical map
	{"flows.md", 160},            // useful diagrams only
	{"plan.md", 220},             // approach and slice strategy
	{"tasks.md", 280},            // vertical slices
	{"traceability.md", 220},     // AC/REQ -> slice -> proof matrix
	{"decisions.md", 200},        // ADR-style decision log
	{"assumptions.md", 160},      // assumption register
	{"questions.md", 180},        // human questions and gates
	{"drift.md", 160},            // spec/plan drift register
	{"state.md", 120},            // compact cursor
	{"status.md", 120},           // status checkpoint alias
	{"evidence.md", 280},         // command/action proof
	{"proof.md", 280},            // evidence alias
	{"browser-evidence.md", 220}, // UI/browser proof
	{"touched-files.md", 160},    // implementation file map
	{"design-brief.md", 160},     // UI design contract
	{"handoff.md", 120},          // cold-resume summary
	{"references.md", 160},       // source pointers
	{"learnings.md", 200},        // retrospective ledger
}

// defaultBudget is the ceiling applied to any other *.md file in the workspace
// that has no explicit budget above.
const defaultBudget = 200

// Budget lints every present workspace file for a feature against its line
// ceiling. root is the .devrites directory; the slug defaults to the active
// feature at <root>/ACTIVE. Args: an optional slug and an optional --strict flag.
func Budget(root string, args []string, stdout, stderr io.Writer) int {
	strict := false
	slug := ""
	for _, a := range args {
		switch {
		case a == "--strict":
			strict = true
		case slug == "":
			slug = a
		default:
			fmt.Fprintln(stderr, "usage: devrites-engine budget [slug] [--strict]")
			return 2
		}
	}
	if slug == "" {
		slug = activeSlug(root)
	}
	if slug == "" {
		fmt.Fprintln(stderr, "budget: no active feature (set one via ACTIVE or pass a slug)")
		return 2
	}
	dir := featureDir(root, slug)
	if !isDir(dir) {
		fmt.Fprintf(stderr, "budget: no such feature: %s\n", slug)
		return 2
	}

	rows := collectBudgetRows(dir)

	fmt.Fprintf(stdout, "budget: %s\n", slug)
	over := 0
	for _, r := range rows {
		status := "ok"
		if r.lines > r.limit {
			status = fmt.Sprintf("OVER by %d", r.lines-r.limit)
			over++
		}
		fmt.Fprintf(stdout, "  %-14s %5d lines  ~%5d tok  limit %4d  %s\n",
			r.name, r.lines, r.tokens, r.limit, status)
	}

	if over == 0 {
		fmt.Fprintf(stdout, "budget: all %d file(s) within budget\n", len(rows))
		return 0
	}
	if strict {
		fmt.Fprintf(stdout, "budget: %d file(s) over the line budget — split or trim them\n", over)
		return 3
	}
	fmt.Fprintf(stdout, "budget: %d file(s) over the line budget (advisory — pass --strict to fail)\n", over)
	return 0
}

// budgetRow is one measured file in the report.
type budgetRow struct {
	name   string
	lines  int
	tokens int
	limit  int
}

// collectBudgetRows measures every present workspace file: the known files (in
// canonical reading order) followed by any other *.md file (sorted), so the
// output is a deterministic function of the directory contents.
func collectBudgetRows(dir string) []budgetRow {
	known := make(map[string]bool, len(budgetLimits))
	var rows []budgetRow
	for _, b := range budgetLimits {
		known[b.name] = true
		if lines, tokens, ok := measureFile(filepath.Join(dir, b.name)); ok {
			rows = append(rows, budgetRow{b.name, lines, tokens, b.limit})
		}
	}
	// Any other Markdown file in the workspace gets the default ceiling. feature.md
	// et al. are already covered above; a stray notes.md is measured too so nothing
	// grows unmeasured.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return rows
	}
	var extras []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || known[name] || filepath.Ext(name) != ".md" {
			continue
		}
		extras = append(extras, name)
	}
	for _, name := range extras {
		if lines, tokens, ok := measureFile(filepath.Join(dir, name)); ok {
			rows = append(rows, budgetRow{name, lines, tokens, defaultBudget})
		}
	}
	return rows
}

// measureFile returns a file's line count and an approximate token count
// (bytes/4, the standard rough English heuristic), or ok=false if it cannot be
// read. Lines count text records, ignoring a single trailing newline, so a
// newline-terminated N-line file measures N, not N+1.
func measureFile(path string) (lines, tokens int, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, false
	}
	lines = len(splitLinesNoTrailing(data))
	tokens = (len(data) + 3) / 4
	return lines, tokens, true
}
