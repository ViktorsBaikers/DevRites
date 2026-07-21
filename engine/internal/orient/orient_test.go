package orient

import (
	"os"
	"path/filepath"
	"testing"
)

func TestActiveSlugHonorsExplicitWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ActiveFile), []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_WORKSPACE", filepath.Join(root, "work", "explicit"))
	slug, err := ActiveSlug(root)
	if err != nil {
		t.Fatal(err)
	}
	if slug != "explicit" {
		t.Fatalf("ActiveSlug = %q, want explicit", slug)
	}
}
