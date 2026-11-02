package lib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/parallel"
)

func TestCloseOutPreservesWorkspacesNamedByRetainedLease(t *testing.T) {
	for _, status := range []string{parallel.StatusRunning, parallel.StatusIntegrateFailed, parallel.StatusAborted, parallel.StatusComplete} {
		for _, slug := range []string{"controller", "child", "unrelated"} {
			t.Run(status+"/"+slug, func(t *testing.T) {
				root := t.TempDir()
				for _, name := range []string{"controller", "child", "unrelated"} {
					phase := "ship"
					if name == "unrelated" {
						phase = "build"
					}
					writeFile(t, filepath.Join(root, "work", name, "state.md"), "| phase | "+phase+" |\n| schema | 4 |\n")
				}
				writeFile(t, filepath.Join(root, "ACTIVE"), "controller\n")
				wright := parallel.WrightPending
				if status == parallel.StatusComplete {
					wright = parallel.WrightIntegrated
				}
				leasePath := filepath.Join(root, "work", "controller", "parallel-lease.md")
				lease := &parallel.Lease{
					BatchID: "b1", CreatedAt: "2026-10-31T00:00:00Z", BaseSHA: "abcdef1",
					N: 2, Status: status, ControlPIDOrSession: "archive-regression",
					Slices: []parallel.LeaseSlice{
						{ID: "SLICE-001", Paths: []string{"src/a.go"}, WrightStatus: wright},
						{Workspace: "child", ID: "SLICE-001", Paths: []string{"src/b.go"}, WrightStatus: wright},
					},
				}
				if err := parallel.WriteLease(leasePath, lease); err != nil {
					t.Fatal(err)
				}
				before, err := os.ReadFile(leasePath)
				if err != nil {
					t.Fatal(err)
				}
				code, out := runCloseOut(t, root, slug)
				if slug == "unrelated" {
					if code != 0 {
						t.Fatalf("unrelated archive blocked: %d %s", code, out)
					}
				} else {
					if code != 3 || !strings.Contains(out, "lease") {
						t.Fatalf("retained workspace archive = %d, want blocked: %s", code, out)
					}
					if _, err := os.Stat(filepath.Join(root, "work", slug, "state.md")); err != nil {
						t.Fatalf("blocked archive moved source: %v", err)
					}
					if _, err := os.Lstat(filepath.Join(root, "archive", slug)); !os.IsNotExist(err) {
						t.Fatalf("blocked archive created destination: %v", err)
					}
				}
				after, err := os.ReadFile(leasePath)
				if err != nil || string(after) != string(before) {
					t.Fatalf("archive changed retained lease: %v", err)
				}
				active, err := os.ReadFile(filepath.Join(root, "ACTIVE"))
				if err != nil || string(active) != "controller\n" {
					t.Fatalf("archive changed controller pointer: %q %v", active, err)
				}
			})
		}
	}
}
