package install

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"strconv"
	"testing"
	"time"

	"github.com/devrites/devrites/internal/hostpack"
	"github.com/devrites/devrites/internal/release"
	"github.com/devrites/devrites/internal/testutil"
)

func TestInstallDryRunWritesNothing(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	target := t.TempDir()
	var out bytes.Buffer

	opts := DefaultOptions(ModeInstall)
	opts.Target = target
	opts.PayloadDir = payload
	opts.DryRun = true
	opts.Stdout = &out
	opts.Stderr = &bytes.Buffer{}

	if err := Apply(opts); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(target, ".claude")) {
		t.Fatal("dry-run created .claude")
	}
	if !strings.Contains(out.String(), "[install] .claude/skills/rite/SKILL.md") {
		t.Fatalf("dry-run output missing planned install:\n%s", out.String())
	}
	if strings.Contains(out.String(), "Next:") {
		t.Fatalf("dry-run output claims a next move:\n%s", out.String())
	}
}

func TestDshInstallLifecycle(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	target := t.TempDir()
	testutil.WriteFile(t, filepath.Join(target, "AGENTS.md"), "user guidance\n")
	testutil.WriteFile(t, filepath.Join(target, ".dsh", "user.yml"), "user configuration\n")
	runInstall(t, target, payload, func(o *Options) {})
	for _, rel := range []string{".dsh/skills/rite/SKILL.md", ".dsh/agents/devrites-code-reviewer.md", ".dsh/cordis.patch.yml", ".dsh/specialist.mjs"} {
		if !exists(filepath.Join(target, filepath.FromSlash(rel))) {
			t.Fatalf("missing %s", rel)
		}
	}
	runInstall(t, target, payload, func(o *Options) {})
	if text := testutil.ReadFile(t, filepath.Join(target, "AGENTS.md")); strings.Count(text, hostpack.DshAgentsMerge.Begin) != 1 || !strings.Contains(text, "user guidance") {
		t.Fatalf("DSH bridge merge: %s", text)
	}
	runInstall(t, target, payload, func(o *Options) { o.WithDsh = false })
	if exists(filepath.Join(target, ".dsh", "cordis.patch.yml")) {
		t.Fatal("DSH runtime survived opt-out")
	}
	if strings.Contains(testutil.ReadFile(t, filepath.Join(target, "AGENTS.md")), hostpack.DshAgentsMerge.Begin) {
		t.Fatal("DSH bridge survived opt-out")
	}
	if !strings.Contains(testutil.ReadFile(t, filepath.Join(target, ManifestName)), "--no-dsh") {
		t.Fatal("DSH opt-out not persisted")
	}
	runInstall(t, target, payload, func(o *Options) {})
	runUninstall(t, target)
	if exists(filepath.Join(target, ".dsh", "specialist.mjs")) {
		t.Fatal("DSH runtime survived uninstall")
	}
	if got := testutil.ReadFile(t, filepath.Join(target, ".dsh", "user.yml")); got != "user configuration\n" {
		t.Fatal("user DSH config changed")
	}
}

func TestInstallPrintsFirstMoveForInstalledHosts(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")

	for _, tc := range []struct {
		name      string
		withCodex bool
		withOmp   bool
		withPi    bool
		withDevin bool
		want      string
	}{
		{name: "Claude, Codex, and OMP", withCodex: true, withOmp: true, want: "Next: reopen the project, then run /rite (Claude) or $rite (Codex) or /skill:rite (omp/pi)."},
		{name: "Claude and Codex", withCodex: true, withOmp: false, withPi: false, want: "Next: reopen the project, then run /rite (Claude) or $rite (Codex)."},
		{name: "Claude, Codex, and Devin", withCodex: true, withDevin: true, want: "Next: reopen the project, then run /rite (Claude/Devin) or $rite (Codex)."},
		{name: "Claude and OMP", withCodex: false, withOmp: true, withPi: false, want: "Next: reopen the project, then run /rite or /skill:rite."},
		{name: "Claude and pi", withCodex: false, withOmp: false, withPi: true, want: "Next: reopen the project, then run /rite or /skill:rite."},
		{name: "Claude and Devin", withCodex: false, withDevin: true, want: "Next: reopen the project, then run /rite."},
		{name: "Claude only", withCodex: false, withOmp: false, withPi: false, want: "Next: reopen the project, then run /rite."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			opts := DefaultOptions(ModeInstall)
			opts.Target = t.TempDir()
			opts.PayloadDir = testPayload(t)
			opts.WithCodex = tc.withCodex
			opts.WithOmp = tc.withOmp
			opts.WithPi = tc.withPi
			opts.WithDevin = tc.withDevin
			opts.Stdout = &out
			opts.Stderr = &bytes.Buffer{}

			if err := Apply(opts); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("install output missing first move %q:\n%s", tc.want, out.String())
			}
		})
	}
}

func TestInstallManifestConflictAndPrune(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	target := t.TempDir()

	runInstall(t, target, payload, func(o *Options) {})
	testutil.WriteFile(t, filepath.Join(target, ".claude", "skills", "foreign", "SKILL.md"), "mine\n")
	runInstall(t, target, payload, func(o *Options) {})
	if got := testutil.ReadFile(t, filepath.Join(target, ".claude", "skills", "foreign", "SKILL.md")); got != "mine\n" {
		t.Fatalf("unmanaged file clobbered: %q", got)
	}

	stale := filepath.Join(target, ".claude", "skills", "stale", "SKILL.md")
	testutil.WriteFile(t, stale, "stale\n")
	testutil.AppendFile(t, filepath.Join(target, ManifestName), ".claude/skills/stale/SKILL.md\n")
	runInstall(t, target, payload, func(o *Options) { o.Force = true })
	if exists(stale) {
		t.Fatal("managed stale file was not pruned")
	}
	if !exists(filepath.Join(target, ".devrites", "ACTIVE")) {
		t.Fatal("ACTIVE not seeded")
	}
	if strings.Contains(testutil.ReadFile(t, filepath.Join(target, ManifestName)), ".devrites/ACTIVE") {
		t.Fatal("ACTIVE should not be manifest-managed")
	}
}

func TestMarkerMergeAndUninstallPreserveUserContent(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	target := t.TempDir()
	testutil.WriteFile(t, filepath.Join(target, "AGENTS.md"), "user guidance\n")
	testutil.WriteFile(t, filepath.Join(target, ".codex", "config.toml"), "model = \"x\"\n")
	userHooks := `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"echo user"}]}]}}` + "\n"
	testutil.WriteFile(t, filepath.Join(target, ".codex", "hooks.json"), userHooks)

	runInstall(t, target, payload, func(o *Options) {})
	runInstall(t, target, payload, func(o *Options) {})
	agents := testutil.ReadFile(t, filepath.Join(target, "AGENTS.md"))
	if strings.Count(agents, "<!-- BEGIN DEVRITES CODEX -->") != 1 {
		t.Fatalf("AGENTS marker duplicated:\n%s", agents)
	}
	if strings.Count(agents, "<!-- BEGIN DEVRITES DEVIN -->") != 1 {
		t.Fatalf("AGENTS Devin marker missing or duplicated:\n%s", agents)
	}
	if !strings.Contains(agents, "user guidance") {
		t.Fatal("AGENTS user content lost")
	}
	config := testutil.ReadFile(t, filepath.Join(target, ".codex", "config.toml"))
	if strings.Count(config, "# BEGIN DEVRITES CODEX PERMISSIONS") != 1 ||
		!strings.Contains(config, `default_permissions = "devrites-orchestrator"`) ||
		!strings.Contains(config, `model = "x"`) {
		t.Fatalf("config preservation wrong:\n%s", config)
	}
	if hooks := testutil.ReadFile(t, filepath.Join(target, ".codex", "hooks.json")); hooks != userHooks {
		t.Fatalf("installer changed native Codex hooks:\n%s", hooks)
	}

	runUninstall(t, target)
	if got := testutil.ReadFile(t, filepath.Join(target, "AGENTS.md")); !strings.Contains(got, "user guidance") || strings.Contains(got, "DEVRITES CODEX") || strings.Contains(got, "DEVRITES DEVIN") {
		t.Fatalf("AGENTS uninstall preservation wrong:\n%s", got)
	}
	if exists(filepath.Join(target, ".devin")) {
		t.Fatal("uninstall left .devin behind")
	}
	if got := testutil.ReadFile(t, filepath.Join(target, ".codex", "config.toml")); !strings.Contains(got, `model = "x"`) || strings.Contains(got, "DEVRITES CODEX") {
		t.Fatalf("config uninstall preservation wrong:\n%s", got)
	}
	if got := testutil.ReadFile(t, filepath.Join(target, ".codex", "hooks.json")); got != userHooks {
		t.Fatalf("hooks uninstall preservation wrong:\n%s", got)
	}
	if !exists(filepath.Join(target, ".devrites", "ACTIVE")) {
		t.Fatal("uninstall removed ACTIVE")
	}
}

