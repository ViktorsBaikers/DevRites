package orient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newBlankRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ActiveFile), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestFirstRunDigestGreenfieldNudgesOnce(t *testing.T) {
	root := newBlankRoot(t)
	text, has := FirstRunDigest(root)
	if !has {
		t.Fatal("first call: want a nudge, got silence")
	}
	if !strings.Contains(text, "/rite-spec") {
		t.Fatalf("greenfield nudge should point at /rite-spec, got %q", text)
	}
	if _, err := os.Stat(filepath.Join(root, FirstRunFile)); err != nil {
		t.Fatalf("marker not written: %v", err)
	}
	if _, has = FirstRunDigest(root); has {
		t.Fatal("second call: want silence, marker should suppress")
	}
}

func TestFirstRunDigestBrownfieldPointsAtAdopt(t *testing.T) {
	root := newBlankRoot(t)
	project := filepath.Dir(root)
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	text, has := FirstRunDigest(root)
	if !has {
		t.Fatal("want a nudge, got silence")
	}
	if !strings.Contains(text, "/rite-adopt") {
		t.Fatalf("brownfield nudge should point at /rite-adopt, got %q", text)
	}
}

func TestFirstRunDigestActiveFeatureStaysSilent(t *testing.T) {
	root := newBlankRoot(t)
	if err := os.WriteFile(filepath.Join(root, ActiveFile), []byte("some-feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, has := FirstRunDigest(root); has {
		t.Fatal("active feature: first-run nudge must stay silent")
	}
	if _, err := os.Stat(filepath.Join(root, FirstRunFile)); !os.IsNotExist(err) {
		t.Fatalf("active feature must not consume the first-run marker, stat err = %v", err)
	}
}

func TestFirstRunDigestUnwritableMarkerStaysSilent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing", ".devrites")
	if _, has := FirstRunDigest(root); has {
		t.Fatal("unwritable marker: want silence, never a per-session nag")
	}
}
