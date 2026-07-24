package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// skipMarkers matches a test that has been muted: a skip/focus/ignore marker
// across the common test frameworks. assertMarkers matches an assertion. Both are
// counted per file to detect a test being weakened rather than fixed.
var (
	skipMarkers   = regexp.MustCompile(`(\bit|\bdescribe|\btest|\bcontext)\.skip|\bxit\b|\bxdescribe\b|\.only\b|@pytest\.mark\.skip|@unittest\.skip|\bpytest\.skip\b|t\.Skip\(|#\[ignore\]|@Ignore\b|@Disabled\b|\bfdescribe\b|\bfit\(`)
	assertMarkers = regexp.MustCompile(`assert|expect\(|\.should\b|\brequire\.|EXPECT_|ASSERT_|XCTAssert|t\.Error|t\.Fatal`)
)

// TestIntegrity detects tests that were weakened instead of fixing the code. It
// compares test files in the retained reconcile tree with the current tree, or
// uses HEAD as the base outside a slice lifecycle. It does not modify the
// workspace at <root>/features/<slug>.
//
//	0  clean, or skipped because this is not a git repo
//	2  no active workspace, bad args, or missing/corrupt baseline state
//	3  WEAKENED: a test was deleted, skipped, or de-asserted since the base
func TestIntegrity(root string, args []string, stdout, stderr io.Writer) int {
	// Accept both `test-integrity [slug]` and the `check [slug]` alias; a leading
	// dash is treated as "no slug".
	slug := ""
	switch first := argAt(args, 0); {
	case first == "check":
		slug = argAt(args, 1)
	case first == "" || strings.HasPrefix(first, "-"):
		slug = ""
	default:
		slug = first
	}
	if slug == "" {
		slug = activeSlug(root)
	}
	if slug == "" {
		fmt.Fprintln(stderr, "test-integrity: no active workspace (pass a slug or set .devrites/ACTIVE).")
		return 2
	}

	d := featureDir(root, slug)
	if !isDir(d) {
		fmt.Fprintf(stderr, "test-integrity: no workspace at %s.\n", d)
		return 2
	}

	cwd, _ := os.Getwd()
	gitRoot := gitToplevel(cwd)
	if gitRoot == "" {
		fmt.Fprintln(stderr, "test-integrity: not a git repo: gate skipped; inspect the test diff manually.")
		return 0
	}

	trees, err := captureSliceTreeRange(gitRoot, root, d)
	if err != nil {
		fmt.Fprintf(stderr, "test-integrity: cannot capture slice range: %v\n", err)
		return 2
	}
	defer trees.cleanup()
	changedPaths, err := changedTreePaths(gitRoot, trees.env, trees.base, trees.current)
	if err != nil {
		fmt.Fprintf(stderr, "test-integrity: cannot compare slice trees: %v\n", err)
		return 2
	}

	weak := 0
	srcChanged, testChanged := 0, 0
	var report strings.Builder
	for _, f := range changedPaths {
		if f == "" {
			continue
		}
		if !isTestFile(f) {
			if isSourceFile(f) {
				srcChanged++
			}
			continue
		}
		testChanged++
		oldBytes, oldExists, err := treeFileContent(gitRoot, trees.env, trees.base, f)
		if err != nil {
			fmt.Fprintf(stderr, "test-integrity: cannot read baseline %s: %v\n", f, err)
			return 2
		}
		if !oldExists || len(oldBytes) == 0 {
			continue // new test file: adding tests is never a weakening
		}
		newBytes, newExists, err := treeFileContent(gitRoot, trees.env, trees.current, f)
		if err != nil {
			fmt.Fprintf(stderr, "test-integrity: cannot read current %s: %v\n", f, err)
			return 2
		}
		if !newExists {
			weak++
			fmt.Fprintf(&report, "  - %s: test file DELETED\n", f)
			continue
		}
		old := string(oldBytes)
		now := string(newBytes)
		oldSkips, newSkips := countMatches(skipMarkers, old), countMatches(skipMarkers, now)
		oldAsserts, newAsserts := countMatches(assertMarkers, old), countMatches(assertMarkers, now)
		switch {
		case newSkips > oldSkips:
			weak++
			fmt.Fprintf(&report, "  - %s: skip/focus markers added (%d→%d)\n", f, oldSkips, newSkips)
		case newAsserts < oldAsserts:
			weak++
			fmt.Fprintf(&report, "  - %s: assertions dropped (%d→%d)\n", f, oldAsserts, newAsserts)
		}
	}

	// When source changes without a test change, print the verification-gap
	// advisory from testing.md. This is not a weakening verdict and appears on
	// both clean and weakened paths.
	gapAdvisory := ""
	if srcChanged > 0 && testChanged == 0 {
		gapAdvisory = fmt.Sprintf("test-integrity: advisory: %d source file(s) changed, 0 test file(s) touched; "+
			"run the verification-gap trace (testing.md) to confirm each changed behavior is asserted.", srcChanged)
	}

	if weak > 0 {
		fmt.Fprintf(stderr, "test-integrity: WEAKENED: %d test file(s) lost coverage since the slice base:\n", weak)
		fmt.Fprint(stderr, report.String())
		fmt.Fprintln(stderr, "Weakening a test to pass the gate is a Critical finding. Restore the test, then fix the code or raise a blocking question.")
		if gapAdvisory != "" {
			fmt.Fprintln(stderr, gapAdvisory)
		}
		return 3
	}
	fmt.Fprintln(stdout, "test-integrity: OK: no test deleted, skipped, or de-asserted since the base.")
	if gapAdvisory != "" {
		fmt.Fprintln(stdout, gapAdvisory)
	}
	return 0
}

