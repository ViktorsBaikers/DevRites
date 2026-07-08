package main_test

// Issue 07: multi-process concurrency. DevRites fans out reviewer subagents that
// each spawn the binary, so the real contention is between short-lived
// processes, not goroutines. This spawns many concurrent devrites processes
// against ONE workspace and asserts they all succeed with consistent output —
// exercising WAL + busy_timeout so SQLITE_BUSY never surfaces as a hard failure.

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// runProc runs the binary without touching *testing.T, so it is safe to call
// from many goroutines. It returns stdout and exit code (and a fatal error only
// for a non-exit failure like the binary being unrunnable).
func runProc(root string, args ...string) (stdout string, code int, fatal error) {
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), "DEVRITES_ROOT="+root)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	err := cmd.Run()
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			return out.String(), -1, err
		}
		code = ee.ExitCode()
	}
	return out.String(), code, nil
}

func TestConcurrentStatusProcessesStayConsistent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process stress test in -short mode")
	}
	root := newWorkspace(t)
	// Warm the index once, then fan out — this is the real DevRites pattern: the
	// index is built up front, then many short-lived processes read and heal it
	// concurrently (with the occasional rebuild) while reviewer subagents run.
	if _, _, code := runDevrites(t, root, "reindex"); code != 0 {
		t.Fatal("warm-up reindex failed")
	}
	want := goldenStatus(t, "auth-tokens")

	const n = 24
	type res struct {
		out   string
		code  int
		fatal error
	}
	results := make([]res, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Mix readers (status) with the occasional index rebuild (reindex) to
			// force writer-vs-writer and writer-vs-reader contention on SQLite.
			if i%6 == 0 {
				out, code, fatal := runProc(root, "reindex")
				results[i] = res{out, code, fatal}
				return
			}
			out, code, fatal := runProc(root, "status", "auth-tokens")
			results[i] = res{out, code, fatal}
		}(i)
	}
	wg.Wait()

	for i, r := range results {
		if r.fatal != nil {
			t.Fatalf("proc %d could not run: %v", i, r.fatal)
		}
		if r.code != 0 {
			t.Errorf("proc %d exit = %d, want 0 (SQLITE_BUSY not handled?)\n%s", i, r.code, r.out)
			continue
		}
		if i%6 != 0 && r.out != want {
			t.Errorf("proc %d status output diverged under contention\n--- got ---\n%s--- want ---\n%s", i, r.out, want)
		}
	}
}

// TestConcurrentStopGateAppendsNeverTear spawns many concurrent stop-gate
// processes (observe mode) that each append a would-block record to the same
// per-feature log. Every record must land exactly once and intact — the
// multi-PROCESS proof of the O_APPEND log path (the in-process proof lives in
// internal/state/concurrency_test.go).
func TestConcurrentStopGateAppendsNeverTear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process stress test in -short mode")
	}
	root := newWorkspace(t)
	setPhase(t, root, "auth-tokens", "seal") // claims done; proof.md is empty → would-block
	writeActive(t, root, "auth-tokens")

	const n = 24
	var wg sync.WaitGroup
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, code, fatal := runProc(root, "hook", "stop-gate", "--harness=claude")
			if fatal != nil {
				code = -1
			}
			codes[i] = code
		}(i)
	}
	wg.Wait()

	for i, c := range codes {
		if c != 0 {
			t.Errorf("stop-gate proc %d exit = %d, want 0", i, c)
		}
	}

	raw, err := os.ReadFile(filepath.Join(root, "features", "auth-tokens", ".stop-gate.log"))
	if err != nil {
		t.Fatalf("stop-gate log not written: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != n {
		t.Fatalf("got %d log records, want %d (lost or torn appends)", len(lines), n)
	}
	for _, l := range lines {
		if !strings.HasPrefix(l, "WOULD-BLOCK\t") || !strings.HasSuffix(l, "before stopping") {
			t.Fatalf("interleaved/torn append record: %q", l)
		}
	}
}
