package parallel

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLandingWriteFailuresRetryExactlyOnce(t *testing.T) {
	for _, phase := range []string{"prepare", "prepared-sync", "final", "final-sync"} {
		t.Run(phase, func(t *testing.T) {
			repo, base := rollingLease(t)
			greenSlice(t, repo, "slice-a", "src/a.go", "A")
			swapSeams(t)
			failed := false
			writeLease = func(path string, lease *Lease) error {
				prepare := lease.PendingLanding != nil
				if !failed && ((strings.HasPrefix(phase, "prepare") && prepare) || (strings.HasPrefix(phase, "final") && !prepare && lease.LastLanding != nil)) {
					failed = true
					if strings.HasSuffix(phase, "sync") {
						if err := WriteLease(path, lease); err != nil {
							return err
						}
					}
					return errors.New("injected durable write failure")
				}
				return WriteLease(path, lease)
			}
			opts := IntegrateOpts{Root: repo, Slug: rollSlug, Slice: "slice-a", ApplyToControl: true}
			if _, _, err := Integrate(opts); err == nil || !failed {
				t.Fatalf("fault not surfaced: %v", err)
			}
			head := gitOk(t, repo, "rev-parse", "HEAD")
			if strings.HasPrefix(phase, "prepare") && head != base {
				t.Fatal("control advanced before durable preparation succeeded")
			}
			if strings.HasPrefix(phase, "final") && head == base {
				t.Fatal("final fault did not exercise landed control")
			}
			writeLease = WriteLease
			tip, lease, err := Integrate(opts)
			if err != nil || lease.PendingLanding != nil || lease.ControlTip != tip || lease.LastLanding == nil {
				t.Fatalf("retry did not accept landing: %s %+v %v", tip, lease, err)
			}
			if tip == base || gitOk(t, repo, "rev-list", "--count", base+"..HEAD") != "1" {
				t.Fatal("retry duplicated or omitted control commit")
			}
			again, _, err := Integrate(opts)
			if err != nil || again != tip || gitOk(t, repo, "rev-parse", "HEAD") != tip {
				t.Fatalf("accepted receipt not idempotent: %s %v", again, err)
			}
		})
	}
}

// pendingLandingT leaves real Git at exact prior or exact new, with durable intent.
func pendingLandingT(t *testing.T, landed bool) (string, string) {
	t.Helper()
	repo, base := rollingLease(t)
	greenSlice(t, repo, "slice-a", "src/a.go", "A")
	swapSeams(t)
	writeLease = func(path string, lease *Lease) error {
		if lease.PendingLanding != nil && !landed {
			if err := WriteLease(path, lease); err != nil {
				return err
			}
			return errors.New("prepared replacement sync failure")
		}
		if lease.PendingLanding == nil && lease.LastLanding != nil && landed {
			return errors.New("final write failure")
		}
		return WriteLease(path, lease)
	}
	if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: "slice-a", ApplyToControl: true}); err == nil {
		t.Fatal("expected persistence fault")
	}
	writeLease = WriteLease
	if readLeaseT(t, repo).PendingLanding == nil {
		t.Fatal("pending intent missing")
	}
	return repo, base
}

func TestPendingLandingRejectsExternalIdentityAndMovement(t *testing.T) {
	for _, movement := range []string{"descendant", "different-tip", "same-sha-branch", "worker-identity"} {
		t.Run(movement, func(t *testing.T) {
			repo, base := pendingLandingT(t, true)
			switch movement {
			case "descendant":
				gitOk(t, repo, "commit", "--allow-empty", "-m", "external descendant")
			case "different-tip":
				gitOk(t, repo, "reset", "--hard", base)
				gitOk(t, repo, "commit", "--allow-empty", "-m", "other tip")
			case "same-sha-branch":
				gitOk(t, repo, "checkout", "-b", "external/same-tip")
			case "worker-identity":
				path, _ := LeasePath(repo, rollSlug)
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				// Change only the row, not its durable journal snapshot.
				data = []byte(strings.Replace(string(data), `"branch": "devrites/parallel/demo-feature/batch1/slice-a"`, `"branch": "external/worker"`, 1))
				if err := os.WriteFile(path, data, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			head := gitOk(t, repo, "rev-parse", "HEAD")
			if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: "slice-a", ApplyToControl: true}); err == nil {
				t.Fatal("external change treated as accepted landing")
			}
			if gitOk(t, repo, "rev-parse", "HEAD") != head {
				t.Fatal("refusal changed external control")
			}
		})
	}
}

