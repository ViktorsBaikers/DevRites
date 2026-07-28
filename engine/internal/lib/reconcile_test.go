package lib

import (
	"bytes"
	"encoding/json"
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

func TestReconcileFailsClosedWhenGitIsUnavailable(t *testing.T) {
	newGitRepo(t)
	root := workspace(t, "slice")
	t.Setenv("PATH", t.TempDir())

	code, output := runReconcile(t, root, "snapshot", "slice")
	if code != 6 {
		t.Fatalf("reconcile snapshot code=%d, want 6 when git is unavailable\n%s", code, output)
	}
	if !strings.Contains(output, "cannot resolve git worktree") {
		t.Fatalf("missing fail-closed diagnostic:\n%s", output)
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

func TestReconcileRepeatedCheckRejectsAllowlistedSourceDrift(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")
	writeWrightAllowlist(t, root, "feat", "seed.go")

	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	writeFile(t, filepath.Join(gitRoot, "seed.go"), "package main\n\nfunc builtByWright() {}\n")
	if code, out := runReconcile(t, root, "check", "feat"); code != 0 {
		t.Fatalf("first check = %d, want 0\n%s", code, out)
	}
	if code, out := runReconcile(t, root, "check", "feat"); code != 0 {
		t.Fatalf("unchanged repeated check = %d, want 0\n%s", code, out)
	}

	writeFile(t, filepath.Join(gitRoot, "seed.go"), "package main\n\nfunc changedByArtifactGate() {}\n")
	code, out := runReconcile(t, root, "check", "feat")
	if code != 5 {
		t.Fatalf("post-gate drift check = %d, want 5\n%s", code, out)
	}
	if !strings.Contains(out, "source changed after the last clean check") ||
		!strings.Contains(out, "seed.go") {
		t.Fatalf("post-gate drift diagnostic missing reason/path:\n%s", out)
	}
}

func TestReconcileRestoreCheckRollsBackOnlyPostCheckDelta(t *testing.T) {
	gitRoot := newGitRepo(t)
	if _, err := runGitCommand(gitRoot, nil, "config", "core.autocrlf", "true"); err != nil {
		t.Fatal(err)
	}
	root := workspace(t, "feat")
	writeWrightAllowlist(t, root, "feat", "seed.go", "kept.go")
	writeFile(t, filepath.Join(gitRoot, "kept.go"), "package main\n\nfunc beforeWright() {}\n")

	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	wrightSeed := "package main\n\nfunc builtByWright() {}\n"
	wrightKept := "package main\n\nfunc keptByWright() {}\n"
	writeFile(t, filepath.Join(gitRoot, "seed.go"), wrightSeed)
	writeFile(t, filepath.Join(gitRoot, "kept.go"), wrightKept)
	if code, out := runReconcile(t, root, "check", "feat"); code != 0 {
		t.Fatalf("check = %d, want 0\n%s", code, out)
	}

	writeFile(t, filepath.Join(gitRoot, "seed.go"), "package main\n\nfunc corruptedByGate() {}\n")
	if err := os.Remove(filepath.Join(gitRoot, "kept.go")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(gitRoot, "gate-added.go"), "package main\n")

	code, out := runReconcile(t, root, "restore-check", "feat")
	if code != 0 {
		t.Fatalf("restore-check = %d, want 0\n%s", code, out)
	}
	for _, changedPath := range []string{"gate-added.go", "kept.go", "seed.go"} {
		if !strings.Contains(out, changedPath) {
			t.Errorf("restore output omitted %s:\n%s", changedPath, out)
		}
	}
	if data, err := os.ReadFile(filepath.Join(gitRoot, "seed.go")); err != nil {
		t.Fatal(err)
	} else if string(data) != wrightSeed {
		t.Fatalf("seed.go = %q, want retained wright content %q", data, wrightSeed)
	}
	if data, err := os.ReadFile(filepath.Join(gitRoot, "kept.go")); err != nil {
		t.Fatal(err)
	} else if string(data) != wrightKept {
		t.Fatalf("kept.go = %q, want retained wright content %q", data, wrightKept)
	}
	if _, err := os.Stat(filepath.Join(gitRoot, "gate-added.go")); !os.IsNotExist(err) {
		t.Fatalf("post-check added source remains: %v", err)
	}
	if code, out := runReconcile(t, root, "check", "feat"); code != 0 {
		t.Fatalf("post-restore check = %d, want 0\n%s", code, out)
	}
}

func TestReconcileAbortRestoresOriginalSourceAndClosesWindow(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := filepath.Join(gitRoot, ".devrites")
	if err := os.MkdirAll(filepath.Join(root, "work", "feat"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeWrightAllowlist(t, root, "feat", "seed.go", "user-dirty.go")
	userDirty := "package main\n\nfunc userWorkBeforeDispatch() {}\n"
	writeFile(t, filepath.Join(gitRoot, "user-dirty.go"), userDirty)

	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	writeFile(t, filepath.Join(gitRoot, "seed.go"), "package main\n\nfunc rejectedWriterChange() {}\n")
	if err := os.Remove(filepath.Join(gitRoot, "user-dirty.go")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(gitRoot, "writer-added.go"), "package main\n")
	evidence := filepath.Join(featureDir(root, "feat"), "evidence.md")
	writeFile(t, evidence, "rejection evidence stays\n")

	code, out := runReconcile(t, root, "abort", "feat")
	if code != 0 {
		t.Fatalf("abort = %d, want 0\n%s", code, out)
	}
	if !strings.Contains(out, "restored 3 source path(s)") {
		t.Errorf("abort output omitted restored scope:\n%s", out)
	}
	if data, err := os.ReadFile(filepath.Join(gitRoot, "seed.go")); err != nil {
		t.Fatal(err)
	} else if string(data) != "package main\n" {
		t.Fatalf("seed.go = %q, want original source", data)
	}
	if data, err := os.ReadFile(filepath.Join(gitRoot, "user-dirty.go")); err != nil {
		t.Fatal(err)
	} else if string(data) != userDirty {
		t.Fatalf("pre-snapshot user work = %q, want %q", data, userDirty)
	}
	if _, err := os.Stat(filepath.Join(gitRoot, "writer-added.go")); !os.IsNotExist(err) {
		t.Fatalf("writer-added source remains after abort: %v", err)
	}
	if data, err := os.ReadFile(evidence); err != nil {
		t.Fatal(err)
	} else if string(data) != "rejection evidence stays\n" {
		t.Fatalf("abort changed canonical evidence: %q", data)
	}
	feature := featureDir(root, "feat")
	if isFile(filepath.Join(feature, reconcileBaseName)) || isDir(filepath.Join(feature, reconcileObjectsName)) {
		t.Fatal("abort retained the private reconciliation window")
	}
	receipts, err := filepath.Glob(filepath.Join(feature, ".reconcile-abort-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(receipts) != 1 {
		t.Fatalf("abort receipts = %v, want exactly one", receipts)
	}
	data, err := os.ReadFile(receipts[0])
	if err != nil {
		t.Fatal(err)
	}
	var receipt reconcileAbortReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.SchemaVersion != reconcileAbortSchema || receipt.Slug != "feat" ||
		!receipt.WindowClosed || receipt.SourceEntryCount != 2 ||
		len(receipt.RestoredPaths) != 3 {
		t.Fatalf("unexpected abort receipt: %+v", receipt)
	}
	entries, err := os.ReadDir(feature)
	if err != nil {
		t.Fatal(err)
	}
	var receiptEntry os.DirEntry
	for _, entry := range entries {
		if entry.Name() == filepath.Base(receipts[0]) {
			receiptEntry = entry
			break
		}
	}
	if receiptEntry == nil {
		t.Fatal("abort receipt directory entry not found")
	}
	if ok, err := reconcileRootOwnedOperationalFile(receipts[0], "work/feat/"+filepath.Base(receipts[0]), receiptEntry); err != nil || !ok {
		t.Fatalf("valid abort receipt operational=%v err=%v", ok, err)
	}
	if err := os.Chmod(receipts[0], 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receipts[0], append(data, 'x'), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := reconcileRootOwnedOperationalFile(receipts[0], "work/feat/"+filepath.Base(receipts[0]), receiptEntry); err == nil {
		t.Fatal("tampered content-addressed abort receipt was accepted")
	}
}

func TestReconcileAbortFailsClosedWhenObjectDatabaseIsMissing(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")
	writeWrightAllowlist(t, root, "feat", "seed.go")
	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	rejected := "package main\n\nfunc rejectedWriterChange() {}\n"
	writeFile(t, filepath.Join(gitRoot, "seed.go"), rejected)
	feature := featureDir(root, "feat")
	if err := os.RemoveAll(filepath.Join(feature, reconcileObjectsName)); err != nil {
		t.Fatal(err)
	}

	code, out := runReconcile(t, root, "abort", "feat")
	if code != 6 {
		t.Fatalf("abort = %d, want 6\n%s", code, out)
	}
	if data, err := os.ReadFile(filepath.Join(gitRoot, "seed.go")); err != nil {
		t.Fatal(err)
	} else if string(data) != rejected {
		t.Fatalf("failed abort mutated source: %q", data)
	}
	if !isFile(filepath.Join(feature, reconcileBaseName)) {
		t.Fatal("failed abort removed its retained baseline marker")
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

func TestReconcileUsesWrightStartAsCanonicalStateBoundary(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")
	dir := featureDir(root, "feat")
	writeWrightAllowlist(t, root, "feat", "seed.go")
	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}

	// Root-owned canonical records may legitimately change while a retained
	// source baseline survives technical recovery. The next wright start is the
	// boundary that separates those changes from the writer's delta.
	for _, name := range []string{
		"action.log",
		"browser-evidence.md",
		"decisions.md",
		"evidence.md",
		"footprint.log",
		"state.md",
		"touched-files.md",
	} {
		writeFile(t, filepath.Join(dir, name), "Root-owned recovery record\n")
	}
	if err := CaptureReconcileWrightBoundary(root, "feat"); err != nil {
		t.Fatalf("capture wright boundary: %v", err)
	}
	writeFile(t, filepath.Join(gitRoot, "seed.go"), "package main\n\nfunc repaired() {}\n")

	if code, out := runReconcile(t, root, "check", "feat"); code != 0 {
		t.Fatalf("check = %d, want 0\n%s", code, out)
	}

	// A canonical mutation after the wright starts is still a violation.
	writeFile(t, filepath.Join(dir, "state.md"), "Phase: prove\n")
	code, out := runReconcile(t, root, "check", "feat")
	if code != 5 {
		t.Fatalf("check after writer-time canonical mutation = %d, want 5\n%s", code, out)
	}
	if !strings.Contains(out, ".devrites/work/feat/state.md") {
		t.Fatalf("writer-time canonical mutation missing from output:\n%s", out)
	}
}

func TestReconcileAllowsRootOwnedOperationalFiles(t *testing.T) {
	newGitRepo(t)
	root := workspace(t, "feat")
	dir := featureDir(root, "feat")
	writeWrightAllowlist(t, root, "feat", "seed.go")
	if code := Stuck(root, []string{"log", "feat", "dispatch", "SLICE-010"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("stuck log = %d, want 0", code)
	}
	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}

	writeFile(t, filepath.Join(root, "timeline.jsonl"), "telemetry\n")
	writeFile(t, filepath.Join(dir, "events.jsonl"), "telemetry\n")
	writeFile(t, filepath.Join(dir, ".a1-guard.log"), "WOULD-BLOCK\n")
	writeFile(t, filepath.Join(dir, ".reviewer-ro.log"), "WOULD-BLOCK\n")
	writeFile(t, filepath.Join(dir, ".stop-gate.log"), "WOULD-BLOCK\n")
	writeFile(t, filepath.Join(dir, ".wright-scope.log"), "WOULD-BLOCK\n")
	writeFile(t, filepath.Join(dir, ".red"), "npm test\n")
	writeFile(t, filepath.Join(dir, "handoff.md"), "## Handoff snapshot\n")
	if code := RecoveryAttempts(
		root,
		[]string{"record", "--class", "proof_tool_defect", "reconcile telemetry", "changed logs", "feat"},
		&bytes.Buffer{},
		&bytes.Buffer{},
	); code != 0 {
		t.Fatalf("recovery record = %d, want 0", code)
	}

	if code, out := runReconcile(t, root, "check", "feat"); code != 0 {
		t.Fatalf("check = %d, want 0\n%s", code, out)
	}
	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("refresh = %d, want 0\n%s", code, out)
	}

	writeFile(t, filepath.Join(dir, ".unknown.log"), "not engine-owned\n")
	code, out := runReconcile(t, root, "check", "feat")
	if code != 5 {
		t.Fatalf("check after canonical mutation = %d, want 5\n%s", code, out)
	}
	if !strings.Contains(out, ".devrites/work/feat/.unknown.log") {
		t.Fatalf("canonical mutation missing from output:\n%s", out)
	}
}

func TestReconcileAcceptsLegacyOperationalFingerprints(t *testing.T) {
	newGitRepo(t)
	root := workspace(t, "feat")
	dir := featureDir(root, "feat")
	writeWrightAllowlist(t, root, "feat", "seed.go")
	writeFile(t, filepath.Join(root, "timeline.jsonl"), "before\n")
	writeFile(t, filepath.Join(dir, "events.jsonl"), "before\n")
	writeFile(t, filepath.Join(dir, ".a1-guard.log"), "before\n")
	writeFile(t, filepath.Join(dir, ".red"), "before\n")
	writeFile(t, filepath.Join(dir, "handoff.md"), "before\n")
	if code := RecoveryAttempts(
		root,
		[]string{"record", "--class", "proof_tool_defect", "reconcile telemetry", "first failure", "feat"},
		&bytes.Buffer{},
		&bytes.Buffer{},
	); code != 0 {
		t.Fatalf("initial recovery record = %d, want 0", code)
	}
	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}

	snapshotPath := filepath.Join(dir, reconcileDevritesName)
	state, err := readDevritesState(snapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	have := make(map[string]bool, len(state))
	for _, item := range state {
		have[item.Path] = true
	}
	for _, path := range []string{
		"timeline.jsonl",
		"work/feat/events.jsonl",
		"work/feat/.a1-guard.log",
		"work/feat/.red",
		"work/feat/handoff.md",
		"work/feat/recovery-attempts.jsonl",
	} {
		if !have[path] {
			state = append(state, devritesStateEntry{Path: path, Fingerprint: "file:0600:legacy"})
		}
	}
	if err := writeDevritesState(snapshotPath, state); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(root, "timeline.jsonl"), "after\n")
	writeFile(t, filepath.Join(dir, "events.jsonl"), "after\n")
	writeFile(t, filepath.Join(dir, ".a1-guard.log"), "after\n")
	writeFile(t, filepath.Join(dir, ".red"), "after\n")
	writeFile(t, filepath.Join(dir, "handoff.md"), "after\n")
	if code := RecoveryAttempts(
		root,
		[]string{"record", "--class", "proof_tool_defect", "reconcile telemetry", "second failure", "feat"},
		&bytes.Buffer{},
		&bytes.Buffer{},
	); code != 0 {
		t.Fatalf("second recovery record = %d, want 0", code)
	}

	if code, out := runReconcile(t, root, "check", "feat"); code != 0 {
		t.Fatalf("check = %d, want 0\n%s", code, out)
	}
}

func TestReconcileRejectsCorruptRootOwnedRecoveryLedger(t *testing.T) {
	newGitRepo(t)
	root := workspace(t, "feat")
	writeWrightAllowlist(t, root, "feat", "seed.go")
	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	writeFile(t, filepath.Join(featureDir(root, "feat"), recoveryAttemptsFile), "{not-json}\n")

	code, out := runReconcile(t, root, "check", "feat")
	if code != 6 {
		t.Fatalf("check = %d, want 6\n%s", code, out)
	}
	if !strings.Contains(out, "validate root-owned recovery ledger") {
		t.Fatalf("corrupt recovery ledger missing from output:\n%s", out)
	}
}

func TestReconcileRootOwnedOperationalPathsAreExact(t *testing.T) {
	for _, path := range []string{
		"timeline.jsonl",
		"work/feat/.red",
		"features/feat/events.jsonl",
		"work/feat/.reconcile-abort-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.json",
	} {
		if !reconcileRootOwnedOperationalPath(path) {
			t.Errorf("%q should be root-owned operational state", path)
		}
	}
	for _, path := range []string{
		"work/feat/.unknown.log",
		"work/feat/action.log",
		"work/feat/footprint.log",
		"work/feat/state.md",
	} {
		if reconcileRootOwnedOperationalPath(path) {
			t.Errorf("%q must remain fingerprinted", path)
		}
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

func TestPostSliceTestIntegrityUsesBaselineUntilClose(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")
	writeFile(t, filepath.Join(gitRoot, "README.md"), "fixture\n")
	commitAll(t, gitRoot, "baseline")

	// This user delta predates the slice and must not be blamed on the wright.
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
	writeFile(t, filepath.Join(featureDir(root, "feat"), reconcileWrightStateName), "stale\n")
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
	if isFile(filepath.Join(featureDir(root, "feat"), reconcileWrightStateName)) {
		t.Fatal("refresh retained the prior wright-start boundary")
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

func TestReconcileCloseRequiresCleanCheck(t *testing.T) {
	newGitRepo(t)
	root := workspace(t, "feat")
	writeWrightAllowlist(t, root, "feat", "seed.go")
	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}

	code, out := runReconcile(t, root, "close", "feat")
	if code != 6 {
		t.Fatalf("unchecked close = %d, want 6\n%s", code, out)
	}
	if !strings.Contains(out, "no clean check marker") {
		t.Fatalf("missing clean-check diagnostic:\n%s", out)
	}
	if !isFile(filepath.Join(featureDir(root, "feat"), reconcileBaseName)) ||
		!isDir(filepath.Join(featureDir(root, "feat"), reconcileObjectsName)) {
		t.Fatal("unchecked close destroyed the retained baseline")
	}
}

func TestReconcileCloseRejectsPostCheckSourceDrift(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")
	writeWrightAllowlist(t, root, "feat", "seed.go")
	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	writeFile(t, filepath.Join(gitRoot, "seed.go"), "package main\n\nfunc builtByWright() {}\n")
	if code, out := runReconcile(t, root, "check", "feat"); code != 0 {
		t.Fatalf("check = %d, want 0\n%s", code, out)
	}
	writeFile(t, filepath.Join(gitRoot, "seed.go"), "package main\n\nfunc changedAfterCheck() {}\n")

	code, out := runReconcile(t, root, "close", "feat")
	if code != 5 {
		t.Fatalf("drifted close = %d, want 5\n%s", code, out)
	}
	if !strings.Contains(out, "source changed after the last clean check") ||
		!strings.Contains(out, "seed.go") {
		t.Fatalf("missing post-check drift diagnostic:\n%s", out)
	}
	if !isFile(filepath.Join(featureDir(root, "feat"), reconcileBaseName)) ||
		!isDir(filepath.Join(featureDir(root, "feat"), reconcileObjectsName)) {
		t.Fatal("drifted close destroyed the retained baseline")
	}
}

func TestReconcileCloseRejectsWorkspaceSymlinkEscape(t *testing.T) {
	newGitRepo(t)
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "work", "feat")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	writeFile(t, filepath.Join(outside, reconcileBaseName), "outside\n")
	if err := os.MkdirAll(filepath.Join(outside, reconcileObjectsName), 0o700); err != nil {
		t.Fatal(err)
	}

	code, out := runReconcile(t, root, "close", "feat")
	if code != 6 {
		t.Fatalf("symlinked close = %d, want 6\n%s", code, out)
	}
	if !strings.Contains(out, "workspace") {
		t.Fatalf("missing workspace-path diagnostic:\n%s", out)
	}
	if !isFile(filepath.Join(outside, reconcileBaseName)) ||
		!isDir(filepath.Join(outside, reconcileObjectsName)) {
		t.Fatal("symlinked close deleted state outside the DevRites root")
	}
}
