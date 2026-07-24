package main_test

// TestParityLearnings checks the learnings command against the golden snapshots.
// Learning state lives at the root selected by DEVRITES_ROOT. The mine and nudge
// fixtures use counts of 3 and 2 so sorting never depends on a platform-specific
// tie.

import (
	"path/filepath"
	"testing"
)

func TestParityLearnings(t *testing.T) {
	learn := func(a ...string) []string { return append([]string{"learnings"}, a...) }

	// Add to a fresh workspace. Bash creates the ledger and a dated entry, then Go
	// appends another. Only stdout ("learnings: recorded.") and exit 0 belong to
	// the contract because the dated file entry is nondeterministic.
	addw := t.TempDir()
	mkdirAllT(t, addw, ".devrites")
	t.Run("add", func(t *testing.T) {
		(parityCase{
			workdir: addw, env: libRootEnv(addw),
			goArgs: learn("add", "my-slug", "a recurring correction", "note"),
		}).assertEqual(t)
	})

	// List a prepared ledger. Both implementations print the file unchanged.
	listw := t.TempDir()
	writeFile(t, listw, filepath.Join(".devrites", "learnings.md"),
		"# DevRites learnings ledger\n\n"+
			"- [2026-01-01] (note · feat-x) prefer explicit error handling\n"+
			"- [2026-01-02] (bug · feat-y) the linter false-positive recurs\n")
	t.Run("list", func(t *testing.T) {
		(parityCase{
			workdir: listw, env: libRootEnv(listw),
			goArgs: learn("list"),
		}).assertEqual(t)
	})

	// Mine and nudge use two archived features. Normalization strips IDs, paths,
	// and numbers, so "recurring finding" has count 3 and "second dead end" has
	// count 2. The distinct counts keep sort order stable.
	arch := t.TempDir()
	writeFile(t, arch, filepath.Join(".devrites", "archive", "feat-a", "decisions.md"),
		"# decisions for feat-a\n"+
			"- recurring finding: stale cache after deploy 3 times on `foo.js`\n"+
			"- recurring finding: stale cache after deploy 8 times on `baz.ts`\n"+
			"- second dead end: retry backoff exceeded 5 attempts\n"+
			"some prose line that does not match the grep at all\n")
	writeFile(t, arch, filepath.Join(".devrites", "archive", "feat-b", "decisions.md"),
		"# decisions for feat-b\n"+
			"- recurring finding: stale cache after deploy 7 times on `bar.js`\n"+
			"- second dead end: retry backoff exceeded 9 attempts\n"+
			"- a one-off dismiss note about the flaky linter run\n")

	// Both implementations resolve the default archive to the same directory.
	t.Run("mine", func(t *testing.T) {
		(parityCase{
			workdir: arch, env: libRootEnv(arch),
			goArgs: learn("mine"),
		}).assertEqual(t)
	})

	// Both implementations resolve an explicit relative path against workdir, so
	// the missing path in the message is identical.
	t.Run("mine-no-archive", func(t *testing.T) {
		(parityCase{
			workdir: arch, env: libRootEnv(arch),
			goArgs: learn("mine", "no-such-archive"),
		}).assertEqual(t)
	})

	// Two features, three occurrences, and no review marker produce a nudge.
	t.Run("nudge", func(t *testing.T) {
		(parityCase{
			workdir: arch, env: libRootEnv(arch),
			goArgs: learn("nudge"),
		}).assertEqual(t)
	})

	// An unknown command writes usage to stderr and exits 2 without stdout.
	t.Run("unknown", func(t *testing.T) {
		(parityCase{
			workdir: arch, env: libRootEnv(arch),
			goArgs: learn("bogus"),
		}).assertEqual(t)
	})
}
