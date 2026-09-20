package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
)

// windows.go — the broken-windows check. Review prose flags a new TODO, stub, or
// skipped test as a finding; this gate makes the register countable instead: any
// deferral marker the change *introduces* must carry a recorded waiver row in the
// workspace's windows.md, or the check fails. A silent deferral is the bug.
//
//	devrites-engine check windows <slug> [--worktree|--staged|--base <ref>] [--cwd <dir>]
//
//	0  no unwaived markers    2  usage error    3  unwaived markers or blocked
//
// Only added lines count: a marker already on the baseline is pre-existing debt,
// never attributable to this change. Each windows.md row waives exactly one hit —
// one blanket row cannot cover a file's five new TODOs.
//
// windows.md row grammar (one waiver per line):
//
//	- <project-relative path> | <marker token> | <reason>
//
// A row waives one hit at that path carrying that marker; empty reasons do not
// waive. Rows never suppress coverage — they explain it.

const windowsUsage = `usage: devrites-engine check windows <slug> [--worktree|--staged|--base <ref>] [--cwd <dir>]

Reports deferral markers introduced by the change (added lines only) and requires
each to be waived by a row in .devrites/work/<slug>/windows.md:
  - <path> | <marker> | <reason>
Each row waives one hit at that path with that marker token.
Exit 0 clean or fully waived, 3 unwaived markers or blocked.`

// deferralTokenRe catches uppercase comment markers: TODO, FIXME, XXX, HACK.
var deferralTokenRe = regexp.MustCompile(`(?:^|[^A-Za-z0-9_])(TODO|FIXME|XXX|HACK)(?:[^A-Za-z0-9_]|$)`)

// deferralIdioms are substring signals for skipped tests and stub bodies.
var deferralIdioms = []string{
	"t.Skip(", "t.Skipf(",
	"test.skip(", "it.skip(", "describe.skip(", "context.skip(",
	"xit(", "xdescribe(", "xtest(", "xcontext(", "fit(", "fdescribe(",
	"pytest.mark.skip", "unittest.skip", "SkipTest",
	"@Disabled", "#[ignore]",
	"todo!(", "unimplemented!(",
	"NotImplementedError",
	"panic(\"TODO", "panic(\"unimplemented", "panic(\"not implemented",
	"throw new Error(\"unimplemented", "throw new Error(\"not implemented",
	"TODO: implement", "not yet implemented",
}

type windowHit struct {
	Path   string
	Line   int
	Marker string
	Text   string
}

type windowWaiver struct {
	Path   string
	Marker string
}

// windowsMarkers returns the deferral markers present on one added line. An
// idiom that already contains a token ("panic(\"TODO...") supersedes the bare
// token, so one line does not demand two waiver rows for a single marker.
func windowsMarkers(line string) []string {
	var idioms []string
	for _, idiom := range deferralIdioms {
		if strings.Contains(line, idiom) {
			idioms = append(idioms, idiom)
		}
	}
	var out []string
	for _, m := range deferralTokenRe.FindAllStringSubmatch(line, -1) {
		superseded := false
		for _, idiom := range idioms {
			if strings.Contains(idiom, m[1]) {
				superseded = true
				break
			}
		}
		if !superseded {
			out = append(out, m[1])
		}
	}
	return append(out, idioms...)
}

