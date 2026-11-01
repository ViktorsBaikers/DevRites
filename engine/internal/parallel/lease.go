package parallel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/fsutil"
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
	// WrightIntegrated marks a slice a per-slice integrate already landed on
	// control; it no longer holds a parallel slot.
	WrightIntegrated = "integrated"

	MinParallelSlices = 1
	MaxParallelSlices = 10
)

var (
	slugRE      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	batchRE     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	sliceRE     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	shaRE       = regexp.MustCompile(`(?i)^[0-9a-f]{7,40}$`)
	jsonFenceRE = regexp.MustCompile("(?s)```json\\n(.*?)\\n```")
)

// Lease is the machine-readable parallel batch lease (control tree only).
// ControlTip is control HEAD after this lease's most recent per-slice landing;
// empty means BaseSHA (nothing landed yet).
type Lease struct {
	BatchID             string       `json:"batch_id"`
	CreatedAt           string       `json:"created_at"`
	BaseSHA             string       `json:"base_sha"`
	ControlTip          string       `json:"control_tip,omitempty"`
	ControlCheckout     string       `json:"control_checkout,omitempty"`
	ControlRef          string       `json:"control_ref,omitempty"`
	PendingLanding      *landing     `json:"pending_landing,omitempty"`
	LastLanding         *landing     `json:"last_landing,omitempty"`
	N                   int          `json:"n"`
	Capacity            int          `json:"capacity,omitempty"`
	Status              string       `json:"status"`
	ControlPIDOrSession string       `json:"control_pid_or_session"`
	Slices              []LeaseSlice `json:"slices"`
}

// LeaseSlice records one worker worktree under the lease. BaseSHA is the
// commit its worktree was created from; empty means Lease.BaseSHA (a slice
// from the original create).
type LeaseSlice struct {
	ID             string   `json:"id"`
	Workspace      string   `json:"workspace,omitempty"`
	Paths          []string `json:"paths"`
	WorktreePath   string   `json:"worktree_path"`
	Branch         string   `json:"branch"`
	WrightStatus   string   `json:"wright_status"`
	TransferCommit string   `json:"transfer_commit,omitempty"`
	BaseSHA        string   `json:"base_sha,omitempty"`
}

// expectedControl is where control must sit: the tip of this lease's latest
// landing, or the batch base before any landing.
func expectedControl(lease *Lease) string {
	if lease.ControlTip != "" {
		return lease.ControlTip
	}
	return lease.BaseSHA
}

// sliceBase is the commit sl's worktree was created from.
func sliceBase(lease *Lease, sl LeaseSlice) string {
	if sl.BaseSHA != "" {
		return sl.BaseSHA
	}
	return lease.BaseSHA
}

