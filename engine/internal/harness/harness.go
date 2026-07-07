// Package harness holds the thin per-harness adapters at the engine's edge.
//
// One binary serves both Claude Code and Codex. The shared gate/orientation
// logic lives in the middle (internal/orient, internal/gate); this package
// translates each harness's hook stdin schema and its exit/decision convention.
// The engine never calls a model — these adapters only shape stdin/stdout.
package harness

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Harness identifies which agent runtime a hook is speaking to.
type Harness string

const (
	Claude Harness = "claude"
	Codex  Harness = "codex"
)

// Parse validates a --harness flag value. An unknown harness is an error so a
// typo surfaces loudly rather than silently picking a default convention.
func Parse(s string) (Harness, error) {
	switch Harness(s) {
	case Claude, Codex:
		return Harness(s), nil
	case "":
		return "", fmt.Errorf("missing --harness (want %q or %q)", Claude, Codex)
	default:
		return "", fmt.Errorf("unknown harness %q (want %q or %q)", s, Claude, Codex)
	}
}

// SessionStartContext returns the hook stdout that injects text as SessionStart
// orientation.
//
// Claude Code and Codex both consume the same
// hookSpecificOutput.additionalContext envelope today (the shipped pack wires
// both harnesses' SessionStart hook to a script that emits exactly this shape),
// so the two branches are identical. The switch is the seam: when a harness's
// convention diverges, only its branch changes — callers stay unaware.
func (h Harness) SessionStartContext(text string) (string, error) {
	switch h {
	case Claude, Codex:
		env := sessionStartEnvelope{}
		env.HookSpecificOutput.HookEventName = "SessionStart"
		env.HookSpecificOutput.AdditionalContext = text
		b, err := json.Marshal(env)
		if err != nil {
			return "", err
		}
		return string(b), nil
	default:
		return "", fmt.Errorf("unsupported harness %q", h)
	}
}

type sessionStartEnvelope struct {
	HookSpecificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	} `json:"hookSpecificOutput"`
}

// StopBlock returns the hook stdout that blocks a Stop with a human-readable
// reason. Both harnesses honour the {"decision":"block","reason":...} convention
// the legacy devrites-stop-gate.sh emits; the seam again lets one diverge later.
func (h Harness) StopBlock(reason string) (string, error) {
	switch h {
	case Claude, Codex:
		b, err := json.Marshal(stopDecision{
			Decision: "block",
			Reason:   "DevRites stop-gate: " + reason + ". (devrites-stop-gate)",
		})
		if err != nil {
			return "", err
		}
		return string(b), nil
	default:
		return "", fmt.Errorf("unsupported harness %q", h)
	}
}

type stopDecision struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// StopInput is the subset of a Stop hook's stdin payload the gate reads. The
// loop guard (stop_hook_active) is the one field that keeps a blocking gate from
// wedging the session: once the harness reports it already re-entered the Stop
// hook, the gate must let the turn end.
type StopInput struct {
	StopHookActive bool `json:"stop_hook_active"`
}

// ParseStopInput decodes a Stop hook's stdin. An empty or malformed payload is
// not an error — it decodes to the zero value (StopHookActive false), so a
// harness that sends nothing simply gets a first-pass gate check. This keeps the
// hook fail-open: bad stdin never crashes it.
func (h Harness) ParseStopInput(r io.Reader) StopInput {
	var in StopInput
	data, err := io.ReadAll(r)
	if err != nil || len(strings.TrimSpace(string(data))) == 0 {
		return in
	}
	_ = json.Unmarshal(data, &in) // best-effort; zero value on failure
	return in
}
