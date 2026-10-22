package lib

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runFreezeT(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunCandidateFreeze(root, append([]string{"feature"}, args...), stdout, stderr)
	return code, stdout.String() + stderr.String()
}

func runPacketT(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunPacket(root, append([]string{"feature"}, args...), stdout, stderr)
	return code, stdout.String() + stderr.String()
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestCandidateFreezeWritesApplicableDiffAndCandidateDigest(t *testing.T) {
	project, root := writeCandidateTestWorkspace(t, "")
	workspace := filepath.Join(root, "work", "feature")
	initDupRepo(t, project)
	writeCandidateTestFile(t, project, "mod.txt", "old\n")
	writeCandidateTestFile(t, project, "gone.txt", "bye\n")
	writeCandidateTestFile(t, project, "same.txt", "same\n")
	runDupGit(t, project, "add", "mod.txt", "gone.txt", "same.txt")
	runDupGit(t, project, "commit", "-q", "-m", "chore: baseline")
	writeCandidateTestFile(t, project, "mod.txt", "new\n")
	if err := os.Remove(filepath.Join(project, "gone.txt")); err != nil {
		t.Fatal(err)
	}
	writeCandidateTestFile(t, project, "new.txt", "fresh\n")

	code, out := runFreezeT(t, root, "--wave", "r1", "--paths", "new.txt", "mod.txt", "gone.txt", "never.txt", "mod.txt")
	if code != 0 || !strings.Contains(out, "diff: .devrites/work/feature/packets/r1.diff") {
		t.Fatalf("freeze: code=%d\n%s", code, out)
	}
	diffPath := filepath.Join(workspace, "packets", "r1.diff")
	diff := readTestFile(t, diffPath)
	for _, want := range []string{"a/mod.txt", "deleted file mode", "a/gone.txt", "new file mode", "b/new.txt"} {
		if !strings.Contains(diff, want) {
			t.Fatalf("diff lacks %q:\n%s", want, diff)
		}
	}
	// The diff is exactly base -> worktree: it reverse-applies to the worktree.
	runDupGit(t, project, "apply", "--check", "-R", diffPath)

	sums := readTestFile(t, filepath.Join(workspace, "packets", "r1.sha256"))
	head := strings.TrimSpace(string(mustGitOutput(t, project, "rev-parse", "HEAD")))
	wantRows := []string{
		"devrites-candidate-sha256 v1",
		"base " + head,
		fmt.Sprintf("diff %x packets/r1.diff", sha256.Sum256([]byte(diff))),
		"file absent gone.txt",
		fmt.Sprintf("file %x mod.txt", sha256.Sum256([]byte("new\n"))),
		"file absent never.txt",
		fmt.Sprintf("file %x new.txt", sha256.Sum256([]byte("fresh\n"))),
	}
	lines := strings.Split(strings.TrimSuffix(sums, "\n"), "\n")
	if len(lines) != len(wantRows)+1 || strings.Join(lines[:len(wantRows)], "\n") != strings.Join(wantRows, "\n") {
		t.Fatalf("sha256 rows:\n%s\nwant prefix:\n%s", sums, strings.Join(wantRows, "\n"))
	}
	// The aggregate is the check candidate digest of the same rows.
	manifest := candidateTestManifest(
		candidateTestRow{state: "deleted", path: "gone.txt", slice: "S-1", reason: "Removed."},
		candidateTestRow{state: "present", path: "mod.txt", slice: "S-1", reason: "Changed."},
		candidateTestRow{state: "deleted", path: "never.txt", slice: "S-1", reason: "Absent."},
		candidateTestRow{state: "present", path: "new.txt", slice: "S-1", reason: "Added."},
	)
	writeFile(t, filepath.Join(workspace, "touched-files.md"), manifest)
	digest, _, err := CandidateIdentity(root, "feature")
	if err != nil || lines[len(lines)-1] != "aggregate "+digest {
		t.Fatalf("aggregate %q, check candidate %q (%v)", lines[len(lines)-1], digest, err)
	}

	if code, out := runFreezeT(t, root, "--wave", "r1", "--paths", "mod.txt", "gone.txt", "new.txt", "never.txt"); code != 0 {
		t.Fatalf("an identical refreeze is a no-op: code=%d\n%s", code, out)
	}
	writeCandidateTestFile(t, project, "mod.txt", "newer\n")
	if code, out := runFreezeT(t, root, "--wave", "r1", "--paths", "mod.txt"); code != 3 || !strings.Contains(out, "already freezes a different candidate") {
		t.Fatalf("a changed candidate must not replace a frozen wave: code=%d\n%s", code, out)
	}
	if readTestFile(t, diffPath) != diff {
		t.Fatal("a refused refreeze rewrote the frozen diff")
	}
	if code, out := runFreezeT(t, root, "--wave", "r2", "--paths", "same.txt"); code != 3 || !strings.Contains(out, "no change against the base") {
		t.Fatalf("an unchanged path set has nothing to freeze: code=%d\n%s", code, out)
	}
	if code, out := runFreezeT(t, root, "--wave", "r2", "--paths", "../escape.txt"); code != 3 || !strings.Contains(out, "traversal") {
		t.Fatalf("traversal must be refused: code=%d\n%s", code, out)
	}
	if code, out := runFreezeT(t, root, "--wave", "r2", "--paths", ".devrites/work/feature/state.md"); code != 3 || !strings.Contains(out, "workflow state") {
		t.Fatalf("workflow state is never a candidate path: code=%d\n%s", code, out)
	}
}

func TestCandidateFreezeTakesSliceAllowlist(t *testing.T) {
	project, root := writeCandidateTestWorkspace(t, "")
	workspace := filepath.Join(root, "work", "feature")
	initDupRepo(t, project)
	writeCandidateTestFile(t, project, "src/a.go", "package a\n")
	writeCandidateTestFile(t, project, "src/b.go", "package b\n")
	runDupGit(t, project, "add", "src")
	runDupGit(t, project, "commit", "-q", "-m", "chore: baseline")
	writeCandidateTestFile(t, project, "src/a.go", "package a // changed\n")
	writeCandidateTestFile(t, project, "src/b.go", "package b // other slice\n")
	writeFile(t, filepath.Join(workspace, "tasks.md"), "# Tasks\n\n## SLICE-001 A\nDependencies: none\nWriter allowlist: src/a.go\n\n## SLICE-002 B\nDependencies: none\nWriter allowlist: src/b.go\n")

	if code, out := runFreezeT(t, root, "--wave", "s1", "--slice", "SLICE-001"); code != 0 || !strings.Contains(out, "(1 paths,") {
		t.Fatalf("slice freeze: code=%d\n%s", code, out)
	}
	diff := readTestFile(t, filepath.Join(workspace, "packets", "s1.diff"))
	if !strings.Contains(diff, "src/a.go") || strings.Contains(diff, "src/b.go") {
		t.Fatalf("slice freeze must cover only its Writer allowlist:\n%s", diff)
	}
	if code, out := runFreezeT(t, root, "--wave", "s2", "--slice", "SLICE-009"); code != 3 || !strings.Contains(out, "names no Writer allowlist") {
		t.Fatalf("unknown slice: code=%d\n%s", code, out)
	}
	for _, args := range [][]string{
		{"--wave", "s3", "--slice", "SLICE-001", "--paths", "src/a.go"},
		{"--wave", "s3"},
		{"--wave", "../s3", "--slice", "SLICE-001"},
		{"--slice", "SLICE-001"},
	} {
		if code, _ := runFreezeT(t, root, args...); code != 2 {
			t.Fatalf("%v: want usage exit 2, got %d", args, code)
		}
	}
}

// resolvedTempDir is t.TempDir() with symlinks resolved (macOS /var is one):
// checkedWorktree refuses a symlinked spelling.
func resolvedTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// linkedWorktreeRepo commits src/a.go in a fresh candidate workspace and adds
// a linked worktree on its own branch, as parallel create does for a sibling.
func linkedWorktreeRepo(t *testing.T) (project, root, base, wt string) {
	t.Helper()
	project, root = writeCandidateTestWorkspace(t, "")
	initDupRepo(t, project)
	writeCandidateTestFile(t, project, "src/a.go", "package a\n")
	runDupGit(t, project, "add", "src")
	runDupGit(t, project, "commit", "-q", "-m", "chore: baseline")
	base = strings.TrimSpace(string(mustGitOutput(t, project, "rev-parse", "HEAD")))
	wt = filepath.Join(resolvedTempDir(t), "wt")
	runDupGit(t, project, "worktree", "add", "-q", "-b", "sibling", wt, base)
	return project, root, base, wt
}

func TestCandidateFreezeReadsALinkedWorktree(t *testing.T) {
	_, root, base, wt := linkedWorktreeRepo(t)
	workspace := filepath.Join(root, "work", "feature")
	writeFile(t, filepath.Join(workspace, "tasks.md"), "# Tasks\n\n## SLICE-001 A\nDependencies: none\nWriter allowlist: src/a.go; src/a_test.go\n")
	writeCandidateTestFile(t, wt, "src/a.go", "package a // sibling\n")
	runDupGit(t, wt, "commit", "-q", "-am", "feat: sibling")
	head := strings.TrimSpace(string(mustGitOutput(t, wt, "rev-parse", "HEAD")))

	if code, out := runFreezeT(t, root, "--wave", "s0", "--slice", "SLICE-001", "--base", base); code != 3 || !strings.Contains(out, "no change against the base") {
		t.Fatalf("the control checkout does not hold the sibling's change: code=%d\n%s", code, out)
	}
	if code, out := runFreezeT(t, root, "--wave", "s1", "--slice", "SLICE-001", "--worktree", wt); code != 2 {
		t.Fatalf("--worktree without --base is a usage error: code=%d\n%s", code, out)
	}
	if code, out := runFreezeT(t, root, "--wave", "s1", "--slice", "SLICE-001", "--base", base, "--worktree", wt); code != 0 {
		t.Fatalf("worktree freeze: code=%d\n%s", code, out)
	}
	sums := readTestFile(t, filepath.Join(workspace, "packets", "s1.sha256"))
	for _, want := range []string{
		"base " + base + "\nhead " + head + "\ndiff ",
		fmt.Sprintf("file %x src/a.go\n", sha256.Sum256([]byte("package a // sibling\n"))),
		"file absent src/a_test.go\n",
	} {
		if !strings.Contains(sums, want) {
			t.Fatalf("sha256 lacks %q:\n%s", want, sums)
		}
	}
	if diff := readTestFile(t, filepath.Join(workspace, "packets", "s1.diff")); !strings.Contains(diff, "+package a // sibling") {
		t.Fatalf("the frozen diff is not the worktree's change:\n%s", diff)
	}
	if code, out := runFreezeT(t, root, "--wave", "s2", "--paths", ".devrites/work/feature/state.md", "--base", base, "--worktree", wt); code != 3 || !strings.Contains(out, "workflow state") {
		t.Fatalf("workflow state is never a candidate path: code=%d\n%s", code, out)
	}
	writeCandidateTestFile(t, wt, "src/a_test.go", "package a\n")
	if code, out := runFreezeT(t, root, "--wave", "s2", "--slice", "SLICE-001", "--base", base, "--worktree", wt); code != 3 || !strings.Contains(out, "uncommitted changes") {
		t.Fatalf("a frozen worktree candidate must be its HEAD transfer commit: code=%d\n%s", code, out)
	}
}

func TestCheckedWorktreeRefusesAnythingButALinkedWorktree(t *testing.T) {
	project, root, _, wt := linkedWorktreeRepo(t)
	if got, err := checkedWorktree(root, wt); err != nil || got != wt {
		t.Fatalf("a linked worktree: %q, %v", got, err)
	}
	other := resolvedTempDir(t)
	initDupRepo(t, other)
	link := filepath.Join(resolvedTempDir(t), "link")
	if err := os.Symlink(wt, link); err != nil {
		t.Fatal(err)
	}
	control, err := filepath.EvalSymlinks(project)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ path, want string }{
		{resolvedTempDir(t), "is not a git worktree"},
		{filepath.Join(wt, "src"), "is not a worktree's top level"},
		{other, "belongs to another repository"},
		{control, "is the control checkout"},
		{link, "symlinked component"},
	} {
		if _, err := checkedWorktree(root, tc.path); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err=%v, want %q", tc.path, err, tc.want)
		}
	}
}

