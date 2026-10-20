package gate

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/devrites/devrites/internal/acceptance"
	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/state"
)

var (
	// testDriverRE matches a test driver script: a file under a scripts/
	// directory whose basename carries a whole test/tests word
	// (scripts/dev-run-test.sh, scripts/run-tests.mjs). Word boundaries keep
	// latest-release.sh and attest.sh out, and unit test files outside
	// scripts/ are run by a driver rather than being one.
	testDriverRE = regexp.MustCompile(`(?:^|/)scripts/(?:[^/]+/)*(?:[^/]*[-_.])?tests?(?:[-_.][^/]*)?\.(?:sh|bash|py|mjs|js)$`)
	// writerAllowlistRE reads a tasks.md slice field the way the task parser
	// does: a column-1 `Writer allowlist:` line of `,`/`;` separated paths.
	writerAllowlistRE = regexp.MustCompile(`(?im)^writer allowlist:[ \t]*(.*)$`)
	// touchedPresentRE reads a `present` row of touched-files.md's candidate
	// manifest table (| State | File | Slice | Reason |).
	touchedPresentRE = regexp.MustCompile("(?m)^\\|[ \t]*present[ \t]*\\|[ \t]*`([^`]+)`[ \t]*\\|")
	// driverWaiverRE reads `Driver waiver: <path> — <DEC-id> <reason>` from
	// gates.md; the first DEC id on the line is the backing decision, and a
	// reason must follow it after a space or `:`/`,`/`.`/`;`.
	driverWaiverRE    = regexp.MustCompile("(?m)^Driver waiver:[ \t]*`?([^\\s`]+)`?.*?\\b(DEC-[A-Za-z0-9-]*[A-Za-z0-9])[ \t:,.;]+\\S")
	decisionHeadingRE = regexp.MustCompile(`(?m)^#{1,6}[ \t]+(DEC-[A-Za-z0-9-]*[A-Za-z0-9])`)
)

// testDriverProblems blocks a test driver script the feature plans to write (a
// tasks.md Writer allowlist, before proof) or touched (a touched-files.md
// present row, once proof is required) when no gates.md CHECK executes it and
// no `Driver waiver:` line cites a DEC id that decisions.md records. A touched
// driver no gate runs can be red at seal unnoticed. Waived drivers are returned
// separately so the gate output names every deferral.
func testDriverProblems(observation *state.WorkspaceObservation, policy state.PhasePolicy) (problems, waived []string) {
	if !slices.Contains(policy.RequiredArtifacts, state.ArtifactPath(acceptance.GatesFile)) {
		return nil, nil
	}
	gatesText := structuralFact(observation, state.ArtifactPath(acceptance.GatesFile))
	doc := acceptance.ParseLedger(gatesText)
	if len(doc.Errors) > 0 {
		return nil, nil // gateLedgerProblems already blocks a malformed ledger
	}

	source, re := "tasks.md", writerAllowlistRE
	if policy.ProofRequired {
		source, re = "touched-files.md", touchedPresentRE
	}
	var drivers []string
	for _, match := range re.FindAllStringSubmatch(structuralFact(observation, state.ArtifactPath(source)), -1) {
		for _, part := range strings.FieldsFunc(match[1], func(r rune) bool { return r == ',' || r == ';' }) {
			driver := strings.TrimPrefix(strings.Trim(strings.TrimSpace(part), "`'\""), "./")
			if testDriverRE.MatchString(driver) && !slices.Contains(drivers, driver) {
				drivers = append(drivers, driver)
			}
		}
	}
	if len(drivers) == 0 {
		return nil, nil
	}
	slices.Sort(drivers)

	waivers := map[string]string{}
	for _, match := range driverWaiverRE.FindAllStringSubmatch(gatesText, -1) {
		waivers[strings.TrimPrefix(match[1], "./")] = match[2]
	}
	decisions := map[string]bool{}
	for _, match := range decisionHeadingRE.FindAllStringSubmatch(structuralFact(observation, "decisions.md"), -1) {
		decisions[match[1]] = true
	}
	for _, driver := range drivers {
		if driverExecuted(driver, doc.Gates) {
			continue
		}
		decision, ok := waivers[driver]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf(
				"test driver %s is in %s but no gates.md CHECK executes it; add a CHECK or a DEC-backed `Driver waiver: %s — DEC-### <reason>` line",
				driver, source, driver))
		case !decisions[decision]:
			problems = append(problems, fmt.Sprintf(
				"test driver %s waiver cites %s, which has no decisions.md heading; record that decision or add a CHECK",
				driver, decision))
		default:
			waived = append(waived, fmt.Sprintf("unexecuted driver (waived by %s): %s", decision, driver))
		}
	}
	return problems, waived
}

// driverExecuted reports whether a CHECK names the driver by its
// repository-relative path, or by its path relative to the gate's CWD.
func driverExecuted(driver string, gates []*acceptance.Gate) bool {
	for _, gate := range gates {
		if gate.Check == "" {
			continue
		}
		if namesPath(gate.Check, driver) {
			return true
		}
		cwd := strings.TrimSuffix(strings.TrimPrefix(gate.Cwd, "./"), "/")
		if rel, ok := strings.CutPrefix(driver, cwd+"/"); ok && namesPath(gate.Check, rel) {
			return true
		}
	}
	return false
}

// namesPath reports whether check contains path as a whole path: at the start,
// or after whitespace, a quote, '=' or a leading "./" or "../", so
// tools/scripts/test.sh never satisfies scripts/test.sh.
func namesPath(check, path string) bool {
	for from := 0; ; {
		i := strings.Index(check[from:], path)
		if i < 0 {
			return false
		}
		at := from + i
		if at == 0 || strings.ContainsRune(" \t'\"=", rune(check[at-1])) || strings.HasSuffix(check[:at], "./") {
			return true
		}
		from = at + 1
	}
}

// structuralFact returns a present artifact's text with fenced examples
// masked, or "" when it is absent or unreadable.
func structuralFact(observation *state.WorkspaceObservation, path state.ArtifactPath) string {
	fact, ok := observation.Fact(path)
	if !ok || fact.State() != state.ArtifactPresent {
		return ""
	}
	structural, err := markdowntext.Structural(fact.Bytes())
	if err != nil {
		return ""
	}
	return string(structural)
}
