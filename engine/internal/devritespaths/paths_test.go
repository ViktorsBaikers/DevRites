package devritespaths

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceOverrideCheckedRequiresPhysicalRootContainment(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, DevritesRootName)
	workspace := filepath.Join(root, "work", "feature")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DEVRITES_WORKSPACE", workspace)
	got, err := WorkspaceOverrideChecked(root, "feature")
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("WorkspaceOverrideChecked() = %q, want canonical %q", got, want)
	}
	t.Setenv("DEVRITES_WORKSPACE", filepath.Join(DevritesRootName, "work", "feature"))
	if got, err := WorkspaceOverrideChecked(root, "feature"); err != nil || got != want {
		t.Fatalf("relative WorkspaceOverrideChecked() = %q, %v; want %q", got, err, want)
	}

	outside := filepath.Join(t.TempDir(), "feature")
	t.Setenv("DEVRITES_WORKSPACE", outside)
	if _, err := WorkspaceOverrideChecked(root, "feature"); err == nil || !strings.Contains(err.Error(), "DRV-WORKSPACE-OUTSIDE-ROOT") {
		t.Fatalf("outside workspace error = %v, want stable containment diagnostic", err)
	}
	if got := WorkspaceOverride(root, "feature"); got != "" {
		t.Fatalf("unsafe compatibility result = %q, want empty", got)
	}

	t.Setenv("DEVRITES_WORKSPACE", root)
	if _, err := WorkspaceOverrideChecked(root, ""); err == nil || !strings.Contains(err.Error(), "DRV-WORKSPACE-INVALID") {
		t.Fatalf("root-as-workspace error = %v", err)
	}

	arbitrary := filepath.Join(root, "extensions", "feature")
	if err := os.MkdirAll(arbitrary, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_WORKSPACE", arbitrary)
	if _, err := WorkspaceOverrideChecked(root, ""); err == nil || !strings.Contains(err.Error(), "DRV-WORKSPACE-INVALID") {
		t.Fatalf("non-feature path error = %v", err)
	}
}

func TestWorkspaceOverrideCheckedRejectsSymlinkEscapeAndSlugMismatch(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, DevritesRootName)
	if err := os.MkdirAll(filepath.Join(root, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	link := filepath.Join(root, "work", "feature")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Setenv("DEVRITES_WORKSPACE", link)
	if _, err := WorkspaceOverrideChecked(root, "feature"); err == nil || !strings.Contains(err.Error(), "DRV-WORKSPACE-OUTSIDE-ROOT") {
		t.Fatalf("symlink escape error = %v", err)
	}

	inside := filepath.Join(root, "work", "other")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_WORKSPACE", inside)
	if _, err := WorkspaceOverrideChecked(root, "feature"); err == nil || !strings.Contains(err.Error(), "DRV-WORKSPACE-SLUG-MISMATCH") {
		t.Fatalf("slug mismatch error = %v", err)
	}
}

func TestActiveSlugRejectsPathTraversal(t *testing.T) {
	root := filepath.Join(t.TempDir(), DevritesRootName)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte("../../outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ActiveSlug(root); err == nil || !strings.Contains(err.Error(), "DRV-ACTIVE-INVALID") {
		t.Fatalf("ActiveSlug error = %v, want stable invalid-pointer diagnostic", err)
	}

	if err := os.Remove(filepath.Join(root, "ACTIVE")); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "ACTIVE")
	if err := os.WriteFile(target, []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "ACTIVE")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := ActiveSlug(root); err == nil || !strings.Contains(err.Error(), "DRV-ACTIVE-SYMLINK") {
		t.Fatalf("ActiveSlug symlink error = %v, want stable pointer diagnostic", err)
	}
}