func mustGitOutput(t *testing.T, project string, args ...string) []byte {
	t.Helper()
	out, err := runGitCommand(project, nil, args...)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// packetTestWorkspace is a non-Git workspace with every role's bundle, a
// frozen candidate for wave r1, and the artifacts wright packets resolve.
func packetTestWorkspace(t *testing.T) (root, workspace, sums string) {
	t.Helper()
	_, root = writeCandidateTestWorkspace(t, candidateTestManifest())
	workspace = filepath.Join(root, "work", "feature")
	for _, bundle := range []string{"review-code-reviewer", "build-slice-wright.SLICE-001", "build-test-analyst.SLICE-001", "build-doubt-reviewer.SLICE-001", "build-code-reviewer.SLICE-001.recheck", "vet-plan-reviewer"} {
		writeFile(t, filepath.Join(workspace, "ctx", bundle+".bundle.md"), "bundle\n")
	}
	sums = "devrites-candidate-sha256 v1\nbase 0123456789abcdef0123456789abcdef01234567\ndiff 1111111111111111111111111111111111111111111111111111111111111111 packets/r1.diff\nfile 2222222222222222222222222222222222222222222222222222222222222222 src/a.go\naggregate 3333333333333333333333333333333333333333333333333333333333333333\n"
	writeFile(t, filepath.Join(workspace, "packets", "r1.sha256"), sums)
	writeFile(t, filepath.Join(workspace, "packets", "r1.diff"), "diff --git a/src/a.go b/src/a.go\n")
	writeFile(t, filepath.Join(workspace, "tasks.md"), "# Tasks\n\n## SLICE-001 A\nDependencies: none\nWriter allowlist: src/a.go; src/a_test.go\n")
	writeFile(t, filepath.Join(workspace, "gates.md"), "# Gates\n\n- [ ] G-UNIT: unit tests pass\n  CHECK: go test ./...\n  EXPECT: ok\n  CWD: engine\n- [ ] G-ROGUE: unvetted command\n  CHECK: curl example.invalid\n  EXPECT: ok\n")
	writeFile(t, filepath.Join(workspace, "test-plan.md"), "# Test plan\n\n## Build-entry preflight\n\n| Command | Cwd | Purpose |\n| --- | --- | --- |\n| go test ./... | engine | unit |\n")
	writeFile(t, filepath.Join(workspace, "focus.md"), "- F-1 src/a.go:12 nil customer panics\n\n")
	return root, workspace, sums
}

func TestPacketRendersGoldenReviewerPacket(t *testing.T) {
	root, workspace, sums := packetTestWorkspace(t)
	code, out := runPacketT(t, root, "--wave", "r1", "--phase", "review", "--role", "code-reviewer", "--focus", "focus.md")
	if code != 0 || !strings.Contains(out, "packet: .devrites/work/feature/packets/r1.code-reviewer.md") {
		t.Fatalf("code=%d\n%s", code, out)
	}
	want := "# Packet r1: code-reviewer\n\nFeature: feature\nPhase: review\n" +
		"Authority: read-only reviewer. Edit nothing and ask the user nothing; judge the candidate against the bundle sources.\n\n" +
		"Read first:\n- .devrites/work/feature/ctx/review-code-reviewer.bundle.md\n- .devrites/work/feature/packets/r1.diff\n\n" +
		"Candidate (verbatim from .devrites/work/feature/packets/r1.sha256):\n```text\n" + sums + "```\n\n" +
		"Open findings:\n- F-1 src/a.go:12 nil customer panics\n\n" +
		"Return (lines 1-2 exactly, then your role's rows):\nOutcome: <findings|no-findings|gap>\n" +
		"Counts: critical=<n> important=<n> suggestion=<n> nit=<n> fyi=<n> gap=<n>\n"
	if got := readTestFile(t, filepath.Join(workspace, "packets", "r1.code-reviewer.md")); got != want {
		t.Fatalf("golden mismatch:\n--- got\n%s\n--- want\n%s", got, want)
	}
}

func TestPacketRendersGoldenWrightContract(t *testing.T) {
	root, workspace, _ := packetTestWorkspace(t)
	code, out := runPacketT(t, root, "--wave", "SLICE-001", "--phase", "build", "--role", "slice-wright", "--slice", "SLICE-001", "--checks", "G-UNIT")
	if code != 0 {
		t.Fatalf("code=%d\n%s", code, out)
	}
	want := "# Packet SLICE-001: slice-wright\n\nFeature: feature\nPhase: build\nSlice: SLICE-001\nIsolation: same-worktree\n" +
		"Authority: sole writer for this slice. Edit only the Allowlist paths; never edit .devrites/**, never commit or push, and ask the user nothing.\n\n" +
		"Read first:\n- .devrites/work/feature/ctx/build-slice-wright.SLICE-001.bundle.md\n\n" +
		"Allowlist:\n- src/a.go\n- src/a_test.go\n\n" +
		"Writer-safe checks: G-UNIT\n- G-UNIT (cwd engine): `go test ./...`\n\n" +
		"Return: every field of your role's return block, with each Writer-safe check's command and exit status.\n"
	if got := readTestFile(t, filepath.Join(workspace, "packets", "SLICE-001.contract.md")); got != want {
		t.Fatalf("golden mismatch:\n--- got\n%s\n--- want\n%s", got, want)
	}
}

func TestPacketRolesPassDispatchLints(t *testing.T) {
	root, workspace, _ := packetTestWorkspace(t)
	cases := []struct {
		args   []string
		file   string
		counts string
	}{
		{[]string{"--wave", "r1", "--phase", "review", "--role", "code-reviewer"}, "r1.code-reviewer.md", "Counts: critical=<n> important=<n> suggestion=<n> nit=<n> fyi=<n> gap=<n>"},
		{[]string{"--wave", "b1", "--phase", "build", "--role", "test-analyst", "--slice", "SLICE-001"}, "b1.test-analyst.md", "Counts: critical=<n> important=<n> suggestion=<n> nit=<n> fyi=<n> gap=<n>"},
		{[]string{"--wave", "b1", "--phase", "build", "--role", "doubt-reviewer", "--slice", "SLICE-001", "--focus", "focus.md"}, "b1.doubt-reviewer.md", "Counts: critical=<n> important=<n> suggestion=<n> nit=<n> fyi=<n> gap=<n>"},
		{[]string{"--wave", "v1", "--phase", "vet", "--role", "plan-reviewer"}, "v1.plan-reviewer.md", "Counts: critical=<n> important=<n> suggestion=<n> nit=<n> fyi=<n> gap=<n> contract=<n> mechanism=<n>"},
		{[]string{"--wave", "SLICE-001", "--phase", "build", "--role", "slice-wright", "--slice", "SLICE-001", "--checks", "G-UNIT"}, "SLICE-001.contract.md", ""},
	}
	for _, tc := range cases {
		if code, out := runPacketT(t, root, tc.args...); code != 0 {
			t.Fatalf("%v: code=%d\n%s", tc.args, code, out)
		}
		path := filepath.Join(workspace, "packets", tc.file)
		text := readTestFile(t, path)
		if tc.counts != "" && !strings.Contains(text, "\n"+tc.counts+"\n") {
			t.Fatalf("%s lacks %q:\n%s", tc.file, tc.counts, text)
		}
		if strings.Contains(text, "agents/devrites-") {
			t.Fatalf("%s names an installed role contract:\n%s", tc.file, text)
		}
		if strings.Contains(text, "packets/b1.diff") || strings.Contains(text, "packets/v1.diff") {
			t.Fatalf("%s names a diff no freeze wrote:\n%s", tc.file, text)
		}
		var lintErr error
		if tc.counts == "" {
			lintErr = lintWrightContract(path)
		} else if lintErr = lintReviewerPacket(path); lintErr == nil && strings.Contains(tc.file, "r1.") {
			lintErr = lintReviewCandidate(root, "feature", path)
		}
		if lintErr != nil {
			t.Fatalf("%s fails a dispatch open lint: %v", tc.file, lintErr)
		}
	}
}

func TestPacketRefusesUnsafeInputs(t *testing.T) {
	root, workspace, _ := packetTestWorkspace(t)
	writeFile(t, filepath.Join(workspace, "narrative.md"), "- F-1 src/a.go:12 nil customer panics\nParent observed three completed passes and the join makes all3green.\n")
	writeFile(t, filepath.Join(workspace, "seeded.md"), "- F-2 src/a.go:3 the smoke test already passed on this candidate\n")
	writeFile(t, filepath.Join(workspace, "long.md"), strings.Repeat("- DEC-001 decisions.md:5 the boundary holds\n", 4))
	cases := []struct {
		name string
		args []string
		code int
		want string
		file string // must not exist after a refusal
	}{
		{"narrative focus", []string{"--wave", "r1", "--phase", "review", "--role", "code-reviewer", "--focus", "narrative.md"}, 3, "narrative.md:2 is not an open-finding row", "r1.code-reviewer.md"},
		{"seeded focus row", []string{"--wave", "r1", "--phase", "review", "--role", "code-reviewer", "--focus", "seeded.md"}, 3, "seeds a conclusion", "r1.code-reviewer.md"},
		{"unknown gate", []string{"--wave", "SLICE-001", "--phase", "build", "--role", "slice-wright", "--slice", "SLICE-001", "--checks", "G-UNIT,G-NOPE"}, 3, `gate "G-NOPE" is not in gates.md`, "SLICE-001.contract.md"},
		{"unapproved gate", []string{"--wave", "SLICE-001", "--phase", "build", "--role", "slice-wright", "--slice", "SLICE-001", "--checks", "G-ROGUE"}, 3, "not an approved test-plan.md Build-entry preflight row", "SLICE-001.contract.md"},
		{"wright without checks", []string{"--wave", "SLICE-001", "--phase", "build", "--role", "slice-wright", "--slice", "SLICE-001"}, 3, "names no Tests/proof in tasks.md and --checks names no gate", "SLICE-001.contract.md"},
		{"review without freeze", []string{"--wave", "r9", "--phase", "review", "--role", "code-reviewer"}, 3, "needs a frozen candidate", "r9.code-reviewer.md"},
		{"missing bundle", []string{"--wave", "r1", "--phase", "seal", "--role", "code-reviewer"}, 3, "seal-code-reviewer.bundle.md is missing", "r1.code-reviewer.md"},
		{"non-reviewer role", []string{"--wave", "r1", "--phase", "plan", "--role", "plan-drafter"}, 2, "usage:", ""},
		{"reviewer with checks", []string{"--wave", "r1", "--phase", "review", "--role", "code-reviewer", "--checks", "G-UNIT"}, 2, "usage:", ""},
		{"wright without slice", []string{"--wave", "w", "--phase", "build", "--role", "slice-wright", "--checks", "G-UNIT"}, 2, "usage:", ""},
		{"doubt without claim", []string{"--wave", "b1", "--phase", "build", "--role", "doubt-reviewer", "--slice", "SLICE-001"}, 2, "usage:", ""},
		{"doubt claim over 3 rows", []string{"--wave", "b1", "--phase", "build", "--role", "doubt-reviewer", "--slice", "SLICE-001", "--focus", "long.md"}, 3, "1-3 row claim", "b1.doubt-reviewer.md"},
		{"recheck outside build", []string{"--wave", "r1", "--phase", "review", "--role", "code-reviewer", "--recheck"}, 2, "usage:", ""},
		{"missing slice bundle", []string{"--wave", "b1", "--phase", "build", "--role", "security-auditor", "--slice", "SLICE-001", "--recheck"}, 3, "--role security-auditor --slice SLICE-001 --recheck <inventory> first", "b1.security-auditor.md"},
		{"traversal wave", []string{"--wave", "../r1", "--phase", "review", "--role", "code-reviewer"}, 2, "usage:", ""},
	}
	for _, tc := range cases {
		code, out := runPacketT(t, root, tc.args...)
		if code != tc.code || !strings.Contains(out, tc.want) {
			t.Fatalf("%s: want exit %d with %q, got %d\n%s", tc.name, tc.code, tc.want, code, out)
		}
		if tc.file != "" {
			if _, err := os.Stat(filepath.Join(workspace, "packets", tc.file)); !os.IsNotExist(err) {
				t.Fatalf("%s: refused packet %s was left on disk (%v)", tc.name, tc.file, err)
			}
		}
	}
}

func TestPacketNamesRecheckBundleAndDoubtClaim(t *testing.T) {
	root, workspace, _ := packetTestWorkspace(t)
	if code, out := runPacketT(t, root, "--wave", "b2", "--phase", "build", "--role", "code-reviewer", "--slice", "SLICE-001", "--recheck"); code != 0 {
		t.Fatalf("recheck: code=%d\n%s", code, out)
	}
	if text := readTestFile(t, filepath.Join(workspace, "packets", "b2.code-reviewer.md")); !strings.Contains(text, "- .devrites/work/feature/ctx/build-code-reviewer.SLICE-001.recheck.bundle.md\n") {
		t.Fatalf("a recheck packet must name the recheck bundle:\n%s", text)
	}
	writeFile(t, filepath.Join(workspace, "claim.md"), "- DEC-001 decisions.md:5 the store owns the lock, so callers never retry\n")
	if code, out := runPacketT(t, root, "--wave", "b2", "--phase", "build", "--role", "doubt-reviewer", "--slice", "SLICE-001", "--focus", "claim.md"); code != 0 {
		t.Fatalf("doubt: code=%d\n%s", code, out)
	}
	if text := readTestFile(t, filepath.Join(workspace, "packets", "b2.doubt-reviewer.md")); !strings.Contains(text, "\nClaim to attack:\n- DEC-001 decisions.md:5 the store owns the lock, so callers never retry\n") {
		t.Fatalf("a doubt packet must carry its claim:\n%s", text)
	}
}

func TestPacketRefusedRerenderKeepsPriorPacket(t *testing.T) {
	root, workspace, _ := packetTestWorkspace(t)
	if code, out := runPacketT(t, root, "--wave", "r1", "--phase", "review", "--role", "code-reviewer", "--focus", "focus.md"); code != 0 {
		t.Fatalf("code=%d\n%s", code, out)
	}
	path := filepath.Join(workspace, "packets", "r1.code-reviewer.md")
	prior := readTestFile(t, path)
	writeFile(t, filepath.Join(workspace, "seeded.md"), "- F-2 src/a.go:3 the smoke test already passed on this candidate\n")
	if code, out := runPacketT(t, root, "--wave", "r1", "--phase", "review", "--role", "code-reviewer", "--focus", "seeded.md"); code != 3 || !strings.Contains(out, "seeds a conclusion") {
		t.Fatalf("seeded re-render: code=%d\n%s", code, out)
	}
	if readTestFile(t, path) != prior {
		t.Fatal("a refused re-render replaced the packet already on disk")
	}
	if _, err := os.Stat(filepath.Join(workspace, "packets", ".r1.code-reviewer.md.draft")); !os.IsNotExist(err) {
		t.Fatalf("a refused draft was left on disk (%v)", err)
	}
}

func TestPacketKeepsHeaderRowsOfLargeCandidate(t *testing.T) {
	root, workspace, _ := packetTestWorkspace(t)
	var sums strings.Builder
	sums.WriteString("devrites-candidate-sha256 v1\nbase 0123456789abcdef0123456789abcdef01234567\ndiff 1111111111111111111111111111111111111111111111111111111111111111 packets/r3.diff\n")
	for i := 0; i < 600; i++ {
		fmt.Fprintf(&sums, "file 2222222222222222222222222222222222222222222222222222222222222222 src/pkg/file%04d.go\n", i)
	}
	sums.WriteString("aggregate 3333333333333333333333333333333333333333333333333333333333333333\n")
	writeFile(t, filepath.Join(workspace, "packets", "r3.sha256"), sums.String())
	writeFile(t, filepath.Join(workspace, "packets", "r3.diff"), "diff --git a/src/a.go b/src/a.go\n")
	if code, out := runPacketT(t, root, "--wave", "r3", "--phase", "review", "--role", "code-reviewer"); code != 0 {
		t.Fatalf("large candidate: code=%d\n%s", code, out)
	}
	text := readTestFile(t, filepath.Join(workspace, "packets", "r3.code-reviewer.md"))
	if strings.Contains(text, "file0000.go") || !strings.Contains(text, "header rows verbatim; per-file rows in .devrites/work/feature/packets/r3.sha256") ||
		!strings.Contains(text, "packets/r3.diff\naggregate 3333333333333333333333333333333333333333333333333333333333333333\n```") {
		t.Fatalf("a large candidate keeps only its header rows:\n%s", text)
	}
}

func TestPacketWrightListsSliceTestsProof(t *testing.T) {
	root, workspace, _ := packetTestWorkspace(t)
	writeFile(t, filepath.Join(workspace, "tasks.md"), "# Tasks\n\n## SLICE-001 A\nDependencies: none\nWriter allowlist: src/a.go; src/a_test.go\nTests/proof: `go test ./src/...` ok incl. TestA\n")
	contract := filepath.Join(workspace, "packets", "SLICE-001.contract.md")
	if code, out := runPacketT(t, root, "--wave", "SLICE-001", "--phase", "build", "--role", "slice-wright", "--slice", "SLICE-001"); code != 0 {
		t.Fatalf("a slice's own Tests/proof needs no gate: code=%d\n%s", code, out)
	}
	if text := readTestFile(t, contract); !strings.Contains(text, "Writer-safe checks: SLICE-001 Tests/proof\n- SLICE-001 Tests/proof (tasks.md): `go test ./src/...` ok incl. TestA\n\n") {
		t.Fatalf("the contract lacks the slice's Tests/proof:\n%s", text)
	}
	if code, out := runPacketT(t, root, "--wave", "SLICE-001", "--phase", "build", "--role", "slice-wright", "--slice", "SLICE-001", "--checks", "G-UNIT"); code != 0 {
		t.Fatalf("code=%d\n%s", code, out)
	}
	if text := readTestFile(t, contract); !strings.Contains(text, "Writer-safe checks: SLICE-001 Tests/proof, G-UNIT\n- SLICE-001 Tests/proof (tasks.md): `go test ./src/...` ok incl. TestA\n- G-UNIT (cwd engine): `go test ./...`\n\n") {
		t.Fatalf("--checks gates follow the slice's Tests/proof:\n%s", text)
	}
}

func TestPacketWrightNamesGatesPrecheckForWriterGates(t *testing.T) {
	root, workspace, _ := packetTestWorkspace(t)
	writeFile(t, filepath.Join(workspace, "gates.md"), "# Gates\n\n- [ ] G-UNIT: unit tests pass\n  CHECK: go test ./...\n  EXPECT: ok\n  CWD: engine\n  RUN: writer\n")
	if code, out := runPacketT(t, root, "--wave", "SLICE-001", "--phase", "build", "--role", "slice-wright", "--slice", "SLICE-001", "--checks", "G-UNIT"); code != 0 {
		t.Fatalf("code=%d\n%s", code, out)
	}
	text := readTestFile(t, filepath.Join(workspace, "packets", "SLICE-001.contract.md"))
	if !strings.Contains(text, "- G-UNIT (cwd engine): `go test ./...`\n- gates precheck (cwd repository root): `devrites-engine gates precheck feature --worktree . --slice SLICE-001`\n\n") {
		t.Fatalf("a RUN: writer ledger needs a slice-scoped gates precheck in the Writer-safe checks:\n%s", text)
	}
}