func TestMarkerMergeDryRunReportsUnreadableTarget(t *testing.T) {
	payload := t.TempDir()
	if err := os.WriteFile(filepath.Join(payload, "block.md"), []byte("DevRites\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	if err := os.Mkdir(filepath.Join(target, "AGENTS.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	r := runner{
		opts:      Options{DryRun: true, Stdout: &bytes.Buffer{}},
		target:    target,
		payloadFS: os.DirFS(payload),
	}
	err := r.mergeMarkerFile(hostpack.MarkerMerge{
		TargetRel:  "AGENTS.md",
		PayloadRel: "block.md",
		Begin:      "<!-- BEGIN -->",
		End:        "<!-- END -->",
	})
	if err == nil || !strings.Contains(err.Error(), "cannot read AGENTS.md") {
		t.Fatalf("mergeMarkerFile error = %v, want target read error", err)
	}
}

func TestUninstallReportsUnreadableManifest(t *testing.T) {
	target := t.TempDir()
	if err := os.Mkdir(filepath.Dir(filepath.Join(target, ManifestName)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(target, ManifestName), 0o755); err != nil {
		t.Fatal(err)
	}
	opts := DefaultOptions(ModeUninstall)
	opts.Target = target
	err := Apply(opts)
	if err == nil || !strings.Contains(err.Error(), "read manifest") {
		t.Fatalf("Apply error = %v, want manifest read error", err)
	}
	if info, statErr := os.Stat(filepath.Join(target, ManifestName)); statErr != nil || !info.IsDir() {
		t.Fatalf("manifest was replaced: info=%v err=%v", info, statErr)
	}
}

func TestInstallMergesClaudePermissionsIntoExistingSettings(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	target := t.TempDir()
	testutil.WriteFile(t, filepath.Join(target, ".claude", "settings.json"), `{
  "$comment": "keep my local notes",
  "theme": "dark",
  "statusLine": {"type":"command","command":"DEVRITES_THEME=dark echo user-status"},
  "permissions": {
    "defaultMode": "plan",
    "allow": ["Bash(user-tool *)", "Bash(git checkout *)", "Bash(devrites-engine retired-cmd *)"],
    "ask": ["Bash(rm *)"]
  },
  "hooks": {"Stop":[{"hooks":[
    {"type":"command","command":"echo user-stop"},
    {"type":"command","command":"devrites-engine hook old-gate --harness=claude"}
  ]}]}
}`+"\n")

	var stderr bytes.Buffer
	runInstall(t, target, payload, func(o *Options) { o.Stderr = &stderr })
	runInstall(t, target, payload, func(o *Options) { o.Stderr = &stderr })

	settings := testutil.ReadFile(t, filepath.Join(target, ".claude", "settings.json"))
	for _, preserved := range []string{"keep my local notes", `"theme": "dark"`, "echo user-status", "echo user-stop", "Bash(user-tool *)", "Bash(git checkout *)", "Bash(rm *)"} {
		if !strings.Contains(settings, preserved) {
			t.Fatalf("Claude settings lost user content %q:\n%s", preserved, settings)
		}
	}
	if strings.Count(settings, "Bash(devrites-engine check readiness *)") != 1 {
		t.Fatalf("Claude permission was not merged exactly once:\n%s", settings)
	}
	for _, managed := range []string{"mcp__codegraph__*", "Bash(git diff *)", "Edit(/.devrites/**)"} {
		if !strings.Contains(settings, managed) {
			t.Fatalf("Claude settings merge missed managed rule %q:\n%s", managed, settings)
		}
	}
	if strings.Contains(settings, "retired-cmd") {
		t.Fatalf("Claude settings merge kept a retired engine rule:\n%s", settings)
	}
	if strings.Contains(settings, "devrites-engine hook") {
		t.Fatalf("installer retained a legacy DevRites hook:\n%s", settings)
	}
	manifest := testutil.ReadFile(t, filepath.Join(target, ManifestName))
	if !strings.Contains(manifest, "\n.claude/devrites.claude-hooks-merge\n") {
		t.Fatalf("Claude settings merge marker missing:\n%s", manifest)
	}
	if !strings.Contains(settings, `"defaultMode": "plan"`) {
		t.Fatalf("Claude settings merge dropped the user's own defaultMode:\n%s", settings)
	}

	runUninstall(t, target)
	settings = testutil.ReadFile(t, filepath.Join(target, ".claude", "settings.json"))
	for _, preserved := range []string{"keep my local notes", `"theme": "dark"`, "echo user-status", "echo user-stop", "Bash(user-tool *)", "Bash(git checkout *)", "Bash(rm *)", `"defaultMode": "plan"`} {
		if !strings.Contains(settings, preserved) {
			t.Fatalf("Claude settings uninstall lost user content %q:\n%s", preserved, settings)
		}
	}
	if strings.Contains(settings, "Bash(devrites-engine ") || strings.Contains(settings, "devrites-engine hook") {
		t.Fatalf("Claude settings uninstall left DevRites configuration:\n%s", settings)
	}
	for _, managed := range []string{"mcp__codegraph__*", "Bash(git diff *)", "Edit(/.devrites/**)"} {
		if strings.Contains(settings, managed) {
			t.Fatalf("Claude settings uninstall left managed non-engine rule %q:\n%s", managed, settings)
		}
	}
}

func TestInstallKeepsClaudeMergeOwnershipAcrossReinstall(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	target := t.TempDir()
	settingsPath := filepath.Join(target, ".claude", "settings.json")
	testutil.WriteFile(t, settingsPath, "{}\n")

	runInstall(t, target, payload, func(o *Options) {})
	markerRel := ".claude/devrites.claude-hooks-merge"
	if !exists(filepath.Join(target, filepath.FromSlash(markerRel))) {
		t.Fatal("first install did not create the Claude settings merge marker")
	}
	manifestRecords, err := readManifest(filepath.Join(target, ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := manifestRecords[markerRel]; !ok {
		t.Fatalf("first install did not record the Claude hook merge marker:\n%s", testutil.ReadFile(t, filepath.Join(target, ManifestName)))
	}
	runInstall(t, target, payload, func(o *Options) {})

	settings := testutil.ReadFile(t, settingsPath)
	if strings.Contains(settings, "defaultMode") ||
		strings.Count(settings, "Bash(devrites-engine check readiness *)") != 1 {
		t.Fatalf("Claude permissions lost merge ownership on reinstall (marker exists: %t):\nsettings:\n%s\nmanifest:\n%s",
			exists(filepath.Join(target, filepath.FromSlash(markerRel))), settings, testutil.ReadFile(t, filepath.Join(target, ManifestName)))
	}
	if marker := testutil.ReadFile(t, filepath.Join(target, filepath.FromSlash(markerRel))); strings.Contains(marker, "default-mode=") {
		t.Fatalf("Claude settings marker still records a default mode:\n%s", marker)
	}
	manifest := testutil.ReadFile(t, filepath.Join(target, ManifestName))
	if !strings.Contains(manifest, "\n.claude/devrites.claude-hooks-merge\n") {
		t.Fatalf("Claude settings merge marker missing after reinstall:\n%s", manifest)
	}

	runUninstall(t, target)
	if got := testutil.ReadFile(t, settingsPath); got != "{}\n" {
		t.Fatalf("uninstall did not preserve the pre-existing empty settings file: %q", got)
	}
}

func TestInstallKeepsUserClaudeDefaultMode(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	target := t.TempDir()
	settingsPath := filepath.Join(target, ".claude", "settings.json")
	testutil.WriteFile(t, settingsPath, `{"permissions":{"defaultMode":"acceptEdits"},"theme":"dark"}`+"\n")

	runInstall(t, target, payload, func(o *Options) {})
	if settings := testutil.ReadFile(t, settingsPath); !strings.Contains(settings, `"defaultMode": "acceptEdits"`) || !strings.Contains(settings, "Bash(devrites-engine check readiness *)") {
		t.Fatalf("install did not keep the user's defaultMode beside DevRites rules:\n%s", settings)
	}
	runUninstall(t, target)
	if settings := testutil.ReadFile(t, settingsPath); !strings.Contains(settings, `"defaultMode": "acceptEdits"`) || strings.Contains(settings, "devrites-engine") {
		t.Fatalf("uninstall did not restore the user's settings:\n%s", settings)
	}
}

// Installs before 6.0.5 set plan mode and recorded default-mode=added (or
// =preexisting); update removes only the plan mode DevRites wrote.
func TestUpdateRemovesDevritesOwnedPlanMode(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	for _, tc := range []struct {
		ownership string
		wantPlan  bool
	}{{"added", false}, {"preexisting", true}} {
		t.Run(tc.ownership, func(t *testing.T) {
			target := t.TempDir()
			settingsPath := filepath.Join(target, ".claude", "settings.json")
			seedLegacyClaudeSettingsInstall(t, target, `{"permissions":{"defaultMode":"plan","allow":["Bash(devrites-engine check readiness *)"]}}`+"\n",
				hostpack.ClaudeSettingsMerge.MarkerText+"\ndefault-mode="+tc.ownership+"\nrules=added\n")
			stderr := &bytes.Buffer{}
			runInstall(t, target, payload, func(o *Options) { o.Stderr = stderr })
			settings := testutil.ReadFile(t, settingsPath)
			if got := strings.Contains(settings, `"defaultMode": "plan"`); got != tc.wantPlan {
				t.Fatalf("plan mode present = %t, want %t:\n%s", got, tc.wantPlan, settings)
			}
			// A kept plan mode blocks the root's .devrites writes; say so.
			if got := strings.Contains(stderr.String(), "defaultMode is plan"); got != tc.wantPlan {
				t.Fatalf("plan-mode warning = %t, want %t:\n%s", got, tc.wantPlan, stderr)
			}
		})
	}
}

// A marker written before rules=added/settings=created cannot say who created
// settings.json; once uninstall strips it to nothing, it must not leave "{}".
func TestUninstallAfterLegacyMarkerUpgradeRemovesEmptySettings(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	target := t.TempDir()
	seedLegacyClaudeSettingsInstall(t, target, `{"permissions":{"defaultMode":"plan","allow":["Bash(devrites-engine check readiness *)","mcp__codegraph__*"]}}`+"\n",
		hostpack.ClaudeSettingsMerge.MarkerText+"\ndefault-mode=added\nrule:mcp__codegraph__*")
	runInstall(t, target, payload, func(o *Options) {})
	runUninstall(t, target)
	if path := filepath.Join(target, ".claude", "settings.json"); exists(path) {
		t.Fatalf("uninstall after a legacy-marker upgrade left settings.json: %q", testutil.ReadFile(t, path))
	}
}

func TestGeneratedPayloadInstallsVerbatim(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	target := t.TempDir()
	sentinel := "generated sentinel"
	testutil.WriteFile(t, filepath.Join(payload, "codex", "skills", "rite", "SKILL.md"), sentinel+"\n")
	testutil.WriteFile(t, filepath.Join(payload, "claude", "workflows", "devrites-readonly-review.js"), sentinel+"\n")

	runInstall(t, target, payload, func(o *Options) {})
	if got := testutil.ReadFile(t, filepath.Join(target, ".agents", "skills", "rite", "SKILL.md")); got != sentinel+"\n" {
		t.Fatalf("codex payload was not installed verbatim: %q", got)
	}
	if got := testutil.ReadFile(t, filepath.Join(target, ".claude", "workflows", "devrites-readonly-review.js")); got != sentinel+"\n" {
		t.Fatalf("Claude workflow payload was not installed verbatim: %q", got)
	}
}

func TestInstallBinaryUsesEngineHandoff(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "")
	payload := testPayload(t)
	target := t.TempDir()
	binDir := t.TempDir()
	engine := buildVersionBinary(t, "1.2.3")
	engineBody, err := os.ReadFile(engine)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_ENGINE_CLI", engine)
	t.Setenv("DEVRITES_BIN_DIR", binDir)
	t.Setenv("DEVRITES_REF", "v1.2.3")
	conflictDir := t.TempDir()
	testutil.WriteExecutable(t, filepath.Join(conflictDir, engineBinaryName()), "conflict\n")
	t.Setenv("PATH", conflictDir)

	runInstall(t, target, payload, func(o *Options) {})

	installed := filepath.Join(binDir, engineBinaryName())
	if got, err := os.ReadFile(installed); err != nil || !bytes.Equal(got, engineBody) {
		t.Fatalf("installed binary did not come from DEVRITES_ENGINE_CLI handoff: %v", err)
	}
	info, err := os.Stat(installed)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		t.Fatalf("installed binary is not executable: %v", info.Mode())
	}
}

func TestAcquireBinaryRequiresCompatibleEngineHandoff(t *testing.T) {
	engine := buildVersionBinary(t, "1.2.2")
	t.Setenv("DEVRITES_ENGINE_CLI", engine)

	r := runner{}
	_, cleanup, err := r.acquireBinary("v1.2.3")
	defer cleanup()
	if err == nil || !strings.Contains(err.Error(), "version mismatch") {
		t.Fatalf("acquireBinary error = %v, want handoff version mismatch", err)
	}
}

func TestInstallBinaryRejectsIncompatibleEngineHandoff(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "")
	engine := buildVersionBinary(t, "1.2.2")
	t.Setenv("DEVRITES_ENGINE_CLI", engine)
	t.Setenv("DEVRITES_BIN_DIR", t.TempDir())
	t.Setenv("DEVRITES_REF", "v1.2.3")

	r := runner{opts: DefaultOptions(ModeInstall)}
	if err := r.installBinary(); err == nil || !strings.Contains(err.Error(), "version mismatch") {
		t.Fatalf("installBinary error = %v, want handoff version mismatch", err)
	}
}

func TestInstallBinaryRejectsMissingConfiguredEngineHandoff(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "")
	handoff := filepath.Join(t.TempDir(), "missing-devrites-engine")
	t.Setenv("DEVRITES_ENGINE_CLI", handoff)
	t.Setenv("DEVRITES_BIN_DIR", t.TempDir())
	t.Setenv("DEVRITES_REF", "v1.2.3")

	r := runner{opts: DefaultOptions(ModeInstall)}
	if err := r.installBinary(); err == nil || !strings.Contains(err.Error(), handoff) {
		t.Fatalf("installBinary error = %v, want missing configured handoff error", err)
	}
}

func TestAcquireBinaryRejectsMissingEngineHandoff(t *testing.T) {
	t.Setenv("DEVRITES_ENGINE_CLI", "")

	r := runner{}
	_, cleanup, err := r.acquireBinary("v1.2.3")
	defer cleanup()
	if err == nil || !strings.Contains(err.Error(), "DEVRITES_ENGINE_CLI") {
		t.Fatalf("acquireBinary error = %v, want missing handoff error", err)
	}
}

func TestInstallBinaryFailsWhenPreparedUpdateCannotBeWritten(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "")
	prepared := filepath.Join(t.TempDir(), "devrites-engine")
	testutil.WriteExecutable(t, prepared, "#!/bin/sh\nif [ \"$1\" = version ]; then echo 1.2.3; fi\n")
	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	testutil.WriteFile(t, blocked, "file\n")
	t.Setenv("DEVRITES_BIN_DIR", filepath.Join(blocked, "bin"))
	t.Setenv("DEVRITES_REF", "v1.2.3")

	r := runner{
		opts:           DefaultOptions(ModeInstall),
		preparedBinary: prepared,
	}
	if err := r.installBinary(); err == nil {
		t.Fatal("installBinary accepted a prepared update it could not write")
	}
}

