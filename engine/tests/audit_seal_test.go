package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devrites/devrites/internal/lib"
	"github.com/devrites/devrites/internal/testutil"
)

func TestAuditSealRequiresReviewerSettlement(t *testing.T) {
	for _, barrier := range []string{"open", "sealed"} {
		for _, action := range []string{"seal", "closeout"} {
			t.Run(barrier+"/"+action, func(t *testing.T) {
				_, root := auditSealFixture(t)
				packet := auditSealPacket(t, root)
				auditSealDispatch(t, root, 0, "open", "--phase", "review", "--wave", "audit-current", "--role", "code-reviewer", "--packet", packet)
				if barrier == "sealed" {
					auditSealDispatch(t, root, 0, "start", "--wave", "audit-current", "--role", "code-reviewer", "--handle", "audit-current-handle")
				}
				out, errOut, code := auditSealCLI(t, root, auditSealAction(action)...)
				auditSealBlocked(t, code, out, errOut, "review", "dispatch", "wave")
				if action == "closeout" {
					if code != 3 {
						return // A bypass already moved the workspace; no return can be recorded there.
					}
					auditSealWorkspaceRetained(t, root)
				}
				if barrier == "open" {
					auditSealDispatch(t, root, 0, "start", "--wave", "audit-current", "--role", "code-reviewer", "--handle", "audit-current-handle")
				}
				auditSealReturn(t, root, "audit-current")
				auditSealPass(t, root)
				if action == "closeout" {
					out, errOut, code = auditSealCLI(t, root, "state", "close", "final")
					if code != 0 {
						t.Fatalf("settled closeout: exit=%d stdout=%q stderr=%q", code, out, errOut)
					}
				}
			})
		}
	}
}

func TestAuditSealRequiresEveryLaunchedRoleSettlement(t *testing.T) {
	for _, role := range []string{"proof-runner", "custom-worker"} {
		for _, barrier := range []string{"open", "sealed"} {
			for _, action := range []string{"seal", "closeout"} {
				t.Run(role+"/"+barrier+"/"+action, func(t *testing.T) {
					_, root := auditSealFixture(t)
					auditSealDispatch(t, root, 0, "open", "--phase", "seal", "--wave", "pending", "--role", role)
					if barrier == "sealed" {
						auditSealDispatch(t, root, 0, "start", "--wave", "pending", "--role", role, "--handle", "pending-host")
					}
					out, errOut, code := auditSealCLI(t, root, auditSealAction(action)...)
					auditSealBlocked(t, code, out, errOut, "dispatch", "wave")
					if code != 3 {
						return
					}
					auditSealWorkspaceRetained(t, root)
					if barrier == "open" {
						auditSealDispatch(t, root, 0, "start", "--wave", "pending", "--role", role, "--handle", "pending-host")
					}
					auditSealDispatch(t, root, 0, "return", "--wave", "pending", "--role", role)
					auditSealPass(t, root)
					if action == "closeout" {
						out, errOut, code = auditSealCLI(t, root, "state", "close", "final")
						if code != 0 {
							t.Fatalf("settled closeout: exit=%d stdout=%q stderr=%q", code, out, errOut)
						}
					}
				})
			}
		}
	}
}

