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
