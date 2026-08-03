package lib

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGitCommandIgnoresInheritedRepositoryTargets(t *testing.T) {
	target := initGitRepository(t, filepath.Join(t.TempDir(), "target repo"))
	poison := initGitRepository(t, filepath.Join(t.TempDir(), "poison repo"))
	want, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	environ := append(os.Environ(),
		"GIT_DIR="+filepath.Join(poison, ".git"),
		"GIT_WORK_TREE="+poison,
		"GIT_AUTHOR_NAME=Retained Author",
	)

	out, err := runGitCommand(target, environ, "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatal(err)
	}
	if got := filepath.FromSlash(strings.TrimSpace(string(out))); got != want {
		t.Fatalf("git top level = %q, want %q", got, want)
	}
}

func initGitRepository(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-q", path)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	return path
}
