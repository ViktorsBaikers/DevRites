// Package gate implements deterministic completeness checks. Each gate checks
// only the files required for its transition when the command runs. Missing
// content returns a human-resolvable block instead of an error. Judgment gates
// remain advisory; only deterministic checks can block.
package gate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/state"
)

// Kind names a gate. Its string form is also the CLI subcommand.
type Kind string

const (
	// Readiness checks the sections required to LEAVE the feature's current
	// phase: the gate fired when advancing to the next phase.
	Readiness Kind = "readiness"
	// Seal checks the full seal-phase requirement set regardless of the
	// feature's current phase: the final completeness gate before shipping.
	Seal Kind = "seal"
)

// Result is a gate outcome. Blocked is true iff a required section is missing;
// Missing lists them in canonical order for an actionable message.
type Result struct {
	Kind          Kind
	Slug          string
	Phase         state.Phase
	Target        state.Phase     // the phase whose requirements were checked
	Missing       []state.Section // compact legacy view
	MissingFiles  []string        // authoritative per-file view
	StateProblems []string        // deterministic cross-file invariant failures
	Blocked       bool
	ReasonID      reason.ID
}

// Check runs a gate against feature <slug> under root. It reads workspace files
// directly instead of trusting a cache. Invalid requests and unreadable state
// return errors; missing required content returns a blocked Result.
func Check(kind Kind, root, slug string) (*Result, error) {
	f, err := state.LoadFeature(root, slug)
	if err != nil {
		return nil, fmt.Errorf("gate %s: %w", kind, err)
	}
	target := f.Phase
	if kind == Seal {
		target = state.PhaseSeal
	}
	missing := state.MissingFor(f, target)
	missingFiles := state.MissingWorkspaceFiles(f, target)
	blocked := len(missingFiles) > 0
	readinessStale := false
	var stateProblems []string
	if gates, awaitingHuman := openHumanGates(devritespaths.FeatureDir(root, slug)); len(gates) > 0 {
		problem := fmt.Sprintf("open %s human question(s) remain in questions.md", strings.Join(gates, "/"))
		if !awaitingHuman {
			problem += " but state.md is not awaiting_human"
		}
		stateProblems = append(stateProblems, problem)
		blocked = true
	}
	if len(missingFiles) == 0 && phaseRequiresReadinessBinding(target) {
		expected, bindingErr := verifyReadinessBinding(root, slug)
		if bindingErr != nil {
			readinessStale = true
			blocked = true
			if expected == "" {
				stateProblems = append(stateProblems, bindingErr.Error()+"; repair the input and rerun /rite-vet")
			} else {
				stateProblems = append(stateProblems, "readiness inputs are stale or the binding is invalid; rerun /rite-vet and record exactly one standalone line: "+expected)
			}
		}
	}
	result := &Result{
		Kind:          kind,
		Slug:          slug,
		Phase:         f.Phase,
		Target:        target,
		Missing:       missing,
		MissingFiles:  missingFiles,
		StateProblems: stateProblems,
		Blocked:       blocked,
	}
	result.ReasonID = ResultReasonID(kind, blocked)
	if readinessStale {
		result.ReasonID = reason.GateReadinessStale
	}
	return result, nil
}

// ResultReasonID returns the typed outcome owned by the lifecycle gate.
func ResultReasonID(kind Kind, blocked bool) reason.ID {
	switch kind {
	case Readiness:
		if blocked {
			return reason.GateReadinessMissing
		}
		return reason.GateReadinessPassed
	case Seal:
		if blocked {
			return reason.GateSealMissing
		}
		return reason.GateSealPassed
	default:
		return ""
	}
}

