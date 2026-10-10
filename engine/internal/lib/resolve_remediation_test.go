package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
