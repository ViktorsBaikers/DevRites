package lib

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// skipMarkers matches a test that has been muted — a skip/focus/ignore marker
// across the common test frameworks. assertMarkers matches an assertion. Both are
// counted per file to detect a test being weakened rather than fixed.
var (
	skipMarkers   = regexp.MustCompile(`(\bit|\bdescribe|\btest|\bcontext)\.skip|\bxit\b|\bxdescribe\b|\.only\b|@pytest\.mark\.skip|@unittest\.skip|\bpytest\.skip\b|t\.Skip\(|#\[ignore\]|@Ignore\b|@Disabled\b|\bfdescribe\b|\bfit\(`)
	assertMarkers = regexp.MustCompile(`assert|expect\(|\.should\b|\brequire\.|EXPECT_|ASSERT_|XCTAssert|t\.Error|t\.Fatal`)
)

// TestIntegrity guards against reaching green by weakening tests instead of fixing
// code. It diffs the test files against the slice's base tree (the reconcile
// snapshot if present, else HEAD) and flags any test file that was deleted, gained
// a skip/focus marker, or lost assertions. Read-only; the workspace is
// <root>/features/<slug>.
//
//	0  clean, or skipped because this is not a git repo
//	2  no active workspace / bad args
//	3  WEAKENED — a test was deleted, skipped, or de-asserted since the base
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
		fmt.Fprintln(stderr, "test-integrity: not a git repo — gate skipped, eyeball the test diff by hand.")
		return 0
	}

	baseTree := ""
	if b, err := os.ReadFile(filepath.Join(d, ".reconcile-base")); err == nil {
		baseTree = strings.TrimSpace(string(b))
	}
	ref := baseTree
	if ref == "" {
		ref = "HEAD"
	}

	weak := 0
	srcChanged, testChanged := 0, 0
	var report strings.Builder
	for _, f := range gitDiffNames(gitRoot, ref) {
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
		old, ok := gitShow(gitRoot, ref+":"+f)
		if !ok || old == "" {
			continue // new test file — adding tests is never a weakening
		}
		if !isFile(filepath.Join(gitRoot, f)) {
			weak++
			fmt.Fprintf(&report, "  - %s: test file DELETED\n", f)
			continue
		}
		newBytes, err := os.ReadFile(filepath.Join(gitRoot, f))
		if err != nil {
			continue
		}
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

	// Verification-gap advisory: source changed but no test file did. Not a
	// weakening and never a verdict on its own — a pointer to run the
	// verification-gap trace (testing.md), since a green suite can pass identically
	// with the change reverted. Printed on both the clean and weakened paths.
	gapAdvisory := ""
	if srcChanged > 0 && testChanged == 0 {
		gapAdvisory = fmt.Sprintf("test-integrity: advisory — %d source file(s) changed, 0 test file(s) touched; "+
			"run the verification-gap trace (testing.md) to confirm each changed behavior is asserted.", srcChanged)
	}

	if weak > 0 {
		fmt.Fprintf(stderr, "test-integrity: WEAKENED — %d test file(s) lost coverage since the slice base:\n", weak)
		fmt.Fprint(stderr, report.String())
		fmt.Fprintln(stderr, "A test weakened to pass a gate is a Critical finding. Revert it; fix the code or raise a blocking question.")
		if gapAdvisory != "" {
			fmt.Fprintln(stderr, gapAdvisory)
		}
		return 3
	}
	fmt.Fprintln(stdout, "test-integrity: OK — no test deleted, skipped, or de-asserted since the base.")
	if gapAdvisory != "" {
		fmt.Fprintln(stdout, gapAdvisory)
	}
	return 0
}

// isSourceFile recognises a path as project source worth pairing with a test —
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

// isTestFile recognises a path as a test by its filename/directory conventions
// across the common ecosystems (JS/TS, Python, Go, Rust, JUnit).
func isTestFile(f string) bool {
	for _, marker := range []string{
		"_test.", ".test.", ".spec.", "_spec.", "Test.", "Tests.", "Spec.",
		"/__tests__/", "/tests/", "/test/", "/spec/",
		"test", "spec",
	} {
		if strings.Contains(f, marker) {
			return true
		}
	}
	return strings.HasPrefix(f, "test_")
}

// countMatches counts the non-overlapping matches of re in text.
func countMatches(re *regexp.Regexp, text string) int {
	return len(re.FindAllStringIndex(text, -1))
}

// gitShow returns the content of an object (e.g. "HEAD:path"), and false when it
// does not exist at that ref.
func gitShow(gitRoot, spec string) (string, bool) {
	out, err := exec.Command("git", "-C", gitRoot, "show", spec).Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}
