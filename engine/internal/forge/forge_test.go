package forge

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
)

const (
	acceptanceHash     = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testPlanHash       = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	forgeWorkerTestEnv = "DEVRITES_FORGE_TEST_WORKER"
)

func TestForgeFullTreeMergeCleanupAndIdempotence(t *testing.T) {
	repo := newRepo(t)
	result := planFixture(t, repo)
	m := result.Manifest
	a := candidate(t, m, CandidateA)
	b := candidate(t, m, CandidateB)

	worker := startWorker(t, repo, m.RunID, CandidateA, "worker-a")
	writeFile(t, filepath.Join(a.Worktree, "tracked.txt"), []byte("staged\n"), 0o644)
	gitOK(t, a.Worktree, "add", "tracked.txt")
	appendFile(t, filepath.Join(a.Worktree, "tracked.txt"), []byte("unstaged\n"))
	writeFile(t, filepath.Join(a.Worktree, "binary.bin"), []byte{0, 1, 2, 0xff}, 0o644)
	if err := os.Remove(filepath.Join(a.Worktree, "delete.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(a.Worktree, "script.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Symlink("tracked.txt", filepath.Join(a.Worktree, "tracked.link")); err != nil {
			t.Fatal(err)
		}
	}
	worker.finish(t, StateFinished)
	m, err := Extract(repo, m.RunID, CandidateA)
	if err != nil {
		t.Fatal(err)
	}
	a = candidate(t, m, CandidateA)
	if a.State != StateExtracted || a.Commit == "" || a.Tree == "" || a.DeltaSHA256 == "" {
		t.Fatalf("candidate A extraction incomplete: %+v", a)
	}
	if got := gitRaw(t, a.Worktree, "show", a.Commit+":binary.bin"); !bytes.Equal(got, []byte{0, 1, 2, 0xff}) {
		t.Fatalf("binary snapshot=%v", got)
	}
	tree := string(gitRaw(t, a.Worktree, "ls-tree", "-r", a.Commit))
	if !strings.Contains(tree, "100755 blob") || !strings.Contains(tree, "\tscript.sh") {
		t.Fatalf("executable mode missing from tree:\n%s", tree)
	}
	if runtime.GOOS != "windows" && (!strings.Contains(tree, "120000 blob") || !strings.Contains(tree, "\ttracked.link")) {
		t.Fatalf("symlink missing from tree:\n%s", tree)
	}

	worker = startWorker(t, repo, m.RunID, CandidateB, "worker-b")
	writeFile(t, filepath.Join(b.Worktree, "runner-only.txt"), []byte("loser\n"), 0o644)
	worker.finish(t, StateFinished)
	if _, err := Extract(repo, m.RunID, CandidateB); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordWinner(repo, m.RunID, CandidateA, "judge-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := Merge(repo, m.RunID, CandidateA); err != nil {
		t.Fatal(err)
	}
	if _, err := Merge(repo, m.RunID, CandidateA); err != nil {
		t.Fatalf("idempotent merge: %v", err)
	}
	if got := readFile(t, filepath.Join(repo, "tracked.txt")); got != "staged\nunstaged\n" {
		t.Fatalf("winner content=%q", got)
	}
	if _, err := os.Stat(filepath.Join(repo, "runner-only.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("runner-up content landed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "delete.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deletion did not land: %v", err)
	}
	if mode := fileMode(t, filepath.Join(repo, "script.sh")); mode.Perm() != 0o755 {
		t.Fatalf("script mode=%o", mode.Perm())
	}

	if _, err := RecordVerification(repo, m.RunID, true, "verifier-1"); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(b.Worktree, "keep.ignored"), []byte("preserve\n"), 0o600)
	m, err = Cleanup(repo, m.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if m.Cleanup.State != "partial" || !strings.Contains(m.Cleanup.Preserved["B"], "dirt") {
		t.Fatalf("cleanup did not preserve ignored dirt: %+v", m.Cleanup)
	}
	if _, err := os.Stat(a.Worktree); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("clean winner worktree remains: %v", err)
	}
	if _, err := os.Stat(b.Worktree); err != nil {
		t.Fatalf("dirty runner-up was removed: %v", err)
	}
	if err := os.Remove(filepath.Join(b.Worktree, "keep.ignored")); err != nil {
		t.Fatal(err)
	}
	m, err = Cleanup(repo, m.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if m.Cleanup.State != "complete" {
		t.Fatalf("cleanup state=%s", m.Cleanup.State)
	}
	if _, err := Cleanup(repo, m.RunID); err != nil {
		t.Fatalf("idempotent cleanup: %v", err)
	}
	for _, c := range m.Candidates {
		if got := strings.TrimSpace(string(gitRaw(t, repo, "rev-parse", "--verify", "refs/heads/"+c.Branch))); got == "" {
			t.Fatalf("branch %s was deleted", c.Branch)
		}
	}
}

func TestMergeDriftLeavesPrimaryByteIdentical(t *testing.T) {
	repo := newRepo(t)
	m := planFixture(t, repo).Manifest
	for _, id := range []CandidateID{CandidateA, CandidateB} {
		c := candidate(t, m, id)
		worker := startWorker(t, repo, m.RunID, id, "worker-"+strings.ToLower(string(id)))
		writeFile(t, filepath.Join(c.Worktree, "tracked.txt"), []byte(string(id)+"\n"), 0o644)
		worker.finish(t, StateFinished)
		var err error
		m, err = Extract(repo, m.RunID, id)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := RecordWinner(repo, m.RunID, CandidateA, "judge"); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo, "tracked.txt"), []byte("primary edit\n"), 0o644)
	beforeWorktree := gitRaw(t, repo, "diff", "--binary")
	beforeIndex := gitRaw(t, repo, "diff", "--cached", "--binary")
	beforeHead := strings.TrimSpace(string(gitRaw(t, repo, "rev-parse", "HEAD")))
	if _, err := Merge(repo, m.RunID, CandidateA); err == nil {
		t.Fatal("merge succeeded with dirty primary")
	}
	if got := gitRaw(t, repo, "diff", "--binary"); !bytes.Equal(got, beforeWorktree) {
		t.Fatal("failed merge changed primary worktree")
	}
	if got := gitRaw(t, repo, "diff", "--cached", "--binary"); !bytes.Equal(got, beforeIndex) {
		t.Fatal("failed merge changed primary index")
	}
	if got := strings.TrimSpace(string(gitRaw(t, repo, "rev-parse", "HEAD"))); got != beforeHead {
		t.Fatalf("failed merge changed HEAD: %s -> %s", beforeHead, got)
	}
}

