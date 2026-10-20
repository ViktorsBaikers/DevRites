package lib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runDiffScopeT(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunCheckDiffScope(root, append([]string{"feat"}, args...), stdout, stderr)
	return code, stdout.String() + stderr.String()
}

// A path dirty before dispatch and byte-identical after is not a writer
// violation under --baseline; the same path changed, or a new path, still is.
func TestDiffScopeBaselineExcludesOnlyUnchangedPreexistingPaths(t *testing.T) {
	repo := diffScopeRepo(t)
	root := filepath.Join(repo, ".devrites")
	writeFile(t, filepath.Join(root, "work", "feat", "state.md"), "# State\n")
	writeFile(t, filepath.Join(repo, "src", "a.go"), "package a\n")
	runDiffScopeGit(t, repo, "add", "-A")
	runDiffScopeGit(t, repo, "commit", "-qm", "base")
	writeFile(t, filepath.Join(repo, ".gitignore"), "node_modules\n") // pre-existing untracked dirt

	if code, out := runDiffScopeT(t, root, "--snapshot"); code != 0 || !strings.Contains(out, "snapshot (1 changed path(s))") {
		t.Fatalf("snapshot: code=%d\n%s", code, out)
	}
	writeFile(t, filepath.Join(repo, "src", "a.go"), "package a // wright\n")

	// The snapshot file itself is an engine machine file, never a task path.
	if code, out := runDiffScopeT(t, root, "--allow", "src", "--baseline"); code != 0 || !strings.Contains(out, "preexisting (unchanged): .gitignore") || !strings.Contains(out, "ok (1 changed") {
		t.Fatalf("unchanged preexisting path: code=%d\n%s", code, out)
	}
	if code, out := runDiffScopeT(t, root, "--allow", "src"); code != 3 || !strings.Contains(out, "BLOCKED: .gitignore outside allowlist") {
		t.Fatalf("without --baseline the gate is unchanged: code=%d\n%s", code, out)
	}
	writeFile(t, filepath.Join(repo, ".gitignore"), "node_modules\nweb\n")
	if code, out := runDiffScopeT(t, root, "--allow", "src", "--baseline"); code != 3 || !strings.Contains(out, "BLOCKED: .gitignore outside allowlist") {
		t.Fatalf("changed preexisting path must block: code=%d\n%s", code, out)
	}
	writeFile(t, filepath.Join(repo, ".gitignore"), "node_modules\n")
	writeFile(t, filepath.Join(repo, "other.txt"), "x\n")
	if code, out := runDiffScopeT(t, root, "--allow", "src", "--baseline"); code != 3 || !strings.Contains(out, "BLOCKED: other.txt outside allowlist") {
		t.Fatalf("new path must block: code=%d\n%s", code, out)
	}
	if err := os.Remove(filepath.Join(repo, "other.txt")); err != nil {
		t.Fatal(err)
	}
	runDiffScopeGit(t, repo, "checkout", "--", "src/a.go")
	if code, out := runDiffScopeT(t, root, "--allow", "src", "--baseline"); code != 3 || !strings.Contains(out, "empty diff") {
		t.Fatalf("only preexisting dirt is an empty diff: code=%d\n%s", code, out)
	}
	if code, out := runDiffScopeT(t, root, "--snapshot", "--baseline"); code != 2 {
		t.Fatalf("--snapshot with --baseline: code=%d\n%s", code, out)
	}
	if err := os.Remove(filepath.Join(root, "work", "feat", diffScopeBaselineFile)); err != nil {
		t.Fatal(err)
	}
	if code, out := runDiffScopeT(t, root, "--allow", "src", "--baseline"); code != 2 || !strings.Contains(out, "run --snapshot") {
		t.Fatalf("missing baseline must fail closed: code=%d\n%s", code, out)
	}
}

// Hundreds of violations print as bounded groups with the same exit;
// --verbose still lists every path.
func TestDiffScopeGroupsManyViolations(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	repo := diffScopeRepo(t)
	for i := range 550 {
		writeFile(t, filepath.Join(repo, "web", "node_modules", fmt.Sprintf("p%03d", i), "index.js"), "x\n")
	}
	for i := range 25 {
		writeFile(t, filepath.Join(repo, "web", ".svelte-kit", fmt.Sprintf("s%02d", i)), "x\n")
		writeFile(t, filepath.Join(repo, fmt.Sprintf("top%02d.txt", i)), "x\n")
	}
	code, out := runDiffScopeT(t, root, "--allow", "src", "--cwd", repo)
	lines := strings.Count(out, "\n")
	if code != 3 || lines >= 40 || !strings.Contains(out, "600 path(s) outside allowlist in 27 group(s)") ||
		!strings.Contains(out, "web/node_modules/ 550 path(s), e.g. web/node_modules/p000/index.js") ||
		!strings.Contains(out, "7 more group(s)") {
		t.Fatalf("grouped: code=%d lines=%d\n%s", code, lines, out)
	}
	code, out = runDiffScopeT(t, root, "--allow", "src", "--cwd", repo, "--verbose")
	if code != 3 || strings.Count(out, " outside allowlist\n") != 600 {
		t.Fatalf("--verbose: code=%d listed=%d", code, strings.Count(out, " outside allowlist\n"))
	}
}