func TestAuditSealAbandonedProofNeedsFreshSettledAccount(t *testing.T) {
	for _, role := range []string{"proof-runner", "devrites-proof-runner", "proof-runner.lane", "devrites-proof-runner.lane"} {
		for _, action := range []string{"seal", "closeout"} {
			t.Run(role+"/"+action, func(t *testing.T) {
				_, root := auditSealFixture(t)
				auditSealDispatch(t, root, 0, "open", "--phase", "seal", "--wave", "old-proof", "--role", role)
				auditSealDispatch(t, root, 0, "start", "--wave", "old-proof", "--role", role, "--handle", "old-proof-host")
				auditSealDispatch(t, root, 3, "abandon", "--wave", "old-proof", "--reason", "proof interrupted")
				out, errOut, code := auditSealCLI(t, root, auditSealAction(action)...)
				auditSealBlocked(t, code, out, errOut, "dispatch", "wave")
				if code != 3 {
					return
				}
				auditSealWorkspaceRetained(t, root)
				auditSealDispatch(t, root, 3, "abandon", "--wave", "old-proof", "--reason", "host confirms proof stopped", "--release")
				out, errOut, code = auditSealCLI(t, root, auditSealAction(action)...)
				auditSealBlocked(t, code, out, errOut, "dispatch", "wave")
				if code != 3 {
					return
				}
				auditSealWorkspaceRetained(t, root)
				auditSealDispatch(t, root, 0, "open", "--phase", "seal", "--wave", "fresh-proof", "--role", role)
				auditSealDispatch(t, root, 0, "start", "--wave", "fresh-proof", "--role", role, "--handle", "fresh-proof-host")
				out, errOut, code = auditSealCLI(t, root, auditSealAction(action)...)
				auditSealBlocked(t, code, out, errOut, "dispatch", "wave")
				auditSealDispatch(t, root, 0, "return", "--wave", "fresh-proof", "--role", role)
				auditSealPass(t, root)
				if action == "closeout" {
					out, errOut, code = auditSealCLI(t, root, "state", "close", "final")
					if code != 0 {
						t.Fatalf("reconciled proof closeout: exit=%d stdout=%q stderr=%q", code, out, errOut)
					}
				}
			})
		}
	}
}

func TestAuditSealAbandonedWriterRequiresHostTerminal(t *testing.T) {
	for _, claimed := range []bool{false, true} {
		for _, started := range []bool{false, true} {
			for _, action := range []string{"seal", "closeout"} {
				t.Run(fmt.Sprintf("claim=%t/started=%t/%s", claimed, started, action), func(t *testing.T) {
					_, root := auditSealFixture(t)
					args := []string{"open", "--phase", "seal", "--wave", "old-writer", "--role", "custom-worker"}
					if claimed {
						args = append(args, "--claim-session", "audit-writer", "--claim", "writer-owned.go")
					}
					auditSealDispatch(t, root, 0, args...)
					if started {
						auditSealDispatch(t, root, 0, "start", "--wave", "old-writer", "--role", "custom-worker", "--handle", "old-writer-host")
					}
					auditSealDispatch(t, root, 3, "abandon", "--wave", "old-writer", "--reason", "termination unknown")
					path := filepath.Join(root, "work", "final", "dispatch.json")
					body, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					var ledger struct {
						Waves map[string]map[string]any `json:"waves"`
					}
					if err := json.Unmarshal(body, &ledger); err != nil {
						t.Fatal(err)
					}
					role := "custom-worker"
					if !claimed {
						role = "slice-wright"
						roles := ledger.Waves["old-writer"]["roles"].(map[string]any)
						roles[role] = roles["custom-worker"]
						delete(roles, "custom-worker")
					}
					// A fresh replacement proves settlement, not termination of an old writer.
					ledger.Waves["replacement"] = auditSealWave("seal", "complete", role, "replacement-host", "2099-01-01T00:00:00Z", "2099-01-01T00:00:01Z")
					body, err = json.Marshal(ledger)
					if err != nil {
						t.Fatal(err)
					}
					testutil.WriteFile(t, path, string(body))
					out, errOut, code := auditSealCLI(t, root, auditSealAction(action)...)
					auditSealBlocked(t, code, out, errOut, "dispatch", "wave")
					if code != 3 {
						return
					}
					auditSealWorkspaceRetained(t, root)
					auditSealDispatch(t, root, 3, "abandon", "--wave", "old-writer", "--reason", "host confirms terminal", "--release")
					body, err = os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(body, &ledger); err != nil {
						t.Fatal(err)
					}
					delete(ledger.Waves, "replacement")
					body, err = json.Marshal(ledger)
					if err != nil {
						t.Fatal(err)
					}
					testutil.WriteFile(t, path, string(body))
					auditSealPass(t, root)
					if action == "closeout" {
						out, errOut, code = auditSealCLI(t, root, "state", "close", "final")
						if code != 0 {
							t.Fatalf("confirmed terminal closeout: exit=%d stdout=%q stderr=%q", code, out, errOut)
						}
					}
				})
			}
		}
	}
}

