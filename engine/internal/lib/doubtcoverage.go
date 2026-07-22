package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DoubtCoverage answers a single deterministic question: did the build doubt the
// decisions it stood? It reads only records /rite-build writes, so it is
// zero-token and cannot fabricate a verdict-per-decision match: the seal's prose
// gate and the reviewer fan-out judge the rest.
//
// Two signals, checked in order:
//
//  1. decisions.md `doubt: MISSING`: a stood decision recorded but never
//     doubted. The unambiguous skip; checked first, on every path.
//  2. footprint.log `wright` vs `doubt` lines: wright dispatch(es) with zero
//     doubt is the coarse "no doubt ran at all" signal.
//
// base is the project directory that contains the .devrites tree.
//
//	0  doubt covered, or not assessable (no log / no wright / inline build) → pass
//	1  wright dispatch(es) but ZERO doubt → verify against decisions.md
//	2  usage (no slug)
//	3  decisions.md records a stood decision with `doubt: MISSING`
func DoubtCoverage(root string, args []string, stdout, stderr io.Writer) int {
	slug := ""
	if len(args) > 0 {
		slug = args[0]
	}
	if slug == "" {
		fmt.Fprintln(stderr, "usage: devrites-engine doubt-coverage <slug>")
		return 2
	}

	dir := featureDir(root, slug)

	// A recorded stood decision that was never doubted is the unambiguous
	// failure: checked first, and on every path (it survives the inline-build
	// case below, where the footprint heuristic cannot apply).
	if dec, ok := readFileOK(filepath.Join(dir, "decisions.md")); ok && strings.Contains(dec, "doubt: MISSING") {
		fmt.Fprintln(stdout, "doubt-coverage: SKIPPED: a '## Decisions stood' entry in decisions.md has doubt: MISSING.")
		fmt.Fprintln(stdout, "A decision was stood and recorded but never doubted. Re-dispatch doubt or escalate.")
		return 3
	}

	// Inline build (Task tool unavailable): the wright + doubt ran in-context,
	// dispatching nothing, so the footprint heuristic does not apply: the
	// decisions.md check above is the assessment.
	if isFile(filepath.Join(dir, ".reconcile-inline")) {
		fmt.Fprintln(stdout, "doubt-coverage: inline build (.reconcile-inline): footprint heuristic n/a; no MISSING")
		fmt.Fprintln(stdout, "verdict recorded. Verify by hand that each stood decision carries a devrites-doubt verdict.")
		return 0
	}

	f, err := os.Open(filepath.Join(dir, "footprint.log"))
	if err != nil {
		fmt.Fprintln(stdout, "doubt-coverage: no footprint log: nothing to assess (pass)")
		return 0
	}
	defer f.Close()

	var wright, doubt int
	sc := newLineScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		switch fields[1] {
		case "wright":
			wright++
		case "doubt":
			doubt++
		}
	}
	fmt.Fprintf(stdout, "doubt-coverage: %d wright · %d doubt\n", wright, doubt)

	if wright > 0 && doubt == 0 {
		fmt.Fprintf(stdout, "doubt-coverage: no doubt dispatched across %d wright run(s). Either every slice's\n", wright)
		fmt.Fprintln(stdout, "'Decisions stood' was genuinely empty, or doubt was skipped: verify against decisions.md.")
		return 1
	}
	return 0
}
