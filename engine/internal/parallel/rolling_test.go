package parallel

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const rollSlug, rollBatch = "demo-feature", "batch1"

func runCLI(t *testing.T, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = Run("parallel", args, strings.NewReader(stdin), &out, &errOut)
	return code, out.String(), errOut.String()
}

func readLeaseT(t *testing.T, repo string) *Lease {
	t.Helper()
	path, err := LeasePath(repo, rollSlug)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := ReadLease(path)
	if err != nil {
		t.Fatal(err)
	}
	return lease
}

func sliceByID(t *testing.T, lease *Lease, id string) LeaseSlice {
	t.Helper()
	for _, sl := range lease.Slices {
		if sl.ID == id {
			return sl
		}
	}
	t.Fatalf("slice %s not in lease", id)
	return LeaseSlice{}
}

// rollingLease creates a running lease with pending slice-a (src/a.go) and
// slice-b (src/b.go), both cut from the repo base.
func rollingLease(t *testing.T) (repo, base string) {
	t.Helper()
	repo, base = setupRepo(t)
	if _, err := Create(CreateOpts{
		Root:    repo,
		Slug:    rollSlug,
		BatchID: rollBatch,
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	return repo, base
}

// greenSlice commits marker into file inside the slice worktree and records
// the slice green.
func greenSlice(t *testing.T, repo, id, file, marker string) string {
	t.Helper()
	tc := commitIn(t, WorkerWorktreePath(repo, rollBatch, id), file, marker)
	if _, err := RecordGreen(repo, rollSlug, id, tc); err != nil {
		t.Fatal(err)
	}
	return tc
}

// landSlice greens id and lands it alone on control, returning control HEAD.
func landSlice(t *testing.T, repo, id, file, marker string) string {
	t.Helper()
	greenSlice(t, repo, id, file, marker)
	tip, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: id, ApplyToControl: true})
	if err != nil {
		t.Fatal(err)
	}
	if head := gitOk(t, repo, "rev-parse", "HEAD"); head != tip {
		t.Fatalf("control head %s want integrate tip %s", head, tip)
	}
	return tip
}

