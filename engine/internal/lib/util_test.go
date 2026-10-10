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

// cappedGitOutput is installed as both cmd.Stdout and cmd.Stderr; this drives
// two concurrent writers at it so -race flags any unsynchronized access.
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
