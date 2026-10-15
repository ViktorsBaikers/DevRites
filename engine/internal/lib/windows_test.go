package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

// windowsStandardPath is the canonical pack standard for this check, relative
// to engine/internal/lib.
const windowsStandardPath = "../../../pack/.claude/skills/devrites-lib/reference/standards/windows.md"

var (
	// The untracked-file guarantee must name the mode that implements it: no git
	// diff shows an untracked path, so only the whole-file scan in windowsHits
	// covers one.
	windowsWorktreeScopeRe = regexp.MustCompile(`(?i)untracked files (?:as|a) wholly new (?:in|under|by) (?:` + "`" + `--)?worktree(?: mode)?`)
	// The other two modes must state the narrower scope they actually have.
	windowsOtherModeLimitRe = regexp.MustCompile(`(?i)--staged[^\n]{0,60}--base[^\n]{0,40}(?:diff only|tracked diff)`)
	// The defective wording: the guarantee stated with no mode qualifier, in the
	// same sentence that offers worktree, staged and base as one set.
	windowsUnscopedClaim = "treats untracked files as wholly new,"
)

func windowsWorkspace(t *testing.T, project, slug string) string {
	t.Helper()
	t.Setenv("DEVRITES_WORKSPACE", "")
	root := filepath.Join(project, ".devrites")
	ws := filepath.Join(root, "work", slug)
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func initWindowsRepo(t *testing.T, project string) {
	t.Helper()
	runSecretScanGit(t, project, "init", "-q")
	runSecretScanGit(t, project, "config", "user.email", "tests@example.invalid")
	runSecretScanGit(t, project, "config", "user.name", "DevRites Tests")
}

func TestWindowsMarkersTokensAndIdioms(t *testing.T) {
	cases := []struct {
		line string
		want []string
	}{
		{"// TODO: backfill index", []string{"TODO"}},
		{"x := 1 //FIXME", []string{"FIXME"}},
		{"ATODOX is not a marker", nil},
		{"t.Skip(\"flaky\")", []string{"t.Skip("}},
		{"describe.skip(\"x\", fn)", []string{"describe.skip("}},
		{"raise NotImplementedError", []string{"NotImplementedError"}},
		{"panic(\"TODO wire cache\")", []string{"panic(\"TODO"}},
		{"// @ts-ignore", []string{"@ts-ignore"}},
		{"x = f()  # noqa: E501", []string{"noqa"}},
		{"it.only(\"focus\", fn)", []string{"it.only("}},
		{"try { f() } catch (e) {}", []string{"empty-catch"}},
		{"except ValueError: pass", []string{"empty-catch"}},
		{"} catch (e) { log(e) }", nil},
		{"plain line", nil},
	}
	for _, tc := range cases {
		got := windowsMarkers(tc.line)
		if len(got) != len(tc.want) {
			t.Fatalf("%q: markers=%v want %v", tc.line, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("%q: marker[%d]=%q want %q", tc.line, i, got[i], tc.want[i])
			}
		}
	}
}

func TestAddedLinesFromDiffTracksPathsWithoutHits(t *testing.T) {
	diff := "diff --git a/a.go b/a.go\nindex 1..2 100644\n--- a/a.go\n+++ b/a.go\n@@ -1 +1,2 @@\n+fmt.Println(\"hi\")\ndiff --git a/b.go b/b.go\nindex 3..4 100644\n--- a/b.go\n+++ b/b.go\n@@ -5 +6 @@\n+\t// TODO later\n"
	hits, paths := addedLinesFromDiff(diff)
	if len(hits) != 1 || hits[0].Path != "b.go" || hits[0].Line != 6 || hits[0].Marker != "TODO" {
		t.Fatalf("hits=%v", hits)
	}
	if !paths["a.go"] || !paths["b.go"] {
		t.Fatalf("diffPaths=%v", paths)
	}
}

func TestAddedLinesFromDiffSkipsDeletedAndBinary(t *testing.T) {
	diff := "diff --git a/gone.go b/gone.go\nindex 1..2 100644\n--- a/gone.go\n+++ /dev/null\n@@ -1 +0,0 @@\n-// TODO removed\ndiff --git a/bin b/bin\nindex 1..2 100644\nBinary files a/bin and b/bin differ\n"
	hits, paths := addedLinesFromDiff(diff)
	if len(hits) != 0 {
		t.Fatalf("hits=%v", hits)
	}
	if paths["gone.go"] {
		t.Fatal("deleted path recorded as new-side path")
	}
}

func TestCheckWindowsBlocksUnwaivedWorktreeMarker(t *testing.T) {
	project := t.TempDir()
	initWindowsRepo(t, project)
	root := windowsWorkspace(t, project, "feat")
	testutil.WriteFile(t, filepath.Join(project, "a.go"), "package a\n\nfunc A() {}\n")
	runSecretScanGit(t, project, "add", "--", "a.go")
	runSecretScanGit(t, project, "commit", "-qm", "baseline")
	testutil.AppendFile(t, filepath.Join(project, "a.go"), "\n// TODO: wire retry\n")

	var out, errBuf bytes.Buffer
	code := RunCheckWindows(root, []string{"feat"}, &out, &errBuf)
	if code != 3 {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errBuf.String())
	}
	if !strings.Contains(out.String(), "unwaived: a.go:5 TODO") {
		t.Fatalf("out=%q", out.String())
	}
}

