package lib

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Executable acceptance criteria: a port of check-acceptance.sh. Grades the
// AC-### acceptance criteria in spec.md against seal.md: each spec
// criterion must be checked ([x]) in seal.md. Without ids it falls back to a
// count/unchecked heuristic. The spec-time mirror is SpecValidate (shape); this
// grades proof. Legacy [ACn] ids remain supported for existing workspaces.
//
// Exit codes: 0 every criterion proven · 1 gap (uncovered/unproven) · 2 usage ·
// 5 missing spec.md or seal.md.
var (
	acHeadingRe = regexp.MustCompile(`^##[[:space:]]+[Aa]cceptance [Cc]riteria`)
	h2Re        = regexp.MustCompile(`^##[[:space:]]`)
	acIDRe      = regexp.MustCompile(`\[AC[0-9]+\]|\bAC-[0-9]{3}\b`)
	checkedRe   = regexp.MustCompile(`^[[:space:]]*-[[:space:]]*\[[xX]\]`)
	bulletRe    = regexp.MustCompile(`^[[:space:]]*-[[:space:]]`)
	checkboxRe  = regexp.MustCompile(`^[[:space:]]*-[[:space:]]*\[[ xX]\]`)
	uncheckedRe = regexp.MustCompile(`^[[:space:]]*-[[:space:]]*\[[[:space:]]\]`)
)

// CheckAcceptance grades the acceptance criteria of the workspace dir `ws`. It
// writes to stdout/stderr and returns the process exit code.
func CheckAcceptance(ws string, stdout, stderr io.Writer) int {
	if ws == "" || !isDir(ws) {
		fmt.Fprintln(stderr, "usage: devrites-engine check-acceptance <workspace-dir>")
		return 2
	}
	spec := ws + "/spec.md"
	seal := ws + "/seal.md"
	if !isFile(spec) {
		fmt.Fprintf(stderr, "check-acceptance: no spec.md in %s\n", ws)
		return 5
	}
	if !isFile(seal) {
		fmt.Fprintf(stderr, "check-acceptance: no seal.md in %s: feature not sealed\n", ws)
		return 5
	}

	specBody, err := acSection(spec)
	if err != nil {
		fmt.Fprintf(stderr, "check-acceptance: %v\n", err)
		return 2
	}
	sealBody, err := acSection(seal)
	if err != nil {
		fmt.Fprintf(stderr, "check-acceptance: %v\n", err)
		return 2
	}

	if sectionEmpty(specBody) {
		fmt.Fprintln(stderr, "check-acceptance: spec.md has no \"## Acceptance criteria\" section: nothing to grade")
		return 1
	}

	specIDs := sortedUniqueIDs(specBody)

	if len(specIDs) > 0 {
		// ID mode: each spec AC id must be checked ([x]) in seal.md.
		checkedIDs := map[string]bool{}
		for _, line := range sealBody {
			if checkedRe.MatchString(line) {
				for _, id := range acIDRe.FindAllString(line, -1) {
					checkedIDs[id] = true
				}
			}
		}
		var missing []string
		for _, id := range specIDs {
			if !checkedIDs[id] {
				missing = append(missing, id)
			}
		}
		total := len(specIDs)
		if len(missing) > 0 {
			fmt.Fprintf(stderr, "check-acceptance: %d/%d criteria proven; UNPROVEN/UNCOVERED:%s (must be checked [x] in seal.md)\n",
				total-len(missing), total, " "+strings.Join(missing, " "))
			return 1
		}
		fmt.Fprintf(stdout, "check-acceptance: OK: all %d acceptance criteria (%s) proven + checked in seal.md\n",
			total, strings.Join(specIDs, " "))
		return 0
	}

	// Fallback (no ids): count spec criteria vs seal checked/unchecked items.
	specN := countMatching(specBody, bulletRe)
	sealTotal := countMatching(sealBody, checkboxRe)
	sealUnchecked := countMatching(sealBody, uncheckedRe)

	if sealTotal < specN {
		fmt.Fprintf(stderr, "check-acceptance: seal.md lists %d acceptance items but spec.md has %d criteria: %d dropped. (Tag criteria AC-### to grade by id.)\n",
			sealTotal, specN, specN-sealTotal)
		return 1
	}
	if sealUnchecked > 0 {
		fmt.Fprintf(stderr, "check-acceptance: %d acceptance criterion(s) unchecked in seal.md (unproven). (Tag criteria AC-### to grade by id.)\n", sealUnchecked)
		return 1
	}
	fmt.Fprintf(stdout, "check-acceptance: OK: %d acceptance items all checked in seal.md (no ids; add AC-### for id-level grading)\n", sealTotal)
	return 0
}

// acSection returns the body lines under an "## Acceptance Criteria" heading
// (case-insensitive on the words), up to the next "## " heading: mirroring the
// awk one-liner in check-acceptance.sh.
func acSection(file string) ([]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("read acceptance section: %w", err)
	}
	defer f.Close()

	var body []string
	in := false
	sc := newLineScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if acHeadingRe.MatchString(line) {
			in = true
			continue // the heading itself is not body (awk's `next`)
		}
		if h2Re.MatchString(line) {
			in = false
		}
		if in {
			body = append(body, line)
		}
	}
	return body, sc.Err()
}

// sectionEmpty reports whether the acceptance body is empty in the bash sense:
// awk emits each in-section line with a trailing newline, $() strips the trailing
// newlines, and `[ -z ]` tests the remainder. A body of only blank lines collapses
// to "", so a heading followed by blanks is "nothing to grade", not a fallback.
func sectionEmpty(lines []string) bool {
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") == ""
}

// sortedUniqueIDs extracts canonical AC-### and legacy [ACn] ids from the lines,
// then returns them sorted and de-duplicated.
func sortedUniqueIDs(lines []string) []string {
	set := map[string]bool{}
	for _, line := range lines {
		for _, id := range acIDRe.FindAllString(line, -1) {
			set[id] = true
		}
	}
	if len(set) == 0 {
		return nil
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids) // byte order == `sort` under LC_ALL=C
	return ids
}

func countMatching(lines []string, re *regexp.Regexp) int {
	n := 0
	for _, line := range lines {
		if re.MatchString(line) {
			n++
		}
	}
	return n
}
