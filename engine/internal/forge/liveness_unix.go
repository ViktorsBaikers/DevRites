//go:build !windows

package forge

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ProcessStartToken returns a stable token for the current incarnation of pid.
// Callers record it with the worker PID so PID reuse is distinguishable.
func ProcessStartToken(pid int) (string, error) {
	if pid <= 0 {
		return "", errors.New("forge: PID must be positive")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ps", "-o", "lstart=,command=", "-p", strconv.Itoa(pid))
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return "", fmt.Errorf("forge: cannot prove process start for PID %d", pid)
	}
	return sha256Hex([]byte(strings.TrimSpace(string(out)))), nil
}

func processLiveness(pid int) liveness {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := exec.CommandContext(ctx, "ps", "-p", strconv.Itoa(pid)).Run()
	if err == nil {
		return livenessLive
	}
	if ctx.Err() != nil {
		return livenessUnknown
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return livenessDead
	}
	return livenessUnknown
}