func TestInstallBinaryWithoutBinaryReportsSkip(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "")
	var stdout bytes.Buffer
	opts := DefaultOptions(ModeInstall)
	opts.WithBinary = false
	opts.Stdout = &stdout

	r := runner{opts: opts}
	if err := r.installBinary(); err != nil {
		t.Fatalf("installBinary() error = %v", err)
	}

	output := stdout.String()
	if output != "  engine binary: skipped (--no-binary).\n" {
		t.Fatalf("stdout = %q, want binary skip diagnostic", output)
	}
}

func TestInstallBinaryEnvSkipNamesEnvVar(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	var stdout bytes.Buffer
	opts := DefaultOptions(ModeInstall)
	opts.WithBinary = true
	opts.Stdout = &stdout

	r := runner{opts: opts}
	if err := r.installBinary(); err != nil {
		t.Fatalf("installBinary() error = %v", err)
	}

	if got, want := stdout.String(), "  engine binary: skipped (DEVRITES_NO_BINARY=1).\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestInstallBinaryDoesNotRequirePATH(t *testing.T) {
	payload := testPayload(t)
	target := t.TempDir()
	binDir := t.TempDir()
	engine := buildVersionBinary(t, "1.2.3")
	pathDir := t.TempDir()
	var stderr bytes.Buffer

	t.Setenv("DEVRITES_ENGINE_CLI", engine)
	t.Setenv("DEVRITES_BIN_DIR", binDir)
	t.Setenv("DEVRITES_REF", "v1.2.3")
	t.Setenv("PATH", pathDir)

	runInstall(t, target, payload, func(o *Options) {
		o.Stderr = &stderr
	})

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want no PATH warning", stderr.String())
	}
}

func TestBinaryInstallFailureWarnsAndContinues(t *testing.T) {
	t.Setenv("DEVRITES_ENGINE_CLI", "")
	var stderr bytes.Buffer
	opts := DefaultOptions(ModeInstall)
	opts.Stderr = &stderr

	r := runner{opts: opts}
	if err := r.binaryInstallFailure(fmt.Errorf("handoff unavailable")); err != nil {
		t.Fatalf("binaryInstallFailure() error = %v, want nil", err)
	}

	warning := stderr.String()
	if !strings.Contains(warning, "handoff unavailable") {
		t.Fatalf("warning = %q, want handoff failure", warning)
	}
	if !strings.Contains(warning, "continuing without it") {
		t.Fatalf("warning = %q, want non-blocking fallback", warning)
	}
}

func buildVersionBinary(t *testing.T, version string) string {
	t.Helper()
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	testutil.WriteFile(t, source, fmt.Sprintf("package main\nimport \"fmt\"\nfunc main() { fmt.Println(%q) }\n", version))
	binary := filepath.Join(dir, engineBinaryName())
	if out, err := exec.Command("go", "build", "-o", binary, source).CombinedOutput(); err != nil {
		t.Fatalf("build test engine: %v\n%s", err, out)
	}
	return binary
}

func buildUpdateHandoffBinary(t *testing.T, version string) string {
	t.Helper()
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	testutil.WriteFile(t, source, fmt.Sprintf(`package main
import (
	"fmt"
	"os"
	"strings"
)
func main() {
	if len(os.Args) == 2 && os.Args[1] == "version" {
		fmt.Println(%q)
		return
	}
	output := strings.Join(os.Args[1:], "\n") + "\n" +
		"DEVRITES_UPDATE_HANDOFF=" + os.Getenv("DEVRITES_UPDATE_HANDOFF") + "\n" +
		"DEVRITES_ENGINE_CLI=" + os.Getenv("DEVRITES_ENGINE_CLI") + "\n"
	if err := os.WriteFile(os.Getenv("DEVRITES_TEST_UPDATE_HANDOFF"), []byte(output), 0600); err != nil {
		panic(err)
	}
}
`, version))
	binary := filepath.Join(dir, engineBinaryName())
	if out, err := exec.Command("go", "build", "-o", binary, source).CombinedOutput(); err != nil {
		t.Fatalf("build update handoff engine: %v\n%s", err, out)
	}
	return binary
}

func TestBinaryCandidatesSkipsUnsetBinDir(t *testing.T) {
	t.Setenv("DEVRITES_BIN_DIR", "")

	for _, candidate := range binaryCandidates() {
		if candidate == "devrites-engine" {
			t.Fatalf("unset DEVRITES_BIN_DIR produced relative candidate %q", candidate)
		}
	}
}

func TestUpdateRejectsRemovedAcquisitionFlags(t *testing.T) {
	for _, arg := range []string{"--to=v2.0.0", "--pre"} {
		t.Run(arg, func(t *testing.T) {
			var stderr bytes.Buffer
			if code := Run([]string{arg}, &bytes.Buffer{}, &stderr, ModeUpdate); code != 2 {
				t.Fatalf("Run(%q) code = %d, want 2", arg, code)
			}
			if !strings.Contains(stderr.String(), "flag provided but not defined") {
				t.Fatalf("Run(%q) stderr = %q, want unknown flag error", arg, stderr.String())
			}
		})
	}
}

func TestUpdateInstallsLocalPayload(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	oldPayload := testPayload(t)
	target := t.TempDir()
	oldSource := testSource(t, "1.0.0")
	runInstall(t, target, oldPayload, func(o *Options) {
		o.SourceDir = oldSource
		o.WithAgents = false
	})

	source := testSource(t, "2.0.0")
	payload := filepath.Join(t.TempDir(), "generated")
	writeTestPayload(t, payload)
	testutil.WriteFile(t, filepath.Join(payload, "codex", "skills", "rite", "SKILL.md"), "updated rite\n")

	opts := DefaultOptions(ModeUpdate)
	opts.Target = target
	opts.SourceDir = source
	opts.PayloadDir = payload
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	check := opts
	check.UpdateCheck = true
	check.PayloadDir = ""
	if err := Apply(check); err == nil || !strings.Contains(err.Error(), "update available: 1.0.0 -> 2.0.0") {
		t.Fatalf("update check error = %v, want local candidate report", err)
	}
	if got := manifestHeader(filepath.Join(target, ManifestName), "devrites-version"); got != "1.0.0" {
		t.Fatalf("update check changed manifest version to %q", got)
	}
	if err := Apply(opts); err != nil {
		t.Fatal(err)
	}

	if got := testutil.ReadFile(t, filepath.Join(target, ".agents", "skills", "rite", "SKILL.md")); got != "updated rite\n" {
		t.Fatalf("update did not install local payload: %q", got)
	}
	if got := manifestHeader(filepath.Join(target, ManifestName), "devrites-version"); got != "2.0.0" {
		t.Fatalf("manifest version = %q, want 2.0.0", got)
	}
	if exists(filepath.Join(target, ".claude", "agents")) {
		t.Fatal("update did not replay --no-agents from the manifest")
	}
}

func TestUpdateAcquiresLatestReleaseAndHandsOff(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	target := t.TempDir()
	runInstall(t, target, testPayload(t), func(o *Options) {
		o.SourceDir = testSource(t, "1.0.0")
	})
	source := testSource(t, "2.0.0")
	payload := testPayload(t)
	engine := buildUpdateHandoffBinary(t, "2.0.0")
	marker := filepath.Join(t.TempDir(), "handoff.txt")
	t.Setenv("DEVRITES_TEST_UPDATE_HANDOFF", marker)

	oldResolve, oldAcquire := resolveLatestRelease, acquireRelease
	resolveLatestRelease = func(context.Context, string) (string, error) { return "v2.0.0", nil }
	cleaned := false
	acquireRelease = func(context.Context, string, string) (release.Candidate, func(), error) {
		return release.Candidate{
			SourceDir:  source,
			PayloadDir: payload,
			EnginePath: engine,
			BundleURL:  "https://example.invalid/devrites-v2.0.0.tar.gz",
		}, func() { cleaned = true }, nil
	}
	t.Cleanup(func() {
		resolveLatestRelease, acquireRelease = oldResolve, oldAcquire
	})

	var stdout, stderr bytes.Buffer
	opts := DefaultOptions(ModeUpdate)
	opts.Target = target
	opts.Stdout = &stdout
	opts.Stderr = &stderr
	if err := Apply(opts); err != nil {
		t.Fatalf("remote update: %v\n%s", err, stderr.String())
	}
	if !cleaned {
		t.Fatal("remote update did not clean acquired release")
	}
	handoff := testutil.ReadFile(t, marker)
	for _, want := range []string{
		"update\n",
		"--target\n" + target + "\n",
		"--source-dir\n" + source + "\n",
		"--payload-dir\n" + payload + "\n",
		"DEVRITES_UPDATE_HANDOFF=1\n",
		"DEVRITES_ENGINE_CLI=" + engine + "\n",
	} {
		if !strings.Contains(handoff, want) {
			t.Fatalf("handoff missing %q:\n%s", want, handoff)
		}
	}
	if !strings.Contains(stdout.String(), "latest:    2.0.0") || !strings.Contains(stdout.String(), "bundle:   https://example.invalid/") {
		t.Fatalf("remote update output:\n%s", stdout.String())
	}
}

func TestRemoteUpdateCheckDoesNotDownload(t *testing.T) {
	target := t.TempDir()
	testutil.WriteFile(t, filepath.Join(target, ManifestName), "# devrites-version: 1.0.0\n")
	oldResolve, oldAcquire := resolveLatestRelease, acquireRelease
	resolveLatestRelease = func(context.Context, string) (string, error) { return "v2.0.0", nil }
	acquireRelease = func(context.Context, string, string) (release.Candidate, func(), error) {
		t.Fatal("--check downloaded release assets")
		return release.Candidate{}, func() {}, nil
	}
	t.Cleanup(func() {
		resolveLatestRelease, acquireRelease = oldResolve, oldAcquire
	})
	opts := DefaultOptions(ModeUpdate)
	opts.Target = target
	opts.UpdateCheck = true
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	if err := Apply(opts); err == nil || !strings.Contains(err.Error(), "update available: 1.0.0 -> 2.0.0") {
		t.Fatalf("remote update check error = %v", err)
	}
}