// TestRollingSliceFlow lands each green slice on its own and refills the
// freed slot from the new control tip, while slices cut from the old base
// still land on top of it.
func TestRollingSliceFlow(t *testing.T) {
	repo, base := rollingLease(t)

	greenSlice(t, repo, "slice-a", "src/a.go", "A")
	code, out, errOut := runCLI(t, "", "integrate", "--root", repo, "--slug", rollSlug, "--slice", "slice-a", "--apply-to-control")
	if code != ExitOK || !strings.Contains(out, "status=running") || !strings.Contains(out, "slice=slice-a") {
		t.Fatalf("integrate --slice slice-a: code=%d out=%q err=%q", code, out, errOut)
	}
	tipA := gitOk(t, repo, "rev-parse", "HEAD")
	if tipA == base {
		t.Fatal("control did not move after landing slice-a")
	}
	lease := readLeaseT(t, repo)
	if lease.Status != StatusRunning || lease.ControlTip != tipA {
		t.Fatalf("after slice-a: status=%s control_tip=%s want running/%s", lease.Status, lease.ControlTip, tipA)
	}
	if a, b := sliceByID(t, lease, "slice-a"), sliceByID(t, lease, "slice-b"); a.WrightStatus != WrightIntegrated || b.WrightStatus != WrightPending {
		t.Fatalf("slice-a=%s slice-b=%s want integrated/pending", a.WrightStatus, b.WrightStatus)
	}
	if _, err := RecordGreen(repo, rollSlug, "slice-a", tipA); err == nil || !strings.Contains(err.Error(), "already integrated") {
		t.Fatalf("re-greening a landed slice must refuse, got %v", err)
	}

	// A landed slice's id stays taken; its paths are free again.
	if code, _, errOut := runCLI(t, `[{"id":"slice-a","paths":["src/c.go"]}]`, "add", "--root", repo, "--slug", rollSlug, "--json", "-"); code != ExitBlocked || !strings.Contains(errOut, "already in lease") {
		t.Fatalf("add reusing a landed id: code=%d err=%q", code, errOut)
	}
	code, out, errOut = runCLI(t, `{"slices":[{"id":"slice-c","paths":["src/a.go"]}]}`, "add", "--root", repo, "--slug", rollSlug, "--json", "-")
	if want := fmt.Sprintf("added: 1 worktrees; batch=%s active=2 base=%s\n", rollBatch, tipA); code != ExitOK || out != want {
		t.Fatalf("add slice-c: code=%d out=%q want %q err=%q", code, out, want, errOut)
	}
	wtC := WorkerWorktreePath(repo, rollBatch, "slice-c")
	if head := gitOk(t, wtC, "rev-parse", "HEAD"); head != tipA {
		t.Fatalf("slice-c worktree cut from %s want control tip %s", head, tipA)
	}
	lease = readLeaseT(t, repo)
	if c := sliceByID(t, lease, "slice-c"); c.BaseSHA != tipA || c.WrightStatus != WrightPending || lease.N != 3 {
		t.Fatalf("slice-c base=%s status=%s n=%d", c.BaseSHA, c.WrightStatus, lease.N)
	}
	report, err := StatusReport(repo, rollSlug)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"control_tip: " + tipA, "slice slice-a: wright=integrated", "slice slice-c: wright=pending", "base=" + tipA} {
		if !strings.Contains(report, want) {
			t.Fatalf("status report missing %q:\n%s", want, report)
		}
	}

	// slice-b was cut from the original base and lands on top of slice-a.
	tipB := landSlice(t, repo, "slice-b", "src/b.go", "B")
	if parent := gitOk(t, repo, "rev-parse", tipB+"^"); parent != tipA {
		t.Fatalf("slice-b landed on %s want control tip %s", parent, tipA)
	}
	if lease = readLeaseT(t, repo); lease.Status != StatusRunning || lease.ControlTip != tipB {
		t.Fatalf("after slice-b: status=%s control_tip=%s", lease.Status, lease.ControlTip)
	}

	landSlice(t, repo, "slice-c", "src/a.go", "C")
	if lease = readLeaseT(t, repo); lease.Status != StatusComplete {
		t.Fatalf("all slices landed: status=%s want complete", lease.Status)
	}
	if n := gitOk(t, repo, "rev-list", "--count", base+"..HEAD"); n != "3" {
		t.Fatalf("control should gain one commit per slice, got %s", n)
	}
	if a, b := mustRead(t, filepath.Join(repo, "src", "a.go")), mustRead(t, filepath.Join(repo, "src", "b.go")); !strings.Contains(a, "func C") || !strings.Contains(b, "func B") {
		t.Fatalf("landed contents: a=%q b=%q", a, b)
	}

	if _, err := Cleanup(repo, rollSlug, false); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"slice-a", "slice-b", "slice-c"} {
		if _, err := os.Stat(WorkerWorktreePath(repo, rollBatch, id)); !os.IsNotExist(err) {
			t.Fatalf("worktree %s survived cleanup: %v", id, err)
		}
	}
	for _, br := range []string{WorkerBranch(rollSlug, rollBatch, "slice-a"), WorkerBranch(rollSlug, rollBatch, "slice-b"), WorkerBranch(rollSlug, rollBatch, "slice-c"), IntegrateBranchName(rollSlug, rollBatch)} {
		if exists, err := branchExists(repo, br); err != nil || exists {
			t.Fatalf("branch %s survived cleanup: exists=%v err=%v", br, exists, err)
		}
	}
}

