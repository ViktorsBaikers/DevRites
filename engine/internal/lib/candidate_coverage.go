package lib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
)

// CandidateOmissions lists the project paths the feature's own
// `WIP(<slug>):` checkpoint commits changed (net, feature base to HEAD) that
// touched-files.md lists neither present nor deleted. The base is the parent
// of the oldest such commit, or the empty tree for a root commit. A project
// outside Git, or with no checkpoint yet, has nothing to compare.
func CandidateOmissions(root, slug string) ([]string, error) {
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return nil, err
	}
	raw, err := readBoundedRegularFile(filepath.Join(workspace, "touched-files.md"), maxCandidateManifestBytes)
	if err != nil {
		return nil, fmt.Errorf("candidate manifest: %w", err)
	}
	rows, err := parseCandidateManifest(raw)
	if err != nil {
		return nil, fmt.Errorf("candidate manifest: %w", err)
	}
	project := projectDir(root)
	if _, err := runGitCommand(project, nil, "rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		return nil, nil
	}
	log, err := runGitCommand(project, nil, "log", "--fixed-strings", "--grep=WIP("+slug+"):", "--format=%H%x1f%P%x1f%s", "HEAD")
	if err != nil {
		return nil, err
	}
	var commits []string
	base := ""
	for _, line := range strings.Split(strings.TrimSpace(string(log)), "\n") {
		fields := strings.SplitN(line, "\x1f", 3)
		if len(fields) != 3 || !strings.HasPrefix(fields[2], "WIP("+slug+"):") {
			continue
		}
		commits = append(commits, fields[0])
		base = "" // log lists newest first, so the oldest commit's parent wins
		if parents := strings.Fields(fields[1]); len(parents) > 0 {
			base = parents[0]
		}
	}
	if len(commits) == 0 {
		return nil, nil
	}
	if base == "" {
		tree, err := runGitCommandIO(project, nil, []byte{}, nil, "hash-object", "-t", "tree", "--stdin")
		if err != nil {
			return nil, err
		}
		base = strings.TrimSpace(string(tree))
	}
	touched := map[string]bool{}
	for _, commit := range commits {
		names, err := runGitCommand(project, nil, "diff-tree", "-r", "--root", "--no-commit-id", "--name-only", "--no-renames", "--relative", "-z", commit)
		if err != nil {
			return nil, err
		}
		for _, name := range strings.Split(string(names), "\x00") {
			if name != "" {
				touched[name] = true
			}
		}
	}
	net, err := runGitCommand(project, nil, "diff", "--name-only", "--no-renames", "--relative", "-z", base, "HEAD")
	if err != nil {
		return nil, err
	}
	listed := make(map[string]bool, len(rows))
	for _, row := range rows {
		listed[row.path] = true
	}
	var omitted []string
	for _, name := range strings.Split(string(net), "\x00") {
		if name != "" && touched[name] && !listed[name] && !forbiddenCandidateStatePath(name) {
			omitted = append(omitted, name)
		}
	}
	sort.Strings(omitted)
	return omitted, nil
}

// CheckCandidateCoverage fails when touched-files.md omits a path the
// feature's checkpoints changed.
func CheckCandidateCoverage(root, slug string) error {
	omitted, err := CandidateOmissions(root, slug)
	if err != nil {
		return err
	}
	if len(omitted) > 0 {
		return omissionError(omitted)
	}
	return nil
}

// omissionError renders a bounded BLOCKED reason for omitted candidate paths.
func omissionError(omitted []string) error {
	shown := omitted
	if len(shown) > 10 {
		shown = shown[:10]
	}
	more := ""
	if len(omitted) > len(shown) {
		more = ", …"
	}
	return fmt.Errorf("candidate manifest omits %d path(s) the feature's WIP checkpoints changed: %s%s; add each to touched-files.md (present or deleted) before freezing the candidate",
		len(omitted), strings.Join(shown, ", "), more)
}

var diffPathRE = regexp.MustCompile(`[A-Za-z0-9_./@+-]+\.(?:diff|patch)\b`)

// lintReviewCandidate gates a Review or Seal reviewer wave: the candidate
// manifest must cover every path the feature's checkpoints changed, and the
// packet must name the frozen candidate diff the reviewers judge. A digest
// alone, or a manifest that drops changed files, voids every reviewer at once.
func lintReviewCandidate(root, slug, packet string) error {
	if err := CheckCandidateCoverage(root, slug); err != nil {
		return err
	}
	return lintPacketDiffs(root, slug, packet, true)
}
