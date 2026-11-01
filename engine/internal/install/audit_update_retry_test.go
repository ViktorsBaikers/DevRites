package install

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

func TestAuditUpdateRetriesAfterSelectedBinaryFailure(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	t.Setenv("DEVRITES_REF", "")
	target := t.TempDir()
	payload := testPayload(t)
	runInstall(t, target, payload, func(o *Options) {
		o.SourceDir = testSource(t, "1.0.0")
		o.WithAgents = false
	})
	manifest := filepath.Join(target, ManifestName)
	before := testutil.ReadFile(t, manifest)
	binDir := filepath.Join(t.TempDir(), "bin")
	testutil.WriteFile(t, binDir, "blocked destination\n")
	t.Setenv("DEVRITES_BIN_DIR", binDir)
	t.Setenv("DEVRITES_NO_BINARY", "")
	engine := buildVersionBinary(t, "2.0.0")
	t.Setenv("DEVRITES_ENGINE_CLI", engine)
	testutil.WriteFile(t, filepath.Join(payload, "codex", "skills", "rite", "SKILL.md"), "updated rite\n")

	opts := DefaultOptions(ModeUpdate)
	opts.Target = target
	opts.SourceDir = testSource(t, "2.0.0")
	opts.PayloadDir = payload
	if err := Apply(opts); err == nil || !strings.Contains(err.Error(), "install engine binary") {
		t.Fatalf("update error = %v, want selected binary installation failure", err)
	}
	if got := testutil.ReadFile(t, manifest); got != before {
		t.Error("failed binary installation published incoming manifest")
	}
	if got := testutil.ReadFile(t, filepath.Join(target, ".agents", "skills", "rite", "SKILL.md")); got != "codex rite\n" {
		t.Errorf("failed binary installation changed managed payload: %q", got)
	}
	if err := os.Remove(binDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Apply(opts); err != nil {
		t.Fatalf("ordinary same-candidate retry: %v", err)
	}
	installed, err := os.ReadFile(filepath.Join(binDir, engineBinaryName()))
	if err != nil {
		t.Fatalf("retry did not install selected engine: %v", err)
	}
	incoming, err := os.ReadFile(engine)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(installed, incoming) {
		t.Fatal("retry installed different engine bytes")
	}
	if got := manifestHeader(manifest, "devrites-version"); got != "2.0.0" {
		t.Fatalf("completed retry version = %q, want 2.0.0", got)
	}
	if got := testutil.ReadFile(t, filepath.Join(target, ".agents", "skills", "rite", "SKILL.md")); got != "updated rite\n" {
		t.Fatalf("retry payload = %q, want updated rite", got)
	}
	if exists(filepath.Join(target, ".claude", "agents")) || !strings.Contains(manifestHeader(manifest, "devrites-flags"), "--no-agents") {
		t.Fatal("retry lost selected --no-agents install flag")
	}
	// A completed install must still short-circuit without requiring staging.
	opts.PayloadDir = ""
	t.Setenv("DEVRITES_ENGINE_CLI", "")
	if err := Apply(opts); err != nil {
		t.Fatalf("completed same-version update: %v", err)
	}
}
