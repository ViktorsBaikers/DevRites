package lib

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
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

func TestGitStatusPathsLeavesStaleIndexUntouched(t *testing.T) {
	t.Setenv("GIT_OPTIONAL_LOCKS", "")
	if err := os.Unsetenv("GIT_OPTIONAL_LOCKS"); err != nil {
		t.Fatal(err)
	}
	repo := initGitRepository(t, filepath.Join(t.TempDir(), "repo"))
	file := filepath.Join(repo, "a.txt")
	if err := os.WriteFile(file, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	add := exec.Command("git", "-C", repo, "add", "a.txt")
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	stale := time.Now().Add(-time.Hour)
	if err := os.Chtimes(file, stale, stale); err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(repo, ".git", "index")
	before, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gitStatusPaths(repo); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("git status rewrote .git/index")
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

// cappedGitOutput must stay safe for concurrent writers; this drives two at it
// so -race flags any unsynchronized access.
func TestCappedGitOutputConcurrentWriters(t *testing.T) {
	const writes, size = 2000, 64
	chunk := []byte(strings.Repeat("x", size))

	var seq cappedGitOutput
	for i := 0; i < 2*writes; i++ {
		_, _ = seq.Write(chunk)
	}
	if seq.Len() != 2*writes*size || seq.truncated {
		t.Fatalf("sequential: got %d bytes truncated=%v, want %d", seq.Len(), seq.truncated, 2*writes*size)
	}

	var out cappedGitOutput
	var wg sync.WaitGroup
	for w := 0; w < 2; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < writes; i++ {
				_, _ = out.Write(chunk)
			}
		}()
	}
	wg.Wait()
	t.Logf("concurrent bytes %d of %d", out.Len(), 2*writes*size)
}

func TestCappedGitOutputBoundsIOCopy(t *testing.T) {
	src := func() io.Reader { return io.LimitReader(zeroReader{}, 25<<20) }

	var control boundedSecretScanBuffer
	control.limit = gitOutputLimit
	_, _ = io.Copy(&control, src())
	if control.Len() != gitOutputLimit || !control.overflow {
		t.Fatalf("control: got %d bytes overflow=%v, want %d and overflow", control.Len(), control.overflow, gitOutputLimit)
	}

	var out cappedGitOutput
	_, _ = io.Copy(&out, src())
	if out.Len() > gitOutputLimit || !out.truncated {
		t.Fatalf("got %d bytes truncated=%v, want at most %d and truncated", out.Len(), out.truncated, gitOutputLimit)
	}
}

