package state

import (
	"fmt"
	"strings"
)

// ClarifyIntent is the only transition family piloted through a pure next-state
// function. Other lifecycle mutations intentionally remain unchanged.
type ClarifyIntent string

const (
	ClarifyEnter   ClarifyIntent = "enter"
	ClarifyRestore ClarifyIntent = "restore"
)

// ClarifyCursor is the complete cursor surface owned by clarify-return.
type ClarifyCursor struct {
	Phase            Phase
	Status           string
	NextAction       string
	ReturnPhase      Phase
	ReturnNextAction string
	HasReturn        bool
}

// ClarifyTransitionInput carries the current typed cursor plus host-rendered
// defaults. NextClarifyCursor is pure: it does no file or environment I/O.
type ClarifyTransitionInput struct {
	Cursor                  ClarifyCursor
	ClarifyNextAction       string
	DefaultReturnNextAction string
}

// ClarifyFieldPolicy is emitted in the workflow manifest for documentation.
type ClarifyFieldPolicy struct {
	Field  string `json:"field"`
	Policy string `json:"policy"`
}

// ClarifyFieldPolicies declares which fields the pilot derives and preserves.
func ClarifyFieldPolicies() []ClarifyFieldPolicy {
	return []ClarifyFieldPolicy{
		{Field: CursorPhase, Policy: "derived"},
		{Field: CursorStatus, Policy: "derived"},
		{Field: CursorNextAction, Policy: "derived"},
		{Field: CursorReturnPhase, Policy: "derived"},
		{Field: CursorReturnNextAction, Policy: "curated when present; otherwise derived"},
		{Field: "all other state.md content", Policy: "curated and preserved byte-for-byte"},
	}
}

// ClarifyTransitionError reports an illegal typed transition.
type ClarifyTransitionError struct {
	Intent ClarifyIntent
	Reason string
}

func (e *ClarifyTransitionError) Error() string {
	return fmt.Sprintf("clarify %s: %s", e.Intent, e.Reason)
}

// NextClarifyCursor returns the full next clarify-owned cursor surface.
func NextClarifyCursor(intent ClarifyIntent, input ClarifyTransitionInput) (ClarifyCursor, error) {
	current := input.Cursor
	switch intent {
	case ClarifyEnter:
		if current.Phase == PhaseClarify {
			return current, nil
		}
		if !PhaseAfterClarify(current.Phase) {
			return ClarifyCursor{}, &ClarifyTransitionError{Intent: intent, Reason: fmt.Sprintf("phase %q is not a later-phase retrofit", current.Phase)}
		}
		returnNext := strings.TrimSpace(current.NextAction)
		if returnNext == "" {
			returnNext = input.DefaultReturnNextAction
		}
		return ClarifyCursor{
			Phase:            PhaseClarify,
			Status:           "running",
			NextAction:       input.ClarifyNextAction,
			ReturnPhase:      current.Phase,
			ReturnNextAction: returnNext,
			HasReturn:        true,
		}, nil
	case ClarifyRestore:
		if !current.HasReturn {
			return current, nil
		}
		if !PhaseAfterClarify(current.ReturnPhase) {
			return ClarifyCursor{}, &ClarifyTransitionError{Intent: intent, Reason: fmt.Sprintf("invalid return phase %q", current.ReturnPhase)}
		}
		next := strings.TrimSpace(current.ReturnNextAction)
		if next == "" {
			next = input.DefaultReturnNextAction
		}
		return ClarifyCursor{
			Phase:      current.ReturnPhase,
			Status:     "running",
			NextAction: next,
		}, nil
	default:
		return ClarifyCursor{}, &ClarifyTransitionError{Intent: intent, Reason: "unknown intent"}
	}
}

// ParseClarifyCursor reads the clarify-owned fields from either cursor format.
func ParseClarifyCursor(lines []string) (ClarifyCursor, error) {
	rawPhase, ok := CursorField(lines, CursorPhase)
	if !ok || strings.TrimSpace(rawPhase) == "" {
		return ClarifyCursor{}, fmt.Errorf("state.md has no phase")
	}
	phase, ok := firstKnownPhase(rawPhase)
	if !ok {
		return ClarifyCursor{}, fmt.Errorf("unknown phase %q", rawPhase)
	}
	current := ClarifyCursor{Phase: phase}
	current.Status, _ = CursorField(lines, CursorStatus)
	current.NextAction, _ = CursorField(lines, CursorNextAction)
	rawReturn, hasReturn := CursorField(lines, CursorReturnPhase)
	if hasReturn && strings.TrimSpace(rawReturn) != "" {
		returnPhase, ok := firstKnownPhase(rawReturn)
		if !ok {
			return ClarifyCursor{}, fmt.Errorf("invalid return phase %q", rawReturn)
		}
		current.ReturnPhase = returnPhase
		current.HasReturn = true
	}
	current.ReturnNextAction, _ = CursorField(lines, CursorReturnNextAction)
	return current, nil
}

func firstKnownPhase(raw string) (Phase, bool) {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(raw)))
	if len(fields) == 0 {
		return "", false
	}
	return PhaseForName(fields[0])
}

// ApplyClarifyCursor serializes only clarify-owned fields and preserves all
// unrelated Markdown through the existing presentation-preserving mutators.
func ApplyClarifyCursor(lines []string, cursor ClarifyCursor) []string {
	lines = UpsertCursorField(lines, CursorPhase, string(cursor.Phase))
	lines = UpsertCursorField(lines, CursorStatus, cursor.Status)
	lines = UpsertCursorField(lines, CursorNextAction, cursor.NextAction)
	if cursor.HasReturn {
		lines = UpsertCursorField(lines, CursorReturnPhase, string(cursor.ReturnPhase))
		lines = UpsertCursorField(lines, CursorReturnNextAction, cursor.ReturnNextAction)
		return lines
	}
	lines = DeleteCursorField(lines, CursorReturnPhase)
	return DeleteCursorField(lines, CursorReturnNextAction)
}

// PhaseAfterClarify reports whether clarify-return may retrofit this phase.
func PhaseAfterClarify(phase Phase) bool {
	switch phase {
	case PhaseTemper, PhaseDefine, PhasePlan, PhaseVet, PhaseBuild, PhaseConverge,
		PhaseProve, PhasePolish, PhaseReview, PhaseSeal, PhaseShip:
		return true
	default:
		return false
	}
}
