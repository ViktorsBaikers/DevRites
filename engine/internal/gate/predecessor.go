package gate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/state"
)

const predecessorLabel = "Predecessor readiness: "
const consumedPlanningLabel = "Consumed planning inputs: "

var predecessorLineRE = regexp.MustCompile(`^Predecessor readiness: ([A-Za-z0-9][A-Za-z0-9._-]*) ([0-9a-f]{64})$`)

// predecessorProblems checks planning snapshots at phases requiring Vet readiness.
func predecessorProblems(root string, observation *state.WorkspaceObservation) []string {
	phase, err := observation.DeclaredPhase()
	if err != nil {
		return nil
	}
	policy, ok := state.PolicyFor(phase)
	if !ok || !phaseRequiresReadinessBinding(policy) {
		return nil
	}
	return ConsumedPlanningInputProblems(root, observation, false)
}

// ConsumedPlanningInputProblems validates the exact declared set of planning
// producers against Vet snapshots. Legacy plans may omit the declaration only
// when requireDeclaration is false. Runtime workspace dependencies are separate.
func ConsumedPlanningInputProblems(root string, observation *state.WorkspaceObservation, requireDeclaration bool) []string {
	declarations, err := planningContractLines(observation, "plan.md", "Consumed planning inputs")
	if err != nil {
		return []string{err.Error()}
	}
	explicit := len(declarations) > 0
	if !explicit && requireDeclaration {
		return []string{"plan.md missing Consumed planning inputs declaration; native Plan/Vet must record `Consumed planning inputs: none` or related workspace slugs"}
	}
	if len(declarations) > 1 {
		return []string{"plan.md holds more than one Consumed planning inputs declaration; native Plan/Vet must record exactly one"}
	}
	var producers []string
	declared := map[string]*state.WorkspaceObservation{}
	if explicit {
		value, ok := strings.CutPrefix(declarations[0], consumedPlanningLabel)
		if !ok || value == "" {
			return []string{"malformed Consumed planning inputs declaration; native Plan/Vet must record none or a unique comma-separated list of related workspace slugs"}
		}
		if value != "none" {
			for _, item := range strings.Split(value, ",") {
				slug := strings.TrimSpace(item)
				if slug == "none" || strings.ContainsAny(slug, " \t") || strings.HasPrefix(slug, "-") {
					return []string{fmt.Sprintf("malformed Consumed planning inputs workspace %q; repair plan.md and rerun /rite-vet", slug)}
				}
				if declared[slug] != nil {
					return []string{fmt.Sprintf("duplicate Consumed planning inputs workspace %s; repair plan.md and rerun /rite-vet", slug)}
				}
				if slug == observation.Slug() {
					return []string{"Consumed planning inputs cannot include the workspace itself; repair plan.md and rerun /rite-vet"}
				}
				producer, err := state.ObserveReferencedWorkspace(root, slug)
				if err != nil {
					return []string{fmt.Sprintf("malformed or unavailable Consumed planning inputs workspace %q (%v); repair plan.md and rerun /rite-vet", slug, err)}
				}
				declared[slug] = producer
				producers = append(producers, slug)
			}
		}
	}
	lines, err := planningContractLines(observation, "eng-review.md", "Predecessor readiness")
	if err != nil {
		return []string{err.Error()}
	}
	if !explicit && len(lines) > 1 {
		return []string{"eng-review.md holds more than one Predecessor readiness line; rerun /rite-vet and record exactly one"}
	}
	if explicit && len(producers) == 0 {
		if len(lines) != 1 || lines[0] != predecessorLabel+"none" {
			return []string{"eng-review.md must record exactly `Predecessor readiness: none` for Consumed planning inputs: none; rerun /rite-vet"}
		}
		return nil
	}
	bindings := map[string]string{}
	for _, line := range lines {
		match := predecessorLineRE.FindStringSubmatch(line)
		if match == nil {
			return []string{fmt.Sprintf("malformed %q; rerun /rite-vet and record `%s<slug> <64-hex digest>`", line, predecessorLabel)}
		}
		slug := match[1]
		if _, ok := bindings[slug]; ok {
			return []string{fmt.Sprintf("duplicate Predecessor readiness binding for %s; rerun /rite-vet", slug)}
		}
		if explicit && declared[slug] == nil {
			return []string{fmt.Sprintf("extra Predecessor readiness binding for undeclared producer %s; rerun /rite-vet", slug)}
		}
		bindings[slug] = match[2]
		if !explicit {
			producers = append(producers, slug)
		}
	}
	var anchor string
	if explicit && len(producers) > 0 {
		anchor, err = planningSequenceRoot(root, observation)
		if err != nil {
			return []string{fmt.Sprintf("Consumed planning inputs sequence cannot be verified: %v; repair sequence state and rerun /rite-vet", err)}
		}
	}
	for _, slug := range producers {
		recorded, ok := bindings[slug]
		if !ok {
			return []string{fmt.Sprintf("missing Predecessor readiness binding for consumed producer %s; rerun /rite-vet", slug)}
		}
		producer := declared[slug]
		if producer == nil {
			producer, err = state.ObserveReferencedWorkspace(root, slug)
			if err != nil {
				return []string{fmt.Sprintf("predecessor %s readiness cannot be computed (%v); rerun /rite-vet against its current plan", slug, err)}
			}
		}
		if explicit {
			producerRoot, err := planningSequenceRoot(root, producer)
			if err != nil || producerRoot != anchor {
				return []string{fmt.Sprintf("Consumed planning inputs workspace %s is unrelated or has invalid sequence ancestry; repair plan.md and rerun /rite-vet", slug)}
			}
		}
		current, err := readinessBindingFromObservation(producer)
		if err != nil {
			return []string{fmt.Sprintf("predecessor %s readiness cannot be computed (%v); rerun /rite-vet against its current plan", slug, err)}
		}
		if strings.TrimPrefix(current, readinessBindingLabel) != recorded {
			return []string{fmt.Sprintf("predecessor %s changed its planning inputs since this plan was vetted; rerun /rite-vet to re-check this plan against them and record `%s%s %s`",
				slug, predecessorLabel, slug, strings.TrimPrefix(current, readinessBindingLabel))}
		}
	}
	return nil
}

