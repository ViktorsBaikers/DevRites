package lib

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func recallManifest(paths ...string) string {
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	var b strings.Builder
	b.WriteString("# Touched files\n\n## Touched files\nCandidate paths are declared only in the manifest below.\n\n## Candidate manifest\n| State | File | Slice | Reason |\n| --- | --- | --- | --- |\n")
	for _, p := range sorted {
		fmt.Fprintf(&b, "| present | `%s` | SLICE-001 | Reason. |\n", p)
	}
	return b.String()
}

// writeRecallFeature archives slug with a manifest over paths plus extra
// artifacts, then pins the archive directory mtime to age ago.
func writeRecallFeature(t *testing.T, root, slug string, age time.Duration, extra map[string]string, paths ...string) string {
	t.Helper()
	dir := filepath.Join(root, "archive", slug)
	writeFile(t, filepath.Join(dir, "touched-files.md"), recallManifest(paths...))
	for name, body := range extra {
		writeFile(t, filepath.Join(dir, name), body)
	}
	at := time.Now().Add(-age)
	if err := os.Chtimes(dir, at, at); err != nil {
		t.Fatal(err)
	}
	return dir
}

func runRecall(t *testing.T, root string, args ...string) (int, string, string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunRecall(root, args, stdout, stderr)
	return code, stdout.String(), stderr.String()
}

func TestRecallRanksByOverlapThenRecency(t *testing.T) {
	root := t.TempDir()
	writeRecallFeature(t, root, "older", 3*time.Hour, nil, "src/a.go")
	writeRecallFeature(t, root, "newer", time.Hour, nil, "src/a.go")
	writeRecallFeature(t, root, "both", 5*time.Hour, nil, "src/a.go", "src/b.go")
	writeRecallFeature(t, root, "unrelated", 0, nil, "docs/x.md")

	code, out, errOut := runRecall(t, root, "--paths", "src/a.go", "--paths", "src/b.go")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	want := "recall: 3 archived feature(s) touched these paths\n" +
		"feature both overlap=2 paths=src/a.go,src/b.go\n" +
		"feature newer overlap=1 paths=src/a.go\n" +
		"feature older overlap=1 paths=src/a.go\n"
	if out != want {
		t.Fatalf("output:\n%s\nwant:\n%s", out, want)
	}

	code, out, _ = runRecall(t, root, "--paths", "src/a.go,src/b.go", "--limit", "1")
	if code != 0 || !strings.Contains(out, "recall: truncated to 1 feature(s) (--limit)\nfeature both ") || strings.Contains(out, "feature newer") {
		t.Fatalf("limit: code=%d out:\n%s", code, out)
	}
}

func TestRecallDirectoryPrefixAndCaseFold(t *testing.T) {
	root := t.TempDir()
	writeRecallFeature(t, root, "feat", 0, nil, "Engine/Internal/Lib/Recall.go")
	for _, query := range []string{"engine/internal", "ENGINE/internal/lib/recall.go", "./engine/internal/lib/"} {
		code, out, _ := runRecall(t, root, "--paths", query)
		if code != 0 || !strings.Contains(out, "feature feat overlap=1") {
			t.Fatalf("query %q: code=%d out:\n%s", query, code, out)
		}
	}
	code, out, _ := runRecall(t, root, "--paths", "engine/int")
	if code != 0 || out != "recall: no archived feature touched these paths\n" {
		t.Fatalf("partial segment matched: code=%d out:\n%s", code, out)
	}
}