func TestPendingLandingReadOnlyAndRawConsumers(t *testing.T) {
	repo, _ := pendingLandingT(t, true)
	path, _ := LeasePath(repo, rollSlug)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range [][]string{{"status"}, {"lease-read", "--field", "json"}} {
		args := append(command, "--root", repo, "--slug", rollSlug)
		if code, _, stderr := runCLI(t, "", args...); code != ExitOK {
			t.Fatalf("read failed: %d %s", code, stderr)
		}
	}
	if after, err := os.ReadFile(path); err != nil || string(after) != string(before) {
		t.Fatal("read-only command reconciled pending state")
	}
	if _, _, _, err := SelectForSlug(repo, rollSlug, 10, []SlicePaths{{ID: "next", Paths: []string{"src/c.go"}}}); err == nil {
		t.Fatal("pending landing admitted work")
	}
	lease := readLeaseT(t, repo)
	lease.PendingLanding = nil
	data, _ := json.Marshal(lease)
	if code, _, _ := runCLI(t, string(data), "lease-write", "--root", repo, "--slug", rollSlug, "--json", "-"); code != ExitBlocked {
		t.Fatal("raw lease-write removed pending intent")
	}
	if code, _, _ := runCLI(t, "", "lease-clear", "--root", repo, "--slug", rollSlug); code != ExitBlocked {
		t.Fatal("raw lease-clear removed pending intent")
	}
	if after, _ := os.ReadFile(path); string(after) != string(before) {
		t.Fatal("raw refusal changed lease")
	}
	if _, err := Add(AddOpts{Root: repo, Slug: rollSlug, Capacity: 1}); err != nil {
		t.Fatalf("cap update failed to finalize exact-new landing first: %v", err)
	}
	if lease := readLeaseT(t, repo); lease.PendingLanding != nil || lease.Slices[0].WrightStatus != WrightIntegrated || lease.Capacity != 1 {
		t.Fatalf("cap update lost recovered landing: %+v", lease)
	}
}

func TestPendingPriorBlocksMutationAndAbortCancelsWithoutFF(t *testing.T) {
	repo, base := pendingLandingT(t, false)
	if _, err := RecordGreen(repo, rollSlug, "slice-b", base); err == nil {
		t.Fatal("record-green bypassed exact-prior preparation")
	}
	if _, err := Prune(repo, rollSlug); err == nil {
		t.Fatal("prune bypassed exact-prior preparation")
	}
	if _, err := Cleanup(repo, rollSlug, true); err == nil {
		t.Fatal("cleanup bypassed exact-prior preparation")
	}
	lease, err := Abort(repo, rollSlug)
	if err != nil || lease.PendingLanding != nil || lease.Status != StatusAborted || gitOk(t, repo, "rev-parse", "HEAD") != base {
		t.Fatalf("abort did not cancel exact-prior intent: %+v %v", lease, err)
	}
}

func TestPendingExactNewMutationConsumersFinalizeFirst(t *testing.T) {
	for _, operation := range []string{"record-green", "abort", "prune", "cleanup"} {
		t.Run(operation, func(t *testing.T) {
			repo, base := pendingLandingT(t, true)
			var err error
			switch operation {
			case "record-green":
				_, err = RecordGreen(repo, rollSlug, "slice-b", base)
			case "abort":
				_, err = Abort(repo, rollSlug)
			case "prune":
				_, err = Prune(repo, rollSlug)
			case "cleanup":
				_, err = Cleanup(repo, rollSlug, true)
			}
			if err != nil {
				t.Fatal(err)
			}
			if operation != "cleanup" {
				lease := readLeaseT(t, repo)
				if lease.PendingLanding != nil || lease.Slices[0].WrightStatus != WrightIntegrated {
					t.Fatalf("mutation lost accepted landing: %+v", lease)
				}
			}
			if gitOk(t, repo, "rev-list", "--count", base+"..HEAD") != "1" {
				t.Fatal("mutation duplicated landing")
			}
		})
	}
}

