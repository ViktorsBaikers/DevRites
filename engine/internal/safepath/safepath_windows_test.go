//go:build windows

package safepath

import (
	"path/filepath"
	"testing"
)

func TestWindowsNativeVolumeAndSeparatorContainment(t *testing.T) {
	parent := t.TempDir()
	if !WithinResolved(filepath.Join(parent, "child", "file"), parent) {
		t.Fatal("native Windows child path should remain contained")
	}
	volume := filepath.VolumeName(parent)
	if volume == "" {
		t.Fatalf("temporary path %q has no Windows volume", parent)
	}
	other := `Z:\outside`
	if filepath.VolumeName(parent) == "Z:" {
		other = `Y:\outside`
	}
	if WithinResolved(other, parent) {
		t.Fatalf("different Windows volumes reported contained: %q in %q", other, parent)
	}
}
