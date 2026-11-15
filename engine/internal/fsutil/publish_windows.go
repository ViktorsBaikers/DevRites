//go:build windows && (amd64 || arm64)

package fsutil

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	publishNTDLL                = syscall.NewLazyDLL("ntdll.dll")
	publishNtCreateFile         = publishNTDLL.NewProc("NtCreateFile")
	publishNtSetInformationFile = publishNTDLL.NewProc("NtSetInformationFile")
	publishNtStatusToDosError   = publishNTDLL.NewProc("RtlNtStatusToDosError")
	publishGetVolumeInformation = syscall.NewLazyDLL("kernel32.dll").NewProc("GetVolumeInformationByHandleW")
)

// Layouts match Go 1.27.4 internal/syscall/windows and the Windows NT ABI.
type publishUnicodeString struct {
	Length        uint16
	MaximumLength uint16
	Buffer        *uint16
}

type publishObjectAttributes struct {
	Length             uint32
	RootDirectory      syscall.Handle
	ObjectName         *publishUnicodeString
	Attributes         uint32
	SecurityDescriptor uintptr
	SecurityQoS        uintptr
}

type publishIOStatus struct {
	Status      uintptr
	Information uintptr
}

func publishNTError(status uintptr) error {
	if int32(status) >= 0 {
		return nil
	}
	if err := publishNtStatusToDosError.Find(); err != nil {
		return fmt.Errorf("NT status 0x%x: %w", uint32(status), err)
	}
	code, _, _ := publishNtStatusToDosError.Call(status)
	return syscall.Errno(code)
}

func nativePublishOpen(parent *os.File, name string, directory, movable bool) (*os.File, error) {
	if err := publishNtCreateFile.Find(); err != nil {
		return nil, err
	}
	ntName := name
	if ntName == "." {
		ntName = "" // Relative empty name reopens the retained directory.
	}
	chars, err := syscall.UTF16FromString(ntName)
	if err != nil {
		return nil, err
	}
	if len(chars) > 256 {
		return nil, syscall.EINVAL
	}
	text := publishUnicodeString{Length: uint16(2 * (len(chars) - 1)), MaximumLength: uint16(2 * len(chars)), Buffer: &chars[0]}
	attrs := publishObjectAttributes{
		RootDirectory: syscall.Handle(parent.Fd()),
		ObjectName:    &text,
		Attributes:    0x40 | 0x1000, // OBJ_CASE_INSENSITIVE | OBJ_DONT_REPARSE
	}
	attrs.Length = uint32(unsafe.Sizeof(attrs))
	// FILE_GENERIC_READ | FILE_GENERIC_WRITE; use specific rights, not generic
	// access bits, because NtCreateFile forbids generic rights on directories.
	access := uint32(0x120089 | 0x120116)
	if movable {
		access |= 0x10000 // DELETE, required by FileRenameInformation
	}
	options := uint32(0x20 | 0x4000 | 0x200000) // synchronous, backup intent, open reparse point
	if directory {
		options |= 1 // FILE_DIRECTORY_FILE
	} else {
		options |= 0x40 // FILE_NON_DIRECTORY_FILE
	}
	var handle syscall.Handle
	var io publishIOStatus
	status, _, _ := publishNtCreateFile.Call(
		uintptr(unsafe.Pointer(&handle)), uintptr(access), uintptr(unsafe.Pointer(&attrs)),
		uintptr(unsafe.Pointer(&io)), 0, syscall.FILE_ATTRIBUTE_NORMAL,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		1, uintptr(options), 0, 0, // FILE_OPEN, never create or truncate
	)
	runtime.KeepAlive(parent)
	runtime.KeepAlive(chars)
	if err := publishNTError(status); err != nil {
		return nil, &os.PathError{Op: "publication NtCreateFile", Path: name, Err: err}
	}
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(handle, &info); err != nil {
		_ = syscall.CloseHandle(handle)
		return nil, err
	}
	if info.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		_ = syscall.CloseHandle(handle)
		return nil, fmt.Errorf("publication path %q is a reparse point", name)
	}
	if directory {
		if err := requirePublicationNTFS(handle); err != nil {
			_ = syscall.CloseHandle(handle)
			return nil, err
		}
	}
	return os.NewFile(uintptr(handle), name), nil
}

func nativePublishRename(sourceParent, stage, targetParent *os.File, from, to string) error {
	if err := publishNtSetInformationFile.Find(); err != nil {
		return err
	}
	chars, err := syscall.UTF16FromString(to)
	if err != nil {
		return err
	}
	// Only a validated base name is passed; never reconstruct root.Name().
	var info struct {
		ReplaceIfExists byte
		RootDirectory   syscall.Handle
		FileNameLength  uint32
		FileName        [256]uint16
	}
	if len(chars) > len(info.FileName) {
		return syscall.EINVAL
	}
	info.RootDirectory = syscall.Handle(targetParent.Fd())
	info.FileNameLength = uint32(2 * (len(chars) - 1))
	copy(info.FileName[:], chars)
	var io publishIOStatus
	status, _, _ := publishNtSetInformationFile.Call(stage.Fd(), uintptr(unsafe.Pointer(&io)), uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info), 10) // FileRenameInformation; ReplaceIfExists=false
	runtime.KeepAlive(stage)
	runtime.KeepAlive(targetParent)
	if err := publishNTError(status); err != nil {
		return &os.LinkError{Op: "NtSetInformationFile noreplace", Old: from, New: to, Err: err}
	}
	return nil
}

func nativePublishSync(file *os.File) error {
	// Directory handles have mapped write rights and were verified as NTFS.
	// Even there, a failed flush must never be treated as durable success.
	err := syscall.FlushFileBuffers(syscall.Handle(file.Fd()))
	runtime.KeepAlive(file)
	if err != nil {
		return &os.PathError{Op: "publication FlushFileBuffers", Path: file.Name(), Err: err}
	}
	return nil
}

func requirePublicationNTFS(handle syscall.Handle) error {
	if err := publishGetVolumeInformation.Find(); err != nil {
		return err
	}
	var name [261]uint16
	ok, _, err := publishGetVolumeInformation.Call(uintptr(handle), 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(&name[0])), uintptr(len(name)))
	if ok == 0 {
		return fmt.Errorf("publication volume information: %w", err)
	}
	// MS-FSA 2.1.5.7 footnote 80: non-NTFS directory flush can be a no-op.
	if filesystem := syscall.UTF16ToString(name[:]); filesystem != "NTFS" {
		return fmt.Errorf("publication requires NTFS directory durability, got %q: %w", filesystem, errors.ErrUnsupported)
	}
	return nil
}
