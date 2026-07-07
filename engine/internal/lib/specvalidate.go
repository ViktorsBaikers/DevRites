// Package lib holds the Go ports of the devrites-lib shell helpers — the
// deterministic, zero-token workflow checks that used to be individual .sh
// scripts. Each exported function is byte-parity (stdout + exit code) with the
// bash script it replaces, proven by the parity oracle in the CLI test suite.
package lib

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// Spec-grammar validator — deterministic lint of the structured
// Requirement/Scenario acceptance grammar in a spec.md. A port of
// spec-validate.sh; see standards/spec-grammar.md for the grammar.
//
// Exit codes: 0 valid (or no structured requirements — nothing to lint) ·
// 1 grammar violation(s) · 2 usage · 5 missing spec.md.
var (
	reqHeaderRe  = regexp.MustCompile(`^[[:space:]]*###[[:space:]]+[Rr]equirement:`)
	reqPrefixRe  = regexp.MustCompile(`^[[:space:]]*###[[:space:]]+[Rr]equirement:[[:space:]]*`)
	scenHeaderRe = regexp.MustCompile(`^[[:space:]]*####[[:space:]]+[Ss]cenario:`)
	scenPrefixRe = regexp.MustCompile(`^[[:space:]]*####[[:space:]]+[Ss]cenario:[[:space:]]*`)
	trailWSRe    = regexp.MustCompile(`[[:space:]]+$`)
	shallRe      = regexp.MustCompile(`(^|[^A-Za-z])(SHALL|MUST)([^A-Za-z]|$)`)
	whenRe       = regexp.MustCompile(`(^|[^A-Za-z])WHEN([^A-Za-z]|$)`)
	thenRe       = regexp.MustCompile(`(^|[^A-Za-z])THEN([^A-Za-z]|$)`)
)

// SpecValidate lints the structured grammar in a spec.md. arg is a workspace dir
// (its spec.md is linted) or a direct spec.md path. cwd relativizes the printed
// path, matching the bash ${spec#"$PWD"/}. It writes to stdout/stderr and returns
// the process exit code.
func SpecValidate(arg, cwd string, stdout, stderr io.Writer) int {
	if arg == "" {
		fmt.Fprintln(stderr, "usage: devrites spec-validate <workspace-dir | spec.md path>")
		return 2
	}

	var spec string
	switch {
	case isDir(arg):
		spec = arg + "/spec.md" // string concat, not filepath.Join — preserve the exact path bytes
	case isFile(arg):
		spec = arg
	default:
		fmt.Fprintf(stderr, "spec-validate: no such workspace or file: %s\n", arg)
		return 2
	}
	if !isFile(spec) {
		fmt.Fprintf(stderr, "spec-validate: no spec.md at %s\n", spec)
		return 5
	}

	reqs, scens, findings, err := lintSpec(spec)
	if err != nil {
		fmt.Fprintf(stderr, "spec-validate: %v\n", err)
		return 2
	}

	rel := strings.TrimPrefix(spec, cwd+"/")

	if reqs == 0 {
		fmt.Fprintf(stdout, "spec-validate: %s uses the simple acceptance form (no \"### Requirement:\" blocks) — nothing to lint\n", rel)
		return 0
	}

	if len(findings) > 0 {
		for _, f := range findings {
			fmt.Fprintln(stderr, f)
		}
		fmt.Fprintf(stderr, "spec-validate: %s — %d requirement(s) / %d scenario(s); %d grammar error(s) (see standards/spec-grammar.md)\n",
			rel, reqs, scens, len(findings))
		return 1
	}

	fmt.Fprintf(stdout, "spec-validate: OK — %s: %d requirement(s) / %d scenario(s) well-formed (SHALL + WHEN/THEN, unique headers)\n",
		rel, reqs, scens)
	return 0
}

