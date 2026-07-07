package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/harness"
	"github.com/devrites/devrites/internal/orient"
	"github.com/devrites/devrites/internal/state"
)

// hookOrient emits the SessionStart orientation for the active feature. It is
// FAIL-OPEN and read-only: on a non-DevRites directory, an unreadable workspace,
// or nothing to say, it stays silent and exits 0 so a session is never wedged.
// The engine's inline shell guard handles a missing binary; this handles a
// present binary run outside a workspace.
func hookOrient(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	// The SessionStart payload is drained but unused — orientation is a pure read
	// of the workspace files, not a function of the harness's stdin.
	_, _ = io.Copy(io.Discard, stdin)

	root, err := orient.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		return exitOK // not a DevRites workspace — silent no-op
	}
	text, has, err := orient.Digest(root)
	if err != nil {
		debugf(stderr, "orient: %v", err)
		return exitOK
	}
	if !has {
		return exitOK // no active feature — silent
	}
	out, err := h.SessionStartContext(text)
	if err != nil {
		debugf(stderr, "orient: %v", err)
		return exitOK
	}
	fmt.Fprintln(stdout, out)
	return exitOK
}

// hookStopGate refuses to end a turn at a provably inconsistent rest point. It
// mirrors devrites-stop-gate.sh: OBSERVE by default, blocking only when
// DEVRITES_STOP_GATE=enforce; loop-guarded by the harness's stop_hook_active so
// it can never wedge the session; fully fail-open.
func hookStopGate(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	in := h.ParseStopInput(stdin)
	if in.StopHookActive {
		return exitOK // already re-entered this stop cycle — let it stop
	}
	root, err := orient.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		return exitOK
	}
	res, err := gate.StopGate(root)
	if err != nil || !res.Blocked {
		return exitOK
	}
	if os.Getenv("DEVRITES_STOP_GATE") != "enforce" {
		// Observe mode: never block. Record a would-block to the feature's
		// append-only log (mirroring devrites-stop-gate.sh) so the invariant is
		// visible without gating; the append is O_APPEND-atomic across the
		// concurrent processes DevRites spawns.
		logPath := filepath.Join(root, "features", res.Slug, ".stop-gate.log")
		if err := state.AppendLog(logPath, "WOULD-BLOCK\t"+res.Reason); err != nil {
			debugf(stderr, "stop-gate log: %v", err)
		}
		return exitOK
	}
	out, err := h.StopBlock(res.Reason)
	if err != nil {
		debugf(stderr, "stop-gate: %v", err)
		return exitOK
	}
	fmt.Fprintln(stdout, out)
	return exitOK
}

// debugf writes an operational diagnostic only when DEVRITES_DEBUG is set, so a
// fail-open hook can surface WHY it stayed silent without ever polluting normal
// hook stdout/stderr.
func debugf(stderr io.Writer, format string, args ...any) {
	if os.Getenv("DEVRITES_DEBUG") != "" {
		fmt.Fprintf(stderr, "devrites: "+format+"\n", args...)
	}
}
