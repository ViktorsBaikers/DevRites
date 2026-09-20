package install

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/fsutil"
)

func (r *runner) installBinary() error {
	if !r.opts.WithBinary || os.Getenv("DEVRITES_NO_BINARY") == "1" {
		fmt.Fprintln(r.opts.Stdout, "  engine binary: skipped (--no-binary).")
		return nil
	}
	if r.opts.DryRun {
		fmt.Fprintln(r.opts.Stdout, "  would install the global devrites-engine control-plane binary")
		return nil
	}
	tag := r.binaryTag()
	incoming := strings.TrimPrefix(tag, "v")
	dest := binaryDest()
	if exists(dest) {
		if ev := engineVersion(dest); semverLike(ev) && semverLike(incoming) && verGT(ev, incoming) {
			fmt.Fprintf(r.opts.Stderr, "warning: engine binary: installed %s is newer than %s - refusing to downgrade (kept).\n", ev, tag)
			return nil
		}
	}
	staged, cleanup := r.preparedBinary, func() {}
	if staged == "" {
		var err error
		staged, cleanup, err = r.acquireBinary(tag)
		if err != nil {
			return r.binaryInstallFailure(err)
		}
	}
	defer cleanup()
	if _, err := verifyEngineBinary(staged, tag, 30*time.Second); err != nil {
		return fmt.Errorf("verify staged engine binary %s: %w", staged, err)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	// #nosec G304 -- staged artifact from the checksum-verified release temp dir
	data, err := os.ReadFile(staged)
	if err != nil {
		return err
	}
	backup, oldMode, hadOld, err := backupBinary(dest)
	if err != nil {
		return err
	}
	if backup != "" {
		defer func() { _ = os.Remove(backup) }()
	}
	if err := fsutil.WriteFileAtomic(dest, data, 0o755); err != nil {
		if restoreErr := restoreBinary(dest, backup, oldMode, hadOld); restoreErr != nil {
			return fmt.Errorf("install binary: %v; restore previous binary: %w", err, restoreErr)
		}
		return err
	}
	_ = os.Chmod(dest, 0o755)
	if _, err := verifyEngineBinary(dest, tag, 30*time.Second); err != nil {
		if restoreErr := restoreBinary(dest, backup, oldMode, hadOld); restoreErr != nil {
			return fmt.Errorf("verify installed engine binary: %v; restore previous binary: %w", err, restoreErr)
		}
		if hadOld {
			return fmt.Errorf("verify installed engine binary: %w (previous binary restored)", err)
		}
		return fmt.Errorf("verify installed engine binary: %w (bad binary removed)", err)
	}
	fmt.Fprintf(r.opts.Stdout, "  engine binary: installed %s\n", dest)
	return nil
}

func (r *runner) binaryTag() string {
	if r.requiredBinaryTag != "" {
		return r.requiredBinaryTag
	}
	if tag := os.Getenv("DEVRITES_REF"); semverLike(tag) {
		return tag
	}
	return "v" + strings.TrimPrefix(installedVersion(r.source), "v")
}

func backupBinary(dest string) (string, fs.FileMode, bool, error) {
	info, err := os.Lstat(dest)
	if os.IsNotExist(err) {
		return "", 0, false, nil
	}
	if err != nil {
		return "", 0, false, fmt.Errorf("inspect existing engine binary: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", 0, false, fmt.Errorf("refusing to replace non-regular engine binary: %s", dest)
	}
	// #nosec G304 -- existing engine binary under the operator install dir; regular-file checked above
	data, err := os.ReadFile(dest)
	if err != nil {
		return "", 0, false, fmt.Errorf("read existing engine binary: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(dest), "."+filepath.Base(dest)+".backup-*")
	if err != nil {
		return "", 0, false, fmt.Errorf("create engine binary backup: %w", err)
	}
	backup := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(backup)
		return "", 0, false, fmt.Errorf("create engine binary backup: %w", err)
	}
	if err := fsutil.WriteFileAtomic(backup, data, info.Mode().Perm()); err != nil {
		_ = os.Remove(backup)
		return "", 0, false, fmt.Errorf("write engine binary backup: %w", err)
	}
	return backup, info.Mode().Perm(), true, nil
}

func restoreBinary(dest, backup string, mode fs.FileMode, hadOld bool) error {
	if !hadOld {
		if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	// #nosec G304 -- backup of the same install-dir binary
	data, err := os.ReadFile(backup)
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(dest, data, mode)
}

func (r *runner) binaryInstallFailure(err error) error {
	if r.preparedBinary != "" {
		return err
	}
	if os.Getenv("DEVRITES_ENGINE_CLI") != "" {
		return err
	}
	fmt.Fprintf(r.opts.Stderr, "warning: engine binary: %v; continuing without it.\n", err)
	return nil
}

func (r *runner) acquireBinary(tag string) (string, func(), error) {
	path := os.Getenv("DEVRITES_ENGINE_CLI")
	if path == "" {
		return "", func() {}, fmt.Errorf("no DEVRITES_ENGINE_CLI handoff")
	}
	if !exists(path) {
		return "", func() {}, fmt.Errorf("DEVRITES_ENGINE_CLI points to a missing binary: %s", path)
	}
	if _, err := verifyEngineBinary(path, tag, 30*time.Second); err != nil {
		return "", func() {}, fmt.Errorf("DEVRITES_ENGINE_CLI is incompatible: %w", err)
	}
	return path, func() {}, nil
}

func (r *runner) removeBinary() error {
	if r.opts.KeepBinary {
		fmt.Fprintln(r.opts.Stdout, "  kept the global devrites-engine binary (--keep-binary).")
		return nil
	}
	for _, p := range binaryCandidates() {
		if p == "" || !exists(p) {
			continue
		}
		if r.opts.DryRun {
			fmt.Fprintf(r.opts.Stdout, "  [remove] %s (global engine binary)\n", p)
			continue
		}
		if err := os.Remove(p); err == nil {
			fmt.Fprintf(r.opts.Stdout, "  [remove] %s\n", p)
			return nil
		}
	}
	return nil
}

func binaryDest() string {
	if dir := os.Getenv("DEVRITES_BIN_DIR"); dir != "" {
		return filepath.Join(dir, engineBinaryName())
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "bin", engineBinaryName())
	}
	return filepath.Join("/usr/local/bin", engineBinaryName())
}

func engineBinaryName() string {
	if runtime.GOOS == "windows" {
		return "devrites-engine.exe"
	}
	return "devrites-engine"
}

func binaryCandidates() []string {
	candidates := []string{}
	if dir := os.Getenv("DEVRITES_BIN_DIR"); dir != "" {
		candidates = append(candidates, filepath.Join(dir, engineBinaryName()))
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		candidates = append(candidates, filepath.Join(home, ".local", "bin", engineBinaryName()))
	}
	return append(candidates, filepath.Join("/usr/local/bin", engineBinaryName()))
}

func engineVersion(path string) string {
	got, err := readEngineVersion(path, 30*time.Second)
	if err != nil {
		return ""
	}
	return got
}

func verifyEngineBinary(path, want string, timeout time.Duration) (string, error) {
	want = strings.TrimPrefix(strings.TrimSpace(want), "v")
	if want == "" || want == "dev" || !semverLike(want) {
		return "", fmt.Errorf("invalid requested version %q", want)
	}
	got, err := readEngineVersion(path, timeout)
	if err != nil {
		return "", err
	}
	if strings.TrimPrefix(got, "v") != want {
		return "", fmt.Errorf("version mismatch: got %s want %s", got, want)
	}
	return got, nil
}

func readEngineVersion(path string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// #nosec G204,G702 -- execs the staged engine binary itself for a version probe
	out, err := exec.CommandContext(ctx, path, "version").Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("version command timed out after %s", timeout)
		}
		return "", fmt.Errorf("run exact path %s version: %w", path, err)
	}
	line := strings.TrimSuffix(string(out), "\n")
	line = strings.TrimSuffix(line, "\r")
	if strings.ContainsAny(line, "\r\n") {
		return "", fmt.Errorf("invalid multi-line version output")
	}
	line = strings.TrimSpace(line)
	if line == "" || line == "dev" || !semverLike(line) {
		return "", fmt.Errorf("invalid version output %q", line)
	}
	return line, nil
}
