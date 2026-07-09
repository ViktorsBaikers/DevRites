package lib

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/workflow"
)

// BuildReadiness decides whether a feature is ready to build, reading only its
// recorded state (state.md) — an open HITL question, a blocked plan, or an
// unapproved plan each stops the build with a distinct, actionable exit code
// rather than trusting the model to honour a prose checklist. It never mutates
// the workspace.
//
// The command is `build-readiness` (the plain `readiness` name is the separate
// completeness gate), but its messages keep the `readiness:` prefix. Exit codes:
//
//	0  ready to build
//	2  plan not approved   → /rite-define, then confirm the plan
//	3  awaiting a human     → /rite-resolve <qid> "<answer>"
//	4  blocked              → /rite-plan repair (or unblock)
//	5  no workspace/state.md → /rite-spec <feature>
func BuildReadiness(root string, args []string, stdout, stderr io.Writer) int {
	slug := slugOrActive(root, args)
	s := featureFile(root, slug, "state.md")
	if slug == "" || !isFile(s) {
		shown := slug
		if shown == "" {
			shown = "<unset>"
		}
		fmt.Fprintf(stderr, "readiness: no active workspace/state.md (slug=%s) — run %s <feature>\n", shown, workflow.ForVerb("spec").Both())
		return 5
	}

	data, _ := os.ReadFile(s)
	lines := splitLinesNoTrailing(data)

	status := readinessField(lines, "Status")
	switch status {
	case "awaiting_human":
		fmt.Fprintf(stderr, "readiness: STOP — Status: awaiting_human. Resume with %s <qid> \"<answer>\".\n", workflow.ForVerb("resolve").Both())
		return 3
	case "blocked":
		fmt.Fprintf(stderr, "readiness: STOP — Status: blocked. Repair with %s repair (or unblock).\n", workflow.ForVerb("plan").Both())
		return 4
	}

	approved := readinessField(lines, "Plan approved")
	if approved == "" || approved == "none" {
		fmt.Fprintf(stderr, "readiness: STOP — plan not approved (state.md has no \"Plan approved\"). Run %s.\n", workflow.ForVerb("define").Both())
		return 2
	}

	shownStatus := status
	if shownStatus == "" {
		shownStatus = "running"
	}
	fmt.Fprintf(stdout, "readiness: OK — plan approved %s, status %s. Ready to build.\n", approved, shownStatus)
	return 0
}

// fieldAnnotation trims a trailing " # comment" or " | none" annotation off a
// state.md field value.
var fieldAnnotation = regexp.MustCompile(`[[:space:]]*(#|\|).*$`)

// readinessField reads a `Key: value` field from state.md, tolerating a leading
// "- " bullet and trimming any trailing annotation and blanks. It returns the
// first match, or "" when the key is absent. A key may contain spaces (e.g.
// "Plan approved") — it is matched literally.
func readinessField(lines []string, key string) string {
	field := regexp.MustCompile(`^[[:space:]]*-?[[:space:]]*` + key + `:[[:space:]]*`)
	for _, line := range lines {
		loc := field.FindStringIndex(line)
		if loc == nil {
			continue
		}
		value := line[loc[1]:]
		if m := fieldAnnotation.FindStringIndex(value); m != nil {
			value = value[:m[0]]
		}
		return strings.TrimRight(value, spaceChars)
	}
	return ""
}
