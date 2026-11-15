//go:build windows && (amd64 || arm64)

package fsutil

import (
	"syscall"
	"testing"
	"unsafe"
)

func TestPublicationWindowsNativeLayouts(t *testing.T) {
	var attrs publishObjectAttributes
	var text publishUnicodeString
	var status publishIOStatus
	if unsafe.Sizeof(attrs) != 48 || unsafe.Offsetof(attrs.RootDirectory) != 8 || unsafe.Offsetof(attrs.ObjectName) != 16 || unsafe.Offsetof(attrs.Attributes) != 24 {
		t.Fatal("OBJECT_ATTRIBUTES does not match 64-bit NT ABI")
	}
	if unsafe.Sizeof(text) != 16 || unsafe.Offsetof(text.Buffer) != 8 {
		t.Fatal("UNICODE_STRING does not match 64-bit NT ABI")
	}
	if unsafe.Sizeof(status) != 16 || unsafe.Offsetof(status.Information) != 8 {
		t.Fatal("IO_STATUS_BLOCK does not match 64-bit NT ABI")
	}
}

func TestPublicationWindowsVolumeQueryFailsClosed(t *testing.T) {
	if err := requirePublicationNTFS(syscall.InvalidHandle); err == nil {
		t.Fatal("invalid volume handle accepted as durable NTFS")
	}
}