func TestUpdateCleansLegacyCodexHooks(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	source := testSource(t, "4.0.0")

	t.Run("preserves user hooks", func(t *testing.T) {
		target := t.TempDir()
		hooks := `{
  "$comment": "DevRites hooks for Codex. Project hooks load only after trust.",
  "hooks": {"Stop":[{"hooks":[
    {"type":"command","command":"devrites-engine hook stop-gate --harness=codex"},
    {"type":"command","command":"echo user"}
  ]}]}
}` + "\n"
		seedLegacyCodexHooksInstall(t, target, hooks)

		runLocalUpdate(t, target, source, payload)

		if exists(filepath.Join(target, ".claude", "devrites.codex-hooks-merge")) {
			t.Fatal("update kept the legacy Codex hooks marker")
		}
		got := testutil.ReadFile(t, filepath.Join(target, ".codex", "hooks.json"))
		if strings.Contains(got, "devrites-engine hook") || strings.Contains(got, "DevRites hooks") || !strings.Contains(got, "echo user") {
			t.Fatalf("legacy Codex hook cleanup mismatch:\n%s", got)
		}
	})

	t.Run("removes empty hooks file", func(t *testing.T) {
		target := t.TempDir()
		hooks := `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"devrites-engine hook stop-gate --harness=codex"}]}]}}` + "\n"
		seedLegacyCodexHooksInstall(t, target, hooks)

		runLocalUpdate(t, target, source, payload)

		if exists(filepath.Join(target, ".codex", "hooks.json")) {
			t.Fatal("update kept an empty legacy Codex hooks file")
		}
	})

	t.Run("preserves unrecognized user hook shape", func(t *testing.T) {
		target := t.TempDir()
		seedLegacyCodexHooksInstall(t, target, `{"hooks":"user-defined"}`+"\n")

		runLocalUpdate(t, target, source, payload)

		got := testutil.ReadFile(t, filepath.Join(target, ".codex", "hooks.json"))
		if !strings.Contains(got, `"hooks": "user-defined"`) {
			t.Fatalf("legacy cleanup removed unrecognized user hooks: %s", got)
		}
	})

	t.Run("malformed hooks keep ownership marker", func(t *testing.T) {
		target := t.TempDir()
		seedLegacyCodexHooksInstall(t, target, "{\n")

		err := applyUpdate(target, source, payload)
		if err == nil || !strings.Contains(err.Error(), "load hooks config") {
			t.Fatalf("update error = %v, want malformed legacy hooks error", err)
		}
		if !exists(filepath.Join(target, ".claude", "devrites.codex-hooks-merge")) {
			t.Fatal("failed cleanup removed the legacy ownership marker")
		}
		if got := testutil.ReadFile(t, filepath.Join(target, ".codex", "hooks.json")); got != "{\n" {
			t.Fatalf("failed cleanup changed hooks: %q", got)
		}
	})

	t.Run("uninstall preserves user hooks", func(t *testing.T) {
		target := t.TempDir()
		hooks := `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"devrites-engine hook stop-gate"},{"type":"command","command":"echo user"}]}]}}` + "\n"
		seedLegacyCodexHooksInstall(t, target, hooks)

		runUninstall(t, target)

		got := testutil.ReadFile(t, filepath.Join(target, ".codex", "hooks.json"))
		if strings.Contains(got, "devrites-engine hook") || !strings.Contains(got, "echo user") {
			t.Fatalf("legacy Codex hook uninstall mismatch:\n%s", got)
		}
	})
}

func TestLegacyCodexHooksMergeIsCleanupOnly(t *testing.T) {
	legacy := hostpack.LegacyCodexHooksMerge
	merge, ok := hostpack.ManagedMergeForMarker(legacy.MarkerRel)
	if !ok || merge != legacy {
		t.Fatalf("legacy marker lookup = %#v, %t", merge, ok)
	}
	if slices.Contains(hostpack.RequiredPayload(true, true, true, true, true), "codex/hooks.json") {
		t.Fatal("legacy Codex hooks entered the required payload")
	}
	r := runner{opts: DefaultOptions(ModeInstall), payloadFS: os.DirFS(testPayload(t))}
	desired, err := r.desiredInstallPaths()
	if err != nil {
		t.Fatal(err)
	}
	if desired[legacy.MarkerRel] || desired[legacy.TargetRel] || slices.Contains(r.installMergeTargets(), legacy.TargetRel) {
		t.Fatalf("legacy Codex hooks entered active install paths: desired=%v merge-targets=%v", desired, r.installMergeTargets())
	}
}

func TestUpdateRejectsMissingGeneratedPayload(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	oldPayload := testPayload(t)
	target := t.TempDir()
	runInstall(t, target, oldPayload, func(o *Options) { o.SourceDir = testSource(t, "1.0.0") })

	source := testSource(t, "2.1.0")
	payload := filepath.Join(t.TempDir(), "generated")
	builderMarker := filepath.Join(t.TempDir(), "builder-invoked")
	t.Setenv("DEVRITES_TEST_BUILDER_MARKER", builderMarker)
	testutil.WriteExecutable(t, filepath.Join(source, "scripts", "build-host-artifacts.sh"), `#!/usr/bin/env bash
touch "${DEVRITES_TEST_BUILDER_MARKER:?}"
exit 42
`)

	opts := DefaultOptions(ModeUpdate)
	opts.Target = target
	opts.SourceDir = source
	opts.PayloadDir = payload
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	err := Apply(opts)
	if err == nil ||
		!strings.Contains(err.Error(), "local update payload") ||
		!strings.Contains(err.Error(), "missing claude/skills") ||
		!strings.Contains(err.Error(), payload) {
		t.Fatalf("update error = %v, want actionable missing payload error", err)
	}
	if exists(builderMarker) {
		t.Fatal("update invoked the host artifact builder")
	}
	if got := testutil.ReadFile(t, filepath.Join(target, ".agents", "skills", "rite", "SKILL.md")); got != "codex rite\n" {
		t.Fatalf("rejected update changed installed payload: %q", got)
	}
	if got := manifestHeader(filepath.Join(target, ManifestName), "devrites-version"); got != "1.0.0" {
		t.Fatalf("manifest version = %q, want 1.0.0 after rejected update", got)
	}
}

func TestManagedFilePolicy(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	const rel = ".claude/skills/rite/SKILL.md"

	t.Run("refresh preserves customized unless forced", func(t *testing.T) {
		payload := testPayload(t)
		target := t.TempDir()
		runInstall(t, target, payload, func(o *Options) {})
		dest := filepath.Join(target, filepath.FromSlash(rel))
		testutil.WriteFile(t, dest, "customized\n")
		testutil.WriteFile(t, filepath.Join(payload, "claude", "skills", "rite", "SKILL.md"), "new\n")

		err := applyInstall(target, payload, false, false, nil)
		if err == nil || !strings.Contains(err.Error(), "--force") {
			t.Fatalf("customized refresh error = %v, want --force remediation", err)
		}
		if got := testutil.ReadFile(t, dest); got != "customized\n" {
			t.Fatalf("default refresh changed customized file: %q", got)
		}

		var out bytes.Buffer
		if err := applyInstall(target, payload, true, true, &out); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "[overwrite(force-customized)] "+rel) {
			t.Fatalf("forced dry-run did not predict overwrite:\n%s", out.String())
		}
		if got := testutil.ReadFile(t, dest); got != "customized\n" {
			t.Fatalf("dry-run changed customized file: %q", got)
		}

		if err := applyInstall(target, payload, true, false, nil); err != nil {
			t.Fatal(err)
		}
		if got := testutil.ReadFile(t, dest); got != "new\n" {
			t.Fatalf("forced refresh = %q, want new payload", got)
		}
	})

	t.Run("legacy manifest requires force", func(t *testing.T) {
		payload := testPayload(t)
		target := t.TempDir()
		runInstall(t, target, payload, func(o *Options) {})
		manifest := filepath.Join(target, ManifestName)
		lines := strings.Split(testutil.ReadFile(t, manifest), "\n")
		var next []string
		for _, line := range lines {
			if !strings.HasPrefix(line, "# managed: "+rel+" ") {
				next = append(next, line)
			}
		}
		testutil.WriteFile(t, manifest, strings.Join(next, "\n"))

		err := applyInstall(target, payload, false, false, nil)
		if err == nil || !strings.Contains(err.Error(), "legacy manifest entry") {
			t.Fatalf("legacy refresh error = %v", err)
		}
		if err := applyInstall(target, payload, true, false, nil); err != nil {
			t.Fatalf("forced legacy refresh: %v", err)
		}
	})

	t.Run("missing is recreated and absent uninstall is a no-op", func(t *testing.T) {
		payload := testPayload(t)
		target := t.TempDir()
		runInstall(t, target, payload, func(o *Options) {})
		dest := filepath.Join(target, filepath.FromSlash(rel))
		if err := os.Remove(dest); err != nil {
			t.Fatal(err)
		}
		if err := applyInstall(target, payload, false, false, nil); err != nil {
			t.Fatal(err)
		}
		if !exists(dest) {
			t.Fatal("refresh did not recreate missing managed file")
		}
		if err := os.Remove(dest); err != nil {
			t.Fatal(err)
		}
		runUninstall(t, target)
	})

	t.Run("foreign is preserved unless forced", func(t *testing.T) {
		payload := testPayload(t)
		target := t.TempDir()
		dest := filepath.Join(target, filepath.FromSlash(rel))
		testutil.WriteFile(t, dest, "mine\n")
		runInstall(t, target, payload, func(o *Options) {})
		if got := testutil.ReadFile(t, dest); got != "mine\n" {
			t.Fatalf("default install overwrote foreign file: %q", got)
		}
		if err := applyInstall(target, payload, true, false, nil); err != nil {
			t.Fatal(err)
		}
		if got := testutil.ReadFile(t, dest); got != "claude rite\n" {
			t.Fatalf("forced install did not replace foreign file: %q", got)
		}
	})
}

func TestManagedPruneAndUninstallPolicy(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	const staleRel = ".claude/skills/stale/SKILL.md"

	t.Run("prune", func(t *testing.T) {
		payload := testPayload(t)
		target := t.TempDir()
		runInstall(t, target, payload, func(o *Options) {})
		stale := filepath.Join(target, filepath.FromSlash(staleRel))
		testutil.WriteFile(t, stale, "owned\n")
		addManifestRecord(t, filepath.Join(target, ManifestName), staleRel, []byte("owned\n"))
		testutil.WriteFile(t, stale, "customized\n")

		err := applyInstall(target, payload, false, false, nil)
		if err == nil || !strings.Contains(err.Error(), staleRel) || !exists(stale) {
			t.Fatalf("default prune error = %v, exists = %t", err, exists(stale))
		}
		var out bytes.Buffer
		if err := applyInstall(target, payload, true, true, &out); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "[prune(force-customized)] "+staleRel) || !exists(stale) {
			t.Fatalf("forced prune dry-run mismatch:\n%s", out.String())
		}
		if err := applyInstall(target, payload, true, false, nil); err != nil {
			t.Fatal(err)
		}
		if exists(stale) {
			t.Fatal("forced prune kept customized dropped file")
		}
	})

	t.Run("uninstall", func(t *testing.T) {
		payload := testPayload(t)
		target := t.TempDir()
		runInstall(t, target, payload, func(o *Options) {})
		dest := filepath.Join(target, ".claude", "skills", "rite", "SKILL.md")
		testutil.WriteFile(t, dest, "customized\n")

		opts := DefaultOptions(ModeUninstall)
		opts.Target = target
		opts.KeepBinary = true
		opts.Stdout = &bytes.Buffer{}
		opts.Stderr = &bytes.Buffer{}
		err := Apply(opts)
		if err == nil || !strings.Contains(err.Error(), "--force") || !exists(dest) {
			t.Fatalf("default uninstall error = %v, exists = %t", err, exists(dest))
		}

		var out bytes.Buffer
		opts.Force = true
		opts.DryRun = true
		opts.Stdout = &out
		if err := Apply(opts); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "[remove(force-customized)] .claude/skills/rite/SKILL.md") || !exists(dest) {
			t.Fatalf("forced uninstall dry-run mismatch:\n%s", out.String())
		}
		opts.DryRun = false
		if err := Apply(opts); err != nil {
			t.Fatal(err)
		}
		if exists(dest) {
			t.Fatal("forced uninstall kept customized managed file")
		}
	})
}

