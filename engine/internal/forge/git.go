package forge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type repoIdentity struct {
	root      string
	commonDir string
	branch    string
	head      string
	tree      string
	treeHash  string
}

func inspectRepo(root string) (repoIdentity, error) {
	top, err := gitText(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return repoIdentity{}, err
	}
	top, err = physicalExisting(top)
	if err != nil {
		return repoIdentity{}, err
	}
	root, err = physicalExisting(root)
	if err != nil {
		return repoIdentity{}, err
	}
	if root != top {
		return repoIdentity{}, fmt.Errorf("forge: ambiguous repository root: got %s want %s", root, top)
	}
	common, err := gitText(root, "rev-parse", "--git-common-dir")
	if err != nil {
		return repoIdentity{}, err
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(root, common)
	}
	common, err = physicalExisting(common)
	if err != nil {
		return repoIdentity{}, err
	}
	branch, err := gitText(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil || branch == "" {
		return repoIdentity{}, errors.New("forge: detached or ambiguous primary branch")
	}
	head, err := gitText(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return repoIdentity{}, err
	}
	tree, err := gitText(root, "rev-parse", "--verify", "HEAD^{tree}")
	if err != nil {
		return repoIdentity{}, err
	}
	listing, err := gitBytes(root, nil, "ls-tree", "-r", "-z", "--full-tree", "HEAD")
	if err != nil {
		return repoIdentity{}, err
	}
	sum := sha256.Sum256(listing)
	return repoIdentity{
		root:      root,
		commonDir: common,
		branch:    branch,
		head:      head,
		tree:      tree,
		treeHash:  hex.EncodeToString(sum[:]),
	}, nil
}

func validatePrimary(m *Manifest, allowMerged bool) (repoIdentity, error) {
	id, err := inspectRepo(m.PrimaryRoot)
	if err != nil {
		return repoIdentity{}, err
	}
	if id.commonDir != m.GitCommonDir || id.branch != m.Primary.Branch {
		return repoIdentity{}, errors.New("forge: primary repository identity drifted")
	}
	wantHead := m.Primary.BaseCommit
	wantTree := m.Primary.BaselineTree
	if allowMerged && m.Merge.State == "landed" {
		wantHead, wantTree = m.Merge.Commit, m.Merge.Tree
	}
	if id.head != wantHead || id.tree != wantTree {
		return repoIdentity{}, fmt.Errorf("forge: primary baseline drifted: head=%s tree=%s", id.head, id.tree)
	}
	manifestFiles, err := validatedManifestPaths(m.PrimaryRoot)
	if err != nil {
		return repoIdentity{}, err
	}
	if dirty, err := dirtyStatus(m.PrimaryRoot, false, manifestFiles); err != nil {
		return repoIdentity{}, err
	} else if dirty {
		return repoIdentity{}, errors.New("forge: primary index or worktree is dirty")
	}
	return id, nil
}

func validateCandidateRepo(m *Manifest, c *Candidate) error {
	physical, err := physicalExisting(c.Worktree)
	if err != nil {
		return err
	}
	if physical != c.Worktree {
		return fmt.Errorf("forge: candidate %s worktree path drifted", c.ID)
	}
	id, err := inspectRepo(c.Worktree)
	if err != nil {
		return err
	}
	if id.root != c.Worktree || id.commonDir != m.GitCommonDir || id.branch != c.Branch {
		return fmt.Errorf("forge: candidate %s repository identity drifted", c.ID)
	}
	head, err := gitText(m.PrimaryRoot, "rev-parse", "--verify", "refs/heads/"+c.Branch+"^{commit}")
	if err != nil {
		return err
	}
	if head != id.head {
		return fmt.Errorf("forge: candidate %s branch does not pin its worktree HEAD", c.ID)
	}
	return nil
}

func dirtyStatus(root string, includeIgnored bool, excludeFiles []string) (bool, error) {
	args := []string{"status", "--porcelain=v1", "-z", "--untracked-files=all"}
	if includeIgnored {
		args = append(args, "--ignored=matching")
	}
	args = append(args, "--", ".")
	for _, path := range excludeFiles {
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return false, fmt.Errorf("forge: manifest exclusion escapes root: %s", path)
		}
		args = append(args, ":(exclude,top)"+filepath.ToSlash(rel))
	}
	out, err := gitBytes(root, nil, args...)
	return len(out) != 0, err
}

func inProgress(common string) bool {
	for _, rel := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "BISECT_LOG", "rebase-apply", "rebase-merge"} {
		if _, err := os.Stat(filepath.Join(common, rel)); err == nil {
			return true
		}
	}
	return false
}

func registeredWorktrees(root string) (map[string]string, error) {
	out, err := gitBytes(root, nil, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	var current string
	for _, field := range bytes.Split(out, []byte{0}) {
		switch {
		case bytes.HasPrefix(field, []byte("worktree ")):
			current = string(bytes.TrimPrefix(field, []byte("worktree ")))
			if physical, err := physicalExisting(current); err == nil {
				current = physical
			} else {
				current = filepath.Clean(current)
			}
			result[current] = ""
		case bytes.HasPrefix(field, []byte("branch ")) && current != "":
			result[current] = strings.TrimPrefix(string(field), "branch refs/heads/")
		}
	}
	return result, nil
}

func gitText(root string, args ...string) (string, error) {
	out, err := gitBytes(root, nil, args...)
	return strings.TrimSpace(string(out)), err
}

func gitBytes(root string, stdin []byte, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmdArgs := append([]string{"-C", root}, args...)
	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_NOSYSTEM=1",
		"LC_ALL=C",
	)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("forge: git %s timed out", strings.Join(args, " "))
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("forge: git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.Bytes(), nil
}

type liveness int

const (
	livenessUnknown liveness = iota
	livenessLive
	livenessDead
)

// ProcessStartToken returns a stable token for the current incarnation of pid.
// Callers record it with the worker PID so PID reuse is distinguishable.
func ProcessStartToken(pid int) (string, error) {
	if pid <= 0 {
		return "", errors.New("forge: PID must be positive")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ps", "-o", "lstart=,command=", "-p", strconv.Itoa(pid))
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return "", fmt.Errorf("forge: cannot prove process start for PID %d", pid)
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(string(out))))
	return hex.EncodeToString(sum[:]), nil
}

func workerLiveness(w Worker) (liveness, string) {
	if w.PID <= 0 || w.ProcessStart == "" {
		return livenessUnknown, "worker liveness proof is missing"
	}
	token, err := ProcessStartToken(w.PID)
	if err != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "ps", "-p", strconv.Itoa(w.PID))
		if runErr := cmd.Run(); runErr != nil {
			return livenessDead, ""
		}
		return livenessUnknown, "worker process exists but its start token is unverifiable"
	}
	if token != w.ProcessStart {
		return livenessDead, ""
	}
	return livenessLive, "worker process is still live"
}

func sha256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
