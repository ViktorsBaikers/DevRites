// Package harness translates Claude Code and Codex hook payloads into the shared
// engine format. It handles stdin, stdout, and decision envelopes but never calls
// a model.
package harness

import (
	"bytes"
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

// Parse validates a --harness value. Unknown values return an error instead of
// choosing a default.
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
// both harnesses' SessionStart hook to a script that emits exactly this shape).
func (h Harness) SessionStartContext(text string) (string, error) {
	if h != Claude && h != Codex {
		return "", fmt.Errorf("unsupported harness %q", h)
	}
	env := sessionStartEnvelope{}
	env.HookSpecificOutput.HookEventName = "SessionStart"
	env.HookSpecificOutput.AdditionalContext = text
	b, err := json.Marshal(env)
	if err != nil {
		return "", fmt.Errorf("marshal SessionStart envelope: %w", err)
	}
	return string(b), nil
}

type sessionStartEnvelope struct {
	HookSpecificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	} `json:"hookSpecificOutput"`
}

// StopBlock returns the hook stdout that blocks a Stop with a human-readable
// reason. Both harnesses honour the {"decision":"block","reason":...} convention
// the legacy devrites-stop-gate.sh emits.
func (h Harness) StopBlock(reason string) (string, error) {
	if h != Claude && h != Codex {
		return "", fmt.Errorf("unsupported harness %q", h)
	}
	b, err := json.Marshal(stopDecision{
		Decision: "block",
		Reason:   "DevRites stop-gate: " + reason + ". (devrites-stop-gate)",
	})
	if err != nil {
		return "", fmt.Errorf("marshal stop decision: %w", err)
	}
	return string(b), nil
}

type stopDecision struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// StopInput contains the fields the gate reads from a Stop hook payload.
// stop_hook_active prevents a blocking gate from reentering indefinitely.
type StopInput struct {
	StopHookActive        bool `json:"stop_hook_active"`
	StopHookActivePresent bool `json:"-"`
}

// ParseStopInput decodes a Stop hook's stdin. An empty or malformed payload is
// not an error, but it must not be confused with an explicit first pass. Only a
// present boolean stop_hook_active is valid enough for a blocking gate.
func (h Harness) ParseStopInput(r io.Reader) StopInput {
	var payload struct {
		StopHookActive *bool `json:"stop_hook_active"`
	}
	data, err := io.ReadAll(r)
	if err != nil || len(strings.TrimSpace(string(data))) == 0 {
		return StopInput{}
	}
	if json.Unmarshal(data, &payload) != nil || payload.StopHookActive == nil {
		return StopInput{}
	}
	return StopInput{
		StopHookActive:        *payload.StopHookActive,
		StopHookActivePresent: true,
	}
}

// PreToolInput contains the tool, command, and optional agent identity read by
// the reviewer-readonly guard. Missing fields leave the guard silent.
type PreToolInput struct {
	ToolName  string
	Command   string
	AgentType string // the calling subagent's name; empty on the main thread
}

// ParsePreToolInput decodes a PreToolUse payload. Empty or malformed input
// returns the zero value. AgentType reads only `agent_type` to match
// devrites-reviewer-readonly.sh; SubagentAgentType handles the aliases needed by
// the subagent hook.
func (h Harness) ParsePreToolInput(r io.Reader) PreToolInput {
	var raw struct {
		ToolName  string `json:"tool_name"`
		ToolInput struct {
			Command string `json:"command"`
		} `json:"tool_input"`
		AgentType string `json:"agent_type"`
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return PreToolInput{}
	}
	_ = json.Unmarshal(data, &raw) // best-effort; zero value on failure

	return PreToolInput{
		ToolName:  raw.ToolName,
		Command:   raw.ToolInput.Command,
		AgentType: raw.AgentType,
	}
}

// SubagentAgentType reads the spawned agent name from agent_type, subagent_type,
// or agentType. This matches the fallback order in
// devrites-subagent-orient.sh. Invalid input returns an empty string.
func (h Harness) SubagentAgentType(r io.Reader) string {
	var raw struct {
		AgentType    string `json:"agent_type"`
		SubagentType string `json:"subagent_type"`
		AgentTypeAlt string `json:"agentType"`
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return ""
	}
	_ = json.Unmarshal(data, &raw) // best-effort; "" on failure
	return firstNonEmpty(raw.AgentType, raw.SubagentType, raw.AgentTypeAlt)
}

