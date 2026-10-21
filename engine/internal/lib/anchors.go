package lib

import (
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/devrites/devrites/internal/anchor"
	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/safepath"
)

// anchors.go: deterministic quote anchoring for one reviewer return.
// agents.md § Result admission has root anchor each Critical/Important row's
// quote at its cited file:line and its `breaks` quote in the spec or standard it
// names before the row counts as a finding. This check does that with the same
// matcher /overhaul admit uses, so no row reaches a repair round on a quote
// nobody can find.
//
//	devrites-engine check anchors <slug> <return-file> [--skills-root <dir>]
//
//	0  every Critical/Important row anchored (exact or relocated)
//	2  usage error, workspace or return unreadable
//	3  blocked: an Unverified or Deferred row, which never drives a repair round
//
// A row's site is the first `;`-separated location in its second cell
// (`path:line[-line]`, or a workspace .md file with no line); its quote is the
// first backticked span in its observed cell. Paths resolve in the workspace,
// then the project. Exact match at the cited lines anchors; a quote on exactly
// one other line relocates; otherwise the row is Unverified (anchor lost or
// ambiguous). The `breaks "<quote>"` cell must quote spec.md/decisions.md for a
// spec ID, or the named shipped standard (under --skills-root when a host passes
// it); with no quote the row is Deferred.

// AnchorsUsage is the help and usage-error text for `check anchors`.
const AnchorsUsage = "usage: devrites-engine check anchors <slug> <return-file> [--skills-root <dir>]"

var (
	anchorSiteRE     = regexp.MustCompile(`^(\S+?)(?::([0-9]+)(?::[0-9]+)?(?:[-–]([0-9]+))?)?$`)
	anchorSeverityRE = regexp.MustCompile(`(?i)^\W*(critical|important)\b`)
)

type anchorer struct{ root, project, workspace, skillsRoot string }

// RunCheckAnchors implements `devrites-engine check anchors`.
func RunCheckAnchors(root string, args []string, stdout, stderr io.Writer) int {
	var positional []string
	skillsRoot := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--skills-root" && i+1 < len(args) {
			skillsRoot = args[i+1]
			i++
			continue
		}
		positional = append(positional, args[i])
	}
	if len(positional) != 2 || strings.HasPrefix(positional[0], "-") || strings.HasPrefix(positional[1], "-") {
		fmt.Fprintln(stderr, AnchorsUsage)
		return 2
	}
	args = positional
	slug := args[0]
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "check anchors: %v\n", err)
		return 2
	}
	raw, err := readBoundedRegularFile(args[1], maxReturnBytes)
	if err != nil {
		fmt.Fprintf(stderr, "check anchors: %s: %v\n", args[1], err)
		return 2
	}
	// The digest lets root confirm it anchored on the candidate the reviewer saw.
	if digest, n, err := CandidateIdentity(root, slug); err == nil {
		fmt.Fprintf(stdout, "check anchors: candidate %s (%d files)\n", digest, n)
	} else {
		fmt.Fprintf(stdout, "check anchors: no candidate (%v); quotes read the live tree\n", err)
	}
	a := anchorer{root: root, project: projectDir(root), workspace: workspace, skillsRoot: skillsRoot}
	counts := map[string]int{}
	rows := 0
	for i, line := range strings.Split(string(raw), "\n") {
		// Fail closed on format drift: a bulleted row or a bold/colon severity is still a blocker row.
		row, ok := strings.CutPrefix(strings.TrimLeft(line, " \t-*•"), "Finding:")
		if !ok {
			continue
		}
		cells := splitFindingCells(row)
		if !anchorSeverityRE.MatchString(cells[0]) {
			continue
		}
		rows++
		status, detail := a.row(cells)
		counts[status]++
		switch claim := strings.TrimSpace(row); status {
		case "Unverified":
			fmt.Fprintf(stdout, "%d: Unverified: %s — %s\n", i+1, claim, detail)
		case "Deferred":
			fmt.Fprintf(stdout, "%d: Deferred: %s — no spec anchor\n", i+1, claim)
		default:
			fmt.Fprintf(stdout, "%d: %s %s\n", i+1, status, detail)
		}
	}
	summary := fmt.Sprintf("anchored=%d relocated=%d unverified=%d deferred=%d",
		counts["anchored"], counts["relocated"], counts["Unverified"], counts["Deferred"])
	recordMetricNote(root, slug, cursorPhase(root, slug), "anchors", "", summary)
	// A head that declares more blockers than there are Finding: rows means
	// rows were written as prose; zero parsed rows must not read as clean.
	if declared := declaredBlockers(string(raw)); declared > rows {
		fmt.Fprintf(stdout, "check anchors: BLOCKED: Counts declares %d Critical/Important but %d are Finding: rows; rewrite each as a Finding: row and rerun\n", declared, rows)
		return 3
	}
	if counts["Unverified"]+counts["Deferred"] > 0 {
		fmt.Fprintf(stdout, "check anchors: BLOCKED: %d Critical/Important rows, %s; Unverified and Deferred rows never drive a repair round\n", rows, summary)
		return 3
	}
	fmt.Fprintf(stdout, "check anchors: OK: %d Critical/Important rows, %s\n", rows, summary)
	return 0
}

// declaredBlockers sums the critical and important entries of the return's
// first Counts: line, or 0 when there is none.
func declaredBlockers(text string) int {
	for _, line := range strings.Split(text, "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "Counts:")
		if !ok {
			continue
		}
		total := 0
		for _, field := range strings.Fields(rest) {
			if m := countRE.FindStringSubmatch(field); m != nil && (m[1] == "critical" || m[1] == "important") {
				n, _ := strconv.Atoi(m[2])
				total += n
			}
		}
		return total
	}
	return 0
}

