package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/devritespaths"
)

func TestEnvironmentReportForDetectsManifestAndIndexes(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, devritespaths.DevritesRootName)
	manifestPath := filepath.Join(project, devritespaths.ManifestName)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte("devrites\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, ".codegraph"), 0o755); err != nil {
		t.Fatal(err)
	}

	report, err := EnvironmentReportFor(root, project)
	if err != nil {
		t.Fatal(err)
	}
	if report.ProjectRoot != project {
		t.Fatalf("project root = %q, want %q", report.ProjectRoot, project)
	}
	if !report.Manifest.Present {
		t.Fatal("expected manifest present")
	}
	if len(report.Indexes) != 3 {
		t.Fatalf("indexes = %d, want 3", len(report.Indexes))
	}
	if !report.Indexes[0].Present {
		t.Fatalf("expected .codegraph present, got %+v", report.Indexes)
	}
}

func TestRunEnvironmentCheckWritesJSON(t *testing.T) {
	project := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := RunEnvironmentCheck("", []string{"--root", project}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"engine_version"`) || !strings.Contains(stdout.String(), `"manifest"`) {
		t.Fatalf("stdout=%q", stdout.String())
	}
}