// PreToolDeny returns the hook stdout that denies a tool call with a
// human-readable reason. Both harnesses honour the hookSpecificOutput envelope
// the legacy PreToolUse guards emit; the seam lets one diverge later.
func (h Harness) PreToolDeny(reason string) (string, error) {
	return h.preToolDecision("deny", reason)
}

// PreToolAllow returns the hook stdout that auto-approves a tool call, bypassing
// the normal permission prompt.
func (h Harness) PreToolAllow(reason string) (string, error) {
	return h.preToolDecision("allow", reason)
}

func (h Harness) preToolDecision(decision, reason string) (string, error) {
	switch h {
	case Claude, Codex:
		var env preToolEnvelope
		env.HookSpecificOutput.HookEventName = "PreToolUse"
		env.HookSpecificOutput.PermissionDecision = decision
		env.HookSpecificOutput.PermissionDecisionReason = reason
		return marshalCompact(env)
	default:
		return "", fmt.Errorf("unsupported harness %q", h)
	}
}

type preToolEnvelope struct {
	HookSpecificOutput struct {
		HookEventName            string `json:"hookEventName"`
		PermissionDecision       string `json:"permissionDecision"`
		PermissionDecisionReason string `json:"permissionDecisionReason"`
	} `json:"hookSpecificOutput"`
}

// GuardInput contains the tool, command or target path, agent identity, and
// response text used by write guards. Claude sends agent_id, while Codex sends
// agent_type. Invalid input returns the zero value.
type GuardInput struct {
	ToolName     string
	Command      string
	FilePath     string
	AgentType    string
	AgentID      string
	ToolResponse string
}

// ParseGuardInput decodes a guard hook's stdin. tool_response may arrive as a
// string or a structured object; an object is rendered as its compact JSON, the
// way the shell hooks' `JSON.stringify` node parse does.
func (h Harness) ParseGuardInput(r io.Reader) GuardInput {
	var raw struct {
		ToolName  string `json:"tool_name"`
		ToolInput struct {
			Command  string `json:"command"`
			FilePath string `json:"file_path"`
			Path     string `json:"path"`
		} `json:"tool_input"`
		AgentType    string          `json:"agent_type"`
		AgentID      string          `json:"agent_id"`
		ToolResponse json.RawMessage `json:"tool_response"`
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return GuardInput{}
	}
	_ = json.Unmarshal(data, &raw)

	resp := ""
	if len(raw.ToolResponse) > 0 {
		if raw.ToolResponse[0] == '"' {
			_ = json.Unmarshal(raw.ToolResponse, &resp) // JSON string → its value
		} else {
			resp = string(raw.ToolResponse) // object/array/number → raw JSON text
		}
	}
	return GuardInput{
		ToolName:     raw.ToolName,
		Command:      raw.ToolInput.Command,
		FilePath:     firstNonEmpty(raw.ToolInput.FilePath, raw.ToolInput.Path),
		AgentType:    raw.AgentType,
		AgentID:      raw.AgentID,
		ToolResponse: resp,
	}
}

// PostToolContext returns the hook stdout that injects text as PostToolUse
// additionalContext (e.g. the fail-on-red notice).
func (h Harness) PostToolContext(text string) (string, error) {
	return h.additionalContext("PostToolUse", text)
}

// SubagentStartContext returns the hook stdout that injects text as
// SubagentStart orientation for a spawned subagent.
func (h Harness) SubagentStartContext(text string) (string, error) {
	return h.additionalContext("SubagentStart", text)
}

func (h Harness) additionalContext(event, text string) (string, error) {
	switch h {
	case Claude, Codex:
		var env additionalContextEnvelope
		env.HookSpecificOutput.HookEventName = event
		env.HookSpecificOutput.AdditionalContext = text
		return marshalCompact(env)
	default:
		return "", fmt.Errorf("unsupported harness %q", h)
	}
}

type additionalContextEnvelope struct {
	HookSpecificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	} `json:"hookSpecificOutput"`
}

// marshalCompact encodes v as compact JSON without Go's default HTML escaping.
// Legacy shell hooks use JSON.stringify, so matching its output preserves byte
// parity. The caller controls line endings.
func marshalCompact(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", fmt.Errorf("encode hook output: %w", err)
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