func TestRecallDecisionAndConstraintPointers(t *testing.T) {
	root := t.TempDir()
	decisions := strings.Join([]string{
		"# Decisions",                     // 1
		"",                                // 2
		"## Decision log",                 // 3
		"",                                // 4
		"## DEC-001 — Reuse the store",    // 5
		"- Decision: reuse.",              // 6
		"- Files: `src/a.go`, `src/c.go`", // 7
		"",                                // 8
		"## DEC-002 — Unrelated",          // 9
		"- Files: other/x.go",             // 10
		"",                                // 11
		"## DEC-003 — No files field",     // 12
		"- Decision: mentions src/a.go in prose only.", // 13
		"",                            // 14
		"```markdown",                 // 15
		"## DEC-009 — Fenced example", // 16
		"- Files: src/a.go",           // 17
		"- example — from SLICE-009 (EVID-9); Files: src/a.go", // 18
		"```",                          // 19
		"",                             // 20
		"## Carry-forward constraints", // 21
		"",                             // 22
		"- Validate before write — from SLICE-002 (EVID-004); Files: src/a.go", // 23
		"- Unrelated rule — from SLICE-003 (EVID-005); Files: other/y.go",      // 24
		"- Always run gofmt — from SLICE-004 (review R2)",                      // 25
		"- Keep docs in sync — from SLICE-005 (EVID-006); Files: none",         // 26
	}, "\n") + "\n"
	writeRecallFeature(t, root, "feat", 0, map[string]string{"decisions.md": decisions}, "src/a.go")

	code, out, errOut := runRecall(t, root, "--paths", "src")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	want := "recall: 1 archived feature(s) touched these paths\n" +
		"feature feat overlap=1 paths=src\n" +
		"  archive/feat/decisions.md:5  ## DEC-001 — Reuse the store\n" +
		"  archive/feat/decisions.md:23  - Validate before write — from SLICE-002 (EVID-004); Files: src/a.go\n" +
		"  archive/feat/decisions.md:25  - Always run gofmt — from SLICE-004 (review R2)\n" +
		"  archive/feat/decisions.md:26  - Keep docs in sync — from SLICE-005 (EVID-006); Files: none\n"
	if out != want {
		t.Fatalf("output:\n%s\nwant:\n%s", out, want)
	}
}

func TestRecallMentionPointersArePathBoundaryAware(t *testing.T) {
	root := t.TempDir()
	review := "# Review\n\n- `src/a.go:12` off-by-one fixed\n- src/a.gox is a different file\n- mysrc/a.go is a different file\n- lib/src/a.go is a different file\n"
	seal := "# Seal\n\n```text\nsrc/a.go fenced example\n```\nSRC/A.go reviewed.\n"
	drift := "# Drift\n\n## DRIFT-001\nChanged (src/a.go) after vet.\n" + strings.Repeat("x", 200) + " src/a.go\n"
	writeRecallFeature(t, root, "feat", 0, map[string]string{"review.md": review, "seal.md": seal, "drift.md": drift}, "src/a.go")

	code, out, errOut := runRecall(t, root, "--paths", "src/a.go")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	want := "recall: 1 archived feature(s) touched these paths\n" +
		"feature feat overlap=1 paths=src/a.go\n" +
		"  archive/feat/review.md:3  - `src/a.go:12` off-by-one fixed\n" +
		"  archive/feat/seal.md:6  SRC/A.go reviewed.\n" +
		"  archive/feat/drift.md:4  Changed (src/a.go) after vet.\n" +
		"  archive/feat/drift.md:5  " + strings.Repeat("x", 120) + "…\n"
	if out != want {
		t.Fatalf("output:\n%s\nwant:\n%s", out, want)
	}
}

func TestRecallCapsPointersPerFeature(t *testing.T) {
	root := t.TempDir()
	var review strings.Builder
	for i := 1; i <= 11; i++ {
		fmt.Fprintf(&review, "- finding %d in src/a.go\n", i)
	}
	writeRecallFeature(t, root, "feat", 0, map[string]string{"review.md": review.String()}, "src/a.go")
	_, out, _ := runRecall(t, root, "--paths", "src/a.go")
	if got := strings.Count(out, "  archive/feat/review.md:"); got != 8 {
		t.Fatalf("pointers=%d want 8:\n%s", got, out)
	}
	if !strings.HasSuffix(out, "  archive/feat/review.md:8  - finding 8 in src/a.go\n  +3 more\n") {
		t.Fatalf("missing +N more tail:\n%s", out)
	}
}

