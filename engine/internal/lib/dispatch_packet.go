package lib

import (
	"fmt"
	"regexp"
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
