package main_test

// Issue 05: `doctor` version triangle, at the CLI seam.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorPrintsTriangle(t *testing.T) {
	root := newWorkspace(t)
	out, errOut, code := runDevrites(t, root, "doctor")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	for _, leg := range []string{"binary:", "pack:", "state-schema:", "verdict:"} {
		if !strings.Contains(out, leg) {
			t.Errorf("doctor output missing %q leg\n%s", leg, out)
		}
	}
}

func TestDoctorRefusesNewerStateSchema(t *testing.T) {
	root := newWorkspace(t)
	// Drop in a feature declaring a far-newer schema than the binary supports.
	dir := filepath.Join(root, "features", "from-the-future")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	fm := "---\nslug: from-the-future\nphase: build\nschemaVersion: 99\n---\n\nx\n"
	if err := os.WriteFile(filepath.Join(dir, "feature.md"), []byte(fm), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code := runDevrites(t, root, "doctor")
	if code != 3 {
		t.Fatalf("exit = %d, want 3 (refuse) for a newer-major state schema\n%s", code, out)
	}
	if !strings.Contains(out, "REFUSE") {
		t.Errorf("verdict not a REFUSE\n%s", out)
	}
}
