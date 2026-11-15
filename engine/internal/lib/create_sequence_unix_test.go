//go:build unix

package lib

import (
	"path/filepath"
	"syscall"
	"testing"
)

func TestCreateSequenceRejectsFIFOBrief(t *testing.T) {
	root, _ := sequenceCreationFixture(t, true)
	brief := filepath.Join(t.TempDir(), "scope.fifo")
	if err := syscall.Mkfifo(brief, 0o600); err != nil {
		t.Fatal(err)
	}
	sequenceRefused(t, root, "future17", brief)
}