func TestAuditSealAbandonedReviewNeedsFreshSettledRoster(t *testing.T) {
	_, root := auditSealFixture(t)
	packet := auditSealPacket(t, root)
	auditSealDispatch(t, root, 0, "open", "--phase", "review", "--wave", "audit-stale", "--role", "code-reviewer", "--packet", packet)
	auditSealDispatch(t, root, 0, "start", "--wave", "audit-stale", "--role", "code-reviewer", "--handle", "audit-stale-handle")
	auditSealDispatch(t, root, 3, "abandon", "--wave", "audit-stale", "--reason", "superseded synthetic review; no terminal account")
	out, errOut, code := auditSealCLI(t, root, "check", "seal", "final")
	auditSealBlocked(t, code, out, errOut, "review", "dispatch", "wave")

	auditSealDispatch(t, root, 0, "open", "--phase", "review", "--wave", "audit-fresh", "--role", "code-reviewer", "--packet", packet)
	auditSealDispatch(t, root, 0, "start", "--wave", "audit-fresh", "--role", "code-reviewer", "--handle", "audit-fresh-handle")
	out, errOut, code = auditSealCLI(t, root, "check", "seal", "final")
	auditSealBlocked(t, code, out, errOut, "review", "dispatch", "wave")
	auditSealReturn(t, root, "audit-fresh")
	auditSealPass(t, root)
}

func TestAuditSealNeverStartedAbandonmentCanReconcile(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprintf("legacy=%t", legacy), func(t *testing.T) {
			_, root := auditSealFixture(t)
			packet := auditSealPacket(t, root)
			auditSealDispatch(t, root, 0, "open", "--phase", "review", "--wave", "z-refused", "--role", "code-reviewer", "--packet", packet)
			auditSealDispatch(t, root, 3, "abandon", "--wave", "z-refused", "--reason", "host refused launch")
			path := filepath.Join(root, "work", "final", "dispatch.json")
			if legacy {
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var ledger struct {
					Waves map[string]map[string]any `json:"waves"`
				}
				if err := json.Unmarshal(body, &ledger); err != nil {
					t.Fatal(err)
				}
				delete(ledger.Waves["z-refused"], "abandoned_at")
				body, err = json.Marshal(ledger)
				if err != nil {
					t.Fatal(err)
				}
				testutil.WriteFile(t, path, string(body))
				auditSealDispatch(t, root, 0, "open", "--phase", "review", "--wave", "old-account", "--role", "code-reviewer", "--packet", packet)
				auditSealDispatch(t, root, 0, "start", "--wave", "old-account", "--role", "code-reviewer", "--handle", "old-host")
				auditSealReturn(t, root, "old-account")
				out, errOut, code := auditSealCLI(t, root, "check", "seal", "final")
				auditSealBlocked(t, code, out, errOut, "review", "dispatch", "wave")
				auditSealDispatch(t, root, 3, "abandon", "--wave", "z-refused", "--reason", "host confirms launch never started")
				out, errOut, code = auditSealCLI(t, root, "check", "seal", "final")
				auditSealBlocked(t, code, out, errOut, "review", "dispatch", "wave")
			}
			auditSealDispatch(t, root, 0, "open", "--phase", "review", "--wave", "a-fresh", "--role", "code-reviewer", "--packet", packet)
			auditSealDispatch(t, root, 0, "start", "--wave", "a-fresh", "--role", "code-reviewer", "--handle", "fresh-host")
			auditSealReturn(t, root, "a-fresh")
			auditSealPass(t, root)
		})
	}
}

