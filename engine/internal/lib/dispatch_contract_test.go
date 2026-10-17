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
	if code, out := runDispatchT(t, root, "open", "--phase", "build", "--wave", "b1", "--role", "slice-wright", "--slice", "SLICE-001", "--contract", wrightContractPath(t)); code != 0 || !strings.Contains(out, "wright round 1") {
		t.Fatalf("valid contract after refusals must open round 1: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "open", "--phase", "vet", "--wave", "v1", "--role", "plan-reviewer", "--packet", reviewerPacketPath(t)); code != 0 {
		t.Fatalf("non-wright waves need no contract: code=%d\n%s", code, out)
	}
}