func TestManifestPathTamperingAndSymlinkEscapeFailClosed(t *testing.T) {
	t.Run("candidate path", func(t *testing.T) {
		repo := newRepo(t)
		m := planFixture(t, repo).Manifest
		foreign := filepath.Join(t.TempDir(), "foreign")
		if err := os.MkdirAll(foreign, 0o755); err != nil {
			t.Fatal(err)
		}
		m.Candidates[0].Worktree = foreign
		path, err := ManifestPath(repo, m.FeatureSlug, m.RunID)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.MarshalIndent(m, "", "  ")
		writeFile(t, path, append(raw, '\n'), 0o600)
		if _, _, err := Load(repo, m.RunID); err == nil {
			t.Fatal("tampered manifest loaded")
		}
		if _, err := os.Stat(foreign); err != nil {
			t.Fatalf("foreign path was touched: %v", err)
		}
	})

	t.Run("manifest parent symlink", func(t *testing.T) {
		repo := newRepo(t)
		foreign := t.TempDir()
		if err := os.Symlink(foreign, filepath.Join(repo, ".devrites")); err != nil {
			t.Fatal(err)
		}
		if _, err := Plan(repo, planOptions()); err == nil {
			t.Fatal("plan accepted symlinked manifest root")
		}
		entries, err := os.ReadDir(foreign)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Fatalf("symlink target was written: %v", entries)
		}
	})
}