func TestCheckWindowsWaivedRowPasses(t *testing.T) {
	project := t.TempDir()
	initWindowsRepo(t, project)
	root := windowsWorkspace(t, project, "feat")
	testutil.WriteFile(t, filepath.Join(project, "a.go"), "package a\n")
	runSecretScanGit(t, project, "add", "--", "a.go")
	runSecretScanGit(t, project, "commit", "-qm", "baseline")
	testutil.AppendFile(t, filepath.Join(project, "a.go"), "// HACK: hotfix until v2\n")
	testutil.WriteFile(t, filepath.Join(root, "work", "feat", "windows.md"),
		"# Deferred\n\n- a.go | HACK | follow-up tracked in next slice\n")

	var out, errBuf bytes.Buffer
	if code := RunCheckWindows(root, []string{"feat"}, &out, &errBuf); code != 0 {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errBuf.String())
	}
	if !strings.Contains(out.String(), "all waived") {
		t.Fatalf("out=%q", out.String())
	}
}

func TestCheckWindowsIgnoresPreexistingMarkers(t *testing.T) {
	project := t.TempDir()
	initWindowsRepo(t, project)
	root := windowsWorkspace(t, project, "feat")
	testutil.WriteFile(t, filepath.Join(project, "a.go"), "package a\n\n// TODO: pre-existing debt\n")
	runSecretScanGit(t, project, "add", "--", "a.go")
	runSecretScanGit(t, project, "commit", "-qm", "baseline")
	testutil.AppendFile(t, filepath.Join(project, "a.go"), "\nfunc Clean() {}\n")

	var out, errBuf bytes.Buffer
	if code := RunCheckWindows(root, []string{"feat"}, &out, &errBuf); code != 0 {
		t.Fatalf("pre-existing marker flagged: code=%d out=%q", code, out.String())
	}
}

