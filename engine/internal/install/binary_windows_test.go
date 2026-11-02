package install

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

func TestWindowsInstalledEngineUninstall(t *testing.T) {
	engine := windowsEngine(t)
	for _, scenario := range []string{"installed path", "case alias", "hard link alias", "keep binary", "no binary", "no binary env", "dry run", "other selected binary"} {
		t.Run(scenario, func(t *testing.T) {
			target, binDir, installed := windowsProject(t, engine)
			before := windowsProjectSnapshot(t, target)
			binaryBefore, err := os.ReadFile(installed)
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"uninstall", "--target", target}
			launch := installed
			selected := installed
			refused := false
			switch scenario {
			case "installed path":
				refused = true
			case "case alias":
				t.Setenv("DEVRITES_BIN_DIR", strings.ToUpper(binDir))
				refused = true
			case "hard link alias":
				launch = filepath.Join(t.TempDir(), "engine-alias.exe")
				if err := os.Link(installed, launch); err != nil {
					t.Fatal(err)
				}
				refused = true
			case "keep binary":
				args = append(args, "--keep-binary")
			case "no binary":
				args = append(args, "--no-binary")
			case "no binary env":
				t.Setenv("DEVRITES_NO_BINARY", "1")
			case "dry run":
				args = append(args, "--dry-run")
			case "other selected binary":
				otherDir := t.TempDir()
				t.Setenv("DEVRITES_BIN_DIR", otherDir)
				selected = filepath.Join(otherDir, "devrites-engine.exe")
				testutil.WriteFile(t, selected, "other installation\n")
			}
			cmd := exec.Command(launch, args...)
			out, err := cmd.CombinedOutput()
			if refused {
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 1 {
					t.Fatalf("direct uninstall error=%v output=%s, want exit 1", err, out)
				}
				for _, advice := range []string{"running engine binary", "npx devrites@latest uninstall", "--keep-binary"} {
					if !strings.Contains(string(out), advice) {
						t.Errorf("refusal missing %q: %s", advice, out)
					}
				}
				if strings.Contains(string(out), "DevRites uninstalled") {
					t.Errorf("refusal reported success: %s", out)
				}
			} else if err != nil {
				t.Fatalf("uninstall error=%v output=%s", err, out)
			}
			if refused || scenario == "dry run" {
				if after := windowsProjectSnapshot(t, target); !reflect.DeepEqual(before, after) {
					t.Error("uninstall refusal/dry run changed project inventory or bytes")
				}
			} else {
				if exists(filepath.Join(target, ManifestName)) || exists(filepath.Join(target, ".claude", "skills", "rite", "SKILL.md")) {
					t.Error("successful uninstall left managed project files")
				}
				for rel, want := range map[string]string{
					"AGENTS.md":                        "user guidance",
					"README.txt":                       "user project",
					".devrites/work/feature/notes.txt": "user feature data",
				} {
					if got := strings.TrimSpace(testutil.ReadFile(t, filepath.Join(target, filepath.FromSlash(rel)))); got != want {
						t.Errorf("uninstall changed user file %s: %q", rel, got)
					}
				}
			}
			if after, err := os.ReadFile(installed); err != nil || !bytes.Equal(after, binaryBefore) {
				t.Errorf("uninstall changed executing installation: error=%v", err)
			}
			if selected != installed && exists(selected) {
				t.Error("uninstall did not remove explicitly selected other installation")
			}
		})
	}
}

// Build the actual engine once in each calling test, then share it across scenarios.
func windowsEngine(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate engine source")
	}
	engineRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	binary := filepath.Join(t.TempDir(), "devrites-engine.exe")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = engineRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build Windows engine: %v\n%s", err, out)
	}
	return binary
}

func windowsProject(t *testing.T, engine string) (target, binDir, installed string) {
	t.Helper()
	profile := t.TempDir()
	t.Setenv("HOME", profile)
	t.Setenv("USERPROFILE", profile)
	t.Setenv("DEVRITES_NO_BINARY", "")
	t.Setenv("DEVRITES_ENGINE_CLI", "")
	t.Setenv("DEVRITES_SOURCE_DIR", "")
	t.Setenv("DEVRITES_HOST_ARTIFACT_DIR", "")
	binDir = t.TempDir()
	t.Setenv("DEVRITES_BIN_DIR", binDir)
	installed = filepath.Join(binDir, "devrites-engine.exe")
	data, err := os.ReadFile(engine)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installed, data, 0o755); err != nil {
		t.Fatal(err)
	}
	target = t.TempDir()
	testutil.WriteFile(t, filepath.Join(target, "AGENTS.md"), "user guidance\n")
	testutil.WriteFile(t, filepath.Join(target, "README.txt"), "user project\n")
	testutil.WriteFile(t, filepath.Join(target, ".devrites", "work", "feature", "notes.txt"), "user feature data\n")
	runInstall(t, target, testPayload(t), func(o *Options) { o.WithBinary = false })
	return target, binDir, installed
}

func windowsProjectSnapshot(t *testing.T, target string) map[string]string {
	t.Helper()
	inventory := map[string]string{}
	err := filepath.WalkDir(target, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(target, path)
		if err != nil {
			return err
		}
		inventory[rel] = fmt.Sprint(entry.Type())
		if entry.Type().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			inventory[rel] += ":" + string(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return inventory
}