func TestAuditDispatchMalformedLedgerRefusesWithoutMutation(t *testing.T) {
	for _, ledger := range []string{
		`null`,
		`{"waves":null}`,
		`{"waves":{"broken":null}}`,
		`{"waves":{"broken":{"phase":"review","state":"open","roles":null}}}`,
		`{"waves":{"broken":{"phase":"review","state":"open","roles":{"custom":null}}}}`,
		`{"waves":{"broken":{"phase":"review","state":"impossible","roles":{"custom":{}}}}}`,
		`{"waves":{"broken":{"phase":"review","state":"complete","roles":{"custom":{}}}}}`,
		`{"waves":{"broken":{"phase":"review","state":"sealed","roles":{"custom":{}}}}}`,
		`{"waves":{"broken":{"phase":"review","state":"open","roles":{"custom":{"returned":true}}}}}`,
		`{"waves":{"broken":{"phase":"build","state":"abandoned","terminal_at":"invalid","abandoned_at":"2026-01-01T00:00:01Z","roles":{"slice-wright":{}}}}}`,
		`{"waves":{"broken":{"phase":"build","state":"abandoned","terminal_at":"2026-01-01T00:00:00Z","abandoned_at":"2026-01-01T00:00:01Z","roles":{"slice-wright":{}}}}}`,
		`{"waves":{"broken":{"phase":"build","state":"open","terminal_at":"2026-01-01T00:00:01Z","roles":{"slice-wright":{}}}}}`,
	} {
		t.Run(ledger, func(t *testing.T) {
			_, root := auditSealFixture(t)
			path := filepath.Join(root, "work", "final", "dispatch.json")
			testutil.WriteFile(t, path, ledger)
			for _, args := range [][]string{
				{"dispatch", "final", "status"},
				{"dispatch", "final", "abandon", "--wave", "broken", "--reason", "refused"},
				{"dispatch", "final", "open", "--phase", "custom", "--wave", "new", "--role", "custom"},
			} {
				out, errOut, code := auditSealCLI(t, root, args...)
				if code != 3 || !strings.Contains(errOut, "malformed") {
					t.Fatalf("malformed ledger: args=%v exit=%d stdout=%q stderr=%q", args, code, out, errOut)
				}
				body, err := os.ReadFile(path)
				if err != nil || string(body) != ledger {
					t.Fatalf("refusal mutated ledger: body=%q err=%v", body, err)
				}
			}
		})
	}
}