// Render returns stable, greppable output with a trailing newline. A blocked
// result names the missing files and the command to rerun.
func (r *Result) Render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "gate: %s\n", r.Kind)
	fmt.Fprintf(&b, "feature: %s\n", r.Slug)
	fmt.Fprintf(&b, "phase: %s\n", r.Phase)
	if !r.Blocked {
		b.WriteString("result: pass\n")
		fmt.Fprintf(&b, "reason: %s\n", r.ReasonID)
		return b.String()
	}
	missing := r.MissingFiles
	switch {
	case len(missing) == 0:
		b.WriteString("result: blocked (state invariant)\n")
	case r.Kind == Seal:
		fmt.Fprintf(&b, "result: blocked (missing to seal: %s)\n", strings.Join(missing, ", "))
	default:
		fmt.Fprintf(&b, "result: blocked (missing to leave %q: %s)\n", r.Phase, strings.Join(missing, ", "))
	}
	fmt.Fprintf(&b, "reason: %s\n", r.ReasonID)
	if len(missing) > 0 {
		fmt.Fprintf(&b, "next: add real content to %s\n", strings.Join(missing, ", "))
	}
	for _, problem := range r.StateProblems {
		fmt.Fprintf(&b, "invariant: %s\n", problem)
	}
	fmt.Fprintf(&b, "retry: devrites-engine check %s %s\n", r.Kind, r.Slug)
	return b.String()
}

const gateSpaceChars = " \t\n\v\f\r"

func openHumanGates(featureDir string) ([]string, bool) {
	qdata, err := os.ReadFile(filepath.Join(featureDir, "questions.md"))
	if err != nil {
		return nil, false
	}
	gates := openBlockingQuestionGates(qdata)
	if len(gates) == 0 {
		return nil, false
	}
	sdata, err := os.ReadFile(filepath.Join(featureDir, "state.md"))
	return gates, err == nil && stateAwaitingHuman(sdata)
}

func openBlockingQuestionGates(data []byte) []string {
	lines := splitLinesNoTrailing(data)
	seen := map[string]bool{}
	var gates []string
	inQ := false
	status, gate := "", ""
	tableOpen := false
	statusColumn, gateColumn := -1, -1
	add := func(questionStatus, questionGate string) {
		questionStatus = strings.ToLower(strings.TrimSpace(questionStatus))
		questionGate = strings.ToLower(strings.TrimSpace(questionGate))
		if questionStatus == "open" && (questionGate == "blocking" || questionGate == "validating" || questionGate == "escalating") && !seen[questionGate] {
			seen[questionGate] = true
			gates = append(gates, questionGate)
		}
	}
	finalize := func() {
		if inQ {
			add(status, gate)
		}
	}
	for _, line := range lines {
		if cells, ok := questionTableCells(line); ok {
			if !tableOpen {
				tableOpen = true
				statusColumn = tableColumn(cells, "status")
				gateColumn = tableColumn(cells, "gate")
			}
			if statusColumn >= 0 && statusColumn < len(cells) && gateColumn >= 0 && gateColumn < len(cells) {
				add(cells[statusColumn], cells[gateColumn])
			}
		} else {
			tableOpen = false
			statusColumn, gateColumn = -1, -1
		}
		switch {
		case strings.HasPrefix(strings.ToLower(line), "## q-"):
			finalize()
			inQ, status, gate = true, "", ""
		case inQ && strings.HasPrefix(line, "status:"):
			status = strings.TrimLeft(strings.TrimPrefix(line, "status:"), gateSpaceChars)
		case inQ && strings.HasPrefix(line, "gate:"):
			gate = strings.TrimLeft(strings.TrimPrefix(line, "gate:"), gateSpaceChars)
		case inQ && isHeadingLine(line):
			finalize()
			inQ = false
		}
	}
	finalize()
	return gates
}

func questionTableCells(line string) ([]string, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil, false
	}
	parts := strings.Split(strings.Trim(line, "|"), "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts, true
}

func tableColumn(cells []string, name string) int {
	for i, cell := range cells {
		if strings.EqualFold(cell, name) {
			return i
		}
	}
	return -1
}

func stateAwaitingHuman(data []byte) bool {
	status, _ := state.CursorField(splitLinesNoTrailing(data), state.CursorStatus)
	return status == "awaiting_human"
}

func isHeadingLine(line string) bool {
	if !strings.HasPrefix(line, "##") {
		return false
	}
	rest := line[2:]
	return rest != "" && strings.ContainsRune(gateSpaceChars, rune(rest[0]))
}

func splitLinesNoTrailing(data []byte) []string {
	s := string(data)
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
