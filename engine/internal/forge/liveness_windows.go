//go:build windows

package forge

import (
	"errors"
	"fmt"
	"strconv"
	"syscall"
)

const (
	processQueryLimitedInformation = 0x1000
	statusStillActive              = 259
	errorInvalidParameter          = syscall.Errno(87)
)

var errProcessNotRunning = errors.New("process is not running")

// ProcessStartToken returns a stable token for the current incarnation of pid.
// Callers record it with the worker PID so PID reuse is distinguishable.
func ProcessStartToken(pid int) (string, error) {
	handle, err := openLiveProcess(pid)
	if err != nil {
		return "", fmt.Errorf("forge: cannot prove process start for PID %d", pid)
	}
	defer syscall.CloseHandle(handle)

	var creation, exit, kernel, user syscall.Filetime
	if err := syscall.GetProcessTimes(handle, &creation, &exit, &kernel, &user); err != nil {
		return "", fmt.Errorf("forge: cannot prove process start for PID %d", pid)
	}
	start := uint64(creation.HighDateTime)<<32 | uint64(creation.LowDateTime)
	return sha256Hex([]byte(strconv.FormatUint(start, 10))), nil
}

func processLiveness(pid int) liveness {
	handle, err := openLiveProcess(pid)
	if err == nil {
		syscall.CloseHandle(handle)
		return livenessLive
	}
	if errors.Is(err, errProcessNotRunning) || errors.Is(err, errorInvalidParameter) {
		return livenessDead
	}
	return livenessUnknown
}

func openLiveProcess(pid int) (syscall.Handle, error) {
	if pid <= 0 || uint64(pid) > uint64(^uint32(0)) {
		return 0, errors.New("process ID is outside the Windows process ID range")
	}
	handle, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return 0, err
	}
	var exitCode uint32
	if err := syscall.GetExitCodeProcess(handle, &exitCode); err != nil {
		syscall.CloseHandle(handle)
		return 0, err
	}
	if exitCode != statusStillActive {
		syscall.CloseHandle(handle)
		return 0, errProcessNotRunning
	}
	return handle, nil
}
