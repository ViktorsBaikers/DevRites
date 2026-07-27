package lib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var errNotGitRepository = errors.New("not a git repository")

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
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()
	commandArgs := append([]string{"-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", commandArgs...)
	cmd.WaitDelay = 2 * time.Second
	if env == nil {
		env = os.Environ()
	} else {
		env = append([]string{}, env...)
	}
	cmd.Env = append(
		env,
		"GIT_TERMINAL_PROMPT=0",
		"GCM_INTERACTIVE=never",
		"GIT_PAGER=cat",
		"PAGER=cat",
		"LC_ALL=C",
	)
	var output cappedGitOutput
	cmd.Stdout = &output
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

func gitErrorExitCode(err error) (int, bool) {
	var commandErr *gitCommandError
	if !errors.As(err, &commandErr) {
		return 0, false
	}
	return commandErr.exitCode, commandErr.exitCode >= 0
}

// gitToplevel returns the absolute path of the Git working tree containing dir.
// Repository absence is distinct from Git execution failure so integrity gates
// can fail closed on a missing binary, corrupt repository, or poisoned runtime.
func gitToplevel(dir string) (string, error) {
	out, err := runGitCommand(dir, nil, "rev-parse", "--show-toplevel")
	if err != nil {
		var commandErr *gitCommandError
		if errors.As(err, &commandErr) {
			detail := strings.ToLower(commandErr.output)
			if strings.Contains(detail, "not a git repository") ||
				strings.Contains(detail, "not a git work tree") {
				return "", errNotGitRepository
			}
		}
		return "", err
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return "", fmt.Errorf("git rev-parse returned an empty top-level path")
	}
	return filepath.Clean(root), nil
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

// isAllDigits reports whether s is a non-empty run of ASCII digits.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// featureFile is the path to a leaf file inside a feature's directory:
// <root>/features/<slug>/<name>.
func featureFile(root, slug, name string) string {
	return filepath.Join(featureDir(root, slug), name)
}

// slugOrActive returns the slug named in args[0], falling back to the active
// feature when none is given.
func slugOrActive(root string, args []string) string {
	if slug := argAt(args, 0); slug != "" {
		return slug
	}
	return activeSlug(root)
}
