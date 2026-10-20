package lib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
)

// seedingRE matches packet text that hands a reviewer a conclusion instead of
// scope: a gate's success token, a reviewer outcome or count, or a claim that
// something already passed. Each one voids an independent review.
var seedingRE = regexp.MustCompile(`\b[A-Z][A-Z0-9]*(?:-[A-Z0-9]+)+-PASS\b` +
	`|(?i:\bno-findings\b)` +
	`|(?i:^\s*(?:outcome|counts|verdict)\s*:\s*[^<\s])` +
	`|(?i:\b(?:already|previously)\s+(?:passed|passes|green|printed|verified|approved|clean|fixed|closed|resolved)\b)` +
	`|(?i:\b(?:critical|important)\s*[:=]?\s*0\b)` +
	// Phrasings observed in voided reviews: a parent/sibling report, a passed
	// check, a returned severity or finding, a green outcome, a return file.
	`|(?i:\b(?:parent|root|sibling|drafter|wright|implementer)(?:\s+self-attack)?\s+(?:observed|reports?|reported|concluded|confirmed|exposes|exposed)\b)` +
	`|(?i:\b(?:check|gate|scope|tests?)\s+passed\b)` +
	`|(?i:\breturned\s+(?:critical|important|minor|findings|no-findings)\b)` +
	`|(?i:\b(?:critical|important|minor)\s+F\d+\b)` +
	`|(?i:\bclosed\s+by\b)` +
	`|(?i:\b(?:makes|made|turned|went)\s+[\w/-]*\s*green\b)` +
	`|packets/\S+\.return\.md`)

// roleTagRE matches a role-tagged finding fingerprint ("Plan F1", "DevEx S2").
// The tag is a reviewer role's leading word: Perf for performance-reviewer.
var roleTagRE = regexp.MustCompile(`\b(Plan|Doubt|Code|DevEx|Test|Security|Spec|Strategy|Perf|Frontend|Simplifier)\s+[FS]\d+\b`)

// countsTemplateRE matches a Counts line whose every value is a placeholder
// (the return template), which names keys but no conclusion.
var countsTemplateRE = regexp.MustCompile(`(?i)^\s*counts\s*:(?:\s+[a-z]+=<[^<>]*>)+\s*$`)

// roleBase is the canonical role a --role names: an optional "devrites-"
// agent-name prefix and ".lane" suffix dropped.
func roleBase(role string) string {
	base, _, _ := strings.Cut(strings.TrimPrefix(role, "devrites-"), ".")
	return base
}

// isReviewerRole reports whether role is an independent read-only reviewer:
// a *-reviewer, *-auditor, or test-analyst.
func isReviewerRole(role string) bool {
	base := roleBase(role)
	return strings.HasSuffix(base, "-reviewer") || strings.HasSuffix(base, "-auditor") || base == "test-analyst"
}

// lintKnownRoles refuses a --role that names no pack agent, so a free-form
// label ("test-analyst-022-r5") cannot skip the reviewer lints. The agent path
// resolves as context --role does: the phase skill's loads manifest, else the
// devrites-<role> convention, in any project-level host install; a Codex
// install keeps role agents at .codex/agents/<name>.toml.
// shortcut: only project-level installs are checked (a global-only install or
// none skips the check); check the user install too if free-form roles recur there.
func lintKnownRoles(root, phase string, roles []string) error {
	project := filepath.Dir(root)
	var skillsRoots []string
	if dir, err := resolveSkillsRoot(root, ""); err == nil {
		if rel, err := filepath.Rel(project, dir); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			skillsRoots = append(skillsRoots, dir)
		}
	}
	for _, rel := range []string{".omp/skills", ".agents/skills", ".pi/skills", ".devin/skills"} {
		dir := filepath.Join(project, filepath.FromSlash(rel))
		if info, err := os.Stat(filepath.Join(dir, "devrites-lib")); err == nil && info.IsDir() {
			skillsRoots = append(skillsRoots, dir)
		}
	}
	manifests := make([]loadsManifest, len(skillsRoots))
	for i, dir := range skillsRoots {
		if raw, err := os.ReadFile(filepath.Join(dir, "rite-"+strings.ToLower(phase), "SKILL.md")); err == nil { // #nosec G304 -- resolved under a project skills root
			manifests[i], _, _ = parseLoadsManifest(string(raw))
		}
	}
	isRegular := func(path string) bool {
		info, err := os.Stat(path)
		return err == nil && info.Mode().IsRegular()
	}
	for _, role := range roles {
		base := roleBase(role)
		known := len(skillsRoots) == 0
		for i, dir := range skillsRoots {
			if !devritespaths.ValidSlug(base) {
				break
			}
			rel := filepath.FromSlash(roleAgentPath(manifests[i], base))
			if isRegular(filepath.Join(dir, "..", "agents", rel)) ||
				isRegular(filepath.Join(project, ".codex", "agents", strings.TrimSuffix(filepath.Base(rel), ".md")+".toml")) {
				known = true
				break
			}
		}
		if !known {
			return fmt.Errorf("role %q names no pack agent; use a canonical role, optionally with a .<lane> suffix", role)
		}
	}
	return nil
}

// lintReviewerPacket refuses a reviewer packet that seeds a conclusion or
// tells the reviewer to read its installed role contract, so a voided review
// is rejected before launch rather than discovered after it.
// Placeholder text inside <...> (an output template) is not a conclusion.
// With no wave roles, every role-tagged fingerprint is refused.
func lintReviewerPacket(path string) error {
	return lintReviewerWavePacket(path, nil)
}

// lintReviewerWavePacket is lintReviewerPacket for a wave: a role-tagged
// fingerprint passes only when roles hold exactly one reviewer role and the
// tag names it.
func lintReviewerWavePacket(path string, roles []string) error {
	if path == "" {
		return fmt.Errorf("a reviewer wave needs --packet <file> holding the exact packet the reviewers receive")
	}
	raw, err := readBoundedRegularFile(path, maxContractBytes)
	if err != nil {
		return fmt.Errorf("reviewer packet %s: %w", path, err)
	}
	var reviewers []string
	for _, role := range roles {
		if base := roleBase(role); isReviewerRole(base) && !slices.Contains(reviewers, base) {
			reviewers = append(reviewers, base)
		}
	}
	for i, line := range strings.Split(string(raw), "\n") {
		if countsTemplateRE.MatchString(line) {
			continue
		}
		for _, loc := range seedingRE.FindAllStringIndex(line, -1) {
			if insidePlaceholder(line, loc[0], loc[1]) {
				continue
			}
			return fmt.Errorf("reviewer packet %s:%d seeds a conclusion (%q); a packet carries scope, paths, diff and rubric only — never a gate result, a reviewer outcome or count, or a claim that something already passed (agents.md § Independence). Weigh root proof at reconciliation instead",
				path, i+1, strings.TrimSpace(line[loc[0]:loc[1]]))
		}
		for _, m := range roleTagRE.FindAllStringSubmatchIndex(line, -1) {
			tag := strings.ToLower(line[m[2]:m[3]])
			if insidePlaceholder(line, m[0], m[1]) || (len(reviewers) == 1 && strings.HasPrefix(reviewers[0], tag)) {
				continue
			}
			return fmt.Errorf("reviewer packet %s:%d carries fingerprint %q from another role; a recheck packet lists only fingerprints that same role raised, so send it to that role's recheck alone (agents.md § Independence)",
				path, i+1, line[m[0]:m[1]])
		}
	}
	return lintContractRead("reviewer packet", path, raw)
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
	if keys, ok := returnCountKeys[roleBase(role)]; ok {
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
