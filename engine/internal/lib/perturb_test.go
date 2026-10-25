package lib

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/acceptance"
)

// perturbCheck needs the dirty and untracked candidate: a probe run on a HEAD
// copy fails setup instead of reaching GREEN.
const perturbCheck = `grep -q 'guard v2' impl.txt && test -f new.txt || { echo 'setup: candidate missing'; exit 2; }
case "$(cat impl.txt)" in
*styled*) printf '  \033[31m✕\033[39m rejects \033[1malias\033[22m (3 ms)\n'; exit 1;;
*broken*) echo '    --- FAIL: TestAlias (0.00s)'; echo FAIL; exit 1;;
*other*) echo '--- FAIL: TestRacing (0.00s)'; exit 1;;
*nobuild*) echo './impl.go:3:1: syntax error'; echo 'FAIL example [build failed]'; exit 1;;
esac
echo 'ok all'
`

func perturbMutant(word string) string {
	return "diff --git a/impl.txt b/impl.txt\n--- a/impl.txt\n+++ b/impl.txt\n@@ -1 +1 @@\n-guard v2\n+guard v2 " + word + "\n"
}

// perturbTestWorkspace commits impl.txt and check.sh, then leaves a modified
// impl.txt and an untracked new.txt as the manifest candidate. nested puts the
// git toplevel one directory above the project.
func perturbTestWorkspace(t *testing.T, nested bool) (project, root, workspace string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("gate CHECK fixtures use /bin/sh")
	}
	project, root = writeCandidateTestWorkspace(t, candidateTestManifest(
		candidateTestRow{state: "present", path: "impl.txt", slice: "SLICE-001", reason: "Guard."},
		candidateTestRow{state: "present", path: "new.txt", slice: "SLICE-001", reason: "Added."},
	))
	workspace = filepath.Join(root, "work", "feature")
	if nested {
		initDupRepo(t, filepath.Dir(project))
	} else {
		initDupRepo(t, project)
	}
	writeCandidateTestFile(t, project, "impl.txt", "guard\n")
	writeCandidateTestFile(t, project, "check.sh", perturbCheck)
	runDupGit(t, project, "add", "impl.txt", "check.sh")
	runDupGit(t, project, "commit", "-q", "-m", "chore: baseline")
	writeCandidateTestFile(t, project, "impl.txt", "guard v2\n")
	writeCandidateTestFile(t, project, "new.txt", "fresh\n")
	drift := filepath.Join(project, "new.txt")
	writeFile(t, filepath.Join(workspace, "gates.md"), "# Gates\n\n"+
		"- [ ] G-1: suite passes\n  CHECK: `sh check.sh`\n  EXPECT: ok all\n\n"+
		"- [ ] G-2: unvetted\n  CHECK: `sh check.sh --all`\n  EXPECT: ok all\n\n"+
		"- [ ] G-3: drifts the checkout\n  CHECK: `sh check.sh; s=$?; echo drift >> "+drift+"; exit $s`\n  EXPECT: ok all\n")
	writeFile(t, filepath.Join(workspace, "test-plan.md"), "# Test plan\n\n## Build-entry preflight\n\n"+
		"| Gate | Command | Cwd | Expected |\n| --- | --- | --- | --- |\n"+
		"| G-1 | `sh check.sh` | repository | ok all |\n"+
		"| G-3 | `sh check.sh; s=$?; echo drift >> "+drift+"; exit $s` | repository | ok all |\n")
	return project, root, workspace
}

func runPerturbT(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunCheckPerturb(root, append([]string{"feature"}, args...), stdout, stderr)
	return code, stdout.String() + stderr.String()
}

