package parallel

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestControllerLeaseIdentity(t *testing.T) {
	for _, tc := range []struct {
		name      string
		switchRef bool
		legacy    bool
		wantError bool
	}{
		{name: "recorded identity"},
		{name: "branch changed at same HEAD", switchRef: true, wantError: true},
		{name: "legacy without historical identity", switchRef: true, legacy: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, base := setupRepo(t)
			lease, err := Create(CreateOpts{
				Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base,
				Slices: []SlicePaths{{ID: "slice-a", Paths: []string{"src/a.go"}}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if lease.ControlCheckout == "" || lease.ControlRef != "refs/heads/main" {
				t.Fatalf("fixture missing recorded controller identity: %+v", lease)
			}
			path, err := LeasePath(repo, rollSlug)
			if err != nil {
				t.Fatal(err)
			}
			if tc.legacy {
				lease.ControlCheckout = ""
				lease.ControlRef = ""
				if err := WriteLease(path, lease); err != nil {
					t.Fatal(err)
				}
			}
			if tc.switchRef {
				gitOk(t, repo, "checkout", "-b", "other-control")
				if head := gitOk(t, repo, "rev-parse", "HEAD"); head != base {
					t.Fatalf("branch switch changed HEAD: got %s want %s", head, base)
				}
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			controller, got, err := ControllerLease(repo)
			if tc.wantError {
				if err == nil || !strings.Contains(err.Error(), "control checkout/ref changed") {
					t.Fatalf("changed controller identity resolved: controller=%q lease=%+v err=%v", controller, got, err)
				}
				if controller != "" || got != nil {
					t.Fatalf("refused identity exposed controller lease: controller=%q lease=%+v", controller, got)
				}
			} else if err != nil || controller != rollSlug || got == nil || got.BatchID != lease.BatchID {
				t.Fatalf("valid controller identity refused: controller=%q lease=%+v err=%v", controller, got, err)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("controller resolution changed retained lease")
			}
		})
	}
}

func TestControllerLeaseRefusesOverlappingActiveReservations(t *testing.T) {
	repo, _ := rollingLease(t)
	if controller, lease, err := ControllerLease(repo); err != nil || controller != rollSlug || lease == nil {
		t.Fatalf("valid reservations refused: controller=%q lease=%+v err=%v", controller, lease, err)
	}
	lease := readLeaseT(t, repo)
	lease.Slices[1].Paths = append([]string(nil), lease.Slices[0].Paths...)
	path, err := LeasePath(repo, rollSlug)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteLease(path, lease); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	controller, got, err := ControllerLease(repo)
	if !errors.Is(err, ErrPathOverlap) {
		t.Fatalf("overlapping active reservations resolved: controller=%q lease=%+v err=%v", controller, got, err)
	}
	if controller != "" || got != nil {
		t.Fatalf("refused overlap exposed controller lease: controller=%q lease=%+v", controller, got)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("controller resolution changed overlapping retained lease")
	}
	if _, err := ReadLease(path); err != nil {
		t.Fatalf("overlapping lease must remain readable for recovery: %v", err)
	}
}

func TestControllerLeaseAllowsHistoricalOverlapWithActiveReservations(t *testing.T) {
	repo, _ := rollingLease(t)
	transfer := greenSlice(t, repo, "slice-a", "src/a.go", "A")
	if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: "slice-a", ApplyToControl: true}); err != nil {
		t.Fatal(err)
	}
	lease, err := Add(AddOpts{Root: repo, Slug: rollSlug, Slices: []SlicePaths{{ID: "slice-c", Paths: []string{"src/a.go"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if lease.Slices[0].WrightStatus != WrightIntegrated || lease.Slices[0].TransferCommit != transfer || len(activeSlices(lease)) != 2 {
		t.Fatalf("fixture lacks historical overlap and two active reservations: %+v", lease)
	}
	path, err := LeasePath(repo, rollSlug)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	controller, got, err := ControllerLease(repo)
	if err != nil || controller != rollSlug || got == nil || len(got.Slices) != 3 {
		t.Fatalf("historical overlap blocked active reservations: controller=%q lease=%+v err=%v", controller, got, err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("controller resolution changed historical retained lease")
	}
}
