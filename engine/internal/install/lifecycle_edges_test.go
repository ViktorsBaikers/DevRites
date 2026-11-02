package install

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/hostpack"
	"github.com/devrites/devrites/internal/testutil"
)

func TestUpdateValidatesSelectedHosts(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	for _, scenario := range []string{"recorded opt-outs", "explicit opt-outs", "explicit missing host"} {
		t.Run(scenario, func(t *testing.T) {
			target := t.TempDir()
			runInstall(t, target, testPayload(t), func(o *Options) {
				o.SourceDir = testSource(t, "1.0.0")
				if scenario != "explicit opt-outs" {
					o.WithCodex, o.WithOmp, o.WithPi, o.WithDevin, o.WithDsh = false, false, false, false, false
				}
			})
			payload := testPayload(t)
			for _, host := range []string{"codex", "omp", "pi", "devin", "dsh"} {
				if err := os.RemoveAll(filepath.Join(payload, host)); err != nil {
					t.Fatal(err)
				}
			}
			testutil.WriteFile(t, filepath.Join(payload, "claude/skills/rite/SKILL.md"), "beta rite\n")
			args := []string{"--target", target, "--source-dir", testSource(t, "2.0.0"), "--payload-dir", payload}
			switch scenario {
			case "explicit opt-outs":
				args = append(args, "--no-codex", "--no-omp", "--no-pi", "--no-devin", "--no-dsh")
			case "explicit missing host":
				args = append(args, "--no-codex=false")
			}
			before := hashTree(t, target)
			var stdout, stderr bytes.Buffer
			code := Run(args, &stdout, &stderr, ModeUpdate)
			if scenario == "explicit missing host" {
				if code != 1 || !strings.Contains(stderr.String(), "missing codex/skills") {
					t.Fatalf("update exit=%d stderr=%q, want enabled host payload refusal", code, stderr.String())
				}
				if hashTree(t, target) != before {
					t.Fatal("missing enabled host payload changed project")
				}
				return
			}
			if code != 0 {
				t.Fatalf("Claude-only update exit=%d stderr=%q", code, stderr.String())
			}
			if got := testutil.ReadFile(t, filepath.Join(target, ".claude/skills/rite/SKILL.md")); got != "beta rite\n" {
				t.Fatalf("updated Claude skill=%q", got)
			}
			if got := manifestHeader(filepath.Join(target, ManifestName), "devrites-version"); got != "2.0.0" {
				t.Fatalf("updated version=%q, want 2.0.0", got)
			}
			for _, host := range []string{".agents", ".omp", ".pi", ".devin", ".dsh"} {
				if exists(filepath.Join(target, host, "skills")) {
					t.Errorf("disabled host skills remain: %s", host)
				}
			}
		})
	}
}

func TestSharedMergeRefusesLateForeignMarker(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	for _, merge := range []hostpack.MarkerMerge{
		hostpack.CodexAgentsMerge, hostpack.PiAgentsMerge,
		hostpack.DevinAgentsMerge, hostpack.DshAgentsMerge, hostpack.CodexConfigMerge,
	} {
		t.Run(merge.MarkerRel, func(t *testing.T) {
			target := t.TempDir()
			original := "alpha guidance\n"
			if merge.TargetRel == "AGENTS.md" {
				for _, block := range []hostpack.MarkerMerge{hostpack.CodexAgentsMerge, hostpack.PiAgentsMerge, hostpack.DevinAgentsMerge, hostpack.DshAgentsMerge} {
					original += "\n" + block.Begin + "\nalpha block\n" + block.End + "\n"
				}
			} else {
				original = "model = \"alpha\"\n"
			}
			shared := filepath.Join(target, filepath.FromSlash(merge.TargetRel))
			marker := filepath.Join(target, filepath.FromSlash(merge.MarkerRel))
			testutil.WriteFile(t, shared, original)
			injected := false
			afterRecheckPath = func(rel string) {
				if rel == merge.TargetRel && !injected {
					injected = true
					testutil.WriteFile(t, marker, "alpha-owned marker\n")
				}
			}
			t.Cleanup(func() { afterRecheckPath = nil })
			opts := DefaultOptions(ModeInstall)
			opts.Target, opts.PayloadDir = target, testPayload(t)
			opts.WithCodex = merge == hostpack.CodexAgentsMerge || merge == hostpack.CodexConfigMerge
			opts.WithOmp, opts.WithPi, opts.WithDevin, opts.WithDsh = false, merge == hostpack.PiAgentsMerge, merge == hostpack.DevinAgentsMerge, merge == hostpack.DshAgentsMerge
			err := Apply(opts)
			afterRecheckPath = nil
			if !injected {
				t.Fatal("fixture missed shared-content recheck")
			}
			if err == nil || !strings.Contains(err.Error(), "without owning "+merge.MarkerRel) {
				t.Errorf("merge error=%v, want marker ownership refusal", err)
			}
			if got := testutil.ReadFile(t, shared); got != original {
				t.Errorf("refused merge changed shared content: %q", got)
			}
			if got := testutil.ReadFile(t, marker); got != "alpha-owned marker\n" {
				t.Errorf("refused merge changed foreign marker: %q", got)
			}
		})
	}
}

