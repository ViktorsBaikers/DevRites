package lib

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/workflow"
)

// Analyze cross-checks a feature's spec.md against its tasks.md before any code is
// written, so a coverage gap surfaces as a one-line plan edit rather than a reslice
// mid-build. It emits a markdown report and flags:
//
//	Coverage    : a spec AC id that no slice Satisfies                 (CRITICAL)
//	Consistency : a slice that Satisfies an AC the spec never defines (CRITICAL)
//	Orphan slice: a slice that satisfies no acceptance criterion       (warn)
//	Ambiguity   : an unquantified vague adjective or unresolved
//	               placeholder in the spec                              (warn)
//
// It closes with a Metrics line (criteria count, coverage %, orphan + ambiguity
// counts) so the vet gate reports a number, not just a pass/fail.
//
// Exit codes: 0 clear · 1 at least one CRITICAL finding · 2 no workspace (no active
// slug, or spec.md/tasks.md missing).
var (
	taskACRe           = regexp.MustCompile(`\b(?:AC-[0-9]{3}|AC[0-9]+)\b`)
	sliceHeadRe        = regexp.MustCompile(`(?i)^##[[:space:]]*(?:SLICE-[0-9]{3}\b|Slice\b)`)
	sliceNamePrefixRe  = regexp.MustCompile(`^##[[:space:]]*`) // stripped to leave the slice name
	analyzeSatisfiesRe = regexp.MustCompile(`^[[:space:]]*Satisfies:`)
	// Ambiguity scan: a vague quality adjective with no number on the line is an
	// unfalsifiable criterion; an unresolved placeholder is unfinished spec text.
	vagueAdjRe     = regexp.MustCompile(`(?i)\b(fast|scalable|secure|intuitive|robust|performant|seamless|efficient|flexible|reliable|user-friendly|graceful(?:ly)?|snappy|lightweight)\b`)
	analyzeDigitRe = regexp.MustCompile(`[0-9]`)
	placeholderRe  = regexp.MustCompile(`(?i)(\bTODO\b|\bTKTK\b|\bFIXME\b|\bXXX\b|\?\?\?|<placeholder>)`)
	fenceRe        = regexp.MustCompile("^[[:space:]]*```")
)

func Analyze(root string, args []string, stdout, stderr io.Writer) int {
	slug := slugOrActive(root, args)
	if slug == "" {
		fmt.Fprintln(stderr, "analyze: no active workspace.")
		return 2
	}

	spec := featureFile(root, slug, "spec.md")
	tasks := featureFile(root, slug, "tasks.md")
	if !isFile(spec) || !isFile(tasks) {
		fmt.Fprintf(stderr, "analyze: need spec.md + tasks.md in %s.\n", featureDir(root, slug))
		return 2
	}

	specData, err := os.ReadFile(spec)
	if err != nil {
		fmt.Fprintf(stderr, "analyze: read spec.md: %v\n", err)
		return 2
	}
	tasksData, err := os.ReadFile(tasks)
	if err != nil {
		fmt.Fprintf(stderr, "analyze: read tasks.md: %v\n", err)
		return 2
	}

	specACs := sortedACIDs(acIDRe, specData, true) // legacy brackets stripped: "[AC1]" -> "AC1"
	taskACs := sortedACIDs(taskACRe, tasksData, false)
	specSet := stringSet(specACs)
	taskSet := stringSet(taskACs)

	crit := 0
	fmt.Fprintf(stdout, "# Cross-artifact analysis: %s\n", slug)
	fmt.Fprintln(stdout)

	// Coverage: spec ACs with no slice reference.
	covered := 0
	fmt.Fprintln(stdout, "## Coverage")
	if len(specACs) == 0 {
		fmt.Fprintln(stdout, "- [warn] spec.md: no AC-### acceptance ids found; tag criteria for a machine-checkable gate.")
	} else {
		for _, ac := range specACs {
			if taskSet[ac] {
				covered++
				fmt.Fprintf(stdout, "- [ok] %s: covered\n", ac)
			} else {
				fmt.Fprintf(stdout, "- [CRITICAL] %s (spec.md): no slice Satisfies it (uncovered)\n", ac)
				crit++
			}
		}
	}

	// Dangling refs: tasks Satisfies an AC the spec doesn't define.
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "## Consistency")
	if len(taskACs) > 0 {
		for _, ac := range taskACs {
			if !specSet[ac] {
				fmt.Fprintf(stdout, "- [CRITICAL] %s (tasks.md): referenced by a slice but not defined in spec.md (dangling)\n", ac)
				crit++
			}
		}
	}

	// Orphan slices: a slice with no Satisfies line (printed directly, no blank before).
	orphans := 0
	for _, name := range orphanSlices(tasksData) {
		if name != "" {
			orphans++
			fmt.Fprintf(stdout, "- [warn] slice '%s': satisfies no acceptance criterion (add Satisfies, or justify)\n", name)
		}
	}

	// Ambiguity: unquantified vague adjectives + unresolved placeholders in the spec.
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "## Ambiguity")
	ambig := ambiguityFindings(specData)
	if len(ambig) == 0 {
		fmt.Fprintln(stdout, "- [ok] no unquantified adjectives or unresolved placeholders found")
	} else {
		for _, line := range ambig {
			fmt.Fprintln(stdout, line)
		}
	}

	// Metrics: a single machine-readable line so the vet gate reports a number.
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "## Metrics")
	if len(specACs) == 0 {
		fmt.Fprintf(stdout, "- Acceptance criteria: 0 (none tagged) · orphan slices %d · ambiguity flags %d\n", orphans, len(ambig))
	} else {
		fmt.Fprintf(stdout, "- Acceptance criteria: %d · covered %d/%d (%d%%) · orphan slices %d · ambiguity flags %d\n",
			len(specACs), covered, len(specACs), covered*100/len(specACs), orphans, len(ambig))
	}

	fmt.Fprintln(stdout)
	if crit > 0 {
		fmt.Fprintf(stdout, "## Verdict: BLOCKED: %d CRITICAL finding(s). Resolve before %s.\n", crit, workflow.ForVerb("build").Both())
		return 1
	}
	fmt.Fprintln(stdout, "## Verdict: clear: spec/tasks consistent and fully mapped.")
	return 0
}

