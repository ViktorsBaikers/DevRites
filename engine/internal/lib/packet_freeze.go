package lib

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
)

// CandidateFreezeUsage is the help and usage-error text for `candidate freeze`.
const CandidateFreezeUsage = `usage: devrites-engine candidate freeze <slug> --wave <w> (--slice <SLICE-ID> | --paths <path>...) [--base <ref>]

Freezes one review candidate. It writes packets/<w>.diff (the path set's
tracked changes against --base, default HEAD, plus each untracked file as a
/dev/null diff) and packets/<w>.sha256 (base commit, diff digest, per-file
sha256, and the aggregate candidate digest under the check candidate domain
and bounds). --slice takes that slice's Writer allowlist (else Files likely
touched) from tasks.md. A wave's frozen candidate is never replaced by a
different one; freeze a changed candidate under a new wave.
`

// packetNameRE bounds a wave, role, phase, or slice used in a packets/ or
// ctx/ file name: one path component, no traversal.
var packetNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// RunCandidateFreeze implements `candidate freeze`: the producer of the frozen
// .diff a Review or Seal reviewer packet must name.
func RunCandidateFreeze(root string, args []string, stdout, stderr io.Writer) int {
	var wave, slice, base string
	var paths, positional []string
	for i := 0; i < len(args); i++ {
		value := argAt(args, i+1)
		switch args[i] {
		case "--wave":
			wave = value
		case "--slice":
			slice = value
		case "--base":
			base = value
		case "--paths":
			for i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				paths = append(paths, args[i])
			}
			continue
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "candidate freeze: unknown flag %q\n", args[i])
				return 2
			}
			positional = append(positional, args[i])
			continue
		}
		i++
	}
	if len(positional) != 1 || !packetNameRE.MatchString(wave) || (slice == "") == (len(paths) == 0) {
		fmt.Fprint(stderr, CandidateFreezeUsage)
		return 2
	}
	slug := positional[0]
	fail := func(err error) int {
		fmt.Fprintf(stderr, "candidate freeze: %v\n", err)
		return 3
	}
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return fail(err)
	}
	if slice != "" {
		raw, err := readWorkspaceArtifact(root, slug, "tasks.md")
		if err != nil {
			return fail(err)
		}
		writes, _ := sliceWritePaths(raw, ParseTaskGraph(raw))
		if paths = writes[slice]; len(paths) == 0 {
			return fail(fmt.Errorf("slice %s names no Writer allowlist or Files likely touched in tasks.md", slice))
		}
	}
	project := projectDir(root)
	rows, err := freezeRows(project, paths)
	if err != nil {
		return fail(err)
	}
	if err := validateCandidateRows(project, rows); err != nil {
		return fail(err)
	}
	diff, baseCommit, err := freezeDiff(project, base, rows)
	if err != nil {
		return fail(err)
	}
	if len(diff) == 0 {
		return fail(errors.New("the path set has no change against the base; there is no candidate to freeze"))
	}
	digest, err := hashCandidate(project, rows)
	if err != nil {
		return fail(err)
	}
	var sums strings.Builder
	fmt.Fprintf(&sums, "%s %s\nbase %s\ndiff %x packets/%s.diff\n", candidateDigestDomain, candidateDigestVersion, baseCommit, sha256.Sum256(diff), wave)
	for _, row := range rows {
		if row.state != "present" {
			fmt.Fprintf(&sums, "file absent %s\n", row.path)
			continue
		}
		file := sha256.New()
		if err := hashCandidateFile(file, project, row); err != nil {
			return fail(fmt.Errorf("candidate %q: %w", row.path, err))
		}
		fmt.Fprintf(&sums, "file %x %s\n", file.Sum(nil), row.path)
	}
	fmt.Fprintf(&sums, "aggregate %s\n", digest)

	dir := filepath.Join(workspace, "packets")
	diffPath, sumPath := filepath.Join(dir, wave+".diff"), filepath.Join(dir, wave+".sha256")
	if prev, err := readBoundedRegularFile(sumPath, workspaceArtifactLimit); err == nil && string(prev) != sums.String() {
		return fail(fmt.Errorf("packets/%s.sha256 already freezes a different candidate; a frozen candidate is never replaced, so freeze this one under a new wave", wave))
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fail(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fail(err)
	}
	if err := state.AtomicWrite(diffPath, diff, 0o600); err != nil {
		return fail(err)
	}
	if err := state.AtomicWrite(sumPath, []byte(sums.String()), 0o600); err != nil {
		return fail(err)
	}
	rel := workspaceRel(root, workspace)
	fmt.Fprintf(stdout, "candidate: %s (%d paths, base %s)\ndiff: %s/packets/%s.diff\nsha256: %s/packets/%s.sha256\n", digest, len(rows), baseCommit, rel, wave, rel, wave)
	return 0
}