func TestManagedPathsRejectLinksAndRaces(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows; the same confinement code runs there")
	}
	for _, tc := range []struct {
		name string
		link func(target, outside string) error
	}{
		{
			name: "final symlink",
			link: func(target, outside string) error {
				dest := filepath.Join(target, ".claude", "skills", "rite", "SKILL.md")
				if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
					return err
				}
				return os.Symlink(filepath.Join(outside, "sentinel"), dest)
			},
		},
		{
			name: "ancestor symlink",
			link: func(target, outside string) error {
				return os.Symlink(outside, filepath.Join(target, ".agents"))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := testPayload(t)
			target := t.TempDir()
			outside := t.TempDir()
			testutil.WriteFile(t, filepath.Join(outside, "sentinel"), "outside\n")
			if err := tc.link(target, outside); err != nil {
				t.Fatal(err)
			}
			err := applyInstall(target, payload, true, false, nil)
			if err == nil || !strings.Contains(err.Error(), "refusing") {
				t.Fatalf("linked install error = %v", err)
			}
			if got := testutil.ReadFile(t, filepath.Join(outside, "sentinel")); got != "outside\n" {
				t.Fatalf("linked install changed outside file: %q", got)
			}
		})
	}

	t.Run("recheck detects change", func(t *testing.T) {
		target := t.TempDir()
		const rel = "managed.txt"
		testutil.WriteFile(t, filepath.Join(target, rel), "before\n")
		r := runner{target: target, preflight: map[string]pathSnapshot{}}
		if _, err := r.rememberPath(rel); err != nil {
			t.Fatal(err)
		}
		testutil.WriteFile(t, filepath.Join(target, rel), "after\n")
		if err := r.recheckPath(rel); err == nil || !strings.Contains(err.Error(), "changed after preflight") {
			t.Fatalf("recheck error = %v", err)
		}
	})
}

func TestVerifyEngineBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixtures are Unix-only")
	}
	for _, tc := range []struct {
		name string
		body string
		want string
		ok   bool
	}{
		{name: "exact", body: "echo v1.2.3", want: "1.2.3", ok: true},
		{name: "wrong", body: "echo 1.2.4", want: "1.2.3"},
		{name: "dev", body: "echo dev", want: "1.2.3"},
		{name: "multiline", body: "printf '1.2.3\\nextra\\n'", want: "1.2.3"},
		{name: "empty", body: ":", want: "1.2.3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "engine")
			testutil.WriteExecutable(t, path, "#!/bin/sh\n"+tc.body+"\n")
			_, err := verifyEngineBinary(path, tc.want, 5*time.Second)
			if (err == nil) != tc.ok {
				t.Fatalf("verifyEngineBinary error = %v, ok = %t", err, tc.ok)
			}
		})
	}
	t.Run("timeout", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "engine")
		testutil.WriteExecutable(t, path, "#!/bin/sh\nsleep 2\n")
		_, err := verifyEngineBinary(path, "1.2.3", 20*time.Millisecond)
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Fatalf("timeout error = %v", err)
		}
	})
}

func TestInstallBinaryRollsBackVerificationFailure(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "")
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only; backup/restore portability is covered separately")
	}
	for _, hadOld := range []bool{false, true} {
		t.Run(fmt.Sprintf("had-old-%t", hadOld), func(t *testing.T) {
			binDir := t.TempDir()
			dest := filepath.Join(binDir, "devrites-engine")
			old := "#!/bin/sh\nif [ \"$1\" = version ]; then echo 1.0.0; fi\n"
			if hadOld {
				testutil.WriteExecutable(t, dest, old)
				if err := os.Chmod(dest, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			staged := filepath.Join(t.TempDir(), "devrites-engine")
			body := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = version ]; then case \"$0\" in %q) echo 1.2.4;; *) echo 1.2.3;; esac; fi\n", dest)
			testutil.WriteExecutable(t, staged, body)
			t.Setenv("DEVRITES_BIN_DIR", binDir)
			t.Setenv("DEVRITES_REF", "v1.2.3")
			t.Setenv("PATH", t.TempDir())
			r := runner{opts: DefaultOptions(ModeInstall), preparedBinary: staged}

			err := r.installBinary()
			wantOutcome := "bad binary removed"
			if hadOld {
				wantOutcome = "previous binary restored"
			}
			if err == nil || !strings.Contains(err.Error(), wantOutcome) {
				t.Fatalf("installBinary error = %v", err)
			}
			if hadOld {
				if got := testutil.ReadFile(t, dest); got != old {
					t.Fatalf("rollback bytes = %q, want old binary", got)
				}
				info, statErr := os.Stat(dest)
				if statErr != nil {
					t.Fatal(statErr)
				}
				if info.Mode().Perm() != 0o700 {
					t.Fatalf("rollback mode = %v, want 0700", info.Mode().Perm())
				}
			} else if exists(dest) {
				t.Fatal("failed first install left bad binary")
			}
		})
	}
}

func TestBackupRestoreBinaryIsPlatformSafe(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "devrites-engine")
	testutil.WriteFile(t, dest, "old\n")
	if err := os.Chmod(dest, 0o600); err != nil {
		t.Fatal(err)
	}
	backup, mode, hadOld, err := backupBinary(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(backup) }()
	if filepath.Dir(backup) != filepath.Dir(dest) {
		t.Fatalf("backup %s is not beside %s", backup, dest)
	}
	testutil.WriteFile(t, dest, "bad\n")
	if err := restoreBinary(dest, backup, mode, hadOld); err != nil {
		t.Fatal(err)
	}
	if got := testutil.ReadFile(t, dest); got != "old\n" {
		t.Fatalf("restored bytes = %q", got)
	}
}

func applyInstall(target, payload string, force, dryRun bool, out *bytes.Buffer) error {
	opts := DefaultOptions(ModeInstall)
	opts.Target = target
	opts.PayloadDir = payload
	opts.Force = force
	opts.DryRun = dryRun
	if out == nil {
		out = &bytes.Buffer{}
	}
	opts.Stdout = out
	opts.Stderr = &bytes.Buffer{}
	return Apply(opts)
}

func applyUpdate(target, source, payload string) error {
	opts := DefaultOptions(ModeUpdate)
	opts.Target = target
	opts.SourceDir = source
	opts.PayloadDir = payload
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	return Apply(opts)
}

func runLocalUpdate(t *testing.T, target, source, payload string) {
	t.Helper()
	if err := applyUpdate(target, source, payload); err != nil {
		t.Fatal(err)
	}
}

func seedLegacyCodexHooksInstall(t *testing.T, target, hooks string) {
	t.Helper()
	markerRel := ".claude/devrites.codex-hooks-merge"
	marker := []byte(".codex/hooks.json contains DevRites managed hook entries.\n")
	testutil.WriteFile(t, filepath.Join(target, ".codex", "hooks.json"), hooks)
	testutil.WriteFile(t, filepath.Join(target, filepath.FromSlash(markerRel)), string(marker))
	testutil.WriteFile(t, filepath.Join(target, ManifestName), "# DevRites install manifest - do not edit by hand.\n# devrites-version: 3.2.27\n# devrites-flags: \n")
	addManifestRecord(t, filepath.Join(target, ManifestName), markerRel, marker)
}

func TestIsGlobalTargetResolvesSymlinkedHome(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "home")
	if err := os.Symlink(real, link); err != nil {
		t.Skip(err)
	}
	t.Setenv("HOME", link)
	t.Setenv("USERPROFILE", link) // os.UserHomeDir reads USERPROFILE on Windows
	resolved, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{resolved, filepath.Join(resolved, ".claude"), filepath.Join(resolved, ".codex", "x")} {
		if !isGlobalTarget(target) {
			t.Errorf("isGlobalTarget(%s) = false under symlinked HOME", target)
		}
	}
	if isGlobalTarget(filepath.Join(resolved, "project")) {
		t.Error("a project under HOME was treated as a global agent home")
	}
}

func seedLegacyClaudeSettingsInstall(t *testing.T, target, settings, marker string) {
	t.Helper()
	markerRel := hostpack.ClaudeSettingsMerge.MarkerRel
	testutil.WriteFile(t, filepath.Join(target, ".claude", "settings.json"), settings)
	testutil.WriteFile(t, filepath.Join(target, filepath.FromSlash(markerRel)), marker)
	testutil.WriteFile(t, filepath.Join(target, ManifestName), "# DevRites install manifest - do not edit by hand.\n# devrites-version: 6.0.4\n# devrites-flags: \n")
	addManifestRecord(t, filepath.Join(target, ManifestName), markerRel, []byte(marker))
}

func addManifestRecord(t *testing.T, manifest, rel string, data []byte) {
	t.Helper()
	hash := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	testutil.AppendFile(t, manifest, "# managed: "+rel+" "+hash+"\n"+rel+"\n")
}

func runInstall(t *testing.T, target, payload string, mutate func(*Options)) {
	t.Helper()
	opts := DefaultOptions(ModeInstall)
	opts.Target = target
	opts.PayloadDir = payload
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	mutate(&opts)
	if err := Apply(opts); err != nil {
		t.Fatal(err)
	}
}

func TestUninstallNoBinaryEnvKeepsGlobalBinary(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("DEVRITES_BIN_DIR", t.TempDir())
	bin := filepath.Join(home, ".local", "bin", engineBinaryName())
	testutil.WriteFile(t, bin, "engine\n")
	target := t.TempDir()
	runInstall(t, target, testPayload(t), func(*Options) {})

	opts := DefaultOptions(ModeUninstall)
	opts.Target = target
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	if err := Apply(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("uninstall with DEVRITES_NO_BINARY=1 removed global binary: %v", err)
	}
}

func runUninstall(t *testing.T, target string) {
	t.Helper()
	opts := DefaultOptions(ModeUninstall)
	opts.Target = target
	opts.KeepBinary = true
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	if err := Apply(opts); err != nil {
		t.Fatal(err)
	}
}

func testPayload(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestPayload(t, root)
	return root
}

