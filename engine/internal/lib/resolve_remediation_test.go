package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	mustWrite(t, filepath.Join(root, "ACTIVE"), "feat\n")
	mustWrite(t, filepath.Join(root, "work", "feat", "questions.md"), questions)
	mustWrite(t, filepath.Join(root, "work", "feat", "state.md"), "- Status: running\n- Next step: continue\n")
	return root
}
