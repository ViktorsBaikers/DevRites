package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/state"
)

func continuationWorkspace(t *testing.T, phase, status, action string) (string, string) {
	t.Helper()
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(project, ".devrites")
	work := filepath.Join(root, "work", "alpha")
	writeFile(t, filepath.Join(root, "ACTIVE"), "alpha\n")
	writeFile(t, filepath.Join(root, "AFK"), "allow_gates: [advisory, validating]\n")
	writeFile(t, filepath.Join(work, "state.md"), cursor(phase, status, action)+"| schema | 4 |\n")
	writeFile(t, filepath.Join(work, "questions.md"), "# Questions\n")
	return root, work
}

func TestContinuationRecognizesSupportedHostForms(t *testing.T) {
	for _, command := range []string{"/rite-vet", "$rite-vet", "/skill:rite-vet"} {
		t.Run(command, func(t *testing.T) {
			root, _ := continuationWorkspace(t, "vet", "running", command+" alpha")
			var out, errOut bytes.Buffer
			if code := RunCheckContinuation(root, []string{"alpha"}, &out, &errOut); code != 0 {
				t.Fatalf("exit %d: %s", code, errOut.String())
			}
			if !strings.Contains(out.String(), "continuation: required\n") || !strings.Contains(out.String(), "invoke: "+command+" alpha\n") {
				t.Fatalf("host cursor lost continuation:\n%s", out.String())
			}
		})
	}
}

func TestContinuationRefusesForeignSource(t *testing.T) {
	for _, action := range []string{"/rite-vet beta", "$rite-vet beta", "/skill:rite-vet beta", "/rite-plan repair beta"} {
		t.Run(action, func(t *testing.T) {
			root, work := continuationWorkspace(t, "vet", "running", action)
			before, err := os.ReadFile(filepath.Join(work, "state.md"))
			if err != nil {
				t.Fatal(err)
			}
			var out, errOut bytes.Buffer
			if code := RunCheckContinuation(root, []string{"alpha"}, &out, &errOut); code == 0 || errOut.Len() == 0 {
				t.Fatalf("foreign cursor needs explicit refusal, exit=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
			}
			if strings.Contains(out.String(), "continuation: required") || strings.Contains(out.String(), "continuation: clear") {
				t.Fatalf("foreign cursor received legitimate verdict: %s", out.String())
			}
			after, err := os.ReadFile(filepath.Join(work, "state.md"))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("continuation mutated cursor: %v", err)
			}
		})
	}
}

