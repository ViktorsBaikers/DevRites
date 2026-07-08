package main_test

// TestParityLearnings checks the learnings command against the golden snapshots.
// learnings state is ROOT-LEVEL, read from DEVRITES_ROOT=work/.devrites when the
// case runs in workdir=work with libRootEnv(work). The mine/nudge pipelines are
// deterministic from fixtures. Fixture counts are DISTINCT (3 vs 2) so ordering
// never depends on a platform-specific tie-break.

import (
	"path/filepath"
	"testing"
)

func TestParityLearnings(t *testing.T) {
	learn := func(a ...string) []string { return append([]string{"learnings"}, a...) }

	// --- add: fresh workspace. bash creates the ledger + a dated entry; Go then
	// appends a second entry. Only stdout ("learnings: recorded.") + exit 0 are the
	// contract — the dated file line is a nondeterministic side effect, not stdout.
	addw := t.TempDir()
	t.Run("add", func(t *testing.T) {
		(parityCase{
			workdir: addw, env: libRootEnv(addw),
			goArgs: learn("add", "my-slug", "a recurring correction", "note"),
		}).assertEqual(t)
	})

	// --- list: a pre-written ledger; both sides cat the same file verbatim.
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

	// --- mine + nudge: two archived features. The shared "recurring finding" bullet
	// normalizes identically across features despite differing numbers/paths (the sed
	// id/path/number stripping), giving count 3; the "second dead end" bullet gives
	// count 2. Distinct counts ⇒ deterministic order on any `sort`.
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

	// mine: default archive (bash .devrites/archive, Go <root>/archive — same dir).
	t.Run("mine", func(t *testing.T) {
		(parityCase{
			workdir: arch, env: libRootEnv(arch),
			goArgs: learn("mine"),
		}).assertEqual(t)
	})

	// mine: no archive. An explicit relative path both sides resolve against workdir,
	// so the path embedded in the message is identical for bash and Go.
	t.Run("mine-no-archive", func(t *testing.T) {
		(parityCase{
			workdir: arch, env: libRootEnv(arch),
			goArgs: learn("mine", "no-such-archive"),
		}).assertEqual(t)
	})

	// nudge: >=2 features + a class recurring 3x + no review marker ⇒ the nudge line.
	t.Run("nudge", func(t *testing.T) {
		(parityCase{
			workdir: arch, env: libRootEnv(arch),
			goArgs: learn("nudge"),
		}).assertEqual(t)
	})

	// --- unknown cmd: usage to stderr, exit 2 (stdout empty on both sides).
	t.Run("unknown", func(t *testing.T) {
		(parityCase{
			workdir: arch, env: libRootEnv(arch),
			goArgs: learn("bogus"),
		}).assertEqual(t)
	})
}
