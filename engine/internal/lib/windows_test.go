package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

func windowsWorkspace(t *testing.T, project, slug string) string {
	t.Helper()
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