func TestAuditSealRequiresRunnableGateOriginWithoutReexecution(t *testing.T) {
	for _, mutation := range []string{"deleted", "corrupt"} {
		for _, action := range []string{"seal", "closeout"} {
			t.Run(mutation+"/"+action, func(t *testing.T) {
				project, root := auditSealFixture(t)
				work := filepath.Join(root, "work", "final")
				const check = "echo x>>audit-runs.txt && echo AUDIT-ORIGIN-OK"
				testutil.WriteFile(t, filepath.Join(work, "test-plan.md"), "# Test plan\n\nRun synthetic origin probe for AC-001.\n\n## Build-entry preflight\n\n| Command | Cwd | Purpose |\n| --- | --- | --- |\n| "+check+" | . | synthetic origin probe |\n")
				testutil.WriteFile(t, filepath.Join(work, "gates.md"), "# Gates\n\n- [ ] G1: synthetic origin probe\n  CHECK: "+check+"\n  CWD: .\n  EXPECT: AUDIT-ORIGIN-OK\n  EVIDENCE: pending\n")
				binding, bindingErr, bindingCode := auditSealCLI(t, root, "check", "readiness", "--emit-binding", "final")
				if bindingCode != 0 {
					t.Fatalf("runnable fixture readiness binding: exit=%d stdout=%q stderr=%q", bindingCode, binding, bindingErr)
				}
				testutil.WriteFile(t, filepath.Join(work, "eng-review.md"), "# Engineering review\n\nPASS\n\n"+binding)
				out, errOut, code := auditSealCLI(t, root, "gates", "run", "final", "--transcripts", "--timeout", "5")
				if code != 0 {
					t.Fatalf("approved gate execution: exit=%d stdout=%q stderr=%q", code, out, errOut)
				}
				var origin string
				for _, line := range strings.Split(out, "\n") {
					if path, ok := strings.CutPrefix(line, "transcript: "); ok {
						if origin != "" {
							t.Fatalf("expected one executed gate origin: %q", out)
						}
						origin = path
					}
				}
				if origin == "" {
					t.Fatalf("gate execution retained no origin: %q", out)
				}
				if !filepath.IsAbs(origin) {
					origin = filepath.Join(project, origin)
				}
				original, err := os.ReadFile(origin)
				if err != nil {
					t.Fatal(err)
				}
				counter := filepath.Join(project, "audit-runs.txt")
				before, err := os.ReadFile(counter)
				if err != nil || strings.Count(string(before), "x") != 1 {
					t.Fatalf("gate must execute once: counter=%q err=%v", before, err)
				}
				auditSealPass(t, root)
				if mutation == "deleted" {
					if err := os.Remove(origin); err != nil {
						t.Fatal(err)
					}
				} else {
					testutil.WriteFile(t, origin, string(original[:len(original)/2]))
				}
				out, errOut, code = auditSealCLI(t, root, auditSealAction(action)...)
				auditSealBlocked(t, code, out, errOut, "origin", "transcript")
				after, err := os.ReadFile(counter)
				if err != nil || !bytes.Equal(before, after) {
					t.Fatalf("final prerequisite reran consumptive command: before=%q after=%q err=%v", before, after, err)
				}
				if action == "closeout" {
					if code != 3 {
						return
					}
					auditSealWorkspaceRetained(t, root)
				}
				testutil.WriteFile(t, origin, string(original))
				auditSealPass(t, root)
				if action == "closeout" {
					out, errOut, code = auditSealCLI(t, root, "state", "close", "final")
					if code != 0 {
						t.Fatalf("restored origin must clear closeout: exit=%d stdout=%q stderr=%q", code, out, errOut)
					}
				}
				after, err = os.ReadFile(counter)
				if err != nil || !bytes.Equal(before, after) {
					t.Fatalf("restored origin must clear Seal without execution: before=%q after=%q err=%v", before, after, err)
				}
			})
		}
	}
}

