package lib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func wrightContractPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "contract.md")
	body := "# Goal\nBuild SLICE-001.\nWriter-safe checks: sh -n scripts/run.sh; go test ./pkg/...; scripts/run-test.sh --self-check\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLintWrightContract(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"names checks", "Writer-safe checks: go test ./...\nNo tests weakened or deleted.\nNever skip failure oracles.\n", ""},
		{"skip tests mid-flight", "Writer-safe checks: go test ./...\nSkip build/lint/tests/formatters mid-flight; root runs once after return.\n", "forbids writer-safe execution"},
		{"no test commands", "- Writer-safe checks: sh -n x.sh\nNo agents, no Git, no Docker/test commands.\n", ""},
		{"no tests run", "Writer-safe checks: go vet\nNo tests run by you.\n", "forbids writer-safe execution"},
		{"run nothing", "Writer-safe checks: go vet\nRun nothing; the root proves it.\n", "forbids writer-safe execution"},
		{"do not run any tests", "Writer-safe checks: go vet\nDo not run any tests.\n", "forbids writer-safe execution"},
		{"missing line", "# Goal\nImplement it.\n", "names no `Writer-safe checks:`"},
		{"empty line", "Writer-safe checks:\n", "names no `Writer-safe checks:`"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "c.md")
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			err := lintWrightContract(path)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("want pass, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
		})
	}
	if err := lintWrightContract(""); err == nil || !strings.Contains(err.Error(), "needs --contract") {
		t.Fatalf("missing flag: %v", err)
	}
}

func TestDispatchOpenRefusesBlindWrightContract(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	blind := filepath.Join(t.TempDir(), "blind.md")
	if err := os.WriteFile(blind, []byte("Writer-safe checks: none\nSkip tests mid-flight.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out := runDispatchT(t, root, "open", "--phase", "build", "--wave", "b1", "--role", "slice-wright", "--slice", "SLICE-001"); code != 3 || !strings.Contains(out, "needs --contract") {
		t.Fatalf("missing contract: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "open", "--phase", "build", "--wave", "b1", "--role", "slice-wright", "--slice", "SLICE-001", "--contract", blind); code != 3 || !strings.Contains(out, "forbids writer-safe execution") {
		t.Fatalf("blind contract: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "open", "--phase", "build", "--wave", "b1", "--role", "slice-wright", "--slice", "SLICE-001", "--contract", wrightContractPath(t)); code != 0 || !strings.Contains(out, "wright round 1") || !strings.Contains(out, "dispatch: wave b1 open") || !strings.Contains(out, "dispatch: ADVISORY: wright contract reminder:") {
		t.Fatalf("valid contract after refusals must open round 1: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "open", "--phase", "vet", "--wave", "v1", "--role", "plan-reviewer", "--packet", reviewerPacketPath(t)); code != 0 {
		t.Fatalf("non-wright waves need no contract: code=%d\n%s", code, out)
	}
}

// A named agent already has its role contract as its system prompt, so both
// packet lints refuse a line that tells it to read an installed contract.
func TestLintsRefuseRoleContractRead(t *testing.T) {
	cases := []struct {
		name, body string
		refuse     bool
	}{
		{"read-first heading", "## Read first\n\n- /root/proj/.omp/agents/devrites-code-reviewer.md\n- spec.md\n", true},
		{"read-first label", "**Read first:**\n- .codex/agents/devrites-code-reviewer.toml\n", true},
		{"inline read", "Read agents/devrites-slice-wright.md before editing.\n", true},
		{"canonical source", "## Read first\n- pack/.claude/agents/devrites-code-reviewer.md\n", false},
		{"changed path under scope", "## Read first\n- spec.md\n## Scope\n- .claude/agents/devrites-code-reviewer.md\n", false},
		{"read-only is not read", "Read-only review of .omp/agents/devrites-doubt.md wording.\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for name, lint := range map[string]func(string) error{"reviewer packet": lintReviewerPacket, "wright contract": lintWrightContract} {
				path := filepath.Join(t.TempDir(), "p.md")
				if err := os.WriteFile(path, []byte(tc.body+"\nWriter-safe checks: go vet ./...\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				err := lint(path)
				if !tc.refuse {
					if err != nil {
						t.Fatalf("%s: want pass, got %v", name, err)
					}
					continue
				}
				if err == nil || !strings.Contains(err.Error(), name+" "+path+":") || !strings.Contains(err.Error(), "tells the agent to read a role contract") {
					t.Fatalf("%s: want refusal, got %v", name, err)
				}
			}
		})
	}
}

// A gates.md `RUN: writer` row makes `gates precheck` a required writer-safe
// check; without such a row, or without gates.md, the contract lint is unchanged.
func TestLintWrightContractForWriterGates(t *testing.T) {
	const runnable = "- [ ] G1: names stay out\n  CHECK: ./names.sh\n  EXPECT: clean\n"
	cases := []struct {
		name, ledger, contract, want string
	}{
		{"no gates.md", "", "Writer-safe checks: go vet ./...\n", ""},
		{"no writer row", runnable, "Writer-safe checks: go vet ./...\n", ""},
		{"writer row, precheck missing", runnable + "  RUN: writer\n", "Writer-safe checks: go vet ./...\n", "must include `devrites-engine gates precheck feat"},
		{"writer row, precheck inline", runnable + "  RUN: writer\n", "Writer-safe checks: go vet ./...; devrites-engine gates precheck feat --worktree .\n", ""},
		{"writer row, precheck listed", runnable + "  RUN: writer\n", "Writer-safe checks:\n- go vet ./...\n- devrites-engine gates precheck feat --worktree .\n", ""},
		{"writer row, sliced precheck", runnable + "  RUN: writer\n", "Writer-safe checks:\n- go vet ./...\n- devrites-engine gates precheck feat --worktree . --slice SLICE-001\n", ""},
		{"writer row, precheck outside block", runnable + "  RUN: writer\n", "Writer-safe checks: go vet ./...\n\nNotes: gates precheck is for later.\n", "must include"},
		{"abandoned writer row", runnable + "  RUN: writer\nABANDON: G1 vendor gone\n", "Writer-safe checks: go vet ./...\n", ""},
		{"root row", runnable + "  RUN: root\n", "Writer-safe checks: go vet ./...\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, featureDir := newWorkspace(t, "build")
			if tc.ledger != "" {
				writeFile(t, filepath.Join(featureDir, "gates.md"), tc.ledger)
			}
			path := filepath.Join(t.TempDir(), "c.md")
			if err := os.WriteFile(path, []byte(tc.contract), 0o644); err != nil {
				t.Fatal(err)
			}
			err := lintWrightContractFor(root, "feat", path)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("want pass, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
		})
	}
	// The base contract lint still runs first.
	root, _ := newWorkspace(t, "build")
	if err := lintWrightContractFor(root, "feat", ""); err == nil || !strings.Contains(err.Error(), "needs --contract") {
		t.Fatalf("missing contract: %v", err)
	}
}
