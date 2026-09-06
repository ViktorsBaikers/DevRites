package lib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/gitenv"
)

const (
	gitCommandTimeout = 30 * time.Second
	gitOutputLimit    = 16 << 20
)

type cappedGitOutput struct {
	bytes.Buffer
	truncated bool
}

func (w *cappedGitOutput) Write(p []byte) (int, error) {
	remaining := gitOutputLimit - w.Len()
	if remaining > 0 {
		if len(p) > remaining {
			_, _ = w.Buffer.Write(p[:remaining])
			w.truncated = true
		} else {
			_, _ = w.Buffer.Write(p)
		}
	} else {
		w.truncated = true
	}
	return len(p), nil
}

type gitCommandError struct {
	args     []string
	output   string
	exitCode int
	err      error
}

func (e *gitCommandError) Error() string {
	detail := strings.TrimSpace(e.output)
	if detail == "" {
		return fmt.Sprintf("git %s: %v", strings.Join(e.args, " "), e.err)
	}
	return fmt.Sprintf("git %s: %v: %s", strings.Join(e.args, " "), e.err, detail)
}

func (e *gitCommandError) Unwrap() error { return e.err }

func runGitCommand(dir string, env []string, args ...string) ([]byte, error) {
	return runGitCommandIO(dir, env, nil, nil, args...)
}

func runGitCommandIO(dir string, env []string, input []byte, stdout io.Writer, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()
	commandArgs := append([]string{"-C", dir}, args...)
	// #nosec G204 -- git -C <operator repo dir>; fixed binary, sanitized env
	cmd := exec.CommandContext(ctx, "git", commandArgs...)
	cmd.WaitDelay = 2 * time.Second
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	if env == nil {
		env = os.Environ()
	}
	env = gitenv.Sanitize(env)
	cmd.Env = append(
		env,
		"GIT_TERMINAL_PROMPT=0",
		"GCM_INTERACTIVE=never",
		"GIT_PAGER=cat",
		"PAGER=cat",
		"LC_ALL=C",
	)
	var output cappedGitOutput
	if stdout == nil {
		cmd.Stdout = &output
	} else {
		cmd.Stdout = stdout
	}
	cmd.Stderr = &output
	err := cmd.Run()
	if err == nil {
		return output.Bytes(), nil
	}
	code := -1
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	detail := output.String()
	if output.truncated {
		detail += "\n[git output truncated]"
	}
	return nil, &gitCommandError{
		args:     args,
		output:   detail,
		exitCode: code,
		err:      err,
	}
}

// gitDiffNames lists the paths that differ, relative to gitRoot. With one ref it
// diffs the working tree against that ref; with two it diffs the two tree objects.
func gitDiffNames(gitRoot string, refs ...string) ([]string, error) {
	args := append([]string{"diff", "--name-only"}, refs...)
	out, err := runGitCommand(gitRoot, nil, args...)
	if err != nil {
		return nil, err
	}
	return splitLinesNoTrailing(out), nil
}

// argAt returns args[i] or "": the Go analogue of bash's ${N:-} positional reads.
func argAt(args []string, i int) string {
	if i < len(args) {
		return args[i]
	}
	return ""
}

func featureDir(root, slug string) string {
	return devritespaths.FeatureDir(root, slug)
}

func activeSlug(root string) string {
	slug, err := devritespaths.ActiveSlug(root)
	if err != nil {
		return ""
	}
	return slug
}

func splitLinesNoTrailing(data []byte) []string {
	value := strings.TrimSuffix(string(data), "\n")
	if value == "" {
		return nil
	}
	return strings.Split(value, "\n")
}
