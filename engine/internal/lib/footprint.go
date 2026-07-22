package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Footprint reports the fan-out footprint of a feature run: how many subagents
// were dispatched, of which kinds, across how many slices and minutes: from the
// append-only dispatch log at .devrites/work/<slug>/footprint.log. It never
// derives a token or dollar figure: those cannot be sourced truthfully, and
// inventing one would violate the proof thesis.
//
// base is the project directory that contains the .devrites tree. The stdout
// output and the exit code are the interface; stderr carries only usage errors.
//
//	log <slug> <kind> [label]  record one dispatch (exit 0; 2 on usage)
//	render <slug>              print the one-line footprint
//	roster <slug>              review-accounting gate: 0 complete · 1 always-on skipped · 3 unaccounted
func Footprint(root string, args []string, stdout, stderr io.Writer) int {
	const usage = "usage: devrites-engine footprint log|render|roster <slug> [...]"

	cmd, slug := "", ""
	if len(args) > 0 {
		cmd = args[0]
	}
	if len(args) > 1 {
		slug = args[1]
	}
	if cmd == "" || slug == "" {
		fmt.Fprintln(stderr, usage)
		return 2
	}

	dir := featureDir(root, slug)
	logPath := filepath.Join(dir, "footprint.log")
	switch cmd {
	case "log":
		return footprintLog(dir, logPath, args[2:], stderr)
	case "render":
		return footprintRender(logPath, stdout)
	case "roster":
		return footprintRoster(logPath, stdout)
	default:
		fmt.Fprintln(stderr, usage)
		return 2
	}
}

// footprintLog appends one dispatch record. Recording is best-effort: with no
// workspace, or an unwritable log, it silently succeeds: bookkeeping must never
// fail the orchestrator it instruments.
func footprintLog(dir, logPath string, rest []string, stderr io.Writer) int {
	kind, label := "", ""
	if len(rest) > 0 {
		kind = rest[0]
	}
	if len(rest) > 1 {
		label = rest[1]
	}
	if kind == "" {
		fmt.Fprintln(stderr, "usage: devrites-engine footprint log <slug> <kind> [label]")
		return 2
	}
	if !isDir(dir) {
		return 0
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0
	}
	defer f.Close()

	// Match footprint.sh's `printf '%s %s %s\n'` exactly: three fields always,
	// so an empty label still writes its trailing space. render/roster tokenize
	// on whitespace, so the extra field is harmless, but keeping the byte format
	// identical means the persisted log is parity-clean too.
	record := strconv.FormatInt(time.Now().Unix(), 10) + " " + kind + " " + label
	fmt.Fprintln(f, record)
	return 0
}

// footprintRender prints the one-line footprint: the subagent count with a
// per-kind breakdown, the slice count (one per wright dispatch), and the active
// span in whole minutes between the first and last record. A "skip" record: a
// conscious non-dispatch: is not a subagent and never counts.
func footprintRender(logPath string, stdout io.Writer) int {
	f, err := os.Open(logPath)
	if err != nil {
		fmt.Fprintln(stdout, "Footprint: n/a (no dispatch records)")
		return 0
	}
	defer f.Close()

	var total int
	byKind := map[string]int{}
	var first, last int64
	seen := false
	sc := newLineScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 || fields[1] == "skip" {
			continue
		}
		total++
		byKind[fields[1]]++
		epoch, _ := strconv.ParseInt(fields[0], 10, 64)
		if !seen || epoch < first {
			first = epoch
		}
		if epoch > last {
			last = epoch
		}
		seen = true
	}

	minutes := 0
	if last > first {
		minutes = int((last - first) / 60)
	}
	slices := byKind["wright"]
	fmt.Fprintf(stdout, "Footprint: %d subagent%s (%s) · %d slice%s · ~%dm active\n",
		total, plural(total), kindBreakdown(byKind), slices, plural(slices), minutes)
	return 0
}

// kindBreakdown renders the dispatched kinds in a fixed order: wright, reviewer,
// audit, doubt: each with its count, joined by " · ". Only kinds
// present appear; reviewers and audits pluralize, wright and doubt do not.
func kindBreakdown(byKind map[string]int) string {
	kinds := []struct {
		key       string
		pluralize bool
	}{
		{"wright", false},
		{"reviewer", true},
		{"audit", true},
		{"doubt", false},
	}
	var parts []string
	for _, k := range kinds {
		n := byKind[k.key]
		if n == 0 {
			continue
		}
		noun := k.key
		if k.pluralize {
			noun += plural(n)
		}
		parts = append(parts, fmt.Sprintf("%d %s", n, noun))
	}
	return strings.Join(parts, " · ")
}

// footprintRoster is the review fan-out accounting gate. Every reviewer on the
// roster must carry a recorded decision: dispatched, or a conscious skip. An
// unaccounted reviewer is the silent omission this catches. The roster mirrors
// devrites-lib/reference/parallel-dispatch.md, its single source of truth.
//
//	3  a reviewer has neither a dispatch nor a skip record
//	1  an always-on reviewer was skip-recorded (allowed; verify it is a carry-forward)
//	0  every reviewer accounted for
func footprintRoster(logPath string, stdout io.Writer) int {
	alwaysOn := []string{"spec-reviewer", "code-reviewer", "test-analyst"}
	conditional := []string{"frontend-reviewer", "security-auditor", "performance-reviewer", "devex-reviewer"}

	f, err := os.Open(logPath)
	if err != nil {
		fmt.Fprintln(stdout, "roster: n/a (no dispatch records: fan-out did not run)")
		return 0
	}
	defer f.Close()

	dispatched, skipped := map[string]bool{}, map[string]bool{}
	sc := newLineScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}
		name := strings.TrimPrefix(fields[2], "devrites-") // names log with or without the prefix
		switch fields[1] {
		case "reviewer":
			dispatched[name] = true
		case "skip":
			skipped[name] = true
		}
	}

	var unaccounted, alwaysOnSkipped []string
	for _, r := range alwaysOn {
		switch {
		case dispatched[r]:
			fmt.Fprintf(stdout, "  %s: dispatched\n", r)
		case skipped[r]:
			fmt.Fprintf(stdout, "  %s: skipped (always-on: verify carry-forward)\n", r)
			alwaysOnSkipped = append(alwaysOnSkipped, r)
		default:
			fmt.Fprintf(stdout, "  %s: UNACCOUNTED\n", r)
			unaccounted = append(unaccounted, r)
		}
	}
	for _, r := range conditional {
		switch {
		case dispatched[r]:
			fmt.Fprintf(stdout, "  %s: dispatched\n", r)
		case skipped[r]:
			fmt.Fprintf(stdout, "  %s: skipped\n", r)
		default:
			fmt.Fprintf(stdout, "  %s: UNACCOUNTED\n", r)
			unaccounted = append(unaccounted, r)
		}
	}

	switch {
	case len(unaccounted) > 0:
		fmt.Fprintf(stdout, "roster: UNACCOUNTED: %s\n", strings.Join(unaccounted, " "))
		return 3
	case len(alwaysOnSkipped) > 0:
		fmt.Fprintf(stdout, "roster: always-on skipped: %s\n", strings.Join(alwaysOnSkipped, " "))
		return 1
	default:
		fmt.Fprintln(stdout, "roster: complete (every reviewer accounted for)")
		return 0
	}
}

// plural returns the English plural suffix for a count: "" for one, "s" otherwise.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