func TestCheckWindowsScansUntrackedAndSkipsDevrites(t *testing.T) {
	project := t.TempDir()
	initWindowsRepo(t, project)
	root := windowsWorkspace(t, project, "feat")
	testutil.WriteFile(t, filepath.Join(project, "b.go"), "package b\n\n// FIXME: fresh file\n")
	testutil.WriteFile(t, filepath.Join(project, ".devrites", "scratch.md"), "TODO: not product code\n")

	var out, errBuf bytes.Buffer
	code := RunCheckWindows(root, []string{"feat"}, &out, &errBuf)
	if code != 3 {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
	if !strings.Contains(out.String(), "unwaived: b.go:3 FIXME") {
		t.Fatalf("out=%q", out.String())
	}
	if strings.Contains(out.String(), "scratch.md") {
		t.Fatalf(".devrites path flagged: %q", out.String())
	}
}

func TestCheckWindowsStagedMode(t *testing.T) {
	project := t.TempDir()
	initWindowsRepo(t, project)
	root := windowsWorkspace(t, project, "feat")
	testutil.WriteFile(t, filepath.Join(project, "a.go"), "package a\n")
	runSecretScanGit(t, project, "add", "--", "a.go")
	runSecretScanGit(t, project, "commit", "-qm", "baseline")
	testutil.AppendFile(t, filepath.Join(project, "a.go"), "// XXX: staged stub\n")
	runSecretScanGit(t, project, "add", "--", "a.go")

	var out, errBuf bytes.Buffer
	if code := RunCheckWindows(root, []string{"feat", "--staged"}, &out, &errBuf); code != 3 {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
}

func TestCheckWindowsUnbornHEADScansAll(t *testing.T) {
	project := t.TempDir()
	initWindowsRepo(t, project)
	root := windowsWorkspace(t, project, "feat")
	testutil.WriteFile(t, filepath.Join(project, "a.go"), "package a\n\n// TODO: first commit pending\n")

	var out, errBuf bytes.Buffer
	if code := RunCheckWindows(root, []string{"feat"}, &out, &errBuf); code != 3 {
		t.Fatalf("unborn HEAD: code=%d out=%q err=%q", code, out.String(), errBuf.String())
	}
}

func TestCheckWindowsMalformedWaiverWarnsAndBlocks(t *testing.T) {
	project := t.TempDir()
	initWindowsRepo(t, project)
	root := windowsWorkspace(t, project, "feat")
	testutil.WriteFile(t, filepath.Join(project, "a.go"), "package a\n")
	runSecretScanGit(t, project, "add", "--", "a.go")
	runSecretScanGit(t, project, "commit", "-qm", "baseline")
	testutil.AppendFile(t, filepath.Join(project, "a.go"), "// TODO: needs waiver\n")
	testutil.WriteFile(t, filepath.Join(root, "work", "feat", "windows.md"), "- a.go | TODO |\n")

	var out, errBuf bytes.Buffer
	code := RunCheckWindows(root, []string{"feat"}, &out, &errBuf)
	if code != 3 {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
	if !strings.Contains(out.String(), "no reason") {
		t.Fatalf("missing empty-reason warning: %q", out.String())
	}
}

func TestCheckWindowsBaseModeScopesToFeatureDiff(t *testing.T) {
	project := t.TempDir()
	initWindowsRepo(t, project)
	root := windowsWorkspace(t, project, "feat")
	testutil.WriteFile(t, filepath.Join(project, "a.go"), "package a\n")
	runSecretScanGit(t, project, "add", "--", "a.go")
	runSecretScanGit(t, project, "commit", "-qm", "base")
	runSecretScanGit(t, project, "rev-parse", "HEAD")
	testutil.WriteFile(t, filepath.Join(project, "feat.go"), "package feat\n\n// TODO: slice 2\n")
	runSecretScanGit(t, project, "add", "--", "feat.go")
	runSecretScanGit(t, project, "commit", "-qm", "feature commit")
	// `git diff <base>` is base→worktree: an uncommitted marker still flags.
	testutil.AppendFile(t, filepath.Join(project, "a.go"), "// FIXME: uncommitted follow-up\n")

	var out, errBuf bytes.Buffer
	code := RunCheckWindows(root, []string{"feat", "--base", "HEAD~1"}, &out, &errBuf)
	if code != 3 {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errBuf.String())
	}
	if !strings.Contains(out.String(), "unwaived: feat.go:3 TODO") ||
		!strings.Contains(out.String(), "unwaived: a.go:2 FIXME") {
		t.Fatalf("out=%q", out.String())
	}
}

func TestCheckWindowsUsageErrors(t *testing.T) {
	project := t.TempDir()
	root := windowsWorkspace(t, project, "feat")
	var out, errBuf bytes.Buffer
	if code := RunCheckWindows(root, nil, &out, &errBuf); code != 2 {
		t.Fatalf("no slug: code=%d", code)
	}
	errBuf.Reset()
	if code := RunCheckWindows(root, []string{"feat", "--base"}, &out, &errBuf); code != 2 {
		t.Fatalf("missing base ref: code=%d", code)
	}
}

// TestCheckWindowsUntrackedScopePerMode pins which modes scan an untracked file.
// The whole-file scan is inside the worktree branch because neither `git diff
// HEAD` nor `git diff --cached` nor `git diff <ref>` reports an untracked path;
// staged and base modes read the index and the tracked diff only. This is the
// behaviour half of the drift guard in
// TestWindowsStandardScopesUntrackedFiles.
func TestCheckWindowsUntrackedScopePerMode(t *testing.T) {
	project := t.TempDir()
	initWindowsRepo(t, project)
	root := windowsWorkspace(t, project, "feat")
	testutil.WriteFile(t, filepath.Join(project, "base.go"), "package base\n")
	runSecretScanGit(t, project, "add", "--", "base.go")
	runSecretScanGit(t, project, "commit", "-qm", "baseline")
	// Never staged: only the whole-file scan can see this marker.
	testutil.WriteFile(t, filepath.Join(project, "new.go"), "package n\n\n// TODO: untracked\n")

	var out, errBuf bytes.Buffer
	if code := RunCheckWindows(root, []string{"feat"}, &out, &errBuf); code != 3 {
		t.Fatalf("--worktree must scan untracked files: code=%d out=%q err=%q", code, out.String(), errBuf.String())
	}
	if !strings.Contains(out.String(), "unwaived: new.go:3 TODO") {
		t.Fatalf("--worktree must report the untracked marker: out=%q", out.String())
	}
	for _, args := range [][]string{{"feat", "--staged"}, {"feat", "--base", "HEAD"}} {
		out.Reset()
		errBuf.Reset()
		if code := RunCheckWindows(root, args, &out, &errBuf); code != 0 {
			t.Fatalf("%v covers the tracked diff only, so an untracked file is out of scope: code=%d out=%q", args, code, out.String())
		}
	}
}

// TestWindowsStandardScopesUntrackedFiles keeps windows.md honest about the
// scope each diff mode has. The standard states the untracked-file guarantee
// next to the list of modes, so an unqualified claim reads as covering all
// three; --staged and --base then pass on a workspace holding a not-yet-staged
// marker. The assertions pair with TestCheckWindowsUntrackedScopePerMode:
// changing either the engine's per-mode scope or the wording requires changing
// both.
func TestWindowsStandardScopesUntrackedFiles(t *testing.T) {
	raw, err := os.ReadFile(windowsStandardPath)
	if err != nil {
		t.Fatalf("read canonical standard: %v", err)
	}
	prose := strings.Join(strings.Fields(string(raw)), " ")
	if !windowsWorktreeScopeRe.MatchString(prose) {
		t.Errorf("%s must scope the untracked-file guarantee to --worktree mode: no unqualified or wrong-mode claim found", windowsStandardPath)
	}
	if !windowsOtherModeLimitRe.MatchString(prose) {
		t.Errorf("%s must state that --staged and --base cover the tracked diff only", windowsStandardPath)
	}
	if strings.Contains(prose, windowsUnscopedClaim) {
		t.Errorf("%s still states the untracked-file guarantee with no mode qualifier (%q); --staged and --base do not scan untracked files", windowsStandardPath, windowsUnscopedClaim)
	}
}

// TestCheckWindowsFailsWhenStatusScanFails pins that a failed untracked-file
// listing is an error, never a clean result: the whole-file scan is the only
// reader of untracked files, so skipping it silently leaves them unexamined.
func TestCheckWindowsFailsWhenStatusScanFails(t *testing.T) {
	project := t.TempDir()
	initWindowsRepo(t, project)
	root := windowsWorkspace(t, project, "feat")
	testutil.WriteFile(t, filepath.Join(project, "a.go"), "package a\n")
	runSecretScanGit(t, project, "add", "--", "a.go")
	runSecretScanGit(t, project, "commit", "-qm", "baseline")
	testutil.WriteFile(t, filepath.Join(project, "new.go"), "package n\n\n// TODO: untracked\n")
	// git status rejects this value; git diff never reads it.
	runSecretScanGit(t, project, "config", "status.renames", "bogus")

	var out, errBuf bytes.Buffer
	code := RunCheckWindows(root, []string{"feat"}, &out, &errBuf)
	if code == 0 || strings.Contains(out.String(), "windows: ok") {
		t.Fatalf("status failure must not report clean: code=%d out=%q err=%q", code, out.String(), errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "windows:") {
		t.Fatalf("status failure must be reported on stderr: err=%q", errBuf.String())
	}
}

func TestRemovedTestHits(t *testing.T) {
	diff := "diff --git a/old_test.go b/old_test.go\ndeleted file mode 100644\n--- a/old_test.go\n+++ /dev/null\n@@ -1,2 +0,0 @@\n-package a\n-\tassert(true)\n" +
		"diff --git a/a_test.go b/a_test.go\n--- a/a_test.go\n+++ b/a_test.go\n@@ -3,2 +3,0 @@\n-\tif got != 1 { t.Fatalf(\"got %d\", got) }\n-\trequire.NoError(t, err)\n" +
		"diff --git a/moved.test.ts b/moved.test.ts\n--- a/moved.test.ts\n+++ b/moved.test.ts\n@@ -2 +2,0 @@\n-  expect(x).toBe(1)\n@@ -9,0 +9 @@\n+  expect(x).toBe(1)\n" +
		"diff --git a/lib.go b/lib.go\n--- a/lib.go\n+++ b/lib.go\n@@ -1 +0,0 @@\n-\tassert(ok)\n" +
		"diff --git a/gone.go b/gone.go\ndeleted file mode 100644\n--- a/gone.go\n+++ /dev/null\n@@ -1 +0,0 @@\n-package a\n"
	hits := removedTestHits(diff)
	if len(hits) != 2 {
		t.Fatalf("hits=%+v", hits)
	}
	if hits[0].Path != "old_test.go" || hits[0].Marker != "deleted-test" {
		t.Fatalf("deleted test hit=%+v", hits[0])
	}
	if hits[1].Path != "a_test.go" || hits[1].Marker != "removed-assertion" || !strings.Contains(hits[1].Text, "net -2") {
		t.Fatalf("removed assertion hit=%+v", hits[1])
	}
}

func TestIsTestPath(t *testing.T) {
	for p, want := range map[string]bool{
		"pkg/a_test.go": true, "src/a.test.ts": true, "src/a.spec.js": true,
		"tests/test_api.py": true, "spec/user_spec.rb": true, "src/__tests__/a.js": true,
		"FooTest.java": true, "tests/integration.rs": true,
		"pkg/a.go": false, "src/latest.ts": false, "tests/fixture.json": false, "contest.py": false,
	} {
		if got := isTestPath(p); got != want {
			t.Fatalf("isTestPath(%q)=%v want %v", p, got, want)
		}
	}
}

func TestCheckWindowsBlocksLoweredBarUntilWaived(t *testing.T) {
	project := t.TempDir()
	initWindowsRepo(t, project)
	root := windowsWorkspace(t, project, "feat")
	testutil.WriteFile(t, filepath.Join(project, "a_test.go"), "package a\n\nfunc TestA(t *testing.T) {\n\tif A() != 1 {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n")
	testutil.WriteFile(t, filepath.Join(project, "b_test.go"), "package a\n")
	runSecretScanGit(t, project, "add", "--", "a_test.go", "b_test.go")
	runSecretScanGit(t, project, "commit", "-qm", "baseline")
	testutil.WriteFile(t, filepath.Join(project, "a_test.go"), "package a\n\nfunc TestA(t *testing.T) {\n\t_ = A()\n}\n")
	if err := os.Remove(filepath.Join(project, "b_test.go")); err != nil {
		t.Fatal(err)
	}

	var out, errBuf bytes.Buffer
	if code := RunCheckWindows(root, []string{"feat"}, &out, &errBuf); code != 3 {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errBuf.String())
	}
	for _, want := range []string{"unwaived: a_test.go removed-assertion", "unwaived: b_test.go deleted-test"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q in out=%q", want, out.String())
		}
	}

	testutil.WriteFile(t, filepath.Join(root, "work", "feat", "windows.md"),
		"- a_test.go | removed-assertion | assertion moved to TestB in SLICE-002\n- b_test.go | deleted-test | feature removed per spec AC-3\n")
	out.Reset()
	if code := RunCheckWindows(root, []string{"feat"}, &out, &errBuf); code != 0 {
		t.Fatalf("waived: code=%d out=%q", code, out.String())
	}
}

func TestCheckWindowsRenamedTestIsNotDeleted(t *testing.T) {
	project := t.TempDir()
	initWindowsRepo(t, project)
	root := windowsWorkspace(t, project, "feat")
	runSecretScanGit(t, project, "config", "diff.renames", "false")
	body := "package a\n\nfunc TestA(t *testing.T) {\n\tif A() != 1 {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n"
	testutil.WriteFile(t, filepath.Join(project, "a_test.go"), body)
	runSecretScanGit(t, project, "add", "--", "a_test.go")
	runSecretScanGit(t, project, "commit", "-qm", "baseline")
	runSecretScanGit(t, project, "mv", "a_test.go", "b_test.go")

	var out, errBuf bytes.Buffer
	if code := RunCheckWindows(root, []string{"feat"}, &out, &errBuf); code != 0 {
		t.Fatalf("rename flagged: code=%d out=%q err=%q", code, out.String(), errBuf.String())
	}
}

// A --base value that looks like a flag must never reach git as an option.
func TestCheckWindowsAndDiffScopeRejectFlagLikeBase(t *testing.T) {
	project := t.TempDir()
	initWindowsRepo(t, project)
	root := windowsWorkspace(t, project, "feat")
	testutil.WriteFile(t, filepath.Join(project, "a.go"), "package a\n")
	runSecretScanGit(t, project, "add", "--", "a.go")
	runSecretScanGit(t, project, "commit", "-qm", "baseline")
	leak := filepath.Join(t.TempDir(), "leak.txt")
	var out, errBuf bytes.Buffer
	if code := RunCheckWindows(root, []string{"feat", "--base", "--output=" + leak}, &out, &errBuf); code != 2 {
		t.Fatalf("windows: code=%d out=%q err=%q", code, out.String(), errBuf.String())
	}
	if code := RunCheckDiffScope(root, []string{"feat", "--allow", "a.go", "--base", "--output=" + leak}, &out, &errBuf); code != 2 {
		t.Fatalf("diff-scope: code=%d out=%q err=%q", code, out.String(), errBuf.String())
	}
	if _, err := os.Stat(leak); err == nil {
		t.Fatal("--base value was passed to git as an option")
	}
}

// Repository diff configuration (color, external diff, mnemonic prefixes) and
// quoted non-ASCII paths must not hide, misattribute or duplicate markers.
func TestCheckWindowsIgnoresDiffConfigAndUnquotesPaths(t *testing.T) {
	project := t.TempDir()
	initWindowsRepo(t, project)
	root := windowsWorkspace(t, project, "feat")
	testutil.WriteFile(t, filepath.Join(project, "a.go"), "package a\n")
	testutil.WriteFile(t, filepath.Join(project, "café.go"), "package a\n")
	runSecretScanGit(t, project, "add", "--", "a.go", "café.go")
	runSecretScanGit(t, project, "commit", "-qm", "baseline")
	for _, kv := range [][2]string{{"color.ui", "always"}, {"diff.external", "true"}, {"diff.mnemonicPrefix", "true"}} {
		runSecretScanGit(t, project, "config", kv[0], kv[1])
	}
	testutil.AppendFile(t, filepath.Join(project, "a.go"), "// TODO: hidden by config\n")
	testutil.AppendFile(t, filepath.Join(project, "café.go"), "// FIXME: accent\n")
	runSecretScanGit(t, project, "add", "--", "a.go")
	testutil.WriteFile(t, filepath.Join(root, "work", "feat", "windows.md"), "- a.go | TODO | tracked follow-up\n")

	var out, errBuf bytes.Buffer
	// Staged mode has no whole-file fallback: only the diff can see the marker.
	if code := RunCheckWindows(root, []string{"feat", "--staged"}, &out, &errBuf); code != 0 || !strings.Contains(out.String(), "waived: a.go:2 TODO") {
		t.Fatalf("staged: code=%d out=%q err=%q", code, out.String(), errBuf.String())
	}
	out.Reset()
	if code := RunCheckWindows(root, []string{"feat"}, &out, &errBuf); code != 3 {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errBuf.String())
	}
	got := out.String()
	if !strings.Contains(got, "waived: a.go:2 TODO") || strings.Count(got, "unwaived: café.go:2 FIXME") != 1 || strings.Contains(got, `\303`) {
		t.Fatalf("out=%q", got)
	}
}

