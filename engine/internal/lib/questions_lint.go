package lib

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	questionLineAdvisory  = 300
	questionEntryAdvisory = 2500
)

var (
	questionIDPattern   = regexp.MustCompile(`^q-[0-9]{4}-[0-9]{2}-[0-9]{2}-[0-9]{3}$`)
	questionKeyPattern  = regexp.MustCompile(`^([a-z_]+):(.*)$`)
	questionItemPattern = regexp.MustCompile(`^\s*[0-9]+\.\s+(.*)$`)
	questionGates       = []string{"advisory", "validating", "blocking", "escalating"}
)

// lintQuestionEntry checks a new questions.md `## q-...` entry against the
// afk-hitl.md schema. Problems block the append; advisories only warn. A
// recommendation stays optional: without one, proposed: must be absent so the
// AFK auto-pick never reads a non-recommendation as an answer.
func lintQuestionEntry(heading string, bodyLines []string, bodySize int) (problems, advisories []string) {
	if id := headingID(heading); !questionIDPattern.MatchString(id) {
		problems = append(problems, fmt.Sprintf("heading %q is not q-YYYY-MM-DD-NNN", heading))
	}
	values := map[string][]string{}
	var options []string
	inOptions := false
	for _, line := range bodyLines {
		if m := questionKeyPattern.FindStringSubmatch(line); m != nil {
			values[m[1]] = append(values[m[1]], strings.TrimSpace(m[2]))
			inOptions = m[1] == "options"
			continue
		}
		if m := questionItemPattern.FindStringSubmatch(line); inOptions && m != nil {
			options = append(options, strings.TrimSpace(m[1]))
		}
	}
	one := func(key string) (string, bool) {
		if len(values[key]) != 1 {
			problems = append(problems, fmt.Sprintf("needs exactly one %s: line", key))
			return "", false
		}
		return values[key][0], true
	}
	if v, ok := one("status"); ok && v != "open" {
		problems = append(problems, "status must be open (state resolve answers or drops it)")
	}
	if v, ok := one("slice"); ok && v == "" {
		problems = append(problems, "slice is empty")
	}
	if v, ok := one("gate"); ok && !slices.Contains(questionGates, v) {
		problems = append(problems, "gate must be "+strings.Join(questionGates, "|"))
	}
	if v, ok := one("question"); ok {
		if v == "" {
			problems = append(problems, "question is empty")
		} else if n := len([]rune(v)); n > questionLineAdvisory {
			advisories = append(advisories, fmt.Sprintf("question line is %d chars; aim for one crisp sentence under %d", n, questionLineAdvisory))
		}
	}
	if v, ok := one("raised_at"); ok {
		if _, err := time.Parse(time.RFC3339, v); err != nil {
			problems = append(problems, fmt.Sprintf("raised_at %q is not a full RFC 3339 timestamp such as 2026-10-19T08:17:08Z", v))
		}
	}
	if _, ok := one("options"); ok {
		if len(options) < 3 || len(options) > 5 {
			problems = append(problems, fmt.Sprintf("options has %d numbered items; want 2-4 concrete options plus the escape hatch", len(options)))
		} else if !strings.HasPrefix(strings.ToLower(options[len(options)-1]), "something else") {
			problems = append(problems, "the last option must be the escape hatch \"Something else — describe it\"")
		}
	}
	recommended := -1
	for i, opt := range options {
		if strings.Contains(strings.ToLower(opt), "(recommended)") {
			if recommended >= 0 || i != 0 {
				problems = append(problems, "only option 1 may carry (Recommended)")
				break
			}
			recommended = i
		}
	}
	switch proposed := values["proposed"]; {
	case recommended == 0 && (len(proposed) != 1 || proposed[0] == ""):
		problems = append(problems, "option 1 is (Recommended), so add one proposed: line restating it")
	case recommended < 0 && len(proposed) > 0:
		problems = append(problems, "proposed: is the AFK auto-pick; omit it when no option is (Recommended)")
	}
	if bodySize > questionEntryAdvisory {
		advisories = append(advisories, fmt.Sprintf("entry is %d bytes; keep impact and evidence, trim repetition below %d", bodySize, questionEntryAdvisory))
	}
	return problems, advisories
}
