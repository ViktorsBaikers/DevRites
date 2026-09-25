package records

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func gitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q", root)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	return root
}

func initRunCmd(args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := Run(append([]string{"init"}, args...), &o, &e)
	return code, o.String(), e.String()
}

func TestInitCreatesIgnoredRunArea(t *testing.T) {
	repo := gitRepo(t)
	code, out, errs := initRunCmd(repo, "ov-1")
	if code != 0 {
		t.Fatalf("init: %d %s", code, errs)
	}
	var got struct{ Run, Ignore string }
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output %q: %v", out, err)
	}
	want := filepath.Join(repo, ".devrites", "overhaul", "ov-1")
	if r1, _ := filepath.EvalSymlinks(got.Run); r1 != mustEval(t, want) || got.Ignore != "added-to-info-exclude" {
		t.Fatalf("got %+v, want run %s added-to-info-exclude", got, want)
	}
	if fi, err := os.Stat(want); err != nil || !fi.IsDir() || (runtime.GOOS != "windows" && fi.Mode().Perm() != 0o700) {
		t.Fatalf("run dir: %v %v", fi, err)
	}
	b, _ := os.ReadFile(filepath.Join(repo, ".git", "info", "exclude"))
	if !strings.Contains(string(b), "/.devrites/overhaul/\n") {
		t.Fatalf("exclude file: %q", b)
	}
	// Idempotent exclude, refused reuse of an existing run id.
	if code, _, _ := initRunCmd(repo, "ov-2"); code != 0 {
		t.Fatal("second run id refused")
	}
	b2, _ := os.ReadFile(filepath.Join(repo, ".git", "info", "exclude"))
	if strings.Count(string(b2), "/.devrites/overhaul/") != 1 {
		t.Fatalf("exclude line duplicated: %q", b2)
	}
	if code, _, errs := initRunCmd(repo, "ov-1"); code != 2 || !strings.Contains(errs, "already exists") {
		t.Fatalf("reuse: %d %s", code, errs)
	}
}

func TestInitRespectsExistingIgnore(t *testing.T) {
	repo := gitRepo(t)
	write(t, filepath.Join(repo, ".gitignore"), ".devrites/\n")
	code, out, errs := initRunCmd(repo, "ov-1")
	if code != 0 || !strings.Contains(out, `"already-ignored"`) {
		t.Fatalf("init: %d %s %s", code, out, errs)
	}
	if b, err := os.ReadFile(filepath.Join(repo, ".git", "info", "exclude")); err == nil && strings.Contains(string(b), "overhaul") {
		t.Fatalf("exclude touched although already ignored: %q", b)
	}
}

func TestInitRejectsBadInput(t *testing.T) {
	repo := gitRepo(t)
	for _, id := range []string{"", "../x", "a/b", ".hidden", "x y"} {
		if code, _, _ := initRunCmd(repo, id); code != 2 {
			t.Errorf("run id %q accepted", id)
		}
	}
	if code, _, _ := initRunCmd(t.TempDir(), "ov-1"); code != 2 {
		t.Error("non-repository accepted")
	}
}

func mustEval(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
