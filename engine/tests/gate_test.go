package main_test

// CLI coverage for deterministic completeness and state-invariant gates.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	internalgate "github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/lib"
	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/testutil"
)

func TestReadinessBlocksIncompletePhase(t *testing.T) {
	root := newWorkspace(t)
	out, errOut, code := runDevrites(t, root, "check", "readiness", "auth-tokens") // build phase, tasks empty
	wantOut := "gate: readiness\n" +
		"feature: auth-tokens\n" +
		"phase: build\n" +
		"result: blocked (missing to leave \"build\": tasks.md)\n" +
		"reason: DRV-GATE-READINESS-MISSING\n" +
		"next: add real content to tasks.md\n" +
		"retry: devrites-engine check readiness auth-tokens\n"
	if code != 3 || out != wantOut || errOut != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q, want code=3 stdout=%q stderr empty", code, out, errOut, wantOut)
	}
}

func TestReadinessPassesCompletePhase(t *testing.T) {
	root := newWorkspace(t)
	out, errOut, code := runDevrites(t, root, "check", "readiness", "search-ranking")
	wantOut := "gate: readiness\n" +
		"feature: search-ranking\n" +
		"phase: spec\n" +
		"result: pass\n" +
		"reason: DRV-GATE-READINESS-PASSED\n"
	if code != 0 || out != wantOut || errOut != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q, want code=0 stdout=%q stderr empty", code, out, errOut, wantOut)
	}
}

func TestReadinessBlocksAbsentRequiredArtifactExactCLIContract(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	const slug = "absent-required"
	writeCompleteGateCLIWorkspace(t, root, slug, state.PhaseSpec, state.PhaseSpec, "none\n")
	if err := os.Remove(filepath.Join(root, "work", slug, "spec.md")); err != nil {
		t.Fatal(err)
	}

	out, errOut, code := runDevrites(t, root, "check", "readiness", slug)
	wantOut := "gate: readiness\n" +
		"feature: absent-required\n" +
		"phase: spec\n" +
		"result: blocked (missing to leave \"spec\": spec.md)\n" +
		"reason: DRV-GATE-READINESS-MISSING\n" +
		"next: add real content to spec.md\n" +
		"retry: devrites-engine check readiness absent-required\n"
	if code != 3 || out != wantOut || errOut != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q, want code=3 stdout=%q stderr empty", code, out, errOut, wantOut)
	}
}

func TestReadinessBlocksStaleBindingExactCLIContract(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	const slug = "stale-readiness"
	writeCompleteGateCLIWorkspace(t, root, slug, state.PhaseBuild, state.PhaseBuild, "none\n")
	workspace := filepath.Join(root, "work", slug)
	binding, err := internalgate.ReadinessBinding(root, slug)
	if err != nil {
		t.Fatal(err)
	}
	testutil.AppendFile(t, filepath.Join(workspace, "eng-review.md"), "\n"+binding+"\n")
	testutil.WriteFile(t, filepath.Join(workspace, "plan.md"), "# Plan\n\nChanged after Vet.\n")
	expected, err := internalgate.ReadinessBinding(root, slug)
	if err != nil {
		t.Fatal(err)
	}

	out, errOut, code := runDevrites(t, root, "check", "readiness", slug)
	wantOut := "gate: readiness\n" +
		"feature: " + slug + "\n" +
		"phase: build\n" +
		"result: blocked (state invariant)\n" +
		"reason: DRV-GATE-READINESS-STALE\n" +
		"invariant: readiness inputs are stale or the binding is invalid; rerun /rite-vet and record exactly one standalone line: " + expected + "\n" +
		"retry: devrites-engine check readiness " + slug + "\n"
	if code != 3 || out != wantOut || errOut != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q, want code=3 stdout=%q stderr empty", code, out, errOut, wantOut)
	}
}

