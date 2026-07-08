package main_test

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParityTestIntegrity sets up a real git repo with a committed test file, then
// a working-tree change that weakens (or doesn't) the test, diffs the repo against
// HEAD, and checks stdout + exit against the golden snapshot. Read-only w.r.t. git.
func TestParityTestIntegrity(t *testing.T) {
	requireGit(t)

	const baseline = "test('a', () => {\n  expect(1).toBe(1);\n  expect(2).toBe(2);\n});\n"

	// setup builds a fresh committed repo and returns its dir; mutate applies the
	// working-tree change under test (nil = leave clean).
	newRepo := func(t *testing.T) string {
		t.Helper()
		work := t.TempDir()
		initGitRepo(t, work)
		writeFile(t, work, "foo_test.js", baseline)
		gitCommitAll(t, work, "baseline")
		makeFeatureDir(t, work, "feat")
		return work
	}
	runCase := func(t *testing.T, work string, goArgs ...string) {
		c := parityCase{
			workdir: work,
			env:     libRootEnv(work),
			goArgs:  append([]string{"test-integrity"}, goArgs...),
		}
		c.assertEqual(t)
	}

	t.Run("clean", func(t *testing.T) {
		runCase(t, newRepo(t), "feat")
	})
	t.Run("skip-added", func(t *testing.T) {
		work := newRepo(t)
		writeFile(t, work, "foo_test.js", baseline+"it.skip('later', () => {});\n")
		runCase(t, work, "feat")
	})
	t.Run("assertion-dropped", func(t *testing.T) {
		work := newRepo(t)
		writeFile(t, work, "foo_test.js", "test('a', () => {\n  expect(1).toBe(1);\n});\n")
		runCase(t, work, "feat")
	})
	t.Run("deleted", func(t *testing.T) {
		work := newRepo(t)
		if err := os.Remove(filepath.Join(work, "foo_test.js")); err != nil {
			t.Fatal(err)
		}
		runCase(t, work, "feat")
	})
	t.Run("ghost-workspace", func(t *testing.T) {
		runCase(t, newRepo(t), "ghost")
	})
	t.Run("not-git", func(t *testing.T) {
		work := t.TempDir() // no initGitRepo -> gate skipped
		makeFeatureDir(t, work, "feat")
		runCase(t, work, "feat")
	})
}