func TestCheckPerturbProbesAnIsolatedCandidateCopy(t *testing.T) {
	cases := []struct {
		name, gate, mutant, expect string
		code                       int
		want                       []string
	}{
		{"attributable RED", "G-1", perturbMutant("broken"), "TestAlias", 0,
			[]string{"VERDICT: verified", "RED_MATCH: --- FAIL: TestAlias (0.00s)", "GREEN_MET: true", "PATCH_FILES: impl.txt"}},
		{"styled attributable RED", "G-1", perturbMutant("styled"), "rejects alias", 0,
			[]string{"VERDICT: verified", "RED_MATCH: \x1b[31m✕\x1b[39m rejects \x1b[1malias\x1b[22m (3 ms)\n", "--- RED OUTPUT (tail) ---\n  \x1b[31m✕\x1b[39m rejects \x1b[1malias\x1b[22m (3 ms)\n", "RED_FAILED: rejects alias\n"}},
		{"RED from another test", "G-1", perturbMutant("other"), "TestAlias", 3,
			[]string{"VERDICT: cannot_verify", "RED names other tests (TestRacing), not TestAlias", "GREEN_EXIT: not run"}},
		{"build failure", "G-1", perturbMutant("nobuild"), "TestAlias", 3,
			[]string{"VERDICT: cannot_verify", "no runner FAIL line naming a test"}},
		{"surviving mutant", "G-1", perturbMutant("harmless"), "TestAlias", 3,
			[]string{"VERDICT: cannot_verify", "the mutant survived: the gate exited 0"}},
		{"patch does not apply", "G-1", strings.Replace(perturbMutant("broken"), "-guard v2", "-guard v9", 1), "TestAlias", 3,
			[]string{"VERDICT: cannot_verify", "does not apply to the candidate copy", "RED_EXIT: not run"}},
		{"checkout drift", "G-3", perturbMutant("broken"), "TestAlias", 3,
			[]string{"VERDICT: cannot_verify", "the checkout candidate changed during the probe"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			project, root, workspace := perturbTestWorkspace(t, false)
			writeFile(t, filepath.Join(workspace, "packets", "mutant.patch"), tc.mutant)
			before, _, err := CandidateIdentity(root, "feature")
			if err != nil {
				t.Fatal(err)
			}
			code, out := runPerturbT(t, root, "--patch", ".devrites/work/feature/packets/mutant.patch", "--expect", tc.expect, "--gate", tc.gate)
			if code != tc.code || !strings.Contains(out, "transcript: .devrites/work/feature/evidence/perturb/"+tc.gate+"-"+before[:12]+"-a1.log") {
				t.Fatalf("code=%d want %d\n%s", code, tc.code, out)
			}
			transcript := readTestFile(t, filepath.Join(workspace, "evidence", "perturb", tc.gate+"-"+before[:12]+"-a1.log"))
			for _, want := range append(tc.want, "CANDIDATE_BEFORE: "+before, "PATCH: .devrites/work/feature/packets/mutant.patch") {
				if !strings.Contains(transcript, want) {
					t.Fatalf("transcript lacks %q:\n%s", want, transcript)
				}
			}
			if got := readTestFile(t, filepath.Join(project, "impl.txt")); got != "guard v2\n" {
				t.Fatalf("the probe mutated the checkout: impl.txt = %q", got)
			}
			if worktrees := string(mustGitOutput(t, project, "worktree", "list", "--porcelain")); strings.Count(worktrees, "worktree ") != 1 {
				t.Fatalf("the probe left its worktree registered:\n%s", worktrees)
			}
		})
	}
}

// A project below the git toplevel gets its candidate and gate CWD at the same
// prefix in the copy, while the patch keeps toplevel-relative paths.
func TestCheckPerturbProbesASubdirectoryProject(t *testing.T) {
	project, root, workspace := perturbTestWorkspace(t, true)
	mutant := strings.ReplaceAll(perturbMutant("broken"), "/impl.txt", "/"+filepath.Base(project)+"/impl.txt")
	writeFile(t, filepath.Join(workspace, "packets", "mutant.patch"), mutant)
	code, out := runPerturbT(t, root, "--patch", ".devrites/work/feature/packets/mutant.patch", "--expect", "TestAlias", "--gate", "G-1")
	if code != 0 || !strings.Contains(out, "transcript: ") {
		t.Fatalf("code=%d\n%s", code, out)
	}
	if got := readTestFile(t, filepath.Join(project, "impl.txt")); got != "guard v2\n" {
		t.Fatalf("the probe mutated the checkout: impl.txt = %q", got)
	}
}