// addedLinesFromDiff parses a `git diff -U0` stream into added (path, line,
// text) rows and returns the set of paths the diff touched. Deleted files
// (+++ /dev/null) contribute nothing; binary hunks carry no + lines and pass
// through harmlessly.
func addedLinesFromDiff(diff string) ([]windowHit, map[string]bool) {
	var hits []windowHit
	diffPaths := map[string]bool{}
	var path string
	newLine := 0
	inHunk := false
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ "):
			p := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
			p = strings.Trim(p, `"`)
			if p == "/dev/null" {
				path = ""
				continue
			}
			path = strings.TrimPrefix(p, "b/")
			diffPaths[path] = true
		case strings.HasPrefix(line, "@@"):
			inHunk = false
			// @@ -a,b +c,d @@ — the +c side is the new-file start.
			if i := strings.Index(line, "+"); i >= 0 {
				rest := line[i+1:]
				end := strings.IndexAny(rest, " ,@")
				if end < 0 {
					end = len(rest)
				}
				if n, err := strconv.Atoi(strings.TrimSuffix(rest[:end], ",")); err == nil {
					newLine = n
					inHunk = path != ""
				}
			}
		case inHunk && strings.HasPrefix(line, "+"):
			text := line[1:]
			for _, marker := range windowsMarkers(text) {
				hits = append(hits, windowHit{Path: path, Line: newLine, Marker: marker, Text: strings.TrimSpace(text)})
			}
			newLine++
		case inHunk && strings.HasPrefix(line, "-"):
			// removed line: new-file line number does not advance
		case inHunk && strings.HasPrefix(line, `\`):
			// "\ No newline at end of file"
		default:
			newLine++
		}
	}
	return hits, diffPaths
}

// scanWholeFile treats every line of an untracked file as added.
func scanWholeFile(root, rel string, hits []windowHit) []windowHit {
	full := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(full)
	if err != nil || !info.Mode().IsRegular() || info.Size() > workspaceArtifactLimit {
		return hits
	}
	// #nosec G304 -- project-relative path from git status, stat'd regular file
	raw, err := os.ReadFile(full)
	if err != nil {
		return hits
	}
	if strings.ContainsRune(string(raw[:min(len(raw), 512)]), '\x00') {
		return hits // binary
	}
	for i, line := range strings.Split(string(raw), "\n") {
		for _, marker := range windowsMarkers(line) {
			hits = append(hits, windowHit{Path: rel, Line: i + 1, Marker: marker, Text: strings.TrimSpace(line)})
		}
	}
	return hits
}

// windowsHits collects introduced deferral markers for the requested diff mode.
func windowsHits(opts diffScopeOpts) ([]windowHit, error) {
	dir := opts.cwd
	if dir == "" {
		dir = opts.root
	}
	var hits []windowHit
	var diffArgs []string
	switch opts.mode {
	case "staged":
		diffArgs = []string{"diff", "--cached", "-U0", "--"}
	case "base":
		diffArgs = []string{"diff", opts.base, "-U0", "--"}
	default:
		diffArgs = []string{"diff", "HEAD", "-U0", "--"}
	}
	diff, err := runGitCommand(dir, nil, diffArgs...)
	var diffPaths map[string]bool
	switch {
	case err == nil:
		hits, diffPaths = addedLinesFromDiff(string(diff))
	case opts.mode == "worktree" && isUnbornHEAD(dir):
		// No commits yet: every changed file is wholly new.
		diffPaths = map[string]bool{}
	default:
		return nil, err
	}
	if opts.mode == "worktree" {
		// Whole files whose diff produced no hunks: staged additions land in
		// `git diff HEAD`, untracked files never do — scan those in full.
		// Status paths are repo-root-relative, so reads anchor at toplevel.
		repoRoot, rerr := runGitCommand(dir, nil, "rev-parse", "--show-toplevel")
		status, serr := gitStatusPaths(dir)
		if serr == nil && rerr == nil {
			repo := strings.TrimSpace(string(repoRoot))
			for _, p := range status {
				if diffPaths[p] || strings.HasPrefix(p, ".devrites/") {
					continue
				}
				hits = scanWholeFile(repo, p, hits)
			}
		}
	}
	var kept []windowHit
	for _, h := range hits {
		if !strings.HasPrefix(h.Path, ".devrites/") {
			kept = append(kept, h)
		}
	}
	return kept, nil
}

func isUnbornHEAD(dir string) bool {
	_, err := runGitCommand(dir, nil, "rev-parse", "--verify", "HEAD")
	return err != nil
}

// parseWindowsWaivers reads windows.md waiver rows: `- path | marker | reason`.
// Rows missing fields or with an empty reason are reported as warnings and
// waive nothing.
func parseWindowsWaivers(raw string) (map[windowWaiver]int, []string) {
	waivers := map[windowWaiver]int{}
	var warnings []string
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "*") {
			continue
		}
		body := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, "-"), "*"))
		parts := strings.Split(body, "|")
		if len(parts) < 3 {
			if strings.Contains(body, "|") {
				warnings = append(warnings, fmt.Sprintf("windows.md: malformed waiver row %q (want `path | marker | reason`)", body))
			}
			continue
		}
		w := windowWaiver{
			Path:   strings.Trim(strings.TrimSpace(parts[0]), "`'\""),
			Marker: strings.Trim(strings.TrimSpace(parts[1]), "`'\""),
		}
		if strings.TrimSpace(parts[2]) == "" {
			warnings = append(warnings, fmt.Sprintf("windows.md: waiver for %s %s has no reason — does not waive", w.Path, w.Marker))
			continue
		}
		waivers[w]++
	}
	return waivers, warnings
}

