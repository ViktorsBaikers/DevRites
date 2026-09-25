package snapshot

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/devrites/devrites/internal/gitenv"
)

const (
	envSecret = "hunter2-env-value"
	keySecret = "secretkeybody-xyz"
)

func gitRun(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(gitenv.Sanitize(os.Environ()), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
	return cmd.CombinedOutput()
}

func gitT(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := gitRun(dir, args...); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func lines(first, last string) string {
	l := []string{first, "two", "three", "four", "five", "six", "seven", "eight", "nine", last}
	return strings.Join(l, "\n") + "\n"
}

// newRepo commits src/a.py, src/b.py, src/clean.py and gone.txt.
func newRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	gitT(t, root, "init", "-q", "-b", "main")
	write(t, root, "src/a.py", lines("one", "ten"))
	write(t, root, "src/b.py", "b\n")
	write(t, root, "gone.txt", "gone\n")
	write(t, root, "src/clean.py", "clean\n")
	gitT(t, root, "add", "-A")
	gitT(t, root, "commit", "-q", "-m", "init")
	return root
}

// dirtyRepo adds a staged edit, an unstaged edit on top, an unstaged edit
// elsewhere, a deleted tracked file, an untracked note and two secrets.
func dirtyRepo(t *testing.T) string {
	t.Helper()
	root := newRepo(t)
	write(t, root, "src/a.py", lines("ONE-staged", "ten"))
	gitT(t, root, "add", "src/a.py")
	write(t, root, "src/a.py", lines("ONE-staged", "TEN-unstaged"))
	write(t, root, "src/b.py", "B-unstaged\n")
	if err := os.Remove(filepath.Join(root, "gone.txt")); err != nil {
		t.Fatal(err)
	}
	write(t, root, "note.txt", "note\n")
	write(t, root, ".env", "TOKEN="+envSecret+"\n")
	write(t, root, "id_material.txt", "-----BEGIN OPENSSH PRIVATE KEY-----\n"+keySecret+"\n-----END OPENSSH PRIVATE KEY-----\n")
	write(t, root, "deploy.tfvars", "token = \"staged-"+envSecret+"\"\n")
	gitT(t, root, "add", "deploy.tfvars")
	write(t, root, "deploy.tfvars", "token = \"unstaged-"+keySecret+"\"\n")
	return root
}

func run(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func indexState(t *testing.T, root string) (string, time.Time) {
	t.Helper()
	p := filepath.Join(root, ".git", "index")
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return read(t, p), fi.ModTime()
}

func loadManifest(t *testing.T, out string) manifest {
	t.Helper()
	var m manifest
	if err := json.Unmarshal([]byte(read(t, filepath.Join(out, "manifest.json"))), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func captureOK(t *testing.T, root string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "snap")
	if code, _, stderr := run("capture", root, out); code != 0 {
		t.Fatalf("capture exit %d: %s", code, stderr)
	}
	return out
}

func TestCaptureDirtyRepo(t *testing.T) {
	root := dirtyRepo(t)
	idxBefore, mtimeBefore := indexState(t, root)
	out := filepath.Join(t.TempDir(), "snap")
	code, stdout, stderr := run("capture", root, out)
	if code != 0 {
		t.Fatalf("capture exit %d: %s", code, stderr)
	}
	if idx, mtime := indexState(t, root); idx != idxBefore || !mtime.Equal(mtimeBefore) {
		t.Fatal(".git/index bytes or mtime changed during capture")
	}

	m := loadManifest(t, out)
	if m.Schema != "overhaul.snapshot/1" || m.Head == nil || m.IndexSHA256 == "" || m.Fingerprint == "" {
		t.Fatalf("manifest header: %+v", m)
	}
	want := map[string]entry{
		"src/a.py":        {Source: "tracked", Copied: true},
		"src/b.py":        {Source: "tracked", Copied: true},
		"gone.txt":        {Source: "tracked", SHA256: "deleted"},
		"src/clean.py":    {Source: "tracked"}, // clean: git holds it, so only its hash is kept
		"note.txt":        {Source: "untracked", Copied: true},
		".env":            {Source: "untracked", Sensitive: true},
		"id_material.txt": {Source: "untracked", Sensitive: true},
	}
	for p, w := range want {
		g := m.Files[p]
		if g == nil || g.Source != w.Source || g.Copied != w.Copied || g.Sensitive != w.Sensitive ||
			(w.SHA256 != "" && g.SHA256 != w.SHA256) {
			t.Errorf("%s: got %+v want %+v", p, g, w)
		}
		_, err := os.Lstat(filepath.Join(out, "tree", p))
		if copied := err == nil; copied != w.Copied {
			t.Errorf("%s: copied on disk = %v, want %v", p, copied, w.Copied)
		}
	}
	if got := read(t, filepath.Join(out, "tree", "src/a.py")); got != lines("ONE-staged", "TEN-unstaged") {
		t.Errorf("tree copy of src/a.py = %q", got)
	}

	staged := read(t, filepath.Join(out, "staged.patch"))
	unstaged := read(t, filepath.Join(out, "unstaged.patch"))
	if !strings.Contains(staged, "+ONE-staged") || strings.Contains(staged, "TEN-unstaged") ||
		strings.Contains(staged, "B-unstaged") || strings.Contains(staged, "gone.txt") {
		t.Errorf("staged.patch:\n%s", staged)
	}
	if strings.Contains(unstaged, "+ONE-staged") || !strings.Contains(unstaged, "+TEN-unstaged") ||
		!strings.Contains(unstaged, "+B-unstaged") || !strings.Contains(unstaged, "deleted file mode") {
		t.Errorf("unstaged.patch:\n%s", unstaged)
	}
	for _, name := range []string{"status.bin", "index.bin"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Error(err)
		}
	}

	var summary struct {
		Withheld []string `json:"sensitive_not_copied"`
	}
	if err := json.Unmarshal([]byte(stdout), &summary); err != nil ||
		!slices.Equal(summary.Withheld, []string{".env", "deploy.tfvars", "id_material.txt"}) {
		t.Errorf("summary %q: %v", stdout, err)
	}
	leaks := stdout + stderr
	if err := filepath.WalkDir(out, func(p string, d os.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			leaks += read(t, p)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{envSecret, keySecret} {
		if strings.Contains(leaks, s) {
			t.Errorf("secret %q leaked into output or snapshot", s)
		}
	}

	fi, err := os.Stat(out)
	if err != nil || (runtime.GOOS != "windows" && fi.Mode().Perm() != 0o700) {
		t.Errorf("snapshot dir mode: %v %v", fi.Mode(), err)
	}
}

func TestCaptureRefusals(t *testing.T) {
	root := newRepo(t)

	inside := filepath.Join(root, "snap")
	code, _, stderr := run("capture", root, inside)
	if code != 2 || !strings.Contains(stderr, "snapshot must live outside the target repository or inside its .devrites/overhaul/ run area") {
		t.Errorf("inside target: exit %d %q", code, stderr)
	}
	if _, err := os.Lstat(inside); err == nil {
		t.Error("snapshot created inside the target")
	}

	// The run area inside the target is the one allowed place, and never counts as repository content.
	before := fingerprintOf(t, root)
	runArea := filepath.Join(root, ".devrites", "overhaul", "r1")
	if code, _, stderr := run("capture", root, filepath.Join(runArea, "baseline")); code != 0 {
		t.Fatalf("capture into run area: exit %d %q", code, stderr)
	}
	write(t, runArea, "g0001/run.json", "{}\n")
	if after := fingerprintOf(t, root); after != before {
		t.Error("run area contents changed the repository fingerprint")
	}

	existing := t.TempDir()
	write(t, existing, "keep.txt", "keep")
	code, _, stderr = run("capture", root, existing)
	if code != 2 || !strings.Contains(stderr, "refusing to overwrite existing snapshot") {
		t.Errorf("overwrite: exit %d %q", code, stderr)
	}
	if entries, _ := os.ReadDir(existing); len(entries) != 1 {
		t.Errorf("existing snapshot dir was modified: %v", entries)
	}
}

func TestCaptureRefusesUnmergedIndex(t *testing.T) {
	root := newRepo(t)
	gitT(t, root, "checkout", "-q", "-b", "other")
	write(t, root, "src/b.py", "other\n")
	gitT(t, root, "commit", "-q", "-am", "other")
	gitT(t, root, "checkout", "-q", "main")
	write(t, root, "src/b.py", "main\n")
	gitT(t, root, "commit", "-q", "-am", "main")
	if out, err := gitRun(root, "merge", "other"); err == nil {
		t.Fatalf("expected a merge conflict: %s", out)
	}
	out := filepath.Join(t.TempDir(), "snap")
	code, _, stderr := run("capture", root, out)
	if code != 2 || !strings.Contains(stderr, "index has unmerged entries") {
		t.Fatalf("exit %d %q", code, stderr)
	}
	if _, err := os.Lstat(out); err == nil {
		t.Error("snapshot dir created despite refusal")
	}
}

type report struct {
	IndexUnchanged bool     `json:"index_unchanged"`
	Agent          []string `json:"agent_owned_changes"`
	User           []string `json:"changes_outside_agent_paths"`
}

func verifyRun(t *testing.T, root, out string) (int, report) {
	t.Helper()
	agentFile := filepath.Join(t.TempDir(), "agent.txt")
	write(t, filepath.Dir(agentFile), "agent.txt", "src/a.py\n\n")
	code, stdout, stderr := run("verify", root, out, "--agent-paths", agentFile)
	var r report
	if err := json.Unmarshal([]byte(stdout), &r); err != nil {
		t.Fatalf("verify exit %d output %q %s: %v", code, stdout, stderr, err)
	}
	return code, r
}

func TestVerifySeparatesAgentAndUserChanges(t *testing.T) {
	root := dirtyRepo(t)
	out := captureOK(t, root)

	write(t, root, "src/a.py", "agent edit\n")
	code, r := verifyRun(t, root, out)
	if code != 0 || !r.IndexUnchanged || !slices.Equal(r.Agent, []string{"src/a.py"}) || len(r.User) != 0 {
		t.Fatalf("agent-only change: exit %d %+v", code, r)
	}

	write(t, root, "note.txt", "user edit\n")
	code, r = verifyRun(t, root, out)
	if code != 1 || !slices.Equal(r.User, []string{"note.txt"}) || !slices.Equal(r.Agent, []string{"src/a.py"}) {
		t.Fatalf("user change: exit %d %+v", code, r)
	}
}

func TestVerifyDetectsIndexChange(t *testing.T) {
	root := dirtyRepo(t)
	out := captureOK(t, root)
	gitT(t, root, "add", "src/b.py")
	code, r := verifyRun(t, root, out)
	if code != 1 || r.IndexUnchanged || len(r.User) != 0 {
		t.Fatalf("exit %d %+v", code, r)
	}
}

func TestCaptureStaysReadOnlyAtBoundaries(t *testing.T) {
	root := newRepo(t)
	gitT(t, root, "config", "diff.noprefix", "true")
	gitT(t, root, "config", "diff.external", "false")

	inner := filepath.Join(root, "vendor", "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	gitT(t, inner, "init", "-q", "-b", "main")
	write(t, inner, "lib.txt", "inner\n")
	gitT(t, inner, "add", "-A")
	gitT(t, inner, "commit", "-q", "-m", "inner")
	gitT(t, root, "add", "vendor/inner")

	write(t, root, "src/a.py", lines("ONE-staged", "ten"))
	gitT(t, root, "add", "src/a.py")
	write(t, root, "src/a.py", lines("ONE-staged", "TEN-unstaged"))
	// Stale stat info: content matches the index but the timestamp does not.
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(filepath.Join(root, "src/b.py"), old, old); err != nil {
		t.Fatal(err)
	}

	idxBefore, mtimeBefore := indexState(t, root)
	out := captureOK(t, root)
	if idx, mtime := indexState(t, root); idx != idxBefore || !mtime.Equal(mtimeBefore) {
		t.Fatal(".git/index changed during capture")
	}

	m := loadManifest(t, out)
	if !slices.Contains(m.Submodules, "vendor/inner") {
		t.Errorf("submodules_not_captured = %v", m.Submodules)
	}
	if e := m.Files["vendor/inner"]; e == nil || e.SHA256 != "directory" || e.Copied {
		t.Errorf("gitlink entry = %+v", e)
	}
	if _, err := os.Lstat(filepath.Join(out, "tree", "vendor", "inner", "lib.txt")); err == nil {
		t.Error("nested repository content was copied")
	}
	for _, name := range []string{"staged.patch", "unstaged.patch"} {
		if p := read(t, filepath.Join(out, name)); !strings.Contains(p, "diff --git a/src/a.py b/src/a.py") {
			t.Errorf("%s lacks prefixed header:\n%s", name, p)
		}
	}
}

func TestDeltaReportsOnlyAttemptChanges(t *testing.T) {
	root := dirtyRepo(t)
	before := filepath.Join(t.TempDir(), "before.json")
	if code, _, stderr := run("state", root, before); code != 0 {
		t.Fatalf("state exit %d: %s", code, stderr)
	}
	write(t, root, "src/b.py", "changed during the attempt\n")
	code, stdout, stderr := run("delta", before, root)
	if code != 0 || stdout != "src/b.py\n" {
		t.Fatalf("delta exit %d stdout %q stderr %q", code, stdout, stderr)
	}
}

func fingerprintOf(t *testing.T, root string) string {
	t.Helper()
	code, stdout, stderr := run("fingerprint", root)
	if code != 0 {
		t.Fatalf("fingerprint exit %d: %s", code, stderr)
	}
	return strings.TrimSpace(stdout)
}
