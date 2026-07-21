package lib

import (
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

// findingLabel matches a bold severity label — the shape a *real* finding carries
// in review.md (`- **Critical** — ...`). A plain summary count ("Critical 0 /
// Important 0") is deliberately NOT matched: an all-zero tally is the rubber-stamp
// this gate exists to catch, not evidence that review happened.
var findingLabel = regexp.MustCompile(`\*\*(Critical|Important|Suggestion|Nit|FYI)\*\*`)

// noFindingsMarker matches the justification an adversarial reviewer must leave
// when it genuinely finds nothing — the axis stayed clean, and here is the
// account of the passes that came back empty. Presence is all this gate checks;
// the quality of the account is the reviewer fan-out's job (same contract as
// doubt-coverage's `doubt: MISSING` and footprint's roster).
var noFindingsMarker = regexp.MustCompile(`(?i)no[- ]findings`)

// reviewAxes are the adversarial axes rite-review always dispatches. Each writes a
// `## Spec` / `## Code review` section into review.md; a section that carries
// neither a finding nor a no-findings justification is a *silent* reviewer.
var reviewAxes = []struct {
	label   string   // display name
	headers []string // lower-cased heading spellings that open the section
}{
	{"Spec", []string{"spec"}},
	{"Code review", []string{"code review", "code-review"}},
}

// ReviewIntegrity guards the opposite failure from the rest of review discipline:
// not the noisy false positive (which confidence-banding suppresses) but the
// silent false negative — a reviewer that returns "looks good, nothing found".
// An empty findings list is treated as suspicious, not as a pass: an adversarial
// axis that reports nothing must justify the emptiness, exactly as it must quote a
// line to raise a finding. Read-only; the workspace is <root>/work/<slug>.
//
//	0  every axis accounted for — has findings, has a no-findings justification, or review.md absent/freeform
//	2  usage (no slug)
//	1  an adversarial axis is silent AND unjustified — a suspected rubber-stamp
func ReviewIntegrity(root string, args []string, stdout, stderr io.Writer) int {
	slug := argAt(args, 0)
	if slug == "" {
		slug = activeSlug(root)
	}
	if slug == "" {
		fmt.Fprintln(stderr, "usage: devrites-engine review-integrity <slug>")
		return 2
	}

	dir := featureDir(root, slug)
	review, ok := readFileOK(filepath.Join(dir, "review.md"))
	if !ok {
		fmt.Fprintln(stdout, "review-integrity: no review.md — nothing to assess (pass)")
		return 0
	}

	sections := axisSections(review)
	if len(sections) == 0 {
		// A freeform review with no `## Spec` / `## Code review` headings can't be
		// assessed mechanically — say so and pass rather than flag a format choice.
		fmt.Fprintln(stdout, "review-integrity: no per-axis sections in review.md — heading-based check n/a (pass)")
		return 0
	}

	var silent []string
	for _, ax := range reviewAxes {
		body, present := sections[ax.label]
		if !present {
			continue // axis not written as its own section — nothing to check here
		}
		hasFinding := findingLabel.MatchString(body)
		justified := noFindingsMarker.MatchString(body)
		switch {
		case hasFinding:
			fmt.Fprintf(stdout, "  %s: findings present\n", ax.label)
		case justified:
			fmt.Fprintf(stdout, "  %s: clean — no-findings justification present\n", ax.label)
		default:
			fmt.Fprintf(stdout, "  %s: SILENT (no finding, no justification)\n", ax.label)
			silent = append(silent, ax.label)
		}
	}

	if len(silent) > 0 {
		fmt.Fprintf(stdout, "review-integrity: SILENT — %s reviewed nothing and justified nothing.\n", strings.Join(silent, ", "))
		fmt.Fprintln(stdout, "A zero-findings axis is suspicious, not a pass. Re-run the axis, or record a")
		fmt.Fprintln(stdout, "'No-findings:' justification naming the adversarial passes that came back empty.")
		return 1
	}
	fmt.Fprintln(stdout, "review-integrity: OK — every adversarial axis has findings or a justified clean bill.")
	return 0
}

// axisSections splits review.md into the `## Spec` / `## Code review` axis bodies,
// keyed by the canonical axis label. A section runs from its `## ` heading to the
// next `## ` (or end of file); deeper `### ` headings stay inside it.
func axisSections(md string) map[string]string {
	// Map every recognised heading spelling to its canonical axis label.
	canon := map[string]string{}
	for _, ax := range reviewAxes {
		for _, h := range ax.headers {
			canon[h] = ax.label
		}
	}

	out := map[string]string{}
	curLabel := ""
	var cur strings.Builder
	flush := func() {
		if curLabel != "" {
			out[curLabel] = cur.String()
		}
		cur.Reset()
	}
	for _, line := range strings.Split(md, "\n") {
		if h, ok := cutH2(line); ok {
			flush()
			curLabel = canon[strings.ToLower(strings.TrimSpace(h))] // "" for a non-axis H2
			continue
		}
		if curLabel != "" {
			cur.WriteString(line)
			cur.WriteByte('\n')
		}
	}
	flush()
	return out
}

// cutH2 returns the text of a level-2 heading ("## Foo" → "Foo", true) and false
// for any other line. Deeper headings ("### ") are not level-2 and stay in the body.
func cutH2(line string) (string, bool) {
	t := strings.TrimRight(line, " \t")
	if strings.HasPrefix(t, "## ") && !strings.HasPrefix(t, "### ") {
		return strings.TrimSpace(t[len("## "):]), true
	}
	return "", false
}
