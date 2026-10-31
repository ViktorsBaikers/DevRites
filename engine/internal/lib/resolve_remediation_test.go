package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/testutil"
)

func TestResolveTreatsQuestionIDAsLiteralText(t *testing.T) {
	root := resolveWorkspace(t, "## q-1.* blocking\nstatus: open\n")
	var stdout, stderr bytes.Buffer

	if code := Resolve(root, []string{"q-1[", "answer"}, &stdout, &stderr); code != 3 {
		t.Fatalf("code = %d, want 3; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "qid not found") {
		t.Fatalf("stderr = %q, want qid-not-found error", stderr.String())
	}
}

func TestResolveCompletesMissingAnswerFieldsInSingleRewrite(t *testing.T) {
	root := resolveWorkspace(t, "## q-1 blocking\nstatus: open\n")
	var stdout, stderr bytes.Buffer

	if code := Resolve(root, []string{"q-1", "literal answer"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(root, "work", "feat", "questions.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"status: answered", "answered_at:", "answer: literal answer"} {
		if !strings.Contains(text, want) {
			t.Fatalf("questions.md missing %q:\n%s", want, text)
		}
	}
}

func TestResolveAcceptsCanonicalUppercaseQuestionID(t *testing.T) {
	root := resolveWorkspace(t, "## Q-001 blocking\nstatus: open\n")
	var stdout, stderr bytes.Buffer

	if code := Resolve(root, []string{"Q-001", "canonical answer"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
}

func TestResolveIgnoresFencedAwaitingHumanExamples(t *testing.T) {
	for _, wait := range []string{"cursor", "section"} {
		for _, qid := range []string{"Q-001", "Q-002"} {
			t.Run(wait+"/"+qid, func(t *testing.T) {
				root := resolveWorkspace(t, "## Q-001\nstatus: open\n\n## Q-002\nstatus: open\n")
				example := "## Notes\n```md\n## Awaiting human\n- qid: Q-001\n## Log\nexample log\n```\n"
				text := "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | build |\n| status | awaiting_human |\n| next_action | wait |\n| schema | 4 |\n"
				if wait == "cursor" {
					text += "| question_id | Q-002 |\n"
				}
				text += "\n" + example
				if wait == "section" {
					text += "\n## Awaiting human\n- qid: Q-002\n"
				}
				text += "\n## Log\n- kept\n"
				path := filepath.Join(root, "work", "feat", "state.md")
				testutil.WriteFile(t, path, text)
				var out, errOut bytes.Buffer
				if code := Resolve(root, []string{qid, "answer"}, &out, &errOut); code != 0 {
					t.Fatalf("exit %d: %s", code, errOut.String())
				}
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				got := string(raw)
				if qid == "Q-001" {
					if got != text {
						t.Fatalf("resolving other question changed live wait:\n%s", got)
					}
				} else if !strings.Contains(got, "| status | running |") || strings.Contains(got, "| question_id |") || strings.Contains(got, "- qid: Q-002") || !strings.Contains(got, "resolved Q-002") {
					t.Fatalf("genuine wait did not resume:\n%s", got)
				}
				if !strings.Contains(got, example) {
					t.Fatalf("fenced example changed:\n%s", got)
				}
			})
		}
	}
}

func resolveWorkspace(t *testing.T, questions string) string {
	t.Helper()
	root := t.TempDir()
	testutil.WriteFile(t, filepath.Join(root, "ACTIVE"), "feat\n")
	testutil.WriteFile(t, filepath.Join(root, "work", "feat", "questions.md"), questions)
	testutil.WriteFile(t, filepath.Join(root, "work", "feat", "state.md"), "- Status: running\n- Next step: continue\n- Schema: 4\n")
	return root
}

func TestResolveHumanFlagRecordsResolverIdentity(t *testing.T) {
	root := resolveWorkspace(t, "## q-1 blocking\nstatus: open\n")
	var stdout, stderr bytes.Buffer

	if code := Resolve(root, []string{"--human", "q-1", "yes"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(root, "work", "feat", "questions.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "answered_by: human\n") {
		t.Fatalf("questions.md missing answered_by: human:\n%s", data)
	}
}

func TestResolveWithoutHumanFlagDropsPreSeededIdentity(t *testing.T) {
	root := resolveWorkspace(t, "## q-1 blocking\nstatus: open\nanswered_by: human\n")
	var stdout, stderr bytes.Buffer

	if code := Resolve(root, []string{"q-1", "yes"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(root, "work", "feat", "questions.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "answered_by") {
		t.Fatalf("forged answered_by survived an unflagged resolve:\n%s", data)
	}
}

func TestResolveWithoutHumanFlagDropsVariantPreSeededIdentity(t *testing.T) {
	for _, seed := range []string{"  answered_by: human", "Answered_By: human", "\tANSWERED_BY: human"} {
		root := resolveWorkspace(t, "## q-1 blocking\nstatus: open\n"+seed+"\n")
		var stdout, stderr bytes.Buffer

		if code := Resolve(root, []string{"q-1", "yes"}, &stdout, &stderr); code != 0 {
			t.Fatalf("%q: code = %d, want 0; stderr=%q", seed, code, stderr.String())
		}
		if text := readQuestions(t, root); strings.Contains(strings.ToLower(text), "answered_by") {
			t.Fatalf("%q: forged answered_by survived an unflagged resolve:\n%s", seed, text)
		}
	}
}

func TestResolveHumanFlagKeepsSingleCanonicalIdentity(t *testing.T) {
	root := resolveWorkspace(t, "## q-1 blocking\nstatus: open\n  answered_by: agent\nAnswered_By: agent\n")
	var stdout, stderr bytes.Buffer

	if code := Resolve(root, []string{"--human", "q-1", "yes"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
	text := readQuestions(t, root)
	if strings.Count(strings.ToLower(text), "answered_by") != 1 || !strings.Contains(text, "\nanswered_by: human\n") {
		t.Fatalf("want exactly one canonical answered_by: human:\n%s", text)
	}
}

func TestResolveDropHonorsHumanFlag(t *testing.T) {
	seeded := "## q-1 blocking\nstatus: open\n  answered_by: human\n"
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"--human", "--drop", "q-1", "obsolete"}, true},
		{[]string{"--drop", "q-1", "obsolete"}, false},
	} {
		root := resolveWorkspace(t, seeded)
		var stdout, stderr bytes.Buffer

		if code := Resolve(root, tc.args, &stdout, &stderr); code != 0 {
			t.Fatalf("%v: code = %d, want 0; stderr=%q", tc.args, code, stderr.String())
		}
		text := readQuestions(t, root)
		if got := strings.Contains(text, "\nanswered_by: human\n"); got != tc.want || !strings.Contains(text, "status: dropped") {
			t.Fatalf("%v: canonical answered_by = %v, want %v:\n%s", tc.args, got, tc.want, text)
		}
		if !tc.want && strings.Contains(text, "answered_by") {
			t.Fatalf("%v: answered_by survived:\n%s", tc.args, text)
		}
	}
}

func TestResolveBatchHonorsHumanFlag(t *testing.T) {
	questions := "## q-1 blocking\nstatus: open\n  answered_by: human\n## q-2 blocking\nstatus: open\n"
	batch := "q-1: yes\n--drop q-2: obsolete\n"
	for _, tc := range []struct {
		args []string
		want int
	}{
		{[]string{"--human"}, 2},
		{nil, 0},
	} {
		root := resolveWorkspace(t, questions)
		file := filepath.Join(t.TempDir(), "batch.txt")
		testutil.WriteFile(t, file, batch)
		var stdout, stderr bytes.Buffer

		if code := Resolve(root, append(tc.args, "--batch", file), &stdout, &stderr); code != 0 {
			t.Fatalf("%v: code = %d, want 0; stderr=%q", tc.args, code, stderr.String())
		}
		text := readQuestions(t, root)
		if got := strings.Count(text, "\nanswered_by: human\n"); got != tc.want || (tc.want == 0 && strings.Contains(text, "answered_by")) {
			t.Fatalf("%v: answered_by count = %d, want %d:\n%s", tc.args, got, tc.want, text)
		}
	}
}

func readQuestions(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "work", "feat", "questions.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestResolveBatchAppliesUnterminatedFinalLine(t *testing.T) {
	root := resolveWorkspace(t, "## q-1 blocking\nstatus: open\n## q-2 blocking\nstatus: open\n")
	file := filepath.Join(t.TempDir(), "batch.txt")
	testutil.WriteFile(t, file, "q-1: yes\n--drop q-2: obsolete")
	var stdout, stderr bytes.Buffer

	if code := Resolve(root, []string{"--batch", file}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if text := readQuestions(t, root); strings.Contains(text, "status: open") {
		t.Fatalf("a question stayed open:\n%s", text)
	}
}

func TestResolveAcceptsCRLFQuestions(t *testing.T) {
	root := resolveWorkspace(t, "## q-1 blocking\r\nstatus: open\r\n")
	var stdout, stderr bytes.Buffer

	if code := Resolve(root, []string{"q-1", "postgres"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if text := readQuestions(t, root); !strings.Contains(text, "status: answered") {
		t.Fatalf("question not answered:\n%s", text)
	}
}

func TestResolveHumanFlagAnywhere(t *testing.T) {
	root := resolveWorkspace(t, "## q-5 blocking\nstatus: open\n")
	var stdout, stderr bytes.Buffer

	if code := Resolve(root, []string{"q-5", "ans", "--human"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if text := readQuestions(t, root); !strings.Contains(text, "answered_by: human") || !strings.Contains(text, "answer: ans") {
		t.Fatalf("trailing --human not honoured:\n%s", text)
	}
}

func TestResolveRejectsMalformedArguments(t *testing.T) {
	for _, args := range [][]string{
		{"q-1", "--human"},
		{"q-1", "ans", "extra"},
		{"--dorp", "q-1"},
		{"q-1", "--force"},
		{"--drop", "q-1", "reason", "extra"},
		{"--drop", "--human-ish"},
	} {
		root := resolveWorkspace(t, "## q-1 blocking\nstatus: open\n")
		var stdout, stderr bytes.Buffer
		if code := Resolve(root, args, &stdout, &stderr); code != 5 {
			t.Fatalf("%v: code = %d, want 5; stderr=%q", args, code, stderr.String())
		}
		if text := readQuestions(t, root); !strings.Contains(text, "status: open") {
			t.Fatalf("%v: question mutated:\n%s", args, text)
		}
	}
}

func TestResolveBatchIsAllOrNothing(t *testing.T) {
	for _, tc := range []struct {
		batch string
		want  int
	}{
		{"q-1: yes\nq-2 no colon\n", 5},
		{"q-1: yes\n--drop q-2\n", 5},
		{"q-1: yes\nq-9: unknown\n", 3},
		{"q-1: yes\nq-1: again\n", 4},
	} {
		root := resolveWorkspace(t, "## q-1 blocking\nstatus: open\n## q-2 blocking\nstatus: open\n")
		file := filepath.Join(t.TempDir(), "batch.txt")
		testutil.WriteFile(t, file, tc.batch)
		var stdout, stderr bytes.Buffer
		if code := Resolve(root, []string{"--batch", file}, &stdout, &stderr); code != tc.want {
			t.Fatalf("%q: code = %d, want %d; stderr=%q", tc.batch, code, tc.want, stderr.String())
		}
		if text := readQuestions(t, root); strings.Count(text, "status: open") != 2 {
			t.Fatalf("%q: batch partially applied:\n%s", tc.batch, text)
		}
	}
}

func TestResolveQuestionBlockEndsAtAnyLevelTwoHeading(t *testing.T) {
	root := resolveWorkspace(t, "## q-1 blocking\nstatus: open\n\n## Notes\nanswer: keep me\n")
	var stdout, stderr bytes.Buffer
	if code := Resolve(root, []string{"q-1", "yes"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if text := readQuestions(t, root); !strings.Contains(text, "## Notes\nanswer: keep me\n") || !strings.Contains(text, "answer: yes\n## Notes") {
		t.Fatalf("answer written outside the question block:\n%s", text)
	}

	root = resolveWorkspace(t, "## q-1 blocking\n\n## Notes\nstatus: draft\n")
	stderr.Reset()
	if code := Resolve(root, []string{"q-1", "yes"}, &stdout, &stderr); code != 0 {
		t.Fatalf("a status line under ## Notes blocked q-1: code = %d; stderr=%q", code, stderr.String())
	}
	if text := readQuestions(t, root); !strings.Contains(text, "## Notes\nstatus: draft\n") {
		t.Fatalf("## Notes status rewritten:\n%s", text)
	}
}

func TestResolveAgreesWithGateOnStatusSpelling(t *testing.T) {
	for _, status := range []string{"status: Open", "status: open ", "Status: OPEN\t"} {
		body := "## q-1 blocking\n" + status + "\ngate: blocking\n"
		if gates := gate.OpenBlockingQuestionGates([]byte(body)); len(gates) != 1 {
			t.Fatalf("%q: gate sees %v, want one open gate", status, gates)
		}
		root := resolveWorkspace(t, body)
		var stdout, stderr bytes.Buffer
		if code := Resolve(root, []string{"q-1", "yes"}, &stdout, &stderr); code != 0 {
			t.Fatalf("%q: code = %d, want 0; stderr=%q", status, code, stderr.String())
		}
		if gates := gate.OpenBlockingQuestionGates([]byte(readQuestions(t, root))); len(gates) != 0 {
			t.Fatalf("%q: gate still open after resolve: %v\n%s", status, gates, readQuestions(t, root))
		}
	}
}

func TestResolveBatchToleratesCRLFAndWhitespaceLines(t *testing.T) {
	root := resolveWorkspace(t, "## q-1 blocking\nstatus: open\n## q-2 blocking\nstatus: open\n")
	file := filepath.Join(t.TempDir(), "batch.txt")
	testutil.WriteFile(t, file, "  q-1: yes\r\n   \r\n\r\n\t--drop q-2: obsolete\r\n")
	var stdout, stderr bytes.Buffer
	if code := Resolve(root, []string{"--batch", file}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if text := readQuestions(t, root); strings.Contains(text, "status: open") || !strings.Contains(text, "answer: yes\n") {
		t.Fatalf("batch not applied cleanly:\n%s", text)
	}
}

func TestResolveSkipsFencedExampleQuestionBlocks(t *testing.T) {
	fence := "```md\n## Q-001 blocking\nstatus: open\nanswer:\n```\n"
	body := "# Questions\n\nExample:\n\n" + fence + "\n## Q-001 blocking\nstatus: open\ngate: blocking\n"
	for _, args := range [][]string{{"Q-001", "yes"}, {"--drop", "Q-001", "obsolete"}, {"--batch", ""}} {
		root := resolveWorkspace(t, body)
		if args[0] == "--batch" {
			args[1] = filepath.Join(t.TempDir(), "batch.txt")
			testutil.WriteFile(t, args[1], "Q-001: yes\n")
		}
		var stdout, stderr bytes.Buffer
		if code := Resolve(root, args, &stdout, &stderr); code != 0 {
			t.Fatalf("%v: code = %d, want 0; stderr=%q", args, code, stderr.String())
		}
		text := readQuestions(t, root)
		if !strings.Contains(text, "Example:\n\n"+fence+"\n") {
			t.Fatalf("%v: fenced example changed:\n%s", args, text)
		}
		if gates := gate.OpenBlockingQuestionGates([]byte(text)); len(gates) != 0 {
			t.Fatalf("%v: real question left open (%v):\n%s", args, gates, text)
		}
	}
}
