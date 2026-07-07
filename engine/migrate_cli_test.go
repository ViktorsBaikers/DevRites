package main_test

// Issue 06: `migrate` old-layout → new schema, at the CLI seam.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// oldWorkspace writes a pre-v1 .devrites workspace (the flat work/<slug>/ layout)
// with one feature and an ACTIVE pointer, and returns the .devrites root.
func oldWorkspace(t *testing.T) (root, slug string) {
	t.Helper()
	root = filepath.Join(t.TempDir(), ".devrites")
	slug = "legacy-feat"
	work := filepath.Join(root, "work", slug)
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"state.md":     "# State\n\nphase: build\n\nWorking on it.\n",
		"spec.md":      "# Spec\n\nRotate tokens.\n",
		"plan.md":      "# Plan\n\nStep 1, step 2.\n",
		"tasks.md":     "# Tasks\n\n- [x] one\n",
		"evidence.md":  "# Evidence\n\nTests green.\n",
		"decisions.md": "# Decisions\n\nUse HMAC.\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(work, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte(slug+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, slug
}

func backupDirs(t *testing.T, root string) []string {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(root, ".migrate-backup-*"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMigrateProducesNewSchema(t *testing.T) {
	root, slug := oldWorkspace(t)
	out, errOut, code := runDevrites(t, root, "migrate")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if !strings.Contains(out, "migrated 1 feature(s)") {
		t.Errorf("unexpected migrate output\n%s", out)
	}

	feat := filepath.Join(root, "features", slug)
	// evidence.md → proof.md, state.md → status.md, plus a generated feature.md.
	for _, want := range []string{"feature.md", "spec.md", "plan.md", "decisions.md", "tasks.md", "proof.md", "status.md"} {
		if _, err := os.Stat(filepath.Join(feat, want)); err != nil {
			t.Errorf("missing new-schema file %s: %v", want, err)
		}
	}
	fm, err := os.ReadFile(filepath.Join(feat, "feature.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fm), "phase: build") {
		t.Errorf("feature.md phase not derived from state.md\n%s", fm)
	}
	if !strings.Contains(string(fm), "schemaVersion: 1") {
		t.Errorf("feature.md missing schemaVersion\n%s", fm)
	}
	proof, _ := os.ReadFile(filepath.Join(feat, "proof.md"))
	if !strings.Contains(string(proof), "Tests green") {
		t.Errorf("proof.md did not carry evidence.md content\n%s", proof)
	}

	// A backup of the pre-migration state must exist.
	if got := backupDirs(t, root); len(got) != 1 {
		t.Errorf("want exactly one backup dir, got %v", got)
	}
}

func TestMigratePostStatusWorks(t *testing.T) {
	root, slug := oldWorkspace(t)
	if _, _, code := runDevrites(t, root, "migrate"); code != 0 {
		t.Fatal("migrate failed")
	}
	out, errOut, code := runDevrites(t, root, "status", slug)
	if code != 0 {
		t.Fatalf("status after migrate exit = %d (stderr: %s)", code, errOut)
	}
	if !strings.Contains(out, "phase: build") {
		t.Errorf("migrated feature status is wrong\n%s", out)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	root, _ := oldWorkspace(t)
	if _, _, code := runDevrites(t, root, "migrate"); code != 0 {
		t.Fatal("first migrate failed")
	}
	before := backupDirs(t, root)

	out, _, code := runDevrites(t, root, "migrate")
	if code != 0 {
		t.Fatalf("second migrate exit = %d, want 0", code)
	}
	if !strings.Contains(out, "already up to date") {
		t.Errorf("second migrate was not a no-op\n%s", out)
	}
	// A no-op re-run must not create another backup.
	if after := backupDirs(t, root); len(after) != len(before) {
		t.Errorf("re-run created a new backup: before=%v after=%v", before, after)
	}
}
