package lib

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/workflow"
)

// BuildReadiness decides whether a feature is ready to build from its recorded
// state plus the upstream readiness artifacts. An open HITL question, a
// blocked or unapproved plan, incomplete decision coverage, or an unvetted plan
// each stops the build with a distinct, actionable exit code rather than
// trusting the model to honour a prose checklist. It never mutates the workspace.
//
// The command is `build-readiness` (the plain `readiness` name is the separate
// completeness gate), but its messages keep the `readiness:` prefix. Stable exit
// reasons and remediation verbs are owned by readiness_contract.json.
func BuildReadiness(root string, args []string, stdout, stderr io.Writer) int {
	slug := slugOrActive(root, args)
	s := featureFile(root, slug, "state.md")
	if slug == "" || !isFile(s) {
		shown := slug
		if shown == "" {
			shown = "<unset>"
		}
		fmt.Fprintf(stderr, "readiness: no active workspace/state.md (slug=%s): run %s <feature>\n", shown, workflow.ForVerb("spec").Both())
		return readinessCode("workspace-missing")
	}

	data, _ := os.ReadFile(s)
	lines := splitLinesNoTrailing(data)

	status := readinessField(lines, "Status")
	switch status {
	case "awaiting_human":
		fmt.Fprintf(stderr, "readiness: STOP: Status: awaiting_human. Resume with %s <qid> \"<answer>\".\n", workflow.ForVerb("resolve").Both())
		return readinessCode("awaiting-human")
	case "blocked":
		fmt.Fprintf(stderr, "readiness: STOP: Status: blocked. Repair with %s repair (or unblock).\n", workflow.ForVerb("plan").Both())
		return readinessCode("plan-blocked")
	}

	// Lifecycle order matters: clarification owns written-spec decision gaps.
	// Check it before plan approval so a workspace missing both does not bounce
	// through define merely to be sent back to clarify.
	if err := validateDecisionCoverage(root, slug); err != nil {
		fmt.Fprintf(stderr, "readiness: STOP: decision coverage is not buildable (%v). Run %s.\n", err, workflow.ForVerb("clarify").Both())
		return readinessCode("coverage-not-clear")
	}
	if err := validateArtifactContract(root, slug, readinessContract.Coverage); err != nil {
		fmt.Fprintf(stderr, "readiness: STOP: planning artifacts need a semantic upgrade (%v). Run %s.\n", err, workflow.ForVerb("upgrade").Both())
		return readinessCode("upgrade-required")
	}

	approved := readinessField(lines, "Plan approved")
	if approved == "" || approved == "none" {
		fmt.Fprintf(stderr, "readiness: STOP: plan not approved (state.md has no \"Plan approved\"). Run %s.\n", workflow.ForVerb("define").Both())
		return readinessCode("plan-unapproved")
	}

	phase := strings.ToLower(strings.TrimSpace(readinessField(lines, "Phase")))
	if fields := strings.Fields(phase); len(fields) > 0 {
		phase = fields[0]
	}
	if phase != string(state.PhaseVet) && phase != string(state.PhaseBuild) {
		fmt.Fprintf(stderr, "readiness: STOP: plan is not in a vetted/build phase. Run %s.\n", workflow.ForVerb("vet").Both())
		return readinessCode("engineering-not-ready")
	}

	if err := validateEngineeringReadiness(root, slug); err != nil {
		fmt.Fprintf(stderr, "readiness: STOP: implementation readiness is not buildable (%v). Run %s.\n", err, workflow.ForVerb("vet").Both())
		return readinessCode("engineering-not-ready")
	}
	if err := validateArtifactContract(root, slug, readinessContract.TestPlan, readinessContract.Engineering); err != nil {
		fmt.Fprintf(stderr, "readiness: STOP: planning artifacts need a semantic upgrade (%v). Run %s.\n", err, workflow.ForVerb("upgrade").Both())
		return readinessCode("upgrade-required")
	}

	shownStatus := status
	if shownStatus == "" {
		shownStatus = "running"
	}
	fmt.Fprintf(stdout, "readiness: OK: plan approved %s, decision coverage CLEAR, implementation READY, status %s. Ready to build.\n", approved, shownStatus)
	return readinessCode("ready")
}

// fieldAnnotation trims a trailing " # comment" or " | none" annotation off a
// state.md field value.
var fieldAnnotation = regexp.MustCompile(`[[:space:]]*(#|\|).*$`)

// readinessField reads canonical cursor tables and legacy bullet fields, then
// trims the legacy trailing annotations accepted by the original command.
func readinessField(lines []string, key string) string {
	value, ok := state.CursorField(lines, key)
	if !ok {
		return ""
	}
	if m := fieldAnnotation.FindStringIndex(value); m != nil {
		value = value[:m[0]]
	}
	return strings.TrimRight(value, spaceChars)
}