func TestAddRefuses(t *testing.T) {
	nine := make([]SlicePaths, 9)
	for i := range nine {
		nine[i] = SlicePaths{ID: fmt.Sprintf("x%d", i), Paths: []string{fmt.Sprintf("src/x%d.go", i)}}
	}
	cases := []struct {
		name    string
		prep    func(t *testing.T, repo string)
		add     []SlicePaths
		want    string
		overlap bool
	}{
		{name: "overlap with an active slice", add: []SlicePaths{{ID: "slice-c", Paths: []string{"src/b.go"}}}, want: "overlap", overlap: true},
		{name: "directory claim over an active slice", add: []SlicePaths{{ID: "slice-c", Paths: []string{"src"}}}, want: "overlap", overlap: true},
		{name: "duplicate id", add: []SlicePaths{{ID: "slice-b", Paths: []string{"src/c.go"}}}, want: "already in lease"},
		{name: "more than ten active", add: nine, want: "at most 10"},
		{
			name: "control moved externally",
			prep: func(t *testing.T, repo string) {
				if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("drift\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				gitOk(t, repo, "commit", "-am", "drift")
			},
			add:  []SlicePaths{{ID: "slice-c", Paths: []string{"src/c.go"}}},
			want: "refusing to rewind",
		},
		{
			name: "lease not running",
			prep: func(t *testing.T, repo string) {
				if _, err := Abort(repo, rollSlug); err != nil {
					t.Fatal(err)
				}
			},
			add:  []SlicePaths{{ID: "slice-c", Paths: []string{"src/c.go"}}},
			want: "requires status=running",
		},
		{
			name: "dirty control on the new paths",
			prep: func(t *testing.T, repo string) {
				if err := os.WriteFile(filepath.Join(repo, "src", "c.go"), []byte("user wip\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			add:  []SlicePaths{{ID: "slice-c", Paths: []string{"src/c.go"}}},
			want: "uncommitted changes",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, _ := rollingLease(t)
			if tc.prep != nil {
				tc.prep(t, repo)
			}
			head := gitOk(t, repo, "rev-parse", "HEAD")
			_, err := Add(AddOpts{Root: repo, Slug: rollSlug, Slices: tc.add})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("add: got %v want %q", err, tc.want)
			}
			if tc.overlap {
				if !errors.Is(err, ErrPathOverlap) {
					t.Fatalf("overlap refusal must wrap ErrPathOverlap: %v", err)
				}
				payload, _ := json.Marshal(tc.add)
				if code, _, _ := runCLI(t, string(payload), "add", "--root", repo, "--slug", rollSlug, "--json", "-"); code != ExitBlocked {
					t.Fatalf("cli overlap exit=%d want %d", code, ExitBlocked)
				}
			}
			if got := gitOk(t, repo, "rev-parse", "HEAD"); got != head {
				t.Fatalf("control moved from %s to %s", head, got)
			}
			if lease := readLeaseT(t, repo); lease.N != 2 || len(lease.Slices) != 2 {
				t.Fatalf("refused add changed the lease: n=%d slices=%d", lease.N, len(lease.Slices))
			}
			for _, sp := range tc.add {
				if sp.ID == "slice-b" {
					continue
				}
				if _, err := os.Stat(WorkerWorktreePath(repo, rollBatch, sp.ID)); !os.IsNotExist(err) {
					t.Fatalf("refused add left worktree %s: %v", sp.ID, err)
				}
				if exists, _ := branchExists(repo, WorkerBranch(rollSlug, rollBatch, sp.ID)); exists {
					t.Fatalf("refused add left branch for %s", sp.ID)
				}
			}
		})
	}
}

func TestIntegrateSliceRefuses(t *testing.T) {
	cases := []struct {
		name   string
		prep   func(t *testing.T, repo string)
		slice  string
		apply  bool
		want   string
		status string // lease status after the refusal
	}{
		{
			name: "control moved externally",
			prep: func(t *testing.T, repo string) {
				if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("drift\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				gitOk(t, repo, "commit", "-am", "drift")
			},
			slice: "slice-a", apply: true, want: "refusing to rewind", status: StatusRunning,
		},
		{name: "slice not green", slice: "slice-b", apply: true, want: "is not green", status: StatusRunning},
		{name: "unknown slice", slice: "slice-z", apply: true, want: "slice not in lease", status: StatusRunning},
		{
			name: "already integrated",
			prep: func(t *testing.T, repo string) {
				if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: "slice-a", ApplyToControl: true}); err != nil {
					t.Fatal(err)
				}
			},
			slice: "slice-a", apply: true, want: "already integrated", status: StatusRunning,
		},
		{name: "missing --apply-to-control", slice: "slice-a", want: "requires --apply-to-control", status: StatusRunning},
		{
			name: "transfer strays outside the slice paths",
			prep: func(t *testing.T, repo string) {
				wt := WorkerWorktreePath(repo, rollBatch, "slice-b")
				if err := os.WriteFile(filepath.Join(wt, "README.md"), []byte("stray\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				gitOk(t, wt, "add", "README.md")
				greenSlice(t, repo, "slice-b", "src/b.go", "B")
			},
			slice: "slice-b", apply: true, want: "exact-path proof failed", status: StatusIntegrateFailed,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, _ := rollingLease(t)
			greenSlice(t, repo, "slice-a", "src/a.go", "A")
			if tc.prep != nil {
				tc.prep(t, repo)
			}
			head := gitOk(t, repo, "rev-parse", "HEAD")
			_, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: tc.slice, ApplyToControl: tc.apply})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("integrate: got %v want %q", err, tc.want)
			}
			if got := gitOk(t, repo, "rev-parse", "HEAD"); got != head {
				t.Fatalf("control moved from %s to %s", head, got)
			}
			if lease := readLeaseT(t, repo); lease.Status != tc.status {
				t.Fatalf("lease status=%s want %s", lease.Status, tc.status)
			}
		})
	}

	t.Run("cli usage error without --apply-to-control", func(t *testing.T) {
		code, _, errOut := runCLI(t, "", "integrate", "--root", t.TempDir(), "--slug", rollSlug, "--slice", "slice-a")
		if code != ExitUsage || !strings.Contains(errOut, "--slice requires --apply-to-control") {
			t.Fatalf("code=%d err=%q want usage error", code, errOut)
		}
	})
}