// Leaving Vet, readiness replays Build-entry `check slice` over every unbuilt
// slice, so a plan defect is found once at Vet instead of once per dispatch.
func TestReadinessReplaysSliceCheckForUnbuiltSlicesLeavingVet(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	const slug = "vet-slices"
	writeCompleteGateCLIWorkspace(t, root, slug, state.PhaseVet, state.PhaseVet, "none\n")
	workspace := filepath.Join(root, "work", slug)
	testutil.WriteFile(t, filepath.Join(workspace, "spec.md"), "# Spec\n\n## Acceptance criteria\n- AC-001: one\n- AC-002: two\n")
	testutil.WriteFile(t, filepath.Join(workspace, "test-plan.md"), "# Test plan\n\nAC-001 and AC-002: go test ./...\n")
	readiness := func(built, allowlist2 string) (string, int) {
		t.Helper()
		testutil.WriteFile(t, filepath.Join(workspace, "state.md"), "- Phase: vet\n- Status: running\n"+built)
		testutil.WriteFile(t, filepath.Join(workspace, "tasks.md"), "# Tasks\n\n"+
			"## SLICE-001 One\nGoal: one\nSatisfies: AC-001\nDependencies: none\nWriter allowlist: src/a.go\nFiles likely touched: src/a.go\nTests/proof: go test ./...\n\n"+
			"## SLICE-002 Two\nGoal: two\nSatisfies: AC-002\nDependencies: none\n"+allowlist2+"Files likely touched: src/b.go\nTests/proof: go test ./...\n\n"+
			"## SLICE-003 Deferred hardening\nGoal: later\nSatisfies: none\nDependencies: none\n")
		testutil.WriteFile(t, filepath.Join(workspace, "eng-review.md"), "# Engineering review\n\nREADY\n")
		binding, err := internalgate.ReadinessBinding(root, slug)
		if err != nil {
			t.Fatal(err)
		}
		testutil.AppendFile(t, filepath.Join(workspace, "eng-review.md"), "\n"+binding+"\n")
		out, errOut, code := runDevrites(t, root, "check", "readiness", slug)
		if errOut != "" {
			t.Fatalf("stderr=%q", errOut)
		}
		return out, code
	}
	const pass = "gate: readiness\nfeature: " + slug + "\nphase: vet\nresult: pass\nreason: DRV-GATE-READINESS-PASSED\n"

	out, code := readiness("", "")
	want := "gate: readiness\n" +
		"feature: " + slug + "\n" +
		"phase: vet\n" +
		"result: blocked (state invariant)\n" +
		"reason: DRV-GATE-READINESS-MISSING\n" +
		"invariant: slice-check: SLICE-002 would fail Build-entry `check slice`: Writer allowlist missing or empty — dispatch requires exact project-relative paths; fix it in /rite-plan before READY\n" +
		"retry: devrites-engine check readiness " + slug + "\n"
	if code != 3 || out != want {
		t.Fatalf("unbuilt slice without a Writer allowlist: code=%d stdout=%q, want code=3 stdout=%q", code, out, want)
	}
	// The allowlist clears it; the deferred Satisfies: none slice never blocks.
	if out, code := readiness("", "Writer allowlist: src/b.go\n"); code != 0 || out != pass {
		t.Fatalf("with a Writer allowlist: code=%d stdout=%q, want code=0 stdout=%q", code, out, pass)
	}
	// A slice state.md checks built is never dispatched again, so it is exempt.
	if out, code := readiness("- [x] SLICE-002 built — abc1234\n", ""); code != 0 || out != pass {
		t.Fatalf("built slice without a Writer allowlist: code=%d stdout=%q, want code=0 stdout=%q", code, out, pass)
	}
}

func TestReadinessUsesStructuralArtifactsOnly(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	testutil.WriteFile(t, filepath.Join(root, "work", "tempered", "state.md"), `# State

## Cursor
| Key | Value |
| --- | --- |
| phase | temper |
| status | running |
| next_action | /rite-define |
`)
	testutil.WriteFile(t, filepath.Join(root, "work", "tempered", "spec.md"), "# Spec\n\nReady.\n")
	for name, body := range map[string]string{
		"brief.md":             "# Brief\n\nReady.\n",
		"decisions.md":         "# Decisions\n\nNone open.\n",
		"assumptions.md":       "# Assumptions\n\nNone.\n",
		"questions.md":         "# Questions\n\nNone.\n",
		"decision-coverage.md": "# Decision coverage\n\nNOT CLEAR: native agents own this judgment.\n",
	} {
		testutil.WriteFile(t, filepath.Join(root, "work", "tempered", name), body)
	}

	out, errOut, code := runDevrites(t, root, "check", "readiness", "tempered")
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr:\n%s", code, errOut)
	}
	if !strings.Contains(out, "phase: temper") || !strings.Contains(out, "result: pass") {
		t.Fatalf("unexpected readiness output:\n%s", out)
	}
}

