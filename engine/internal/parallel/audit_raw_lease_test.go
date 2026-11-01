package parallel

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func auditRawLeaseWorker(t *testing.T) (repo, base, transfer string) {
	t.Helper()
	repo, base = setupRepo(t)
	lease, err := Create(CreateOpts{
		Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base,
		Slices: []SlicePaths{{ID: "slice-a", Paths: []string{"src/a.go"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	transfer = commitIn(t, lease.Slices[0].WorktreePath, "src/a.go", "AuditUnlanded")
	return repo, base, transfer
}

func auditRawLeaseWrite(t *testing.T, repo string, lease *Lease) (int, string) {
	t.Helper()
	data, err := json.Marshal(lease)
	if err != nil {
		t.Fatal(err)
	}
	code, _, stderr := runCLI(t, string(data), "lease-write", "--root", repo, "--slug", rollSlug, "--json", "-")
	return code, stderr
}

func TestAuditRawLeaseRefusesUnlandedIntegratedState(t *testing.T) {
	for _, status := range []string{WrightPending, WrightGreen} {
		t.Run(status, func(t *testing.T) {
			repo, base, transfer := auditRawLeaseWorker(t)
			if status == WrightGreen {
				if _, err := RecordGreen(repo, rollSlug, "slice-a", transfer); err != nil {
					t.Fatal(err)
				}
			}
			lease := readLeaseT(t, repo)
			lease.Slices[0].TransferCommit = transfer
			// A pending worker may already have committed without recording green.
			if code, stderr := auditRawLeaseWrite(t, repo, lease); code != ExitOK {
				t.Fatalf("valid %s worker lease refused: %d %s", status, code, stderr)
			}
			code, data, stderr := runCLI(t, "", "lease-read", "--root", repo, "--slug", rollSlug, "--field", "json")
			if code != ExitOK {
				t.Fatalf("lease round-trip read failed: %d %s", code, stderr)
			}
			if err := json.Unmarshal([]byte(data), &lease); err != nil {
				t.Fatal(err)
			}
			worker := lease.Slices[0]
			if worker.WrightStatus != status || worker.TransferCommit != transfer || lease.ControlTip != "" || lease.PendingLanding != nil || lease.LastLanding != nil {
				t.Fatalf("fixture must retain unlanded worker identity: %+v", lease)
			}
			if head := gitOk(t, repo, "rev-parse", "HEAD"); head != base || head == transfer {
				t.Fatalf("fixture control already moved: %s", head)
			}
			if err := ensureControllerIdentity(repo, lease); err != nil {
				t.Fatalf("fixture controller identity invalid: %v", err)
			}
			path, err := LeasePath(repo, rollSlug)
			if err != nil {
				t.Fatal(err)
			}
			paths := []string{path, filepath.Join(worker.WorktreePath, "src/a.go"), filepath.Join(worker.WorktreePath, ".git"), filepath.Join(repo, "src/a.go"), filepath.Join(repo, ".git", "HEAD")}
			before := make([][]byte, len(paths))
			for i, path := range paths {
				before[i], err = os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
			}
			registrations := gitOk(t, repo, "worktree", "list", "--porcelain")
			controlStatus := gitOk(t, repo, "status", "--porcelain", "--untracked-files=all")
			controlRef := gitOk(t, repo, "symbolic-ref", "HEAD")
			lease.Slices[0].WrightStatus = WrightIntegrated
			lease.Status = StatusComplete
			if err := ValidateLease(lease); err != nil {
				t.Fatalf("mutation must reach raw-transition guard, not fail lease validation: %v", err)
			}
			if code, stderr := auditRawLeaseWrite(t, repo, lease); code != ExitBlocked {
				t.Errorf("lease-write accepted unlanded %s worker as integrated/complete: code=%d stderr=%s", status, code, stderr)
			}
			for i, path := range paths {
				after, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(after, before[i]) {
					t.Errorf("raw refusal changed recovery/control bytes at %s: %v", path, err)
				}
			}
			if got := gitOk(t, repo, "rev-parse", "refs/heads/"+worker.Branch); got != transfer {
				t.Errorf("worker recovery ref changed: %s want %s", got, transfer)
			}
			if got := gitOk(t, worker.WorktreePath, "rev-parse", "HEAD"); got != transfer {
				t.Errorf("worker checkout changed: %s want %s", got, transfer)
			}
			if got := gitOk(t, repo, "rev-parse", "HEAD"); got != base {
				t.Errorf("control tip changed: %s want %s", got, base)
			}
			if got := gitOk(t, repo, "symbolic-ref", "HEAD"); got != controlRef {
				t.Errorf("control ref changed: %s want %s", got, controlRef)
			}
			if got := gitOk(t, repo, "worktree", "list", "--porcelain"); got != registrations {
				t.Error("raw refusal changed worktree registration")
			}
			if got := gitOk(t, repo, "status", "--porcelain", "--untracked-files=all"); got != controlStatus {
				t.Error("raw refusal changed control checkout status")
			}
		})
	}
}

func TestAuditRawLeaseAllowsCanonicalAndLegacyIntegratedRoundTrips(t *testing.T) {
	repo, base, transfer := auditRawLeaseWorker(t)
	if _, err := RecordGreen(repo, rollSlug, "slice-a", transfer); err != nil {
		t.Fatal(err)
	}
	tip, lease, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, ApplyToControl: true})
	if err != nil {
		t.Fatalf("canonical integration refused: %v", err)
	}
	if tip == base || lease.Status != StatusComplete || lease.Slices[0].WrightStatus != WrightIntegrated || lease.LastLanding == nil {
		t.Fatalf("canonical integration did not retain accepted landing: %+v", lease)
	}
	if got := gitOk(t, repo, "show", "HEAD:src/a.go"); got != "package main\n\nfunc AuditUnlanded() {}" {
		t.Fatalf("canonical integration lost worker change: %q", got)
	}
	for _, legacy := range []bool{false, true} {
		if legacy {
			// Historical integrated rows can predate landing receipts.
			lease.LastLanding = nil
			path, err := LeasePath(repo, rollSlug)
			if err != nil {
				t.Fatal(err)
			}
			if err := WriteLease(path, lease); err != nil {
				t.Fatal(err)
			}
		}
		if code, stderr := auditRawLeaseWrite(t, repo, lease); code != ExitOK {
			t.Fatalf("unchanged integrated round-trip refused (legacy=%t): %d %s", legacy, code, stderr)
		}
		if got := readLeaseT(t, repo); !reflect.DeepEqual(got, lease) {
			t.Fatalf("round-trip changed integrated evidence (legacy=%t): %+v", legacy, got)
		}
		if got := gitOk(t, repo, "rev-parse", "HEAD"); got != tip {
			t.Fatalf("round-trip moved landed control: %s want %s", got, tip)
		}
	}
}
