package profile

import (
	"path/filepath"
	"testing"
)

func TestDeriveReportsUnreadableRoot(t *testing.T) {
	_, err := derive(filepath.Join(t.TempDir(), "missing"), "root", "head")
	if err == nil {
		t.Fatal("derive succeeded for a missing repository root")
	}
}
