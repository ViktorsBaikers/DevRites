package parallel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ignoreLateFiles(t *testing.T, repo string) {
	t.Helper()
	path := filepath.Join(repo, ".git", "info", "exclude")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("\n*.ignored\n"); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPruneMixedSourceRollingRefillPreservesProofRowsAndCap(t *testing.T) {
	repo, base := setupRepo(t)
	liveSource(t, repo, "child")
	lease, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Capacity: 2, Slices: []SlicePaths{
		{ID: "SLICE-001", Paths: []string{"src/a.go"}},
		{ID: "SLICE-001", Workspace: "child", Paths: []string{"src/b.go"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	worker := qualifiedSlice(t, lease, "child/SLICE-001")
	tc := commitIn(t, worker.WorktreePath, "src/b.go", "Child")
	if _, err := RecordGreen(repo, rollSlug, "child/SLICE-001", tc); err != nil {
		t.Fatal(err)
	}
	tip, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, Slice: "child/SLICE-001", ApplyToControl: true})
	if err != nil {
		t.Fatal(err)
	}
	if ancestor, err := isAncestor(repo, tc, tip); err != nil || ancestor {
		t.Fatalf("fixture must use rewritten nonancestral transfer: %v %v", ancestor, err)
	}
	proof := filepath.Join(repo, ".devrites", "work", "child", "proof.txt")
	if err := os.WriteFile(proof, []byte("worker-bound proof"), 0o600); err != nil {
		t.Fatal(err)
	}
	path, _ := LeasePath(repo, rollSlug)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for attempt := range 2 {
		code, out, stderr := runCLI(t, "", "prune", "--root", repo, "--slug", rollSlug)
		if code != ExitOK || !strings.Contains(out, "branches and lease retained") {
			t.Fatalf("prune attempt %d: %d %s %s", attempt, code, out, stderr)
		}
	}
	if _, err := os.Stat(worker.WorktreePath); !os.IsNotExist(err) {
		t.Fatalf("integrated checkout remains: %v", err)
	}
	if _, err := os.Stat(lease.Slices[0].WorktreePath); err != nil {
		t.Fatalf("pending worker removed: %v", err)
	}
	if gitOk(t, repo, "rev-parse", "refs/heads/"+worker.Branch) != tc || gitOk(t, repo, "rev-parse", "HEAD") != tip {
		t.Fatal("prune changed transfer or control ref")
	}
	if after, err := os.ReadFile(path); err != nil || string(before) != string(after) {
		t.Fatal("prune changed lease rows or controller")
	}
	if data, err := os.ReadFile(proof); err != nil || string(data) != "worker-bound proof" {
		t.Fatalf("prune lost proof: %q %v", data, err)
	}
	lease, err = Add(AddOpts{Root: repo, Slug: rollSlug, Slices: []SlicePaths{{ID: "SLICE-002", Workspace: "child", Paths: []string{"src/b.go"}}}})
	if err != nil || lease.Capacity != 2 || len(lease.Slices) != 3 || len(activeSlices(lease)) != 2 {
		t.Fatalf("prune/refill lost rolling cap or history: %+v %v", lease, err)
	}
	if qualifiedSlice(t, lease, "child/SLICE-002").BaseSHA != tip {
		t.Fatal("refill not cut from accepted control")
	}
	if _, err := Add(AddOpts{Root: repo, Slug: rollSlug, Slices: []SlicePaths{{ID: "extra", Paths: []string{"src/c.go"}}}}); err == nil {
		t.Fatal("prune allowed cap overbooking")
	}
}

func TestIntegratedPruneAndCleanupPreserveLateDataAndRefs(t *testing.T) {
	for _, operation := range []string{"prune", "cleanup", "force-cleanup"} {
		for _, change := range []string{"tracked", "untracked", "ignored", "extra-commit", "absent-extra-commit", "switched-branch", "replacement", "symlink"} {
			t.Run(operation+"/"+change, func(t *testing.T) {
				repo, base := setupRepo(t)
				lease, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Slices: []SlicePaths{{ID: "slice-a", Paths: []string{"src/a.go"}}}})
				if err != nil {
					t.Fatal(err)
				}
				worker := lease.Slices[0]
				tc := commitIn(t, worker.WorktreePath, "src/a.go", "Accepted")
				if _, err := RecordGreen(repo, rollSlug, "slice-a", tc); err != nil {
					t.Fatal(err)
				}
				if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, ApplyToControl: true}); err != nil {
					t.Fatal(err)
				}
				ignoreLateFiles(t, repo)
				dataPath := filepath.Join(worker.WorktreePath, "late.txt")
				switch change {
				case "tracked":
					dataPath = filepath.Join(worker.WorktreePath, "src/a.go")
				case "ignored":
					dataPath = filepath.Join(worker.WorktreePath, "late.ignored")
				case "switched-branch":
					gitOk(t, worker.WorktreePath, "checkout", "-b", "human/worker")
				case "replacement", "symlink":
					gitOk(t, repo, "worktree", "remove", worker.WorktreePath)
					if change == "replacement" {
						if err := os.MkdirAll(worker.WorktreePath, 0o755); err != nil {
							t.Fatal(err)
						}
					} else {
						if err := os.Symlink(t.TempDir(), worker.WorktreePath); err != nil {
							t.Fatal(err)
						}
					}
				}
				if err := os.WriteFile(dataPath, []byte("late human data"), 0o644); err != nil {
					t.Fatal(err)
				}
				if change == "extra-commit" || change == "absent-extra-commit" {
					gitOk(t, worker.WorktreePath, "add", "late.txt")
					gitOk(t, worker.WorktreePath, "commit", "-m", "late human commit")
					if change == "absent-extra-commit" {
						gitOk(t, repo, "worktree", "remove", worker.WorktreePath)
					}
				}
				branchTip := gitOk(t, repo, "rev-parse", "refs/heads/"+worker.Branch)
				if operation == "prune" {
					if _, err := Prune(repo, rollSlug); err == nil {
						t.Fatal("prune accepted changed worker")
					}
				} else {
					results, err := Cleanup(repo, rollSlug, operation == "force-cleanup")
					if err != nil || len(results) == 0 || results[0].Branch != worker.Branch {
						t.Fatalf("cleanup omitted recovery ref: %+v %v", results, err)
					}
					if change != "absent-extra-commit" && results[0].Worktree != worker.WorktreePath {
						t.Fatalf("cleanup omitted recovery path: %+v", results)
					}
				}
				if gitOk(t, repo, "rev-parse", "refs/heads/"+worker.Branch) != branchTip {
					t.Fatal("maintenance deleted or changed late worker ref")
				}
				if change != "absent-extra-commit" {
					if data, err := os.ReadFile(dataPath); err != nil || string(data) != "late human data" {
						t.Fatalf("maintenance lost late data: %q %v", data, err)
					}
				} else if data := gitOk(t, repo, "show", worker.Branch+":late.txt"); data != "late human data" {
					t.Fatal("maintenance lost checkout-absent late commit")
				}
			})
		}
	}
}

