package harness

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseHarness(t *testing.T) {
	for _, tc := range []struct {
		name    string
		in      string
		want    Harness
		wantErr string
	}{
		{name: "claude", in: "claude", want: Claude},
		{name: "codex", in: "codex", want: Codex},
		{name: "missing", wantErr: "missing --harness"},
		{name: "unknown", in: "other", wantErr: `unknown harness "other"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.in)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Parse(%q) err=%v, want substring %q", tc.in, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected err: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("Parse(%q)=%q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestHarnessEnvelopes(t *testing.T) {
	session, err := Claude.SessionStartContext("hello <world>")
	if err != nil {
		t.Fatal(err)
	}
	var sessionEnv struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(session), &sessionEnv); err != nil {
		t.Fatal(err)
	}
	if sessionEnv.HookSpecificOutput.HookEventName != "SessionStart" ||
		sessionEnv.HookSpecificOutput.AdditionalContext != "hello <world>" {
		t.Fatalf("SessionStartContext envelope = %#v", sessionEnv.HookSpecificOutput)
	}

	stop, err := Codex.StopBlock("proof missing")
	if err != nil {
		t.Fatal(err)
	}
	var stopEnv struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(stop), &stopEnv); err != nil {
		t.Fatal(err)
	}
	if stopEnv.Decision != "block" || !strings.Contains(stopEnv.Reason, "proof missing") ||
		!strings.Contains(stopEnv.Reason, "devrites-stop-gate") {
		t.Fatalf("StopBlock envelope = %#v", stopEnv)
	}

	deny, err := Claude.PreToolDeny("readonly")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(deny, `\u003c`) {
		t.Fatalf("PreToolDeny HTML-escaped JSON: %s", deny)
	}
}

func TestHarnessParsersFailOpenAndFallbacks(t *testing.T) {
	if got := Claude.ParseStopInput(strings.NewReader(`{"stop_hook_active":true}`)); !got.StopHookActive {
		t.Fatalf("ParseStopInput did not read stop_hook_active")
	}
	if got := Claude.ParseStopInput(strings.NewReader(`not-json`)); got.StopHookActive {
		t.Fatalf("ParseStopInput malformed payload = %#v, want zero value", got)
	}

	pre := Codex.ParsePreToolInput(strings.NewReader(`{"tool_name":"Bash","tool_input":{"command":"go test ./..."},"agent_type":"reviewer"}`))
	if pre.ToolName != "Bash" || pre.Command != "go test ./..." || pre.AgentType != "reviewer" {
		t.Fatalf("ParsePreToolInput = %#v", pre)
	}

	guard := Claude.ParseGuardInput(strings.NewReader(`{"tool_name":"Edit","tool_input":{"path":"src/app.go"},"agent_id":"a1","tool_response":{"ok":true}}`))
	if guard.ToolName != "Edit" || guard.FilePath != "src/app.go" || guard.AgentID != "a1" || guard.ToolResponse != `{"ok":true}` {
		t.Fatalf("ParseGuardInput = %#v", guard)
	}

	if got := Codex.SubagentAgentType(strings.NewReader(`{"agentType":"executor"}`)); got != "executor" {
		t.Fatalf("SubagentAgentType fallback = %q, want executor", got)
	}
}
