package state

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/markdowntext"
)

var (
	acceptanceHeadingRE = regexp.MustCompile(`(?i)^##\s+Acceptance criteria\s*$`)
	canonicalACRE       = regexp.MustCompile(`\bAC-\d{3}\b`)
	acceptanceEndRE     = regexp.MustCompile(`^#{1,2}\s+`)
	acDeclarationRE     = regexp.MustCompile("^\\s*(?:#{1,6}\\s+|[-+*]\\s+(?:\\[[ xX]\\]\\s+)?|\\d+[.)]\\s+|\\|\\s*)?(?:\\*\\*|__|`)?(AC-\\d{3})(?:\\*\\*|__|`)?(?:\\s*[:|]|[ \\t]+[-–—])")
)

// AcceptanceMapResult is the deterministic ID-presence check from spec.md
// onto tasks.md and test-plan.md. It does not judge coverage quality.
type AcceptanceMapResult struct {
	SpecIDs  []string
	Problems []string
}

// ParseAcceptanceMap extracts canonical AC-### IDs from spec.md's Acceptance
// criteria section and requires each to appear as a literal substring in the
// required planning artifacts.
func ParseAcceptanceMap(spec, tasks, testPlan []byte, requireTasks, requireTestPlan bool) AcceptanceMapResult {
	ids := uniqueCanonicalACs(acceptanceSection(spec))
	result := AcceptanceMapResult{SpecIDs: ids}
	if len(ids) == 0 {
		return result
	}
	if requireTasks {
		text := string(tasks)
		for _, id := range ids {
			if !strings.Contains(text, id) {
				result.Problems = append(result.Problems, fmt.Sprintf("acceptance %s is not referenced in tasks.md", id))
			}
		}
	}
	if requireTestPlan {
		text := string(testPlan)
		for _, id := range ids {
			if !strings.Contains(text, id) {
				result.Problems = append(result.Problems, fmt.Sprintf("acceptance %s is not referenced in test-plan.md", id))
			}
		}
	}
	return result
}

func acceptanceSection(spec []byte) string {
	structural, err := markdowntext.Prose(spec)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(string(structural), "\r\n", "\n"), "\n")
	start, end := acceptanceSectionBounds(lines)
	if start < 0 {
		return ""
	}
	return strings.Join(lines[start:end], "\n")
}

// AcceptanceDeclarationsOutsideSection identifies definitions, not prose
// references, that the canonical section parser would otherwise silently skip.
func AcceptanceDeclarationsOutsideSection(spec []byte) []string {
	structural, err := markdowntext.Prose(spec)
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(string(structural), "\r\n", "\n"), "\n")
	start, end := acceptanceSectionBounds(lines)
	var ids []string
	for i, line := range lines {
		if start >= 0 && i >= start && i < end {
			continue
		}
		if match := acDeclarationRE.FindStringSubmatch(line); match != nil {
			ids = append(ids, match[1])
		}
	}
	return ids
}

func acceptanceSectionBounds(lines []string) (int, int) {
	start := -1
	for i, line := range lines {
		if acceptanceHeadingRE.MatchString(strings.TrimSpace(line)) {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return -1, -1
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		if acceptanceEndRE.MatchString(strings.TrimSpace(lines[i])) {
			end = i
			break
		}
	}
	return start, end
}

func uniqueCanonicalACs(text string) []string {
	found := canonicalACRE.FindAllString(text, -1)
	if len(found) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(found))
	var ids []string
	for _, id := range found {
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}
