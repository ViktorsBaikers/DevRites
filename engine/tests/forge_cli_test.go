package main_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/devrites/devrites/internal/forge"
)

const (
	forgeAcceptanceHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	forgeTestPlanHash   = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func TestForgeCLIPlanRequiresSupportedWorkerBinding(t *testing.T) {
	repo := newForgeCLIRepo(t)
	beforeWorktrees := forgeGitRaw(t, repo, "worktree", "list", "--porcelain")
	beforeBranches := forgeGitRaw(t, repo, "for-each-ref", "--format=%(refname)", "refs/heads/devrites/forge")
	args := []string{
		"forge", "plan", "SLICE-001", "alpha",
		"--strategy", "A=minimal",
		"--strategy", "B=alternate",
		"--acceptance-hash", forgeAcceptanceHash,
		"--test-plan-hash", forgeTestPlanHash,
	}
	out := mustForgeCLI(t, repo, args...)
	var degraded forge.PlanResult
	if err := json.Unmarshal([]byte(out), &degraded); err != nil {
		t.Fatal(err)
	}
	if degraded.Status != "degraded" || degraded.Mode != "serial" || degraded.Manifest != nil {
		t.Fatalf("unbound Forge plan=%+v", degraded)
	}
	if got := forgeGitRaw(t, repo, "worktree", "list", "--porcelain"); !bytes.Equal(got, beforeWorktrees) {
		t.Fatalf("unbound plan changed worktrees:\n%s", got)
	}
	if got := forgeGitRaw(t, repo, "for-each-ref", "--format=%(refname)", "refs/heads/devrites/forge"); !bytes.Equal(got, beforeBranches) {
		t.Fatalf("unbound plan changed branches:\n%s", got)
	}
	if matches, err := filepath.Glob(filepath.Join(repo, ".devrites", "work", "*", ".forge", "*", "manifest.json")); err != nil || len(matches) != 0 {
		t.Fatalf("unbound plan created manifests: matches=%v err=%v", matches, err)
	}

	unknown := append(append([]string{}, args...), "--worker-binding=unknown-v1")
	if _, stderr, code := runDevritesAt(t, repo, repo, "", nil, unknown...); code == 0 || !strings.Contains(stderr, "unsupported worker binding") {
		t.Fatalf("unknown worker binding exit=%d stderr=%s", code, stderr)
	}
	if got := forgeGitRaw(t, repo, "worktree", "list", "--porcelain"); !bytes.Equal(got, beforeWorktrees) {
		t.Fatal("unknown worker binding changed worktrees")
	}

	if manifest := planForgeCLI(t, repo); manifest == nil {
		t.Fatal("supported worker binding did not plan Forge")
	}
}

func TestForgeCLIHappyPathReconcileAndCleanupPreservation(t *testing.T) {
	repo := newForgeCLIRepo(t)
	manifest := planForgeCLI(t, repo)

	if _, stderr, code := runDevritesAt(t, repo, repo, "", nil, "reconcile", "snapshot", "alpha"); code != 0 {
		t.Fatalf("reconcile snapshot exit=%d stderr=%s", code, stderr)
	}

	for _, id := range []forge.CandidateID{forge.CandidateA, forge.CandidateB} {
		candidate, err := manifest.Candidate(id)
		if err != nil {
			t.Fatal(err)
		}
		worker := startForgeCLIWorker(t, repo, manifest.RunID, id, "worker-"+strings.ToLower(string(id)))
		content := "winner\n"
		if id == forge.CandidateB {
			content = "runner-up\n"
		}
		if err := os.WriteFile(filepath.Join(candidate.Worktree, "tracked.txt"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		worker.finish(t, "finished")
		mustForgeCLI(t, repo, "forge", "extract", manifest.RunID, string(id))
	}

	mustForgeCLI(t, repo, "forge", "record", manifest.RunID, "winner", "A", "--worker-id", "judge-1")
	mustForgeCLI(t, repo, "forge", "merge", manifest.RunID, "A")
	if got := forgeReadFile(t, filepath.Join(repo, "tracked.txt")); got != "winner\n" {
		t.Fatalf("landed content=%q", got)
	}
	if stdout, stderr, code := runDevritesAt(t, repo, repo, "", nil, "reconcile", "check", "alpha"); code != 0 {
		t.Fatalf("valid manifest lifecycle tripped reconcile: exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	mustForgeCLI(t, repo, "forge", "record", manifest.RunID, "verification", "verified", "--worker-id", "verifier-1")

	runnerUp, err := manifest.Candidate(forge.CandidateB)
	if err != nil {
		t.Fatal(err)
	}
	ignored := filepath.Join(runnerUp.Worktree, "keep.ignored")
	if err := os.WriteFile(ignored, []byte("preserve\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := mustForgeCLI(t, repo, "forge", "cleanup", manifest.RunID)
	var partial forge.Manifest
	if err := json.Unmarshal([]byte(out), &partial); err != nil {
		t.Fatal(err)
	}
	if partial.Cleanup.State != "partial" || !strings.Contains(partial.Cleanup.Preserved["B"], "dirt") {
		t.Fatalf("dirty runner-up was not preserved: %+v", partial.Cleanup)
	}
	if err := os.Remove(ignored); err != nil {
		t.Fatal(err)
	}
	out = mustForgeCLI(t, repo, "forge", "cleanup", manifest.RunID)
	var complete forge.Manifest
	if err := json.Unmarshal([]byte(out), &complete); err != nil {
		t.Fatal(err)
	}
	if complete.Cleanup.State != "complete" {
		t.Fatalf("cleanup state=%q", complete.Cleanup.State)
	}
	for _, candidate := range manifest.Candidates {
		forgeGit(t, repo, "show-ref", "--verify", "--quiet", "refs/heads/"+candidate.Branch)
	}
}

func TestForgeCLIConflictPreflightLeavesPrimaryByteStable(t *testing.T) {
	repo := newForgeCLIRepo(t)
	manifest := planForgeCLI(t, repo)
	extractForgeCLICandidates(t, repo, manifest)
	mustForgeCLI(t, repo, "forge", "record", manifest.RunID, "winner", "A", "--worker-id", "judge-1")

	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("primary edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	beforeWorktree := forgeGitRaw(t, repo, "diff", "--binary")
	beforeIndex := forgeGitRaw(t, repo, "diff", "--cached", "--binary")
	beforeHead := forgeGitRaw(t, repo, "rev-parse", "HEAD")
	if _, _, code := runDevritesAt(t, repo, repo, "", nil, "forge", "merge", manifest.RunID, "A"); code == 0 {
		t.Fatal("merge succeeded with primary drift")
	}
	if got := forgeGitRaw(t, repo, "diff", "--binary"); !bytes.Equal(got, beforeWorktree) {
		t.Fatal("failed merge changed primary worktree")
	}
	if got := forgeGitRaw(t, repo, "diff", "--cached", "--binary"); !bytes.Equal(got, beforeIndex) {
		t.Fatal("failed merge changed primary index")
	}
	if got := forgeGitRaw(t, repo, "rev-parse", "HEAD"); !bytes.Equal(got, beforeHead) {
		t.Fatal("failed merge changed primary HEAD")
	}
}

func TestForgeCLIReapPreservesInterruptedAndForeignState(t *testing.T) {
	repo := newForgeCLIRepo(t)
	manifest := planForgeCLI(t, repo)
	worker := startForgeCLIWorker(t, repo, manifest.RunID, forge.CandidateA, "worker-a")

	// A repeated started record is the resumable, idempotent handoff.
	mustForgeCLI(t, repo, "forge", "record", manifest.RunID, "A", "started",
		"--worker-id", worker.id,
		"--pid", strconv.Itoa(worker.cmd.Process.Pid),
		"--process-start", worker.token)
	foreign := filepath.Join(filepath.Dir(manifest.ForgeRoot), "foreign-orphan")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	out := mustForgeCLI(t, repo, "forge", "reap", "alpha")
	var results []forge.ReapResult
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Preserved["A"] == "" {
		t.Fatalf("live interrupted run was not preserved: %+v", results)
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Fatalf("foreign state was touched: %v", err)
	}
	worker.finish(t, "failed")
	out = mustForgeCLI(t, repo, "forge", "reap", "alpha")
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatal(err)
	}
	if results[0].Preserved["A"] == "" {
		t.Fatalf("failed unextracted candidate was reaped: %+v", results)
	}
	for _, candidate := range manifest.Candidates {
		forgeGit(t, repo, "show-ref", "--verify", "--quiet", "refs/heads/"+candidate.Branch)
	}
}

func TestForgeReconcileRejectsReportArbitraryContentAndTampering(t *testing.T) {
	t.Run("report and arbitrary content", func(t *testing.T) {
		repo := newForgeCLIRepo(t)
		manifest := planForgeCLI(t, repo)
		mustForgeCLI(t, repo, "reconcile", "snapshot", "alpha")

		report := filepath.Join(repo, ".devrites", "work", "alpha", "forge-report.md")
		if err := os.WriteFile(report, []byte("# premature\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, stderr, code := runDevritesAt(t, repo, repo, "", nil, "reconcile", "check", "alpha"); code != 5 || !strings.Contains(stderr, "forge-report.md") {
			t.Fatalf("premature report was not rejected: exit=%d stderr=%s", code, stderr)
		}
		if err := os.Remove(report); err != nil {
			t.Fatal(err)
		}
		if stdout, stderr, code := runDevritesAt(t, repo, repo, "", nil, "reconcile", "check", "alpha"); code != 0 {
			t.Fatalf("clean pre-report check failed: exit=%d stdout=%s stderr=%s", code, stdout, stderr)
		}
		if err := os.WriteFile(report, []byte("# recorded after check\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(report); err != nil {
			t.Fatalf("post-check report was not recorded: %v", err)
		}
		if err := os.Remove(report); err != nil {
			t.Fatal(err)
		}
		arbitrary := filepath.Join(repo, ".devrites", "work", "alpha", ".forge", "note.txt")
		if err := os.WriteFile(arbitrary, []byte("not operational\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, stderr, code := runDevritesAt(t, repo, repo, "", nil, "reconcile", "check", "alpha"); code != 5 || !strings.Contains(stderr, ".forge/note.txt") {
			t.Fatalf("arbitrary .forge content was hidden: exit=%d stderr=%s", code, stderr)
		}
		_ = manifest
	})

	t.Run("tampered manifest", func(t *testing.T) {
		repo := newForgeCLIRepo(t)
		manifest := planForgeCLI(t, repo)
		mustForgeCLI(t, repo, "reconcile", "snapshot", "alpha")
		path := filepath.Join(repo, ".devrites", "work", "alpha", ".forge", manifest.RunID, "manifest.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		raw = bytes.Replace(raw, []byte(`"feature_slug": "alpha"`), []byte(`"feature_slug": "other"`), 1)
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, stderr, code := runDevritesAt(t, repo, repo, "", nil, "reconcile", "check", "alpha"); code != 6 || !strings.Contains(stderr, "invalid Forge manifest") {
			t.Fatalf("tampered manifest was not rejected: exit=%d stderr=%s", code, stderr)
		}
	})
}

func TestLanesRequiresExactForgeFieldAndDefersAuthorization(t *testing.T) {
	repo := newForgeCLIRepo(t)
	tasks := `## SLICE-001 False positive
Notes: this sentence mentions forge: yes but is not a field.
Files likely touched: ` + "`src/one.go`" + `

## SLICE-002 Real candidate
Forge: yes — two viable seams
Files likely touched: ` + "`src/two.go`" + `
`
	if err := os.WriteFile(filepath.Join(repo, ".devrites", "work", "alpha", "tasks.md"), []byte(tasks), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout := mustForgeCLI(t, repo, "lanes", "plan", "alpha")
	if strings.Contains(stdout, "SLICE-001 False positive: forge-candidate") {
		t.Fatalf("lane parser accepted prose false positive:\n%s", stdout)
	}
	if !strings.Contains(stdout, "SLICE-002 Real candidate: forge-candidate") ||
		!strings.Contains(stdout, "`devrites-engine forge plan` must authorize 2 or 3 strategies") {
		t.Fatalf("lane output implied Forge authorization:\n%s", stdout)
	}
}

type forgeCLIWorker struct {
	repo  string
	runID string
	cand  forge.CandidateID
	id    string
	token string
	cmd   *exec.Cmd
}

func startForgeCLIWorker(t *testing.T, repo, runID string, candidate forge.CandidateID, id string) *forgeCLIWorker {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Forge liveness currently requires ps")
	}
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	var token string
	var err error
	for range 30 {
		token, err = forge.ProcessStartToken(cmd.Process.Pid)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	mustForgeCLI(t, repo, "forge", "record", runID, string(candidate), "started",
		"--worker-id", id,
		"--pid", strconv.Itoa(cmd.Process.Pid),
		"--process-start", token)
	return &forgeCLIWorker{repo: repo, runID: runID, cand: candidate, id: id, token: token, cmd: cmd}
}

func (w *forgeCLIWorker) finish(t *testing.T, state string) {
	t.Helper()
	if w.cmd.ProcessState == nil {
		if err := w.cmd.Process.Kill(); err != nil {
			t.Fatal(err)
		}
		if err := w.cmd.Wait(); err == nil {
			t.Fatal("killed Forge worker exited successfully")
		}
	}
	mustForgeCLI(t, w.repo, "forge", "record", w.runID, string(w.cand), state, "--worker-id", w.id)
}

func extractForgeCLICandidates(t *testing.T, repo string, manifest *forge.Manifest) {
	t.Helper()
	for _, id := range []forge.CandidateID{forge.CandidateA, forge.CandidateB} {
		candidate, err := manifest.Candidate(id)
		if err != nil {
			t.Fatal(err)
		}
		worker := startForgeCLIWorker(t, repo, manifest.RunID, id, "worker-"+strings.ToLower(string(id)))
		if err := os.WriteFile(filepath.Join(candidate.Worktree, "tracked.txt"), []byte(string(id)+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		worker.finish(t, "finished")
		mustForgeCLI(t, repo, "forge", "extract", manifest.RunID, string(id))
	}
}

func planForgeCLI(t *testing.T, repo string) *forge.Manifest {
	t.Helper()
	out := mustForgeCLI(t, repo,
		"forge", "plan", "SLICE-001", "alpha",
		"--strategy", "A=minimal",
		"--strategy", "B=alternate",
		"--acceptance-hash", forgeAcceptanceHash,
		"--test-plan-hash", forgeTestPlanHash,
		"--worker-binding="+forge.WorkerBindingManifestEnvV1)
	var result forge.PlanResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "planned" || result.Manifest == nil {
		t.Fatalf("forge plan=%+v", result)
	}
	return result.Manifest
}

func newForgeCLIRepo(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".devrites", "work", "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	forgeGit(t, repo, "init", "-b", "main")
	forgeGit(t, repo, "config", "user.name", "Forge CLI Test")
	forgeGit(t, repo, "config", "user.email", "forge-cli@example.invalid")
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(".devrites/\n*.ignored\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".devrites", "ACTIVE"), []byte("alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".devrites", "work", "alpha", ".wright-allowlist"), []byte("tracked.txt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	forgeGit(t, repo, "add", ".")
	forgeGit(t, repo, "commit", "-m", "baseline")
	return repo
}

func mustForgeCLI(t *testing.T, repo string, args ...string) string {
	t.Helper()
	stdout, stderr, code := runDevritesAt(t, repo, repo, "", nil, args...)
	if code != 0 {
		t.Fatalf("%s: exit=%d stderr=%s stdout=%s", strings.Join(args, " "), code, stderr, stdout)
	}
	return stdout
}

func runDevritesAt(t *testing.T, repo, cwd, stdin string, extraEnv []string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Dir = cwd
	cmd.Stdin = strings.NewReader(stdin)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "DEVRITES_") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "DEVRITES_ROOT="+filepath.Join(repo, ".devrites"))
	cmd.Env = append(cmd.Env, extraEnv...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run %s: %v", strings.Join(args, " "), err)
		}
		code = exitErr.ExitCode()
	}
	return outBuf.String(), errBuf.String(), code
}

func forgeGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func forgeGitRaw(t *testing.T, dir string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return out
}

func forgeReadFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
