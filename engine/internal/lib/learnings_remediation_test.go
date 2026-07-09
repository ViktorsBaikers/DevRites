package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLearningsAddReportsPersistenceFailure(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "not-a-directory")
	if err := os.WriteFile(root, []byte("occupied"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	if code := Learnings(root, []string{"add", "feat", "lesson"}, &stdout, &stderr); code == 0 {
		t.Fatalf("code = 0, want failure; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no success message", stdout.String())
	}
}
