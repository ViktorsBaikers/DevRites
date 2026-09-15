package parallel

import (
	"bytes"
	"strings"
	"testing"
)

func TestParallelCreateHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run("parallel", []string{"create", "--help"}, strings.NewReader(""), &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr=%q", stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, usageCreate) {
		t.Fatalf("stdout=%q, want %q", got, usageCreate)
	}
}

func TestParallelCreateUnknownArgumentStillErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run("parallel", []string{"create", "--nope"}, strings.NewReader(""), &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("code=%d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr.String(), `unknown argument "--nope"`) {
		t.Fatalf("stderr=%q", stderr.String())
	}
}