// A parallel sibling's candidate lives only in its linked worktree: --worktree
// probes that tree's HEAD plus the slice write set, with --patch resolved there.
func TestCheckPerturbProbesALinkedWorktree(t *testing.T) {
	project, root, workspace := perturbTestWorkspace(t, false)
	writeFile(t, filepath.Join(workspace, "tasks.md"), "# Tasks\n\n## SLICE-001 A\nDependencies: none\nWriter allowlist: impl.txt; new.txt\n")
	wt := filepath.Join(resolvedTempDir(t), "wt")
	runDupGit(t, project, "worktree", "add", "-q", "-b", "sibling", wt, "HEAD")
	writeCandidateTestFile(t, wt, "impl.txt", "guard v2\n")
	writeCandidateTestFile(t, wt, "new.txt", "fresh\n")
	runDupGit(t, wt, "add", "impl.txt", "new.txt")
	runDupGit(t, wt, "commit", "-q", "-m", "feat: sibling")
	writeCandidateTestFile(t, wt, "mutant.patch", perturbMutant("broken"))
	// The control checkout is back at the base, so only the worktree can pass.
	writeCandidateTestFile(t, project, "impl.txt", "guard\n")
	if err := os.Remove(filepath.Join(project, "new.txt")); err != nil {
		t.Fatal(err)
	}

	probe := []string{"--patch", "mutant.patch", "--expect", "TestAlias", "--gate", "G-1"}
	for _, args := range [][]string{append(probe, "--worktree", wt), append(probe, "--slice", "SLICE-001")} {
		if code, _ := runPerturbT(t, root, args...); code != 2 {
			t.Fatalf("%v: want usage exit 2, got %d", args, code)
		}
	}
	if code, out := runPerturbT(t, root, append(probe, "--worktree", project, "--slice", "SLICE-001")...); code != 3 || !strings.Contains(out, "refused: ") {
		t.Fatalf("the control checkout is not a linked worktree: code=%d\n%s", code, out)
	}
	code, out := runPerturbT(t, root, append(probe, "--worktree", wt, "--slice", "SLICE-001")...)
	if code != 0 || !strings.Contains(out, "transcript: .devrites/work/feature/evidence/perturb/G-1-") {
		t.Fatalf("code=%d\n%s", code, out)
	}
	if got := readTestFile(t, filepath.Join(wt, "impl.txt")); got != "guard v2\n" {
		t.Fatalf("the probe mutated the worktree: impl.txt = %q", got)
	}
	if worktrees := string(mustGitOutput(t, project, "worktree", "list", "--porcelain")); strings.Count(worktrees, "worktree ") != 2 {
		t.Fatalf("the probe left its copy registered:\n%s", worktrees)
	}
}

func TestCheckPerturbRefusesBeforeRunning(t *testing.T) {
	_, root, workspace := perturbTestWorkspace(t, false)
	writeFile(t, filepath.Join(workspace, "packets", "mutant.patch"), perturbMutant("broken"))
	patch := []string{"--patch", ".devrites/work/feature/packets/mutant.patch", "--expect", "TestAlias"}
	for _, tc := range []struct {
		args []string
		want string
	}{
		{append(patch, "--gate", "G-2"), "not an approved test-plan.md preflight row"},
		{append(patch, "--gate", "G-9"), "has no gate G-9"},
		{[]string{"--patch", "missing.patch", "--expect", "TestAlias", "--gate", "G-1"}, "--patch missing.patch"},
	} {
		if code, out := runPerturbT(t, root, tc.args...); code != 3 || !strings.Contains(out, "refused: ") || !strings.Contains(out, tc.want) {
			t.Fatalf("%v: code=%d\n%s", tc.args, code, out)
		}
	}
	if _, err := os.Stat(filepath.Join(workspace, "evidence")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused probe wrote evidence: %v", err)
	}
	for _, args := range [][]string{
		{"--expect", "TestAlias", "--gate", "G-1"},
		{"--patch", "p", "--gate", "G-1"},
		{"--patch", "p", "--expect", "TestAlias"},
		{"--patch", "p", "--expect", "TestAlias", "--gate", "G-1", "--timeout", "0"},
		{"--patch", "p", "--expect", "TestAlias", "--gate", "G-1", "--bogus"},
	} {
		if code, _ := runPerturbT(t, root, args...); code != 2 {
			t.Fatalf("%v: want usage exit 2, got %d", args, code)
		}
	}
}