// activeSlices returns the slices that still hold a parallel slot.
func activeSlices(lease *Lease) []SlicePaths {
	var active []SlicePaths
	for _, sl := range lease.Slices {
		if sl.WrightStatus != WrightIntegrated {
			active = append(active, SlicePaths{ID: sl.ID, Workspace: sl.Workspace, Paths: sl.Paths})
		}
	}
	return active
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

func workerBranch(slug, batchID, workspace, id string) string {
	if workspace != "" {
		return WorkerBranch(slug, batchID, "_workspaces/"+workspace+"/"+id)
	}
	return WorkerBranch(slug, batchID, id)
}

func workerWorktreePath(root, batchID, workspace, id string) string {
	if workspace != "" {
		return filepath.Join(ScratchRoot(root), batchID, "_workspaces", workspace, id)
	}
	return WorkerWorktreePath(root, batchID, id)
}

func leaseCapacity(lease *Lease) int {
	if lease.Capacity == 0 {
		return MaxParallelSlices
	}
	return lease.Capacity
}

func validateCapacity(capacity int) error {
	if capacity < 0 || capacity > MaxParallelSlices {
		return fmt.Errorf("cap must be 1-%d (got %d)", MaxParallelSlices, capacity)
	}
	return nil
}

func normalizeLease(lease *Lease, controller string) error {
	for i := range lease.Slices {
		if lease.Slices[i].Workspace == controller {
			lease.Slices[i].Workspace = ""
		}
	}
	return ValidateLease(lease)
}

func normalizeSliceRef(controller, ref string) (string, error) {
	workspace, id, qualified := strings.Cut(ref, "/")
	if !qualified {
		return ref, validateSliceID(ref)
	}
	if err := validateSlug(workspace); err != nil {
		return "", err
	}
	if err := validateSliceID(id); err != nil {
		return "", err
	}
	if workspace == controller {
		workspace = ""
	}
	return SliceKey(workspace, id), nil
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

func checkParallelSliceCount(n int) error {
	if n < MinParallelSlices || n > MaxParallelSlices {
		return fmt.Errorf("parallel requires %d-%d slices (got %d)", MinParallelSlices, MaxParallelSlices, n)
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
	if lease.ControlTip != "" && !shaRE.MatchString(lease.ControlTip) {
		return fmt.Errorf("invalid control_tip %q", lease.ControlTip)
	}
	// A rolling lease grows past the cap as landed slices free their slots;
	// only the slices still active are bounded.
	if len(lease.Slices) < MinParallelSlices {
		return checkParallelSliceCount(len(lease.Slices))
	}
	if err := validateCapacity(lease.Capacity); err != nil {
		return err
	}
	if active := len(activeSlices(lease)); active > leaseCapacity(lease) {
		return fmt.Errorf("parallel requires %d-%d slices active at once (got %d)", MinParallelSlices, leaseCapacity(lease), active)
	}
	if lease.N != len(lease.Slices) {
		return fmt.Errorf("lease n=%d does not match slices=%d", lease.N, len(lease.Slices))
	}
	seen := make(map[string]struct{}, len(lease.Slices))
	for i, sl := range lease.Slices {
		if err := validateSliceID(sl.ID); err != nil {
			return fmt.Errorf("slice %d: %w", i, err)
		}
		if sl.Workspace != "" {
			if err := validateSlug(sl.Workspace); err != nil {
				return fmt.Errorf("slice %d: %w", i, err)
			}
		}
		key := SliceKey(sl.Workspace, sl.ID)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate slice id %q", key)
		}
		seen[key] = struct{}{}
		if _, err := validateSlicePaths(sl.Paths, fmt.Sprintf("slice %q", key), ""); err != nil {
			return err
		}
		switch sl.WrightStatus {
		case "", WrightPending, WrightGreen, WrightRed, WrightGap, WrightIntegrated:
		default:
			return fmt.Errorf("slice %q: invalid wright_status %q", key, sl.WrightStatus)
		}
		if sl.TransferCommit != "" && !shaRE.MatchString(sl.TransferCommit) {
			return fmt.Errorf("slice %q: invalid transfer_commit %q", key, sl.TransferCommit)
		}
		if sl.BaseSHA != "" && !shaRE.MatchString(sl.BaseSHA) {
			return fmt.Errorf("slice %q: invalid base_sha %q", key, sl.BaseSHA)
		}
	}
	if err := validateLanding(lease, lease.PendingLanding, false); err != nil {
		return err
	}
	if err := validateLanding(lease, lease.LastLanding, true); err != nil {
		return err
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
	if lease.ControlTip != "" {
		fmt.Fprintf(&b, "control_tip: %s\n", yamlQuote(lease.ControlTip))
	}
	fmt.Fprintf(&b, "n: %d\n", lease.N)
	if lease.Capacity != 0 {
		fmt.Fprintf(&b, "capacity: %d\n", lease.Capacity)
	}
	fmt.Fprintf(&b, "status: %s\n", yamlQuote(lease.Status))
	fmt.Fprintf(&b, "control_pid_or_session: %s\n", yamlQuote(lease.ControlPIDOrSession))
	b.WriteString("slices:\n")
	for _, sl := range lease.Slices {
		fmt.Fprintf(&b, "  - id: %s\n", yamlQuote(sl.ID))
		if sl.Workspace != "" {
			fmt.Fprintf(&b, "    workspace: %s\n", yamlQuote(sl.Workspace))
		}
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
		if sl.BaseSHA != "" {
			fmt.Fprintf(&b, "    base_sha: %s\n", yamlQuote(sl.BaseSHA))
		}
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
	return fsutil.WriteFileAtomic(path, []byte(text), 0o644)
}

func ReadLease(path string) (*Lease, error) {
	// #nosec G304 -- lease file under the engine-managed work dir
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
	if lease, err := ReadLease(path); err == nil && lease.PendingLanding != nil {
		return fmt.Errorf("lease-clear refuses pending landing; recover with integrate --apply-to-control or abort")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func NowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
