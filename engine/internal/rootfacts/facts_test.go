package rootfacts

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveNormalCheckoutAndOrdinaryAbsence(t *testing.T) {
	repo := initRepository(t, filepath.Join(t.TempDir(), "repo"))
	root := filepath.Join(repo, ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	facts, err := ResolveFrom(filepath.Join(repo, "nested"), "")
	if err == nil {
		t.Fatal("missing nested directory should fail before root resolution")
	}
	if err := os.MkdirAll(filepath.Join(repo, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	facts, err = ResolveFrom(filepath.Join(repo, "nested"), "")
	if err != nil {
		t.Fatal(err)
	}
	wantRoot, _ := physical(root)
	if facts.PhysicalRoot != wantRoot || facts.SelectionReason != "git-ancestor" {
		t.Fatalf("facts = %+v, want root %q selected by Git-bounded search", facts, wantRoot)
	}
	if facts.Git.TopLevel == "" || facts.Git.Dir == "" || facts.Git.CommonDir == "" {
		t.Fatalf("missing Git identities: %+v", facts.Git)
	}

	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	facts, err = ResolveFrom(repo, "")
	if !errors.Is(err, ErrNoRoot) || hazard(facts, "DRV-ROOT-NOT-FOUND") == nil || facts.Refuses() {
		t.Fatalf("ordinary absence facts=%+v err=%v", facts, err)
	}

	t.Setenv("DEVRITES_WORKSPACE", filepath.Join(t.TempDir(), "outside"))
	facts, err = ResolveFrom(repo, "")
	if !errors.Is(err, ErrUnsafeRoot) || hazard(facts, "DRV-WORKSPACE-WITHOUT-ROOT") == nil || !facts.Refuses() {
		t.Fatalf("workspace without root facts=%+v err=%v", facts, err)
	}
}

func TestResolveStopsAtNestedRepositoryAndMultiRepoBoundaries(t *testing.T) {
	parent := t.TempDir()
	outer := initRepository(t, filepath.Join(parent, "outer"))
	if err := os.MkdirAll(filepath.Join(outer, ".devrites"), 0o755); err != nil {
		t.Fatal(err)
	}
	child := initRepository(t, filepath.Join(outer, "services", "child"))

	facts, err := ResolveFrom(child, "")
	if !errors.Is(err, ErrUnsafeRoot) {
		t.Fatalf("nested repo error = %v, want unsafe root", err)
	}
	h := hazard(facts, "DRV-ROOT-OUTSIDE-GIT")
	if h == nil || !h.Refuse || h.Remediation == "" {
		t.Fatalf("nested repo hazard = %+v", h)
	}
	if facts.PhysicalRoot != "" {
		t.Fatalf("nested repository inherited parent root %q", facts.PhysicalRoot)
	}

	multi := t.TempDir()
	if err := os.MkdirAll(filepath.Join(multi, ".devrites"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := initRepository(t, filepath.Join(multi, "a"))
	_ = initRepository(t, filepath.Join(multi, "b"))
	facts, err = ResolveFrom(a, "")
	if !errors.Is(err, ErrUnsafeRoot) || hazard(facts, "DRV-ROOT-OUTSIDE-GIT") == nil {
		t.Fatalf("multi-repository parent facts=%+v err=%v", facts, err)
	}
}

func TestResolveKeepsRepositoryBoundaryWhenGitProbeIsUnavailable(t *testing.T) {
	outer := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outer, ".devrites"), 0o755); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(outer, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "")

	facts, err := ResolveFrom(repo, "")
	if !errors.Is(err, ErrUnsafeRoot) {
		t.Fatalf("Git-unavailable error = %v, want bounded refusal", err)
	}
	if hazard(facts, "DRV-GIT-FACTS-UNAVAILABLE") == nil || hazard(facts, "DRV-ROOT-OUTSIDE-GIT") == nil {
		t.Fatalf("Git-unavailable facts lost boundary diagnostics: %+v", facts)
	}
}

func TestGitPathIgnoresInheritedRepositoryTargets(t *testing.T) {
	target := initRepository(t, filepath.Join(t.TempDir(), "target repo"))
	poison := initRepository(t, filepath.Join(t.TempDir(), "poison repo"))
	t.Setenv("GIT_DIR", filepath.Join(poison, ".git"))
	t.Setenv("GIT_WORK_TREE", poison)
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.worktree")
	t.Setenv("GIT_CONFIG_VALUE_0", poison)
	t.Setenv("GIT_AUTHOR_NAME", "Retained Author")
	want, err := physical(target)
	if err != nil {
		t.Fatal(err)
	}

	got, ok := gitPath(target, "--show-toplevel")
	if !ok || got != want {
		t.Fatalf("gitPath() = %q, %v, want %q, true", got, ok, want)
	}
}

func TestResolveLinkedWorktreeUsesPerWorktreeRootAndCommonDir(t *testing.T) {
	base := t.TempDir()
	main := initRepository(t, filepath.Join(base, "main"))
	worktree := filepath.Join(base, "linked")
	runGit(t, main, "worktree", "add", "-q", "-b", "linked-test", worktree)
	root := filepath.Join(worktree, ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	facts, err := ResolveFrom(worktree, "")
	if err != nil {
		t.Fatal(err)
	}
	if !facts.Git.LinkedWorktree || facts.Git.Dir == facts.Git.CommonDir {
		t.Fatalf("linked worktree identities = %+v", facts.Git)
	}
	want, _ := physical(root)
	if facts.PhysicalRoot != want {
		t.Fatalf("root = %q, want per-worktree %q", facts.PhysicalRoot, want)
	}
}

func TestResolveSubmoduleNeverInheritsSuperprojectState(t *testing.T) {
	base := t.TempDir()
	origin := initRepository(t, filepath.Join(base, "origin"))
	super := initRepository(t, filepath.Join(base, "super"))
	runGit(t, super, "-c", "protocol.file.allow=always", "submodule", "add", "-q", origin, "deps/sub")
	sub := filepath.Join(super, "deps", "sub")
	if err := os.MkdirAll(filepath.Join(super, ".devrites"), 0o755); err != nil {
		t.Fatal(err)
	}

	facts, err := ResolveFrom(sub, "")
	if !errors.Is(err, ErrUnsafeRoot) || !facts.Git.Submodule || facts.Git.Superproject == "" {
		t.Fatalf("submodule without root facts=%+v err=%v", facts, err)
	}
	if facts.PhysicalRoot != "" || hazard(facts, "DRV-ROOT-OUTSIDE-GIT") == nil {
		t.Fatalf("submodule inherited superproject state: %+v", facts)
	}

	subRoot := filepath.Join(sub, ".devrites")
	if err := os.MkdirAll(subRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	facts, err = ResolveFrom(sub, "")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := physical(subRoot)
	if facts.PhysicalRoot != want || !facts.Git.Submodule {
		t.Fatalf("submodule local facts=%+v, want root %q", facts, want)
	}

	superFacts, err := ResolveFrom(super, "")
	if err != nil {
		t.Fatal(err)
	}
	if superFacts.Git.Submodule || superFacts.Git.Superproject != "" {
		t.Fatalf("superproject mislabeled as submodule: %+v", superFacts.Git)
	}
}

func TestResolveCanonicalizesExplicitSymlinkAndRejectsRootEscape(t *testing.T) {
	base := t.TempDir()
	real := initRepository(t, filepath.Join(base, "real"))
	if err := os.MkdirAll(filepath.Join(real, ".devrites"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "project-link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	facts, err := ResolveFrom(base, link)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := physical(filepath.Join(real, ".devrites"))
	if facts.LexicalRoot != filepath.Join(link, ".devrites") || facts.PhysicalRoot != want {
		t.Fatalf("symlink facts=%+v, want lexical link and physical root %q", facts, want)
	}

	escapedProject := filepath.Join(base, "escaped-project")
	if err := os.MkdirAll(escapedProject, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(base, "outside-root")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(escapedProject, ".devrites")); err != nil {
		t.Fatal(err)
	}
	facts, err = ResolveFrom(base, escapedProject)
	if !errors.Is(err, ErrUnsafeRoot) || hazard(facts, "DRV-ROOT-SYMLINK-ESCAPE") == nil {
		t.Fatalf("root symlink escape facts=%+v err=%v", facts, err)
	}
}

func TestResolveRejectsWorkspaceAndActiveSymlinkState(t *testing.T) {
	project := filepath.Join(t.TempDir(), "project")
	root := filepath.Join(project, ".devrites")
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(filepath.Join(root, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	workspaceLink := filepath.Join(root, "work", "escaped")
	if err := os.Symlink(outside, workspaceLink); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	facts, err := ResolveFrom(project, project)
	if !errors.Is(err, ErrUnsafeRoot) || hazard(facts, "DRV-WORKSPACE-SYMLINK-ESCAPE") == nil {
		t.Fatalf("workspace symlink facts=%+v err=%v", facts, err)
	}

	if err := os.Remove(workspaceLink); err != nil {
		t.Fatal(err)
	}
	activeTarget := filepath.Join(outside, "ACTIVE")
	if err := os.WriteFile(activeTarget, []byte("ghost\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(activeTarget, filepath.Join(root, "ACTIVE")); err != nil {
		t.Fatal(err)
	}
	facts, err = ResolveFrom(project, project)
	if !errors.Is(err, ErrUnsafeRoot) || hazard(facts, "DRV-ACTIVE-SYMLINK") == nil {
		t.Fatalf("ACTIVE symlink facts=%+v err=%v", facts, err)
	}
}

func TestResolveReportsStaleActiveAndExternalWorkspace(t *testing.T) {
	project := initRepository(t, filepath.Join(t.TempDir(), "repo"))
	root := filepath.Join(project, ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte("ghost\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	facts, err := ResolveFrom(project, project)
	if err != nil {
		t.Fatal(err)
	}
	h := hazard(facts, "DRV-ACTIVE-STALE")
	if h == nil || !strings.Contains(h.Remediation, "rm -f") {
		t.Fatalf("stale ACTIVE hazard = %+v", h)
	}

	t.Setenv("DEVRITES_WORKSPACE", filepath.Join(t.TempDir(), "ghost"))
	facts, err = ResolveFrom(project, project)
	if !errors.Is(err, ErrUnsafeRoot) {
		t.Fatalf("external workspace error = %v, want unsafe root", err)
	}
	h = hazard(facts, "DRV-WORKSPACE-OUTSIDE-ROOT")
	if h == nil || h.Remediation != "unset DEVRITES_WORKSPACE" || !h.Refuse {
		t.Fatalf("external workspace hazard = %+v", h)
	}

	facts, err = ResolveFrom(project, "")
	if !errors.Is(err, ErrUnsafeRoot) || hazard(facts, "DRV-WORKSPACE-OUTSIDE-ROOT") == nil {
		t.Fatalf("implicit external workspace facts=%+v err=%v", facts, err)
	}
}

func hazard(facts Facts, id string) *Hazard {
	for i := range facts.Hazards {
		if facts.Hazards[i].ID == id {
			return &facts.Hazards[i]
		}
	}
	return nil
}

func initRepository(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, path, "init", "-q")
	runGit(t, path, "config", "user.email", "devrites@example.invalid")
	runGit(t, path, "config", "user.name", "DevRites Test")
	if err := os.WriteFile(filepath.Join(path, ".fixture"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, path, "add", ".fixture")
	runGit(t, path, "commit", "-q", "-m", "fixture")
	return path
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}