// TestIntegrateSliceProvesAgainstOwnBase proves a refill slice's exact-path
// proof diffs from the control tip it was cut from: against the batch base its
// diff would also show the earlier landing's path and be refused.
func TestIntegrateSliceProvesAgainstOwnBase(t *testing.T) {
	repo, base := rollingLease(t)
	tipA := landSlice(t, repo, "slice-a", "src/a.go", "A")
	if _, err := Add(AddOpts{Root: repo, Slug: rollSlug, Slices: []SlicePaths{{ID: "slice-c", Paths: []string{"src/c.go"}}}}); err != nil {
		t.Fatal(err)
	}
	tcC := greenSlice(t, repo, "slice-c", "src/c.go", "C")
	if got, err := diffNames(repo, base, tcC); err != nil || len(got) != 2 {
		t.Fatalf("precondition: batch-base diff should carry slice-a's path too, got %v %v", got, err)
	}
	tipC, lease, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: "slice-c", ApplyToControl: true})
	if err != nil {
		t.Fatal(err)
	}
	if parent := gitOk(t, repo, "rev-parse", tipC+"^"); parent != tipA || lease.ControlTip != tipC {
		t.Fatalf("slice-c landed on %s (control_tip %s) want parent %s", parent, lease.ControlTip, tipA)
	}
}

// TestForcedCleanupPartiallyIntegrated proves forced cleanup treats landed
// slices like a complete batch and judges the rest against their own base.
func TestForcedCleanupPartiallyIntegrated(t *testing.T) {
	repo, _ := rollingLease(t)
	tipA := landSlice(t, repo, "slice-a", "src/a.go", "A")
	tcB := commitIn(t, WorkerWorktreePath(repo, rollBatch, "slice-b"), "src/b.go", "B")
	if _, err := Add(AddOpts{Root: repo, Slug: rollSlug, Slices: []SlicePaths{{ID: "slice-c", Paths: []string{"src/c.go"}}}}); err != nil {
		t.Fatal(err)
	}

	var warnings []string
	swapSeams(t)
	warnf = func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}
	if _, err := Abort(repo, rollSlug); err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("abort with control at control_tip must not warn, got %v", warnings)
	}
	salvaged, err := Cleanup(repo, rollSlug, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(salvaged) != 1 || salvaged[0].SliceID != "slice-b" || salvaged[0].Commit != tcB {
		t.Fatalf("salvaged=%+v want only slice-b at %s", salvaged, tcB)
	}
	for id, keep := range map[string]bool{"slice-a": false, "slice-b": true, "slice-c": false} {
		if exists, err := branchExists(repo, WorkerBranch(rollSlug, rollBatch, id)); err != nil || exists != keep {
			t.Fatalf("branch %s exists=%v want %v (err %v)", id, exists, keep, err)
		}
		if _, err := os.Stat(WorkerWorktreePath(repo, rollBatch, id)); !os.IsNotExist(err) {
			t.Fatalf("worktree %s survived cleanup: %v", id, err)
		}
	}
	if head := gitOk(t, repo, "rev-parse", "HEAD"); head != tipA {
		t.Fatalf("cleanup moved control to %s want %s", head, tipA)
	}
}