func writeTestPayload(t *testing.T, root string) {
	t.Helper()
	testutil.WriteFile(t, filepath.Join(root, "claude", "skills", "rite", "SKILL.md"), "claude rite\n")
	testutil.WriteFile(t, filepath.Join(root, "claude", "agents", "devrites-code-reviewer.md"), "agent\n")
	testutil.WriteFile(t, filepath.Join(root, "claude", "workflows", "devrites-readonly-review.js"), "return { read_only: true }\n")
	testutil.WriteFile(t, filepath.Join(root, "claude", "settings.json"), `{
  "permissions": {
    "allow": ["Bash(devrites-engine check readiness *)", "mcp__codegraph__*", "Bash(git diff *)", "Edit(/.devrites/**)"]
  }
}`+"\n")
	testutil.WriteFile(t, filepath.Join(root, "codex", "skills", "rite", "SKILL.md"), "codex rite\n")
	testutil.WriteFile(t, filepath.Join(root, "codex", "agents", "devrites-code-reviewer.toml"), "name = \"devrites-code-reviewer\"\n")
	testutil.WriteFile(t, filepath.Join(root, "codex", "AGENTS.md"), "<!-- BEGIN DEVRITES CODEX -->\nDevRites\n<!-- END DEVRITES CODEX -->\n")
	testutil.WriteFile(t, filepath.Join(root, "codex", "config.toml"), `# BEGIN DEVRITES CODEX PERMISSIONS
default_permissions = "devrites-orchestrator"
[permissions.devrites-orchestrator]
extends = ":workspace"
# END DEVRITES CODEX PERMISSIONS
`)
	testutil.WriteFile(t, filepath.Join(root, "omp", "skills", "rite", "SKILL.md"), "omp rite\n")
	testutil.WriteFile(t, filepath.Join(root, "omp", "agents", "devrites-code-reviewer.md"), "omp agent\n")
	testutil.WriteFile(t, filepath.Join(root, "omp", "commands", "rite.md"), "omp command\n")
	testutil.WriteFile(t, filepath.Join(root, "omp", ".omp-plugin", "plugin.json"), "{}\n")
	testutil.WriteFile(t, filepath.Join(root, "pi", "skills", "rite", "SKILL.md"), "pi rite\n")
	testutil.WriteFile(t, filepath.Join(root, "pi", "agents", "devrites-code-reviewer.md"), "pi agent\n")
	testutil.WriteFile(t, filepath.Join(root, "pi", "prompts", "rite.md"), "pi prompt\n")
	testutil.WriteFile(t, filepath.Join(root, "pi", "AGENTS.md"), "<!-- BEGIN DEVRITES PI -->\nDevRites\n<!-- END DEVRITES PI -->\n")
	testutil.WriteFile(t, filepath.Join(root, "devin", "skills", "rite", "SKILL.md"), "devin rite\n")
	testutil.WriteFile(t, filepath.Join(root, "devin", "agents", "devrites-code-reviewer.md"), "devin agent\n")
	testutil.WriteFile(t, filepath.Join(root, "devin", "AGENTS.md"), "<!-- BEGIN DEVRITES DEVIN -->\nDevRites\n<!-- END DEVRITES DEVIN -->\n")
	testutil.WriteFile(t, filepath.Join(root, "dsh", "skills", "rite", "SKILL.md"), "dsh rite\n")
	testutil.WriteFile(t, filepath.Join(root, "dsh", "agents", "devrites-code-reviewer.md"), "dsh reviewer\n")
	testutil.WriteFile(t, filepath.Join(root, "dsh", "runtime", "cordis.patch.yml"), "[]\n")
	testutil.WriteFile(t, filepath.Join(root, "dsh", "runtime", "specialist.mjs"), "export const inject = [];\n")
	testutil.WriteFile(t, filepath.Join(root, "dsh", "AGENTS.md"), "<!-- BEGIN DEVRITES DSH -->\nDevRites\n<!-- END DEVRITES DSH -->\n")
}

func testSource(t *testing.T, version string) string {
	t.Helper()
	root := t.TempDir()
	testutil.WriteFile(t, filepath.Join(root, "package.json"), `{"version":"`+version+`"}`+"\n")
	return root
}