// lintSpec walks the spec once, mirroring the single awk pass in spec-validate.sh:
// it counts requirements and scenarios and collects ERROR findings in encounter
// order. `file` is used verbatim in the finding messages (the awk FILE var).
func lintSpec(file string) (reqCount, scenCount int, findings []string, err error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, 0, nil, err
	}
	defer f.Close()

	var (
		curReq        string // "" = not inside a requirement
		reqLine       int
		sawShall      bool
		scenInReq     int
		looseWhenThen bool

		inScenario bool
		scenName   string
		scenLine   int
		scenWhen   bool
		scenThen   bool

		seen = map[string]int{} // lowercased requirement name → first line
	)

	closeScenario := func() {
		if inScenario {
			if !scenWhen {
				findings = append(findings, fmt.Sprintf("ERROR:%s: Requirement \"%s\" > Scenario \"%s\" (line %d) has no WHEN line", file, curReq, scenName, scenLine))
			}
			if !scenThen {
				findings = append(findings, fmt.Sprintf("ERROR:%s: Requirement \"%s\" > Scenario \"%s\" (line %d) has no THEN line", file, curReq, scenName, scenLine))
			}
			inScenario = false
		}
	}
	closeRequirement := func() {
		closeScenario()
		if curReq != "" {
			if !sawShall {
				findings = append(findings, fmt.Sprintf("ERROR:%s: Requirement \"%s\" (line %d) has no SHALL/MUST statement", file, curReq, reqLine))
			}
			if scenInReq == 0 {
				if looseWhenThen {
					findings = append(findings, fmt.Sprintf("ERROR:%s: Requirement \"%s\" (line %d) has WHEN/THEN lines but no \"#### Scenario:\" header — wrap them in a scenario", file, curReq, reqLine))
				} else {
					findings = append(findings, fmt.Sprintf("ERROR:%s: Requirement \"%s\" (line %d) has no \"#### Scenario:\" block", file, curReq, reqLine))
				}
			}
		}
	}

	sc := newLineScanner(f)
	nr := 0
	for sc.Scan() {
		line := sc.Text()
		nr++
		switch {
		case reqHeaderRe.MatchString(line):
			closeRequirement()
			reqCount++
			name := trailWSRe.ReplaceAllString(reqPrefixRe.ReplaceAllString(line, ""), "")
			curReq = name
			reqLine = nr
			sawShall = shallRe.MatchString(line)
			scenInReq = 0
			looseWhenThen = false
			key := strings.ToLower(name)
			if first, dup := seen[key]; dup {
				findings = append(findings, fmt.Sprintf("ERROR:%s: duplicate Requirement header \"%s\" (lines %d and %d) — headers must be unique", file, name, first, nr))
			} else {
				seen[key] = nr
			}
		case scenHeaderRe.MatchString(line):
			closeScenario()
			if curReq == "" {
				findings = append(findings, fmt.Sprintf("ERROR:%s: Scenario at line %d is not under any \"### Requirement:\"", file, nr))
			}
			sName := trailWSRe.ReplaceAllString(scenPrefixRe.ReplaceAllString(line, ""), "")
			inScenario = true
			scenCount++
			scenInReq++
			scenName = sName
			scenLine = nr
			scenWhen = false
			scenThen = false
		default:
			if curReq != "" {
				if shallRe.MatchString(line) {
					sawShall = true
				}
				hasWhen := whenRe.MatchString(line)
				hasThen := thenRe.MatchString(line)
				if inScenario {
					if hasWhen {
						scenWhen = true
					}
					if hasThen {
						scenThen = true
					}
				} else if hasWhen || hasThen {
					looseWhenThen = true
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		return 0, 0, nil, err
	}
	closeRequirement()
	return reqCount, scenCount, findings, nil
}

// maxScanLine caps the line length the section scanners accept. It sits far above
// any realistic spec line, so lines are effectively unbounded (matching awk's
// no-limit record read) without pre-allocating the buffer.
const maxScanLine = 1 << 30

// newLineScanner returns a bufio.Scanner that splits on newlines (stripping a
// trailing CR like awk with RS="\n") and tolerates very long lines.
func newLineScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), maxScanLine)
	return sc
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func isFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}
