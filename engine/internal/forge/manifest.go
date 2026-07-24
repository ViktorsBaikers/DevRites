package forge

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const Schema = "devrites-forge/v1"

var (
	safeSlug  = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	safeRunID = regexp.MustCompile(`^forge-[a-f0-9]{24}$`)
)

type CandidateID string

const (
	CandidateA CandidateID = "A"
	CandidateB CandidateID = "B"
	CandidateC CandidateID = "C"
)

type CandidateState string

const (
	StatePlanned   CandidateState = "planned"
	StateCreated   CandidateState = "created"
	StateRunning   CandidateState = "running"
	StateFinished  CandidateState = "finished"
	StateFailed    CandidateState = "failed"
	StateExtracted CandidateState = "extracted"
	StateMerged    CandidateState = "merged"
	StateCleaned   CandidateState = "cleaned"
)

type Primary struct {
	Branch         string `json:"branch"`
	BaseCommit     string `json:"base_commit"`
	BaselineTree   string `json:"baseline_tree"`
	BaselineSHA256 string `json:"baseline_sha256"`
}

type Worker struct {
	ID           string `json:"id,omitempty"`
	PID          int    `json:"pid,omitempty"`
	ProcessStart string `json:"process_start,omitempty"`
	StartedAt    string `json:"started_at,omitempty"`
	FinishedAt   string `json:"finished_at,omitempty"`
	Result       string `json:"result,omitempty"`
}

type Candidate struct {
	ID             CandidateID    `json:"id"`
	Strategy       string         `json:"strategy"`
	Worktree       string         `json:"worktree"`
	Branch         string         `json:"branch"`
	InitialBase    string         `json:"initial_base"`
	State          CandidateState `json:"state"`
	Worker         Worker         `json:"worker"`
	Commit         string         `json:"commit,omitempty"`
	Tree           string         `json:"tree,omitempty"`
	DeltaSHA256    string         `json:"delta_sha256,omitempty"`
	Preservation   string         `json:"preservation,omitempty"`
	LastTransition string         `json:"last_transition"`
}

type Winner struct {
	Candidate  CandidateID `json:"candidate,omitempty"`
	RecordedBy string      `json:"recorded_by,omitempty"`
	RecordedAt string      `json:"recorded_at,omitempty"`
}

type MergeState struct {
	State      string `json:"state,omitempty"`
	Commit     string `json:"commit,omitempty"`
	Tree       string `json:"tree,omitempty"`
	RecordedAt string `json:"recorded_at,omitempty"`
}

type VerificationState struct {
	State      string `json:"state,omitempty"`
	RecordedBy string `json:"recorded_by,omitempty"`
	RecordedAt string `json:"recorded_at,omitempty"`
}

type CleanupState struct {
	State      string            `json:"state,omitempty"`
	Preserved  map[string]string `json:"preserved,omitempty"`
	RecordedAt string            `json:"recorded_at,omitempty"`
}

type Manifest struct {
	Schema         string            `json:"schema"`
	RunID          string            `json:"run_id"`
	CreationNonce  string            `json:"creation_nonce"`
	CreatedAt      string            `json:"created_at"`
	FeatureSlug    string            `json:"feature_slug"`
	SliceID        string            `json:"slice_id"`
	AcceptanceHash string            `json:"acceptance_hash"`
	TestPlanHash   string            `json:"test_plan_hash"`
	PrimaryRoot    string            `json:"primary_root"`
	GitCommonDir   string            `json:"git_common_dir"`
	ForgeRoot      string            `json:"forge_root"`
	Primary        Primary           `json:"primary"`
	Candidates     []Candidate       `json:"candidates"`
	Winner         Winner            `json:"winner"`
	Merge          MergeState        `json:"merge"`
	Verification   VerificationState `json:"verification"`
	Cleanup        CleanupState      `json:"cleanup"`
}

func (m *Manifest) Candidate(id CandidateID) (*Candidate, error) {
	for i := range m.Candidates {
		if m.Candidates[i].ID == id {
			return &m.Candidates[i], nil
		}
	}
	return nil, fmt.Errorf("forge: candidate %q is not in manifest", id)
}

func ManifestPath(root, slug, runID string) (string, error) {
	if !safeSlug.MatchString(slug) {
		return "", fmt.Errorf("forge: invalid feature slug %q", slug)
	}
	if !safeRunID.MatchString(runID) {
		return "", fmt.Errorf("forge: invalid run ID %q", runID)
	}
	root, err := physicalExisting(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".devrites", "work", slug, ".forge", runID, "manifest.json"), nil
}

// Load returns the unique manifest for runID below root. Duplicate manifests
// fail closed instead of letting a caller choose which ownership record wins.
func Load(root, runID string) (*Manifest, string, error) {
	if !safeRunID.MatchString(runID) {
		return nil, "", fmt.Errorf("forge: invalid run ID %q", runID)
	}
	root, err := physicalExisting(root)
	if err != nil {
		return nil, "", err
	}
	paths, err := manifestPaths(root, "")
	if err != nil {
		return nil, "", err
	}
	var matches []string
	for _, path := range paths {
		if filepath.Base(filepath.Dir(path)) == runID {
			matches = append(matches, path)
		}
	}
	if len(matches) != 1 {
		return nil, "", fmt.Errorf("forge: run %s has %d manifests; want exactly one", runID, len(matches))
	}
	m, err := loadAt(root, matches[0])
	if err != nil {
		return nil, "", err
	}
	return m, matches[0], nil
}