func TestGateQuestionPolicyCLIContract(t *testing.T) {
	const openQuestion = "## q-1\nstatus: open\ngate: blocking\n"
	cases := []struct {
		name       string
		kind       string
		current    state.Phase
		required   state.Phase
		wantCode   int
		wantResult string
		wantReason string
	}{
		{name: "frame-open", kind: "readiness", current: state.PhaseFrame, required: state.PhaseFrame, wantCode: 0, wantResult: "pass", wantReason: "DRV-GATE-READINESS-PASSED"},
		{name: "spec-open", kind: "readiness", current: state.PhaseSpec, required: state.PhaseSpec, wantCode: 0, wantResult: "pass", wantReason: "DRV-GATE-READINESS-PASSED"},
		{name: "clarify-open", kind: "readiness", current: state.PhaseClarify, required: state.PhaseClarify, wantCode: 3, wantResult: "blocked (state invariant)", wantReason: "DRV-GATE-READINESS-MISSING"},
		{name: "seal-from-spec", kind: "seal", current: state.PhaseSpec, required: state.PhaseSeal, wantCode: 3, wantResult: "blocked (state invariant)", wantReason: "DRV-GATE-SEAL-MISSING"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			writeCompleteGateCLIWorkspace(t, root, tc.name, tc.current, tc.required, openQuestion)
			if tc.kind == "seal" {
				binding, err := internalgate.ReadinessBinding(root, tc.name)
				if err != nil {
					t.Fatal(err)
				}
				testutil.AppendFile(t, filepath.Join(root, "work", tc.name, "eng-review.md"), "\n"+binding+"\n")
			}

			out, errOut, code := runDevrites(t, root, "check", tc.kind, tc.name)
			wantOut := "gate: " + tc.kind + "\nfeature: " + tc.name + "\nphase: " + string(tc.current) + "\nresult: " + tc.wantResult + "\nreason: " + tc.wantReason + "\n"
			if tc.wantCode == 3 {
				wantOut += "invariant: open blocking human question(s) remain in questions.md but state.md is not awaiting_human\nretry: devrites-engine check " + tc.kind + " " + tc.name + "\n"
			}
			if code != tc.wantCode || out != wantOut || errOut != "" {
				t.Fatalf("code=%d stdout=%q stderr=%q, want code=%d stdout=%q stderr empty", code, out, errOut, tc.wantCode, wantOut)
			}
		})
	}
}

func TestSealBlocksWhenSealSectionsMissing(t *testing.T) {
	root := newWorkspace(t)
	out, _, code := runDevrites(t, root, "check", "seal", "auth-tokens")
	if code != 3 {
		t.Fatalf("exit = %d, want 3 (blocked)", code)
	}
	if !strings.Contains(out, "result: blocked (missing to seal:") || !strings.Contains(out, "tasks.md") || !strings.Contains(out, "evidence.md") {
		t.Errorf("unexpected seal block line\n%s", out)
	}
}

func TestSealPassesWhenComplete(t *testing.T) {
	project, root := newFinalSealRepo(t)
	source := filepath.Join(project, "source.go")
	if err := os.Chtimes(source, time.Unix(2_000_000_000, 0), time.Unix(2_000_000_000, 0)); err != nil {
		t.Fatal(err)
	}
	digest, _, err := lib.CandidateIdentity(root, "final")
	if err != nil {
		t.Fatal(err)
	}
	chdirForSeal(t, project)

	out, errOut, code := runDevrites(t, root, "check", "seal", "final")
	wantOut := "evidence-fresh: OK: candidate digest " + digest + " matches evidence, review, and seal.\n" +
		"gate: seal\n" +
		"feature: final\n" +
		"phase: seal\n" +
		"result: pass\n" +
		"reason: DRV-GATE-SEAL-PASSED\n"
	if code != 0 || out != wantOut || errOut != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q, want code=0 stdout=%q stderr empty", code, out, errOut, wantOut)
	}
}

