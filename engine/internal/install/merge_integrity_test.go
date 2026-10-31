package install

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/hostpack"
	"github.com/devrites/devrites/internal/testutil"
)

func TestConfigMergesPreservePrivatePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file permissions")
	}
	t.Setenv("DEVRITES_NO_BINARY", "1")
	target, payload, source := t.TempDir(), testPayload(t), testSource(t, "7.2.7")
	configs := map[string]string{
		".claude/settings.json": `{"env":{"FIXTURE_VALUE":"private"}}` + "\n",
		".codex/config.toml":    "model = \"user-model\"\n",
		"AGENTS.md":             "private project guidance\n",
	}
	for rel, body := range configs {
		path := filepath.Join(target, filepath.FromSlash(rel))
		testutil.WriteFile(t, path, body)
		if err := os.Chmod(path, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, stage := range []string{"install", "reinstall", "update", "uninstall"} {
		switch stage {
		case "install", "reinstall":
			runInstall(t, target, payload, func(o *Options) { o.SourceDir = source })
		case "update":
			runLocalUpdate(t, target, testSource(t, "7.2.8"), payload)
		case "uninstall":
			runUninstall(t, target)
		}
		for rel := range configs {
			info, err := os.Stat(filepath.Join(target, filepath.FromSlash(rel)))
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != 0o600 {
				t.Errorf("%s %s mode = %04o, want 0600", stage, rel, got)
			}
		}
	}
}

func TestRejectedJSONUnmergesWriteNothing(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		for _, mode := range []Mode{ModeUninstall, ModeInstall, ModeUpdate} {
			for _, dryRun := range []bool{true, false} {
				t.Run(fmt.Sprintf("legacy=%t/%s/dry=%t", legacy, mode, dryRun), func(t *testing.T) {
					t.Setenv("DEVRITES_NO_BINARY", "1")
					target, payload := t.TempDir(), testPayload(t)
					if legacy {
						seedLegacyCodexHooksInstall(t, target, "{}\n")
						testutil.WriteFile(t, filepath.Join(target, ".codex/hooks.json"), `{"hooks":`)
					} else {
						runInstall(t, target, payload, func(*Options) {})
						testutil.WriteFile(t, filepath.Join(target, ".claude/settings.json"), `{"permissions":`)
					}
					before := hashTree(t, target)
					opts := DefaultOptions(mode)
					opts.Target, opts.PayloadDir = target, payload
					opts.SourceDir = testSource(t, "99.0.0")
					opts.WithSkills, opts.WithAgents, opts.DryRun = false, false, dryRun
					opts.args = []string{"--no-skills", "--no-agents"}
					err := Apply(opts)
					if err == nil || !strings.Contains(err.Error(), "decode JSON") {
						t.Fatalf("error = %v, want invalid JSON refusal", err)
					}
					if hashTree(t, target) != before {
						t.Fatal("rejected JSON unmerge changed the project")
					}
				})
			}
		}
	}
}

func TestInterruptedSettingsMergeRetainsOwnership(t *testing.T) {
	for _, boundary := range []struct {
		name, rel  string
		occurrence int
		updateOnly bool
	}{
		{"before ownership", hostpack.ClaudeSettingsMerge.MarkerRel, 1, false},
		{"before settings", hostpack.ClaudeSettingsMerge.TargetRel, 1, false},
		{"after ownership", hostpack.ClaudeSettingsMerge.TargetRel, 2, false},
		{"before finalization", hostpack.ClaudeSettingsMerge.MarkerRel, 2, true},
	} {
		for _, updating := range []bool{false, true} {
			if boundary.updateOnly && !updating {
				continue
			}
			t.Run(fmt.Sprintf("%s/update=%t", boundary.name, updating), func(t *testing.T) {
				t.Setenv("DEVRITES_NO_BINARY", "1")
				target, payload := t.TempDir(), testPayload(t)
				if updating {
					runInstall(t, target, payload, func(*Options) {})
					testutil.WriteFile(t, filepath.Join(payload, "claude/settings.json"),
						`{"permissions":{"allow":["Bash(devrites-engine check readiness *)","mcp__new_fixture__*"]}}`+"\n")
				}
				const interrupted = "settings merge interrupted"
				didInterrupt := false
				t.Cleanup(func() { afterRecheckPath = nil })
				seen := 0
				afterRecheckPath = func(rel string) {
					if rel == boundary.rel {
						seen++
						if seen == boundary.occurrence {
							panic(interrupted)
						}
					}
				}
				func() {
					defer func() {
						if p := recover(); p == interrupted {
							didInterrupt = true
						} else if p != nil {
							panic(p)
						}
					}()
					runInstall(t, target, payload, func(*Options) {})
				}()
				afterRecheckPath = nil
				if !didInterrupt {
					t.Fatal("fixture did not reach settings merge boundary")
				}
				runInstall(t, target, payload, func(o *Options) { o.Force = updating })
				runUninstall(t, target)
				if _, err := os.Stat(filepath.Join(target, ".claude/settings.json")); !os.IsNotExist(err) {
					t.Fatalf("installer-owned permissions survived retry and uninstall: %v", err)
				}
			})
		}
	}
}

func TestUnownedSettingsMarkerRejectsBeforeWrites(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	target, payload := t.TempDir(), testPayload(t)
	testutil.WriteFile(t, filepath.Join(target, hostpack.ClaudeSettingsMerge.MarkerRel), "user-owned marker\n")
	before := hashTree(t, target)
	for _, dryRun := range []bool{true, false} {
		err := applyInstall(target, payload, false, dryRun, nil)
		if err == nil || !strings.Contains(err.Error(), "without owning") {
			t.Fatalf("dry=%t: error=%v, want marker ownership refusal", dryRun, err)
		}
		if hashTree(t, target) != before {
			t.Fatalf("dry=%t: rejected marker ownership changed the project", dryRun)
		}
	}
}