func TestAuditSealRetainedNativeAccounts(t *testing.T) {
	const oldStart = "2026-01-01T00:00:00Z"
	const oldReturn = "2026-01-01T00:00:01Z"
	const freshStart = "2026-01-01T00:00:02Z"
	const freshReturn = "2026-01-01T00:00:03Z"
	for _, action := range []string{"seal", "closeout"} {
		for _, test := range []struct {
			name   string
			waves  map[string]any
			absent bool
			pass   bool
		}{
			{name: "deleted ledger", absent: true},
			{name: "all waves deleted", waves: map[string]any{}},
			{name: "proof-only cannot replace reviewer roster", waves: map[string]any{
				"proof": auditSealWave("seal", "complete", "proof-runner", "proof-host", oldStart, oldReturn),
			}},
			{name: "return without start identity", waves: map[string]any{"review": auditSealWave("review", "complete", "code-reviewer", "", oldStart, oldReturn)}},
			{name: "return before start", waves: map[string]any{"review": auditSealWave("review", "complete", "code-reviewer", "native-review", freshStart, oldReturn)}},
			{name: "abandoned with older settled account", waves: map[string]any{
				"a-abandoned": auditSealWave("review", "abandoned", "code-reviewer", "abandoned-review", freshStart, ""),
				"z-older":     auditSealWave("review", "complete", "code-reviewer", "older-review", oldStart, oldReturn),
			}},
			{name: "abandoned with equal timestamp account", waves: map[string]any{
				"a-abandoned": auditSealWave("review", "abandoned", "code-reviewer", "abandoned-review", oldStart, ""),
				"z-equal":     auditSealWave("review", "complete", "code-reviewer", "equal-review", oldStart, oldReturn),
			}},
			{name: "abandoned with other phase", waves: map[string]any{
				"a-abandoned": auditSealWave("review", "abandoned", "code-reviewer", "abandoned-review", oldStart, ""),
				"z-other":     auditSealWave("seal", "complete", "code-reviewer", "fresh-review", freshStart, freshReturn),
			}},
			{name: "abandoned with other role", waves: map[string]any{
				"a-abandoned": auditSealWave("review", "abandoned", "code-reviewer", "abandoned-review", oldStart, ""),
				"z-other":     auditSealWave("review", "complete", "security-auditor", "fresh-review", freshStart, freshReturn),
			}},
			{name: "abandoned with reused identity", waves: map[string]any{
				"a-abandoned": auditSealWave("review", "abandoned", "code-reviewer", "same-review", oldStart, ""),
				"z-reused":    auditSealWave("review", "complete", "code-reviewer", "same-review", freshStart, freshReturn),
			}},
			{name: "fresh settled account independent of wave name", pass: true, waves: map[string]any{
				"z-abandoned": auditSealWave("review", "abandoned", "code-reviewer", "abandoned-review", oldStart, oldReturn),
				"a-fresh":     auditSealWave("review", "complete", "code-reviewer", "fresh-review", freshStart, freshReturn),
			}},
			{name: "native outcome account needs no invented telemetry", pass: true, waves: map[string]any{
				"native": auditSealWave("review", "complete", "code-reviewer", "native-review", oldStart, oldReturn),
			}},
			{name: "abandoned writer with actual native return", pass: true, waves: map[string]any{
				"native": auditSealWave("review", "complete", "code-reviewer", "native-review", oldStart, oldReturn),
				"writer": auditSealWave("build", "abandoned", "slice-wright", "writer-host", oldStart, oldReturn),
			}},
		} {
			t.Run(action+"/"+test.name, func(t *testing.T) {
				_, root := auditSealFixture(t)
				path := filepath.Join(root, "work", "final", "dispatch.json")
				if test.absent {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
				} else {
					body, err := json.Marshal(map[string]any{"waves": test.waves})
					if err != nil {
						t.Fatal(err)
					}
					testutil.WriteFile(t, path, string(body))
				}
				out, errOut, code := auditSealCLI(t, root, auditSealAction(action)...)
				if test.pass {
					if code != 0 {
						t.Fatalf("valid native account blocked: exit=%d stdout=%q stderr=%q", code, out, errOut)
					}
				} else {
					auditSealBlocked(t, code, out, errOut, "review", "dispatch", "wave")
					if action == "closeout" {
						auditSealWorkspaceRetained(t, root)
					}
				}
			})
		}
	}
}

func auditSealWave(phase, status, role, handle, started, returned string) map[string]any {
	return map[string]any{
		"phase": phase, "state": status,
		"roles": map[string]any{role: map[string]any{
			"handle": handle, "started": started, "returned": returned != "", "returned_at": returned,
		}},
	}
}