// TestOldFormatLeaseIntegratesWholeBatch proves a lease written before the
// rolling fields existed still reads, validates and lands as one batch.
func TestOldFormatLeaseIntegratesWholeBatch(t *testing.T) {
	repo, base := setupRepo(t)
	lease := greenBatch(t, repo, base, rollSlug, rollBatch)
	type oldSlice struct {
		ID             string   `json:"id"`
		Paths          []string `json:"paths"`
		WorktreePath   string   `json:"worktree_path"`
		Branch         string   `json:"branch"`
		WrightStatus   string   `json:"wright_status"`
		TransferCommit string   `json:"transfer_commit,omitempty"`
	}
	old := struct {
		BatchID             string     `json:"batch_id"`
		CreatedAt           string     `json:"created_at"`
		BaseSHA             string     `json:"base_sha"`
		N                   int        `json:"n"`
		Status              string     `json:"status"`
		ControlPIDOrSession string     `json:"control_pid_or_session"`
		Slices              []oldSlice `json:"slices"`
	}{BatchID: rollBatch, CreatedAt: lease.CreatedAt, BaseSHA: base, N: 2, Status: StatusRunning, ControlPIDOrSession: "legacy"}
	for _, sl := range lease.Slices {
		tc := gitOk(t, repo, "rev-parse", sl.Branch)
		old.Slices = append(old.Slices, oldSlice{sl.ID, sl.Paths, sl.WorktreePath, sl.Branch, WrightGreen, tc})
	}
	payload, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	path, err := LeasePath(repo, rollSlug)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("```json\n"+string(payload)+"\n```\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadLease(path); err != nil {
		t.Fatalf("old-format lease must read: %v", err)
	}

	tip, got, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, ApplyToControl: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusComplete || gitOk(t, repo, "rev-parse", "HEAD") != tip {
		t.Fatalf("status=%s head!=tip %s", got.Status, tip)
	}
	if n := gitOk(t, repo, "rev-list", "--count", base+"..HEAD"); n != "2" {
		t.Fatalf("control should gain one commit per sibling, got %s", n)
	}
}

func TestValidateLeaseRollingBounds(t *testing.T) {
	t.Parallel()
	integrate := func(n int) func(*Lease) {
		return func(l *Lease) {
			for i := 0; i < n; i++ {
				l.Slices[i].WrightStatus = WrightIntegrated
			}
		}
	}
	cases := []struct {
		name  string
		n     int
		edit  func(*Lease)
		want  string // empty: valid
		yaml  []string
		nyaml []string
	}{
		{name: "old format has no rolling fields", n: 2, nyaml: []string{"control_tip", "    base_sha"}},
		{name: "twelve slices with two landed", n: 12, edit: integrate(2)},
		{name: "eleven active", n: 12, edit: integrate(1), want: "2-10 slices"},
		{name: "one slice", n: 1, want: "2-10 slices"},
		{
			name: "rolling fields are mirrored",
			n:    2,
			edit: func(l *Lease) {
				l.ControlTip = "1234567"
				l.Slices[1].BaseSHA = "1234567"
				l.Slices[0].WrightStatus = WrightIntegrated
			},
			yaml: []string{"control_tip: 1234567\n", "    base_sha: 1234567\n", "wright_status: integrated\n"},
		},
		{name: "bad control_tip", n: 2, edit: func(l *Lease) { l.ControlTip = "main" }, want: "invalid control_tip"},
		{name: "bad slice base_sha", n: 2, edit: func(l *Lease) { l.Slices[0].BaseSHA = "HEAD~1" }, want: "invalid base_sha"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lease := validTestLease(tc.n)
			if tc.edit != nil {
				tc.edit(lease)
			}
			text, err := EmitLeaseMarkdown(lease)
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("got %v want %q", err, tc.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range tc.yaml {
				if !strings.Contains(text, s) {
					t.Fatalf("lease markdown missing %q:\n%s", s, text)
				}
			}
			for _, s := range tc.nyaml {
				if strings.Contains(text, s) {
					t.Fatalf("lease markdown has %q:\n%s", s, text)
				}
			}
		})
	}
}

