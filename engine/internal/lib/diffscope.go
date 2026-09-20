package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const diffScopeUsage = `usage: devrites-engine check diff-scope <slug> --allow <csv> | --allow-file <path> [--worktree|--staged|--base <ref>] [--allow-empty] [--cwd <dir>]

Verifies the changed-path set stays inside the declared allowlist. Modes:
  --worktree  git status --porcelain (modified + staged + untracked) [default]
  --staged    git diff --name-only --cached
  --base ref  git diff --name-only <ref> (tracked only)
Allowlist entries match exactly or as directory prefixes ("dir" covers "dir/...").
Exit 0 clean, 3 a path escaped the allowlist or the diff is empty without --allow-empty.`

// RunCheckDiffScope is the mechanical post-return contract gate: a writer's
// touched paths must be a subset of its declared allowlist before any reviewer
// spends context on the diff.
func RunCheckDiffScope(root string, args []string, stdout, stderr io.Writer) int {
	opts, code := parseDiffScopeArgs(root, args, stderr)
	if code != 0 {
		return code
	}
	if len(opts.allow) == 0 {
		fmt.Fprintln(stderr, "diff-scope: an allowlist is required (--allow or --allow-file)")
		return 2
	}
	changed, err := opts.changedPaths()
	if err != nil {
		fmt.Fprintf(stderr, "diff-scope: %v\n", err)
		return 2
	}
	if len(changed) == 0 {
		if opts.allowEmpty {
			fmt.Fprintln(stdout, "diff-scope: empty (0 changed paths; --allow-empty)")
			return 0
		}
		fmt.Fprintln(stdout, "diff-scope: BLOCKED: empty diff; nothing was written (pass --allow-empty to accept)")
		return 3
	}
	var violations []string
	for _, path := range changed {
		if !allowedPath(opts.allow, path) {
			violations = append(violations, path)
		}
	}
	if len(violations) > 0 {
		for _, path := range violations {
			fmt.Fprintf(stdout, "diff-scope: BLOCKED: %s outside allowlist\n", path)
		}
		recordMetric(root, opts.slug, opts.phase, "diff-scope", "", 0)
		return 3
	}
	fmt.Fprintf(stdout, "diff-scope: ok (%d changed path(s) within allowlist)\n", len(changed))
	recordMetric(root, opts.slug, opts.phase, "diff-scope", "", int64(len(changed)))
	return 0
}

type diffScopeOpts struct {
	slug       string
	phase      string
	mode       string // worktree | staged | base
	base       string
	allow      []string
	allowFile  []string
	allowEmpty bool
	cwd        string
	root       string
}

func parseDiffScopeArgs(root string, args []string, stderr io.Writer) (diffScopeOpts, int) {
	opts := diffScopeOpts{mode: "worktree", phase: "build", root: root}
	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--phase":
			opts.phase = argAt(args, i+1)
			i++
		case "--allow":
			for _, entry := range strings.Split(argAt(args, i+1), ",") {
				if entry = strings.TrimSpace(entry); entry != "" {
					opts.allow = append(opts.allow, entry)
				}
			}
			i++
		case "--allow-file":
			opts.allowFile = append(opts.allowFile, argAt(args, i+1))
			i++
		case "--staged":
			opts.mode = "staged"
		case "--worktree":
			opts.mode = "worktree"
		case "--base":
			opts.mode = "base"
			opts.base = argAt(args, i+1)
			i++
		case "--allow-empty":
			opts.allowEmpty = true
		case "--cwd":
			opts.cwd = argAt(args, i+1)
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "diff-scope: unknown flag %q\n", args[i])
				return opts, 2
			}
			positional = append(positional, args[i])
		}
	}
	if len(positional) != 1 || opts.mode == "base" && opts.base == "" {
		fmt.Fprint(stderr, diffScopeUsage)
		return opts, 2
	}
	opts.slug = positional[0]
	for _, file := range opts.allowFile {
		path := file
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, filepath.FromSlash(file))
		}
		data, err := os.ReadFile(path) // #nosec G304 -- path is rooted under the operator repository
		if err != nil {
			fmt.Fprintf(stderr, "diff-scope: allow-file %s: %v\n", file, err)
			return opts, 2
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			opts.allow = append(opts.allow, line)
		}
	}
	return opts, 0
}

func (o diffScopeOpts) changedPaths() ([]string, error) {
	dir := o.cwd
	if dir == "" {
		dir = o.root
	}
	switch o.mode {
	case "staged":
		return gitDiffNames(dir, "--cached")
	case "base":
		return gitDiffNames(dir, o.base)
	default:
		return gitStatusPaths(dir)
	}
}

// gitStatusPaths lists every path the writer touched: tracked modifications,
// staged entries, renames (new name), and untracked files.
func gitStatusPaths(dir string) ([]string, error) {
	out, err := runGitCommand(dir, nil, "status", "--porcelain", "-uall")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, line := range splitLinesNoTrailing(out) {
		if len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if idx := strings.LastIndex(path, " -> "); idx >= 0 {
			path = path[idx+4:]
		}
		path = strings.Trim(path, `"`)
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

// allowedPath reports whether path is inside the allowlist. An entry matches
// the exact path or, as a directory prefix, everything beneath it.
func allowedPath(allow []string, path string) bool {
	for _, entry := range allow {
		entry = strings.TrimSuffix(entry, "/")
		if path == entry || strings.HasPrefix(path, entry+"/") {
			return true
		}
	}
	return false
}
