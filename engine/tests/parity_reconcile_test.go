package main_test

import (
	"fmt"
	"strings"
	"testing"
)

// TestParityReconcile checks reconcile against the golden snapshots: snapshot
// captures the worktree as a tree object, check diffs it and flags changes
// outside the orchestrator-authored allowlist, and close ends the retained
// post-slice baseline. It runs Git in a real repository, so stdout and exit are
// captured from the live command.
func TestParityReconcile(t *testing.T) {
	requireGit(t)

	newRepo := func(t *testing.T) string {
		t.Helper()
		work := t.TempDir()
		initGitRepo(t, work)
		writeFile(t, work, "src/a.js", "console.log(1)\n")
		writeFile(t, work, "src/b.js", "console.log(2)\n")
		gitCommitAll(t, work, "baseline")
		makeFeatureDir(t, work, "feat")
		return work
	}
	// snapshot is also a setup step for the check-* cases, so it snapshots under a
	// distinct key to avoid colliding with check's golden in the same subtest.
	snapshot := func(t *testing.T, work string, allowed ...string) {
		body := ""
		if len(allowed) > 0 {
			body = strings.Join(allowed, "\n") + "\n"
		}
		writeFeatureFile(t, work, "feat", ".wright-allowlist", body)
		out, code := runArgv(t, work, libRootEnv(work), "", binPath, "reconcile", "snapshot", "feat")
		assertGoldenKey(t, t.Name()+"/snapshot", fmt.Sprintf("exit %d\n%s", code, out))
	}
	check := func(t *testing.T, work string) {
		out, code := runArgv(t, work, libRootEnv(work), "", binPath, "reconcile", "check", "feat")
		assertGolden(t, out, code)
	}

	t.Run("snapshot", func(t *testing.T) { snapshot(t, newRepo(t)) })

	t.Run("check-clean", func(t *testing.T) {
		work := newRepo(t)
		snapshot(t, work, "src/a.js")
		writeFile(t, work, "src/a.js", "console.log(999)\n") // the wright's change
		check(t, work)
	})

	t.Run("check-violation", func(t *testing.T) {
		work := newRepo(t)
		snapshot(t, work, "src/a.js")
		writeFile(t, work, "src/a.js", "console.log(999)\n")
		writeFile(t, work, "src/b.js", "console.log(888)\n") // Outside the allowlist; an A1 breach.
		check(t, work)
	})

	t.Run("check-no-base", func(t *testing.T) {
		work := newRepo(t)
		check(t, work) // No snapshot, so exit 6.
	})

	t.Run("snapshot-no-allowlist", func(t *testing.T) {
		work := newRepo(t)
		out, code := runArgv(t, work, libRootEnv(work), "", binPath, "reconcile", "snapshot", "feat")
		assertGolden(t, out, code)
	})

	t.Run("inline-fallback", func(t *testing.T) {
		work := newRepo(t)
		snapshot(t, work)
		out, code := runArgv(t, work, libRootEnv(work), "", binPath, "reconcile", "close", "feat")
		assertGolden(t, out, code)
	})

	t.Run("not-git", func(t *testing.T) {
		work := t.TempDir()
		makeFeatureDir(t, work, "feat")
		check(t, work) // exit 0
	})

	t.Run("bad-mode", func(t *testing.T) {
		work := newRepo(t)
		(parityCase{
			workdir: work, env: libRootEnv(work),
			goArgs: []string{"reconcile", "bogus", "feat"},
		}).assertEqual(t)
	})
}