func TestClassifyPerturbRed(t *testing.T) {
	cases := []struct {
		name, output, expect string
		exit                 int
		err                  error
		match, reason        string
	}{
		{"go subtest", "=== RUN TestA\n    --- FAIL: TestA/alias (0.00s)\n--- FAIL: TestA (0.00s)\nFAIL\n", "TestA/alias", 1, nil, "--- FAIL: TestA/alias (0.00s)", ""},
		{"pytest node id", "FAILED tests/test_a.py::TestAlias::test_guard - assert 1 == 2\n", "test_guard", 1, nil, "FAILED tests/test_a.py::TestAlias::test_guard - assert 1 == 2", ""},
		{"cargo", "test store::alias_guard ... FAILED\n", "alias_guard", 101, nil, "test store::alias_guard ... FAILED", ""},
		{"jest", "  ✕ rejects alias (3 ms)\n", "rejects alias", 1, nil, "✕ rejects alias (3 ms)", ""},
		{"vitest", " × store > rejects alias 4ms\n", "rejects alias", 1, nil, "× store > rejects alias 4ms", ""},
		{"exit 0", "ok\n", "TestA", 0, nil, "", "the mutant survived"},
		{"timeout", "", "TestA", 0, errors.New("check timed out after 1s"), "", "did not complete"},
		{"other test", "--- FAIL: TestB (0.00s)\n", "TestA", 1, nil, "", "names other tests (TestB)"},
		{"prefix is not a match", "--- FAIL: TestAlias2 (0.00s)\n", "TestAlias", 1, nil, "", "names other tests"},
		{"unknown runner", "boom\n", "TestA", 2, nil, "", "no runner FAIL line"},
	}
	for _, tc := range cases {
		match, _, reason := classifyPerturbRed(acceptance.RunOutcome{Output: tc.output, ExitCode: tc.exit, Err: tc.err}, tc.expect)
		if match != tc.match || (tc.reason == "") != (reason == "") || !strings.Contains(reason, tc.reason) {
			t.Fatalf("%s: match=%q reason=%q", tc.name, match, reason)
		}
	}
}

func TestClassifyPerturbRedSGR(t *testing.T) {
	cases := []struct {
		name, line, expect, failed string
	}{
		{"go", "\x1b[31m--- FAIL: Test\x1b[1mAlias\x1b[0m (0.00s)", "TestAlias", "TestAlias"},
		{"pytest", "\x1b[31mFAILED\x1b[0m tests/test_a.py::test_\x1b[1mguard\x1b[0m - assert 1 == 2", "test_guard", "tests/test_a.py::test_guard"},
		{"cargo", "test store::alias_\x1b[1mguard\x1b[0m ... \x1b[31mFAILED\x1b[0m", "alias_guard", "store::alias_guard"},
		{"jest", "  \x1b[31m✕\x1b[39m rejects \x1b[1malias\x1b[22m (3 ms)", "rejects alias", "rejects alias"},
		{"vitest", " \x1b[38;2;255;0;0m×\x1b[0m store > rejects \x1b[38;5;196malias\x1b[m 4ms", "rejects alias", "store > rejects alias"},
		{"colon SGR", "\x1b[38:2::255:0:0m×\x1b[0m rejects alias 4ms", "rejects alias", "rejects alias"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outcome := acceptance.RunOutcome{Output: tc.line + "\r\n", ExitCode: 1}
			match, failed, reason := classifyPerturbRed(outcome, tc.expect)
			if match != strings.TrimSpace(tc.line) || len(failed) != 1 || failed[0] != tc.failed || reason != "" {
				t.Fatalf("match=%q failed=%q reason=%q", match, failed, reason)
			}
			if outcome.Output != tc.line+"\r\n" {
				t.Fatalf("raw output changed: %q", outcome.Output)
			}
		})
	}
}

func TestClassifyPerturbRedRefusesStyledNonTargetFailures(t *testing.T) {
	cases := []struct {
		name, output, reason string
		exit                 int
		err                  error
	}{
		{"other test", "\x1b[31m✕\x1b[0m rejects other (3 ms)\n", "names other tests (rejects other)", 1, nil},
		{"build", "\x1b[31mFAIL example [build failed]\x1b[0m\n", "no runner FAIL line", 1, nil},
		{"setup", "\x1b[31mError: setup failed for rejects alias\x1b[0m\n", "no runner FAIL line", 1, nil},
		{"exit zero", "\x1b[31m✕ rejects alias\x1b[0m\n", "the mutant survived", 0, nil},
		{"timeout", "\x1b[31m✕ rejects alias\x1b[0m\n", "did not complete", 1, errors.New("check timed out after 1s")},
		{"execution error", "\x1b[31m✕ rejects alias\x1b[0m\n", "did not complete", 1, errors.New("execution failed")},
		{"erase control", "\x1b[2K✕ rejects alias\n", "no runner FAIL line", 1, nil},
		{"cursor control", "\x1b[1G✕ rejects alias\n", "no runner FAIL line", 1, nil},
		{"OSC control", "\x1b]0;title\a✕ rejects alias\n", "no runner FAIL line", 1, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			match, _, reason := classifyPerturbRed(acceptance.RunOutcome{Output: tc.output, ExitCode: tc.exit, Err: tc.err}, "rejects alias")
			if match != "" || !strings.Contains(reason, tc.reason) {
				t.Fatalf("match=%q reason=%q", match, reason)
			}
		})
	}
}