// isSourceFile recognises a path as project source worth pairing with a test:
// a code file that is not itself a test. It filters out the common non-code
// changes (docs, config, lockfiles) so the verification-gap advisory keys on
// behavior-bearing edits, not a README or a manifest bump.
func isSourceFile(f string) bool {
	ext := filepath.Ext(f)
	switch ext {
	case ".go", ".js", ".jsx", ".ts", ".tsx", ".py", ".rb", ".rs", ".java", ".kt",
		".c", ".h", ".cc", ".cpp", ".hpp", ".cs", ".php", ".swift", ".scala", ".m", ".mm":
		return true
	default:
		return false
	}
}

// testDirSegments are directory names that mark everything beneath them as test
// code, across the common ecosystems (JS/TS, Python, Go, Rust, JUnit).
var testDirSegments = map[string]bool{
	"test": true, "tests": true, "spec": true, "specs": true,
	"__tests__": true, "__test__": true,
}

// isTestFile recognises a path as a test by its filename/directory conventions.
// Markers must land on a basename or a whole path segment: matching "test" or
// "spec" as a bare substring would classify ordinary source ("internal/latest/",
// "internal/respect/") as tests, deleting them would then trip the WEAKENED gate,
// and they would never count toward the verification-gap advisory.
func isTestFile(f string) bool {
	f = filepath.ToSlash(f)
	base := filepath.Base(f)

	for _, marker := range []string{
		"_test.", ".test.", "_tests.", ".tests.",
		"_spec.", ".spec.", "_specs.", ".specs.",
		"Test.", "Tests.", "Spec.", "Specs.",
	} {
		if strings.Contains(base, marker) {
			return true
		}
	}
	if strings.HasPrefix(base, "test_") || strings.HasPrefix(base, "spec_") {
		return true
	}

	for _, seg := range strings.Split(filepath.ToSlash(filepath.Dir(f)), "/") {
		if testDirSegments[seg] {
			return true
		}
	}
	return false
}

// countMatches counts the non-overlapping matches of re in text.
func countMatches(re *regexp.Regexp, text string) int {
	return len(re.FindAllStringIndex(text, -1))
}