func TestAuditCloseoutPreservesNonfinalExit(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	work := filepath.Join(root, "work", "unfinished")
	testutil.WriteFile(t, filepath.Join(work, "state.md"), "| schema | 4 |\n| phase | review |\n| status | abandoned |\n")
	testutil.WriteFile(t, filepath.Join(work, "dispatch.json"), `{"waves":{"unfinished":{"phase":"review","state":"open","roles":{"code-reviewer":{}}}}}`)
	testutil.WriteFile(t, filepath.Join(root, "ACTIVE"), "unfinished\n")
	out, errOut, code := auditSealCLI(t, root, "state", "close", "unfinished")
	if code != 0 {
		t.Fatalf("explicit nonfinal archive blocked: exit=%d stdout=%q stderr=%q", code, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(root, "archive", "unfinished", "dispatch.json")); err != nil {
		t.Fatalf("nonfinal audit trail not archived: %v", err)
	}
	if active, err := os.ReadFile(filepath.Join(root, "ACTIVE")); err != nil || len(active) != 0 {
		t.Fatalf("nonfinal archive did not clear ACTIVE: active=%q err=%v", active, err)
	}
}

func TestAuditCloseoutFinalTargetsCannotBypassPrerequisites(t *testing.T) {
	for _, cursor := range []struct{ phase, status string }{
		{"seal", "running"}, {"ship", "running"}, {"done", "running"},
		{"build", "sealed"}, {"build", "shipped"}, {"build", "done"},
	} {
		t.Run(cursor.phase+"/"+cursor.status, func(t *testing.T) {
			_, root := auditSealFixture(t)
			work := filepath.Join(root, "work", "final")
			testutil.WriteFile(t, filepath.Join(work, "state.md"), "| schema | 4 |\n| phase | "+cursor.phase+" |\n| status | "+cursor.status+" |\n")
			if err := os.Remove(filepath.Join(work, "dispatch.json")); err != nil {
				t.Fatal(err)
			}
			out, errOut, code := auditSealCLI(t, root, "state", "close", "final")
			auditSealBlocked(t, code, out, errOut, "review", "dispatch", "wave")
			auditSealWorkspaceRetained(t, root)
		})
	}
}

func TestAuditCloseoutCanonicalFinalPhasesRequirePrerequisites(t *testing.T) {
	for _, phase := range []string{"Seal", "Ship", "Done", "seal (annotation)", "ship (annotation)", "done (annotation)"} {
		for _, bullet := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/bullet=%t", phase, bullet), func(t *testing.T) {
				_, root := auditSealFixture(t)
				work := filepath.Join(root, "work", "final")
				cursor := "| schema | 4 |\n| phase | " + phase + " |\n| status | running |\n"
				if bullet {
					cursor = "- Schema: 4\n- Phase: " + phase + "\n- Status: running\n"
				}
				testutil.WriteFile(t, filepath.Join(work, "state.md"), cursor)
				if err := os.Remove(filepath.Join(work, "dispatch.json")); err != nil {
					t.Fatal(err)
				}
				out, errOut, code := auditSealCLI(t, root, "state", "close", "final")
				auditSealBlocked(t, code, out, errOut, "review", "dispatch", "wave")
				auditSealWorkspaceRetained(t, root)
			})
		}
	}
}

func TestAuditSealCanonicalReviewPhaseReconcilesFreshReplacement(t *testing.T) {
	_, root := auditSealFixture(t)
	packet := auditSealPacket(t, root)
	auditSealDispatch(t, root, 0, "open", "--phase", "Review", "--wave", "audit-stale", "--role", "code-reviewer", "--packet", packet)
	auditSealDispatch(t, root, 0, "start", "--wave", "audit-stale", "--role", "code-reviewer", "--handle", "audit-stale-handle")
	auditSealDispatch(t, root, 3, "abandon", "--wave", "audit-stale", "--reason", "superseded synthetic review")
	auditSealDispatch(t, root, 0, "open", "--phase", "review", "--wave", "audit-fresh", "--role", "code-reviewer", "--packet", packet)
	auditSealDispatch(t, root, 0, "start", "--wave", "audit-fresh", "--role", "code-reviewer", "--handle", "audit-fresh-handle")
	auditSealReturn(t, root, "audit-fresh")
	auditSealPass(t, root)
}

func auditSealFixture(t *testing.T) (string, string) {
	t.Helper()
	project, root := newFinalSealRepo(t)
	testutil.WriteFile(t, filepath.Join(root, "work", "final", "state.md"), "| schema | 4 |\n| phase | seal |\n| status | running |\n")
	testutil.WriteFile(t, filepath.Join(root, "ACTIVE"), "final\n")
	chdirForSeal(t, project)
	auditSealPass(t, root)
	return project, root
}

