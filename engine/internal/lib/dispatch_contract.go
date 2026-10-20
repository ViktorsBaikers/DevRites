package lib

import (
	"fmt"
	"regexp"
	"strings"
)

// maxContractBytes bounds a wright contract read; packets stay <= 64 KiB.
const maxContractBytes = 64 << 10

var (
	// writerSafeLineRE is the contract line that names the checks the wright
	// itself runs before returning.
	writerSafeLineRE = regexp.MustCompile(`(?im)^[\s>*#-]*writer-safe checks\s*:\s*\S`)
	// forbidExecutionRE catches contract text that leaves the wright blind:
	// "skip tests", "no tests/build", "do not run any checks", "run nothing".
	forbidExecutionRE = regexp.MustCompile(`(?i)\bskip(?:s|ping)?\s+(?:all\s+|the\s+)?(?:builds?|lint|tests?|testing|checks?|formatters?)\b` +
		`|\bno\s+(?:leaf\s+)?(?:tests?|test\s+commands?|builds?|lint|execution|checks?)\b` +
		`|\b(?:do\s+not|don't|never)\s+run\s+(?:any\s+)?(?:tests?|checks?|builds?|lint|commands?)\b` +
		`|\brun\s+nothing\b`)
	// allowedAfterRE marks a forbid match that is about not weakening tests,
	// not about skipping their execution ("no tests weakened or deleted").
	allowedAfterRE = regexp.MustCompile(`(?i)^[\s/,a-z-]{0,24}\b(?:weaken|delet|remov|disabl|skipp|xfail|loosen)`)
	// agentContractRE names an installed DevRites role contract on any host.
	agentContractRE = regexp.MustCompile(`[\w./~-]*agents/devrites-[a-z0-9-]+\.(?:md|toml)\b`)
	// readWordRE is a read instruction ("Read first", "read:"), not "read-only".
	readWordRE = regexp.MustCompile(`(?i)\bread(?:\s|:|$)`)
	// sectionHeadRE is a Markdown heading or a "Label:" line opening a list.
	sectionHeadRE = regexp.MustCompile(`^\s*(?:#{1,6}\s.*|[^:]*:[\s*]*)$`)
)

// lintContractRead refuses a packet line that tells the agent to read an
// installed role contract, on the line itself or under a "Read ..." heading or
// label: a named agent already has it as its system prompt. Canonical sources
// under pack/ are reviewable files, not the agent's own contract.
func lintContractRead(kind, path string, raw []byte) error {
	inRead := false
	for i, line := range strings.Split(string(raw), "\n") {
		if sectionHeadRE.MatchString(line) {
			inRead = readWordRE.MatchString(line)
		}
		if !inRead && !readWordRE.MatchString(line) {
			continue
		}
		for _, ref := range agentContractRE.FindAllString(line, -1) {
			if !strings.Contains(ref, "pack/") {
				return fmt.Errorf("%s %s:%d tells the agent to read a role contract (%q); a named agent already has its contract as its system prompt, and context --role names it in the bundle — drop the line",
					kind, path, i+1, ref)
			}
		}
	}
	return nil
}

// lintWrightContract refuses a build wright contract that forbids the wright's
// own writer-safe verification, names none, or tells the wright to read its
// installed role contract, so a blind round is rejected before launch rather
// than discovered after the wright returns.
func lintWrightContract(path string) error {
	if path == "" {
		return fmt.Errorf("a build wright wave needs --contract <file> holding the exact wright contract (with a `Writer-safe checks:` line)")
	}
	raw, err := readBoundedRegularFile(path, maxContractBytes)
	if err != nil {
		return fmt.Errorf("wright contract %s: %w", path, err)
	}
	for i, line := range strings.Split(string(raw), "\n") {
		for _, loc := range forbidExecutionRE.FindAllStringIndex(line, -1) {
			if allowedAfterRE.MatchString(line[loc[1]:]) {
				continue
			}
			return fmt.Errorf("wright contract %s:%d forbids writer-safe execution (%q); the wright runs syntax/type/lint, unit and self-check suites and a narrow local smoke itself — list them under `Writer-safe checks:` and keep only artifact-producing, consumptive, or full canonical gates root-owned",
				path, i+1, strings.TrimSpace(line[loc[0]:loc[1]]))
		}
	}
	if err := lintContractRead("wright contract", path, raw); err != nil {
		return err
	}
	if !writerSafeLineRE.Match(raw) {
		return fmt.Errorf("wright contract %s names no `Writer-safe checks:` line; list the syntax/type/lint, unit/self-check and narrow smoke commands the wright runs before returning", path)
	}
	return nil
}
