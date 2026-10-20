package gate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/state"
)

// predecessorLabel opens the line a lookahead Vet records in eng-review.md: the
// predecessor workspace this plan was written against and that predecessor's
// readiness binding digest at the time.
const predecessorLabel = "Predecessor readiness: "

var predecessorLineRE = regexp.MustCompile(`^Predecessor readiness: (\S+) ([0-9a-f]{64})$`)

// predecessorProblems blocks readiness when the predecessor a lookahead plan
// was written against has since changed its planning inputs, so this plan must
// be re-vetted against them. A workspace without the line is never checked.
func predecessorProblems(root string, observation *state.WorkspaceObservation) []string {
	phase, err := observation.DeclaredPhase()
	if err != nil {
		return nil
	}
	policy, ok := state.PolicyFor(phase)
	if !ok || !phaseRequiresReadinessBinding(policy) {
		return nil
	}
	fact, ok := observation.Fact("eng-review.md")
	if !ok || fact.State() != state.ArtifactPresent {
		return nil
	}
	visible, err := markdowntext.Structural(fact.Bytes())
	if err != nil {
		return nil
	}
	var lines []string
	for _, raw := range strings.Split(string(visible), "\n") {
		if line := strings.TrimSuffix(raw, "\r"); strings.Contains(line, predecessorLabel) {
			lines = append(lines, line)
		}
	}
	switch {
	case len(lines) == 0:
		return nil
	case len(lines) > 1:
		return []string{"eng-review.md holds more than one Predecessor readiness line; rerun /rite-vet and record exactly one"}
	}
	match := predecessorLineRE.FindStringSubmatch(lines[0])
	if match == nil || !devritespaths.ValidSlug(match[1]) {
		return []string{fmt.Sprintf("malformed %q; rerun /rite-vet and record `%s<slug> <64-hex digest>`", lines[0], predecessorLabel)}
	}
	slug, recorded := match[1], match[2]
	current, err := ReadinessBinding(root, slug)
	if err != nil {
		return []string{fmt.Sprintf("predecessor %s readiness cannot be computed (%v); rerun /rite-vet against its current plan", slug, err)}
	}
	if strings.TrimPrefix(current, readinessBindingLabel) != recorded {
		return []string{fmt.Sprintf("predecessor %s changed its planning inputs since this plan was vetted; rerun /rite-vet to re-check this plan against them and record `%s%s %s`",
			slug, predecessorLabel, slug, strings.TrimPrefix(current, readinessBindingLabel))}
	}
	return nil
}