func TestSealBlocksContentChangeWithRestoredMtime(t *testing.T) {
	project, root := newFinalSealRepo(t)
	codePath := filepath.Join(project, "source.go")
	info, err := os.Stat(codePath)
	if err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, codePath, "package changed\n")
	if err := os.Chtimes(codePath, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	chdirForSeal(t, project)

	out, errOut, code := runDevrites(t, root, "check", "seal", "final")
	if code != 3 {
		t.Fatalf("exit = %d, want 3\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}
	if !strings.Contains(errOut, "candidate digest line does not match") {
		t.Fatalf("stderr missing freshness block:\n%s", errOut)
	}
	if !strings.Contains(out, "reason: DRV-GATE-SEAL-MISSING") {
		t.Fatalf("stdout missing stable blocked reason:\n%s", out)
	}
}

func TestSealBlocksInvalidCandidateBindings(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{name: "duplicate evidence digest", mutate: func(t *testing.T, dir string) {
			path := filepath.Join(dir, "evidence.md")
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			line := body[strings.LastIndex(string(body), "Candidate SHA-256:"):]
			testutil.WriteFile(t, path, string(body)+string(line))
		}},
		{name: "missing review digest", mutate: func(t *testing.T, dir string) {
			path := filepath.Join(dir, "review.md")
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			start := strings.LastIndex(string(body), "Candidate SHA-256:")
			testutil.WriteFile(t, path, string(body[:start]))
		}},
		{name: "optional browser missing digest", mutate: func(t *testing.T, dir string) {
			testutil.WriteFile(t, filepath.Join(dir, "browser-evidence.md"), "# Browser evidence\n\nCapture present without binding.\n")
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			project, root := newFinalSealRepo(t)
			test.mutate(t, filepath.Join(root, "work", "final"))
			chdirForSeal(t, project)
			out, errOut, code := runDevrites(t, root, "check", "seal", "final")
			if code != 3 || !strings.Contains(out, "reason: DRV-GATE-SEAL-MISSING") || !strings.Contains(errOut, "evidence-fresh: BLOCKED") {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errOut)
			}
		})
	}
}

