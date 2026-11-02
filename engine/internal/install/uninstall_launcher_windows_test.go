package install

import (
	"bytes"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestWindowsUninstallLaunchers(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("Node is required for native Windows launcher integration: %v", err)
	}
	bash := windowsGitBash(t)
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate uninstall launchers")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	launcher := filepath.Join(repoRoot, "bin", "devrites.mjs")
	engine := windowsEngine(t)
	for _, scenario := range []string{
		"Node/environment engine", "Node/PATH engine", "Node/staging failure",
		"Node/inaccessible PATH engine",
		"Git Bash/environment engine", "Git Bash/PATH engine", "Git Bash/staging failure",
	} {
		t.Run(scenario, func(t *testing.T) {
			launcherName, origin, _ := strings.Cut(scenario, "/")
			target, binDir, installed := windowsProject(t, engine)
			t.Setenv("DEVRITES_CLI", "")
			t.Setenv("DEVRITES_REF", "")
			t.Setenv("DEVRITES_REPO", "")
			t.Setenv("NODE_OPTIONS", "")
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			injectedStat := false
			if origin == "inaccessible PATH engine" {
				deniedDir, preload := windowsInaccessibleEnginePath(t, node)
				t.Setenv("PATH", deniedDir+string(os.PathListSeparator)+os.Getenv("PATH"))
				if preload != "" {
					t.Setenv("NODE_OPTIONS", "--import="+preload)
					injectedStat = true
				}
			}
			if !strings.HasSuffix(origin, "PATH engine") {
				t.Setenv("DEVRITES_ENGINE_CLI", installed)
			}
			profile, err := os.UserHomeDir()
			if err != nil {
				t.Fatal(err)
			}
			fallback := filepath.Join(profile, ".local", "bin", "devrites-engine.exe")
			if err := os.MkdirAll(filepath.Dir(fallback), 0o755); err != nil {
				t.Fatal(err)
			}
			fallbackBytes := []byte("unrelated fallback installation\n")
			if err := os.WriteFile(fallback, fallbackBytes, 0o755); err != nil {
				t.Fatal(err)
			}
			before := windowsProjectSnapshot(t, target)
			binaryBefore, err := os.ReadFile(installed)
			if err != nil {
				t.Fatal(err)
			}
			tempRoot := t.TempDir()
			tempPath := tempRoot
			if origin == "staging failure" {
				tempPath = filepath.Join(tempRoot, "not-a-directory")
				if err := os.WriteFile(tempPath, []byte("blocked temp directory\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("TEMP", tempPath)
			t.Setenv("TMP", tempPath)
			t.Setenv("TMPDIR", tempPath)
			cmd := exec.Command(node, launcher, "uninstall", "--target", target)
			if launcherName == "Git Bash" {
				cmd = exec.Command(bash, "--noprofile", "--norc", "-c", `
set -eu
export PATH="/usr/bin:/bin:$PATH"
TMPDIR="$(cygpath -u "$TMPDIR")"
DEVRITES_BIN_DIR="$(cygpath -w "$DEVRITES_BIN_DIR")"
if [ -n "$DEVRITES_ENGINE_CLI" ]; then
  DEVRITES_ENGINE_CLI="$(cygpath -u "$DEVRITES_ENGINE_CLI")"
fi
export TMPDIR DEVRITES_BIN_DIR DEVRITES_ENGINE_CLI
uninstaller="$(cygpath -u "$1")"
target="$(cygpath -u "$2")"
exec /usr/bin/bash "$uninstaller" --target "$target"
`, "devrites-uninstall-test", filepath.Join(repoRoot, "uninstall.sh"), target)
			}
			cmd.Dir = target
			out, err := cmd.CombinedOutput()
			if injectedStat && !strings.Contains(string(out), "injected PATH stat EACCES") {
				t.Errorf("inaccessible PATH fault was not exercised: %s", out)
			}
			if origin == "staging failure" {
				if err == nil || !strings.Contains(string(out), "could not stage devrites-engine for Windows uninstall") {
					t.Fatalf("staging failure error=%v output=%s", err, out)
				}
				if strings.Contains(string(out), "DevRites uninstalled") {
					t.Errorf("staging failure reported success: %s", out)
				}
				if after := windowsProjectSnapshot(t, target); !reflect.DeepEqual(before, after) {
					t.Error("staging failure changed project inventory or bytes")
				}
				if after, err := os.ReadFile(installed); err != nil || !bytes.Equal(after, binaryBefore) {
					t.Errorf("staging failure changed installed engine: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("%s launcher uninstall: %v\n%s", launcherName, err, out)
				}
				if !strings.Contains(string(out), "DevRites uninstalled") {
					t.Errorf("successful uninstall missing success output: %s", out)
				}
				for _, removed := range []string{installed, filepath.Join(target, ManifestName), filepath.Join(target, ".claude", "skills", "rite", "SKILL.md")} {
					if _, err := os.Stat(removed); !os.IsNotExist(err) {
						t.Errorf("uninstall did not remove %s: %v", removed, err)
					}
				}
				for rel, want := range map[string]string{
					"AGENTS.md":                        "user guidance\n",
					"README.txt":                       "user project\n",
					".devrites/work/feature/notes.txt": "user feature data\n",
				} {
					if got, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(rel))); err != nil || string(got) != want {
						t.Errorf("uninstall changed user file %s: %q, %v", rel, got, err)
					}
				}
				entries, err := os.ReadDir(tempRoot)
				if err != nil || len(entries) != 0 {
					t.Errorf("%s launcher left private staging files: %v, %v", launcherName, entries, err)
				}
			}
			if got, err := os.ReadFile(fallback); err != nil || !bytes.Equal(got, fallbackBytes) {
				t.Errorf("uninstall changed unrelated fallback engine: %q, %v", got, err)
			}
		})
	}
}

func windowsGitBash(t *testing.T) string {
	t.Helper()
	git, err := exec.LookPath("git.exe")
	if err != nil {
		t.Fatalf("Git for Windows is required for native launcher integration: %v", err)
	}
	for dir := filepath.Dir(git); ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "bin", "bash.exe")
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	t.Fatalf("Git Bash is required for native Windows launcher integration; no bin/bash.exe found above %s", git)
	return ""
}

func windowsInaccessibleEnginePath(t *testing.T, node string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	candidate := filepath.Join(dir, "devrites-engine.exe")
	if err := os.WriteFile(candidate, []byte("inaccessible PATH candidate\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	account, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	icacls, err := exec.LookPath("icacls.exe")
	if err != nil {
		t.Fatalf("icacls is required for native Windows PATH permission coverage: %v", err)
	}
	sid := "*" + account.Uid
	// Restore the directory first, then its file, before TempDir cleanup.
	for _, path := range []string{candidate, dir} {
		t.Cleanup(func() {
			if out, err := exec.Command(icacls, path, "/remove:d", sid).CombinedOutput(); err != nil {
				t.Errorf("restore temporary PATH ACL: %v\n%s", err, out)
			}
		})
		if out, err := exec.Command(icacls, path, "/deny", sid+":(RA,RD)").CombinedOutput(); err != nil {
			t.Fatalf("deny temporary PATH attributes and directory listing: %v\n%s", err, out)
		}
	}
	probe := exec.Command(node, "-e", `
const fs = require('node:fs');
try {
  fs.statSync(process.argv[1], { throwIfNoEntry: false });
  console.log('accessible');
} catch (error) {
  if (error.code !== 'EACCES' && error.code !== 'EPERM') throw error;
  console.log(error.code);
}
`, candidate)
	out, err := probe.CombinedOutput()
	if err != nil {
		t.Fatalf("probe native Windows Node stat permissions: %v\n%s", err, out)
	}
	if code := strings.TrimSpace(string(out)); code == "EACCES" || code == "EPERM" {
		t.Logf("native ACL produced Node stat %s", code)
		return dir, ""
	} else if code != "accessible" {
		t.Fatalf("unexpected native Windows stat probe output: %s", out)
	}
	// Some Windows stat APIs can read metadata despite these ACLs.
	t.Log("Node stat bypassed denied ACL; using EACCES fault injection, not native ACL proof")
	preload := filepath.Join(t.TempDir(), "inaccessible-path.mjs")
	t.Setenv("DEVRITES_TEST_INACCESSIBLE_ENGINE", candidate)
	if err := os.WriteFile(preload, []byte(`
import fs from 'node:fs';
import { syncBuiltinESMExports } from 'node:module';
const statSync = fs.statSync;
fs.statSync = (path, ...args) => {
  if (String(path).toLowerCase() === process.env.DEVRITES_TEST_INACCESSIBLE_ENGINE.toLowerCase()) {
    process.stderr.write('injected PATH stat EACCES\n');
    throw Object.assign(new Error('injected inaccessible PATH candidate'), { code: 'EACCES' });
  }
  return statSync(path, ...args);
};
syncBuiltinESMExports();
`), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, (&url.URL{Scheme: "file", Path: "/" + filepath.ToSlash(preload)}).String()
}