// ambiguityFindings scans spec text for unfalsifiable criteria: a vague quality
// adjective (fast, robust, secure …) on a line with no number to pin it down, or an
// unresolved placeholder (TODO, ???, <placeholder> …). Code fences are skipped. One
// finding per line: a placeholder takes precedence over an adjective. 1-based lines.
func ambiguityFindings(specData []byte) []string {
	var out []string
	inFence := false
	for i, line := range splitLinesNoTrailing(specData) {
		if fenceRe.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		ln := i + 1
		if m := placeholderRe.FindString(line); m != "" {
			out = append(out, fmt.Sprintf("- [warn] spec.md:L%d: unresolved placeholder %q", ln, m))
			continue
		}
		// Strip the AC id before the digit gate: else the id's own digit masks
		// a vague adjective on the very criterion line it tags.
		bare := acIDRe.ReplaceAllString(line, "")
		bare = taskACRe.ReplaceAllString(bare, "")
		if !analyzeDigitRe.MatchString(bare) {
			if m := vagueAdjRe.FindString(bare); m != "" {
				out = append(out, fmt.Sprintf("- [warn] spec.md:L%d: %q is not quantified; state a measurable target", ln, m))
			}
		}
	}
	return out
}

// sortedACIDs collects every acceptance id matched by re, deduplicated and sorted
// lexicographically. When strip is set the surrounding brackets are removed, so
// Legacy "[AC1]" becomes "AC1". Returns nil when there are none.
func sortedACIDs(re *regexp.Regexp, data []byte, strip bool) []string {
	seen := map[string]bool{}
	var out []string
	for _, line := range splitLinesNoTrailing(data) {
		for _, m := range re.FindAllString(line, -1) {
			if strip {
				m = strings.NewReplacer("[", "", "]", "").Replace(m)
			}
			if !seen[m] {
				seen[m] = true
				out = append(out, m)
			}
		}
	}
	sort.Strings(out)
	return out
}

// stringSet builds a membership set for O(1) lookups.
func stringSet(in []string) map[string]bool {
	m := make(map[string]bool, len(in))
	for _, s := range in {
		m[s] = true
	}
	return m
}

// orphanSlices returns the names of slices in tasks.md that have no Satisfies:
// line. A "## Slice ..." header opens a slice; any Satisfies: line before the next
// header marks it satisfied. Each header (and end of file) closes the previous
// slice, emitting it when it went unsatisfied.
func orphanSlices(data []byte) []string {
	var out []string
	name := ""
	sat := false
	for _, line := range splitLinesNoTrailing(data) {
		if sliceHeadRe.MatchString(line) {
			if name != "" && !sat {
				out = append(out, name)
			}
			name = sliceNamePrefixRe.ReplaceAllString(line, "")
			sat = false
		}
		if analyzeSatisfiesRe.MatchString(line) {
			sat = true
		}
	}
	if name != "" && !sat {
		out = append(out, name)
	}
	return out
}
