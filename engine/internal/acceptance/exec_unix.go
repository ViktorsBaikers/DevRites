//go:build !windows

package acceptance

import (
	"os/exec"
	"syscall"
)

// configureProcessGroup puts the check in its own process group so a timeout
// can signal the shell plus every descendant it spawned.
func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killGroup(cmd *exec.Cmd) {
	// #nosec G204 -- PID comes from the child this process just started; the
	// negative form signals the process group it leads.
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