func manifestEntries(t *testing.T, manifest string) []string {
	t.Helper()
	var out []string
	for _, line := range strings.Split(testutil.ReadFile(t, manifest), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

func slicesContains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func summaryCount(out, key string) (int, error) {
	for _, line := range strings.Split(out, "\n") {
		i := strings.Index(line, key)
		if i < 0 {
			continue
		}
		fields := strings.Fields(line[i+len(key):])
		if len(fields) == 0 {
			continue
		}
		n, err := strconv.Atoi(fields[0])
		if err != nil {
			return 0, fmt.Errorf("parse %s count %q: %w", key, fields[0], err)
		}
		return n, nil
	}
	return 0, fmt.Errorf("install output has no %s counter:\n%s", key, out)
}

func TestInstallAdoptsResidueAfterLostManifest(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	target := t.TempDir()
	runInstall(t, target, payload, func(*Options) {})

	first := manifestEntries(t, filepath.Join(target, ManifestName))
	if len(first) == 0 {
		t.Fatal("first install recorded no managed paths")
	}

	// Exactly the state a run that dies before writeManifest leaves behind.
	if err := os.Remove(filepath.Join(target, ManifestName)); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	opts := DefaultOptions(ModeInstall)
	opts.Target = target
	opts.PayloadDir = payload
	opts.Stdout = &out
	opts.Stderr = &errOut
	if err := Apply(opts); err != nil {
		t.Fatalf("reinstall after manifest loss returned %v; want the tree adopted, not refused\nstdout:\n%s\nstderr:\n%s", err, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "DevRites installed") {
		t.Fatalf("reinstall did not report an install:\n%s", out.String())
	}
	if got := strings.Count(errOut.String(), "not DevRites-managed"); got != 0 {
		t.Fatalf("reinstall rejected %d payload-identical files as foreign:\n%s", got, errOut.String())
	}

	installed, err := summaryCount(out.String(), "installed:")
	if err != nil {
		t.Fatal(err)
	}
	skipped, err := summaryCount(out.String(), "skipped(conflict):")
	if err != nil {
		t.Fatal(err)
	}
	if installed != len(first) || skipped != 0 {
		t.Fatalf("reinstall installed %d of %d payload files and skipped %d; want the whole tree adopted\n%s", installed, len(first), skipped, out.String())
	}

	second := manifestEntries(t, filepath.Join(target, ManifestName))
	if len(second) != len(first) {
		t.Fatalf("reinstall manifest records %d paths, want the %d the first install owned", len(second), len(first))
	}
	for _, rel := range first {
		if !slicesContains(second, rel) {
			t.Fatalf("reinstall manifest dropped %s", rel)
		}
	}

	// The orphan leg: every file the crashed run left on disk must be removable
	// again. This asserts against the first manifest on purpose, so a reinstall
	// that records nothing cannot pass it vacuously.
	opts = DefaultOptions(ModeUninstall)
	opts.Target = target
	opts.KeepBinary = true
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	if err := Apply(opts); err != nil {
		t.Fatalf("uninstall after adopted reinstall returned %v", err)
	}
	for _, rel := range first {
		if exists(filepath.Join(target, filepath.FromSlash(rel))) {
			t.Fatalf("uninstall orphaned %s", rel)
		}
	}
}

func TestReinstallAfterLostManifestSkipsDifferingFile(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	target := t.TempDir()
	runInstall(t, target, payload, func(*Options) {})

	var foreign string
	for _, rel := range manifestEntries(t, filepath.Join(target, ManifestName)) {
		if strings.HasSuffix(rel, "agents/devrites-code-reviewer.md") {
			foreign = rel
		}
	}
	if foreign == "" {
		t.Fatal("first install did not record the agent file")
	}
	path := filepath.Join(target, filepath.FromSlash(foreign))
	if err := os.Remove(filepath.Join(target, ManifestName)); err != nil {
		t.Fatal(err)
	}
	const mine = "user-authored bytes\n"
	testutil.WriteFile(t, path, mine)

	var out, errOut bytes.Buffer
	opts := DefaultOptions(ModeInstall)
	opts.Target = target
	opts.PayloadDir = payload
	opts.Stdout = &out
	opts.Stderr = &errOut
	if err := Apply(opts); err != nil {
		t.Fatalf("reinstall returned %v\n%s", err, errOut.String())
	}

	if got := testutil.ReadFile(t, path); got != mine {
		t.Fatalf("reinstall rewrote a file whose bytes differ from the payload: %q", got)
	}
	if slicesContains(manifestEntries(t, filepath.Join(target, ManifestName)), foreign) {
		t.Fatalf("manifest adopted %s although its bytes differ from the payload", foreign)
	}
	if !strings.Contains(errOut.String(), "skip "+foreign) {
		t.Fatalf("reinstall did not warn about skipping %s:\n%s", foreign, errOut.String())
	}

	opts = DefaultOptions(ModeUninstall)
	opts.Target = target
	opts.KeepBinary = true
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	if err := Apply(opts); err != nil {
		t.Fatalf("uninstall returned %v", err)
	}
	if got := testutil.ReadFile(t, path); got != mine {
		t.Fatalf("uninstall removed or changed the unmanaged file: %q", got)
	}
}

func TestApplyReportsSkippedEngineBinary(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "")
	t.Setenv("DEVRITES_ENGINE_CLI", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("DEVRITES_BIN_DIR", t.TempDir())
	var out, errOut bytes.Buffer
	opts := DefaultOptions(ModeInstall)
	opts.Target = t.TempDir()
	opts.PayloadDir = testPayload(t)
	opts.WithBinary = true
	opts.Stdout = &out
	opts.Stderr = &errOut
	if err := Apply(opts); err != nil {
		t.Fatalf("Apply() error = %v, want nil", err)
	}
	if !strings.Contains(out.String(), "DevRites installed") {
		t.Fatalf("stdout = %q, want install success", out.String())
	}
	if !strings.Contains(out.String(), "engine binary: skipped (no DEVRITES_ENGINE_CLI handoff)") {
		t.Fatalf("stdout = %q, want the skipped engine binary and its reason in the summary", out.String())
	}
}

func TestApplySkippedEngineBinaryKeepsExistingBinary(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "")
	t.Setenv("DEVRITES_ENGINE_CLI", "")
	t.Setenv("HOME", t.TempDir())
	binDir := t.TempDir()
	t.Setenv("DEVRITES_BIN_DIR", binDir)
	old := filepath.Join(binDir, engineBinaryName())
	if err := os.WriteFile(old, []byte("old engine"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	opts := DefaultOptions(ModeInstall)
	opts.Target = t.TempDir()
	opts.PayloadDir = testPayload(t)
	opts.WithBinary = true
	opts.Stdout = &out
	opts.Stderr = &errOut
	if err := Apply(opts); err != nil {
		t.Fatalf("Apply() error = %v, want nil", err)
	}
	if strings.Contains(out.String(), "will not be on PATH") {
		t.Fatalf("stdout = %q, must not claim the command is missing while an older binary is kept", out.String())
	}
	if !strings.Contains(out.String(), "existing devrites-engine binary was kept") {
		t.Fatalf("stdout = %q, want the kept existing binary reported", out.String())
	}
	if got := testutil.ReadFile(t, old); got != "old engine" {
		t.Fatalf("existing binary changed: %q", got)
	}
}

func installForSwap(t *testing.T) (Options, string, string) {
	t.Helper()
	t.Setenv("DEVRITES_NO_BINARY", "1")
	target := filepath.Join(t.TempDir(), "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	opts := DefaultOptions(ModeInstall)
	opts.Target = target
	opts.PayloadDir = testPayload(t)
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	if err := Apply(opts); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(filepath.Dir(target), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	return opts, target, outside
}

// swapDirAfterRecheck replaces dir with a symlink to outside once rel has been
// rechecked, the way a concurrent process could.
func swapDirAfterRecheck(t *testing.T, rel, dir, outside string) *bool {
	t.Helper()
	swapped := new(bool)
	afterRecheckPath = func(r string) {
		if *swapped || r != rel {
			return
		}
		if err := os.Rename(dir, dir+".real"); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, dir); err != nil {
			t.Fatal(err)
		}
		*swapped = true
	}
	t.Cleanup(func() { afterRecheckPath = nil })
	return swapped
}

func TestUninstallStripAfterRecheckStaysInTarget(t *testing.T) {
	opts, target, outside := installForSwap(t)
	cfg := filepath.Join(target, ".codex", "config.toml")
	orig, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	withUser := append([]byte("model = \"user\"\n"), orig...)
	if err := os.WriteFile(cfg, withUser, 0o644); err != nil {
		t.Fatal(err)
	}
	outsideCfg := filepath.Join(outside, "config.toml")
	if err := os.WriteFile(outsideCfg, withUser, 0o600); err != nil {
		t.Fatal(err)
	}
	swapped := swapDirAfterRecheck(t, ".codex/config.toml", filepath.Join(target, ".codex"), outside)
	opts.Mode = ModeUninstall
	_ = Apply(opts)
	if !*swapped {
		t.Fatal("swap never happened")
	}
	after, err := os.ReadFile(outsideCfg)
	if err != nil || !bytes.Equal(after, withUser) {
		t.Fatalf("uninstall rewrote a file outside the target: err=%v content=%q", err, after)
	}
}

func TestUninstallRemoveAfterRecheckStaysInTarget(t *testing.T) {
	opts, target, outside := installForSwap(t)
	victim := filepath.Join(outside, "devrites-code-reviewer.toml")
	if err := os.WriteFile(victim, []byte("precious\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	swapped := swapDirAfterRecheck(t, ".codex/agents/devrites-code-reviewer.toml", filepath.Join(target, ".codex", "agents"), outside)
	opts.Mode = ModeUninstall
	_ = Apply(opts)
	if !*swapped {
		t.Fatal("swap never happened")
	}
	if _, err := os.Stat(victim); err != nil {
		t.Fatalf("uninstall removed a file outside the target: %v", err)
	}
}

func TestUninstallDryRunUnbalancedMarker(t *testing.T) {
	opts, target, _ := installForSwap(t)
	agents := filepath.Join(target, "AGENTS.md")
	data, err := os.ReadFile(agents)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(data), "<!-- END DEVRITES CODEX -->\n", "", 1)
	if broken == string(data) {
		t.Fatalf("no END marker in %q", data)
	}
	if err := os.WriteFile(agents, []byte(broken+"user tail\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts.Mode = ModeUninstall
	dry := opts
	dry.DryRun = true
	dryErr := Apply(dry)
	realErr := Apply(opts)
	if realErr == nil {
		t.Fatal("real uninstall accepted an unterminated marker block")
	}
	if dryErr == nil || dryErr.Error() != realErr.Error() {
		t.Fatalf("dry-run err=%v, real uninstall err=%v; want the same error", dryErr, realErr)
	}
}

func hashTree(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		fmt.Fprintf(h, "%s|%v\n", rel, d.Type())
		if d.Type().IsRegular() {
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			h.Write(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func TestUninstallDryRunMatchesRealMarkerErrors(t *testing.T) {
	const (
		piBegin    = "<!-- BEGIN DEVRITES PI -->\n"
		piEnd      = "<!-- END DEVRITES PI -->\n"
		codexBegin = "<!-- BEGIN DEVRITES CODEX -->\n"
		codexEnd   = "<!-- END DEVRITES CODEX -->\n"
	)
	for _, tc := range []struct {
		name    string
		agents  string
		wantErr bool
	}{
		{name: "balanced", agents: "user\n" + codexBegin + "x\n" + codexEnd + piBegin + "y\n" + piEnd},
		{name: "only blocks", agents: codexBegin + "x\n" + codexEnd},
		{name: "unterminated codex", agents: codexBegin + "x\nuser tail\n", wantErr: true},
		{name: "unterminated pi", agents: piBegin + "x\nuser tail\n", wantErr: true},
		{name: "crossed", agents: "u\n" + piBegin + "A\n" + codexBegin + "B\n" + piEnd + "C\n" + codexEnd + "v\n", wantErr: true},
		{name: "crossed other way", agents: codexBegin + "A\n" + piBegin + "B\n" + codexEnd + "C\n" + piEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts, target, _ := installForSwap(t)
			if err := os.WriteFile(filepath.Join(target, "AGENTS.md"), []byte(tc.agents), 0o644); err != nil {
				t.Fatal(err)
			}
			opts.Mode = ModeUninstall
			dry := opts
			dry.DryRun = true
			before := hashTree(t, target)
			dryErr := Apply(dry)
			if hashTree(t, target) != before {
				t.Fatal("dry-run changed the target tree")
			}
			realErr := Apply(opts)
			if (realErr != nil) != tc.wantErr {
				t.Fatalf("real uninstall err=%v, want error=%v", realErr, tc.wantErr)
			}
			if fmt.Sprint(dryErr) != fmt.Sprint(realErr) {
				t.Fatalf("dry-run err=%v, real uninstall err=%v; want the same error", dryErr, realErr)
			}
		})
	}
}

func TestInstallRejectedMergeWritesNothing(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	for _, tc := range []struct {
		name, rel, data, want string
	}{
		{"claude permissions not an object", ".claude/settings.json", `{"permissions":[]}` + "\n", "must be a JSON object"},
		{"codex default permissions", ".codex/config.toml", "default_permissions = \"mine\"\n", "default_permissions"},
		{"codex default permissions after nested array", ".codex/config.toml", "notify = [\n  [\"a\"],\n]\ndefault_permissions = \"mine\"\n", "default_permissions"},
		{"unterminated AGENTS marker", "AGENTS.md", "user\n<!-- BEGIN DEVRITES PI -->\n", "unterminated"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := t.TempDir()
			testutil.WriteFile(t, filepath.Join(target, "AGENTS.md"), "user guidance\n")
			testutil.WriteFile(t, filepath.Join(target, filepath.FromSlash(tc.rel)), tc.data)
			before := hashTree(t, target)
			err := applyInstall(target, testPayload(t), false, false, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("install error = %v, want %q", err, tc.want)
			}
			if after := hashTree(t, target); after != before {
				t.Fatalf("rejected install changed the target:\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}

// tomlKeyPaths resolves each key line of a simple TOML file to its dotted
// table path, so tests can assert where a key semantically lives.
func tomlKeyPaths(data string) map[string]string {
	out := map[string]string{}
	table := ""
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			table = strings.Trim(line, "[]")
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		path := strings.TrimSpace(key)
		if table != "" {
			path = table + "." + path
		}
		out[path] = strings.TrimSpace(value)
	}
	return out
}

func TestCodexConfigMergeKeepsUserTopLevelKeys(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	for _, tc := range []struct{ table, key, value string }{
		{"profiles.fast", "model", `"mini"`},
		{`mcp_servers."@acme/docs"`, "command", `"npx"`},
		{`profiles."gpt-5:high"`, "model", `"gpt-5"`},
		{`projects."/home/x/repo"`, "trust_level", `"trusted"`}, // no-personal-paths-ignore: fixture
		{`projects.'C:\repo'`, "trust_level", `"trusted"`},
		{`[mcp_servers."a=b,c"]`, "command", `"x"`},
	} {
		t.Run(tc.table, func(t *testing.T) {
			target := t.TempDir()
			configPath := filepath.Join(target, ".codex", "config.toml")
			original := "model = \"o3\"\n\n[" + tc.table + "]\n" + tc.key + " = " + tc.value + "\n"
			testutil.WriteFile(t, configPath, original)

			runInstall(t, target, payload, func(o *Options) {})
			first := testutil.ReadFile(t, configPath)
			runInstall(t, target, payload, func(o *Options) {})
			config := testutil.ReadFile(t, configPath)
			if config != first {
				t.Fatalf("reinstall changed config.toml:\nfirst:\n%s\nsecond:\n%s", first, config)
			}
			keys := tomlKeyPaths(config)
			for path, want := range map[string]string{
				"model": `"o3"`,
				strings.Trim(tc.table, "[]") + "." + tc.key: tc.value,
				"default_permissions":                       `"devrites-orchestrator"`,
				"permissions.devrites-orchestrator.extends": `":workspace"`,
			} {
				if keys[path] != want {
					t.Fatalf("config key %s = %q, want %q:\n%s", path, keys[path], want, config)
				}
			}
			if _, leaked := keys["permissions.devrites-orchestrator.model"]; leaked {
				t.Fatalf("user top-level model moved into the DevRites permissions table:\n%s", config)
			}
			runUninstall(t, target)
			if got := testutil.ReadFile(t, configPath); got != original {
				t.Fatalf("uninstall did not restore config.toml:\nwant %q\ngot  %q", original, got)
			}
		})
	}
}

// Header-looking lines inside multi-line strings or arrays, and a header
// behind a UTF-8 BOM, must not move the DevRites block.
func TestCodexConfigMergeSkipsMultilineValuesAndBOM(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	for _, tc := range []struct{ name, original, header string }{
		{"basic string", "model = \"o3\"\ndeveloper_instructions = \"\"\"\nRules:\n[important]\nbe nice\n\"\"\"\n\n[mcp_servers.a]\ncommand = \"a\"\n", "[mcp_servers.a]\n"},
		{"literal string", "model = \"o3\"\nnote = '''\n[x]\n'''\n[t]\nk = 1\n", "[t]\n"},
		{"array", "model = \"o3\"\nmatrix = [\n  [1, 2],\n\n  [3]\n]\n\n[mcp_servers.a]\ncommand = \"a\"\n", "[mcp_servers.a]\n"},
		{"bom", "\ufeff[model_providers.x]\nname = \"x\"\n\n[profiles.p]\nmodel = \"m\"\n", "[model_providers.x]\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := t.TempDir()
			configPath := filepath.Join(target, ".codex", "config.toml")
			testutil.WriteFile(t, configPath, tc.original)
			runInstall(t, target, payload, func(o *Options) {})
			config := testutil.ReadFile(t, configPath)
			if !strings.Contains(config, "# END DEVRITES CODEX PERMISSIONS\n"+tc.header) {
				t.Fatalf("DevRites block not placed before the first real header %q:\n%s", tc.header, config)
			}
			if strings.Contains(config, "\ufeff") {
				t.Fatalf("BOM left inside config.toml: %q", config)
			}
			runUninstall(t, target)
			if got := testutil.ReadFile(t, configPath); got != strings.TrimPrefix(tc.original, "\ufeff") {
				t.Fatalf("uninstall did not restore config.toml:\nwant %q\ngot  %q", tc.original, got)
			}
		})
	}
}

func TestHasTopLevelTOMLKey(t *testing.T) {
	for _, tc := range []struct {
		data string
		want bool
	}{
		{"default_permissions = \"mine\"\n", true},
		{"\"default_permissions\" = \"mine\"\n", true},
		{"'default_permissions' = \"mine\"\n", true},
		{"\ufeffdefault_permissions = \"mine\"\n", true},
		{"x = \"\"\"\ndefault_permissions = 1\n\"\"\"\n", false},
		{"x = [\n  [1],\n]\ndefault_permissions = \"mine\"\n", true},
		{"[t]\ndefault_permissions = \"mine\"\n", false},
	} {
		if got := hasTopLevelTOMLKey([]byte(tc.data), "default_permissions"); got != tc.want {
			t.Errorf("hasTopLevelTOMLKey(%q) = %t, want %t", tc.data, got, tc.want)
		}
	}
}

func TestInstallPrunesHostsSharingAgentsMD(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	target := t.TempDir()
	runInstall(t, target, payload, func(o *Options) {})
	runInstall(t, target, payload, func(o *Options) {
		o.WithCodex = false
		o.WithPi = false
	})
	agents := testutil.ReadFile(t, filepath.Join(target, "AGENTS.md"))
	if strings.Contains(agents, "DEVRITES CODEX") || strings.Contains(agents, "DEVRITES PI") || !strings.Contains(agents, "DEVRITES DEVIN") {
		t.Fatalf("dropping Codex and pi left the wrong AGENTS.md blocks:\n%s", agents)
	}
}

func TestClaudeSettingsUninstallKeepsPreexistingRule(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	target := t.TempDir()
	settingsPath := filepath.Join(target, ".claude", "settings.json")
	testutil.WriteFile(t, settingsPath, `{"permissions":{"allow":["Bash(git diff *)"]}}`+"\n")

	runInstall(t, target, payload, func(o *Options) {})
	runInstall(t, target, payload, func(o *Options) {})
	if marker := testutil.ReadFile(t, filepath.Join(target, ".claude", "devrites.claude-hooks-merge")); strings.Contains(marker, "rule:Bash(git diff *)") {
		t.Fatalf("settings marker claims the user's pre-existing rule:\n%s", marker)
	}
	runUninstall(t, target)
	settings := testutil.ReadFile(t, settingsPath)
	if !strings.Contains(settings, "Bash(git diff *)") {
		t.Fatalf("uninstall removed the user's pre-existing rule:\n%s", settings)
	}
	if strings.Contains(settings, "mcp__codegraph__*") || strings.Contains(settings, "devrites-engine") {
		t.Fatalf("uninstall left DevRites rules:\n%s", settings)
	}
}

func TestUninstallRemovesClaudeSettingsCreatedByInstall(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	target := t.TempDir()
	runInstall(t, target, testPayload(t), func(o *Options) {})
	runInstall(t, target, testPayload(t), func(o *Options) {})
	runUninstall(t, target)
	if exists(filepath.Join(target, ".claude", "settings.json")) {
		t.Fatalf("uninstall left the DevRites-created settings file: %q", testutil.ReadFile(t, filepath.Join(target, ".claude", "settings.json")))
	}
}

func TestUpdateHonorsExplicitCLIFlags(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	t.Setenv("DEVRITES_ENGINE_CLI", "")
	t.Setenv("DEVRITES_UPDATE_HANDOFF", "")
	payload := testPayload(t)
	target := t.TempDir()
	runInstall(t, target, payload, func(o *Options) {})
	t.Setenv("DEVRITES_NO_BINARY", "")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--target", target, "--source-dir", testSource(t, "9.9.9"), "--payload-dir", payload, "--no-binary", "--no-omp", "--force"}, &stdout, &stderr, ModeUpdate)
	if code != 0 {
		t.Fatalf("update exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "engine binary: skipped (--no-binary)") {
		t.Fatalf("update ignored --no-binary:\n%s", stdout.String())
	}
	if exists(filepath.Join(target, ".omp", "skills")) {
		t.Fatal("update ignored --no-omp")
	}
	if flags := manifestHeader(filepath.Join(target, ManifestName), "devrites-flags"); !strings.Contains(flags, "--no-omp") || !strings.Contains(flags, "--no-binary") {
		t.Fatalf("update did not persist explicit flags: %q", flags)
	}
}

func TestUninstallNoBinaryFlagKeepsGlobalBinary(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	binDir := t.TempDir()
	t.Setenv("DEVRITES_BIN_DIR", binDir)
	bin := filepath.Join(binDir, engineBinaryName())
	testutil.WriteFile(t, bin, "engine\n")
	target := t.TempDir()
	runInstall(t, target, testPayload(t), func(*Options) {})
	t.Setenv("DEVRITES_NO_BINARY", "")

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--target", target, "--no-binary"}, &stdout, &stderr, ModeUninstall); code != 0 {
		t.Fatalf("uninstall exit %d: %s", code, stderr.String())
	}
	if !exists(bin) {
		t.Fatalf("uninstall --no-binary removed the global binary:\n%s", stdout.String())
	}
}

func TestPinnedAliasSurvivesInstallAndUninstalls(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil || runtime.GOOS == "windows" {
		t.Skip("pin.sh needs bash")
	}
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	testutil.WriteFile(t, filepath.Join(payload, "claude", "skills", "rite-build", "SKILL.md"), "build\n")
	target := t.TempDir()
	runInstall(t, target, payload, func(o *Options) {})
	pin := exec.Command("bash", filepath.Join("..", "..", "..", "scripts", "pin.sh"), "--target", target, "add", "b", "rite-build")
	pin.Env = append(os.Environ(), "HOME="+t.TempDir())
	if out, err := pin.CombinedOutput(); err != nil {
		t.Fatalf("pin.sh add: %v\n%s", err, out)
	}
	records, err := readManifest(filepath.Join(target, ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{".claude/skills/b/SKILL.md", ".agents/skills/b/SKILL.md"} {
		snap, err := inspectManagedPath(target, rel)
		if err != nil || records[rel].Hash != snap.hash {
			t.Fatalf("pin.sh recorded %s hash %q, file hash %q (%v)", rel, records[rel].Hash, snap.hash, err)
		}
	}
	// A pin recorded by an older pin.sh carries no hash.
	legacy := filepath.Join(target, ".claude", "skills", "old", "SKILL.md")
	testutil.WriteFile(t, legacy, "---\nname: old\ndescription: Alias of DevRites /rite. Use when the user runs /old. Delegates to /rite.\n---\n")
	testutil.AppendFile(t, filepath.Join(target, ManifestName), ".claude/skills/old/SKILL.md\n")

	if err := applyInstall(target, payload, false, false, nil); err != nil {
		t.Fatalf("install after pin: %v", err)
	}
	for _, rel := range []string{".claude/skills/b/SKILL.md", ".agents/skills/b/SKILL.md", ".claude/skills/old/SKILL.md"} {
		if !exists(filepath.Join(target, filepath.FromSlash(rel))) {
			t.Fatalf("install pruned pinned alias %s", rel)
		}
	}
	runUninstall(t, target)
	for _, rel := range []string{".claude/skills/b/SKILL.md", ".agents/skills/b/SKILL.md", ".claude/skills/old/SKILL.md"} {
		if exists(filepath.Join(target, filepath.FromSlash(rel))) {
			t.Fatalf("uninstall left pinned alias %s", rel)
		}
	}
}

func TestInstallManagesHostConfigs(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	for _, tc := range []struct {
		config hostpack.HostConfig
		drop   func(*Options)
	}{
		{hostpack.OmpConfig, func(o *Options) { o.WithOmp = false }},
		{hostpack.PiSettings, func(o *Options) { o.WithCodex = false }},
	} {
		t.Run(tc.config.Host+" absent", func(t *testing.T) {
			target := t.TempDir()
			configPath := filepath.Join(target, filepath.FromSlash(tc.config.Rel))
			runInstall(t, target, payload, func(o *Options) {})
			if got := testutil.ReadFile(t, configPath); got != tc.config.Content {
				t.Fatalf("%s = %q", tc.config.Rel, got)
			}
			if records, _ := readManifest(filepath.Join(target, ManifestName)); records[tc.config.Rel].Hash == "" {
				t.Fatalf("%s not hash-managed", tc.config.Rel)
			}
			runInstall(t, target, payload, tc.drop)
			if exists(configPath) {
				t.Fatalf("dropping the host kept managed %s", tc.config.Rel)
			}
			runInstall(t, target, payload, func(o *Options) {})
			runUninstall(t, target)
			if exists(configPath) {
				t.Fatalf("uninstall kept managed %s", tc.config.Rel)
			}
		})

		t.Run(tc.config.Host+" user owned", func(t *testing.T) {
			target := t.TempDir()
			configPath := filepath.Join(target, filepath.FromSlash(tc.config.Rel))
			testutil.WriteFile(t, configPath, "mine\n")
			var stderr bytes.Buffer
			runInstall(t, target, payload, func(o *Options) { o.Stderr = &stderr; o.Force = true })
			if got := testutil.ReadFile(t, configPath); got != "mine\n" {
				t.Fatalf("installer edited a user-owned %s: %q", tc.config.Rel, got)
			}
			if !strings.Contains(stderr.String(), "add: "+tc.config.Advice+"\n") {
				t.Fatalf("missing %s advice:\n%s", tc.config.Rel, stderr.String())
			}
			runUninstall(t, target)
			if got := testutil.ReadFile(t, configPath); got != "mine\n" {
				t.Fatalf("uninstall touched a user-owned %s: %q", tc.config.Rel, got)
			}
		})
	}
}

func TestEditedHostConfigIsRelinquished(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	t.Setenv("DEVRITES_UPDATE_HANDOFF", "")
	payload := testPayload(t)
	ops := map[string]func(t *testing.T, target string, force bool, drop func(*Options)) string{
		"update": func(t *testing.T, target string, force bool, _ func(*Options)) string {
			args := []string{"--target", target, "--source-dir", testSource(t, "9.9.9"), "--payload-dir", payload, "--no-binary"}
			if force {
				args = append(args, "--force")
			}
			var stdout, stderr bytes.Buffer
			if code := Run(args, &stdout, &stderr, ModeUpdate); code != 0 {
				t.Fatalf("update exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
			}
			return stderr.String()
		},
		"host drop": func(t *testing.T, target string, force bool, drop func(*Options)) string {
			var stderr bytes.Buffer
			runInstall(t, target, payload, func(o *Options) { drop(o); o.Force = force; o.Stderr = &stderr })
			return stderr.String()
		},
		"uninstall": func(t *testing.T, target string, force bool, _ func(*Options)) string {
			opts := DefaultOptions(ModeUninstall)
			opts.Target = target
			opts.KeepBinary = true
			opts.Force = force
			opts.Stdout = &bytes.Buffer{}
			var stderr bytes.Buffer
			opts.Stderr = &stderr
			if err := Apply(opts); err != nil {
				t.Fatal(err)
			}
			return stderr.String()
		},
	}
	for _, tc := range []struct {
		config hostpack.HostConfig
		drop   func(*Options)
	}{
		{hostpack.OmpConfig, func(o *Options) { o.WithOmp = false }},
		{hostpack.PiSettings, func(o *Options) { o.WithCodex = false }},
	} {
		for name, op := range ops {
			for _, force := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s %s force=%v", tc.config.Host, name, force), func(t *testing.T) {
					target := t.TempDir()
					configPath := filepath.Join(target, filepath.FromSlash(tc.config.Rel))
					runInstall(t, target, payload, func(o *Options) {})
					testutil.WriteFile(t, configPath, "mine\n")
					stderr := op(t, target, force, tc.drop)
					if got := testutil.ReadFile(t, configPath); got != "mine\n" {
						t.Fatalf("%s changed the user-edited %s: %q", name, tc.config.Rel, got)
					}
					if name == "update" && !strings.Contains(stderr, "add: "+tc.config.Advice+"\n") {
						t.Fatalf("missing %s advice:\n%s", tc.config.Rel, stderr)
					}
					if mf := filepath.Join(target, ManifestName); exists(mf) && slices.Contains(manifestEntries(t, mf), tc.config.Rel) {
						t.Fatalf("%s still lists the user-edited %s", ManifestName, tc.config.Rel)
					}
					runInstall(t, target, payload, func(o *Options) {})
					runUninstall(t, target)
					if got := testutil.ReadFile(t, configPath); got != "mine\n" {
						t.Fatalf("later install/uninstall changed the user-edited %s: %q", tc.config.Rel, got)
					}
				})
			}
		}
	}
}