func TestRecallNoArchiveAndNoMatch(t *testing.T) {
	root := t.TempDir()
	code, out, _ := runRecall(t, root, "--paths", "src/a.go")
	if code != 0 || out != "recall: no archive\n" {
		t.Fatalf("no archive: code=%d out=%q", code, out)
	}
	code, out, _ = runRecall(t, filepath.Join(t.TempDir(), "missing", ".devrites"), "--paths", "src/a.go")
	if code != 0 || out != "recall: no archive\n" {
		t.Fatalf("missing root: code=%d out=%q", code, out)
	}
	writeRecallFeature(t, root, "feat", 0, nil, "docs/x.md")
	if err := os.MkdirAll(filepath.Join(root, "archive", "spec-only"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, _ = runRecall(t, root, "--paths", "src/a.go")
	if code != 0 || out != "recall: no archived feature touched these paths\n" {
		t.Fatalf("no match: code=%d out=%q", code, out)
	}
}

func TestRecallRefusesSymlinksAndReportsUnreadable(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "evil", "touched-files.md"), recallManifest("src/a.go"))
	writeFile(t, filepath.Join(outside, "decisions.md"), "## DEC-001 — Outside\n- Files: src/a.go\n")
	if err := os.MkdirAll(filepath.Join(root, "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "evil"), filepath.Join(root, "archive", "evil")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	dir := writeRecallFeature(t, root, "feat", 0, map[string]string{"review.md": "- src/a.go ok\n"}, "src/a.go")
	if err := os.Symlink(filepath.Join(outside, "decisions.md"), filepath.Join(dir, "decisions.md")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "archive", "broken", "touched-files.md"), "# not a manifest\n")

	code, out, errOut := runRecall(t, root, "--paths", "src/a.go")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if strings.Contains(out, "feature evil") || strings.Contains(out, "Outside") {
		t.Fatalf("followed a symlink out of the archive:\n%s", out)
	}
	for _, want := range []string{
		"feature feat overlap=1 paths=src/a.go\n  archive/feat/review.md:1  - src/a.go ok\n  unreadable: archive/feat/decisions.md (",
		"\nunreadable: archive/broken/touched-files.md (",
		"\nunreadable: archive/evil (DRV-WORKSPACE-SYMLINK",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestRecallRejectsBadArguments(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{
		nil,
		{"--paths"},
		{"--paths", ""},
		{"--paths", "--limit"},
		{"--paths", "/etc/passwd"},
		{"--paths", `C:\x`},
		{"--paths", "../x"},
		{"--paths", "src/../../x"},
		{"--paths", "src,-rf"},
		{"--paths", "src", "--limit", "0"},
		{"--paths", "src", "--limit", "x"},
		{"--paths", "src", "--limit"},
		{"--paths", "src", "--bogus"},
		{"--paths", "src", "extra"},
	} {
		code, out, errOut := runRecall(t, root, args...)
		if code != 2 || out != "" || errOut == "" {
			t.Errorf("%q: code=%d stdout=%q stderr=%q, want usage exit 2", args, code, out, errOut)
		}
	}
}

func recallTreeSnapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	snap := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		sig := fmt.Sprintf("%v %d %d", info.Mode(), info.Size(), info.ModTime().UnixNano())
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(path) // #nosec G304 -- test temp tree
			if err != nil {
				return err
			}
			sig += fmt.Sprintf(" %x", sha256.Sum256(data))
		}
		snap[path] = sig
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

func TestRecallNeverWrites(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	writeFile(t, filepath.Join(root, "ACTIVE"), "feat\n")
	writeRecallFeature(t, root, "old", 0, map[string]string{
		"decisions.md": "## DEC-001 — Store\n- Files: src/a.go\n",
		"review.md":    "- src/a.go reviewed\n",
	}, "src/a.go")
	before := recallTreeSnapshot(t, root)
	for _, args := range [][]string{{"--paths", "src/a.go"}, {"--paths", "nothing"}, {"--bogus"}} {
		runRecall(t, root, args...)
	}
	after := recallTreeSnapshot(t, root)
	if fmt.Sprint(before) != fmt.Sprint(after) {
		t.Fatalf("recall changed the tree:\nbefore=%v\nafter=%v", before, after)
	}
}