func TestForceCleanupSalvagesAllowlistButPreservesResidualData(t *testing.T) {
	for _, residual := range []string{"outside.txt", "outside.ignored"} {
		t.Run(residual, func(t *testing.T) {
			repo, _ := rollingLease(t)
			ignoreLateFiles(t, repo)
			worker := readLeaseT(t, repo).Slices[0]
			if err := os.WriteFile(filepath.Join(worker.WorktreePath, "src/a.go"), []byte("allowlisted work"), 0o644); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(worker.WorktreePath, residual)
			if err := os.WriteFile(path, []byte("residual human data"), 0o644); err != nil {
				t.Fatal(err)
			}
			results, err := Cleanup(repo, rollSlug, true)
			if err != nil || len(results) != 1 || results[0].Worktree != worker.WorktreePath || results[0].Branch != worker.Branch {
				t.Fatalf("missing residual recovery identity: %+v %v", results, err)
			}
			if data, err := os.ReadFile(path); err != nil || string(data) != "residual human data" {
				t.Fatalf("cleanup lost residual: %q %v", data, err)
			}
			if data := gitOk(t, repo, "show", worker.Branch+":src/a.go"); data != "allowlisted work" {
				t.Fatal("cleanup failed allowlisted salvage")
			}
		})
	}
}

func TestBranchDeletionBindsExpectedTipAndRegisteredCheckout(t *testing.T) {
	repo, base := rollingLease(t)
	worker := readLeaseT(t, repo).Slices[0]
	if err := deleteBranchAt(repo, worker.Branch, base); err == nil {
		t.Fatal("CAS deletion bypassed registered worker checkout")
	}
	tip := commitIn(t, worker.WorktreePath, "src/a.go", "Late")
	gitOk(t, repo, "worktree", "remove", worker.WorktreePath)
	if err := deleteBranchAt(repo, worker.Branch, base); err == nil {
		t.Fatal("CAS deletion discarded advanced ref")
	}
	if gitOk(t, repo, "rev-parse", "refs/heads/"+worker.Branch) != tip {
		t.Fatal("failed CAS changed branch")
	}
}

