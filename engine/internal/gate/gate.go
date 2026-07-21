// Package gate implements DevRites' deterministic completeness gates.
//
// Enforcement is phase-relative, gate-scoped, and transition-fired: a gate
// checks only the sections required to make its transition, and only when it is
// run — never on every tool call. A block is a HITL pause (a structured
// "missing X" the human resolves and retries), never a crash. Judgment gates
// stay advisory; these deterministic gates are the ones allowed to block.
package gate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/orient"
	"github.com/devrites/devrites/internal/state"
)

// Kind names a gate. Its string form is also the CLI subcommand.
type Kind string

const (
	// Readiness checks the sections required to LEAVE the feature's current
	// phase — the gate fired when advancing to the next phase.
	Readiness Kind = "readiness"
	// Seal checks the full seal-phase requirement set regardless of the
	// feature's current phase — the final completeness gate before shipping.
	Seal Kind = "seal"
)

// Result is a gate outcome. Blocked is true iff a required section is missing;
// Missing lists them in canonical order for an actionable message.
type Result struct {
	Kind    Kind
	Slug    string
	Phase   state.Phase
	Target  state.Phase // the phase whose requirements were checked
	Missing []state.Section
	Blocked bool
}

// Check runs a gate against feature <slug> under root, reading the files
// directly (the files are always the source of truth, so a gate never trusts a
// possibly-stale cache). It returns an error only for a genuinely broken request
// (unknown slug, unreadable state) — a legitimately incomplete feature is a
// Result with Blocked set, not an error.
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
	return &Result{
		Kind:    kind,
		Slug:    slug,
		Phase:   f.Phase,
		Target:  target,
		Missing: missing,
		Blocked: len(missing) > 0,
	}, nil
}

// Render produces the deterministic, greppable gate output (with a trailing
// newline). A block names exactly the missing sections and the resolve step, so
// a human — or an AFK agent — knows precisely what to fix.
func (r *Result) Render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "gate: %s\n", r.Kind)
	fmt.Fprintf(&b, "feature: %s\n", r.Slug)
	fmt.Fprintf(&b, "phase: %s\n", r.Phase)
	if !r.Blocked {
		b.WriteString("result: pass\n")
		return b.String()
	}
	names := sectionNames(r.Missing)
	if r.Kind == Seal {
		fmt.Fprintf(&b, "result: blocked (missing to seal: %s)\n", strings.Join(names, ", "))
	} else {
		fmt.Fprintf(&b, "result: blocked (missing to leave %q: %s)\n", r.Phase, strings.Join(names, ", "))
	}
	fmt.Fprintf(&b, "next: add real content to %s, then re-run: devrites-engine %s %s\n",
		strings.Join(fileNames(r.Missing), ", "), r.Kind, r.Slug)
	return b.String()
}

// StopGate evaluates the stop-hook rest-point invariant for the active feature:
// a feature that CLAIMS completion (it has advanced to seal, ship, or done)
// must not have an empty proof section. This is a rest-point check, NOT
// whole-feature completeness — normal in-progress incompleteness never trips it,
// so a mid-build turn is never blocked.
//
// It returns a zero StopResult (Blocked false) — silent, no block — when there
// is no active feature or the workspace can't be read, keeping the stop hook
// fail-open.
func StopGate(root string) (StopResult, error) {
	slug, err := orient.ActiveSlug(root)
	if err != nil || slug == "" {
		return StopResult{}, nil
	}
	f, err := state.LoadFeature(root, slug)
	if err != nil {
		return StopResult{}, nil // fail-open: a broken workspace never wedges Stop
	}
	// Fail-on-red: the redwatch hook marks a known-red suite by writing .red. A turn
	// must not rest while it is set — evidence over confidence. This is a rest-point
	// invariant (a concrete, provable inconsistency), not whole-feature completeness.
	featureDir := devritespaths.FeatureDir(root, slug)
	if _, statErr := os.Stat(filepath.Join(featureDir, ".red")); statErr == nil {
		return StopResult{
			Slug:    slug,
			Blocked: true,
			Reason: fmt.Sprintf(
				"feature %q has tests/build RED (.red is set) — fix to green, or record the failure and next step, before stopping",
				slug),
		}, nil
	}
	if gates, blocked := unsurfacedHumanGates(featureDir); blocked {
		return StopResult{
			Slug:    slug,
			Blocked: true,
			Reason: fmt.Sprintf(
				"feature %q has open %s human question(s) in questions.md but state.md is not awaiting_human — surface the gate before stopping",
				slug, strings.Join(gates, "/")),
		}, nil
	}
	claimsDone := state.ShippablePhase(f.Phase)
	if claimsDone && !f.Present[state.SectionProof] {
		return StopResult{
			Slug:    slug,
			Blocked: true,
			Reason: fmt.Sprintf(
				"feature %q is at phase %q but proof.md is empty — record acceptance evidence, or move the phase back, before stopping",
				slug, f.Phase),
		}, nil
	}
	return StopResult{Slug: slug}, nil
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
	Slug    string
	Reason  string
	Blocked bool
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
