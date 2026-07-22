package lib

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newGitRepo builds a throwaway git repo with one commit, chdirs into it (Reconcile
// resolves gitRoot from the cwd), and returns its path.
func newGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// macOS /var -> /private/var symlink: git reports the resolved path, so resolve
	// up front or the gitRoot comparison below drifts.
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	writeFile(t, filepath.Join(dir, "seed.go"), "package main\n")
	commitAll(t, dir, "seed")
	t.Chdir(dir)
	return dir
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func commitAll(t *testing.T, dir, msg string) {
	t.Helper()
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", msg}} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// workspace creates <root>/work/<slug> and returns root.
func workspace(t *testing.T, slug string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "work", slug), 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	return root
}

func runReconcile(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := Reconcile(root, args, stdout, stderr)
	return code, stdout.String() + stderr.String()
}

func TestWorktreeTreeErrorsOutsideGitRepo(t *testing.T) {
	dir := t.TempDir()
	tree, err := worktreeTree(dir, t.TempDir())
	if err == nil {
		t.Fatalf("worktreeTree(%q) = %q, nil; want an error outside a git repo", dir, tree)
	}
	if tree != "" {
		t.Errorf("worktreeTree returned tree %q alongside an error; want empty", tree)
	}
}

func TestWorktreeTreeCapturesUntrackedFiles(t *testing.T) {
	dir := newGitRepo(t)
	objects := t.TempDir()

	before, err := worktreeTree(dir, objects)
	if err != nil {
		t.Fatalf("worktreeTree: %v", err)
	}
	writeFile(t, filepath.Join(dir, "untracked.go"), "package main\n")
	after, err := worktreeTree(dir, objects)
	if err != nil {
		t.Fatalf("worktreeTree: %v", err)
	}
	if before == after {
		t.Errorf("worktreeTree did not observe the untracked file: both trees = %s", before)
	}

	// The user's real index must be untouched: the file stays untracked.
	out, err := exec.Command("git", "-C", dir, "status", "--porcelain", "untracked.go").Output()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	if !strings.HasPrefix(string(out), "??") {
		t.Errorf("worktreeTree staged into the real index; git status = %q", out)
	}
}

func TestReconcileSnapshotThenCleanCheck(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")
	objects := t.TempDir()
	if err := os.Chmod(objects, 0o500); err != nil {
		t.Fatalf("make git objects read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(objects, 0o700) })
	t.Setenv("GIT_OBJECT_DIRECTORY", objects)
	writeFile(t, filepath.Join(gitRoot, "untracked.go"), "package main\n")

	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	// The wright changes a file and claims it.
	writeFile(t, filepath.Join(gitRoot, "seed.go"), "package main\n\nfunc main() {}\n")
	writeFile(t, filepath.Join(root, "work", "feat", ".reconcile-claimed"), "seed.go\n")

	code, out := runReconcile(t, root, "check", "feat")
	if code != 0 {
		t.Fatalf("check = %d, want 0 (claimed change)\n%s", code, out)
	}
	if !strings.Contains(out, "reconcile: OK") {
		t.Errorf("check output missing OK line:\n%s", out)
	}
}

func TestReconcileCheckFlagsUnclaimedChange(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")

	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	// The orchestrator edits source it never claimed: the A1 breach.
	writeFile(t, filepath.Join(gitRoot, "rogue.go"), "package main\n")
	writeFile(t, filepath.Join(root, "work", "feat", ".reconcile-claimed"), "seed.go\n")

	code, out := runReconcile(t, root, "check", "feat")
	if code != 5 {
		t.Fatalf("check = %d, want 5 (A1 breach)\n%s", code, out)
	}
	if !strings.Contains(out, "rogue.go") {
		t.Errorf("violation list missing rogue.go:\n%s", out)
	}
}

// A snapshot that failed to capture the worktree must fail the check closed. Before
// the fix, worktreeTree swallowed a "git add -A" failure and returned the empty-tree
// sha; base and now agreed, the diff came back empty, and the gate printed OK on the
// exact breach it exists to catch.
func TestReconcileCheckFailsClosedOnEmptySnapshot(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")

	writeFile(t, filepath.Join(root, "work", "feat", ".reconcile-base"), "\n")
	writeFile(t, filepath.Join(root, "work", "feat", ".reconcile-claimed"), "seed.go\n")
	writeFile(t, filepath.Join(gitRoot, "rogue.go"), "package main\n")

	code, out := runReconcile(t, root, "check", "feat")
	if code == 0 {
		t.Fatalf("check = 0 on an empty snapshot; the gate passed an unclaimed change\n%s", out)
	}
	if code != 6 {
		t.Fatalf("check = %d, want 6 (setup error)\n%s", code, out)
	}
}

func TestReconcileCheckRequiresSnapshot(t *testing.T) {
	newGitRepo(t)
	root := workspace(t, "feat")
	writeFile(t, filepath.Join(root, "work", "feat", ".reconcile-claimed"), "seed.go\n")

	code, out := runReconcile(t, root, "check", "feat")
	if code != 6 {
		t.Fatalf("check without snapshot = %d, want 6\n%s", code, out)
	}
}

func TestReconcileInlineFallbackSkipsGate(t *testing.T) {
	newGitRepo(t)
	root := workspace(t, "feat")
	writeFile(t, filepath.Join(root, "work", "feat", ".reconcile-inline"), "")

	code, out := runReconcile(t, root, "check", "feat")
	if code != 0 {
		t.Fatalf("inline fallback check = %d, want 0\n%s", code, out)
	}
	if !strings.Contains(out, "gate skipped") {
		t.Errorf("expected inline-fallback skip message:\n%s", out)
	}
}
