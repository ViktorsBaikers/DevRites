package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func copyGoldenShippableWorkspace(t *testing.T, workspace string) {
	t.Helper()
	golden := filepath.Join("..", "evals", "golden", "shippable-feature")
	entries, err := os.ReadDir(golden)
	if err != nil {
		t.Fatalf("read golden fixture: %v", err)
	}
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		src := filepath.Join(golden, entry.Name())
		dst := filepath.Join(workspace, entry.Name())
		if entry.IsDir() {
			if err := copyDir(src, dst); err != nil {
				t.Fatalf("copy %s: %v", entry.Name(), err)
			}
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read %s: %v", src, err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func TestOrientMatchesObserveSummaryOnGoldenWorkspace(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	slug := "add-csv-export"
	workspace := filepath.Join(root, "work", slug)
	copyGoldenShippableWorkspace(t, workspace)
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte(slug+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_ROOT", project)

	var orientOut, observeOut, stderr bytes.Buffer
	if code := run([]string{"orient", slug}, strings.NewReader(""), &orientOut, &stderr); code != exitOK {
		t.Fatalf("orient code=%d stderr=%q", code, stderr.String())
	}
	stderr.Reset()
	if code := run([]string{"observe", "summary", slug}, strings.NewReader(""), &observeOut, &stderr); code != exitOK {
		t.Fatalf("observe summary code=%d stderr=%q", code, stderr.String())
	}
	if orientOut.String() != observeOut.String() {
		t.Fatalf("orient and observe summary differ:\norient=%q\nobserve=%q", orientOut.String(), observeOut.String())
	}

	var summary map[string]any
	if err := json.Unmarshal(orientOut.Bytes(), &summary); err != nil {
		t.Fatalf("summary JSON: %v", err)
	}
	if summary["slug"] != slug {
		t.Fatalf("slug=%v, want %q", summary["slug"], slug)
	}
	if summary["phase"] != "done" {
		t.Fatalf("phase=%v, want done", summary["phase"])
	}
}

func TestCheckIndexesJSONContract(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	manifestDir := filepath.Join(project, ".claude")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "devrites.manifest"), []byte("devrites\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, ".codegraph"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_ROOT", root)

	var stdout, stderr bytes.Buffer
	if code := run([]string{"check", "indexes", "--root", project}, strings.NewReader(""), &stdout, &stderr); code != exitOK {
		t.Fatalf("indexes code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	var report struct {
		EngineVersion string `json:"engine_version"`
		Manifest      struct {
			Present bool `json:"present"`
		} `json:"manifest"`
		Indexes []struct {
			Path    string `json:"path"`
			Present bool   `json:"present"`
		} `json:"indexes"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("indexes JSON: %v; stdout=%q", err, stdout.String())
	}
	if report.EngineVersion == "" {
		t.Fatal("missing engine_version")
	}
	if !report.Manifest.Present {
		t.Fatal("expected manifest present")
	}
	wantIndexes := []struct {
		path    string
		present bool
	}{
		{".codegraph", true},
		{".code-review-graph", false},
		{".codebase-memory", false},
	}
	if len(report.Indexes) != len(wantIndexes) {
		t.Fatalf("indexes=%d, want %d", len(report.Indexes), len(wantIndexes))
	}
	for i, want := range wantIndexes {
		got := report.Indexes[i]
		if got.Path != want.path || got.Present != want.present {
			t.Fatalf("index[%d]=%+v, want path=%q present=%v", i, got, want.path, want.present)
		}
	}
}

func TestOrientRoutesLikeObserveSummary(t *testing.T) {
	t.Setenv("DEVRITES_ROOT", t.TempDir())
	for _, args := range [][]string{{"orient", "missing"}, {"observe", "summary", "missing"}} {
		t.Run(strings.Join(args, "-"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(args, strings.NewReader(""), &stdout, &stderr)
			if strings.Contains(stderr.String(), "unknown command") || strings.Contains(stderr.String(), "unknown observe") {
				t.Fatalf("%q was not routed: %s", args, stderr.String())
			}
			_ = code
		})
	}
}
