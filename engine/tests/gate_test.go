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
	out, _, code := runDevrites(t, root, "check", "readiness", "auth-tokens") // build phase, tasks empty
	if code != 3 {
		t.Fatalf("exit = %d, want 3 (blocked)", code)
	}
	if !strings.Contains(out, `result: blocked (missing to leave "build": tasks.md)`) {
		t.Errorf("stdout missing the block line\n%s", out)
	}
	if !strings.Contains(out, "devrites-engine check readiness auth-tokens") {
		t.Errorf("stdout missing the actionable retry step\n%s", out)
	}
}

func TestReadinessPassesCompletePhase(t *testing.T) {
	root := newWorkspace(t)
	out, _, code := runDevrites(t, root, "check", "readiness", "search-ranking") // spec phase, spec present
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (pass)", code)
	}
	if !strings.Contains(out, "result: pass") {
		t.Errorf("stdout not a pass\n%s", out)
	}
	// A not-yet-required section (proof, empty at the spec phase) must not appear
	// as a blocker.
	if strings.Contains(out, "blocked") {
		t.Errorf("an empty not-yet-required section blocked readiness\n%s", out)
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
	chdirForSeal(t, project)

	out, errOut, code := runDevrites(t, root, "check", "seal", "final")
	if code != 0 {
		t.Fatalf("exit = %d, want 0\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}
	for _, want := range []string{
		"result: pass",
		"evidence-fresh: OK",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
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
		"tasks.md":             "# Tasks\n\n- [x] Build final seal.\n",
		"traceability.md":      "# Traceability\n\nAC-001 -> final seal.\n",
		"eng-review.md":        "# Engineering review\n\nPASS\n",
		"test-plan.md":         "# Test plan\n\nRun focused Go tests.\n",
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