func TestCleanupAbsentRegisteredIntegratedWorker(t *testing.T) {
	for _, force := range []bool{false, true} {
		for _, change := range []string{"unchanged", "switched-registration", "extra-commit"} {
			t.Run(fmt.Sprintf("force=%t/%s", force, change), func(t *testing.T) {
				repo, base := setupRepo(t)
				lease, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Slices: []SlicePaths{{ID: "slice-a", Paths: []string{"src/a.go"}}}})
				if err != nil {
					t.Fatal(err)
				}
				worker := lease.Slices[0]
				tc := commitIn(t, worker.WorktreePath, "src/a.go", "Accepted")
				if _, err := RecordGreen(repo, rollSlug, "slice-a", tc); err != nil {
					t.Fatal(err)
				}
				if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, ApplyToControl: true}); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "switched-registration":
					gitOk(t, worker.WorktreePath, "checkout", "-b", "human/missing-worker")
				case "extra-commit":
					gitOk(t, worker.WorktreePath, "commit", "--allow-empty", "-m", "late worker commit")
				}
				tip := gitOk(t, repo, "rev-parse", "refs/heads/"+worker.Branch)
				// Simulate external directory loss without Git unregistering it.
				if err := os.RemoveAll(worker.WorktreePath); err != nil {
					t.Fatal(err)
				}
				registered, err := registeredWorktrees(repo)
				if err != nil || !registered[resolvedPath(worker.WorktreePath)] {
					t.Fatalf("fixture lost fresh registration: %v", err)
				}
				results, err := Cleanup(repo, rollSlug, force)
				if err != nil {
					t.Fatalf("cleanup stranded an absent checkout: %+v %v", results, err)
				}
				registered, err = registeredWorktrees(repo)
				if err != nil {
					t.Fatal(err)
				}
				exists, err := branchExists(repo, worker.Branch)
				if err != nil {
					t.Fatal(err)
				}
				if !registered[resolvedPath(worker.WorktreePath)] || !exists || len(results) != 1 || results[0].Branch != worker.Branch || results[0].Worktree != worker.WorktreePath || gitOk(t, repo, "rev-parse", "refs/heads/"+worker.Branch) != tip {
					t.Fatalf("cleanup discarded missing worker recovery registration/ref/path: %+v %v", results, registered)
				}
				if change == "switched-registration" && gitOk(t, repo, "rev-parse", "refs/heads/human/missing-worker") != tc {
					t.Fatal("cleanup changed replacement registration branch")
				}
				path, _ := LeasePath(repo, rollSlug)
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("cleanup retained completed lease: %v", err)
				}
			})
		}
	}
}

func TestCleanupAbsentRegisteredThroughSymlinkRoot(t *testing.T) {
	repo, base := setupRepo(t)
	alias := filepath.Join(t.TempDir(), "checkout-alias")
	if err := os.Symlink(repo, alias); err != nil {
		t.Fatal(err)
	}
	lease, err := Create(CreateOpts{Root: alias, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Slices: []SlicePaths{{ID: "slice-a", Paths: []string{"src/a.go"}}}})
	if err != nil {
		t.Fatal(err)
	}
	worker := lease.Slices[0]
	tc := commitIn(t, worker.WorktreePath, "src/a.go", "Accepted")
	if _, err := RecordGreen(alias, rollSlug, "slice-a", tc); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Integrate(IntegrateOpts{Root: alias, Slug: rollSlug, ApplyToControl: true}); err != nil {
		t.Fatal(err)
	}
	canonicalCheckout, err := filepath.EvalSymlinks(worker.WorktreePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(worker.WorktreePath); err != nil {
		t.Fatal(err)
	}
	raw := gitOk(t, repo, "worktree", "list", "--porcelain", "-z")
	if !strings.Contains(raw, "worktree "+canonicalCheckout+"\x00") {
		t.Fatalf("fixture lost raw Git registration for %s: %q", canonicalCheckout, raw)
	}
	if results, err := Cleanup(alias, rollSlug, false); err != nil || len(results) != 1 || results[0].Worktree != worker.WorktreePath || results[0].Branch != worker.Branch {
		t.Fatalf("cleanup failed through missing aliased path: %+v %v", results, err)
	}
	raw = gitOk(t, repo, "worktree", "list", "--porcelain", "-z")
	if !strings.Contains(raw, "worktree "+canonicalCheckout+"\x00") {
		t.Fatalf("cleanup discarded missing worker raw Git registration: %q", raw)
	}
	if exists, err := branchExists(repo, worker.Branch); err != nil || !exists {
		t.Fatalf("cleanup discarded missing worker recovery ref: %v %v", exists, err)
	}
	path, _ := LeasePath(alias, rollSlug)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cleanup left completed lease: %v", err)
	}
}

