package main_test

// Issue 06: `migrate` workspace normalization at the CLI seam.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// workWorkspace writes a canonical .devrites/work/<slug> workspace with one
// feature and an ACTIVE pointer, and returns the .devrites root.
func workWorkspace(t *testing.T) (root, slug string) {
	t.Helper()
	root = filepath.Join(t.TempDir(), ".devrites")
	slug = "legacy-feat"
	work := filepath.Join(root, "work", slug)
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"state.md":     "# State\n\n| Key | Value |\n| --- | --- |\n| phase | temper |\n| status | running |\n",
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

func TestMigrateNormalizesCanonicalWorkSchema(t *testing.T) {
	root, slug := workWorkspace(t)
	out, errOut, code := runDevrites(t, root, "migrate")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if !strings.Contains(out, "migrated 1 feature(s)") {
		t.Errorf("unexpected migrate output\n%s", out)
	}

	feat := filepath.Join(root, "work", slug)
	// Current canonical files stay canonical; migration adds only the missing map.
	for _, want := range []string{"README.md", "state.md", "spec.md", "plan.md", "decisions.md", "tasks.md", "evidence.md"} {
		if _, err := os.Stat(filepath.Join(feat, want)); err != nil {
			t.Errorf("missing new-schema file %s: %v", want, err)
		}
	}
	readme, err := os.ReadFile(filepath.Join(feat, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "phase: temper") {
		t.Errorf("README.md phase not derived from state.md\n%s", readme)
	}
	if !strings.Contains(string(readme), "schemaVersion: 1") {
		t.Errorf("README.md missing schemaVersion\n%s", readme)
	}
	for _, unwanted := range []string{"feature.md", "proof.md", "status.md"} {
		if _, err := os.Stat(filepath.Join(feat, unwanted)); !os.IsNotExist(err) {
			t.Errorf("migration created legacy alias %s: %v", unwanted, err)
		}
	}

	// A backup of the pre-migration state must exist.
	if got := backupDirs(t, root); len(got) != 1 {
		t.Errorf("want exactly one backup dir, got %v", got)
	}
}

func TestMigratePostStatusWorks(t *testing.T) {
	root, slug := workWorkspace(t)
	if _, _, code := runDevrites(t, root, "migrate"); code != 0 {
		t.Fatal("migrate failed")
	}
	out, errOut, code := runDevrites(t, root, "status", slug)
	if code != 0 {
		t.Fatalf("status after migrate exit = %d (stderr: %s)", code, errOut)
	}
	if !strings.Contains(out, "phase: temper") {
		t.Errorf("migrated feature status is wrong\n%s", out)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	root, _ := workWorkspace(t)
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

func TestMigrateNormalizesLiveFeatureInPlace(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	slug := "live-feat"
	feat := filepath.Join(root, "features", slug)
	if err := os.MkdirAll(feat, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"feature.md":   "# Legacy feature map\n",
		"status.md":    "# State\n\n- Phase: prove\n- Status: running\n",
		"spec.md":      "# Spec\n\nRotate tokens.\n",
		"plan.md":      "# Plan\n\nStep 1, step 2.\n",
		"tasks.md":     "# Tasks\n\n- [x] one\n",
		"proof.md":     "# Evidence\n\nTests green.\n",
		"decisions.md": "# Decisions\n\nUse HMAC.\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(feat, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	out, errOut, code := runDevrites(t, root, "migrate")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if !strings.Contains(out, "migrated 1 feature(s)") {
		t.Errorf("unexpected migrate output\n%s", out)
	}
	for _, want := range []string{"README.md", "feature.md", "proof.md", "status.md", "state.md", "evidence.md"} {
		if _, err := os.Stat(filepath.Join(feat, want)); err != nil {
			t.Errorf("missing normalized file %s: %v", want, err)
		}
	}
	readme, err := os.ReadFile(filepath.Join(feat, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "Legacy feature map") {
		t.Errorf("README.md did not carry feature.md content\n%s", readme)
	}
	stateFile, _ := os.ReadFile(filepath.Join(feat, "state.md"))
	if !strings.Contains(string(stateFile), "running") {
		t.Errorf("state.md did not carry status.md content\n%s", stateFile)
	}
	evidence, _ := os.ReadFile(filepath.Join(feat, "evidence.md"))
	if !strings.Contains(string(evidence), "Tests green") {
		t.Errorf("evidence.md did not carry proof.md content\n%s", evidence)
	}
	if got := backupDirs(t, root); len(got) != 1 {
		t.Errorf("want exactly one backup dir, got %v", got)
	}

	out, errOut, code = runDevrites(t, root, "status", slug)
	if code != 0 {
		t.Fatalf("status after normalize exit = %d (stderr: %s)", code, errOut)
	}
	if !strings.Contains(out, "phase: prove") || !strings.Contains(out, "proof      present") || !strings.Contains(out, "status     present") {
		t.Errorf("normalized feature status is wrong\n%s", out)
	}

	before := backupDirs(t, root)
	out, _, code = runDevrites(t, root, "migrate")
	if code != 0 {
		t.Fatalf("second migrate exit = %d, want 0", code)
	}
	if !strings.Contains(out, "already up to date") {
		t.Errorf("second migrate was not a no-op\n%s", out)
	}
	if after := backupDirs(t, root); len(after) != len(before) {
		t.Errorf("re-run created a new backup: before=%v after=%v", before, after)
	}
}
