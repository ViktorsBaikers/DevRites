package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/workflow"
)

// spaceChars is the POSIX [[:space:]] set under the C locale — the collation the
// parity oracle pins (LC_ALL=C), so trimming matches the bash awk/sed exactly.
const spaceChars = " \t\n\v\f\r"

// featureDir is the per-feature directory under root. work/ is canonical;
// features/ remains readable as a compatibility alias.
func featureDir(root, slug string) string {
	return devritespaths.FeatureDir(root, slug)
}

// activeSlug reads the active-feature pointer at <root>/ACTIVE (trimmed), or ""
// when absent — the new-schema replacement for the old .devrites/ACTIVE lookup.
func activeSlug(root string) string {
	slug, err := devritespaths.ActiveSlug(root)
	if err != nil {
		return ""
	}
	return slug
}

// preambleArtifacts is the fixed order preamble.sh scans for present artifacts.
// It is NOT sorted — the display order is the workflow order, so the port keeps
// the literal list.
var preambleArtifacts = []string{
	"brief", "spec", "references", "strategy", "plan", "tasks", "eng-review",
	"test-plan", "questions", "decisions", "assumptions", "drift",
	"touched-files", "evidence", "browser-evidence", "design-brief",
	"polish-report", "review", "seal", "ship", "handoff",
}

// Preamble prints the active feature's workspace state — the orientation every
// workspace-operating rite-* skill runs first (step 0). Read-only. root is the
// .devrites directory; slug defaults to the active feature at <root>/ACTIVE.
// Always exits 0: orientation is never a hard dependency.
func Preamble(root string, args []string, stdout, stderr io.Writer) int {
	slug := ""
	if len(args) > 0 {
		slug = args[0]
	}
	if slug == "" {
		slug = activeSlug(root)
	}
	workDir := featureDir(root, slug)
	if slug == "" || !isDir(workDir) {
		fmt.Fprintf(stdout, "No active workspace. Run %s <feature> to start.\n", workflow.ForVerb("spec").Both())
		return 0
	}

	fmt.Fprintf(stdout, "## %s\n", slug)

	if state, err := os.ReadFile(filepath.Join(workDir, "state.md")); err == nil {
		fmt.Fprintln(stdout, "### state.md")
		// Emit the file verbatim, like `cat` — no added trailing newline.
		_, _ = stdout.Write(state)
	}

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "### artifacts present")
	for _, f := range preambleArtifacts {
		if isFile(filepath.Join(workDir, f+".md")) {
			fmt.Fprintf(stdout, "  ✓ %s.md\n", f)
		}
	}

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "### run mode")
	if afk, err := os.ReadFile(filepath.Join(root, "AFK")); err == nil {
		fmt.Fprintln(stdout, "  AFK (sentinel present)")
		// Surface optional config without parsing YAML: print every non-comment,
		// non-blank line verbatim, indented 4 spaces (grep -vE '^\s*(#|$)' | sed).
		for _, line := range splitLinesNoTrailing(afk) {
			trimmed := strings.TrimLeft(line, spaceChars)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			fmt.Fprintf(stdout, "    %s\n", line)
		}
	} else {
		fmt.Fprintln(stdout, "  HITL (no sentinel)")
	}

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "### questions")
	if q, err := os.ReadFile(filepath.Join(workDir, "questions.md")); err == nil {
		b, v, a, e := tallyOpenQuestions(q)
		fmt.Fprintf(stdout, "  open: %d blocking, %d validating, %d advisory, %d escalating\n", b, v, a, e)
	} else {
		fmt.Fprintln(stdout, "  (no questions.md yet)")
	}
	return 0
}

// tallyOpenQuestions counts open questions by gate, mirroring preamble.sh's awk.
// A question block opens at a "## q-" header and closes at the next "## " header
// (question or not) or EOF; a block counts iff its status is exactly "open".
func tallyOpenQuestions(data []byte) (blocking, validating, advisory, escalating int) {
	counts := map[string]int{}
	inQ := false
	status, gate := "", ""
	finalize := func() {
		if inQ && status == "open" {
			counts[gate]++
		}
	}
	for _, line := range splitLinesNoTrailing(data) {
		switch {
		case strings.HasPrefix(line, "## q-"):
			finalize()
			inQ, status, gate = true, "", ""
		case inQ && strings.HasPrefix(line, "status:"):
			status = strings.TrimLeft(strings.TrimPrefix(line, "status:"), spaceChars)
		case inQ && strings.HasPrefix(line, "gate:"):
			gate = strings.TrimLeft(strings.TrimPrefix(line, "gate:"), spaceChars)
		case inQ && isHeadingLine(line):
			// A non-question "## " header ends the block.
			if status == "open" {
				counts[gate]++
			}
			inQ = false
		}
	}
	finalize()
	return counts["blocking"], counts["validating"], counts["advisory"], counts["escalating"]
}

// isHeadingLine reports whether a line is a "## " heading (## followed by a space
// or tab), matching the awk /^##[[:space:]]/ that ends a question block.
func isHeadingLine(line string) bool {
	if !strings.HasPrefix(line, "##") {
		return false
	}
	rest := line[2:]
	return rest != "" && strings.ContainsRune(spaceChars, rune(rest[0]))
}

// splitLinesNoTrailing splits data into lines on '\n', dropping a single trailing
// newline's empty final element so a "\n"-terminated file yields N lines, not N+1
// — matching how awk/grep iterate records.
func splitLinesNoTrailing(data []byte) []string {
	s := string(data)
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
