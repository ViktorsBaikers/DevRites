package main_test

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type jsonEnvelope struct {
	Command  string `json:"command"`
	OK       bool   `json:"ok"`
	ExitCode int    `json:"exitCode"`
}

func TestJSONContractCommandsEmitExactlyOneEnvelope(t *testing.T) {
	root := newWorkspace(t)
	workspace := filepath.Join(root, "features", "auth-tokens")
	cases := []struct {
		name string
		args []string
	}{
		{"status", []string{"status", "auth-tokens", "--json"}},
		{"readiness", []string{"readiness", "auth-tokens", "--json"}},
		{"seal", []string{"seal", "auth-tokens", "--json"}},
		{"spec-validate", []string{"spec-validate", workspace, "--json"}},
		{"evidence-fresh", []string{"evidence-fresh", "auth-tokens", "--json"}},
		{"preamble", []string{"preamble", "auth-tokens", "--json"}},
		{"coverage", []string{"coverage", "auth-tokens", "--json"}},
		{"analyze", []string{"analyze", "auth-tokens", "--json"}},
		{"doctor", []string{"doctor", "--json"}},
		{"ledger-list", []string{"ledger", "list", "--json"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, _, code := runDevrites(t, root, tc.args...)
			var env jsonEnvelope
			decodeExactlyOneJSON(t, out, &env)
			if env.Command == "" {
				t.Fatalf("missing command in envelope: %s", out)
			}
			if env.ExitCode != code {
				t.Fatalf("envelope exitCode=%d process exit=%d\n%s", env.ExitCode, code, out)
			}
			if env.OK != (code == 0) {
				t.Fatalf("envelope ok=%v process exit=%d\n%s", env.OK, code, out)
			}
		})
	}
}

func TestContextShowJSONReportsActiveWorkspace(t *testing.T) {
	root := newWorkspace(t)
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte("auth-tokens\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errOut, code := runDevrites(t, root, "context", "show", "--json")
	if code != 0 {
		t.Fatalf("context show exit=%d stderr=%s", code, errOut)
	}
	var got struct {
		Root            string `json:"root"`
		Project         string `json:"project"`
		ActiveWorkspace string `json:"activeWorkspace"`
		Source          string `json:"source"`
		HostCommands    struct {
			Claude string `json:"claude"`
			Codex  string `json:"codex"`
		} `json:"hostCommands"`
		Status []any `json:"status"`
	}
	decodeExactlyOneJSON(t, out, &got)
	if got.Root != root || got.Project != filepath.Dir(root) {
		t.Fatalf("wrong roots: %+v", got)
	}
	if got.ActiveWorkspace != filepath.Join(".devrites", "features", "auth-tokens") || got.Source != "ACTIVE" {
		t.Fatalf("wrong active workspace: %+v", got)
	}
	if got.HostCommands.Claude != "/rite" || got.HostCommands.Codex != "$rite" {
		t.Fatalf("wrong host commands: %+v", got.HostCommands)
	}
	if got.Status == nil {
		t.Fatalf("status should be an empty array, not omitted: %s", out)
	}
}

func decodeExactlyOneJSON(t *testing.T, text string, v any) {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(text))
	if err := dec.Decode(v); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, text)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		t.Fatalf("stdout has trailing data after first JSON document: %v\n%s", err, text)
	}
}
