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

func writeWrightAllowlist(t *testing.T, root, slug string, paths ...string) {
	t.Helper()
	body := ""
	if len(paths) > 0 {
		body = strings.Join(paths, "\n") + "\n"
	}
	writeFile(t, filepath.Join(featureDir(root, slug), defaultWrightAllowlistName), body)
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

func TestWorktreeTreeCapturesFilesWithoutHead(t *testing.T) {
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	objects := t.TempDir()

	before, err := worktreeTree(dir, objects)
	if err != nil {
		t.Fatalf("empty worktreeTree: %v", err)
	}
	writeFile(t, filepath.Join(dir, "first.go"), "package main\n")
	after, err := worktreeTree(dir, objects)
	if err != nil {
		t.Fatalf("populated worktreeTree: %v", err)
	}
	if before == after {
		t.Errorf("worktreeTree did not observe the first file without HEAD: both trees = %s", before)
	}
}

func TestReconcileCheckFlagsIgnoredTrackedFileDeletion(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")
	writeFile(t, filepath.Join(gitRoot, ".gitignore"), "seed.go\n")
	writeWrightAllowlist(t, root, "feat")

	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	if err := os.Remove(filepath.Join(gitRoot, "seed.go")); err != nil {
		t.Fatal(err)
	}

	code, out := runReconcile(t, root, "check", "feat")
	if code != 5 {
		t.Fatalf("check = %d, want 5 (ignored tracked deletion)\n%s", code, out)
	}
	if !strings.Contains(out, "seed.go") {
		t.Errorf("violation list missing seed.go:\n%s", out)
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
	writeWrightAllowlist(t, root, "feat", "seed.go")

	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	// The wright changes the exact file authorized before dispatch.
	writeFile(t, filepath.Join(gitRoot, "seed.go"), "package main\n\nfunc main() {}\n")

	code, out := runReconcile(t, root, "check", "feat")
	if code != 0 {
		t.Fatalf("check = %d, want 0 (claimed change)\n%s", code, out)
	}
	if !strings.Contains(out, "reconcile: OK") {
		t.Errorf("check output missing OK line:\n%s", out)
	}
	if !isFile(filepath.Join(featureDir(root, "feat"), reconcileBaseName)) || !isDir(filepath.Join(featureDir(root, "feat"), reconcileObjectsName)) {
		t.Fatal("successful check removed the baseline needed by later gates")
	}
	if code, out := runReconcile(t, root, "close", "feat"); code != 0 {
		t.Fatalf("close = %d, want 0\n%s", code, out)
	}
	if isFile(filepath.Join(featureDir(root, "feat"), reconcileBaseName)) || isDir(filepath.Join(featureDir(root, "feat"), reconcileObjectsName)) {
		t.Fatal("close retained private baseline state")
	}
}

func TestReconcileCheckFlagsUnauthorizedChange(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")
	writeWrightAllowlist(t, root, "feat", "seed.go")

	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	// A writer cannot change a path that was not authorized at dispatch.
	writeFile(t, filepath.Join(gitRoot, "rogue.go"), "package main\n")

	code, out := runReconcile(t, root, "check", "feat")
	if code != 5 {
		t.Fatalf("check = %d, want 5 (A1 breach)\n%s", code, out)
	}
	if !strings.Contains(out, "rogue.go") {
		t.Errorf("violation list missing rogue.go:\n%s", out)
	}
}

// A failed worktree capture must make the check fail closed. `git write-tree`
// can still succeed against a seeded index after `git add -A` fails, producing
// a stale tree that could hide an A1 breach.
func TestReconcileCheckFailsClosedOnEmptySnapshot(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")

	writeFile(t, filepath.Join(root, "work", "feat", ".reconcile-base"), "\n")
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

	code, out := runReconcile(t, root, "check", "feat")
	if code != 6 {
		t.Fatalf("check without snapshot = %d, want 6\n%s", code, out)
	}
}

func TestReconcileSnapshotRejectsInvalidAllowlists(t *testing.T) {
	tests := []struct {
		name  string
		body  func(gitRoot string) string
		setup func(*testing.T, string)
	}{
		{name: "absolute", body: func(gitRoot string) string { return filepath.Join(gitRoot, "seed.go") + "\n" }},
		{name: "traversal", body: func(string) string { return "../seed.go\n" }},
		{name: "duplicate", body: func(string) string { return "seed.go\nseed.go\n" }},
		{name: "devrites", body: func(string) string { return ".devrites/work/feat/state.md\n" }},
		{name: "dot segment", body: func(string) string { return "./seed.go\n" }},
		{
			name: "directory",
			body: func(string) string { return "src\n" },
			setup: func(t *testing.T, gitRoot string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(gitRoot, "src"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "symlink escape",
			body: func(string) string { return "escaped.go\n" },
			setup: func(t *testing.T, gitRoot string) {
				t.Helper()
				outside := filepath.Join(t.TempDir(), "outside.go")
				writeFile(t, outside, "package outside\n")
				if err := os.Symlink(outside, filepath.Join(gitRoot, "escaped.go")); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gitRoot := newGitRepo(t)
			root := workspace(t, "feat")
			if test.setup != nil {
				test.setup(t, gitRoot)
			}
			writeFile(t, filepath.Join(featureDir(root, "feat"), defaultWrightAllowlistName), test.body(gitRoot))

			code, out := runReconcile(t, root, "snapshot", "feat")
			if code != 6 {
				t.Fatalf("snapshot = %d, want 6 for invalid allowlist\n%s", code, out)
			}
			if !strings.Contains(out, "invalid wright allowlist") {
				t.Fatalf("missing allowlist diagnostic:\n%s", out)
			}
		})
	}
}

func TestReconcileUsesEnvironmentAllowlistCapturedBeforeDispatch(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")
	allowlist := filepath.Join(featureDir(root, "feat"), "orchestrator-allowlist")
	writeFile(t, allowlist, "seed.go\n")
	t.Setenv(wrightAllowlistFileEnv, allowlist)

	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	// Mutating the source allowlist after dispatch cannot authorize a new path:
	// check consumes the captured copy, not the live orchestrator input.
	writeFile(t, allowlist, "rogue.go\n")
	writeFile(t, filepath.Join(gitRoot, "rogue.go"), "package main\n")

	code, out := runReconcile(t, root, "check", "feat")
	if code != 5 {
		t.Fatalf("check = %d, want 5\n%s", code, out)
	}
	if !strings.Contains(out, "rogue.go") {
		t.Fatalf("unauthorized path missing from output:\n%s", out)
	}
}

func TestReconcileRejectsEnvironmentAllowlistOutsideWorkspace(t *testing.T) {
	newGitRepo(t)
	root := workspace(t, "feat")
	allowlist := filepath.Join(root, "outside-workspace-allowlist")
	writeFile(t, allowlist, "seed.go\n")
	t.Setenv(wrightAllowlistFileEnv, allowlist)

	code, out := runReconcile(t, root, "snapshot", "feat")
	if code != 6 {
		t.Fatalf("snapshot = %d, want 6\n%s", code, out)
	}
	if !strings.Contains(out, "invalid wright allowlist location") {
		t.Fatalf("missing allowlist-location diagnostic:\n%s", out)
	}
}

func TestReconcileRejectsDevritesMutation(t *testing.T) {
	newGitRepo(t)
	root := workspace(t, "feat")
	writeWrightAllowlist(t, root, "feat", "seed.go")
	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	writeFile(t, filepath.Join(featureDir(root, "feat"), "state.md"), "Phase: build\n")

	code, out := runReconcile(t, root, "check", "feat")
	if code != 5 {
		t.Fatalf("check = %d, want 5\n%s", code, out)
	}
	if !strings.Contains(out, ".devrites/work/feat/state.md") {
		t.Fatalf(".devrites mutation missing from output:\n%s", out)
	}
}

func TestReconcileCheckFailsClosedWhenObjectDatabaseIsMissing(t *testing.T) {
	newGitRepo(t)
	root := workspace(t, "feat")
	writeWrightAllowlist(t, root, "feat", "seed.go")
	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	if err := os.RemoveAll(filepath.Join(featureDir(root, "feat"), reconcileObjectsName)); err != nil {
		t.Fatal(err)
	}

	code, out := runReconcile(t, root, "check", "feat")
	if code != 6 {
		t.Fatalf("check = %d, want 6\n%s", code, out)
	}
	if !strings.Contains(out, "partial lifecycle") {
		t.Fatalf("missing fail-closed lifecycle diagnostic:\n%s", out)
	}
}

func TestReconcileCheckFailsClosedWhenObjectDatabaseIsCorrupt(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")
	writeWrightAllowlist(t, root, "feat", "seed.go")
	writeFile(t, filepath.Join(gitRoot, "seed.go"), "package seed\n")
	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	baseData, err := os.ReadFile(filepath.Join(featureDir(root, "feat"), reconcileBaseName))
	if err != nil {
		t.Fatal(err)
	}
	baseTree := strings.TrimSpace(string(baseData))
	treeObject := filepath.Join(featureDir(root, "feat"), reconcileObjectsName, baseTree[:2], baseTree[2:])
	if err := os.Remove(treeObject); err != nil {
		t.Fatal(err)
	}

	code, out := runReconcile(t, root, "check", "feat")
	if code != 6 {
		t.Fatalf("check = %d, want 6\n%s", code, out)
	}
	if !strings.Contains(out, "tree is unavailable") {
		t.Fatalf("missing corrupt-object diagnostic:\n%s", out)
	}
}

func TestPostSliceGatesShareBaselineUntilClose(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")
	writeFile(t, filepath.Join(gitRoot, "package.json"), "{}\n")
	commitAll(t, gitRoot, "manifest")

	// These user deltas predate the slice. In particular, the undeclared import
	// must not be blamed on the wright by package-existence.
	writeFile(t, filepath.Join(gitRoot, "src/user-delta.ts"), `import ghost from "preexisting-user-package";`+"\n")
	writeFile(t, filepath.Join(gitRoot, "tests/user_delta_test.go"), "package tests\n\nfunc TestUserDelta(t *testing.T) { t.Fatal(\"baseline\") }\n")
	writeWrightAllowlist(t, root, "feat", "seed.go")

	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	writeFile(t, filepath.Join(gitRoot, "seed.go"), "package main\n\nfunc main() {}\n")
	if code, out := runReconcile(t, root, "check", "feat"); code != 0 {
		t.Fatalf("check = %d, want 0\n%s", code, out)
	}

	var stdout, stderr bytes.Buffer
	if code := TestIntegrity(root, []string{"feat"}, &stdout, &stderr); code != 0 {
		t.Fatalf("test-integrity = %d, want 0\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := PackageExistence(root, []string{"feat"}, &stdout, &stderr); code != 0 {
		t.Fatalf("package-existence = %d, want 0\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	if code, out := runReconcile(t, root, "close", "feat"); code != 0 {
		t.Fatalf("close = %d, want 0\n%s", code, out)
	}
	if isFile(filepath.Join(featureDir(root, "feat"), reconcileBaseName)) || isDir(filepath.Join(featureDir(root, "feat"), reconcileObjectsName)) {
		t.Fatal("close retained the shared slice baseline")
	}
}

func TestReconcileRefreshPreservesOriginalBaselineAcrossRetry(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")
	writeWrightAllowlist(t, root, "feat", "seed.go")

	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("initial snapshot = %d, want 0\n%s", code, out)
	}
	basePath := filepath.Join(featureDir(root, "feat"), reconcileBaseName)
	originalBase, err := os.ReadFile(basePath)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(gitRoot, "seed.go"), "package main\n\nfunc firstAttempt() {}\n")
	if code, out := runReconcile(t, root, "check", "feat"); code != 0 {
		t.Fatalf("first check = %d, want 0\n%s", code, out)
	}

	// The root may persist recovery bookkeeping and refine the exact allowlist
	// between attempts. Refresh must re-arm those boundaries without replacing
	// the original pre-slice source baseline.
	writeFile(t, filepath.Join(featureDir(root, "feat"), "recovery-attempts.json"), "{}\n")
	writeWrightAllowlist(t, root, "feat", "seed.go", "retry.go")
	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("refresh snapshot = %d, want 0\n%s", code, out)
	} else if !strings.Contains(out, "original slice baseline retained") {
		t.Fatalf("refresh output omitted retained-baseline proof:\n%s", out)
	}
	refreshedBase, err := os.ReadFile(basePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(originalBase, refreshedBase) {
		t.Fatalf("refresh replaced original baseline: before=%q after=%q", originalBase, refreshedBase)
	}

	writeFile(t, filepath.Join(gitRoot, "retry.go"), "package main\n")
	if code, out := runReconcile(t, root, "check", "feat"); code != 0 {
		t.Fatalf("retry check = %d, want 0\n%s", code, out)
	}
}

func TestReconcileRefreshRequiresPriorCleanCheck(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")
	writeWrightAllowlist(t, root, "feat", "seed.go")
	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("initial snapshot = %d, want 0\n%s", code, out)
	}
	writeFile(t, filepath.Join(gitRoot, "seed.go"), "package main\n\nfunc unchecked() {}\n")

	code, out := runReconcile(t, root, "snapshot", "feat")
	if code != 6 {
		t.Fatalf("unchecked refresh = %d, want 6\n%s", code, out)
	}
	if !strings.Contains(out, "no clean check marker") {
		t.Fatalf("missing clean-check diagnostic:\n%s", out)
	}
}

func TestReconcileCloseClearsPrivateWindowState(t *testing.T) {
	newGitRepo(t)
	root := workspace(t, "feat")
	writeFile(t, filepath.Join(root, "work", "feat", reconcileBaseName), "stale\n")
	if err := os.MkdirAll(filepath.Join(root, "work", "feat", reconcileObjectsName), 0o755); err != nil {
		t.Fatal(err)
	}

	code, out := runReconcile(t, root, "close", "feat")
	if code != 0 {
		t.Fatalf("close = %d, want 0\n%s", code, out)
	}
	if !strings.Contains(out, "closed slice window") {
		t.Errorf("expected close message:\n%s", out)
	}
	if isFile(filepath.Join(featureDir(root, "feat"), reconcileBaseName)) || isDir(filepath.Join(featureDir(root, "feat"), reconcileObjectsName)) {
		t.Fatal("close retained private baseline state")
	}
}
