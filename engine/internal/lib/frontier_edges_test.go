package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFrontierContinuationPreparesRecordedSuccessorBeforeShip(t *testing.T) {
	for _, tc := range []struct {
		name, status, policy, remaining string
		blockingQuestion, existingChild bool
		wantBeta                        string
	}{
		{"authorized missing successor", "running", "", "1", false, false, "missing"},
		{"sequence opted out", "running", "continue_sequence: false\n", "1", false, false, ""},
		{"sequence budget exhausted", "running", "", "0", false, false, ""},
		{"human gate blocks missing successor", "awaiting_human", "", "1", false, false, ""},
		{"human gate retains existing child preparation", "awaiting_human", "", "1", false, true, "live"},
		{"blocking question blocks missing successor", "running", "", "1", true, false, ""},
		{"blocking question retains existing child preparation", "running", "", "1", true, true, "live"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			alpha := frontierFixture(t, root, "work", "alpha", "", "seal", tc.status, "/rite-ship alpha", 1, tc.remaining)
			writeFile(t, filepath.Join(alpha, "decisions.md"), "# Decisions\n\n## Accepted sequence\n\nApproved sequence:\n\n| Position | Slug | Objective |\n| --- | --- | --- |\n| 1 | alpha | Complete first workspace |\n| 2 | beta | Prepare independent next workspace |\n")
			writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 5\n"+tc.policy)
			writeFile(t, filepath.Join(root, "ACTIVE"), "alpha\n")
			if tc.blockingQuestion {
				writeFile(t, filepath.Join(alpha, "questions.md"), "## Q-001\nstatus: open\ngate: blocking\n")
			}
			if tc.existingChild {
				frontierFixture(t, root, "work", "beta", "alpha", "plan", "running", "/rite-vet beta", 2, "0")
			}
			before := frontierSnapshot(t, root)
			var out, errOut bytes.Buffer
			if code := RunCheckContinuation(root, []string{"alpha"}, &out, &errOut); code != 0 {
				t.Fatalf("code=%d: %s", code, &errOut)
			}
			text := out.String()
			sourceWait := "frontier-workspace: alpha location=live action=wait "
			if tc.status == "awaiting_human" {
				sourceWait += "reason=human/blocked"
			} else if tc.blockingQuestion {
				sourceWait += "reason=open blocking question"
			}
			for _, want := range []string{"frontier-source: alpha\n", sourceWait} {
				if !strings.Contains(text, want) {
					t.Errorf("missing %q:\n%s", want, text)
				}
			}
			if strings.Contains(text, "/rite-ship") {
				t.Errorf("observation offered Ship invocation:\n%s", text)
			}
			if tc.wantBeta != "" {
				for _, want := range []string{"continuation: required\n", "invoke: /rite-autocomplete\n", "source: alpha\n", "frontier: actionable\n", "frontier-workspace: beta location=" + tc.wantBeta + " action=prepare "} {
					if !strings.Contains("\n"+text, "\n"+want) {
						t.Errorf("missing %q:\n%s", want, text)
					}
				}
			} else if !strings.HasPrefix(text, "continuation: clear\n") || strings.Contains(text, "\ninvoke:") || strings.Contains(text, "frontier-workspace: beta location=missing action=prepare ") {
				t.Errorf("source prohibition overridden:\n%s", text)
			}
			if tc.wantBeta == "missing" {
				if !strings.Contains(text, "\nfrontier-prepare: /rite-spec 'beta' source=alpha ") {
					t.Errorf("missing native successor preparation:\n%s", text)
				}
			} else if strings.Contains(text, "frontier-prepare:") {
				t.Errorf("unauthorized missing-workspace preparation:\n%s", text)
			}
			if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
				t.Error("continuation observation changed workflow files")
			}
			if !tc.existingChild {
				if _, err := os.Stat(filepath.Join(root, "work", "beta")); !os.IsNotExist(err) {
					t.Errorf("continuation observation created beta: %v", err)
				}
			}
		})
	}
}