func TestContinuationSupportsPhaseLocalArguments(t *testing.T) {
	for _, action := range []string{"/rite-vet", "$rite-vet", "/skill:rite-vet", "/rite-build SLICE-001", "$rite-build --parallel 2 SLICE-001", "/skill:rite-build SLICE-001", "/rite-plan repair", "/rite-plan repair alpha", "/rite-polish harden", "/rite-review slice 1", "/rite-review feature --full", "/rite-temper --mode selective"} {
		t.Run(action, func(t *testing.T) {
			root, _ := continuationWorkspace(t, "vet", "running", action)
			writeFile(t, filepath.Join(root, "ACTIVE"), "beta\n")
			var out, errOut bytes.Buffer
			if code := RunCheckContinuation(root, []string{"alpha"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "continuation: required\n") {
				t.Fatalf("valid same-source cursor refused: exit=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
			}
			if !strings.Contains(out.String(), "source: alpha\n") {
				t.Fatalf("selected SOURCE lost: %s", out.String())
			}
			if !strings.Contains(action, "vet") && !strings.Contains(action, "temper") && !strings.Contains(out.String(), "invoke: "+action+"\n") {
				t.Fatalf("phase-local argv changed: %s", out.String())
			}
		})
	}
}

func TestResolveRestoresExecutableContinuation(t *testing.T) {
	for _, mode := range []string{"answer", "drop", "batch"} {
		t.Run(mode, func(t *testing.T) {
			root, work := continuationWorkspace(t, "vet", "awaiting_human", "/rite-resolve Q-001")
			writeFile(t, filepath.Join(root, "ACTIVE"), "beta\n")
			t.Setenv("DEVRITES_WORKSPACE", work)
			writeFile(t, filepath.Join(work, "questions.md"), "## Q-001\nstatus: open\ngate: blocking\n")
			path := filepath.Join(work, "state.md")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, path, string(data)+"| question_id | Q-001 |\n\n## Awaiting human\n- qid: Q-001\n\n## Log\n")
			args := []string{"--human", "Q-001", "approved"}
			if mode == "drop" {
				args = []string{"--human", "--drop", "Q-001", "obsolete"}
			} else if mode == "batch" {
				file := filepath.Join(t.TempDir(), "answers.txt")
				writeFile(t, file, "Q-001: approved\n")
				args = []string{"--human", "--batch", file}
			}
			var out, errOut bytes.Buffer
			if code := Resolve(root, args, &out, &errOut); code != 0 {
				t.Fatalf("resolve exit %d: %s", code, errOut.String())
			}
			data, err = os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(string(data), "\n")
			if action, _ := state.CursorField(lines, state.CursorNextAction); action != "/rite-vet alpha" {
				t.Errorf("resume action=%q, want executable selected-source command", action)
			}
			if qid, _ := state.CursorField(lines, state.CursorQuestionID); qid != "" || strings.Contains(string(data), "## Awaiting human") {
				t.Errorf("matching wait representation survived:\n%s", data)
			}
			out.Reset()
			errOut.Reset()
			if code := RunCheckContinuation(root, []string{"alpha"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "continuation: required\n") {
				t.Fatalf("cold resume lost required continuation: exit=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
			}
		})
	}
}

func TestResolvePreservesConflictingWait(t *testing.T) {
	for _, qid := range []string{"Q-001", "Q-002"} {
		t.Run(qid, func(t *testing.T) {
			root, work := continuationWorkspace(t, "vet", "awaiting_human", "/rite-resolve Q-002")
			writeFile(t, filepath.Join(work, "questions.md"), "## Q-001\nstatus: open\n\n## Q-002\nstatus: open\n")
			path := filepath.Join(work, "state.md")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, path, string(data)+"| question_id | Q-002 |\n\n## Awaiting human\n- qid: Q-001\n- gate: blocking\n\n## Log\n")
			var out, errOut bytes.Buffer
			if code := Resolve(root, []string{qid, "answer"}, &out, &errOut); code != 0 {
				t.Fatalf("resolve exit %d: %s", code, errOut.String())
			}
			data, err = os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			text := string(data)
			if !strings.Contains(text, "| status | awaiting_human |") {
				t.Errorf("conflicting wait resumed prematurely:\n%s", text)
			}
			if qid == "Q-001" {
				if strings.Contains(text, "## Awaiting human") || !strings.Contains(text, "| question_id | Q-002 |") {
					t.Errorf("matching block not cleared independently:\n%s", text)
				}
			} else if !strings.Contains(text, "## Awaiting human\n- qid: Q-001\n- gate: blocking") || strings.Contains(text, "| question_id |") {
				t.Errorf("unrelated block lost or matching cursor retained:\n%s", text)
			}
		})
	}
}

func TestResolveRejectsBlankAnswersWithoutWrites(t *testing.T) {
	for _, mode := range []string{"single", "batch"} {
		for _, answer := range []string{"", " \t "} {
			t.Run(mode+"/"+answer, func(t *testing.T) {
				root, work := continuationWorkspace(t, "vet", "awaiting_human", "/rite-resolve Q-001")
				writeFile(t, filepath.Join(work, "questions.md"), "## Q-001\nstatus: open\n\n## Q-002\nstatus: open\n")
				writeFile(t, filepath.Join(work, "state.md"), cursor("vet", "awaiting_human", "/rite-resolve Q-001")+"| schema | 4 |\n| question_id | Q-001 |\n\n## Awaiting human\n- qid: Q-001\n")
				before := map[string][]byte{}
				for _, name := range []string{"questions.md", "state.md"} {
					data, err := os.ReadFile(filepath.Join(work, name))
					if err != nil {
						t.Fatal(err)
					}
					before[name] = data
				}
				args := []string{"--human", "Q-001", answer}
				if mode == "batch" {
					file := filepath.Join(t.TempDir(), "answers.txt")
					writeFile(t, file, "Q-002: valid\nQ-001: "+answer+"\n")
					args = []string{"--human", "--batch", file}
				}
				var out, errOut bytes.Buffer
				if code := Resolve(root, args, &out, &errOut); code != 5 {
					t.Errorf("blank answer exit=%d, want 5; stdout=%q stderr=%q", code, out.String(), errOut.String())
				}
				for name, original := range before {
					after, err := os.ReadFile(filepath.Join(work, name))
					if err != nil || !bytes.Equal(original, after) {
						t.Errorf("blank answer changed %s: %v", name, err)
					}
				}
			})
		}
	}
}

func TestContinuationRefusesMalformedPhaseActions(t *testing.T) {
	for _, action := range []string{"/rite-vet alpha beta", "/rite-vet --unknown", "resume with /rite-vet alpha", "/rite-build --parallel 0", "/rite-plan beta", "/rite-review beta"} {
		t.Run(action, func(t *testing.T) {
			root, _ := continuationWorkspace(t, "vet", "running", action)
			var out, errOut bytes.Buffer
			if code := RunCheckContinuation(root, []string{"alpha"}, &out, &errOut); code == 0 || errOut.Len() == 0 {
				t.Fatalf("malformed action needs refusal: exit=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
			}
			if strings.Contains(out.String(), "continuation: clear") || strings.Contains(out.String(), "continuation: required") {
				t.Fatalf("malformed action received legitimate verdict: %s", out.String())
			}
		})
	}
}

func TestResolveResumesNativePhaseArguments(t *testing.T) {
	for _, test := range []struct{ phase, action string }{
		{"plan", "/rite-vet alpha"},
		{"build", "/rite-build"},
		{"polish", "/rite-polish"},
		{"review", "/rite-review"},
	} {
		t.Run(test.phase, func(t *testing.T) {
			root, work := continuationWorkspace(t, test.phase, "awaiting_human", "/rite-resolve Q-001")
			writeFile(t, filepath.Join(work, "state.md"), cursor(test.phase, "awaiting_human", "/rite-resolve Q-001")+"| schema | 4 |\n| question_id | Q-001 |\n")
			writeFile(t, filepath.Join(work, "questions.md"), "## Q-001\nstatus: open\n")
			var out, errOut bytes.Buffer
			if code := Resolve(root, []string{"Q-001", "approved"}, &out, &errOut); code != 0 {
				t.Fatalf("resolve exit %d: %s", code, errOut.String())
			}
			data, err := os.ReadFile(filepath.Join(work, "state.md"))
			if err != nil {
				t.Fatal(err)
			}
			if action, _ := state.CursorField(strings.Split(string(data), "\n"), state.CursorNextAction); action != test.action {
				t.Fatalf("resume=%q, want native command %q", action, test.action)
			}
			out.Reset()
			if code := RunCheckContinuation(root, []string{"alpha"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "continuation: required\n") || !strings.Contains(out.String(), "source: alpha\n") {
				t.Fatalf("cold resume lost source: exit=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
			}
		})
	}
}

func TestResolveDropReasonRemainsOptional(t *testing.T) {
	for _, mode := range []string{"single", "batch"} {
		t.Run(mode, func(t *testing.T) {
			root, work := continuationWorkspace(t, "vet", "awaiting_human", "/rite-resolve Q-001")
			writeFile(t, filepath.Join(work, "questions.md"), "## Q-001\nstatus: open\n")
			args := []string{"--drop", "Q-001"}
			if mode == "batch" {
				file := filepath.Join(t.TempDir(), "answers.txt")
				writeFile(t, file, "--drop Q-001: \t\n")
				args = []string{"--batch", file}
			}
			var out, errOut bytes.Buffer
			if code := Resolve(root, args, &out, &errOut); code != 0 {
				t.Fatalf("drop exit %d: %s", code, errOut.String())
			}
			data, err := os.ReadFile(filepath.Join(work, "questions.md"))
			if err != nil || !strings.Contains(string(data), "status: dropped") {
				t.Fatalf("optional drop reason rejected: %v\n%s", err, data)
			}
		})
	}
}

func TestHandoffRetainsContinuationSource(t *testing.T) {
	root, _ := continuationWorkspace(t, "build", "running", "/rite-build SLICE-001")
	writeFile(t, filepath.Join(root, "ACTIVE"), "beta\n")
	var out, errOut bytes.Buffer
	if code := RunHandoff(root, []string{"alpha"}, &out, &errOut); code != 0 {
		t.Fatalf("handoff exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "source: alpha\n") || !strings.Contains(out.String(), "invoke: /rite-build SLICE-001\n") {
		t.Fatalf("handoff lost native command/source binding:\n%s", out.String())
	}
}

func TestResolveRejectsInvalidWaitWithoutPartialWrites(t *testing.T) {
	for _, problem := range []string{"conflicting block", "unknown phase"} {
		for _, mode := range []string{"answer", "drop", "batch"} {
			t.Run(problem+"/"+mode, func(t *testing.T) {
				phase, qid := "vet", "Q-002"
				wait := "| question_id | Q-001 |\n\n## Awaiting human\n- qid: Q-001\n\n## Awaiting human\n- qid: Q-002\n- question_id: Q-003\n"
				if problem == "unknown phase" {
					phase, qid = "unknown", "Q-001"
					wait = "| question_id | Q-001 |\n"
				}
				root, work := continuationWorkspace(t, phase, "awaiting_human", "/rite-resolve "+qid)
				writeFile(t, filepath.Join(work, "state.md"), cursor(phase, "awaiting_human", "/rite-resolve "+qid)+"| schema | 4 |\n"+wait)
				writeFile(t, filepath.Join(work, "questions.md"), "## Q-001\nstatus: open\n\n## Q-002\nstatus: open\n")
				before := map[string][]byte{}
				for _, name := range []string{"questions.md", "state.md"} {
					data, err := os.ReadFile(filepath.Join(work, name))
					if err != nil {
						t.Fatal(err)
					}
					before[name] = data
				}
				args := []string{qid, "approved"}
				if mode == "drop" {
					args = []string{"--drop", qid, "obsolete"}
				} else if mode == "batch" {
					file := filepath.Join(t.TempDir(), "answers.txt")
					writeFile(t, file, "Q-001: approved\nQ-002: approved\n")
					args = []string{"--batch", file}
				}
				var out, errOut bytes.Buffer
				if code := Resolve(root, args, &out, &errOut); code != 1 || errOut.Len() == 0 {
					t.Fatalf("invalid state needs explicit failure: exit=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
				}
				for name, original := range before {
					after, err := os.ReadFile(filepath.Join(work, name))
					if err != nil || !bytes.Equal(original, after) {
						t.Errorf("rejected resolution changed %s: %v", name, err)
					}
				}
			})
		}
	}
}

func TestContinuationAcceptsDocumentedActionReason(t *testing.T) {
	for _, command := range []string{"/rite-plan", "$rite-plan", "/skill:rite-plan"} {
		t.Run(command, func(t *testing.T) {
			root, _ := continuationWorkspace(t, "plan", "running", command+" repair alpha — escalate exhausted workflow attempt")
			var out, errOut bytes.Buffer
			if code := RunCheckContinuation(root, []string{"alpha"}, &out, &errOut); code != 0 {
				t.Fatalf("documented cursor refused: exit=%d stderr=%q", code, errOut.String())
			}
			if !strings.Contains(out.String(), "continuation: required\n") || !strings.Contains(out.String(), "invoke: "+command+" repair alpha\n") || !strings.Contains(out.String(), "source: alpha\n") {
				t.Fatalf("documented reason became invocation argv or lost SOURCE:\n%s", out.String())
			}
		})
	}
}

func TestContinuationReasonCannotHideForeignOrMultipleInvocations(t *testing.T) {
	for _, action := range []string{"/rite-plan repair beta — escalate exhausted workflow attempt", "/rite-plan repair alpha — invoke /rite-vet beta"} {
		t.Run(action, func(t *testing.T) {
			root, _ := continuationWorkspace(t, "plan", "running", action)
			var out, errOut bytes.Buffer
			if code := RunCheckContinuation(root, []string{"alpha"}, &out, &errOut); code == 0 || errOut.Len() == 0 {
				t.Fatalf("unsafe cursor needs explicit refusal: exit=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
			}
			if strings.Contains(out.String(), "continuation: clear") || strings.Contains(out.String(), "continuation: required") {
				t.Fatalf("unsafe cursor received legitimate verdict:\n%s", out.String())
			}
		})
	}
}