func TestParallelOperationsPreserveUnrelatedMissingRegistration(t *testing.T) {
	for _, operation := range []string{"integrate-success", "integrate-failure", "cleanup", "prune"} {
		t.Run(operation, func(t *testing.T) {
			repo, base := setupRepo(t)
			greenBatch(t, repo, base, rollSlug, rollBatch)
			opts := IntegrateOpts{Root: repo, Slug: rollSlug, ApplyToControl: true}
			if operation == "cleanup" || operation == "prune" {
				if _, _, err := Integrate(opts); err != nil {
					t.Fatal(err)
				}
			}
			foreign := filepath.Join(t.TempDir(), "human-recovery")
			gitOk(t, repo, "worktree", "add", "-b", "human/recovery", foreign, base)
			if err := os.WriteFile(filepath.Join(foreign, "precious.txt"), []byte("uncommitted recovery data"), 0o644); err != nil {
				t.Fatal(err)
			}
			gitOk(t, foreign, "add", "precious.txt")
			canonical, err := filepath.EvalSymlinks(foreign)
			if err != nil {
				t.Fatal(err)
			}
			admin := gitOk(t, foreign, "rev-parse", "--absolute-git-dir")
			indexPath := filepath.Join(admin, "index")
			index, err := os.ReadFile(indexPath)
			if err != nil {
				t.Fatal(err)
			}
			head, err := os.ReadFile(filepath.Join(admin, "HEAD"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(foreign); err != nil {
				t.Fatal(err)
			}
			switch operation {
			case "integrate-success":
				// Reusing our clean staging checkout must not prune an
				// unrelated missing registration elsewhere in the repository.
				stage := WorkerWorktreePath(repo, rollBatch, "integrate")
				if err := worktreeAdd(repo, stage, IntegrateBranchName(rollSlug, rollBatch), base); err != nil {
					t.Fatal(err)
				}
				if _, _, err := Integrate(opts); err != nil {
					t.Fatal(err)
				}
			case "integrate-failure":
				lease := readLeaseT(t, repo)
				lease.Slices[0].TransferCommit = strings.Repeat("deadbeef", 5)
				path, _ := LeasePath(repo, rollSlug)
				if err := WriteLease(path, lease); err != nil {
					t.Fatal(err)
				}
				if _, _, err := Integrate(opts); err == nil {
					t.Fatal("expected invalid-transfer failure")
				}
			case "cleanup":
				if _, err := Cleanup(repo, rollSlug, false); err != nil {
					t.Fatal(err)
				}
			case "prune":
				if _, err := Prune(repo, rollSlug); err != nil {
					t.Fatal(err)
				}
			}
			raw := gitOk(t, repo, "worktree", "list", "--porcelain", "-z")
			if !strings.Contains(raw, "worktree "+canonical+"\x00") {
				t.Fatalf("operation discarded unrelated registration: %q", raw)
			}
			if after, err := os.ReadFile(indexPath); err != nil || string(after) != string(index) {
				t.Fatalf("operation discarded unrelated recovery index: %v", err)
			}
			if after, err := os.ReadFile(filepath.Join(admin, "HEAD")); err != nil || string(after) != string(head) {
				t.Fatalf("operation changed unrelated registration HEAD: %v", err)
			}
			if gitOk(t, repo, "rev-parse", "refs/heads/human/recovery") != base {
				t.Fatal("operation changed unrelated recovery ref")
			}
		})
	}
}

func TestCleanupMovedIntegratedWorkerPreservesIndexAndWorkingData(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprintf("force=%t", force), func(t *testing.T) {
			repo, base := setupRepo(t)
			lease, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Slices: []SlicePaths{{ID: "slice-a", Paths: []string{"src/a.go"}}}})
			if err != nil {
				t.Fatal(err)
			}
			worker := lease.Slices[0]
			tc := commitIn(t, worker.WorktreePath, "src/a.go", "Accepted")
			if _, err := RecordGreen(repo, rollSlug, "slice-a", tc); err != nil {
				t.Fatal(err)
			}
			if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, ApplyToControl: true}); err != nil {
				t.Fatal(err)
			}
			tracked := filepath.Join(worker.WorktreePath, "src/a.go")
			if err := os.WriteFile(tracked, []byte("distinct staged recovery bytes"), 0o644); err != nil {
				t.Fatal(err)
			}
			gitOk(t, worker.WorktreePath, "add", "src/a.go")
			if err := os.WriteFile(tracked, []byte("different unstaged working bytes"), 0o644); err != nil {
				t.Fatal(err)
			}
			admin := gitOk(t, worker.WorktreePath, "rev-parse", "--absolute-git-dir")
			indexPath := filepath.Join(admin, "index")
			index, err := os.ReadFile(indexPath)
			if err != nil {
				t.Fatal(err)
			}
			metadataPath := filepath.Join(admin, "custom-recovery")
			if err := os.WriteFile(metadataPath, []byte("human recovery metadata"), 0o600); err != nil {
				t.Fatal(err)
			}
			canonical, err := filepath.EvalSymlinks(worker.WorktreePath)
			if err != nil {
				t.Fatal(err)
			}
			moved := filepath.Join(t.TempDir(), "moved-worker")
			if err := os.Rename(worker.WorktreePath, moved); err != nil {
				t.Fatal(err)
			}
			results, err := Cleanup(repo, rollSlug, force)
			if err != nil || len(results) != 1 || results[0].Branch != worker.Branch || results[0].Worktree != worker.WorktreePath {
				t.Fatalf("cleanup omitted moved-worker recovery identity: %+v %v", results, err)
			}
			if after, err := os.ReadFile(indexPath); err != nil || string(after) != string(index) {
				t.Fatalf("cleanup changed exact recovery index bytes: %v", err)
			}
			if after, err := os.ReadFile(metadataPath); err != nil || string(after) != "human recovery metadata" {
				t.Fatalf("cleanup changed custom recovery metadata: %v", err)
			}
			if staged := gitOk(t, repo, "--git-dir="+admin, "show", ":src/a.go"); staged != "distinct staged recovery bytes" {
				t.Fatalf("cleanup lost staged recovery bytes: %q", staged)
			}
			if working, err := os.ReadFile(filepath.Join(moved, "src/a.go")); err != nil || string(working) != "different unstaged working bytes" {
				t.Fatalf("cleanup changed moved working data: %q %v", working, err)
			}
			raw := gitOk(t, repo, "worktree", "list", "--porcelain", "-z")
			if !strings.Contains(raw, "worktree "+canonical+"\x00") || gitOk(t, repo, "rev-parse", "refs/heads/"+worker.Branch) != tc {
				t.Fatalf("cleanup discarded moved-worker registration/ref: %q", raw)
			}
			path, _ := LeasePath(repo, rollSlug)
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("cleanup retained finished controller lease: %v", err)
			}
		})
	}
}

func TestCleanupPreviouslyPrunedIntegratedWorker(t *testing.T) {
	repo, base := setupRepo(t)
	lease, err := Create(CreateOpts{Root: repo, Slug: rollSlug, BatchID: rollBatch, BaseSHA: base, Slices: []SlicePaths{{ID: "slice-a", Paths: []string{"src/a.go"}}}})
	if err != nil {
		t.Fatal(err)
	}
	worker := lease.Slices[0]
	tc := commitIn(t, worker.WorktreePath, "src/a.go", "Accepted")
	if _, err := RecordGreen(repo, rollSlug, "slice-a", tc); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Integrate(IntegrateOpts{Root: repo, Slug: rollSlug, ApplyToControl: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := Prune(repo, rollSlug); err != nil {
		t.Fatal(err)
	}
	if results, err := Cleanup(repo, rollSlug, false); err != nil || len(results) != 0 {
		t.Fatalf("cleanup refused safely pruned worker: %+v %v", results, err)
	}
	if exists, err := branchExists(repo, worker.Branch); err != nil || exists {
		t.Fatalf("cleanup retained safely pruned accepted ref: %v %v", exists, err)
	}
	path, _ := LeasePath(repo, rollSlug)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cleanup retained finished lease: %v", err)
	}
}
