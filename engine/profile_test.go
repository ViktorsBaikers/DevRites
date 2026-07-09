package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitForProfileTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	runCmd(t, dir, "git", args...)
}

func runCmd(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}

func TestProfileCacheHitAndDirtyInvalidation(t *testing.T) {
	repo := t.TempDir()
	gitForProfileTest(t, repo, "init", "-b", "main")
	gitForProfileTest(t, repo, "config", "user.email", "test@example.com")
	gitForProfileTest(t, repo, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitForProfileTest(t, repo, "add", ".")
	gitForProfileTest(t, repo, "commit", "-m", "init")

	cache := t.TempDir()
	t.Setenv("DEVRITES_PROFILE_CACHE", cache)
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}

	var out, errb bytes.Buffer
	if code := run([]string{"profile", "refresh"}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("refresh code %d stderr %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "go.mod") {
		t.Fatalf("profile missing manifest: %s", out.String())
	}

	out.Reset()
	errb.Reset()
	if code := run([]string{"profile", "get"}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("get code %d stderr %s", code, errb.String())
	}
	if !strings.HasPrefix(out.String(), "HIT\n") {
		t.Fatalf("expected HIT, got %s", out.String())
	}

	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errb.Reset()
	if code := run([]string{"profile", "get"}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("dirty get code %d", code)
	}
	if !strings.HasPrefix(out.String(), "MISS\n") {
		t.Fatalf("expected MISS on dirty input, got %s", out.String())
	}
}