func TestRunGitCommandRejectsTruncatedOutput(t *testing.T) {
	repo := initGitRepository(t, t.TempDir())
	blob := filepath.Join(t.TempDir(), "big")
	if err := os.WriteFile(blob, bytes.Repeat([]byte("x"), gitOutputLimit+1), 0o600); err != nil {
		t.Fatal(err)
	}
	oid, err := runGitCommand(repo, nil, "hash-object", "-w", blob)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runGitCommand(repo, nil, "cat-file", "blob", strings.TrimSpace(string(oid))); err == nil {
		t.Fatal("truncated git output returned without error")
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) { return len(p), nil }

func diffScopeRepo(t *testing.T) string {
	t.Helper()
	repo := initGitRepository(t, t.TempDir())
	runDiffScopeGit(t, repo, "config", "user.email", "tests@example.invalid")
	runDiffScopeGit(t, repo, "config", "user.name", "DevRites Tests")
	return repo
}

// A rename must report the deleted source too: with git's default rename
// detection only the new name surfaced, so moving a file out from under the
// allowlist passed.
func TestDiffScopeReportsRenameSourceInDiffModes(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	repo := diffScopeRepo(t)
	writeFile(t, filepath.Join(repo, "secret.txt"), "classified\n")
	runDiffScopeGit(t, repo, "add", "-A")
	runDiffScopeGit(t, repo, "commit", "-qm", "base")
	runDiffScopeGit(t, repo, "mv", "secret.txt", "src_moved.txt")
	check := func(mode ...string) {
		t.Helper()
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		args := append([]string{"feat", "--allow", "src_moved.txt", "--cwd", repo}, mode...)
		code := RunCheckDiffScope(root, args, stdout, stderr)
		if code != 3 || !strings.Contains(stdout.String(), "BLOCKED: secret.txt outside allowlist") {
			t.Fatalf("%v: code=%d, want rename source blocked\n%s%s", mode, code, stdout.String(), stderr.String())
		}
	}
	check("--staged")
	runDiffScopeGit(t, repo, "commit", "-qm", "move")
	check("--base", "HEAD~1")
}

// Git diagnostics on stderr are never parsed as changed paths.
func TestDiffScopeIgnoresGitStderrWarnings(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	repo := diffScopeRepo(t)
	runDiffScopeGit(t, repo, "config", "core.autocrlf", "true")
	writeFile(t, filepath.Join(repo, "f.txt"), "a\n")
	runDiffScopeGit(t, repo, "add", "-A")
	runDiffScopeGit(t, repo, "commit", "-qm", "base")
	writeFile(t, filepath.Join(repo, "f.txt"), "a\nb\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunCheckDiffScope(root, []string{"feat", "--allow", "f.txt", "--base", "HEAD", "--cwd", repo}, stdout, stderr)
	if code != 0 || strings.Contains(stdout.String(), "warning") {
		t.Fatalf("code=%d, want ok without stderr noise\n%s%s", code, stdout.String(), stderr.String())
	}
}

// The engine's own .devrites machine files are never task paths, even when
// the control tree is not gitignored.
func TestDiffScopeIgnoresProjectDevritesTree(t *testing.T) {
	repo := diffScopeRepo(t)
	root := filepath.Join(repo, ".devrites")
	machine := []string{
		"claims.jsonl", "claims.jsonl.lock",
		"work/feat/dispatch.json", "work/feat/metrics.jsonl", "work/feat/.lock",
		"work/feat/readiness-inputs.json", "work/feat/regression-baseline.json",
		"work/feat/parallel-lease.md", "work/feat/ctx/build.bundle.md",
		"work/feat/ctx/build.bundle.md.sha256", "work/other/dispatch.json",
	}
	for _, rel := range machine {
		writeFile(t, filepath.Join(root, filepath.FromSlash(rel)), "{}\n")
	}
	writeFile(t, filepath.Join(repo, "src", "ok.go"), "package ok\n")
	check := func(mode ...string) {
		t.Helper()
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := RunCheckDiffScope(root, append([]string{"feat", "--allow", "src"}, mode...), stdout, stderr)
		if code != 0 || !strings.Contains(stdout.String(), "1 changed") {
			t.Fatalf("%v: code=%d, want only src/ok.go\n%s%s", mode, code, stdout.String(), stderr.String())
		}
	}
	check("--worktree")
	runDiffScopeGit(t, repo, "add", "-A")
	check("--staged")
	runDiffScopeGit(t, repo, "commit", "-qm", "base")
	for _, rel := range machine {
		writeFile(t, filepath.Join(root, filepath.FromSlash(rel)), "{\"wave\":1}\n")
	}
	writeFile(t, filepath.Join(repo, "src", "ok.go"), "package ok // changed\n")
	check("--base", "HEAD")
}

// Only engine machine files are exempt: a writer editing principles, another
// feature's workspace, or the active workspace's evidence or gates is
// reported like any other out-of-allowlist path.
func TestDiffScopeReportsDevritesArtifactEdits(t *testing.T) {
	for _, rel := range []string{"principles.md", "work/feat/evidence.md", "work/feat/gates.md", "work/other/spec.md"} {
		t.Run(rel, func(t *testing.T) {
			repo := diffScopeRepo(t)
			root := filepath.Join(repo, ".devrites")
			path := filepath.Join(root, filepath.FromSlash(rel))
			writeFile(t, path, "a\n")
			writeFile(t, filepath.Join(repo, "src", "ok.go"), "package ok\n")
			check := func(mode ...string) {
				t.Helper()
				stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
				code := RunCheckDiffScope(root, append([]string{"feat", "--allow", "src"}, mode...), stdout, stderr)
				if code != 3 || !strings.Contains(stdout.String(), "BLOCKED: .devrites/"+rel+" outside allowlist") {
					t.Fatalf("%v: code=%d, want .devrites/%s BLOCKED\n%s%s", mode, code, rel, stdout.String(), stderr.String())
				}
			}
			check("--worktree")
			runDiffScopeGit(t, repo, "add", "-A")
			check("--staged")
			runDiffScopeGit(t, repo, "commit", "-qm", "base")
			writeFile(t, path, "b\n")
			check("--base", "HEAD")
		})
	}
}
