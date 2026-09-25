package records

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/gitenv"
	"github.com/devrites/devrites/internal/overhaul/snapshot"
)

var runIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// excludeLine is what init adds to the local .git/info/exclude when git does
// not already ignore the run area. It never edits the shared .gitignore.
const excludeLine = "/" + snapshot.RunAreaDir

// initRun creates <repo>/.devrites/overhaul/<run-id>/ owner-only and makes
// sure git ignores it, so run records never show up as repository changes.
func initRun(repo, runID string, stdout io.Writer) error {
	if !runIDPattern.MatchString(runID) {
		return fmt.Errorf("run id %q: use 1-64 letters, digits, '.', '_' or '-', starting with a letter or digit", runID)
	}
	top, err := gitText(repo, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("%s is not a git repository: %v", repo, err)
	}
	runDir := filepath.Join(filepath.FromSlash(top), filepath.FromSlash(snapshot.RunAreaDir), runID)
	if _, err := os.Lstat(runDir); err == nil {
		return fmt.Errorf("run area %s already exists; resume it or pick a new run id", runDir)
	}
	probe := snapshot.RunAreaDir + runID + "/probe"
	how := "already-ignored"
	ignored, err := gitIgnores(top, probe)
	if err != nil {
		return err
	}
	if !ignored {
		if err := addExclude(top); err != nil {
			return err
		}
		if ignored, err = gitIgnores(top, probe); err != nil {
			return err
		}
		if !ignored {
			return fmt.Errorf("git still does not ignore %s after adding %s to info/exclude; a negated rule may re-include it", snapshot.RunAreaDir, excludeLine)
		}
		how = "added-to-info-exclude"
	}
	if err := os.MkdirAll(filepath.Dir(runDir), 0o700); err != nil {
		return err
	}
	if err := os.Mkdir(runDir, 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(struct {
		Run    string `json:"run"`
		Ignore string `json:"ignore"`
	}{runDir, how})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, string(b))
	return err
}

func addExclude(top string) error {
	p, err := gitText(top, "rev-parse", "--git-path", "info/exclude")
	if err != nil {
		return err
	}
	path := filepath.FromSlash(p)
	if !filepath.IsAbs(path) {
		path = filepath.Join(top, path)
	}
	// #nosec G304 -- exclude file inside the repo's own resolved git dir
	if data, err := os.ReadFile(path); err == nil {
		for _, l := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(l) == excludeLine {
				return nil
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// #nosec G304 -- exclude file inside the repo's own resolved git dir
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, werr := fmt.Fprintf(f, "\n# /overhaul run records (local only)\n%s\n", excludeLine)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	return werr
}

// gitIgnores reports whether git's ignore rules match rel (tracked or not).
func gitIgnores(top, rel string) (bool, error) {
	cmd := gitCmd(top, "check-ignore", "-q", "--no-index", rel)
	err := cmd.Run()
	var ee *exec.ExitError
	switch {
	case err == nil:
		return true, nil
	case errors.As(err, &ee) && ee.ExitCode() == 1:
		return false, nil
	default:
		return false, fmt.Errorf("git check-ignore: %v", err)
	}
}

func gitText(dir string, args ...string) (string, error) {
	out, err := gitCmd(dir, args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitCmd(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...) // #nosec G204 -- fixed git binary; arguments passed as argv, no shell
	cmd.Env = append(gitenv.Sanitize(os.Environ()), "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0")
	return cmd
}
