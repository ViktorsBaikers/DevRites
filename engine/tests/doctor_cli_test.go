package main_test

// CLI coverage for Issue 05: the `doctor` version triangle.

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorPrintsTriangle(t *testing.T) {
	root := newWorkspace(t)
	out, errOut, code := runDevrites(t, root, "doctor")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	for _, leg := range []string{"project:", "root:", "root-selection:", "binary:", "pack:", "state-schema:", "verdict:"} {
		if !strings.Contains(out, leg) {
			t.Errorf("doctor output missing %q leg\n%s", leg, out)
		}
	}
}

func TestDoctorVerboseMatchesPlainDoctor(t *testing.T) {
	root := newWorkspace(t)
	plainOut, plainErr, plainCode := runDevrites(t, root, "doctor")
	verboseOut, verboseErr, verboseCode := runDevrites(t, root, "doctor", "--verbose")
	if plainCode != 0 || verboseCode != 0 {
		t.Fatalf("doctor exits: plain=%d verbose=%d", plainCode, verboseCode)
	}
	if verboseCode != plainCode || verboseOut != plainOut || verboseErr != plainErr {
		t.Fatalf(
			"doctor --verbose differs from doctor:\nplain: code=%d stdout=%q stderr=%q\nverbose: code=%d stdout=%q stderr=%q",
			plainCode, plainOut, plainErr, verboseCode, verboseOut, verboseErr,
		)
	}
}

