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
	Schema   string `json:"schema"`
	Command  string `json:"command"`
	OK       bool   `json:"ok"`
	ExitCode int    `json:"exitCode"`
	ReasonID string `json:"reason_id"`
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
			if env.Schema != "devrites-command/v1" {
				t.Fatalf("schema=%q\n%s", env.Schema, out)
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

func TestLifecycleGateJSONCarriesRuleOwnedReasonID(t *testing.T) {
	root := newWorkspace(t)
	for _, tc := range []struct {
		command string
		want    string
	}{
		{"readiness", "DRV-GATE-READINESS-MISSING"},
		{"seal", "DRV-GATE-SEAL-MISSING"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			out, _, code := runDevrites(t, root, tc.command, "auth-tokens", "--json")
			if code != 3 {
				t.Fatalf("%s exit=%d\n%s", tc.command, code, out)
			}
			var env jsonEnvelope
			decodeExactlyOneJSON(t, out, &env)
			if env.ReasonID != tc.want {
				t.Fatalf("%s reason_id=%q, want %q\n%s", tc.command, env.ReasonID, tc.want, out)
			}
		})
	}
}

func TestLifecycleGateAppendsRootRelativeV1Provenance(t *testing.T) {
	root := newWorkspace(t)
	_, _, _ = runDevrites(t, root, "readiness", "auth-tokens", "--json")
	raw, err := os.ReadFile(filepath.Join(root, "timeline.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), filepath.Dir(root)) {
		t.Fatalf("timeline retained an absolute project path:\n%s", raw)
	}
	var boundaries []string
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var event struct {
			Schema     string   `json:"schema"`
			Boundary   string   `json:"boundary"`
			RootSource string   `json:"root_source"`
			Workspace  string   `json:"workspace"`
			ReasonID   string   `json:"reason_id"`
			Evidence   []string `json:"evidence_paths"`
		}
		if json.Unmarshal([]byte(line), &event) != nil || event.Schema != "devrites-event/v1" {
			continue
		}
		boundaries = append(boundaries, event.Boundary)
		if event.RootSource != "DEVRITES_ROOT" {
			t.Fatalf("non-canonical root/workspace provenance: %+v", event)
		}
		if event.Boundary == "lifecycle-gate" && event.Workspace != ".devrites/features/auth-tokens" {
			t.Fatalf("gate workspace provenance: %+v", event)
		}
		if event.Boundary == "root-selection" && event.Workspace != "" {
			t.Fatalf("root selection invented an inactive workspace: %+v", event)
		}
		for _, path := range event.Evidence {
			if filepath.IsAbs(path) || strings.HasPrefix(path, "..") {
				t.Fatalf("non-relative evidence %q", path)
			}
		}
	}
	if strings.Join(boundaries, ",") != "root-selection,lifecycle-gate" {
		t.Fatalf("boundaries=%v\n%s", boundaries, raw)
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
		LexicalRoot     string `json:"lexicalRoot"`
		RootSelection   string `json:"rootSelection"`
		ActiveWorkspace string `json:"activeWorkspace"`
		Source          string `json:"source"`
		Git             struct {
			TopLevel       string `json:"topLevel"`
			LinkedWorktree bool   `json:"linkedWorktree"`
			Submodule      bool   `json:"submodule"`
		} `json:"git"`
		HostCommands struct {
			Claude string `json:"claude"`
			Codex  string `json:"codex"`
		} `json:"hostCommands"`
		Status []any `json:"status"`
	}
	decodeExactlyOneJSON(t, out, &got)
	wantRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Root != wantRoot || got.Project != filepath.Dir(wantRoot) {
		t.Fatalf("wrong roots: %+v", got)
	}
	if got.LexicalRoot != root || got.RootSelection != "DEVRITES_ROOT" {
		t.Fatalf("wrong root provenance: %+v", got)
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

func TestContextShowJSONCarriesCanonicalRootHazards(t *testing.T) {
	root := newWorkspace(t)
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte("ghost\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errOut, code := runDevrites(t, root, "context", "show", "--json")
	if code != 0 {
		t.Fatalf("context show exit=%d stderr=%s", code, errOut)
	}
	var got struct {
		RootSelection string `json:"rootSelection"`
		Status        []struct {
			ID          string `json:"id"`
			Remediation string `json:"remediation"`
		} `json:"status"`
	}
	decodeExactlyOneJSON(t, out, &got)
	if got.RootSelection != "DEVRITES_ROOT" || len(got.Status) != 1 ||
		got.Status[0].ID != "DRV-ACTIVE-STALE" || !strings.Contains(got.Status[0].Remediation, "rm -f") {
		t.Fatalf("context root status = %+v\n%s", got, out)
	}
}

func TestContextShowSeparatesRootAndWorkspaceSelection(t *testing.T) {
	root := newWorkspace(t)
	out, errOut, code := runDevrites(t, root, "context", "show", "--json")
	if code != 0 {
		t.Fatalf("context show exit=%d stderr=%s", code, errOut)
	}
	var got struct {
		RootSelection string `json:"rootSelection"`
		Source        string `json:"source"`
	}
	decodeExactlyOneJSON(t, out, &got)
	if got.RootSelection != "DEVRITES_ROOT" || got.Source != "none" {
		t.Fatalf("root/workspace selections are conflated: %+v", got)
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
