package gate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/acceptance"
	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/testutil"
)

const auditOracleLedger = "# Gates\n\n- [ ] G1: probe reports complete\n  CHECK: printf 'probe: unavailable\\n'\n  EXPECT: probe: complete\n  CWD: .\n  INPUTS: probe.txt\n  RUN: root\n  EVIDENCE: pending\n\n- [ ] G2: owner sees expected layout\n  EVIDENCE: pending\n"

func TestAuditGateOracleDefinitionsBindReadiness(t *testing.T) {
	for _, tc := range []struct {
		name, from, to string
	}{
		{"membership added", "# Gates", "# Gates\n\n- [ ] G3: extra outcome"},
		{"membership removed", "\n- [ ] G2: owner sees expected layout\n  EVIDENCE: pending\n", ""},
		{"id", "G1:", "G3:"},
		{"title", "probe reports complete", "probe only starts"},
		{"CHECK", "printf 'probe: unavailable\\n'", "printf 'probe: complete\\n'"},
		{"EXPECT", "EXPECT: probe: complete", "EXPECT: probe:"},
		{"CWD", "CWD: .", "CWD: engine"},
		{"INPUTS", "INPUTS: probe.txt", "INPUTS: narrower.txt"},
		{"RUN", "RUN: root", "RUN: writer"},
		{"runnable to manual", "  CHECK: printf 'probe: unavailable\\n'\n  EXPECT: probe: complete\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			workspace := writeReadinessFixture(t, root, "oracle", "build")
			path := filepath.Join(workspace, "gates.md")
			testutil.WriteFile(t, path, auditOracleLedger)
			before, binding, err := ReadinessInputDigests(root, "oracle")
			if err != nil {
				t.Fatal(err)
			}
			testutil.WriteFile(t, path, strings.Replace(auditOracleLedger, tc.from, tc.to, 1))
			if got := mustReadinessBinding(t, root, "oracle"); got == binding {
				t.Errorf("%s changed gate definition without invalidating Vet", tc.name)
			}
			after, current, err := ReadinessInputDigests(root, "oracle")
			if err != nil {
				t.Fatal(err)
			}
			if current != mustReadinessBinding(t, root, "oracle") {
				t.Fatal("readiness emission and drift introspection disagree")
			}
			if auditGateInput(t, before) == auditGateInput(t, after) {
				t.Error("gate definition change absent from readiness input digests")
			}
		})
	}
}

func TestAuditGateOracleBindingIgnoresProofAndFormatting(t *testing.T) {
	root := t.TempDir()
	workspace := writeReadinessFixture(t, root, "progress", "build")
	path := filepath.Join(workspace, "gates.md")
	testutil.WriteFile(t, path, auditOracleLedger)
	before, binding, err := ReadinessInputDigests(root, "progress")
	if err != nil {
		t.Fatal(err)
	}
	original := acceptance.ParseLedger(auditOracleLedger)
	if len(original.Errors) != 0 {
		t.Fatal(original.Errors)
	}
	for _, tc := range []struct {
		name, content string
	}{
		{"checkbox", strings.ReplaceAll(auditOracleLedger, "[ ]", "[x]")},
		{"evidence", strings.ReplaceAll(auditOracleLedger, "EVIDENCE: pending", "EVIDENCE: owner observed layout in browser")},
		{"automatic evidence", strings.Replace(auditOracleLedger, "EVIDENCE: pending", "EVIDENCE: "+acceptance.FormatEvidence(original.Gates[0], acceptance.OutputDigest("probe: complete\n"), len("probe: complete\n"), "2026-11-01T00:00:00Z"), 1)},
		{"progress prose", auditOracleLedger + "\n## Progress\n\nOne outcome inspected; proof still pending.\n"},
		{"fenced example", auditOracleLedger + "\n```text\n- [ ] EXAMPLE: not an obligation\n  CHECK: false\n  EXPECT: impossible\n```\n"},
		{"CRLF", strings.ReplaceAll(auditOracleLedger, "\n", "\r\n")},
		{"command code span", strings.Replace(auditOracleLedger, "CHECK: printf 'probe: unavailable\\n'", "CHECK: `printf 'probe: unavailable\\n'`", 1)},
		{"gate order", "# Gates\n\n- [ ] G2: owner sees expected layout\n  EVIDENCE: pending\n\n" + strings.Split(strings.TrimPrefix(auditOracleLedger, "# Gates\n\n"), "\n- [ ] G2:")[0]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testutil.WriteFile(t, path, tc.content)
			if got := mustReadinessBinding(t, root, "progress"); got != binding {
				t.Errorf("proof/formatting changed Vet binding: %q != %q", got, binding)
			}
			after, current, err := ReadinessInputDigests(root, "progress")
			if err != nil || current != binding {
				t.Fatalf("introspection binding = %q, %v; want %q", current, err, binding)
			}
			if got := auditGateInput(t, after); got != auditGateInput(t, before) {
				t.Errorf("proof/formatting changed readiness gate digest: %+v", got)
			}
		})
	}
}