func TestSelectGreedySeeded(t *testing.T) {
	t.Parallel()
	seed := []SlicePaths{{ID: "A", Paths: []string{"src/a.go"}}, {ID: "B", Paths: []string{"src/b.go"}}}
	ready := []SlicePaths{
		{ID: "A", Paths: []string{"src/a.go"}},
		{ID: "C", Paths: []string{"src/a.go"}},
		{ID: "D", Paths: []string{"src/d.go"}},
		{ID: "E", Paths: []string{"src/e.go"}},
	}
	cases := []struct {
		cap  int
		want string
	}{
		{cap: 2, want: ""},
		{cap: 3, want: "D"},
		{cap: 10, want: "D,E"},
	}
	for _, tc := range cases {
		got, err := SelectGreedySeeded(tc.cap, seed, ready, "")
		if err != nil {
			t.Fatal(err)
		}
		ids := make([]string, len(got))
		for i, s := range got {
			ids[i] = s.ID
		}
		if strings.Join(ids, ",") != tc.want {
			t.Fatalf("cap=%d picks=%v want %s", tc.cap, ids, tc.want)
		}
	}
}

// TestSelectCLISeededByLease proves --slug seeds from the running lease:
// active slices take cap slots and block overlapping candidates, landed ones
// do neither, and ids the lease already names are never picked.
func TestSelectCLISeededByLease(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	lease := validTestLease(3)
	lease.Slices[0].Paths = []string{"src/a.go"}
	lease.Slices[1].Paths = []string{"src/b.go"}
	lease.Slices[2].Paths = []string{"src/x.go"}
	lease.Slices[2].WrightStatus = WrightIntegrated
	path, err := LeasePath(root, rollSlug)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteLease(path, lease); err != nil {
		t.Fatal(err)
	}
	in := `{"slices":[
		{"id":"s2","paths":["src/q.go"]},
		{"id":"C","paths":["src/a.go"]},
		{"id":"D","paths":["src/x.go"]},
		{"id":"E","paths":["src/e.go"]}
	]}`
	code, out, errOut := runCLI(t, in, "select", "--cap", "4", "--root", root, "--slug", rollSlug, "-")
	if want := "select: n_eff=2 mode=parallel slices=D,E running=2\n"; code != ExitOK || out != want {
		t.Fatalf("code=%d out=%q want %q err=%q", code, out, want, errOut)
	}

	for _, tc := range []struct {
		cap  string
		want string
	}{
		{"3", "select: n_eff=1 mode=parallel slices=D running=2\n"},
		{"2", "select: n_eff=0 mode=none slices= running=2\n"},
	} {
		code, out, errOut := runCLI(t, in, "select", "--cap", tc.cap, "--root", root, "--slug", rollSlug, "-")
		if code != ExitOK || out != tc.want {
			t.Fatalf("cap=%s code=%d out=%q want %q err=%q", tc.cap, code, out, tc.want, errOut)
		}
	}

	// Without a lease, --slug changes nothing.
	code, out, errOut = runCLI(t, in, "select", "--cap", "4", "--root", t.TempDir(), "--slug", rollSlug, "-")
	if want := "select: n_eff=4 mode=parallel slices=s2,C,D,E\n"; code != ExitOK || out != want {
		t.Fatalf("no lease: code=%d out=%q want %q err=%q", code, out, want, errOut)
	}
}
