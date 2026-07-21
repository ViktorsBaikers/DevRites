// Package lib holds native Go implementations of the deterministic, zero-token
// workflow checks DevRites used to run as individual .sh helpers. Each exported
// function owns its logic outright — it does not shell out — and reproduces the
// legacy behavior (stdout + exit code) exactly, verified in the CLI test suite
// by golden assertions or, transitionally, a bash-parity comparison.
package lib

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	reqIDRe      = regexp.MustCompile(`REQ-[0-9]+`)
	acBareIDRe   = regexp.MustCompile(`AC-[0-9]+`)
)

// SpecValidate lints the structured grammar in a spec.md. arg is a workspace dir
// (its spec.md is linted) or a direct spec.md path. against, when non-empty, is a
// capability-ledger root (`.devrites/specs`) the spec's delta sections are
// cross-checked against (an ADDED requirement must not already exist in its
// capability's ledger; a MODIFIED/REMOVED one must). cwd relativizes the printed
// path, matching the bash ${spec#"$PWD"/}. It writes to stdout/stderr and returns
// the process exit code.
func SpecValidate(arg, against, cwd string, stdout, stderr io.Writer) int {
	spec, code, ok := resolveSpecPath("spec-validate", arg, stderr)
	if !ok {
		return code
	}

	reqs, scens, findings, err := lintSpec(spec)
	findings = append(findings, lintEdgeProhibitionTables(spec)...)
	if err != nil {
		fmt.Fprintf(stderr, "spec-validate: %v\n", err)
		return 2
	}

	rel := strings.TrimPrefix(spec, cwd+"/")

	if reqs == 0 && len(findings) == 0 {
		fmt.Fprintf(stdout, "spec-validate: %s uses the simple acceptance form (no \"### Requirement:\" blocks) — nothing to lint\n", rel)
		return 0
	}

	// Delta cross-check (opt-in via --against): a spec written as deltas against a
	// living ledger must classify each requirement correctly. Skipped for a flat
	// snapshot (no delta sections) — there is nothing to reconcile.
	if against != "" {
		findings = append(findings, checkAgainstLedger(spec, against, defaultCapability(arg))...)
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
func lintEdgeProhibitionTables(file string) []string {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	text := string(data)
	knownText := sectionBody(text, "Requirements") + "\n" + sectionBody(text, "Acceptance criteria")
	known := map[string]bool{}
	for _, id := range reqIDRe.FindAllString(knownText, -1) {
		known[id] = true
	}
	for _, id := range acBareIDRe.FindAllString(knownText, -1) {
		known[id] = true
	}
	var findings []string
	for _, row := range markdownTableRows(sectionBody(text, "Edge Coverage")) {
		if len(row) < 4 {
			continue
		}
		id, target, status, reason := row[0], row[1], strings.ToLower(row[3]), ""
		if len(row) > 4 {
			reason = strings.TrimSpace(row[4])
		}
		if !tableTargetKnown(target, known) && !(status == "dismissed" && reason != "") {
			findings = append(findings, fmt.Sprintf("ERROR:%s: Edge Coverage %q targets no known REQ/AC and is not dismissed with a reason", file, id))
		}
	}
	for _, row := range markdownTableRows(sectionBody(text, "Prohibitions (must-NOT)")) {
		if len(row) < 4 {
			continue
		}
		id, target, status, proof := row[0], row[1], strings.ToLower(row[2]), strings.TrimSpace(row[3])
		if !tableTargetKnown(target, known) {
			findings = append(findings, fmt.Sprintf("ERROR:%s: Prohibition %q targets no known REQ/AC", file, id))
		}
		if status == "resolved/test" && proof == "" {
			findings = append(findings, fmt.Sprintf("ERROR:%s: Prohibition %q is resolved/test but has no linked test/evidence", file, id))
		}
	}
	return findings
}

func sectionBody(text, heading string) string {
	start := -1
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.EqualFold(strings.TrimSpace(line), "## "+heading) {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return ""
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if strings.HasPrefix(t, "## ") && !strings.HasPrefix(t, "###") {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

func markdownTableRows(body string) [][]string {
	var rows [][]string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") || strings.Contains(line, "---") {
			continue
		}
		parts := strings.Split(strings.Trim(line, "|"), "|")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if len(parts) > 0 && (strings.EqualFold(parts[0], "Edge ID") || strings.EqualFold(parts[0], "Prohibition ID")) {
			continue
		}
		rows = append(rows, parts)
	}
	return rows
}

func tableTargetKnown(target string, known map[string]bool) bool {
	for id := range known {
		if strings.Contains(target, id) {
			return true
		}
	}
	return false
}

func lintSpec(file string) (reqCount, scenCount int, findings []string, err error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("read spec: %w", err)
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
		return 0, 0, nil, fmt.Errorf("scan %s: %w", file, err)
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

func resolveSpecPath(command, arg string, stderr io.Writer) (string, int, bool) {
	if arg == "" {
		fmt.Fprintf(stderr, "usage: devrites-engine %s <workspace-dir | spec.md path>\n", command)
		return "", 2, false
	}

	var spec string
	switch {
	case isDir(arg):
		spec = arg + "/spec.md" // string concat, not filepath.Join — preserve the exact path bytes
	case isFile(arg):
		spec = arg
	default:
		fmt.Fprintf(stderr, "%s: no such workspace or file: %s\n", command, arg)
		return "", 2, false
	}
	if !isFile(spec) {
		fmt.Fprintf(stderr, "%s: no spec.md at %s\n", command, spec)
		return "", 5, false
	}
	return spec, 0, true
}

// checkAgainstLedger cross-checks a delta spec's requirements against the
// capability ledger rooted at ledgerRoot (`.devrites/specs`). For each requirement
// carrying a delta kind it resolves the capability (the section tag, or defaultCap
// when untagged) and confirms presence matches kind: ADDED must be absent from that
// capability's ledger spec, MODIFIED/REMOVED must be present. Flat requirements (no
// delta kind) are ignored. An unseeded capability reads as an empty key set, so the
// first ADDED against a brand-new capability passes cleanly.
func checkAgainstLedger(spec, ledgerRoot, defaultCap string) []string {
	doc, err := ParseSpec(spec)
	if err != nil || !doc.HasDelta {
		return nil
	}
	cache := map[string]map[string]bool{}
	keysFor := func(capability string) map[string]bool {
		if k, ok := cache[capability]; ok {
			return k
		}
		k := requirementKeys(ledgerRoot + "/" + capability + "/spec.md")
		cache[capability] = k
		return k
	}
	var findings []string
	for _, r := range doc.Requirements {
		if r.Kind == "" {
			continue
		}
		capability := r.Capability
		if capability == "" {
			capability = defaultCap
		}
		exists := keysFor(capability)[r.Key]
		switch r.Kind {
		case DeltaAdded:
			if exists {
				findings = append(findings, fmt.Sprintf("ERROR:%s: Requirement %q (line %d) is marked ADDED but already exists in ledger capability %q — use MODIFIED, or drop the delta", spec, r.Name, r.HeaderLine, capability))
			}
		case DeltaModified, DeltaRemoved:
			if !exists {
				findings = append(findings, fmt.Sprintf("ERROR:%s: Requirement %q (line %d) is marked %s but is absent from ledger capability %q — use ADDED, or fix the capability tag", spec, r.Name, r.HeaderLine, strings.ToUpper(r.Kind), capability))
			}
		}
	}
	return findings
}

// defaultCapability derives the fallback capability for an untagged delta section:
// the workspace slug — the basename of the workspace dir, or of the spec.md's
// parent directory when a bare file path was given.
func defaultCapability(arg string) string {
	if isDir(arg) {
		return filepath.Base(arg)
	}
	return filepath.Base(filepath.Dir(arg))
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func isFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}