func TestAuditGateOracleFreshProofCannotReuseStaleVet(t *testing.T) {
	for _, mutation := range []string{"weaken EXPECT", "convert to manual"} {
		t.Run(mutation, func(t *testing.T) {
			project := t.TempDir()
			root := filepath.Join(project, ".devrites")
			workspace := writeReadinessFixture(t, root, "fresh", "build")
			const ledger = "# Gates\n\n- [ ] G1: probe reports complete\n  CHECK: printf 'probe: unavailable\\n'\n  EXPECT: probe: complete\n  EVIDENCE: pending\n"
			testutil.WriteFile(t, filepath.Join(workspace, "gates.md"), ledger)
			testutil.WriteFile(t, filepath.Join(workspace, "test-plan.md"), "# Test plan\n\n## Build-entry preflight\n\n| Gate | Command | Cwd | Expected |\n| --- | --- | --- | --- |\n| probe | `printf 'probe: unavailable\\n'` | . | probe reports complete |\n")
			binding := mustReadinessBinding(t, root, "fresh")
			testutil.AppendFile(t, filepath.Join(workspace, "eng-review.md"), "\n"+binding+"\n")
			result, err := Check(Readiness, root, "fresh")
			if err != nil || result.Blocked {
				t.Fatalf("valid Build fixture rejected: result=%+v error=%v", result, err)
			}
			var stdout, stderr bytes.Buffer
			if code := acceptance.Run(root, []string{"run", "fresh"}, &stdout, &stderr); code != acceptance.ExitBlocked || !strings.Contains(stdout.String(), "gate G1: fail") {
				t.Fatalf("original oracle must fail on real execution: code=%d stdout=%s stderr=%s", code, &stdout, &stderr)
			}
			args := []string{"reverify", "fresh"}
			changed := strings.Replace(ledger, "EXPECT: probe: complete", "EXPECT: probe:", 1)
			if mutation == "convert to manual" {
				changed = "# Gates\n\n- [ ] G1: probe reports complete\n  EVIDENCE: pending\n"
				args = []string{"attest", "fresh", "G1", "Operator inspected probe result in local terminal"}
			}
			testutil.WriteFile(t, filepath.Join(workspace, "gates.md"), changed)
			stdout.Reset()
			stderr.Reset()
			if code := acceptance.Run(root, args, &stdout, &stderr); code != acceptance.ExitOK {
				t.Fatalf("fresh proof setup failed: code=%d stdout=%s stderr=%s", code, &stdout, &stderr)
			}
			stdout.Reset()
			stderr.Reset()
			if code := acceptance.Run(root, []string{"status", "fresh"}, &stdout, &stderr); code != acceptance.ExitOK || !strings.Contains(stdout.String(), "result: all-met") {
				t.Fatalf("fresh proof must satisfy edited ledger: code=%d stdout=%s stderr=%s", code, &stdout, &stderr)
			}
			testutil.WriteFile(t, filepath.Join(workspace, "state.md"), "| phase | prove |\n| schema | 4 |\n")
			for _, name := range []string{"evidence.md", "touched-files.md", "review.md", "seal.md"} {
				testutil.WriteFile(t, filepath.Join(workspace, name), "# Proof\n\nCurrent proof retained.\n")
			}
			for _, kind := range []Kind{Readiness, Seal} {
				result, err := Check(kind, root, "fresh")
				if err != nil {
					t.Fatal(err)
				}
				if !result.Blocked || result.ReasonID != reason.GateReadinessStale || !strings.Contains(result.Render(), "rerun /rite-vet") {
					t.Errorf("fresh proof accepted old Vet after %s: %s", mutation, result.Render())
				}
			}
		})
	}
}