func TestRawLeaseCannotFabricateLandingReceipt(t *testing.T) {
	repo, _ := rollingLease(t)
	landSlice(t, repo, "slice-a", "src/a.go", "A")
	lease := readLeaseT(t, repo)
	lease.LastLanding.NewTip = lease.BaseSHA
	data, _ := json.Marshal(lease)
	if code, _, _ := runCLI(t, string(data), "lease-write", "--root", repo, "--slug", rollSlug, "--json", "-"); code != ExitBlocked {
		t.Fatal("raw lease-write changed accepted receipt")
	}
	if _, err := os.Stat(filepath.Join(repo, ".devrites", "work", rollSlug, "parallel-lease.md")); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyLeaseNeverInfersHistoricalLanding(t *testing.T) {
	repo, _ := rollingLease(t)
	tc := greenSlice(t, repo, "slice-a", "src/a.go", "A")
	lease := readLeaseT(t, repo)
	lease.ControlCheckout, lease.ControlRef = "", ""
	path, _ := LeasePath(repo, rollSlug)
	if err := WriteLease(path, lease); err != nil {
		t.Fatal(err)
	}
	gitOk(t, repo, "merge", "--ff-only", tc)
	if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: "slice-a", ApplyToControl: true}); err == nil {
		t.Fatal("legacy descendant fabricated historical landing evidence")
	}
	lease = readLeaseT(t, repo)
	if lease.PendingLanding != nil || lease.LastLanding != nil || lease.Slices[0].WrightStatus != WrightGreen {
		t.Fatalf("legacy refusal changed historical evidence: %+v", lease)
	}
}

func TestLegacyAbbreviatedBaseCanLandWithExactJournal(t *testing.T) {
	repo, base := rollingLease(t)
	greenSlice(t, repo, "slice-a", "src/a.go", "A")
	lease := readLeaseT(t, repo)
	lease.BaseSHA = base[:12]
	lease.ControlCheckout, lease.ControlRef = "", ""
	path, _ := LeasePath(repo, rollSlug)
	if err := WriteLease(path, lease); err != nil {
		t.Fatal(err)
	}
	tip, lease, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: "slice-a", ApplyToControl: true})
	if err != nil || lease.LastLanding.PriorTip != base || lease.LastLanding.NewTip != tip {
		t.Fatalf("legacy abbreviation lost exact landing identity: %+v %v", lease, err)
	}
}

func TestAllTargetLandingRecoveryUsesRecordedTargets(t *testing.T) {
	repo, base := setupRepo(t)
	greenBatch(t, repo, base, rollSlug, rollBatch)
	swapSeams(t)
	writeLease = func(path string, lease *Lease) error {
		if lease.PendingLanding == nil && lease.LastLanding != nil {
			return errors.New("final batch write failed")
		}
		return WriteLease(path, lease)
	}
	opts := IntegrateOpts{Root: repo, Slug: rollSlug, ApplyToControl: true}
	if _, _, err := Integrate(opts); err == nil {
		t.Fatal("expected final batch persistence failure")
	}
	if record := readLeaseT(t, repo).PendingLanding; record == nil || len(record.Targets) != 2 {
		t.Fatalf("batch lost exact target set: %+v", record)
	}
	writeLease = WriteLease
	tip, lease, err := Integrate(opts)
	if err != nil || lease.Status != StatusComplete || lease.ControlTip != tip {
		t.Fatalf("batch retry did not finish: %+v %v", lease, err)
	}
	if gitOk(t, repo, "rev-list", "--count", base+"..HEAD") != "2" {
		t.Fatal("batch retry duplicated commits")
	}
}
