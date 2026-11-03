package lib

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestContinuationRequiredForRunningVetEdge(t *testing.T) {
	root, featureDir := newWorkspace(t, "vet")
	writeFile(t, filepath.Join(root, "AFK"), "allow_gates: [advisory, validating]\n")
	writeFile(t, filepath.Join(featureDir, "state.md"), `# State

## Cursor
| Key | Value |
| --- | --- |
| phase | vet |
| status | running |
| next_action | /rite-vet feat |
`)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunCheckContinuation(root, []string{"feat"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"continuation: required\n",
		"invoke: /rite-vet feat\n",
		"reason: AFK status=running internal phase edge\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestContinuationClearWithoutAFKOrOnHumanStops(t *testing.T) {
	tests := []struct {
		name   string
		afk    bool
		state  string
		quest  string
		reason string
	}{
		{
			name:   "no sentinel",
			state:  cursor("vet", "running", "/rite-vet feat"),
			reason: "reason: no AFK sentinel\n",
		},
		{
			name:   "awaiting human",
			afk:    true,
			state:  cursor("vet", "awaiting_human", "/rite-vet feat"),
			reason: "reason: status=awaiting_human\n",
		},
		{
			name:   "ship",
			afk:    true,
			state:  cursor("seal", "running", "/rite-ship feat"),
			reason: "reason: next_action is not an internal phase edge\n",
		},
		{
			name:   "question cursor",
			afk:    true,
			state:  cursor("vet", "running", "/rite-vet feat") + "| question_id | q-1 |\n",
			reason: "reason: question cursor set\n",
		},
		{
			name:   "open blocking question",
			afk:    true,
			state:  cursor("vet", "running", "/rite-build SLICE-001"),
			quest:  "# Questions\n\n## q-1\nstatus: open\ngate: blocking\n",
			reason: "reason: open blocking question\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, featureDir := newWorkspace(t, "vet")
			if test.afk {
				writeFile(t, filepath.Join(root, "AFK"), "allow_gates: [advisory, validating]\n")
			}
			writeFile(t, filepath.Join(featureDir, "state.md"), test.state)
			if test.quest != "" {
				writeFile(t, filepath.Join(featureDir, "questions.md"), test.quest)
			}
			stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
			if code := RunCheckContinuation(root, []string{"feat"}, stdout, stderr); code != 0 {
				t.Fatalf("code=%d stderr=%s", code, stderr.String())
			}
			if !strings.Contains(stdout.String(), "continuation: clear\n") || !strings.Contains(stdout.String(), test.reason) {
				t.Fatalf("stdout:\n%s", stdout.String())
			}
			if strings.Contains(stdout.String(), "invoke:") {
				t.Fatalf("clear verdict printed invoke:\n%s", stdout.String())
			}
		})
	}
}

func TestContinuationRequiredPastValidatingQuestion(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	writeFile(t, filepath.Join(root, "AFK"), "allow_gates: [advisory, validating]\n")
	writeFile(t, filepath.Join(featureDir, "state.md"), cursor("build", "running", "/rite-build SLICE-002"))
	writeFile(t, filepath.Join(featureDir, "questions.md"), "# Questions\n\n## q-1\nstatus: open\ngate: validating\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunCheckContinuation(root, []string{"feat"}, stdout, stderr); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "continuation: required\n") || !strings.Contains(stdout.String(), "invoke: /rite-build SLICE-002\n") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestHandoffPrintsContinuationVerdict(t *testing.T) {
	root, featureDir := newWorkspace(t, "vet")
	writeFile(t, filepath.Join(root, "AFK"), "allow_gates: [advisory, validating]\n")
	writeFile(t, filepath.Join(featureDir, "state.md"), cursor("vet", "running", "/rite-vet feat"))
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunHandoff(root, []string{"feat"}, stdout, stderr); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "continuation: required\n") || !strings.Contains(stdout.String(), "invoke: /rite-vet feat\n") {
		t.Fatalf("handoff missing continuation:\n%s", stdout.String())
	}
}

func cursor(phase, status, next string) string {
	return "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | " + phase + " |\n| status | " + status + " |\n| next_action | " + next + " |\n"
}

func TestContinuationVetReadyKeepsExistingGates(t *testing.T) {
	for _, tc := range []struct {
		name, phase, status, next, extra, questions string
		required                                    bool
		refused                                     bool
	}{
		{name: "ready Vet", phase: "vet", status: "ready", next: "/rite-build SLICE-001", required: true},
		{name: "ready with question", phase: "vet", status: "ready", next: "/rite-build SLICE-001", extra: "| question_id | Q-001 |\n"},
		{name: "ready with human gate", phase: "vet", status: "ready", next: "/rite-build SLICE-001", questions: "## Q-001\nstatus: open\ngate: blocking\n"},
		{name: "awaiting human", phase: "vet", status: "awaiting_human", next: "/rite-build SLICE-001"},
		{name: "other ready phase", phase: "plan", status: "ready", next: "/rite-vet feat"},
		{name: "Ship not approval", phase: "vet", status: "ready", next: "/rite-ship feat"},
		{name: "controller not phase approval", phase: "vet", status: "ready", next: "/rite-autocomplete"},
		{name: "foreign cursor", phase: "vet", status: "ready", next: "/rite-vet other", refused: true},
		{name: "malformed cursor", phase: "vet", status: "ready", next: "/rite-build --parallel nope", refused: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, dir := newWorkspace(t, tc.phase)
			writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 2\n")
			writeFile(t, filepath.Join(dir, "state.md"), cursor(tc.phase, tc.status, tc.next)+tc.extra)
			if tc.questions != "" {
				writeFile(t, filepath.Join(dir, "questions.md"), tc.questions)
			}
			var out, errOut bytes.Buffer
			code := RunCheckContinuation(root, []string{"feat"}, &out, &errOut)
			if tc.refused {
				if code == 0 || strings.Contains(out.String(), "continuation: required") {
					t.Fatalf("unsafe cursor accepted: %d %s %s", code, &out, &errOut)
				}
				return
			}
			if code != 0 || strings.Contains(out.String(), "continuation: required") != tc.required {
				t.Fatalf("ready verdict: %d %s %s", code, &out, &errOut)
			}
		})
	}
}
