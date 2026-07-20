package workflow

import "strings"

// Command names the host-specific invocation forms for one DevRites workflow step.
// The engine must not bake Claude slash commands into its decisions: Claude Code
// uses /rite-*, while Codex uses $rite-* skills. Keep the verb as the canonical
// key and render host forms only at the edge.
type Command struct {
	Verb   string
	Claude string
	Codex  string
}

var phaseToVerb = map[string]string{
	"frame":    "frame",
	"spec":     "spec",
	"temper":   "temper",
	"define":   "define",
	"plan":     "define", // plan is the artifact state produced by rite-define.
	"vet":      "vet",
	"build":    "build",
	"converge": "converge",
	"prove":    "prove",
	"polish":   "polish",
	"review":   "review",
	"seal":     "seal",
	"ship":     "ship",
}

// ForVerb returns the Claude and Codex forms for a public rite verb. The empty
// verb returns an empty command so callers can omit suggestions safely.
func ForVerb(verb string) Command {
	verb = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(verb, "/"), "$"))
	verb = strings.TrimPrefix(verb, "rite-")
	if verb == "" {
		return Command{}
	}
	name := "rite-" + verb
	return Command{Verb: verb, Claude: "/" + name, Codex: "$" + name}
}

// ForAction extracts the first explicit rite invocation from a cursor action.
// Cursor values may append a short reason after the command.
func ForAction(action string) Command {
	for _, field := range strings.Fields(action) {
		field = strings.Trim(field, "`'\"()[]{}<>,.;:")
		if strings.HasPrefix(field, "/rite-") || strings.HasPrefix(field, "$rite-") {
			return ForVerb(field)
		}
	}
	return Command{}
}

// ForPhase returns the next public command for a workflow phase.
func ForPhase(phase string) Command {
	verb := phaseToVerb[strings.TrimSpace(strings.ToLower(phase))]
	return ForVerb(verb)
}

// Both renders a command in host-neutral prose for engine stderr/stdout messages.
func (c Command) Both() string {
	if c.Verb == "" {
		return ""
	}
	return c.Claude + " (Claude) / " + c.Codex + " (Codex)"
}

// ClaudeOrCodex returns the command for a known harness. Empty/unknown harnesses
// deliberately get the host-neutral rendering so CLI output remains safe when the
// engine is run outside a hook.
func (c Command) ClaudeOrCodex(harness string) string {
	switch strings.ToLower(strings.TrimSpace(harness)) {
	case "claude", "claude-code", "claudecode":
		return c.Claude
	case "codex":
		return c.Codex
	default:
		return c.Both()
	}
}