func planningContractLines(observation *state.WorkspaceObservation, name state.ArtifactPath, label string) ([]string, error) {
	fact, ok := observation.Fact(name)
	if !ok || fact.State() == state.ArtifactAbsent {
		return nil, nil
	}
	if fact.State() != state.ArtifactPresent && fact.State() != state.ArtifactEmpty {
		return nil, fmt.Errorf("%s cannot be inspected; repair artifact and rerun /rite-vet", name)
	}
	visible, err := markdowntext.Prose(fact.Bytes())
	if err != nil {
		return nil, fmt.Errorf("%s is malformed; repair artifact and rerun /rite-vet", name)
	}
	var lines []string
	for _, raw := range strings.Split(string(visible), "\n") {
		if line := strings.TrimSuffix(raw, "\r"); strings.Contains(line, label) {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func planningSequenceRoot(root string, observation *state.WorkspaceObservation) (string, error) {
	seen := map[string]bool{}
	for {
		slug := observation.Slug()
		if seen[slug] {
			return "", fmt.Errorf("sequence_parent cycle at %s", slug)
		}
		seen[slug] = true
		fact, ok := observation.Fact(state.LedgerFile)
		if !ok || fact.State() != state.ArtifactPresent {
			return "", fmt.Errorf("%s state.md unavailable", slug)
		}
		visible, err := markdowntext.Structural(fact.Bytes())
		if err != nil {
			return "", fmt.Errorf("%s state.md malformed", slug)
		}
		parent, found := "", false
		for _, line := range strings.Split(string(visible), "\n") {
			if value, ok := state.CursorField([]string{line}, state.CursorSequenceParent); ok {
				if found {
					return "", fmt.Errorf("%s duplicate sequence_parent", slug)
				}
				parent, found = value, true
			}
		}
		if parent == "" {
			return slug, nil
		}
		observation, err = state.ObserveReferencedWorkspace(root, parent)
		if err != nil {
			return "", err
		}
	}
}
