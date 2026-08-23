package parallel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	StatusRunning         = "running"
	StatusAborted         = "aborted"
	StatusIntegrateFailed = "integrate-failed"
	StatusComplete        = "complete"

	WrightPending = "pending"
	WrightGreen   = "green"
	WrightRed     = "red"
	WrightGap     = "gap"
)

var (
	slugRE      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	batchRE     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	sliceRE     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	shaRE       = regexp.MustCompile(`(?i)^[0-9a-f]{7,40}$`)
	jsonFenceRE = regexp.MustCompile("(?s)```json\\n(.*?)\\n```")
)

// Lease is the machine-readable parallel batch lease (control tree only).
type Lease struct {
	BatchID             string       `json:"batch_id"`
	CreatedAt           string       `json:"created_at"`
	BaseSHA             string       `json:"base_sha"`
	N                   int          `json:"n"`
	Status              string       `json:"status"`
	ControlPIDOrSession string       `json:"control_pid_or_session"`
	Slices              []LeaseSlice `json:"slices"`
}

// LeaseSlice records one worker worktree under the lease.
type LeaseSlice struct {
	ID             string   `json:"id"`
	Paths          []string `json:"paths"`
	WorktreePath   string   `json:"worktree_path"`
	Branch         string   `json:"branch"`
	WrightStatus   string   `json:"wright_status"`
	TransferCommit string   `json:"transfer_commit,omitempty"`
}

func LeasePath(repoRoot, slug string) (string, error) {
	if err := validateSlug(slug); err != nil {
		return "", err
	}
	return filepath.Join(repoRoot, ".devrites", "work", slug, "parallel-lease.md"), nil
}

func ScratchRoot(repoRoot string) string {
	return filepath.Join(repoRoot, ".scratch", "parallel-wt")
}

func WorkerBranch(slug, batchID, sliceID string) string {
	return fmt.Sprintf("devrites/parallel/%s/%s/%s", slug, batchID, sliceID)
}

func IntegrateBranchName(slug, batchID string) string {
	return fmt.Sprintf("devrites/parallel/%s/%s/integrate", slug, batchID)
}

func WorkerWorktreePath(repoRoot, batchID, sliceID string) string {
	return filepath.Join(ScratchRoot(repoRoot), batchID, sliceID)
}

func validateSlug(slug string) error {
	if !slugRE.MatchString(slug) {
		return fmt.Errorf("invalid slug %q", slug)
	}
	return nil
}

func validateBatchID(batchID string) error {
	if !batchRE.MatchString(batchID) {
		return fmt.Errorf("invalid batch id %q", batchID)
	}
	return nil
}

func validateSliceID(id string) error {
	if !sliceRE.MatchString(id) {
		return fmt.Errorf("invalid slice id %q", id)
	}
	return nil
}

func ValidateLease(lease *Lease) error {
	if lease == nil {
		return fmt.Errorf("lease is nil")
	}
	if err := validateBatchID(lease.BatchID); err != nil {
		return err
	}
	if lease.CreatedAt == "" {
		return fmt.Errorf("lease missing created_at")
	}
	if !shaRE.MatchString(lease.BaseSHA) {
		return fmt.Errorf("invalid base_sha %q", lease.BaseSHA)
	}
	switch lease.Status {
	case StatusRunning, StatusAborted, StatusIntegrateFailed, StatusComplete:
	default:
		return fmt.Errorf("invalid lease status %q", lease.Status)
	}
	if len(lease.Slices) < 2 || len(lease.Slices) > 3 {
		return fmt.Errorf("lease must have 2 or 3 slices (got %d)", len(lease.Slices))
	}
	if lease.N != len(lease.Slices) {
		return fmt.Errorf("lease n=%d does not match slices=%d", lease.N, len(lease.Slices))
	}
	seen := make(map[string]struct{}, len(lease.Slices))
	for i, sl := range lease.Slices {
		if err := validateSliceID(sl.ID); err != nil {
			return fmt.Errorf("slice %d: %w", i, err)
		}
		if _, ok := seen[sl.ID]; ok {
			return fmt.Errorf("duplicate slice id %q", sl.ID)
		}
		seen[sl.ID] = struct{}{}
		if _, err := validateSlicePaths(sl.Paths, fmt.Sprintf("slice %q", sl.ID), ""); err != nil {
			return err
		}
		switch sl.WrightStatus {
		case "", WrightPending, WrightGreen, WrightRed, WrightGap:
		default:
			return fmt.Errorf("slice %q: invalid wright_status %q", sl.ID, sl.WrightStatus)
		}
		if sl.TransferCommit != "" && !shaRE.MatchString(sl.TransferCommit) {
			return fmt.Errorf("slice %q: invalid transfer_commit %q", sl.ID, sl.TransferCommit)
		}
	}
	return nil
}

