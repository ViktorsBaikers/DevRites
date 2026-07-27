package lib

import (
	"bytes"
	"strings"
	"testing"
)

func TestDispatchWaiveAcceptsOnlyKnownSingleReason(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := DispatchWaive([]string{"wrong-phase"}, &stdout, &stderr); code != 0 {
		t.Fatalf("valid reason code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "accepted wrong-phase") {
		t.Fatalf("missing receipt: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := DispatchWaive([]string{"anything"}, &stdout, &stderr); code != 2 {
		t.Fatalf("invalid reason code=%d, want 2", code)
	}
}
