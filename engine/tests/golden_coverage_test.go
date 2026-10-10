package main_test

// Every file under testdata/golden must be read by an executed golden
// assertion, otherwise a stale snapshot sits in the tree and guards nothing.
//
// The exercised set is observed, not guessed: the rest of this package's tests
// run in a throwaway copy of the engine tree with an empty golden directory and
// UPDATE_GOLDEN=1, so assertGoldenKey writes exactly the keys it was asked for.

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	goldenChildEnv = "DEVRITES_GOLDEN_COVERAGE_CHILD"

	// No test reads these files and no live comparison can regenerate them:
	// the engine rejects each recorded command as unknown (or has no package
	// gate), and each status report records a result no workspace can produce
	// under the current phase policy (see the two status reasons). They stay on
	// disk until the owner prunes them.
	reasonLearnings   = "no `learnings` command: the engine rejects it as unknown"
	reasonProgress    = "no `progress` command: the engine rejects it as unknown"
	reasonPreamble    = "no `preamble` command: the engine rejects it as unknown"
	reasonPackageGate = "no engine code implements a package-existence gate"
	// Both reports match Render()'s layout; only their content differs.
	reasonStatusAuth   = "result line `incomplete (missing: tasks)`: Render() prints `missing files:` and lists workspace file names, and the build phase requires many more files than the four sections shown"
	reasonStatusSearch = "no state.md and an empty decisions section: the phase is read from state.md, and the spec phase requires state.md and decisions.md, so Render() cannot print this report"
)

// unreachableGoldens maps each golden no test can read to the reason. Keep it
// minimal: an entry that becomes readable fails the test as stale.
var unreachableGoldens = map[string]string{
	"TestParityLearnings/add":                    reasonLearnings,
	"TestParityLearnings/list":                   reasonLearnings,
	"TestParityLearnings/mine-no-archive":        reasonLearnings,
	"TestParityLearnings/mine":                   reasonLearnings,
	"TestParityLearnings/nudge":                  reasonLearnings,
	"TestParityLearnings/unknown":                reasonLearnings,
	"TestParityProgress/arg=allbuilt":            reasonProgress,
	"TestParityProgress/arg=done":                reasonProgress,
	"TestParityProgress/arg=ghost":               reasonProgress,
	"TestParityProgress/arg=mid":                 reasonProgress,
	"TestParityProgress/arg=nophase":             reasonProgress,
	"TestParityProgress/arg=noslice":             reasonProgress,
	"TestParityProgress/arg=plan":                reasonProgress,
	"TestParityProgress/arg=seal":                reasonProgress,
	"TestParityPackageExistence/ghost-workspace": reasonPackageGate,
	"TestParityPackageExistence/no-manifest":     reasonPackageGate,
	"TestParityPackageExistence/not-git":         reasonPackageGate,
	"TestParityPackageExistence/undeclared":      reasonPackageGate,
	"TestParityPreamble/afk/a":                   reasonPreamble,
	"TestParityPreamble/hitl/active":             reasonPreamble,
	"TestParityPreamble/hitl/bare":               reasonPreamble,
	"TestParityPreamble/hitl/full":               reasonPreamble,
	"TestParityPreamble/hitl/ghost":              reasonPreamble,
	"TestParityPreamble/hitl/nonl":               reasonPreamble,
	"TestParityPreamble/hitl/openend":            reasonPreamble,
	"status/auth-tokens.txt":                     reasonStatusAuth,
	"status/search-ranking.txt":                  reasonStatusSearch,
}

// goldenKeys lists the keys of every file under root: the slash path, with a
// trailing ".golden" removed. Any other file is keyed with its full name, so a
// stray non-golden file is reported as unread too.
func goldenKeys(t *testing.T, root string) map[string]bool {
	t.Helper()
	keys := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && path == root {
				return nil
			}
			return err
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(root, path)
			keys[strings.TrimSuffix(filepath.ToSlash(rel), ".golden")] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return keys
}

// copyTree copies src to dst, skipping the built binary and the golden tree.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if rel == "devrites" || rel == filepath.Join("testdata", "golden") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoldenTreeIsFullyRead(t *testing.T) {
	if os.Getenv(goldenChildEnv) == "1" {
		t.Skip("running inside the coverage child")
	}
	if testing.Short() {
		t.Skip("copies the engine tree and reruns the package tests")
	}

	all := goldenKeys(t, filepath.Join(engineRoot, "testdata", "golden"))
	if len(all) == 0 {
		t.Fatal("golden tree is empty: coverage would pass vacuously")
	}

	work := t.TempDir()
	copyTree(t, engineRoot, filepath.Join(work, "engine"))
	docs := filepath.Join(engineRoot, "..", "docs", "engine")
	if _, err := os.Stat(docs); err == nil {
		copyTree(t, docs, filepath.Join(work, "docs", "engine"))
	}

	child := exec.Command("go", "test", "./tests/", "-count=1", "-skip", "^TestGoldenTreeIsFullyRead$")
	child.Dir = filepath.Join(work, "engine")
	child.Env = append(os.Environ(), "UPDATE_GOLDEN=1", goldenChildEnv+"=1")
	if out, err := child.CombinedOutput(); err != nil {
		t.Fatalf("package tests failed in the coverage copy: %v\n%s", err, out)
	}

	read := goldenKeys(t, filepath.Join(work, "engine", "testdata", "golden"))
	if len(read) == 0 {
		t.Fatal("no golden was read by any test: the observation is vacuous")
	}

	var unread, stale []string
	for key := range all {
		if !read[key] && unreachableGoldens[key] == "" {
			unread = append(unread, key)
		}
	}
	for key := range unreachableGoldens {
		if read[key] {
			stale = append(stale, key)
		}
		if !all[key] {
			stale = append(stale, key+" (no such golden)")
		}
	}
	sort.Strings(unread)
	sort.Strings(stale)
	if len(unread) > 0 {
		t.Errorf("goldens no test reads (restore a live comparison, or list in unreachableGoldens with a reason):\n  %s", strings.Join(unread, "\n  "))
	}
	if len(stale) > 0 {
		t.Errorf("unreachableGoldens entries that are now read or missing; remove them:\n  %s", strings.Join(stale, "\n  "))
	}
}
