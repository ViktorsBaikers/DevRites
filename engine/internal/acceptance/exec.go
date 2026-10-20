package acceptance

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// MaxOutputBytes caps each side of a check's captured output before the
// canonical matcher string is built; an overflow fails the gate rather than
// truncating into a match.
const MaxOutputBytes = 1 << 20

// DefaultTimeout bounds one check execution.
const DefaultTimeout = 120 * time.Second

// waitDelay bounds how long Execute waits for output pipes to close once the
// check timed out or exited; a descendant outside the process group may hold them.
const waitDelay = 5 * time.Second

// RunOutcome is one executed oracle: the process result plus the canonical
// output string used for EXPECT matching and the evidence fingerprint.
type RunOutcome struct {
	ExitCode    int
	Output      string
	OutputBytes int
	Err         error // process failed to start, timed out, or overflowed
	TimedOut    bool
}

// shellCommand returns the platform shell invocation for one CHECK string.
// CHECK lines are code: they run with the checker's ambient permissions,
// environment, and PATH. This package executes only commands approved against
// the workspace's vetted test-plan.md vocabulary.
func shellCommand(ctx context.Context, check, cwd string) *exec.Cmd {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		comspec := cmdComspec()
		// #nosec G204 -- check is a vetted test-plan.md preflight command; running
		// the approved command text is this package's purpose.
		cmd = exec.CommandContext(ctx, comspec, "/C", check)
	} else {
		// #nosec G204 -- check is a vetted test-plan.md preflight command; running
		// the approved command text is this package's purpose.
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", check)
	}
	cmd.Dir = cwd
	return cmd
}

func cmdComspec() string {
	if comspec := os.Getenv("ComSpec"); comspec != "" {
		return comspec
	}
	return `C:\Windows\System32\cmd.exe`
}

// limitedBuffer captures at most MaxOutputBytes and records overflow.
type limitedBuffer struct {
	buf      bytes.Buffer
	overflow bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > MaxOutputBytes {
		keep := MaxOutputBytes - b.buf.Len()
		b.buf.Write(p[:keep])
		b.overflow = true
		return len(p), nil
	}
	return b.buf.Write(p)
}

// Execute runs one CHECK under the platform shell in cwd and returns the
// canonical outcome. The canonical matcher string is all stdout, one synthetic
// newline when both streams are nonempty, then all stderr.
func Execute(ctx context.Context, check, cwd string, timeout time.Duration) RunOutcome {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := shellCommand(ctx, check, cwd)
	configureProcessGroup(cmd)
	// The default cancel kills only the shell; a forked child holding stdout
	// would keep Wait blocked past the timeout.
	cmd.Cancel = func() error {
		killGroup(cmd)
		return nil
	}
	cmd.WaitDelay = waitDelay
	stdout := &limitedBuffer{}
	stderr := &limitedBuffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	startErr := cmd.Start()
	if startErr != nil {
		return RunOutcome{Err: fmt.Errorf("check could not start: %v", startErr)}
	}
	waitErr := cmd.Wait()
	out, errText := stdout.buf.String(), stderr.buf.String()
	canonical := out
	if out != "" && errText != "" {
		canonical += "\n"
	}
	canonical += errText
	if ctx.Err() == context.DeadlineExceeded {
		// Keep what the check printed before the deadline so its transcript shows
		// how far it got; Err still fails the gate.
		return RunOutcome{Err: fmt.Errorf("check timed out after %s", timeout), TimedOut: true, Output: canonical, OutputBytes: len(canonical)}
	}
	if stdout.overflow || stderr.overflow {
		return RunOutcome{Err: fmt.Errorf("check output exceeded %d bytes", MaxOutputBytes)}
	}
	if len(canonical) > MaxOutputBytes {
		return RunOutcome{Err: fmt.Errorf("check output exceeded %d bytes after decoding", MaxOutputBytes)}
	}
	exit := 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exit = exitErr.ExitCode()
		} else {
			return RunOutcome{Err: fmt.Errorf("check did not complete: %v", waitErr), ExitCode: -1, Output: canonical, OutputBytes: len(canonical)}
		}
	}
	return RunOutcome{ExitCode: exit, Output: canonical, OutputBytes: len(canonical)}
}