// parseWindowsArgs accepts the diff-scope mode flags plus a slug; the allowlist
// flags do not apply to marker scanning.
func parseWindowsArgs(root string, args []string, stderr io.Writer) (diffScopeOpts, int) {
	opts := diffScopeOpts{mode: "worktree", phase: "seal", root: root}
	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--phase":
			opts.phase = argAt(args, i+1)
			i++
		case "--staged":
			opts.mode = "staged"
		case "--worktree":
			opts.mode = "worktree"
		case "--base":
			opts.mode = "base"
			opts.base = argAt(args, i+1)
			i++
		case "--cwd":
			opts.cwd = argAt(args, i+1)
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "windows: unknown flag %q\n", args[i])
				return opts, 2
			}
			positional = append(positional, args[i])
		}
	}
	if len(positional) != 1 || opts.mode == "base" && opts.base == "" {
		fmt.Fprint(stderr, windowsUsage)
		return opts, 2
	}
	opts.slug = positional[0]
	return opts, 0
}

// RunCheckWindows reports deferral markers the change introduced and fails when
// any lacks a recorded waiver in the workspace's windows.md.
func RunCheckWindows(root string, args []string, stdout, stderr io.Writer) int {
	opts, code := parseWindowsArgs(root, args, stderr)
	if code != 0 {
		return code
	}
	hits, err := windowsHits(opts)
	if err != nil {
		fmt.Fprintf(stderr, "windows: %v\n", err)
		return 2
	}
	var waivers map[windowWaiver]int
	var warnings []string
	workspace, werr := devritespaths.ExistingFeatureDirChecked(root, opts.slug)
	if werr != nil {
		fmt.Fprintf(stderr, "windows: workspace unavailable: %v\n", werr)
		return 3
	}
	if raw, rerr := os.ReadFile(filepath.Join(workspace, "windows.md")); rerr == nil { // #nosec G304 -- workspace is validated and filename is fixed
		waivers, warnings = parseWindowsWaivers(string(raw))
	} else if !os.IsNotExist(rerr) {
		fmt.Fprintf(stderr, "windows: cannot read windows.md: %v\n", rerr)
		return 3
	}
	for _, w := range warnings {
		fmt.Fprintf(stdout, "windows: %s\n", w)
	}
	var unwaived, waived int
	for _, h := range hits {
		w := windowWaiver{Path: h.Path, Marker: h.Marker}
		if waivers[w] > 0 {
			waivers[w]--
			waived++
			fmt.Fprintf(stdout, "windows: waived: %s:%d %s\n", h.Path, h.Line, h.Marker)
			continue
		}
		unwaived++
		fmt.Fprintf(stdout, "windows: unwaived: %s:%d %s — %s\n", h.Path, h.Line, h.Marker, h.Text)
	}
	switch {
	case len(hits) == 0:
		fmt.Fprintln(stdout, "windows: ok (no deferral markers introduced)")
		recordMetric(root, opts.slug, opts.phase, "windows", "", 0)
		return 0
	case unwaived == 0:
		fmt.Fprintf(stdout, "windows: ok (%d marker(s), all waived in windows.md)\n", len(hits))
		recordMetric(root, opts.slug, opts.phase, "windows", "", int64(len(hits)))
		return 0
	default:
		fmt.Fprintf(stdout, "windows: BLOCKED: %d unwaived deferral marker(s) introduced — remove them or record waivers in windows.md\n", unwaived)
		recordMetric(root, opts.slug, opts.phase, "windows", "", int64(unwaived))
		return 3
	}
}
