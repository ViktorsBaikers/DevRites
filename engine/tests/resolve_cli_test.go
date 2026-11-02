package main_test

import "testing"

func TestResolveCLIRequiresWorkspace(t *testing.T) {
	// An absent selected workspace is exit 2, not an unsafe explicit-root error.
	project := t.TempDir()
	out, code := runArgv(t, project, libEnv, "", binPath, "state", "resolve", "Q-001", "approved")
	if code != 2 {
		t.Fatalf("resolve without workspace: exit=%d stdout=%q", code, out)
	}
}
