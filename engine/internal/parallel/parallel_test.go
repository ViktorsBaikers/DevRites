package parallel

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitOk(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_AUTHOR_NAME=DevRites Test",
		"GIT_AUTHOR_EMAIL=devrites@example.invalid",
		"GIT_COMMITTER_NAME=DevRites Test",
		"GIT_COMMITTER_EMAIL=devrites@example.invalid",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func setupRepo(t *testing.T) (repo, base string) {
	t.Helper()
	repo = t.TempDir()
	gitOk(t, repo, "init")
	gitOk(t, repo, "checkout", "-b", "main")
	if err := os.MkdirAll(filepath.Join(repo, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "src", "a.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "src", "b.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOk(t, repo, "add", ".")
	gitOk(t, repo, "commit", "-m", "base")
	base = gitOk(t, repo, "rev-parse", "HEAD")
	return repo, base
}

func TestLeaseRoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "parallel-lease.md")
	lease := &Lease{
		BatchID:             "batch1",
		CreatedAt:           "2026-08-23T00:00:00Z",
		BaseSHA:             "abcdef1",
		N:                   2,
		Status:              StatusRunning,
		ControlPIDOrSession: "test",
		Slices: []LeaseSlice{
			{ID: "slice-a", Paths: []string{"src/a.go"}, WorktreePath: "/tmp/a", Branch: "devrites/parallel/x/batch1/slice-a", WrightStatus: WrightPending},
			{ID: "slice-b", Paths: []string{"src/b.go"}, WorktreePath: "/tmp/b", Branch: "devrites/parallel/x/batch1/slice-b", WrightStatus: WrightPending},
		},
	}
	if err := WriteLease(path, lease); err != nil {
		t.Fatal(err)
	}
	got, err := ReadLease(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.BatchID != "batch1" || got.Status != StatusRunning || len(got.Slices) != 2 {
		t.Fatalf("unexpected lease: %#v", got)
	}
}

func TestCreateAbortCleanup(t *testing.T) {
	repo, base := setupRepo(t)
	slug := "demo-feature"
	batch := "batch1"
	lease, err := Create(CreateOpts{
		Root:    repo,
		Slug:    slug,
		BatchID: batch,
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if lease.Status != StatusRunning || lease.N != 2 {
		t.Fatalf("lease=%#v", lease)
	}
	wtA := lease.Slices[0].WorktreePath
	wtB := lease.Slices[1].WorktreePath
	for _, wt := range []string{wtA, wtB} {
		if _, err := os.Stat(wt); err != nil {
			t.Fatalf("missing worktree %s: %v", wt, err)
		}
		if head := gitOk(t, wt, "rev-parse", "HEAD"); head != base {
			t.Fatalf("worktree head %s want %s", head, base)
		}
	}
	if head := gitOk(t, repo, "rev-parse", "HEAD"); head != base {
		t.Fatalf("control moved after create: %s", head)
	}

	// Drift control tip, then abort should restore base.
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("drift\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOk(t, repo, "add", "README.md")
	gitOk(t, repo, "commit", "-m", "drift")
	if _, err := Abort(repo, slug, true); err != nil {
		t.Fatal(err)
	}
	if head := gitOk(t, repo, "rev-parse", "HEAD"); head != base {
		t.Fatalf("abort left control at %s want %s", head, base)
	}
	leasePath, _ := LeasePath(repo, slug)
	lease, err = ReadLease(leasePath)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Status != StatusAborted {
		t.Fatalf("status=%s", lease.Status)
	}
	if _, err := os.Stat(wtA); err != nil {
		t.Fatalf("abort should preserve worktree: %v", err)
	}

	if err := Cleanup(repo, slug, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wtA); !os.IsNotExist(err) {
		t.Fatalf("cleanup should remove worktree A")
	}
	if _, err := os.Stat(leasePath); !os.IsNotExist(err) {
		t.Fatalf("cleanup should clear lease")
	}
}

func TestCreateAcceptsFourSlices(t *testing.T) {
	t.Parallel()
	repo, _ := setupRepo(t)
	for _, name := range []string{"c.go", "d.go"} {
		if err := os.WriteFile(filepath.Join(repo, "src", name), []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitOk(t, repo, "add", ".")
	gitOk(t, repo, "commit", "-m", "more files")
	base := gitOk(t, repo, "rev-parse", "HEAD")
	lease, err := Create(CreateOpts{
		Root:    repo,
		Slug:    "demo-four",
		BatchID: "batch4",
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go"}},
			{ID: "slice-c", Paths: []string{"src/c.go"}},
			{ID: "slice-d", Paths: []string{"src/d.go"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if lease.N != 4 || len(lease.Slices) != 4 {
		t.Fatalf("want 4 slices, got n=%d slices=%d", lease.N, len(lease.Slices))
	}
}

func TestCreateRefusesCountOutsideBounds(t *testing.T) {
	t.Parallel()
	repo, base := setupRepo(t)
	_, err := Create(CreateOpts{
		Root:    repo,
		Slug:    "demo-one",
		BatchID: "batch1",
		BaseSHA: base,
		Slices:  []SlicePaths{{ID: "slice-a", Paths: []string{"src/a.go"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "2-10 slices") {
		t.Fatalf("expected 2-10 bound for N=1, got %v", err)
	}

	slices := make([]SlicePaths, MaxParallelSlices+1)
	for i := range slices {
		name := fmt.Sprintf("src/x%d.go", i)
		if err := os.WriteFile(filepath.Join(repo, name), []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		slices[i] = SlicePaths{ID: fmt.Sprintf("s%d", i), Paths: []string{name}}
	}
	gitOk(t, repo, "add", ".")
	gitOk(t, repo, "commit", "-m", "eleven")
	base = gitOk(t, repo, "rev-parse", "HEAD")
	_, err = Create(CreateOpts{
		Root:    repo,
		Slug:    "demo-eleven",
		BatchID: "batch11",
		BaseSHA: base,
		Slices:  slices,
	})
	if err == nil || !strings.Contains(err.Error(), "2-10 slices") {
		t.Fatalf("expected 2-10 bound for N=11, got %v", err)
	}
}

func TestValidateLeaseBounds(t *testing.T) {
	t.Parallel()
	ok := validTestLease(MaxParallelSlices)
	if err := ValidateLease(ok); err != nil {
		t.Fatalf("N=%d should validate: %v", MaxParallelSlices, err)
	}
	tooMany := validTestLease(MaxParallelSlices + 1)
	if err := ValidateLease(tooMany); err == nil || !strings.Contains(err.Error(), "2-10 slices") {
		t.Fatalf("N=%d should refuse, got %v", MaxParallelSlices+1, err)
	}
}

func validTestLease(n int) *Lease {
	slices := make([]LeaseSlice, n)
	for i := range slices {
		id := fmt.Sprintf("s%d", i)
		slices[i] = LeaseSlice{
			ID:           id,
			Paths:        []string{fmt.Sprintf("src/%s.go", id)},
			WorktreePath: "/tmp/" + id,
			Branch:       "devrites/parallel/x/batch/" + id,
			WrightStatus: WrightPending,
		}
	}
	return &Lease{
		BatchID:             "batch1",
		CreatedAt:           "2026-08-23T00:00:00Z",
		BaseSHA:             "abcdef1",
		N:                   n,
		Status:              StatusRunning,
		ControlPIDOrSession: "test",
		Slices:              slices,
	}
}

func TestCreateRefusesOverlap(t *testing.T) {
	repo, base := setupRepo(t)
	_, err := Create(CreateOpts{
		Root:    repo,
		Slug:    "demo",
		BatchID: "batch1",
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/a.go"}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "overlap") {
		t.Fatalf("expected overlap refusal, got %v", err)
	}
}

func TestIntegrateDivergentSiblings(t *testing.T) {
	repo, base := setupRepo(t)
	slug := "demo-feature"
	batch := "batch1"
	lease, err := Create(CreateOpts{
		Root:    repo,
		Slug:    slug,
		BatchID: batch,
		BaseSHA: base,
		Slices: []SlicePaths{
			{ID: "slice-a", Paths: []string{"src/a.go"}},
			{ID: "slice-b", Paths: []string{"src/b.go"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	wtA := lease.Slices[0].WorktreePath
	wtB := lease.Slices[1].WorktreePath

	if err := os.WriteFile(filepath.Join(wtA, "src", "a.go"), []byte("package main\n\nfunc A() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOk(t, wtA, "add", "src/a.go")
	gitOk(t, wtA, "commit", "-m", "slice-a green")
	tcA := gitOk(t, wtA, "rev-parse", "HEAD")

	if err := os.WriteFile(filepath.Join(wtB, "src", "b.go"), []byte("package main\n\nfunc B() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOk(t, wtB, "add", "src/b.go")
	gitOk(t, wtB, "commit", "-m", "slice-b green")
	tcB := gitOk(t, wtB, "rev-parse", "HEAD")

	if _, err := RecordGreen(repo, slug, "slice-a", tcA); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordGreen(repo, slug, "slice-b", tcB); err != nil {
		t.Fatal(err)
	}

	tip, lease, err := Integrate(IntegrateOpts{Root: repo, Slug: slug, ApplyToControl: true})
	if err != nil {
		t.Fatal(err)
	}
	if lease.Status != StatusComplete {
		t.Fatalf("status=%s", lease.Status)
	}
	if tip == "" || tip == base {
		t.Fatalf("expected advanced tip, got %s", tip)
	}
	head := gitOk(t, repo, "rev-parse", "HEAD")
	if head != tip {
		t.Fatalf("control head %s want tip %s", head, tip)
	}
	a := mustRead(t, filepath.Join(repo, "src", "a.go"))
	b := mustRead(t, filepath.Join(repo, "src", "b.go"))
	if !strings.Contains(a, "func A") || !strings.Contains(b, "func B") {
		t.Fatalf("integrated contents missing: a=%q b=%q", a, b)
	}

	if err := Cleanup(repo, slug, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wtA); !os.IsNotExist(err) {
		t.Fatalf("cleanup should remove workers")
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
