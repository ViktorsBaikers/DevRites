package main_test

// Multi-process concurrency: DevRites fans out reviewer subagents that each
// spawn the binary. This asserts concurrent read-only status calls stay
// consistent against one workspace.

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
	return runProcInput(root, "", args...)
}

func runProcInput(root, stdin string, args ...string) (stdout string, code int, fatal error) {
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), "DEVRITES_ROOT="+root)
	cmd.Stdin = strings.NewReader(stdin)
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
			t.Errorf("proc %d exit = %d, want 0\n%s", i, r.code, r.out)
			continue
		}
		if r.out != want {
			t.Errorf("proc %d status output diverged under contention\n--- got ---\n%s--- want ---\n%s", i, r.out, want)
		}
	}
}

// TestConcurrentStopGateAppendsNeverTear spawns many concurrent stop-gate
// processes (observe mode) that each append a would-block record to the same
// per-feature log. Every record must land exactly once and intact: the
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
			_, code, fatal := runProcInput(root, `{"stop_hook_active":false}`,
				"hook", "stop-gate", "--harness=claude")
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
