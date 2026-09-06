package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/hostpack"
	"github.com/devrites/devrites/internal/release"
)

func runUpdate(opts Options) error {
	target, err := filepath.Abs(opts.Target)
	if err != nil {
		return fmt.Errorf("resolve target %s: %w", opts.Target, err)
	}
	mf := filepath.Join(target, ManifestName)
	if !exists(mf) {
		return fmt.Errorf("no DevRites install found at %s (missing %s)", target, ManifestName)
	}
	installed := manifestHeader(mf, "devrites-version")
	if installed == "" {
		installed = "unknown"
	}
	if opts.SourceDir == "" {
		if opts.PayloadDir != "" {
			return fmt.Errorf("local update with --payload-dir also requires --source-dir")
		}
		return runRemoteUpdate(opts, target, installed)
	}
	source, err := filepath.Abs(opts.SourceDir)
	if err != nil {
		return fmt.Errorf("resolve source %s: %w", opts.SourceDir, err)
	}
	candidate := installedVersion(source)
	if candidate == "" || candidate == "unknown" {
		return fmt.Errorf("derive update version from local source %s: package.json has no version", source)
	}
	candidate = strings.TrimPrefix(candidate, "v")
	if os.Getenv("DEVRITES_UPDATE_HANDOFF") != "1" {
		fmt.Fprintln(opts.Stdout, "DevRites update")
		fmt.Fprintf(opts.Stdout, "  project:   %s\n", target)
		fmt.Fprintf(opts.Stdout, "  installed: %s\n", installed)
		fmt.Fprintf(opts.Stdout, "  candidate: %s\n", candidate)
	}
	if opts.UpdateCheck {
		if installed == candidate || strings.TrimPrefix(installed, "v") == candidate {
			fmt.Fprintln(opts.Stdout, "up to date.")
			return nil
		}
		return fmt.Errorf("update available: %s -> %s", installed, candidate)
	}
	if !opts.Force && (installed == candidate || strings.TrimPrefix(installed, "v") == candidate) {
		fmt.Fprintln(opts.Stdout, "already up to date (use --force to reinstall).")
		return nil
	}
	if opts.PayloadDir == "" {
		return fmt.Errorf("update requires --payload-dir with the locally prepared host payload")
	}
	payload, err := filepath.Abs(opts.PayloadDir)
	if err != nil {
		return fmt.Errorf("resolve payload %s: %w", opts.PayloadDir, err)
	}
	if err := hostpack.ValidatePayload(os.DirFS(payload), true, true); err != nil {
		return fmt.Errorf("local update payload under %s is invalid: %w", payload, err)
	}
	installFlags := manifestHeader(mf, "devrites-flags")
	next := DefaultOptions(ModeInstall)
	next.Stdout = opts.Stdout
	next.Stderr = opts.Stderr
	if installFlags != "" {
		if err := parseArgs(strings.Fields(installFlags), &next); err != nil {
			return fmt.Errorf("parse manifest install flags: %w", err)
		}
	}
	next.Target = target
	next.PayloadDir = payload
	next.SourceDir = source
	next.Force = opts.Force
	if opts.DryRun {
		next.DryRun = true
	}
	r, err := newRunner(next)
	if err != nil {
		return fmt.Errorf("prepare installer: %w", err)
	}
	r.requiredBinaryTag = "v" + candidate
	if next.WithBinary && os.Getenv("DEVRITES_NO_BINARY") != "1" && !next.DryRun {
		staged, cleanup, err := r.acquireBinary(r.requiredBinaryTag)
		if err != nil {
			return fmt.Errorf("prepare engine binary: %w", err)
		}
		defer cleanup()
		r.preparedBinary = staged
	}
	return r.install()
}

var (
	resolveLatestRelease = release.Latest
	acquireRelease       = release.Acquire
)

func runRemoteUpdate(opts Options, target, installed string) error {
	repository := os.Getenv("DEVRITES_REPO")
	if repository == "" {
		repository = release.DefaultRepository
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	tag, err := resolveLatestRelease(ctx, repository)
	if err != nil {
		return err
	}
	latest := strings.TrimPrefix(tag, "v")
	fmt.Fprintln(opts.Stdout, "DevRites update")
	fmt.Fprintf(opts.Stdout, "  project:   %s\n", target)
	fmt.Fprintf(opts.Stdout, "  installed: %s\n", installed)
	fmt.Fprintf(opts.Stdout, "  latest:    %s\n", latest)
	if opts.UpdateCheck {
		if installed == latest || strings.TrimPrefix(installed, "v") == latest {
			fmt.Fprintln(opts.Stdout, "up to date.")
			return nil
		}
		return fmt.Errorf("update available: %s -> %s", installed, latest)
	}
	if !opts.Force && (installed == latest || strings.TrimPrefix(installed, "v") == latest) {
		fmt.Fprintln(opts.Stdout, "already up to date (use --force to reinstall).")
		return nil
	}
	candidate, cleanup, err := acquireRelease(ctx, repository, tag)
	if err != nil {
		return fmt.Errorf("acquire update %s: %w", tag, err)
	}
	defer cleanup()
	fmt.Fprintf(opts.Stdout, "  bundle:   %s\n", candidate.BundleURL)
	if _, err := verifyEngineBinary(candidate.EnginePath, tag, 30*time.Second); err != nil {
		return fmt.Errorf("verify update engine: %w", err)
	}
	args := []string{
		"update",
		"--target", target,
		"--source-dir", candidate.SourceDir,
		"--payload-dir", candidate.PayloadDir,
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}
	if opts.Force {
		args = append(args, "--force")
	}
	// #nosec G204 -- re-execs the located engine binary with fixed migration args
	cmd := exec.CommandContext(ctx, candidate.EnginePath, args...)
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	cmd.Env = replaceEnv(os.Environ(), map[string]string{
		"DEVRITES_ENGINE_CLI":     candidate.EnginePath,
		"DEVRITES_UPDATE_HANDOFF": "1",
	})
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apply update %s: %w", tag, err)
	}
	return nil
}

func replaceEnv(env []string, replacements map[string]string) []string {
	out := make([]string, 0, len(env)+len(replacements))
	for _, entry := range env {
		name, _, found := strings.Cut(entry, "=")
		if found {
			if _, replace := replacements[name]; replace {
				continue
			}
		}
		out = append(out, entry)
	}
	for name, value := range replacements {
		out = append(out, name+"="+value)
	}
	return out
}
