package parallel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/gitenv"
)

const gitTimeout = 60 * time.Second

func git(repo string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	env := append(gitenv.Sanitize(os.Environ()),
		"GIT_TERMINAL_PROMPT=0",
		"GCM_INTERACTIVE=never",
		"GIT_PAGER=cat",
		"PAGER=cat",
		"LC_ALL=C",
	)
	if gitOpNeedsIdentity(args) {
		env = ensureGitIdentityEnv(env)
	}
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = out
		}
		if ctx.Err() != nil {
			return out, fmt.Errorf("git %s: %w", strings.Join(args, " "), ctx.Err())
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return out, fmt.Errorf("git %s: exit %d: %s", strings.Join(args, " "), exitErr.ExitCode(), detail)
		}
		return out, fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, detail)
	}
	return out, nil
}

func gitOpNeedsIdentity(args []string) bool {
	for _, a := range args {
		switch a {
		case "cherry-pick", "commit", "merge", "rebase", "am":
			return true
		}
	}
	return false
}

// ensureGitIdentityEnv fills author/committer when the host has none (common on
// CI images). Prefer existing env identity when present.
func ensureGitIdentityEnv(env []string) []string {
	keys := map[string]string{
		"GIT_AUTHOR_NAME":     "devrites-engine",
		"GIT_AUTHOR_EMAIL":    "engine@devrites.invalid",
		"GIT_COMMITTER_NAME":  "devrites-engine",
		"GIT_COMMITTER_EMAIL": "engine@devrites.invalid",
	}
	have := make(map[string]bool, len(env))
	for _, e := range env {
		if i := strings.IndexByte(e, '='); i > 0 {
			have[e[:i]] = true
		}
	}
	for k, v := range keys {
		if !have[k] {
			env = append(env, k+"="+v)
		}
	}
	return env
}

func revParse(repo, rev string) (string, error) {
	return git(repo, "rev-parse", "--verify", rev+"^{commit}")
}

func isAncestor(repo, ancestor, descendant string) (bool, error) {
	_, err := git(repo, "merge-base", "--is-ancestor", ancestor, descendant)
	if err == nil {
		return true, nil
	}
	if strings.Contains(err.Error(), "exit 1") {
		return false, nil
	}
	return false, err
}

func diffNames(repo, a, b string) ([]string, error) {
	out, err := git(repo, "diff", "--name-only", a, b)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	lines := strings.Split(out, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return lines, nil
}

func headSHA(repo string) (string, error) {
	return revParse(repo, "HEAD")
}

func resetHard(repo, sha string) error {
	_, err := git(repo, "reset", "--hard", sha)
	return err
}

func porcelainDirty(repo string) (bool, error) {
	out, err := git(repo, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

func branchExists(repo, branch string) (bool, error) {
	_, err := git(repo, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	if err == nil {
		return true, nil
	}
	if strings.Contains(err.Error(), "exit 1") {
		return false, nil
	}
	return false, err
}

func deleteBranch(repo, branch string) error {
	exists, err := branchExists(repo, branch)
	if err != nil || !exists {
		return err
	}
	_, err = git(repo, "branch", "-D", branch)
	return err
}

func worktreeAdd(repo, path, branch, startPoint string) error {
	_, err := git(repo, "worktree", "add", "-b", branch, path, startPoint)
	return err
}

func worktreeRemove(repo, path string) error {
	_, err := git(repo, "worktree", "remove", "--force", path)
	if err != nil {
		_ = os.RemoveAll(path)
		_, _ = git(repo, "worktree", "prune")
	}
	return err
}

func mergeFFOnly(repo, commit string) error {
	_, err := git(repo, "merge", "--ff-only", commit)
	return err
}

func cherryPickRange(repo, fromExclusive, toInclusive string) error {
	_, err := git(repo, "cherry-pick", "-x", fromExclusive+".."+toInclusive)
	return err
}

func cherryPickAbort(repo string) {
	_, _ = git(repo, "cherry-pick", "--abort")
}