func TestSharedMergeRetriesAfterContentFailure(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	for _, merge := range []hostpack.MarkerMerge{hostpack.CodexAgentsMerge, hostpack.CodexConfigMerge} {
		t.Run(merge.MarkerRel, func(t *testing.T) {
			target, payload := t.TempDir(), testPayload(t)
			testutil.WriteFile(t, filepath.Join(target, "AGENTS.md"), "alpha guidance\n")
			testutil.WriteFile(t, filepath.Join(target, ".codex/config.toml"), "model = \"alpha\"\n")
			shared := filepath.Join(target, filepath.FromSlash(merge.TargetRel))
			var beforeFailure string
			injected := false
			afterRecheckPath = func(rel string) {
				if rel != merge.MarkerRel || injected {
					return
				}
				injected = true
				beforeFailure = testutil.ReadFile(t, shared)
				if err := os.Remove(shared); err != nil {
					t.Fatal(err)
				}
				testutil.WriteFile(t, filepath.Join(shared, "alpha"), "beta\n")
			}
			t.Cleanup(func() { afterRecheckPath = nil })
			opts := DefaultOptions(ModeInstall)
			opts.Target, opts.PayloadDir = target, payload
			err := Apply(opts)
			afterRecheckPath = nil
			if !injected {
				t.Fatal("fixture missed marker ownership boundary")
			}
			if err == nil || !strings.Contains(err.Error(), merge.TargetRel) {
				t.Errorf("content failure error=%v, want shared target refusal", err)
			}
			if got := testutil.ReadFile(t, filepath.Join(target, filepath.FromSlash(merge.MarkerRel))); got != merge.MarkerText+"\n" {
				t.Fatalf("failed content merge lost pending ownership: %q", got)
			}
			if exists(filepath.Join(target, ManifestName)) {
				t.Error("failed content merge published manifest")
			}
			if err := os.RemoveAll(shared); err != nil {
				t.Fatal(err)
			}
			testutil.WriteFile(t, shared, beforeFailure)
			if err := Apply(opts); err != nil {
				t.Fatalf("ordinary retry failed: %v", err)
			}
			agents := testutil.ReadFile(t, filepath.Join(target, "AGENTS.md"))
			for _, block := range []hostpack.MarkerMerge{hostpack.CodexAgentsMerge, hostpack.PiAgentsMerge, hostpack.DevinAgentsMerge, hostpack.DshAgentsMerge} {
				if strings.Count(agents, block.Begin) != 1 {
					t.Errorf("retry duplicated or lost %s", block.Begin)
				}
			}
			runUninstall(t, target)
			if got := strings.TrimSpace(testutil.ReadFile(t, filepath.Join(target, "AGENTS.md"))); got != "alpha guidance" {
				t.Errorf("retry/uninstall lost user guidance or left managed blocks: %q", got)
			}
			if got := strings.TrimSpace(testutil.ReadFile(t, filepath.Join(target, ".codex/config.toml"))); got != "model = \"alpha\"" {
				t.Errorf("retry/uninstall lost user config or left permissions: %q", got)
			}
		})
	}
}

