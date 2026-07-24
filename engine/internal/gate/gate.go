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
	"github.com/devrites/devrites/internal/orient"
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
	Kind         Kind
	Slug         string
	Phase        state.Phase
	Target       state.Phase     // the phase whose requirements were checked
	Missing      []state.Section // compact legacy view
	MissingFiles []string        // authoritative per-file view
	LegacyLayout bool
	Blocked      bool
	ReasonID     reason.ID
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
	var missingFiles []string
	blocked := len(missing) > 0
	if !f.LegacyLayout {
		missingFiles = state.MissingWorkspaceFiles(f, target)
		blocked = len(missingFiles) > 0
	}
	result := &Result{
		Kind:         kind,
		Slug:         slug,
		Phase:        f.Phase,
		Target:       target,
		Missing:      missing,
		MissingFiles: missingFiles,
		LegacyLayout: f.LegacyLayout,
		Blocked:      blocked,
	}
	result.ReasonID = ResultReasonID(kind, blocked)
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
		return b.String()
	}
	missing := r.MissingFiles
	if r.LegacyLayout {
		missing = make([]string, len(r.Missing))
		for i, section := range r.Missing {
			missing[i] = string(section)
		}
	}
	if r.Kind == Seal {
		fmt.Fprintf(&b, "result: blocked (missing to seal: %s)\n", strings.Join(missing, ", "))
	} else {
		fmt.Fprintf(&b, "result: blocked (missing to leave %q: %s)\n", r.Phase, strings.Join(missing, ", "))
	}
	fmt.Fprintf(&b, "next: add real content to %s, then re-run: devrites-engine %s %s\n",
		strings.Join(missing, ", "), r.Kind, r.Slug)
	return b.String()
}

// StopGate checks whether the active feature can end a turn in its current state.
// A feature in seal, ship, or done must have proof. Normal incompleteness during
// earlier phases does not block.
//
// Missing or unreadable workspace state returns an unblocked zero result.
func StopGate(root string) (StopResult, error) {
	slug, err := orient.ActiveSlug(root)
	if err != nil || slug == "" {
		return StopResult{}, nil
	}
	f, err := state.LoadFeature(root, slug)
	if err != nil {
		return StopResult{}, nil // Unreadable state does not block Stop.
	}
	// redwatch writes .red for a known failing suite. Stop blocks while that file
	// exists, independently of whole-feature completeness.
	featureDir := devritespaths.FeatureDir(root, slug)
	if _, statErr := os.Stat(filepath.Join(featureDir, ".red")); statErr == nil {
		return StopResult{
			Slug:          slug,
			Blocked:       true,
			ReasonID:      reason.HookStopRed,
			EvidenceFiles: []string{".red"},
			Reason: fmt.Sprintf(
				"feature %q has tests/build RED (.red is set): fix to green, or record the failure and next step, before stopping",
				slug),
		}, nil
	}
	if gates, blocked := unsurfacedHumanGates(featureDir); blocked {
		return StopResult{
			Slug:          slug,
			Blocked:       true,
			ReasonID:      reason.HookStopUnsurfacedHumanGate,
			EvidenceFiles: []string{"questions.md", "state.md"},
			Reason: fmt.Sprintf(
				"feature %q has open %s human question(s) in questions.md but state.md is not awaiting_human: surface the gate before stopping",
				slug, strings.Join(gates, "/")),
		}, nil
	}
	claimsDone := state.ShippablePhase(f.Phase)
	if claimsDone && !f.Present[state.SectionProof] {
		return StopResult{
			Slug:          slug,
			Blocked:       true,
			ReasonID:      reason.HookStopMissingProof,
			EvidenceFiles: []string{"evidence.md", "proof.md", "state.md"},
			Reason: fmt.Sprintf(
				"feature %q is at phase %q but proof.md is empty: record acceptance evidence, or move the phase back, before stopping",
				slug, f.Phase),
		}, nil
	}
	return StopResult{Slug: slug, ReasonID: reason.HookStopClear}, nil
}

const gateSpaceChars = " \t\n\v\f\r"

func unsurfacedHumanGates(featureDir string) ([]string, bool) {
	qdata, err := os.ReadFile(filepath.Join(featureDir, "questions.md"))
	if err != nil {
		return nil, false
	}
	gates := openBlockingQuestionGates(qdata)
	if len(gates) == 0 {
		return nil, false
	}
	sdata, err := os.ReadFile(filepath.Join(featureDir, "state.md"))
	if err == nil && stateAwaitingHuman(sdata) {
		return nil, false
	}
	return gates, true
}

func openBlockingQuestionGates(data []byte) []string {
	lines := splitLinesNoTrailing(data)
	seen := map[string]bool{}
	var gates []string
	inQ := false
	status, gate := "", ""
	finalize := func() {
		if inQ && status == "open" && (gate == "blocking" || gate == "validating" || gate == "escalating") && !seen[gate] {
			seen[gate] = true
			gates = append(gates, gate)
		}
	}
	for _, line := range lines {
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

// StopResult is a stop-gate evaluation. Slug names the active feature (empty when
// none); Blocked is true iff the rest-point invariant is violated; Reason is the
// actionable explanation when Blocked.
type StopResult struct {
	Slug          string
	Reason        string
	ReasonID      reason.ID
	EvidenceFiles []string
	Blocked       bool
}

func sectionNames(ss []state.Section) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = string(s)
	}
	return out
}

func fileNames(ss []state.Section) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = string(s) + ".md"
	}
	return out
}