func loadAt(root, path string) (*Manifest, error) {
	if err := validateNoSymlink(root, path); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("forge: stat manifest: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("forge: manifest is not a regular file: %s", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("forge: read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("forge: decode manifest: %w", err)
	}
	if err := validateManifest(root, path, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func validateManifest(root, path string, m *Manifest) error {
	if m.Schema != Schema || !safeRunID.MatchString(m.RunID) || !safeSlug.MatchString(m.FeatureSlug) {
		return errors.New("forge: invalid manifest identity")
	}
	wantPath, _ := ManifestPath(root, m.FeatureSlug, m.RunID)
	if filepath.Clean(path) != wantPath {
		return fmt.Errorf("forge: manifest path mismatch: got %s want %s", path, wantPath)
	}
	if m.PrimaryRoot != root {
		return fmt.Errorf("forge: primary root mismatch: got %s want %s", m.PrimaryRoot, root)
	}
	wantForge := forgeRoot(root, m.RunID)
	if m.ForgeRoot != wantForge {
		return fmt.Errorf("forge: staging root mismatch: got %s want %s", m.ForgeRoot, wantForge)
	}
	if len(m.Candidates) < 2 || len(m.Candidates) > 3 {
		return fmt.Errorf("forge: manifest has %d candidates; want 2 or 3", len(m.Candidates))
	}
	seen := map[CandidateID]bool{}
	for _, c := range m.Candidates {
		if seen[c.ID] || !validCandidateID(c.ID) {
			return fmt.Errorf("forge: invalid or duplicate candidate %q", c.ID)
		}
		seen[c.ID] = true
		wantWorktree := filepath.Join(wantForge, "candidate-"+strings.ToLower(string(c.ID)))
		wantBranch := candidateBranch(m.RunID, c.ID)
		if c.Worktree != wantWorktree || c.Branch != wantBranch || c.InitialBase != m.Primary.BaseCommit {
			return fmt.Errorf("forge: candidate %s ownership fields do not match derived values", c.ID)
		}
	}
	return nil
}

func save(path string, m *Manifest) error {
	root := m.PrimaryRoot
	if err := validateManifest(root, path, m); err != nil {
		return err
	}
	if err := validateNoSymlink(root, filepath.Dir(path)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("forge: create manifest directory: %w", err)
	}
	if err := validateNoSymlink(root, filepath.Dir(path)); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("forge: refusing to replace non-regular manifest %s", path)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("forge: stat manifest: %w", err)
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("forge: encode manifest: %w", err)
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".manifest-*.tmp")
	if err != nil {
		return fmt.Errorf("forge: create manifest temp file: %w", err)
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("forge: chmod manifest temp file: %w", err)
	}
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("forge: write manifest: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("forge: sync manifest: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("forge: close manifest: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("forge: replace manifest: %w", err)
	}
	return nil
}

func manifestPaths(root, slug string) ([]string, error) {
	base := filepath.Join(root, ".devrites", "work")
	if err := validateNoSymlink(root, base); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var paths []string
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) == 4 && parts[1] == ".forge" && parts[3] == "manifest.json" && safeSlug.MatchString(parts[0]) && safeRunID.MatchString(parts[2]) {
			if slug == "" || parts[0] == slug {
				paths = append(paths, path)
			}
		}
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	sort.Strings(paths)
	return paths, err
}

func validatedManifestPaths(root string) ([]string, error) {
	paths, err := manifestPaths(root, "")
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		if _, err := loadAt(root, path); err != nil {
			return nil, err
		}
	}
	return paths, nil
}

func newIdentity() (runID, nonce string, err error) {
	raw := make([]byte, 28)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("forge: generate identity: %w", err)
	}
	return "forge-" + hex.EncodeToString(raw[:12]), hex.EncodeToString(raw[12:]), nil
}

func forgeRoot(root, runID string) string {
	return filepath.Join(filepath.Dir(root), "."+filepath.Base(root)+".devrites-forge", runID)
}

func candidateBranch(runID string, id CandidateID) string {
	return "devrites/forge/" + runID + "/" + strings.ToLower(string(id))
}

func validCandidateID(id CandidateID) bool {
	return id == CandidateA || id == CandidateB || id == CandidateC
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func physicalExisting(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("forge: resolve path: %w", err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("forge: resolve physical path %s: %w", abs, err)
	}
	return filepath.Clean(real), nil
}

func validateNoSymlink(root, path string) error {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("forge: path escapes primary root: %s", path)
	}
	current := root
	if rel == "." {
		return nil
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) {
			return fs.ErrNotExist
		}
		if err != nil {
			return fmt.Errorf("forge: inspect path %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("forge: symlink is not allowed in manifest path: %s", current)
		}
	}
	return nil
}
