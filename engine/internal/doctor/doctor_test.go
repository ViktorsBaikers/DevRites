package doctor

import (
	"os"
	"path/filepath"
	"testing"

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
	// Pack marker newer than the binary → warn, not refuse.
	claude := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claude, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claude, "devrites.version"), []byte("2.0.0\n"), 0o644); err != nil {
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
