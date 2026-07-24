package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/rootfacts"
	"github.com/devrites/devrites/internal/version"
)

// withBinaryVersion temporarily overrides the compiled-in binary version so skew
// verdicts can be exercised deterministically.
func withBinaryVersion(t *testing.T, v string) {
	t.Helper()
	prev := version.Version
	version.Version = v
	t.Cleanup(func() { version.Version = prev })
}

// writeFeature creates a minimal feature.md declaring a schemaVersion under root.
func writeFeature(t *testing.T, root, slug string, schema int) {
	t.Helper()
	dir := filepath.Join(root, "features", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nslug: " + slug + "\nphase: build\nschemaVersion: " +
		itoa(schema) + "\n---\n\nx\n"
	if err := os.WriteFile(filepath.Join(dir, "feature.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func TestDiagnoseOKWhenAligned(t *testing.T) {
	withBinaryVersion(t, "2.6.1")
	root := t.TempDir()
	writeFeature(t, root, "f", 1) // state v1, binary supports v1
	r, err := Diagnose(t.TempDir(), root)
	if err != nil {
		t.Fatal(err)
	}
	if r.Refuse {
		t.Errorf("Refuse = true, want false when aligned")
	}
	if got := r.Verdict; got == "" || got[:2] != "ok" {
		t.Errorf("verdict = %q, want an ok verdict", got)
	}
}

func TestDiagnoseWarnsWhenBinaryOlderThanPack(t *testing.T) {
	withBinaryVersion(t, "1.0.0")
	projectDir := t.TempDir()
	// Installed pack newer than the binary → warn, not refuse.
	claude := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claude, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "# devrites-version: 2.0.0\n.claude/skills/rite/SKILL.md\tabc\n"
	if err := os.WriteFile(filepath.Join(claude, "devrites.manifest"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeFeature(t, root, "f", 1)

	r, err := Diagnose(projectDir, root)
	if err != nil {
		t.Fatal(err)
	}
	if r.Refuse {
		t.Errorf("a pack-vs-binary skew must warn, not refuse")
	}
	if r.Pack != "2.0.0" {
		t.Errorf("pack = %q, want 2.0.0", r.Pack)
	}
	if want := "WARN"; len(r.Verdict) < 4 || r.Verdict[:4] != want {
		t.Errorf("verdict = %q, want a WARN verdict", r.Verdict)
	}
}

func TestPackVersionUsesManifestBeforeLegacyMarker(t *testing.T) {
	projectDir := t.TempDir()
	claude := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claude, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claude, "devrites.manifest"), []byte("# devrites-version: 3.2.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claude, "devrites.version"), []byte("2.9.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := packVersion(projectDir); got != "3.2.0" {
		t.Fatalf("packVersion() = %q, want installer manifest version 3.2.0", got)
	}
}

func TestPackVersionIgnoresProjectPackageJSON(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(`{"version":"99.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := packVersion(projectDir); got != Unknown {
		t.Fatalf("packVersion() = %q, want %q without DevRites install provenance", got, Unknown)
	}
}

func TestDiagnoseRefusesNewerStateSchema(t *testing.T) {
	withBinaryVersion(t, "2.6.1")
	root := t.TempDir()
	writeFeature(t, root, "f", 99) // far newer than the binary supports
	r, err := Diagnose(t.TempDir(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Refuse {
		t.Errorf("Refuse = false, want true for a newer-major state schema")
	}
	if len(r.Verdict) < 6 || r.Verdict[:6] != "REFUSE" {
		t.Errorf("verdict = %q, want a REFUSE verdict", r.Verdict)
	}
}

func TestDiagnoseDevBinaryNeverFalseWarns(t *testing.T) {
	withBinaryVersion(t, "dev") // incomparable
	projectDir := t.TempDir()
	claude := filepath.Join(projectDir, ".claude")
	_ = os.MkdirAll(claude, 0o755)
	_ = os.WriteFile(filepath.Join(claude, "devrites.version"), []byte("9.9.9\n"), 0o644)
	root := t.TempDir()
	writeFeature(t, root, "f", 1)

	r, err := Diagnose(projectDir, root)
	if err != nil {
		t.Fatal(err)
	}
	if r.Refuse || (len(r.Verdict) >= 4 && r.Verdict[:4] == "WARN") {
		t.Errorf("a dev (incomparable) binary must not warn on skew; verdict=%q", r.Verdict)
	}
}

func TestDiagnoseReportsHostArtifactDrift(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".claude", "skills", "rite-build"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".claude", "settings.json"), []byte(`{"hooks":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeFeature(t, root, "f", 1)

	r, err := Diagnose(projectDir, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Checks) == 0 {
		t.Fatal("expected drift checks")
	}
	out := r.Render()
	for _, want := range []string{"devrites-engine hooks", "Codex skill mirror", "devrites.manifest provenance"} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, out)
		}
	}
}

func TestDiagnoseFactsReportsCanonicalRootAndStaleActive(t *testing.T) {
	projectDir := t.TempDir()
	root := filepath.Join(projectDir, ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte("ghost\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	facts, err := rootfacts.ResolveFrom(projectDir, projectDir)
	if err != nil {
		t.Fatal(err)
	}
	r, err := DiagnoseFacts(facts)
	if err != nil {
		t.Fatal(err)
	}
	out := r.Render()
	for _, want := range []string{
		"root: " + facts.PhysicalRoot,
		"root-selection: DEVRITES_ROOT",
		"[DRV-ACTIVE-STALE]",
		"fix: rm -f",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, out)
		}
	}
	if r.Refuse {
		t.Fatal("stale ACTIVE should warn, not turn a read-only doctor run into a refusal")
	}
}

func TestDiagnoseReportsCanonicalGeneratedResidue(t *testing.T) {
	projectDir := t.TempDir()
	canonical := filepath.Join(projectDir, "pack", ".claude")
	generated := filepath.Join(projectDir, "pack", "generated", "claude")
	for _, dir := range []string{canonical, generated} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{
		filepath.Join(canonical, "settings.json"),
		filepath.Join(generated, "settings.json"),
	} {
		if err := os.WriteFile(path, []byte(`{"hooks":{}}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	root := t.TempDir()
	writeFeature(t, root, "f", 1)
	aligned, err := Diagnose(projectDir, root)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(aligned.Render(), "DRV-GENERATED-DRIFT") {
		t.Fatalf("aligned generated tree reported drift:\n%s", aligned.Render())
	}

	if err := os.WriteFile(filepath.Join(generated, "legacy.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	drifted, err := Diagnose(projectDir, root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(drifted.Render(), "[DRV-GENERATED-DRIFT]") ||
		!strings.Contains(drifted.Render(), "bash scripts/build-host-artifacts.sh") {
		t.Fatalf("generated residue not diagnosed:\n%s", drifted.Render())
	}
}