func TestDoctorReportsStaleActiveWithPasteableRepair(t *testing.T) {
	root := newWorkspace(t)
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte("ghost\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errOut, code := runDevrites(t, root, "doctor")
	if code != 0 {
		t.Fatalf("exit = %d, want warning exit 0 (stderr: %s)\n%s", code, errOut, out)
	}
	if !strings.Contains(out, "[DRV-ACTIVE-STALE]") || !strings.Contains(out, "fix: rm -f") {
		t.Fatalf("doctor did not diagnose stale ACTIVE with remediation:\n%s", out)
	}
}

func TestDoctorRefusesParentStateAcrossNestedRepositoryBoundary(t *testing.T) {
	outer := doctorRepo(t, filepath.Join(t.TempDir(), "outer"))
	if err := os.MkdirAll(filepath.Join(outer, ".devrites"), 0o755); err != nil {
		t.Fatal(err)
	}
	child := doctorRepo(t, filepath.Join(outer, "services", "child"))
	out, errOut, code := runDoctorAt(t, child)
	if code != 3 {
		t.Fatalf("exit = %d, want unsafe-root refusal (stderr: %s)\n%s", code, errOut, out)
	}
	if !strings.Contains(out, "[DRV-ROOT-OUTSIDE-GIT]") || !strings.Contains(out, "fix: cd ") {
		t.Fatalf("doctor did not explain nested-repository boundary:\n%s", out)
	}
}

func TestDoctorKeepsOrdinaryRootAbsenceDiagnosable(t *testing.T) {
	repo := doctorRepo(t, filepath.Join(t.TempDir(), "repo"))
	out, errOut, code := runDoctorAt(t, repo)
	if code != 0 {
		t.Fatalf("exit = %d, want diagnostic exit 0 (stderr: %s)\n%s", code, errOut, out)
	}
	if !strings.Contains(out, "root: none") || !strings.Contains(out, "[DRV-ROOT-NOT-FOUND]") {
		t.Fatalf("doctor hid ordinary root absence:\n%s", out)
	}
}

func TestDoctorRefusesWorkspaceOverrideWithoutRoot(t *testing.T) {
	repo := doctorRepo(t, filepath.Join(t.TempDir(), "repo"))
	out, errOut, code := runDoctorAtEnv(t, repo, "DEVRITES_WORKSPACE="+t.TempDir())
	if code != 3 {
		t.Fatalf("exit = %d, want unsafe-root refusal (stderr: %s)\n%s", code, errOut, out)
	}
	if !strings.Contains(out, "[DRV-WORKSPACE-WITHOUT-ROOT]") || !strings.Contains(out, "fix: unset DEVRITES_WORKSPACE") {
		t.Fatalf("doctor hid workspace-without-root remediation:\n%s", out)
	}
}

func TestDoctorReportsLinkedWorktreeAndSubmoduleTopology(t *testing.T) {
	t.Run("linked worktree", func(t *testing.T) {
		base := t.TempDir()
		main := doctorRepo(t, filepath.Join(base, "main"))
		worktree := filepath.Join(base, "linked")
		doctorGit(t, main, "worktree", "add", "-q", "-b", "doctor-linked", worktree)
		if err := os.MkdirAll(filepath.Join(worktree, ".devrites"), 0o755); err != nil {
			t.Fatal(err)
		}
		out, errOut, code := runDoctorAt(t, worktree)
		if code != 0 {
			t.Fatalf("exit = %d stderr=%s\n%s", code, errOut, out)
		}
		if !strings.Contains(out, "linked-worktree=true") {
			t.Fatalf("doctor omitted linked-worktree topology:\n%s", out)
		}
	})

	t.Run("submodule", func(t *testing.T) {
		base := t.TempDir()
		origin := doctorRepo(t, filepath.Join(base, "origin"))
		super := doctorRepo(t, filepath.Join(base, "super"))
		doctorGit(t, super, "-c", "protocol.file.allow=always", "submodule", "add", "-q", origin, "deps/sub")
		sub := filepath.Join(super, "deps", "sub")
		if err := os.MkdirAll(filepath.Join(sub, ".devrites"), 0o755); err != nil {
			t.Fatal(err)
		}
		out, errOut, code := runDoctorAt(t, sub)
		if code != 0 {
			t.Fatalf("exit = %d stderr=%s\n%s", code, errOut, out)
		}
		if !strings.Contains(out, "submodule=true") || !strings.Contains(out, "git-superproject:") {
			t.Fatalf("doctor omitted submodule topology:\n%s", out)
		}
	})
}

func TestDoctorReportsMergeInProgress(t *testing.T) {
	root := newWorkspace(t)
	git := filepath.Join(filepath.Dir(root), ".git")
	if err := os.MkdirAll(git, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(git, "MERGE_HEAD"), []byte("abc123\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, errOut, code := runDevrites(t, root, "doctor")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if !strings.Contains(out, "git-state: merge in progress") || !strings.Contains(out, "git-workflow.md#merge-conflict-recovery") {
		t.Fatalf("doctor did not point to merge conflict recovery\n%s", out)
	}
}

func TestDoctorRefusesNewerStateSchema(t *testing.T) {
	root := newWorkspace(t)
	// Drop in a feature declaring a far-newer schema than the binary supports.
	dir := filepath.Join(root, "features", "from-the-future")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	fm := "---\nslug: from-the-future\nphase: build\nschemaVersion: 99\n---\n\nx\n"
	if err := os.WriteFile(filepath.Join(dir, "feature.md"), []byte(fm), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code := runDevrites(t, root, "doctor")
	if code != 3 {
		t.Fatalf("exit = %d, want 3 (refuse) for a newer-major state schema\n%s", code, out)
	}
	if !strings.Contains(out, "REFUSE") {
		t.Errorf("verdict not a REFUSE\n%s", out)
	}
}

func runDoctorAt(t *testing.T, dir string) (stdout, stderr string, code int) {
	t.Helper()
	return runDoctorAtEnv(t, dir)
}

func runDoctorAtEnv(t *testing.T, dir string, extraEnv ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(binPath, "doctor")
	cmd.Dir = dir
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, "DEVRITES_ROOT=") || strings.HasPrefix(item, "DEVRITES_WORKSPACE=") {
			continue
		}
		cmd.Env = append(cmd.Env, item)
	}
	cmd.Env = append(cmd.Env, extraEnv...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("doctor: %v", err)
		}
		code = exitErr.ExitCode()
	}
	return outBuf.String(), errBuf.String(), code
}

func doctorRepo(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	doctorGit(t, path, "init", "-q")
	doctorGit(t, path, "config", "user.email", "devrites@example.invalid")
	doctorGit(t, path, "config", "user.name", "DevRites Test")
	if err := os.WriteFile(filepath.Join(path, ".fixture"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	doctorGit(t, path, "add", ".fixture")
	doctorGit(t, path, "commit", "-q", "-m", "fixture")
	return path
}

func doctorGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}
