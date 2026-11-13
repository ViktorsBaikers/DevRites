package lib

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/state"
)

type anchorCandidate struct {
	project, digest string
	rows            []candidateRow
	sources         map[string][]byte
}

// Native stored returns bind an original provider return by exact bytes, not its filename.
func resolveAnchorCandidate(root, slug, workspace, wave, role, worktree string, raw []byte) (*anchorCandidate, error) {
	ledger, _, err := loadDispatch(root, slug)
	if err != nil {
		return nil, err
	}
	explicit := wave != "" || role != ""
	if explicit && (!packetNameRE.MatchString(wave) || !packetNameRE.MatchString(role)) {
		return nil, errors.New("--wave and --role must name one review account")
	}
	matches := 0
	hasReviewer := false
	for w, account := range ledger.Waves {
		for r, actor := range account.Roles {
			if !isReviewerRole(r) {
				continue
			}
			hasReviewer = true
			if !packetNameRE.MatchString(w) || !packetNameRE.MatchString(r) {
				return nil, errors.New("invalid native review account name")
			}
			if explicit && (w != wave || r != role) {
				continue
			}
			if account.State == "abandoned" || actor.StartedAt == "" {
				continue
			}
			stored, readErr := readBoundedRegularFile(filepath.Join(workspace, "packets", w+"-"+r+".return.md"), maxReturnBytes)
			if explicit && !actor.Returned && errors.Is(readErr, os.ErrNotExist) {
				matches++
				continue
			}
			if readErr == nil && bytes.Equal(stored, raw) {
				matches++
				wave, role = w, r
			}
		}
	}
	if !hasReviewer {
		entries, err := os.ReadDir(filepath.Join(workspace, "packets"))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".sha256") || strings.HasSuffix(entry.Name(), ".return.md") {
				hasReviewer = true
				break
			}
		}
	}
	if matches == 0 && !hasReviewer && !explicit && worktree == "" {
		return nil, nil
	}
	if matches != 1 {
		return nil, fmt.Errorf("candidate binding requires exactly one native review account (found %d); use --wave and --role for an unstored return", matches)
	}
	packetPath := filepath.Join(workspace, "packets", wave+"."+role+".md")
	packet, err := readBoundedRegularFile(packetPath, maxContractBytes)
	if err != nil {
		return nil, fmt.Errorf("review packet: %w", err)
	}
	candidate, partial, err := frozenCandidate(workspace, wave)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(string(packet), "# Packet "+wave+": "+role+"\n\nFeature: "+slug+"\nPhase: "+ledger.Waves[wave].Phase+"\n") {
		return nil, errors.New("review packet has missing or mismatched account binding")
	}
	if err := lintPacketDiffs(root, slug, packetPath, candidate != ""); err != nil {
		return nil, err
	}
	if candidate == "" {
		if worktree == "" && !strings.Contains(string(packet), candidateDigestDomain) && !diffPathRE.Match(packet) {
			for _, policy := range state.PhasePolicies() {
				if policy.Target == state.PhaseBuild {
					break
				}
				if strings.EqualFold(string(policy.Target), ledger.Waves[wave].Phase) {
					return &anchorCandidate{project: projectDir(root)}, nil
				}
			}
		}
		return nil, errors.New("review packet has missing frozen candidate binding")
	}
	if strings.Count(string(packet), "\n"+candidate+"```\n") != 1 ||
		strings.Count(string(packet), candidateDigestDomain+" "+candidateDigestVersion) != 1 {
		return nil, errors.New("review packet has mismatched frozen candidate binding")
	}
	full, err := readBoundedRegularFile(filepath.Join(workspace, "packets", wave+".sha256"), workspaceArtifactLimit)
	if err != nil {
		return nil, err
	}
	bound := string(full)
	if partial {
		var header strings.Builder
		for _, line := range strings.SplitAfter(bound, "\n") {
			if !strings.HasPrefix(line, "file ") {
				header.WriteString(line)
			}
		}
		bound = header.String()
	}
	if bound != candidate {
		return nil, errors.New("frozen candidate binding changed while reading")
	}
	project := projectDir(root)
	if worktree != "" {
		project, err = checkedWorktree(root, worktree)
		if err != nil {
			return nil, err
		}
	}
	return verifyAnchorCandidate(project, wave, string(full), worktree != "")
}

func verifyAnchorCandidate(project, wave, manifest string, worker bool) (*anchorCandidate, error) {
	lines := strings.Split(strings.TrimSuffix(manifest, "\n"), "\n")
	if len(lines) < 5 || lines[0] != candidateDigestDomain+" "+candidateDigestVersion || !strings.HasPrefix(lines[1], "base ") {
		return nil, errors.New("malformed frozen candidate manifest")
	}
	base := strings.TrimPrefix(lines[1], "base ")
	head := ""
	i := 2
	if strings.HasPrefix(lines[i], "head ") {
		head = strings.TrimPrefix(lines[i], "head ")
		i++
	}
	if worker != (head != "") {
		return nil, errors.New("linked-worktree candidate requires --worktree; same-tree candidate must not select a worker")
	}
	if len(lines) <= i+2 || !strings.HasPrefix(lines[i], "diff ") {
		return nil, errors.New("missing frozen diff binding")
	}
	diffLine := lines[i]
	i++
	var paths []string
	for ; i < len(lines)-1; i++ {
		fields := strings.SplitN(lines[i], " ", 3)
		if len(fields) != 3 || fields[0] != "file" {
			return nil, errors.New("malformed frozen file binding")
		}
		paths = append(paths, fields[2])
	}
	rows, err := freezeRows(project, paths)
	if err != nil {
		return nil, err
	}
	if len(rows) != len(paths) {
		return nil, errors.New("duplicate frozen candidate paths")
	}
	if err := validateCandidateRows(project, rows); err != nil {
		return nil, err
	}
	digest, err := hashCandidate(project, rows, true)
	if err != nil {
		return nil, err
	}
	diff, commit, err := freezeDiff(project, base, rows)
	if err != nil {
		return nil, err
	}
	var expected strings.Builder
	fmt.Fprintf(&expected, "%s %s\nbase %s\n", candidateDigestDomain, candidateDigestVersion, commit)
	if head != "" {
		fmt.Fprintf(&expected, "head %s\n", head)
	}
	fmt.Fprintf(&expected, "diff %x packets/%s.diff\n", sha256.Sum256(diff), wave)
	if !strings.Contains(expected.String(), "\n"+diffLine+"\n") {
		return nil, errors.New("frozen candidate diff or path scope mismatch")
	}
	sources := make(map[string][]byte, len(rows))
	for _, row := range rows {
		identity := "absent"
		if row.state == "present" {
			identity = row.fileDigest
			body, err := readBoundedRegularFile(filepath.Join(project, filepath.FromSlash(row.path)), maxCandidateFileBytes)
			if err != nil {
				return nil, err
			}
			if fmt.Sprintf("%x", sha256.Sum256(body)) != identity {
				return nil, errors.New("candidate source changed while reading")
			}
			sources[row.path] = body
		}
		fmt.Fprintf(&expected, "file %s %s\n", identity, row.path)
	}
	fmt.Fprintf(&expected, "aggregate %s\n", digest)
	if expected.String() != manifest {
		return nil, errors.New("frozen candidate identity, scope or source hashes mismatch")
	}
	if err := revalidateFrozenCandidate(project, base, rows, digest, diff, head); err != nil {
		return nil, err
	}
	return &anchorCandidate{project: project, digest: digest, rows: rows, sources: sources}, nil
}
