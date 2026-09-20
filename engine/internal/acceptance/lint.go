package acceptance

import (
	"fmt"
	"regexp"
	"strings"
)

// Finding is one advisory lint observation: a lexical sign that an oracle may
// be unable to fail honestly. Findings are prompts to sharpen a gate, not
// proof the outcome is wrong.
type Finding struct {
	Level   string // "error" or "warn"
	Gate    string
	Rule    string
	Message string
}

var (
	fixedOutputRE = regexp.MustCompile(`^\s*(?:(?:echo|printf)(?:\s+[^&|;]*)?|true|:|exit\s+0)\s*$`)
	weakExpect    = map[string]bool{
		"ok": true, "okay": true, "done": true, "pass": true, "passed": true,
		"success": true, "successful": true, "succeeded": true, "complete": true,
		"completed": true, "finished": true, "yes": true, "true": true,
		"0": true, "good": true, "fine": true, "working": true,
	}
	activityStartRE = regexp.MustCompile(`(?i)^(work(ing)? on|improve|enhance|handle|support|ensure|make sure|try|attempt|look (at|into)|investigate|consider|review|refactor|clean ?up|polish|update|tidy|address|deal with|add support)\b`)
)

// LintLedger audits oracle quality without executing anything: fixed-output
// commands, expectations that also appear in failure output, path-shaped
// regexes, activity titles, unmeasured numbers, and mostly-manual ledgers.
// approved is the vetted (command, cwd) surface; a runnable CHECK outside it
// is an authoring error because the runner can never execute it.
func LintLedger(doc *Document, approved []ApprovedCommand) []Finding {
	var findings []Finding
	add := func(level, gate, rule, format string, args ...any) {
		findings = append(findings, Finding{Level: level, Gate: gate, Rule: rule, Message: fmt.Sprintf(format, args...)})
	}
	for _, err := range doc.Errors {
		add("error", "", "parse", "%s", err)
	}

	live := 0
	runnable := 0
	for _, gate := range doc.Gates {
		if _, abandoned := doc.Abandoned[gate.ID]; abandoned {
			continue
		}
		live++
		if gate.Check == "" {
			add("warn", gate.ID, "manual-gate",
				"no CHECK, so this outcome is judged by hand and its evidence is only as good as the reader")
			if regexp.MustCompile(`\d`).MatchString(gate.Title) {
				add("warn", gate.ID, "unmeasured-number",
					"title states a number that nothing measures: %q", gate.Title)
			}
		} else {
			runnable++
			if fixedOutputRE.MatchString(gate.Check) {
				add("warn", gate.ID, "tautological-check",
					"CHECK looks like a fixed-output command: %q; use an oracle that observes the named outcome", gate.Check)
			}
			if approved != nil && !Approved(gate.Check, gate.Cwd, approved) {
				add("error", gate.ID, "unapproved-check",
					"CHECK %q with CWD %q matches no Build-entry preflight row in test-plan.md; the runner refuses it", gate.Check, gate.Cwd)
			}
		}
		if gate.Expect != "" {
			if weakExpect[strings.ToLower(strings.TrimSpace(gate.Expect))] {
				add("warn", gate.ID, "weak-expect",
					"EXPECT %q also appears in failure output; match a line only success can print", gate.Expect)
			}
			if expectation, err := CompileExpect(gate.Expect); err == nil && expectation.Kind == "regex" && expectation.PathLike {
				add("warn", gate.ID, "path-read-as-regex",
					"EXPECT %q looks like a literal path but is read as a regular expression, so its dots are wildcards", gate.Expect)
			}
		}
		if activityStartRE.MatchString(gate.Title) {
			add("warn", gate.ID, "activity-not-outcome",
				"names an activity, not an outcome a stranger could judge: %q", gate.Title)
		}
	}
	if live > 0 && runnable*2 < live {
		add("warn", "", "mostly-manual",
			"%d/%d gates are runnable; a mostly manual ledger is prose with checkboxes", runnable, live)
	}
	seen := map[string]string{}
	for _, gate := range doc.Gates {
		if _, abandoned := doc.Abandoned[gate.ID]; abandoned || gate.Check == "" {
			continue
		}
		key := normalizeCommand(gate.Check) + "\x00" + strings.Join(strings.Fields(gate.Expect), " ") + "\x00" + normalizeApprovedCwd(gate.Cwd)
		if first, ok := seen[key]; ok {
			add("warn", gate.ID, "duplicate-oracle",
				"same CHECK+EXPECT+CWD as %s: two gates running one oracle is one method counted twice, not broader coverage", first)
			continue
		}
		seen[key] = gate.ID
	}
	return findings
}

// LintCounts summarizes findings for exit-code decisions: errors always fail;
// warnings fail only under strict.
func LintCounts(findings []Finding) (errors, warnings int) {
	for _, finding := range findings {
		if finding.Level == "error" {
			errors++
		} else {
			warnings++
		}
	}
	return errors, warnings
}