func TestUninstallMergeNeverImportsReplacementRoot(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	for _, fixture := range []struct {
		rel, content, cleaned string
	}{
		{"AGENTS.md", "alpha guidance\n", "alpha guidance"},
		{".codex/config.toml", "model = \"alpha\"\n", "model = \"alpha\""},
		{".claude/settings.json", `{"env":{"VALUE":"alpha"}}` + "\n", `{"env":{"VALUE":"alpha"}}`},
		{".codex/hooks.json", `{"env":{"VALUE":"alpha"},"hooks":{"Stop":[{"hooks":[{"type":"command","command":"devrites-engine hook stop-gate"}]}]}}` + "\n", `{"env":{"VALUE":"alpha"}}`},
	} {
		t.Run(fixture.rel, func(t *testing.T) {
			base, outside := t.TempDir(), t.TempDir()
			target, held := filepath.Join(base, "alpha"), filepath.Join(base, "beta")
			testutil.WriteFile(t, filepath.Join(target, filepath.FromSlash(fixture.rel)), fixture.content)
			if fixture.rel == ".codex/hooks.json" {
				seedLegacyCodexHooksInstall(t, target, fixture.content)
			} else {
				runInstall(t, target, testPayload(t), func(*Options) {})
			}
			installed := testutil.ReadFile(t, filepath.Join(target, filepath.FromSlash(fixture.rel)))
			foreign := strings.ReplaceAll(installed, "alpha", "beta-private")
			testutil.WriteFile(t, filepath.Join(outside, filepath.FromSlash(fixture.rel)), foreign)
			attack := newRootReplacementAttack(t, target, held, outside)
			afterRecheckPath = func(rel string) {
				if rel == fixture.rel {
					attack.attempt()
				}
			}
			t.Cleanup(func() { afterRecheckPath = nil })
			opts := DefaultOptions(ModeUninstall)
			opts.Target, opts.KeepBinary = target, true
			err := Apply(opts)
			afterRecheckPath = nil
			attack.assertOutcome(err)
			if got := testutil.ReadFile(t, filepath.Join(outside, filepath.FromSlash(fixture.rel))); got != foreign {
				t.Error("uninstall changed replacement-root content")
			}
			got := testutil.ReadFile(t, filepath.Join(attack.originalPath(), filepath.FromSlash(fixture.rel)))
			if !strings.Contains(got, "alpha") || strings.Contains(got, "beta-private") {
				t.Errorf("uninstall imported replacement-root content into original target: %q", got)
			}
			if attack.blocked {
				if _, err := os.Stat(filepath.Join(target, ManifestName)); !os.IsNotExist(err) {
					t.Fatalf("completed uninstall retained ownership manifest: %v", err)
				}
				if strings.HasSuffix(fixture.rel, ".json") {
					var actual, expected any
					if err := json.Unmarshal([]byte(got), &actual); err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal([]byte(fixture.cleaned), &expected); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(actual, expected) {
						t.Errorf("completed uninstall did not restore user JSON: %q", got)
					}
				} else if strings.TrimSpace(got) != fixture.cleaned {
					t.Errorf("completed uninstall did not restore user content: %q", got)
				}
			} else if got != installed {
				t.Errorf("refused uninstall changed original merge target: %q", got)
			}
		})
	}
}

func TestUninstallBinaryOverrideIsolation(t *testing.T) {
	for _, scenario := range []string{"blocked", "missing", "regular"} {
		t.Run(scenario, func(t *testing.T) {
			t.Setenv("DEVRITES_NO_BINARY", "1")
			t.Setenv("HOME", t.TempDir())
			binDir := t.TempDir()
			t.Setenv("DEVRITES_BIN_DIR", binDir)
			target := t.TempDir()
			runInstall(t, target, testPayload(t), func(*Options) {})
			configured := filepath.Join(binDir, engineBinaryName())
			fallback := filepath.Join(os.Getenv("HOME"), ".local/bin", engineBinaryName())
			testutil.WriteFile(t, fallback, "alpha engine\n")
			switch scenario {
			case "blocked":
				testutil.WriteFile(t, filepath.Join(configured, "alpha"), "beta\n")
			case "regular":
				testutil.WriteFile(t, configured, "beta engine\n")
			}
			t.Setenv("DEVRITES_NO_BINARY", "")
			var stdout, stderr bytes.Buffer
			code := Run([]string{"--target", target}, &stdout, &stderr, ModeUninstall)
			if scenario == "blocked" {
				if code != 1 || !strings.Contains(stderr.String(), "remove engine binary") || !strings.Contains(stderr.String(), configured) {
					t.Errorf("uninstall exit=%d stderr=%q, want configured removal diagnostic", code, stderr.String())
				}
				if strings.Contains(stdout.String(), "DevRites uninstalled") {
					t.Error("failed removal reported uninstall success")
				}
				if got := testutil.ReadFile(t, filepath.Join(configured, "alpha")); got != "beta\n" {
					t.Errorf("failed removal changed configured directory: %q", got)
				}
				if !exists(filepath.Join(target, ManifestName)) {
					t.Error("failed binary removal discarded uninstall ownership manifest")
				}
				if err := os.RemoveAll(configured); err != nil {
					t.Fatal(err)
				}
				testutil.WriteFile(t, configured, "beta engine\n")
				stdout.Reset()
				stderr.Reset()
				if code := Run([]string{"--target", target}, &stdout, &stderr, ModeUninstall); code != 0 {
					t.Errorf("ordinary uninstall retry exit=%d stderr=%q", code, stderr.String())
				}
				if exists(configured) {
					t.Error("ordinary uninstall retry retained configured binary")
				}
				if exists(filepath.Join(target, ManifestName)) {
					t.Error("completed uninstall retry retained ownership manifest")
				}
			} else {
				if code != 0 {
					t.Errorf("uninstall exit=%d stderr=%q", code, stderr.String())
				}
				if exists(configured) {
					t.Error("uninstall retained configured binary")
				}
			}
			if data, err := os.ReadFile(fallback); err != nil || string(data) != "alpha engine\n" {
				t.Errorf("explicit binary override changed fallback installation: data=%q error=%v", data, err)
			}
		})
	}
}