func auditSealPacket(t *testing.T, root string) string {
	t.Helper()
	testutil.WriteFile(t, filepath.Join(filepath.Dir(root), "source.go"), "package source\n\n// Synthetic current review candidate.\n")
	auditSealDispatch(t, root, 0, "attribute", "--reason", "auditSealPacket fixture owns the synthetic current review candidate write", "--path", "source.go")
	digest, _, err := lib.CandidateIdentity(root, "final")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"evidence.md", "review.md", "seal.md"} {
		path := filepath.Join(root, "work", "final", name)
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(body), "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "Candidate SHA-256: ") {
				lines[i] = "Candidate SHA-256: " + digest
			}
		}
		testutil.WriteFile(t, path, strings.Join(lines, "\n"))
	}
	auditSealPass(t, root)
	out, errOut, code := auditSealCLI(t, root, "candidate", "freeze", "final", "--wave", "audit-candidate", "--paths", "source.go")
	if code != 0 {
		t.Fatalf("freeze current candidate: exit=%d stdout=%q stderr=%q", code, out, errOut)
	}
	work := filepath.Join(root, "work", "final")
	sums, err := os.ReadFile(filepath.Join(work, "packets", "audit-candidate.sha256"))
	if err != nil {
		t.Fatal(err)
	}
	packet := filepath.Join(work, "packets", "audit-review.md")
	testutil.WriteFile(t, packet, "# Scope\nReview source.go against AC-001.\n\n"+string(sums))
	return packet
}

func auditSealReturn(t *testing.T, root, wave string) {
	t.Helper()
	result := filepath.Join(t.TempDir(), "synthetic-review.md")
	testutil.WriteFile(t, result, "Outcome: no-findings\nCounts: critical=0 important=0 gap=0\nScope: synthetic AC-001 fixture account.\n")
	auditSealDispatch(t, root, 0, "return", "--wave", wave, "--role", "code-reviewer", "--result", result)
	auditSealDispatch(t, root, 0, "status", "--wave", wave)
}

func auditSealDispatch(t *testing.T, root string, want int, args ...string) {
	t.Helper()
	out, errOut, code := auditSealCLI(t, root, append([]string{"dispatch", "final"}, args...)...)
	if code != want {
		t.Fatalf("dispatch %v: exit=%d want=%d stdout=%q stderr=%q", args, code, want, out, errOut)
	}
}

func auditSealPass(t *testing.T, root string) {
	t.Helper()
	out, errOut, code := auditSealCLI(t, root, "check", "seal", "final")
	if code != 0 || !strings.Contains(out, "reason: DRV-GATE-SEAL-PASSED") {
		t.Fatalf("otherwise-valid current candidate must pass Seal: exit=%d stdout=%q stderr=%q", code, out, errOut)
	}
}

func auditSealAction(action string) []string {
	if action == "closeout" {
		return []string{"state", "close", "final"}
	}
	return []string{"check", "seal", "final"}
}

func auditSealBlocked(t *testing.T, code int, out, errOut string, markers ...string) {
	t.Helper()
	diagnostic := strings.ToLower(out + errOut)
	semantic := false
	for _, marker := range markers {
		semantic = semantic || strings.Contains(diagnostic, marker)
	}
	if code != 3 || !strings.Contains(diagnostic, "blocked") || !semantic {
		t.Errorf("missing final prerequisite (%v) must block: exit=%d stdout=%q stderr=%q", markers, code, out, errOut)
	}
}

func auditSealWorkspaceRetained(t *testing.T, root string) {
	t.Helper()
	if info, err := os.Stat(filepath.Join(root, "work", "final")); err != nil || !info.IsDir() {
		t.Fatalf("blocked closeout lost workspace: info=%v err=%v", info, err)
	}
	if _, err := os.Lstat(filepath.Join(root, "archive", "final")); !os.IsNotExist(err) {
		t.Fatalf("blocked closeout created archive: %v", err)
	}
	if active, err := os.ReadFile(filepath.Join(root, "ACTIVE")); err != nil || string(active) != "final\n" {
		t.Fatalf("blocked closeout changed ACTIVE: active=%q err=%v", active, err)
	}
}

func auditSealCLI(t *testing.T, root string, args ...string) (string, string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.WaitDelay = 2 * time.Second
	cmd.Env = append(os.Environ(), "DEVRITES_ROOT="+root)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("CLI %v exceeded deadline: %v", args, ctx.Err())
	}
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("CLI %v: %v", args, err)
		}
		code = exit.ExitCode()
	}
	return stdout.String(), stderr.String(), code
}