func EmitLeaseMarkdown(lease *Lease) (string, error) {
	if err := ValidateLease(lease); err != nil {
		return "", err
	}
	payload, err := json.MarshalIndent(lease, "", "  ")
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("# parallel-lease.md — advisory control lease for /rite-build --parallel\n")
	b.WriteString("# Machine block below is authoritative for helpers; YAML mirrors schema.\n\n")
	fmt.Fprintf(&b, "batch_id: %s\n", yamlQuote(lease.BatchID))
	fmt.Fprintf(&b, "created_at: %s\n", yamlQuote(lease.CreatedAt))
	fmt.Fprintf(&b, "base_sha: %s\n", yamlQuote(lease.BaseSHA))
	fmt.Fprintf(&b, "n: %d\n", lease.N)
	fmt.Fprintf(&b, "status: %s\n", yamlQuote(lease.Status))
	fmt.Fprintf(&b, "control_pid_or_session: %s\n", yamlQuote(lease.ControlPIDOrSession))
	b.WriteString("slices:\n")
	for _, sl := range lease.Slices {
		fmt.Fprintf(&b, "  - id: %s\n", yamlQuote(sl.ID))
		if len(sl.Paths) == 0 {
			b.WriteString("    paths: []\n")
		} else {
			b.WriteString("    paths:\n")
			for _, p := range sl.Paths {
				fmt.Fprintf(&b, "      - %s\n", yamlQuote(p))
			}
		}
		fmt.Fprintf(&b, "    worktree_path: %s\n", yamlQuote(sl.WorktreePath))
		fmt.Fprintf(&b, "    branch: %s\n", yamlQuote(sl.Branch))
		status := sl.WrightStatus
		if status == "" {
			status = WrightPending
		}
		fmt.Fprintf(&b, "    wright_status: %s\n", yamlQuote(status))
		if sl.TransferCommit != "" {
			fmt.Fprintf(&b, "    transfer_commit: %s\n", yamlQuote(sl.TransferCommit))
		}
	}
	b.WriteString("\n<!-- machine-readable lease SSOT -->\n")
	b.WriteString("```json\n")
	b.Write(payload)
	b.WriteString("\n```\n")
	return b.String(), nil
}

func yamlQuote(v string) string {
	if v == "" || strings.ContainsAny(v, ":#{}[]&*!|>'\"%@`") || strings.TrimSpace(v) != v {
		raw, _ := json.Marshal(v)
		return string(raw)
	}
	return v
}

func WriteLease(path string, lease *Lease) error {
	text, err := EmitLeaseMarkdown(lease)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(text), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func ReadLease(path string) (*Lease, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	match := jsonFenceRE.FindSubmatch(data)
	if match == nil {
		return nil, fmt.Errorf("lease missing machine-readable json fence: %s", path)
	}
	var lease Lease
	if err := json.Unmarshal(match[1], &lease); err != nil {
		return nil, fmt.Errorf("lease json invalid: %w", err)
	}
	if err := ValidateLease(&lease); err != nil {
		return nil, err
	}
	return &lease, nil
}

func ClearLease(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func NowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
