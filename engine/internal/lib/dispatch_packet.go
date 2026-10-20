package lib

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// seedingRE matches packet text that hands a reviewer a conclusion instead of
// scope: a gate's success token, a reviewer outcome or count, or a claim that
// something already passed. Each one voids an independent review.
var seedingRE = regexp.MustCompile(`\b[A-Z][A-Z0-9]*(?:-[A-Z0-9]+)+-PASS\b` +
	`|(?i:\bno-findings\b)` +
	`|(?i:^\s*(?:outcome|counts|verdict)\s*:\s*[^<\s])` +
	`|(?i:\b(?:already|previously)\s+(?:passed|passes|green|printed|verified|approved|clean)\b)` +
	`|(?i:\b(?:critical|important)\s*[:=]?\s*0\b)`)

// countsTemplateRE matches a Counts line whose every value is a placeholder
// (the return template), which names keys but no conclusion.
var countsTemplateRE = regexp.MustCompile(`(?i)^\s*counts\s*:(?:\s+[a-z]+=<[^<>]*>)+\s*$`)

// isReviewerRole reports whether role is an independent read-only reviewer:
// a *-reviewer, *-auditor, or test-analyst, with an optional ".lane" suffix.
func isReviewerRole(role string) bool {
	base, _, _ := strings.Cut(role, ".")
	return strings.HasSuffix(base, "-reviewer") || strings.HasSuffix(base, "-auditor") || base == "test-analyst"
}

// lintReviewerPacket refuses a reviewer packet that seeds a conclusion, so a
// voided review is rejected before launch rather than discovered after it.
// Placeholder text inside <...> (an output template) is not a conclusion.
func lintReviewerPacket(path string) error {
	if path == "" {
		return fmt.Errorf("a reviewer wave needs --packet <file> holding the exact packet the reviewers receive")
	}
	raw, err := readBoundedRegularFile(path, maxContractBytes)
	if err != nil {
		return fmt.Errorf("reviewer packet %s: %w", path, err)
	}
	for i, line := range strings.Split(string(raw), "\n") {
		if countsTemplateRE.MatchString(line) {
			continue
		}
		for _, loc := range seedingRE.FindAllStringIndex(line, -1) {
			if insidePlaceholder(line, loc[0], loc[1]) {
				continue
			}
			return fmt.Errorf("reviewer packet %s:%d seeds a conclusion (%q); a packet carries scope, paths, diff and rubric only — never a gate result, a reviewer outcome or count, or a claim that something already passed. Weigh root proof at reconciliation instead",
				path, i+1, strings.TrimSpace(line[loc[0]:loc[1]]))
		}
	}
	return nil
}

func insidePlaceholder(line string, start, end int) bool {
	open := strings.LastIndex(line[:start], "<")
	return open >= 0 && open > strings.LastIndex(line[:start], ">") && strings.Contains(line[end:], ">")
}

// severityCountKeys is the Counts vocabulary of a reviewer role that
// returnCountKeys does not list.
var severityCountKeys = []string{"critical", "important", "suggestion", "nit", "fyi", "gap"}

// returnCountKeys is the closed, ordered Counts vocabulary of the reviewer
// roles that differ from severityCountKeys. The agent-template head test reads
// this table, so the engine and the templates share one source.
var returnCountKeys = map[string][]string{
	"plan-reviewer":       {"critical", "important", "suggestion", "nit", "fyi", "gap", "contract", "mechanism"},
	"simplifier-reviewer": {"suggestion", "nit", "fyi", "gap"},
}

func countKeys(role string) []string {
	base, _, _ := strings.Cut(role, ".")
	if keys, ok := returnCountKeys[base]; ok {
		return keys
	}
	return severityCountKeys
}

var countRE = regexp.MustCompile(`^([a-z]+)=([0-9]{1,6})$`)

// parseReturnHead reads a reviewer return's strict head: line 1
// "Outcome: findings|no-findings|gap", line 2 "Counts: key=N ..." with keys
// from the role's vocabulary. findings needs a non-gap count above 0 and
// no-findings needs every count at 0. It returns the outcome and the counts as
// "key=N,key=N"; an error means the head is malformed.
func parseReturnHead(role, text string) (outcome, counts string, err error) {
	lines := strings.SplitN(strings.TrimLeft(text, "\r\n"), "\n", 3)
	if len(lines) < 2 {
		return "", "", errors.New("no Outcome/Counts head on lines 1-2")
	}
	outcome, ok := strings.CutPrefix(strings.TrimSpace(lines[0]), "Outcome: ")
	if !ok || (outcome != "findings" && outcome != "no-findings" && outcome != "gap") {
		return "", "", fmt.Errorf("line 1 is not \"Outcome: findings|no-findings|gap\": %q", strings.TrimSpace(lines[0]))
	}
	rest, ok := strings.CutPrefix(strings.TrimSpace(lines[1]), "Counts:")
	fields := strings.Fields(rest)
	if !ok || len(fields) == 0 {
		return "", "", fmt.Errorf("line 2 is not \"Counts: key=N ...\": %q", strings.TrimSpace(lines[1]))
	}
	allowed := countKeys(role)
	seen := map[string]bool{}
	total, nonGap := 0, 0
	for _, field := range fields {
		m := countRE.FindStringSubmatch(field)
		if m == nil || !slices.Contains(allowed, m[1]) || seen[m[1]] {
			return "", "", fmt.Errorf("line 2 has Counts entry %q, which is not a distinct key=N with key in %s", field, strings.Join(allowed, ","))
		}
		seen[m[1]] = true
		n, _ := strconv.Atoi(m[2])
		total += n
		if m[1] != "gap" {
			nonGap += n
		}
	}
	if outcome == "findings" && nonGap == 0 {
		return "", "", errors.New("line 1 says findings but no non-gap count is above 0")
	}
	if outcome == "no-findings" && total > 0 {
		return "", "", errors.New("line 1 says no-findings but a count is above 0")
	}
	return outcome, strings.Join(fields, ","), nil
}

var siteLineRE = regexp.MustCompile(`(?::[0-9]+)+(?:-[0-9]+)?$`)

// findingKeys returns one role|slice|file|section|severity key per "Finding:"
// row and the number of "Late:" lines. The site drops its line number so a
// re-raise after an edit still matches, and spaces become "_" so each key is
// one note token.
// shortcut: cells split on every "|", so a quoted "|" shifts the cells; parse
// the quote if keys come out wrong.
func findingKeys(role, slice, text string) (keys []string, late int) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Late:") {
			late++
			continue
		}
		row, ok := strings.CutPrefix(line, "Finding:")
		if !ok {
			continue
		}
		cells := strings.Split(row, "|")
		file, section := "", ""
		if len(cells) > 1 {
			file = siteLineRE.ReplaceAllString(strings.TrimSpace(cells[1]), "")
		}
		for _, cell := range cells[min(2, len(cells)):] {
			if cited, ok := strings.CutPrefix(strings.TrimSpace(cell), "breaks "); ok {
				section, _, _ = strings.Cut(cited, `"`)
				break
			}
		}
		severity, _, _ := strings.Cut(strings.TrimSpace(cells[0]), " ")
		keys = append(keys, strings.Join([]string{role, slice, keyToken(file), keyToken(section), keyToken(strings.ToLower(severity))}, "|"))
	}
	return keys, late
}

func keyToken(s string) string {
	if fields := strings.Fields(s); len(fields) > 0 {
		return strings.Join(fields, "_")
	}
	return "-"
}
