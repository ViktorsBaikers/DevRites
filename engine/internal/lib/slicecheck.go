package lib

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/state"
)

// slicecheck.go: pre-dispatch lint for one slice's wright contract. A wright
// dispatch is the most expensive step in a lifecycle; contract defects (missing
// allowlist, non-exact paths, AC references that do not exist in spec.md, no
// proof plan) surface only at return inspection — after the tokens are spent.
// This check fails before dispatch instead.
//
//	devrites-engine check slice <slug> <SLICE-ID>
//
//	0  contract holds    2  usage error    3  blocked: problems listed

var (
	sliceFieldRe     = regexp.MustCompile(`(?m)^([A-Za-z][A-Za-z /_-]*):\s*(.*)$`)
	sliceACRe        = regexp.MustCompile(`\bAC-\d+\b`)
	pathGlobCharRe   = regexp.MustCompile(`[*?\[\]{}]`)
	pathTraversalRe  = regexp.MustCompile(`(^|/)\.\.(/|$)`)
	devritesPathRe   = regexp.MustCompile(`(?i)^\.?devrites(/|$)`)
	windowsDrivePath = regexp.MustCompile(`^[A-Za-z]:[\\/]`)
)

// sliceFields extracts the single-line `Field: value` pairs of a slice block.
func sliceFields(block string) map[string]string {
	fields := map[string]string{}
	for _, m := range sliceFieldRe.FindAllStringSubmatch(block, -1) {
		key := strings.ToLower(strings.TrimSpace(m[1]))
		if _, seen := fields[key]; !seen {
			fields[key] = strings.TrimSpace(m[2])
		}
	}
	return fields
}

// fieldValue returns the first matching field value among aliases.
func fieldValue(fields map[string]string, aliases ...string) (string, bool) {
	for _, name := range aliases {
		if v, ok := fields[name]; ok {
			return v, true
		}
	}
	return "", false
}

// parsePathList splits a `;` or `,` separated project-relative path list.
func parsePathList(raw string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == ';' || r == ',' }) {
		p := strings.Trim(strings.TrimSpace(part), "`'\"")
		if p != "" && !strings.EqualFold(p, "none") {
			out = append(out, p)
		}
	}
	return out
}

// checkAllowlistPaths validates the exact-path contract: project-relative files
// only — no directories, globs, traversal, .devrites, or duplicates.
func checkAllowlistPaths(paths []string) []string {
	var problems []string
	seen := map[string]bool{}
	for _, p := range paths {
		switch {
		case strings.HasPrefix(p, "/") || windowsDrivePath.MatchString(p):
			problems = append(problems, fmt.Sprintf("allowlist path %q is not project-relative", p))
		case pathTraversalRe.MatchString(p):
			problems = append(problems, fmt.Sprintf("allowlist path %q escapes the project (..)", p))
		case pathGlobCharRe.MatchString(p):
			problems = append(problems, fmt.Sprintf("allowlist path %q uses a glob — exact paths only", p))
		case strings.HasSuffix(p, "/"):
			problems = append(problems, fmt.Sprintf("allowlist path %q is a directory — exact files only", p))
		case devritesPathRe.MatchString(p):
			problems = append(problems, fmt.Sprintf("allowlist path %q targets .devrites — wrights never write workflow state", p))
		case seen[p]:
			problems = append(problems, fmt.Sprintf("allowlist path %q is duplicated", p))
		}
		seen[p] = true
	}
	return problems
}

// RunCheckSlice lints one slice's dispatch contract.
func RunCheckSlice(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "usage: devrites-engine check slice <slug> <SLICE-ID>")
		return 2
	}
	slug, sliceID := args[0], args[1]
	if !state.ValidSliceID(sliceID) {
		fmt.Fprintf(stderr, "check slice: %q is not a SLICE-### id\n", sliceID)
		return 2
	}
	tasks, err := readWorkspaceArtifact(root, slug, "tasks.md")
	if err != nil {
		fmt.Fprintf(stderr, "check slice: %v\n", err)
		return 3
	}
	block, ok := state.ExtractTaskSlice(tasks, sliceID)
	if !ok {
		fmt.Fprintf(stderr, "check slice: %s not found in tasks.md\n", sliceID)
		return 3
	}

	var problems []string
	fields := sliceFields(block)

	if goal, ok := fieldValue(fields, "goal"); !ok || goal == "" {
		problems = append(problems, "Goal field missing or empty")
	}

	allowRaw, hasAllow := fieldValue(fields, "writer allowlist")
	var allowlist []string
	if !hasAllow || len(parsePathList(allowRaw)) == 0 {
		problems = append(problems, "Writer allowlist missing or empty — dispatch requires exact project-relative paths")
	} else {
		allowlist = parsePathList(allowRaw)
		problems = append(problems, checkAllowlistPaths(allowlist)...)
	}

	if touchedRaw, ok := fieldValue(fields, "files likely touched"); ok {
		allowed := map[string]bool{}
		for _, p := range allowlist {
			allowed[p] = true
		}
		for _, p := range parsePathList(touchedRaw) {
			if hasAllow && !allowed[p] {
				problems = append(problems, fmt.Sprintf("Files likely touched %q is outside the Writer allowlist", p))
			}
		}
	}

	acs := sliceACRe.FindAllString(block, -1)
	if len(acs) == 0 {
		problems = append(problems, "no AC-### coverage declared — every slice must satisfy acceptance criteria")
	} else {
		spec, specErr := readWorkspaceArtifact(root, slug, "spec.md")
		if specErr != nil {
			problems = append(problems, fmt.Sprintf("spec.md unavailable — cannot verify %d AC reference(s)", len(acs)))
		} else {
			specText := string(spec)
			seen := map[string]bool{}
			var unknown []string
			for _, ac := range acs {
				if seen[ac] {
					continue
				}
				seen[ac] = true
				if !strings.Contains(specText, ac) {
					unknown = append(unknown, ac)
				}
			}
			for _, ac := range unknown {
				problems = append(problems, fmt.Sprintf("%s is referenced but not declared in spec.md", ac))
			}
		}
	}

	if proof, ok := fieldValue(fields, "tests/proof", "tests", "proof"); !ok || proof == "" {
		problems = append(problems, "Tests/proof field missing or empty — no evidence plan for the slice")
	}

	if len(problems) > 0 {
		fmt.Fprintf(stdout, "check slice: %s BLOCKED\n", sliceID)
		for _, p := range problems {
			fmt.Fprintf(stdout, "  - %s\n", p)
		}
		return 3
	}
	fmt.Fprintf(stdout, "check slice: %s ok (paths=%d acs=%d)\n", sliceID, len(allowlist), len(acs))
	return 0
}
