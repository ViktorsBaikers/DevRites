package install

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/devrites/devrites/internal/hostpack"
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

func TestInstallPrintsFirstMoveForInstalledHosts(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")

	for _, tc := range []struct {
		name      string
		withCodex bool
		want      string
	}{
		{name: "Claude and Codex", withCodex: true, want: "Next: reopen the project, then run /rite (Claude) or $rite (Codex)."},
		{name: "Claude only", withCodex: false, want: "Next: reopen the project, then run /rite."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			opts := DefaultOptions(ModeInstall)
			opts.Target = t.TempDir()
			opts.PayloadDir = testPayload(t)
			opts.WithCodex = tc.withCodex
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
	if got := testutil.ReadFile(t, filepath.Join(target, "AGENTS.md")); !strings.Contains(got, "user guidance") || strings.Contains(got, "DEVRITES CODEX") {
		t.Fatalf("AGENTS uninstall preservation wrong:\n%s", got)
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
    "allow": ["Bash(user-tool *)"],
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
	for _, preserved := range []string{"keep my local notes", `"theme": "dark"`, "echo user-status", "echo user-stop", "Bash(user-tool *)", "Bash(rm *)"} {
		if !strings.Contains(settings, preserved) {
			t.Fatalf("Claude settings lost user content %q:\n%s", preserved, settings)
		}
	}
	if strings.Count(settings, "Bash(devrites-engine check readiness *)") != 1 {
		t.Fatalf("Claude permission was not merged exactly once:\n%s", settings)
	}
	if strings.Contains(settings, "devrites-engine hook") {
		t.Fatalf("installer retained a legacy DevRites hook:\n%s", settings)
	}
	manifest := testutil.ReadFile(t, filepath.Join(target, ManifestName))
	if !strings.Contains(manifest, "\n.claude/devrites.claude-hooks-merge\n") {
		t.Fatalf("Claude settings merge marker missing:\n%s", manifest)
	}
	marker := testutil.ReadFile(t, filepath.Join(target, ".claude", "devrites.claude-hooks-merge"))
	if !strings.Contains(marker, "default-mode=preexisting") {
		t.Fatalf("Claude settings marker lost pre-existing plan-mode ownership:\n%s", marker)
	}

	runUninstall(t, target)
	settings = testutil.ReadFile(t, filepath.Join(target, ".claude", "settings.json"))
	for _, preserved := range []string{"keep my local notes", `"theme": "dark"`, "echo user-status", "echo user-stop", "Bash(user-tool *)", "Bash(rm *)", `"defaultMode": "plan"`} {
		if !strings.Contains(settings, preserved) {
			t.Fatalf("Claude settings uninstall lost user content %q:\n%s", preserved, settings)
		}
	}
	if strings.Contains(settings, "Bash(devrites-engine ") || strings.Contains(settings, "devrites-engine hook") {
		t.Fatalf("Claude settings uninstall left DevRites configuration:\n%s", settings)
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
	if strings.Count(settings, `"defaultMode": "plan"`) != 1 ||
		strings.Count(settings, "Bash(devrites-engine check readiness *)") != 1 {
		t.Fatalf("Claude permissions lost merge ownership on reinstall (marker exists: %t):\nsettings:\n%s\nmanifest:\n%s",
			exists(filepath.Join(target, filepath.FromSlash(markerRel))), settings, testutil.ReadFile(t, filepath.Join(target, ManifestName)))
	}
	if marker := testutil.ReadFile(t, filepath.Join(target, filepath.FromSlash(markerRel))); !strings.Contains(marker, "default-mode=added") {
		t.Fatalf("Claude settings marker did not record owned plan mode:\n%s", marker)
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

func TestInstallRejectsConflictingClaudeDefaultMode(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	target := t.TempDir()
	settingsPath := filepath.Join(target, ".claude", "settings.json")
	original := `{"permissions":{"defaultMode":"acceptEdits"},"theme":"dark"}` + "\n"
	testutil.WriteFile(t, settingsPath, original)

	err := applyInstall(target, payload, false, false, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "requires plan mode") {
		t.Fatalf("install error = %v, want conflicting default-mode rejection", err)
	}
	if got := testutil.ReadFile(t, settingsPath); got != original {
		t.Fatalf("failed install changed Claude settings:\n%s", got)
	}
	if exists(filepath.Join(target, ".claude", "devrites.claude-hooks-merge")) {
		t.Fatal("failed install created a Claude settings marker")
	}
}

func TestGeneratedPayloadInstallsVerbatim(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	target := t.TempDir()
	sentinel := "generated sentinel"
	testutil.WriteFile(t, filepath.Join(payload, "codex", "skills", "rite", "SKILL.md"), sentinel+"\n")

	runInstall(t, target, payload, func(o *Options) {})
	if got := testutil.ReadFile(t, filepath.Join(target, ".agents", "skills", "rite", "SKILL.md")); got != sentinel+"\n" {
		t.Fatalf("codex payload was not installed verbatim: %q", got)
	}
}

func TestInstallBinaryUsesEngineHandoff(t *testing.T) {
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
			_, err := verifyEngineBinary(path, tc.want, time.Second)
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
	defer os.Remove(backup)
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
	testutil.WriteFile(t, filepath.Join(root, "claude", "settings.json"), `{
  "permissions": {
    "defaultMode": "plan",
    "allow": ["Bash(devrites-engine check readiness *)"]
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
}

func testSource(t *testing.T, version string) string {
	t.Helper()
	root := t.TempDir()
	testutil.WriteFile(t, filepath.Join(root, "package.json"), `{"version":"`+version+`"}`+"\n")
	return root
}