func TestPlanDegradesInsideSubmoduleBeforeSideEffects(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "source")
	parent := filepath.Join(base, "parent")
	initRepo(t, source)
	initRepo(t, parent)
	gitOK(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", source, "modules/child")
	gitOK(t, parent, "commit", "-am", "add submodule")
	child, err := physicalExisting(filepath.Join(parent, "modules/child"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Plan(child, planOptions())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "degraded" || result.Mode != "serial" {
		t.Fatalf("submodule plan=%+v", result)
	}
	if paths, err := manifestPaths(child, "alpha"); err != nil || len(paths) != 0 {
		t.Fatalf("submodule plan wrote manifests: %v %v", paths, err)
	}
	if _, err := os.Stat(forgeRoot(child, "not-a-real-run")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("submodule plan created staging state: %v", err)
	}
}

func TestReapUsesManifestsPreservesLiveAndForeignState(t *testing.T) {
	repo := newRepo(t)
	m := planFixture(t, repo).Manifest
	for _, id := range []CandidateID{CandidateA, CandidateB} {
		c := candidate(t, m, id)
		worker := startWorker(t, repo, m.RunID, id, "worker-"+strings.ToLower(string(id)))
		writeFile(t, filepath.Join(c.Worktree, "candidate-"+string(id)+".txt"), []byte("content\n"), 0o644)
		worker.finish(t, StateFinished)
		var err error
		m, err = Extract(repo, m.RunID, id)
		if err != nil {
			t.Fatal(err)
		}
	}
	token, err := ProcessStartToken(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	b := candidate(t, m, CandidateB)
	b.Worker.PID = os.Getpid()
	b.Worker.ProcessStart = token
	path, err := ManifestPath(repo, m.FeatureSlug, m.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if err := save(path, m); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(filepath.Dir(filepath.Dir(m.ForgeRoot)), "foreign-orphan")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	results, err := Reap(repo, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || len(results[0].Removed) != 1 || results[0].Removed[0] != CandidateA {
		t.Fatalf("reap result=%+v", results)
	}
	if !strings.Contains(results[0].Preserved["B"], "live") {
		t.Fatalf("live candidate not preserved: %+v", results[0])
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Fatalf("foreign orphan was touched: %v", err)
	}
	if _, err := os.Stat(b.Worktree); err != nil {
		t.Fatalf("live worktree was removed: %v", err)
	}
	for _, c := range m.Candidates {
		gitOK(t, repo, "show-ref", "--verify", "--quiet", "refs/heads/"+c.Branch)
	}
}

func TestRunPlanAcceptsPositionalBeforeFlagsAndRecordIsIdempotent(t *testing.T) {
	repo := newRepo(t)
	var stdout, stderr bytes.Buffer
	args := append(planCLIArgs(), "--worker-binding="+WorkerBindingManifestEnvV1)
	code := Run(repo, args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run plan code=%d stderr=%s", code, stderr.String())
	}
	var result PlanResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	worker := startWorker(t, repo, result.Manifest.RunID, CandidateA, "worker-a")
	opts := RecordOptions{
		RunID:        result.Manifest.RunID,
		Candidate:    CandidateA,
		State:        StateRunning,
		WorkerID:     worker.id,
		PID:          worker.cmd.Process.Pid,
		ProcessStart: worker.token,
	}
	if _, err := Record(repo, opts); err != nil {
		t.Fatalf("idempotent running record: %v", err)
	}
	worker.finish(t, StateFinished)
	if _, err := Record(repo, RecordOptions{
		RunID: result.Manifest.RunID, Candidate: CandidateA, State: StateFinished, WorkerID: worker.id,
	}); err != nil {
		t.Fatalf("idempotent finished record: %v", err)
	}
}

func TestRunPlanRequiresExactWorkerBindingBeforeSideEffects(t *testing.T) {
	repo := newRepo(t)
	beforeWorktrees := gitRaw(t, repo, "worktree", "list", "--porcelain")
	beforeBranches := gitRaw(t, repo, "for-each-ref", "--format=%(refname)", "refs/heads/devrites/forge")

	var stdout, stderr bytes.Buffer
	if code := Run(repo, planCLIArgs(), &stdout, &stderr); code != 0 {
		t.Fatalf("unbound plan code=%d stderr=%s", code, stderr.String())
	}
	var result PlanResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "degraded" || result.Mode != "serial" || result.Manifest != nil {
		t.Fatalf("unbound plan=%+v", result)
	}
	assertNoPlanSideEffects(t, repo, beforeWorktrees, beforeBranches)

	stdout.Reset()
	stderr.Reset()
	args := append(planCLIArgs(), "--worker-binding=unknown-v1")
	if code := Run(repo, args, &stdout, &stderr); code == 0 || !strings.Contains(stderr.String(), "unsupported worker binding") {
		t.Fatalf("unknown binding code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	assertNoPlanSideEffects(t, repo, beforeWorktrees, beforeBranches)
}

func TestRunProcessTokenMatchesAPIAndRejectsInvalidPID(t *testing.T) {
	want, err := ProcessStartToken(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run("", []string{"process-token", strconv.Itoa(os.Getpid())}, &stdout, &stderr); code != 0 {
		t.Fatalf("process-token code=%d stderr=%s", code, stderr.String())
	}
	var result ProcessTokenResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" || result.PID != os.Getpid() || result.ProcessStart != want {
		t.Fatalf("process-token=%+v want pid=%d token=%s", result, os.Getpid(), want)
	}

	for _, pid := range []string{"0", "-1", "+1", "01", " 1", "2147483647"} {
		stdout.Reset()
		stderr.Reset()
		if code := Run("", []string{"process-token", pid}, &stdout, &stderr); code == 0 || stdout.Len() != 0 {
			t.Fatalf("process-token %q code=%d stdout=%s stderr=%s", pid, code, stdout.String(), stderr.String())
		}
	}
}

func planCLIArgs() []string {
	return []string{
		"plan", "slice-1", "alpha",
		"--strategy", "A=first", "--strategy", "B=second",
		"--acceptance-hash", acceptanceHash, "--test-plan-hash", testPlanHash,
	}
}

func assertNoPlanSideEffects(t *testing.T, repo string, wantWorktrees, wantBranches []byte) {
	t.Helper()
	if got := gitRaw(t, repo, "worktree", "list", "--porcelain"); !bytes.Equal(got, wantWorktrees) {
		t.Fatalf("plan changed worktrees:\n%s", got)
	}
	if got := gitRaw(t, repo, "for-each-ref", "--format=%(refname)", "refs/heads/devrites/forge"); !bytes.Equal(got, wantBranches) {
		t.Fatalf("plan changed Forge branches:\n%s", got)
	}
	if paths, err := manifestPaths(repo, ""); err != nil || len(paths) != 0 {
		t.Fatalf("plan created manifests: paths=%v err=%v", paths, err)
	}
}

type workerProcess struct {
	repo  string
	runID string
	id    string
	cand  CandidateID
	cmd   *exec.Cmd
	token string
}

func startWorker(t *testing.T, repo, runID string, candidate CandidateID, id string) *workerProcess {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestForgeWorkerProcess$")
	cmd.Env = append(os.Environ(), forgeWorkerTestEnv+"=1")
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
	for range 20 {
		token, err = ProcessStartToken(cmd.Process.Pid)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Record(repo, RecordOptions{
		RunID: runID, Candidate: candidate, State: StateRunning,
		WorkerID: id, PID: cmd.Process.Pid, ProcessStart: token,
	}); err != nil {
		t.Fatal(err)
	}
	return &workerProcess{repo: repo, runID: runID, id: id, cand: candidate, cmd: cmd, token: token}
}

func TestForgeWorkerProcess(t *testing.T) {
	if os.Getenv(forgeWorkerTestEnv) != "1" {
		return
	}
	for {
		time.Sleep(time.Hour)
	}
}

func (w *workerProcess) finish(t *testing.T, state CandidateState) {
	t.Helper()
	if err := w.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := w.cmd.Wait(); err == nil {
		t.Fatal("killed worker exited successfully")
	}
	if _, err := Record(w.repo, RecordOptions{
		RunID: w.runID, Candidate: w.cand, State: state, WorkerID: w.id,
	}); err != nil {
		t.Fatal(err)
	}
}

func planFixture(t *testing.T, repo string) PlanResult {
	t.Helper()
	result, err := Plan(repo, planOptions())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "planned" || result.Manifest == nil {
		t.Fatalf("plan=%+v", result)
	}
	return result
}

func planOptions() PlanOptions {
	return PlanOptions{
		SliceID: "slice-1",
		Slug:    "alpha",
		Strategies: map[CandidateID]string{
			CandidateA: "minimal",
			CandidateB: "alternate",
		},
		AcceptanceHash: acceptanceHash,
		TestPlanHash:   testPlanHash,
	}
}

func candidate(t *testing.T, m *Manifest, id CandidateID) *Candidate {
	t.Helper()
	c, err := m.Candidate(id)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func newRepo(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "repo")
	initRepo(t, repo)
	return repo
}

func initRepo(t *testing.T, repo string) {
	t.Helper()
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitOK(t, repo, "init", "-b", "main")
	gitOK(t, repo, "config", "user.name", "Test User")
	gitOK(t, repo, "config", "user.email", "test@example.invalid")
	writeFile(t, filepath.Join(repo, ".gitignore"), []byte(".devrites/\n*.ignored\n"), 0o644)
	writeFile(t, filepath.Join(repo, "tracked.txt"), []byte("base\n"), 0o644)
	writeFile(t, filepath.Join(repo, "delete.txt"), []byte("delete me\n"), 0o644)
	writeFile(t, filepath.Join(repo, "script.sh"), []byte("#!/bin/sh\n"), 0o644)
	gitOK(t, repo, "add", ".")
	gitOK(t, repo, "commit", "-m", "baseline")
}

func gitOK(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func gitRaw(t *testing.T, dir string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return out
}

func writeFile(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
}

func appendFile(t *testing.T, path string, data []byte) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode()
}