func TestAuditGateOracleExecutionWritebackPreservesVet(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	workspace := writeReadinessFixture(t, root, "writeback", "build")
	testutil.WriteFile(t, filepath.Join(workspace, "gates.md"), "# Gates\n\n- [ ] G1: probe reports complete\n  CHECK: printf 'probe: complete\\n'\n  EXPECT: probe: complete\n  EVIDENCE: pending\n")
	testutil.WriteFile(t, filepath.Join(workspace, "test-plan.md"), "# Test plan\n\n## Build-entry preflight\n\n| Gate | Command | Cwd | Expected |\n| --- | --- | --- | --- |\n| probe | `printf 'probe: complete\\n'` | . | probe reports complete |\n")
	before, binding, err := ReadinessInputDigests(root, "writeback")
	if err != nil {
		t.Fatal(err)
	}
	testutil.AppendFile(t, filepath.Join(workspace, "eng-review.md"), "\n"+binding+"\n")
	var stdout, stderr bytes.Buffer
	if code := acceptance.Run(root, []string{"run", "writeback"}, &stdout, &stderr); code != acceptance.ExitOK || !strings.Contains(stdout.String(), "gate G1: pass") {
		t.Fatalf("real proof execution failed: code=%d stdout=%s stderr=%s", code, &stdout, &stderr)
	}
	after, current, err := ReadinessInputDigests(root, "writeback")
	if err != nil || current != binding || auditGateInput(t, before) != auditGateInput(t, after) {
		t.Fatalf("runner proof writeback changed vetted input: binding=%q error=%v", current, err)
	}
	result, err := Check(Readiness, root, "writeback")
	if err != nil || result.Blocked {
		t.Fatalf("ordinary proof writeback invalidated Vet: result=%+v error=%v", result, err)
	}
}

func TestAuditGateOracleBindingRejectsUndefinedObligations(t *testing.T) {
	for _, tc := range []struct {
		name, content string
	}{
		{"missing", ""},
		{"empty", ""},
		{"heading only", "# Gates\n"},
		{"no obligations", "# Gates\n\nAll finished.\n"},
		{"partial runnable", "- [ ] G1: outcome\n  CHECK: true\n"},
		{"invalid EXPECT", "- [ ] G1: outcome\n  CHECK: true\n  EXPECT: /[/\n"},
		{"unsafe CWD", "- [ ] G1: outcome\n  CWD: ../outside\n"},
		{"duplicate ID", "- [ ] G1: first\n- [ ] G1: second\n"},
		{"malformed Markdown", "- [ ] G1: outcome\x00\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			workspace := writeReadinessFixture(t, root, "undefined", "vet")
			path := filepath.Join(workspace, "gates.md")
			if tc.name == "missing" {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			} else {
				testutil.WriteFile(t, path, tc.content)
			}
			if binding, err := ReadinessBinding(root, "undefined"); err == nil || !strings.Contains(err.Error(), "gates.md") {
				t.Errorf("undefined gates emitted Vet binding: binding=%q error=%v", binding, err)
			}
			if _, binding, err := ReadinessInputDigests(root, "undefined"); err == nil || !strings.Contains(err.Error(), "gates.md") {
				t.Errorf("undefined gates emitted drift binding: binding=%q error=%v", binding, err)
			}
		})
	}
}

func TestAuditGateOracleBindingPreservesEarlierPhaseOptionality(t *testing.T) {
	root := t.TempDir()
	workspace := writeReadinessFixture(t, root, "planning", "plan")
	if err := os.Remove(filepath.Join(workspace, "gates.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadinessBinding(root, "planning"); err != nil {
		t.Fatalf("gates.md required before Vet: %v", err)
	}
	result, err := Check(Readiness, root, "planning")
	if err != nil || result.Blocked {
		t.Fatalf("planning blocked without optional gates.md: result=%+v error=%v", result, err)
	}
}

func TestAuditGateOracleBindingRejectsPreOracleApproval(t *testing.T) {
	root := t.TempDir()
	workspace := writeReadinessFixture(t, root, "old-vet", "build")
	// Released v1 binding omitted gates.md entirely; it requires normal re-Vet.
	const oldBinding = "Readiness inputs SHA-256: 6a0d4991d86f5a0e78333289840b5fb1a123adbb18b53243b1af34fcffcf1961"
	testutil.AppendFile(t, filepath.Join(workspace, "eng-review.md"), "\n"+oldBinding+"\n")
	result, err := Check(Readiness, root, "old-vet")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Blocked || result.ReasonID != reason.GateReadinessStale || !strings.Contains(result.Render(), "rerun /rite-vet") {
		t.Fatalf("pre-oracle Vet approval must require re-Vet: %s", result.Render())
	}
}

func auditGateInput(t *testing.T, digests []ReadinessInputDigest) ReadinessInputDigest {
	t.Helper()
	for _, input := range digests {
		if input.Path == "gates.md" && input.Present {
			return input
		}
	}
	t.Fatal("gates.md missing from readiness input digests")
	return ReadinessInputDigest{}
}