func TestWindowsMarkersRespectWordBoundaries(t *testing.T) {
	for _, line := range []string{"os.Exit(1)", "benefit(x)", "x := mytodo!(y)", "profit(a)"} {
		if got := windowsMarkers(line); len(got) != 0 {
			t.Fatalf("%q: markers=%v want none", line, got)
		}
	}
	for line, want := range map[string]string{"xit('later', fn)": "xit(", "  fit('focus', fn)": "fit(", "@pytest.mark.skipif(x)": "pytest.mark.skip", "    pytest.skip(\"flaky\")": "pytest.skip(", "pytest.xfail(\"later\")": "pytest.xfail("} {
		if got := windowsMarkers(line); len(got) != 1 || got[0] != want {
			t.Fatalf("%q: markers=%v want [%s]", line, got, want)
		}
	}
}

// A project nested in a repository subdirectory: its own workspace is never
// scanned, and windows.md rows use project-relative paths.
func TestCheckWindowsNestedProject(t *testing.T) {
	repo := t.TempDir()
	initWindowsRepo(t, repo)
	project := filepath.Join(repo, "sub")
	root := windowsWorkspace(t, project, "feat")
	testutil.WriteFile(t, filepath.Join(project, "a.go"), "package a\n")
	runSecretScanGit(t, repo, "add", "--", "sub/a.go")
	runSecretScanGit(t, repo, "commit", "-qm", "baseline")
	testutil.AppendFile(t, filepath.Join(project, "a.go"), "// TODO: waived\n")
	testutil.WriteFile(t, filepath.Join(root, "work", "feat", "notes.md"), "TODO: workspace prose\n")
	testutil.WriteFile(t, filepath.Join(root, "work", "feat", "windows.md"), "- a.go | TODO | tracked follow-up\n")

	var out, errBuf bytes.Buffer
	if code := RunCheckWindows(root, []string{"feat"}, &out, &errBuf); code != 0 {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errBuf.String())
	}
	if strings.Contains(out.String(), ".devrites") {
		t.Fatalf("own workspace scanned: %q", out.String())
	}
}

// The metric carries the workspace's cursor phase, not a hard-coded "seal".
func TestCheckWindowsRecordsCursorPhase(t *testing.T) {
	project := t.TempDir()
	initWindowsRepo(t, project)
	root := windowsWorkspace(t, project, "feat")
	testutil.WriteFile(t, filepath.Join(root, "work", "feat", "state.md"), "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | build |\n| status | running |\n")
	var out, errBuf bytes.Buffer
	if code := RunCheckWindows(root, []string{"feat"}, &out, &errBuf); code != 0 {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errBuf.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "work", "feat", MetricsFile))
	if err != nil || !strings.Contains(string(raw), `"phase":"build","event":"windows"`) {
		t.Fatalf("metrics=%q err=%v", raw, err)
	}
}