// row classifies one blocker row: anchored, relocated, Unverified or Deferred.
func (a anchorer) row(cells []string) (status, detail string) {
	if len(cells) < 3 {
		return "Unverified", "missing file:line"
	}
	status, detail = a.site(cells[1], cells[2])
	if status == "Unverified" {
		return status, detail
	}
	if bs, bd := a.breaks(cells[2:]); bs != "anchored" {
		return bs, bd
	}
	return status, detail
}

// site anchors the observed quote at the row's cited location.
func (a anchorer) site(location, observed string) (status, detail string) {
	cited := strings.TrimSpace(strings.Split(strings.ReplaceAll(location, "`", ""), ";")[0])
	cited, _, _ = strings.Cut(cited, "§")
	cited = strings.TrimSpace(cited)
	m := anchorSiteRE.FindStringSubmatch(cited)
	if m == nil || m[2] == "" && !strings.HasSuffix(m[1], ".md") {
		return "Unverified", "missing file:line"
	}
	_, rest, hasQuote := strings.Cut(observed, "`")
	quote, _, closed := strings.Cut(rest, "`")
	if !hasQuote || !closed || strings.TrimSpace(quote) == "" {
		return "Unverified", "missing quote"
	}
	lines, err := a.lines(m[1])
	if err != nil {
		return "Unverified", "anchor lost: " + err.Error()
	}
	if m[2] != "" {
		s, _ := strconv.Atoi(m[2])
		e := s
		if m[3] != "" {
			e, _ = strconv.Atoi(m[3])
		}
		if 1 <= s && s <= e && e <= len(lines) {
			switch anchor.At(lines, s, e, quote) {
			case anchor.Found:
				return "anchored", cited
			case anchor.TooShort:
				return "Unverified", fmt.Sprintf("anchor ambiguous: quote under %d characters; quote the whole line", anchor.MinRunes)
			}
		}
	}
	// shortcut: relocation searches single lines; a quote spanning lines anchors only at its cited range.
	switch hits := anchor.Find(lines, quote); {
	case len(hits) == 1 && m[2] == "":
		return "anchored", fmt.Sprintf("%s:%d", m[1], hits[0])
	case len(hits) == 1:
		return "relocated", fmt.Sprintf("%s -> %s:%d", cited, m[1], hits[0])
	case len(hits) == 0:
		return "Unverified", "anchor lost"
	}
	return "Unverified", "anchor ambiguous"
}

// breaks anchors the `breaks <anchor> "<quote>"` cell in the spec or standard it names.
func (a anchorer) breaks(cells []string) (status, detail string) {
	for _, cell := range cells {
		cited, ok := strings.CutPrefix(strings.TrimSpace(cell), "breaks ")
		if !ok {
			continue
		}
		open := strings.IndexAny(cited, "\"“")
		closing := strings.LastIndexAny(cited, "\"”")
		if open < 0 || closing <= open {
			return "Deferred", ""
		}
		_, size := utf8.DecodeRuneInString(cited[open:])
		name, quote := cited[:open], cited[open+size:closing]
		var paths []string
		switch base := mdName(name); base {
		case "":
			paths = []string{filepath.Join(a.workspace, "spec.md"), filepath.Join(a.workspace, "decisions.md")}
		case "spec.md", "decisions.md":
			paths = []string{filepath.Join(a.workspace, base)}
		default:
			if skills, err := resolveSkillsRoot(a.root, a.skillsRoot); err == nil {
				paths = []string{filepath.Join(skills, "devrites-lib", "reference", "standards", base)}
			}
		}
		var text []string
		for _, p := range paths {
			if raw, err := readBoundedRegularFile(p, workspaceArtifactLimit); err == nil {
				text = append(text, anchor.Lines(raw)...)
			}
		}
		if len(text) == 0 {
			return "Unverified", fmt.Sprintf("breaks anchor lost: %s is unreadable", strings.TrimSpace(name))
		}
		// One window over the whole text, so a quote wrapped across lines matches.
		switch anchor.At([]string{strings.Join(text, " ")}, 1, 1, quote) {
		case anchor.NotFound:
			return "Unverified", "breaks anchor lost"
		case anchor.TooShort:
			return "Unverified", fmt.Sprintf("breaks anchor ambiguous: quote under %d characters", anchor.MinRunes)
		}
		return "anchored", ""
	}
	return "Deferred", ""
}

// lines reads a cited file from the workspace, else the project, never outside them.
func (a anchorer) lines(rel string) ([]string, error) {
	for _, base := range []string{a.workspace, a.project} {
		full := rel
		if !filepath.IsAbs(rel) {
			full = filepath.Join(base, filepath.FromSlash(rel))
		}
		if !safepath.WithinResolved(full, base) {
			continue
		}
		if raw, err := readBoundedRegularFile(full, maxCandidateFileBytes); err == nil {
			return anchor.Lines(raw), nil
		}
	}
	return nil, fmt.Errorf("%s is not a file under the workspace or project", rel)
}

// mdName is the base name of the first .md file a breaks anchor names, or "".
func mdName(name string) string {
	for _, field := range strings.Fields(strings.ReplaceAll(name, "`", " ")) {
		if field = strings.TrimRight(field, ",;:"); strings.HasSuffix(field, ".md") {
			return filepath.Base(field)
		}
	}
	return ""
}

// splitFindingCells splits a Finding: row on "|" outside backticked code, so a
// quoted `a || b` stays one cell.
func splitFindingCells(row string) []string {
	var cells []string
	var b strings.Builder
	code := false
	for _, r := range row {
		switch {
		case r == '`':
			code = !code
		case r == '|' && !code:
			cells = append(cells, b.String())
			b.Reset()
			continue
		}
		b.WriteRune(r)
	}
	return append(cells, b.String())
}
