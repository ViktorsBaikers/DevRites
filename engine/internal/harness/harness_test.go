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
	for _, test := range []struct {
		name    string
		payload string
		present bool
		active  bool
	}{
		{"explicit false", `{"stop_hook_active":false}`, true, false},
		{"explicit true", `{"stop_hook_active":true}`, true, true},
		{"empty", ``, false, false},
		{"malformed", `not-json`, false, false},
		{"missing", `{}`, false, false},
		{"null", `{"stop_hook_active":null}`, false, false},
		{"wrong type", `{"stop_hook_active":"false"}`, false, false},
	} {
		t.Run("stop/"+test.name, func(t *testing.T) {
			got := Claude.ParseStopInput(strings.NewReader(test.payload))
			if got.StopHookActivePresent != test.present || got.StopHookActive != test.active {
				t.Fatalf("ParseStopInput(%q) = %#v, want present=%v active=%v",
					test.payload, got, test.present, test.active)
			}
		})
	}

	pre := Codex.ParsePreToolInput(strings.NewReader(`{"tool_name":"Bash","tool_input":{"command":"go test ./..."},"agent_type":"reviewer"}`))
	if pre.ToolName != "Bash" || pre.Command != "go test ./..." || pre.AgentType != "reviewer" {
		t.Fatalf("ParsePreToolInput = %#v", pre)
	}

	guard := Claude.ParseGuardInput(strings.NewReader(`{"tool_name":"Edit","tool_input":{"path":"src/app.go"},"agent_id":"a1","tool_response":{"ok":true}}`))
	if guard.ToolName != "Edit" || guard.FilePath != "src/app.go" || guard.AgentID != "a1" || guard.ToolResponse != `{"ok":true}` {
		t.Fatalf("ParseGuardInput = %#v", guard)
	}
	guard = Codex.ParseGuardInput(strings.NewReader(`{"tool_name":"functions.exec_command","tool_input":{"cmd":"sha256sum /tmp/agent-packet.yaml"}}`))
	if guard.ToolName != "functions.exec_command" || guard.Command != "sha256sum /tmp/agent-packet.yaml" {
		t.Fatalf("ParseGuardInput Codex cmd = %#v", guard)
	}

	if got := Codex.SubagentAgentType(strings.NewReader(`{"agentType":"executor"}`)); got != "executor" {
		t.Fatalf("SubagentAgentType fallback = %q, want executor", got)
	}
}
