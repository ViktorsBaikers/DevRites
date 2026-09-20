//go:build windows

package acceptance

import "os/exec"

// configureProcessGroup is a no-op on Windows: CommandContext already wires
// the kill to the child handle, and a detached supervisor is beyond the
// stdlib-only engine boundary.
func configureProcessGroup(cmd *exec.Cmd) {}

func killGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
