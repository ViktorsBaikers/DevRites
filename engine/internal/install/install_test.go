package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
}

func TestExtractTarGzRejectsTooManyEntries(t *testing.T) {
	tarball := filepath.Join(t.TempDir(), "many.tar.gz")
	f, err := os.Create(tarball)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for i := 0; i <= maxUpdateEntries; i++ {
		hdr := &tar.Header{Name: fmt.Sprintf("entry-%05d", i), Typeflag: tar.TypeDir, Mode: 0o755}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	err = extractTarGz(tarball, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "too many entries") {
		t.Fatalf("extractTarGz error = %v, want entry-count rejection", err)
	}
}

func FuzzExtractTarGzPaths(f *testing.F) {
	f.Add("safe/file.txt", []byte("content"))
	f.Add("../escape", []byte("blocked"))
	f.Add("/absolute", []byte("blocked"))
	f.Fuzz(func(t *testing.T, name string, content []byte) {
		if len(name) > 256 || len(content) > 4096 || strings.ContainsRune(name, '\x00') {
			t.Skip()
		}
		base := t.TempDir()
		tarball := filepath.Join(base, "input.tar.gz")
		file, err := os.Create(tarball)
		if err != nil {
			t.Fatal(err)
		}
		gz := gzip.NewWriter(file)
		tw := tar.NewWriter(gz)
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
			_ = tw.Close()
			_ = gz.Close()
			_ = file.Close()
			return
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatal(err)
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		dest := filepath.Join(base, "dest")
		if err := os.Mkdir(dest, 0o755); err != nil {
			t.Fatal(err)
		}
		_ = extractTarGz(tarball, dest)
		if _, err := os.Stat(filepath.Join(base, "escape")); !os.IsNotExist(err) {
			t.Fatalf("archive escaped destination through %q", name)
		}
	})
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
	testutil.WriteFile(t, filepath.Join(target, ".codex", "hooks.json"), `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"echo user"}]}]}}`+"\n")

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
	if strings.Contains(config, "DEVRITES CODEX") || !strings.Contains(config, `model = "x"`) {
		t.Fatalf("config preservation wrong:\n%s", config)
	}
	hooks := testutil.ReadFile(t, filepath.Join(target, ".codex", "hooks.json"))
	if strings.Count(hooks, "devrites-engine hook stop-gate") != 1 || !strings.Contains(hooks, "echo user") {
		t.Fatalf("hooks merge wrong:\n%s", hooks)
	}

	runUninstall(t, target)
	if got := testutil.ReadFile(t, filepath.Join(target, "AGENTS.md")); !strings.Contains(got, "user guidance") || strings.Contains(got, "DEVRITES CODEX") {
		t.Fatalf("AGENTS uninstall preservation wrong:\n%s", got)
	}
	if got := testutil.ReadFile(t, filepath.Join(target, ".codex", "config.toml")); !strings.Contains(got, `model = "x"`) || strings.Contains(got, "DEVRITES CODEX") {
		t.Fatalf("config uninstall preservation wrong:\n%s", got)
	}
	if got := testutil.ReadFile(t, filepath.Join(target, ".codex", "hooks.json")); !strings.Contains(got, "echo user") || strings.Contains(got, "devrites-engine hook") {
		t.Fatalf("hooks uninstall preservation wrong:\n%s", got)
	}
	if !exists(filepath.Join(target, ".devrites", "ACTIVE")) {
		t.Fatal("uninstall removed ACTIVE")
	}
}

func TestInstallMergesClaudeHooksIntoExistingSettings(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	testutil.WriteFile(t, filepath.Join(payload, "claude", "settings.json"), `{
  "statusLine": {"type":"command","command":"devrites-engine hook statusline --harness=claude"},
  "hooks": {
    "Stop": [{"hooks":[{"type":"command","command":"devrites-engine hook stop-gate --harness=claude"}]}],
    "SessionStart": [{"hooks":[{"type":"command","command":"devrites-engine hook orient --harness=claude"}]}]
  }
}`+"\n")
	target := t.TempDir()
	testutil.WriteFile(t, filepath.Join(target, ".claude", "settings.json"), `{
  "$comment": "DevRites hooks — keep my local notes",
  "theme": "dark",
  "statusLine": {"type":"command","command":"DEVRITES_THEME=dark echo user-status"},
  "hooks": {"Stop":[{"hooks":[
    {"type":"command","command":"echo user-stop"},
    {"type":"command","command":"devrites-engine hook old-gate --harness=claude"}
  ]}]}
}`+"\n")

	var stderr bytes.Buffer
	runInstall(t, target, payload, func(o *Options) { o.Stderr = &stderr })
	runInstall(t, target, payload, func(o *Options) { o.Stderr = &stderr })

	settings := testutil.ReadFile(t, filepath.Join(target, ".claude", "settings.json"))
	for _, preserved := range []string{"DevRites hooks — keep my local notes", `"theme": "dark"`, "echo user-status", "echo user-stop"} {
		if !strings.Contains(settings, preserved) {
			t.Fatalf("Claude settings lost user content %q:\n%s", preserved, settings)
		}
	}
	for _, command := range []string{"devrites-engine hook stop-gate", "devrites-engine hook orient"} {
		if strings.Count(settings, command) != 1 {
			t.Fatalf("Claude hook %q was not merged exactly once:\n%s", command, settings)
		}
	}
	if !strings.Contains(stderr.String(), "preserved existing Claude statusLine") {
		t.Fatalf("missing statusLine conflict warning:\n%s", stderr.String())
	}
	manifest := testutil.ReadFile(t, filepath.Join(target, ManifestName))
	if !strings.Contains(manifest, "\n.claude/devrites.claude-hooks-merge\n") {
		t.Fatalf("Claude hook merge marker missing:\n%s", manifest)
	}

	runUninstall(t, target)
	settings = testutil.ReadFile(t, filepath.Join(target, ".claude", "settings.json"))
	for _, preserved := range []string{"DevRites hooks — keep my local notes", `"theme": "dark"`, "echo user-status", "echo user-stop"} {
		if !strings.Contains(settings, preserved) {
			t.Fatalf("Claude settings uninstall lost user content %q:\n%s", preserved, settings)
		}
	}
	if strings.Contains(settings, "devrites-engine hook") {
		t.Fatalf("Claude settings uninstall left DevRites hooks:\n%s", settings)
	}
}

func TestInstallKeepsClaudeMergeOwnershipAcrossReinstall(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	testutil.WriteFile(t, filepath.Join(payload, "claude", "settings.json"), `{
  "hooks": {"Stop":[{"hooks":[{"type":"command","command":"devrites-engine hook stop-gate --harness=claude"}]}]}
}`+"\n")
	target := t.TempDir()
	settingsPath := filepath.Join(target, ".claude", "settings.json")
	testutil.WriteFile(t, settingsPath, "{}\n")

	runInstall(t, target, payload, func(o *Options) {})
	markerRel := ".claude/devrites.claude-hooks-merge"
	if !exists(filepath.Join(target, filepath.FromSlash(markerRel))) {
		t.Fatal("first install did not create the Claude hook merge marker")
	}
	if !readManifest(filepath.Join(target, ManifestName))[markerRel] {
		t.Fatalf("first install did not record the Claude hook merge marker:\n%s", testutil.ReadFile(t, filepath.Join(target, ManifestName)))
	}
	runInstall(t, target, payload, func(o *Options) {})

	settings := testutil.ReadFile(t, settingsPath)
	if strings.Count(settings, "devrites-engine hook stop-gate") != 1 {
		t.Fatalf("Claude hooks lost merge ownership on reinstall (marker exists: %t):\nsettings:\n%s\nmanifest:\n%s",
			exists(filepath.Join(target, filepath.FromSlash(markerRel))), settings, testutil.ReadFile(t, filepath.Join(target, ManifestName)))
	}
	manifest := testutil.ReadFile(t, filepath.Join(target, ManifestName))
	if !strings.Contains(manifest, "\n.claude/devrites.claude-hooks-merge\n") {
		t.Fatalf("Claude hook merge marker missing after reinstall:\n%s", manifest)
	}

	runUninstall(t, target)
	if got := testutil.ReadFile(t, settingsPath); got != "{}\n" {
		t.Fatalf("uninstall did not preserve the pre-existing empty settings file: %q", got)
	}
}

func TestInstallRefreshesSeededClaudeHooksWithoutManagingSettings(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	settingsPayload := filepath.Join(payload, "claude", "settings.json")
	testutil.WriteFile(t, settingsPayload, `{
  "$comment": "DevRites hooks",
  "hooks": {"Stop":[{"hooks":[{"type":"command","command":"bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/devrites-stop-gate.sh\""}]}]}
}`+"\n")
	target := t.TempDir()

	runInstall(t, target, payload, func(o *Options) {})
	testutil.WriteFile(t, settingsPayload, `{
  "$comment": "DevRites hooks — auto-approve the read-only orientation/gate scripts.",
  "hooks": {"Stop":[{"hooks":[{"type":"command","command":"devrites-engine hook new-gate --harness=claude"}]}]}
}`+"\n")
	runInstall(t, target, payload, func(o *Options) {})

	settings := testutil.ReadFile(t, filepath.Join(target, ".claude", "settings.json"))
	if strings.Contains(settings, ".claude/hooks/devrites-") || strings.Count(settings, "new-gate") != 1 {
		t.Fatalf("seeded Claude hooks were not refreshed:\n%s", settings)
	}
	manifest := testutil.ReadFile(t, filepath.Join(target, ManifestName))
	if strings.Contains(manifest, ".claude/devrites.claude-hooks-merge") {
		t.Fatalf("seeded Claude settings became merge-managed:\n%s", manifest)
	}

	runUninstall(t, target)
	settings = testutil.ReadFile(t, filepath.Join(target, ".claude", "settings.json"))
	if !strings.Contains(settings, "new-gate") {
		t.Fatalf("uninstall removed refreshed seeded Claude settings:\n%s", settings)
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
	engine := filepath.Join(t.TempDir(), "devrites-engine")
	engineBody := "#!/bin/sh\nif [ \"$1\" = version ]; then echo 1.2.3; exit 0; fi\necho handoff\n"
	testutil.WriteExecutable(t, engine, engineBody)
	t.Setenv("DEVRITES_ENGINE_CLI", engine)
	t.Setenv("DEVRITES_BIN_DIR", binDir)
	t.Setenv("DEVRITES_REF", "v1.2.3")

	runInstall(t, target, payload, func(o *Options) {})

	installed := filepath.Join(binDir, "devrites-engine")
	if got := testutil.ReadFile(t, installed); got != engineBody {
		t.Fatalf("installed binary did not come from DEVRITES_ENGINE_CLI handoff:\n%s", got)
	}
	info, err := os.Stat(installed)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("installed binary is not executable: %v", info.Mode())
	}
}

func TestInstallBinaryWarnsWhenHooksCannotResolveEngine(t *testing.T) {
	payload := testPayload(t)
	target := t.TempDir()
	binDir := t.TempDir()
	engine := filepath.Join(t.TempDir(), "devrites-engine")
	testutil.WriteExecutable(t, engine, "#!/bin/sh\nif [ \"$1\" = version ]; then echo 1.2.3; exit 0; fi\n")
	pathDir := t.TempDir()
	var stderr bytes.Buffer

	t.Setenv("DEVRITES_ENGINE_CLI", engine)
	t.Setenv("DEVRITES_BIN_DIR", binDir)
	t.Setenv("DEVRITES_REF", "v1.2.3")
	t.Setenv("PATH", pathDir)

	runInstall(t, target, payload, func(o *Options) {
		o.Stderr = &stderr
	})

	if !strings.Contains(stderr.String(), "not on PATH") {
		t.Fatalf("missing PATH reachability warning:\n%s", stderr.String())
	}
}

func TestBinaryCandidatesSkipsUnsetBinDir(t *testing.T) {
	t.Setenv("DEVRITES_BIN_DIR", "")

	for _, candidate := range binaryCandidates() {
		if candidate == "devrites-engine" {
			t.Fatalf("unset DEVRITES_BIN_DIR produced relative candidate %q", candidate)
		}
	}
}

func TestUpdateInstallsRequestedBundle(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	oldPayload := testPayload(t)
	target := t.TempDir()
	oldSource := testSource(t, "1.0.0")
	runInstall(t, target, oldPayload, func(o *Options) { o.SourceDir = oldSource })

	bundleRoot := t.TempDir()
	bundle := filepath.Join(bundleRoot, "devrites-v2.0.0")
	testutil.WriteFile(t, filepath.Join(bundle, "install.sh"), "#!/usr/bin/env bash\n")
	testutil.WriteFile(t, filepath.Join(bundle, "package.json"), `{"version":"2.0.0"}`+"\n")
	payload := filepath.Join(bundle, "pack", "generated")
	writeTestPayload(t, payload)
	testutil.WriteFile(t, filepath.Join(payload, "codex", "skills", "rite", "SKILL.md"), "updated rite\n")
	t.Setenv("DEVRITES_UPDATE_BUNDLE", tarGzDir(t, bundleRoot))

	opts := DefaultOptions(ModeUpdate)
	opts.Target = target
	opts.UpdateTo = "v2.0.0"
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	if err := Apply(opts); err != nil {
		t.Fatal(err)
	}

	if got := testutil.ReadFile(t, filepath.Join(target, ".agents", "skills", "rite", "SKILL.md")); got != "updated rite\n" {
		t.Fatalf("update did not install requested bundle payload: %q", got)
	}
	if got := manifestHeader(filepath.Join(target, ManifestName), "devrites-version"); got != "2.0.0" {
		t.Fatalf("manifest version = %q, want 2.0.0", got)
	}
}

func TestUpdateBuildsGeneratedPayloadForSourceArchive(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	oldPayload := testPayload(t)
	target := t.TempDir()
	runInstall(t, target, oldPayload, func(o *Options) { o.SourceDir = testSource(t, "1.0.0") })

	bundleRoot := t.TempDir()
	bundle := filepath.Join(bundleRoot, "devrites-v2.1.0")
	testutil.WriteFile(t, filepath.Join(bundle, "install.sh"), "#!/usr/bin/env bash\n")
	testutil.WriteFile(t, filepath.Join(bundle, "package.json"), `{"version":"2.1.0"}`+"\n")
	testutil.WriteExecutable(t, filepath.Join(bundle, "scripts", "build-host-artifacts.sh"), `#!/usr/bin/env bash
set -eu
out="${DEVRITES_HOST_ARTIFACT_DIR:?}"
mkdir -p "$out/claude/skills/rite" "$out/claude/agents" "$out/codex/skills/rite" "$out/codex/agents"
printf 'source archive claude\n' > "$out/claude/skills/rite/SKILL.md"
printf 'agent\n' > "$out/claude/agents/devrites-code-reviewer.md"
printf '{}\n' > "$out/claude/settings.json"
printf 'source archive codex\n' > "$out/codex/skills/rite/SKILL.md"
printf 'name = "devrites-code-reviewer"\n' > "$out/codex/agents/devrites-code-reviewer.toml"
printf '<!-- BEGIN DEVRITES CODEX -->\nDevRites\n<!-- END DEVRITES CODEX -->\n' > "$out/codex/AGENTS.md"
printf '{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"devrites-engine hook stop-gate"}]}]}}\n' > "$out/codex/hooks.json"
`)
	t.Setenv("DEVRITES_UPDATE_BUNDLE", tarGzDir(t, bundleRoot))

	opts := DefaultOptions(ModeUpdate)
	opts.Target = target
	opts.UpdateTo = "v2.1.0"
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	if err := Apply(opts); err != nil {
		t.Fatal(err)
	}

	if got := testutil.ReadFile(t, filepath.Join(target, ".agents", "skills", "rite", "SKILL.md")); got != "source archive codex\n" {
		t.Fatalf("source archive update did not build generated payload: %q", got)
	}
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
	testutil.WriteFile(t, filepath.Join(root, "claude", "settings.json"), "{}\n")
	testutil.WriteFile(t, filepath.Join(root, "codex", "skills", "rite", "SKILL.md"), "codex rite\n")
	testutil.WriteFile(t, filepath.Join(root, "codex", "agents", "devrites-code-reviewer.toml"), "name = \"devrites-code-reviewer\"\n")
	testutil.WriteFile(t, filepath.Join(root, "codex", "AGENTS.md"), "<!-- BEGIN DEVRITES CODEX -->\nDevRites\n<!-- END DEVRITES CODEX -->\n")
	hooks := map[string]any{"hooks": map[string]any{"Stop": []any{map[string]any{"hooks": []any{map[string]any{"type": "command", "command": "devrites-engine hook stop-gate"}}}}}}
	data, err := json.MarshalIndent(hooks, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, filepath.Join(root, "codex", "hooks.json"), string(data)+"\n")
}

func testSource(t *testing.T, version string) string {
	t.Helper()
	root := t.TempDir()
	testutil.WriteFile(t, filepath.Join(root, "package.json"), `{"version":"`+version+`"}`+"\n")
	return root
}

func tarGzDir(t *testing.T, root string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = tw.Write(data)
		return err
	})
	if closeErr := tw.Close(); err == nil {
		err = closeErr
	}
	if closeErr := gz.Close(); err == nil {
		err = closeErr
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		t.Fatal(err)
	}
	return out
}