func TestSealTracksCanonicalQuestionTableStatus(t *testing.T) {
	project, root := newFinalSealRepo(t)
	questions := filepath.Join(root, "work", "final", "questions.md")
	table := `# Questions

## Question register
| Question ID | Status | Gate | Question | Answer | Impact |
| --- | --- | --- | --- | --- | --- |
| Q-001 | answered | validating | Include a header? | Yes. | AC-001 |
`
	testutil.WriteFile(t, questions, table)
	chdirForSeal(t, project)

	if out, errOut, code := runDevrites(t, root, "check", "seal", "final"); code != 0 {
		t.Fatalf("answered table row blocked seal: exit=%d\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}

	testutil.WriteFile(t, questions, strings.Replace(table, "| answered |", "| open |", 1))
	out, errOut, code := runDevrites(t, root, "check", "seal", "final")
	if code != 3 {
		t.Fatalf("open table row exit=%d, want 3\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}
	if !strings.Contains(out, "open validating human question(s) remain") {
		t.Fatalf("seal did not report the canonical open table row:\n%s", out)
	}
}

func TestWorkspaceObservationEarlyStateErrorsCLIContract(t *testing.T) {
	for _, test := range []struct {
		name        string
		slug        string
		mutate      func(*testing.T, string)
		wantErr     string
		notDisclose []string
	}{
		{
			name: "absent state",
			slug: "absent-state",
			mutate: func(t *testing.T, workspace string) {
				if err := os.Remove(filepath.Join(workspace, "state.md")); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: `devrites: gate readiness: feature "absent-state": state.md is absent; add real content to state.md and retry
`,
		},
		{
			name: "empty state",
			slug: "empty-state",
			mutate: func(t *testing.T, workspace string) {
				testutil.WriteFile(t, filepath.Join(workspace, "state.md"), "")
			},
			wantErr: `devrites: gate readiness: feature "empty-state": state.md is empty; add real content to state.md and retry
`,
		},
		{
			name: "malformed state",
			slug: "malformed-state",
			mutate: func(t *testing.T, workspace string) {
				testutil.WriteFile(t, filepath.Join(workspace, "state.md"), "# State\n\nhostile-secret\x00\n")
			},
			wantErr: `devrites: gate readiness: feature "malformed-state": state.md is malformed (malformed_markdown); repair state.md and retry
`,
			notDisclose: []string{"hostile-secret"},
		},
		{
			name: "unsafe state final symlink",
			slug: "unsafe-state",
			mutate: func(t *testing.T, workspace string) {
				const target = "hostile-state-target.md"
				statePath := filepath.Join(workspace, "state.md")
				if err := os.Remove(statePath); err != nil {
					t.Fatal(err)
				}
				testutil.WriteFile(t, filepath.Join(workspace, target), "- Phase: spec\n- Status: running\n")
				if err := os.Symlink(target, statePath); err != nil {
					t.Fatalf("create final symlink fixture: %v", err)
				}
			},
			wantErr: `devrites: gate readiness: feature "unsafe-state": state.md is unsafe (final_symlink); repair state.md and retry
`,
			notDisclose: []string{"hostile-state-target.md"},
		},
		{
			name: "state without phase",
			slug: "state-without-phase",
			mutate: func(t *testing.T, workspace string) {
				testutil.WriteFile(t, filepath.Join(workspace, "state.md"), "# State\n\n- Status: running\n")
			},
			wantErr: `devrites: gate readiness: feature "state-without-phase": no phase in state.md ledger; record phase in state.md and retry
`,
		},
		{
			name: "state with unknown phase",
			slug: "state-with-unknown-phase",
			mutate: func(t *testing.T, workspace string) {
				testutil.WriteFile(t, filepath.Join(workspace, "state.md"), "# State\n\n- Phase: mystery\n- Status: running\n")
			},
			wantErr: `devrites: gate readiness: feature "state-with-unknown-phase": unknown phase "mystery"; record a known phase in state.md and retry
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			writeCompleteGateCLIWorkspace(t, root, test.slug, state.PhaseSpec, state.PhaseSpec, "none\n")
			workspace := filepath.Join(root, "work", test.slug)
			test.mutate(t, workspace)

			out, errOut, code := runDevrites(t, root, "check", "readiness", test.slug)
			if code != 2 || out != "" || errOut != test.wantErr {
				t.Fatalf("code=%d stdout=%q stderr=%q, want code=2 stdout empty stderr=%q", code, out, errOut, test.wantErr)
			}
			for _, secret := range test.notDisclose {
				if strings.Contains(out, secret) || strings.Contains(errOut, secret) {
					t.Errorf("CLI output disclosed hostile fixture content %q: stdout=%q stderr=%q", secret, out, errOut)
				}
			}
		})
	}
}

func TestWorkspaceObservationGateDiagnosticCLIContract(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeCompleteGateCLIWorkspace(t, root, "diagnostic", state.PhaseSpec, state.PhaseSpec, "none\n")
	testutil.WriteFile(t, filepath.Join(root, "work", "diagnostic", "spec.md"), "hostile-secret\x00")

	out, errOut, code := runDevrites(t, root, "check", "readiness", "diagnostic")
	wantOut := "gate: readiness\n" +
		"feature: diagnostic\n" +
		"phase: spec\n" +
		"result: blocked (missing to leave \"spec\": spec.md)\n" +
		"reason: DRV-GATE-READINESS-MISSING\n" +
		"artifact: spec.md: malformed (malformed_markdown)\n" +
		"next: repair spec.md: replace invalid Markdown with valid Markdown; required artifacts need substantive content\n" +
		"retry: devrites-engine check readiness diagnostic\n"
	if code != 3 || out != wantOut || errOut != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q, want code=3 stdout=%q stderr empty", code, out, errOut, wantOut)
	}
}

func TestWorkspaceObservationStandaloneBindingDiagnosticCLIContract(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeCompleteGateCLIWorkspace(t, root, "binding-diagnostic", state.PhaseBuild, state.PhaseBuild, "none\n")
	testutil.WriteFile(t, filepath.Join(root, "work", "binding-diagnostic", "strategy.md"), "hostile-secret\x00")

	out, errOut, code := runDevrites(t, root, "check", "readiness", "--emit-binding", "binding-diagnostic")
	wantErr := "readiness-binding: BLOCKED: readiness input strategy.md is malformed (malformed_markdown); replace invalid Markdown with valid Markdown\n"
	if code != 3 || out != "" || errOut != wantErr {
		t.Fatalf("code=%d stdout=%q stderr=%q, want code=3 stdout empty stderr=%q", code, out, errOut, wantErr)
	}
}

func TestWorkspaceObservationConstructibleDiagnosticsCLIContract(t *testing.T) {
	for _, test := range []struct {
		name       string
		slug       string
		mutate     func(*testing.T, string)
		diagnostic string
		recovery   string
	}{
		{
			name: "final symlink gate",
			slug: "final-symlink",
			mutate: func(t *testing.T, workspace string) {
				path := filepath.Join(workspace, "spec.md")
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("brief.md", path); err != nil {
					t.Fatalf("create final symlink fixture: %v", err)
				}
			},
			diagnostic: "artifact: spec.md: unsafe (final_symlink)\n",
			recovery:   "next: repair spec.md: replace the symlink with a regular file\n",
		},
		{
			name: "nonregular gate",
			slug: "nonregular",
			mutate: func(t *testing.T, workspace string) {
				path := filepath.Join(workspace, "spec.md")
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatal(err)
				}
			},
			diagnostic: "artifact: spec.md: unsafe (non_regular)\n",
			recovery:   "next: repair spec.md: replace the non-regular entry with a regular file\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			writeCompleteGateCLIWorkspace(t, root, test.slug, state.PhaseSpec, state.PhaseSpec, "none\n")
			workspace := filepath.Join(root, "work", test.slug)
			test.mutate(t, workspace)

			out, errOut, code := runDevrites(t, root, "check", "readiness", test.slug)
			wantOut := "gate: readiness\n" +
				"feature: " + test.slug + "\n" +
				"phase: spec\n" +
				"result: blocked (missing to leave \"spec\": spec.md)\n" +
				"reason: DRV-GATE-READINESS-MISSING\n" +
				test.diagnostic +
				test.recovery +
				"retry: devrites-engine check readiness " + test.slug + "\n"
			if code != 3 || out != wantOut || errOut != "" {
				t.Fatalf("code=%d stdout=%q stderr=%q, want code=3 stdout=%q stderr empty", code, out, errOut, wantOut)
			}
		})
	}

	t.Run("file too large standalone binding", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), ".devrites")
		writeCompleteGateCLIWorkspace(t, root, "binding-too-large", state.PhaseBuild, state.PhaseBuild, "none\n")
		testutil.WriteFile(t, filepath.Join(root, "work", "binding-too-large", "strategy.md"), strings.Repeat("x", (1<<20)+1))

		out, errOut, code := runDevrites(t, root, "check", "readiness", "--emit-binding", "binding-too-large")
		wantErr := "readiness-binding: BLOCKED: readiness input strategy.md is unsafe (file_too_large); reduce the file to at most 1 MiB\n"
		if code != 3 || out != "" || errOut != wantErr {
			t.Fatalf("code=%d stdout=%q stderr=%q, want code=3 stdout empty stderr=%q", code, out, errOut, wantErr)
		}
	})
}

func TestWorkspaceObservationWholeFailureCLIChannels(t *testing.T) {
	t.Run("workspace invalid gate", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), ".devrites")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		out, errOut, code := runDevrites(t, root, "check", "readiness", "missing")
		wantErr := "devrites: gate readiness: workspace observation: workspace_invalid: workspace is unavailable; verify the selected logical workspace and canonical workspace override, then retry\n"
		if code != 2 || out != "" || errOut != wantErr {
			t.Fatalf("code=%d stdout=%q stderr=%q, want code=2 stdout empty stderr=%q", code, out, errOut, wantErr)
		}
	})

	t.Run("workspace invalid binding", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), ".devrites")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		out, errOut, code := runDevrites(t, root, "check", "readiness", "--emit-binding", "missing")
		wantErr := "readiness-binding: BLOCKED: workspace observation: workspace_invalid: workspace is unavailable; verify the selected logical workspace and canonical workspace override, then retry\n"
		if code != 3 || out != "" || errOut != wantErr {
			t.Fatalf("code=%d stdout=%q stderr=%q, want code=3 stdout empty stderr=%q", code, out, errOut, wantErr)
		}
	})

	t.Run("aggregate too large gate and binding", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), ".devrites")
		writeCompleteGateCLIWorkspace(t, root, "aggregate", state.PhaseBuild, state.PhaseBuild, "none\n")
		for i, name := range []string{"state.md", "brief.md", "spec.md", "decisions.md", "assumptions.md", "questions.md", "decision-coverage.md", "architecture.md", "plan.md"} {
			prefix := "retained content\n"
			if i == 0 {
				prefix = "- Phase: build\n"
			}
			body := prefix + strings.Repeat("x", (1<<20)-len(prefix))
			testutil.WriteFile(t, filepath.Join(root, "work", "aggregate", name), body)
		}
		wantFailure := "workspace observation: aggregate_too_large: retained content exceeds the 8 MiB aggregate limit; reduce retained Markdown below 8 MiB, then retry"
		out, errOut, code := runDevrites(t, root, "check", "readiness", "aggregate")
		if code != 2 || out != "" || errOut != "devrites: gate readiness: "+wantFailure+"\n" {
			t.Fatalf("gate code=%d stdout=%q stderr=%q", code, out, errOut)
		}
		out, errOut, code = runDevrites(t, root, "check", "readiness", "--emit-binding", "aggregate")
		if code != 3 || out != "" || errOut != "readiness-binding: BLOCKED: "+wantFailure+"\n" {
			t.Fatalf("binding code=%d stdout=%q stderr=%q", code, out, errOut)
		}
	})
}

func writeCompleteGateCLIWorkspace(t *testing.T, root, slug string, current, required state.Phase, questions string) {
	t.Helper()
	policy, ok := state.PolicyFor(required)
	if !ok {
		t.Fatalf("PolicyFor(%q) returned unknown", required)
	}
	questionsRequired := false
	for _, artifact := range policy.RequiredArtifacts {
		name := string(artifact)
		content := "# " + name + "\n\nreal\n"
		switch name {
		case "state.md":
			content = "- Phase: " + string(current) + "\n- Status: running\n"
		case "questions.md":
			questionsRequired = true
			content = questions
		case "tasks.md":
			content = testutil.CanonicalTasksMarkdown
		case "gates.md":
			content = testutil.CanonicalGatesMarkdown
		}
		testutil.WriteFile(t, filepath.Join(root, "work", slug, name), content)
	}
	if !questionsRequired {
		testutil.WriteFile(t, filepath.Join(root, "work", slug, "questions.md"), questions)
	}
}

func newFinalSealRepo(t *testing.T) (string, string) {
	t.Helper()
	requireGit(t)
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	dir := filepath.Join(root, "work", "final")
	initGitRepo(t, project)
	runGit(t, project, "commit", "--allow-empty", "-q", "-m", "test: empty final candidate base")
	for name, body := range map[string]string{
		"state.md":             "| phase | seal |\n| status | running |\n",
		"brief.md":             "# Brief\n\nFinal seal aggregate.\n",
		"spec.md":              "# Spec\n\n## Acceptance Criteria\n- AC-001: final seal passes every deterministic gate.\n",
		"decisions.md":         "# Decisions\n\n## Decisions stood\n- none\n",
		"assumptions.md":       "# Assumptions\n\n- none\n",
		"questions.md":         "# Questions\n\n- none\n",
		"decision-coverage.md": "# Decision coverage\n\nCLEAR\n",
		"architecture.md":      "# Architecture\n\nUse existing gates.\n",
		"plan.md":              "# Plan\n\nCompose the final checks.\n",
		"tasks.md":             testutil.CanonicalTasksMarkdown,
		"traceability.md":      "# Traceability\n\nAC-001 -> final seal.\n",
		"eng-review.md":        "# Engineering review\n\nPASS\n",
		"test-plan.md":         "# Test plan\n\nRun focused Go tests for AC-001.\n",
		"gates.md":             testutil.CanonicalGatesMarkdown,
		"evidence.md":          "# Evidence\n\nFocused Go tests passed.\n",
		"touched-files.md":     "# Touched files\n\n## Touched files\nCandidate paths are declared only in the manifest below.\n\n## Candidate manifest\n| State | File | Slice | Reason |\n| --- | --- | --- | --- |\n| present | `source.go` | S-1 | Final Seal fixture. |\n",
		"review.md": `# Review

## Spec
No-findings: checked AC-001 against the final candidate.

## Code review
No-findings: checked the aggregate order and exit mapping.
`,
		"seal.md": finalSeal(true),
	} {
		testutil.WriteFile(t, filepath.Join(dir, name), body)
	}
	testutil.WriteFile(t, filepath.Join(project, "source.go"), "package source\n")
	bindingOut, bindingErr, bindingCode := runDevrites(t, root, "check", "readiness", "--emit-binding", "final")
	if bindingCode != 0 {
		t.Fatalf("emit readiness binding exit=%d\nstdout:\n%s\nstderr:\n%s", bindingCode, bindingOut, bindingErr)
	}
	testutil.AppendFile(t, filepath.Join(dir, "eng-review.md"), "\n"+strings.TrimSuffix(bindingOut, "\n")+"\n")
	digest, _, err := lib.CandidateIdentity(root, "final")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"evidence.md", "review.md", "seal.md"} {
		path := filepath.Join(dir, name)
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		testutil.WriteFile(t, path, string(body)+"\nCandidate SHA-256: "+digest+"\n")
	}
	// Retain a synthetic native reviewer account, not only the prose roster.
	if out, errOut, code := runDevrites(t, root, "candidate", "freeze", "final", "--wave", "fixture-candidate", "--paths", "source.go"); code != 0 {
		t.Fatalf("freeze fixture candidate: exit=%d stdout=%q stderr=%q", code, out, errOut)
	}
	sums, err := os.ReadFile(filepath.Join(dir, "packets", "fixture-candidate.sha256"))
	if err != nil {
		t.Fatal(err)
	}
	packet := filepath.Join(dir, "packets", "fixture-review.md")
	testutil.WriteFile(t, packet, "# Scope\nSynthetic fixture review of source.go against AC-001.\n\n"+string(sums))
	result := filepath.Join(t.TempDir(), "fixture-review.return.md")
	testutil.WriteFile(t, result, "Outcome: no-findings\nCounts: critical=0 important=0 gap=0\nScope: synthetic final Seal fixture only.\n")
	for _, args := range [][]string{
		{"open", "--phase", "review", "--wave", "fixture-review", "--role", "code-reviewer", "--packet", packet},
		{"start", "--wave", "fixture-review", "--role", "code-reviewer", "--handle", "synthetic-fixture-reviewer"},
		{"return", "--wave", "fixture-review", "--role", "code-reviewer", "--handle", "synthetic-fixture-reviewer", "--result", result},
	} {
		if out, errOut, code := runDevrites(t, root, append([]string{"dispatch", "final"}, args...)...); code != 0 {
			t.Fatalf("native fixture account %v: exit=%d stdout=%q stderr=%q", args, code, out, errOut)
		}
	}
	gitCommitAll(t, project, "test: final seal fixture")
	return project, root
}

func finalSeal(proven bool) string {
	checkbox := " "
	if proven {
		checkbox = "x"
	}
	return `# Seal

## Acceptance Criteria
- [` + checkbox + `] AC-001: final seal passes every deterministic gate.

## Reviewer Accounts
- devrites-spec-reviewer: review.md ` + "`## Spec`" + `
- devrites-code-reviewer: review.md ` + "`## Code review`" + `
- devrites-test-analyst: No-findings: focused and full test gates passed; no assertion gaps found.
- devrites-frontend-reviewer: Not-applicable: no UI changes.
- devrites-security-auditor: No-findings: checked CLI boundary and fail-closed exits.
- devrites-performance-reviewer: Not-applicable: no hot path changed.
- devrites-devex-reviewer: No-findings: checked command output and documentation.
`
}

func chdirForSeal(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}