// freezeRows normalizes, deduplicates, and sorts the path set, marking each
// row present or deleted (absent) as hashCandidate expects.
func freezeRows(project string, paths []string) ([]candidateRow, error) {
	seen := map[string]bool{}
	var rows []candidateRow
	for _, p := range paths {
		name, err := normalizeCandidatePath(p)
		if err != nil {
			return nil, fmt.Errorf("path %q: %w", p, err)
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		_, exists, err := candidatePathInfo(project, name)
		if err != nil {
			return nil, fmt.Errorf("candidate %q: %w", name, err)
		}
		row := candidateRow{state: "deleted", path: name}
		if exists {
			row.state = "present"
		}
		rows = append(rows, row)
	}
	if len(rows) > maxCandidateRows {
		return nil, errors.New("path set exceeds 4096 rows")
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].path < rows[j].path })
	return rows, nil
}

// freezeDiff renders the path set's tracked changes against base, then each
// untracked file as a /dev/null diff, with fixed prefixes so git apply takes it
// whatever the user's diff config.
func freezeDiff(project, base string, rows []candidateRow) ([]byte, string, error) {
	if base == "" {
		base = "HEAD"
	}
	if strings.HasPrefix(base, "-") {
		return nil, "", fmt.Errorf("--base %q is not a revision", base)
	}
	out, err := runGitCommand(project, nil, "rev-parse", "--verify", "--quiet", base+"^{commit}")
	if err != nil {
		return nil, "", fmt.Errorf("--base %q is not a commit: %w", base, err)
	}
	commit := strings.TrimSpace(string(out))
	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = row.path
	}
	flags := []string{"--binary", "--no-color", "--no-ext-diff", "--no-textconv", "--no-renames", "--src-prefix=a/", "--dst-prefix=b/"}
	diff, err := runGitCommand(project, nil, append(append(append([]string{"--literal-pathspecs", "diff"}, flags...), "--relative", commit, "--"), names...)...)
	if err != nil {
		return nil, "", err
	}
	diff = append([]byte(nil), diff...)
	untracked, err := runGitCommand(project, nil, append([]string{"--literal-pathspecs", "ls-files", "--others", "-z", "--"}, names...)...)
	if err != nil {
		return nil, "", err
	}
	for _, name := range strings.Split(string(untracked), "\x00") {
		if name == "" {
			continue
		}
		var part cappedGitOutput
		_, err := runGitCommandIO(project, nil, nil, &part, append(append([]string{"diff", "--no-index"}, flags...), "--", "/dev/null", name)...)
		var gitErr *gitCommandError
		if err != nil && !(errors.As(err, &gitErr) && gitErr.exitCode == 1) {
			return nil, "", err // --no-index exits 1 when the files differ
		}
		if part.truncated || len(diff)+part.Len() > gitOutputLimit {
			return nil, "", errors.New("candidate diff exceeds the git output limit")
		}
		diff = append(diff, part.Bytes()...)
	}
	return diff, commit, nil
}

// workspaceRel is the feature workspace as a slash path relative to the
// project, the form packets hand agents; it falls back to the absolute path.
func workspaceRel(root, workspace string) string {
	if rel, err := filepath.Rel(projectDir(root), workspace); err == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(workspace)
}