func TestFrontierTerminalQuestionsBlockMissingSuccessorNotExistingChild(t *testing.T) {
	for _, location := range []string{"work", "archive"} {
		for _, gate := range []string{"blocking", "escalating"} {
			for _, child := range []string{"missing", "existing"} {
				t.Run(location+"/"+gate+"/"+child, func(t *testing.T) {
					root := filepath.Join(t.TempDir(), ".devrites")
					status := "running"
					if location == "archive" {
						status = "shipped"
					}
					alpha := frontierFixture(t, root, location, "alpha", "", "done", status, "/rite-spec beta", 1, "1")
					writeFile(t, filepath.Join(alpha, "questions.md"), "## Q-001\nstatus: open\ngate: "+gate+"\n")
					writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 5\n")
					if child == "existing" {
						frontierFixture(t, root, "work", "beta", "alpha", "plan", "running", "/rite-vet beta", 2, "0")
					}
					before := frontierSnapshot(t, root)
					var out, errOut bytes.Buffer
					run := RunCheckContinuation
					if location == "archive" {
						run = RunFrontier
					}
					if code := run(root, []string{"alpha"}, &out, &errOut); code != 0 {
						t.Fatalf("code=%d: %s", code, &errOut)
					}
					text := out.String()
					if strings.Contains(text, "frontier-prepare:") || strings.Contains(text, "beta location=missing action=prepare") || strings.Contains(text, "\ninvoke: /rite-spec") {
						t.Errorf("terminal workspace bypassed open %s question:\n%s", gate, text)
					}
					if child == "existing" {
						wants := []string{"frontier: actionable\n", "frontier-source: alpha\n", "frontier-workspace: beta location=live action=prepare "}
						if location == "work" {
							wants = append(wants, "continuation: required\n", "invoke: /rite-autocomplete\n", "source: alpha\n")
						}
						for _, want := range wants {
							if !strings.Contains("\n"+text, "\n"+want) {
								t.Errorf("ancestor question hid existing child; missing %q:\n%s", want, text)
							}
						}
					} else {
						verdict := "continuation: clear\n"
						if location == "archive" {
							verdict = "frontier: blocked\n"
						}
						if !strings.HasPrefix(text, verdict) || strings.Contains(text, "\ninvoke:") {
							t.Errorf("question-gated missing child was actionable:\n%s", text)
						}
						if _, err := os.Stat(filepath.Join(root, "work", "beta")); !os.IsNotExist(err) {
							t.Errorf("continuation observation created beta: %v", err)
						}
					}
					if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
						t.Error("continuation observation changed workflow files")
					}
				})
			}
		}
	}
}

func TestFrontierExistingSuccessorSpecSurvivesFuturePreparationLimits(t *testing.T) {
	for _, tc := range []struct {
		name, remaining, policy string
	}{
		{"exhausted", "0", ""},
		{"opted out", "1", "continue_sequence: false\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			frontierFixture(t, root, "work", "alpha", "", "done", "running", "/rite-spec beta", 1, tc.remaining)
			frontierFixture(t, root, "work", "beta", "alpha", "plan", "running", "/rite-vet beta", 2, "0")
			writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 5\n"+tc.policy)
			before := frontierSnapshot(t, root)
			var out, errOut bytes.Buffer
			if code := RunCheckContinuation(root, []string{"alpha"}, &out, &errOut); code != 0 {
				t.Fatalf("existing bound successor rejected, code=%d: %s", code, &errOut)
			}
			text := out.String()
			for _, want := range []string{"continuation: required\n", "invoke: /rite-autocomplete\n", "source: alpha\n", "frontier: actionable\n", "frontier-workspace: beta location=live action=prepare "} {
				if !strings.Contains("\n"+text, "\n"+want) {
					t.Errorf("future preparation limit hid existing child; missing %q:\n%s", want, text)
				}
			}
			if strings.Contains(text, "frontier-prepare:") || strings.Contains(text, "location=missing action=prepare") || strings.Contains(text, "\ninvoke: /rite-spec") {
				t.Errorf("existing child cursor authorized missing-workspace preparation:\n%s", text)
			}
			if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
				t.Error("continuation observation changed workflow files")
			}
		})
	}
}

func TestFrontierConflictingMapsPreserveExistingSuccessorOnly(t *testing.T) {
	for _, child := range []string{"existing", "missing"} {
		t.Run(child, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			alpha := frontierFixture(t, root, "work", "alpha", "", "done", "running", "/rite-spec beta", 1, "1")
			writeFile(t, filepath.Join(alpha, "decisions.md"), "# Sequence\n\n| Position | Slug |\n| --- | --- |\n| 1 | alpha |\n| 2 | beta |\n\nAccepted amendment\n\n| Position | Slug |\n| --- | --- |\n| 1 | alpha |\n| 2 | gamma |\n")
			writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 5\n")
			if child == "existing" {
				frontierFixture(t, root, "work", "beta", "alpha", "plan", "running", "/rite-vet beta", 2, "0")
			}
			before := frontierSnapshot(t, root)
			var out, errOut bytes.Buffer
			code := RunCheckContinuation(root, []string{"alpha"}, &out, &errOut)
			text := out.String()
			if child == "existing" {
				if code != 0 {
					t.Fatalf("conflicting opening authority rejected existing child, code=%d: %s", code, &errOut)
				}
				for _, want := range []string{"continuation: required\n", "invoke: /rite-autocomplete\n", "source: alpha\n", "frontier: actionable\n", "frontier-workspace: beta location=live action=prepare ", "frontier-review-sequence: "} {
					if !strings.Contains("\n"+text, "\n"+want) {
						t.Errorf("conflicting maps hid existing child or normalization; missing %q:\n%s", want, text)
					}
				}
			} else {
				if code != 0 && code != 2 {
					t.Errorf("unexpected continuation exit %d: %s", code, &errOut)
				}
				if _, err := os.Stat(filepath.Join(root, "work", "beta")); !os.IsNotExist(err) {
					t.Errorf("continuation observation created beta: %v", err)
				}
			}
			if strings.Contains(text, "frontier-prepare:") || strings.Contains(text, "location=missing action=prepare") || strings.Contains(text, "\ninvoke: /rite-spec") || strings.Contains(text, "\ninvoke: /rite-build") {
				t.Errorf("conflicting maps authorized new preparation or direct execution:\n%s", text)
			}
			if !reflect.DeepEqual(before, frontierSnapshot(t, root)) {
				t.Error("continuation observation changed workflow files")
			}
		})
	}
}
