package main_test

// Golden + diagnostic tests for the ported devrites-lib commands (issue 08).
//
// Golden: each command is run through the Go binary against a fixture, and its
// stdout + exit code are checked against a recorded golden snapshot (stderr is
// diagnostic, not contract). Snapshots are regenerated with UPDATE_GOLDEN=1.
//
// Diagnostic: a second pass asserts the Go stderr carries the same actionable
// finding substrings a consumer greps for, so the port keeps its diagnostic
// quality even though stderr is excluded from the golden check.

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/devrites/devrites/internal/testutil"
)

// libEnv forces byte-order collation so `sort` in the bash script and
// sort.Strings in Go agree (matters for check-acceptance's id ordering).
var libEnv = []string{"LC_ALL=C", "DEVRITES_ROOT="}

// libRootEnv points DEVRITES_ROOT at the workspace's .devrites directory so the
// command reads the fixture under <root>/features/<slug>. LC_ALL=C pins byte-order
// collation so id ordering is deterministic.
func libRootEnv(work string) []string {
	return []string{"LC_ALL=C", "DEVRITES_ROOT=" + filepath.Join(work, ".devrites")}
}

// writeFeatureFile writes fixture content to <work>/.devrites/features/<slug>/<name>.
func writeFeatureFile(t *testing.T, work, slug, name, content string) {
	t.Helper()
	writeFile(t, work, filepath.Join(".devrites/features", slug, name), content)
}

// makeFeatureDir creates an empty feature directory in the workspace.
func makeFeatureDir(t *testing.T, work, slug string) {
	t.Helper()
	mkdirAllT(t, work, filepath.Join(".devrites/features", slug))
}

// writeSpec writes content to <workdir>/<name>/spec.md and returns the subdir name.
func writeSpec(t *testing.T, workdir, name, content string) string {
	t.Helper()
	dir := filepath.Join(workdir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return name
}

func writeFile(t *testing.T, workdir, rel, content string) {
	t.Helper()
	testutil.WriteFile(t, filepath.Join(workdir, rel), content)
}

// runGo runs the built binary in workdir with libEnv, returning stdout, stderr, code.
func runGo(t *testing.T, workdir string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), libEnv...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("running %s %v: %v", binPath, args, err)
		}
		code = ee.ExitCode()
	}
	return out.String(), errBuf.String(), code
}

// ---- spec-validate --------------------------------------------------------

// Spec fixtures cover the well-formed spec plus each malformed shape so the
// golden check exercises every spec-validate verdict.
const (
	specValid = `# Spec
## Acceptance criteria
### Requirement: Tokens expire after inactivity
The system SHALL reject a session token older than 15 minutes.
#### Scenario: expired token
- [ ] [AC1] **WHEN** a token aged > 15m is presented **THEN** respond 401
#### Scenario: fresh token
- [ ] [AC2] **WHEN** a token aged < 15m is presented **THEN** allow
### Requirement: Logout revokes the token
Logout MUST invalidate the token server-side.
#### Scenario: replay after logout
- [ ] [AC3] **WHEN** a logged-out token is replayed **THEN** respond 401
`
	specFlat = `# Spec
## Acceptance criteria
- [ ] [AC1] export returns a CSV with a header row
- [ ] [AC2] an empty dataset returns 204
`
	specNoShall = `### Requirement: Export behaves nicely
The export produces a file and emails the user.
#### Scenario: happy path
- **WHEN** a user requests an export **THEN** a file is produced
`
	specNoScenario = `### Requirement: It works
The system SHALL do the thing.
`
	specNoThen = `### Requirement: It works
The system SHALL do the thing.
#### Scenario: half a scenario
- **WHEN** something happens
`
	specDup = `### Requirement: Same name
The system SHALL alpha.
#### Scenario: a
- **WHEN** a **THEN** b
### Requirement: Same name
The system SHALL beta.
#### Scenario: c
- **WHEN** c **THEN** d
`
)

func TestParitySpecValidate(t *testing.T) {
	work := t.TempDir()

	writeSpec(t, work, "valid", specValid)
	writeSpec(t, work, "flat", specFlat)
	writeSpec(t, work, "noshall", specNoShall)
	writeSpec(t, work, "noscenario", specNoScenario)
	writeSpec(t, work, "nothen", specNoThen)
	writeSpec(t, work, "dup", specDup)
	if err := os.MkdirAll(filepath.Join(work, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	// arg is the positional argument passed to both implementations, relative to work.
	for _, arg := range []string{
		"valid",         // well-formed structured spec → exit 0
		"flat",          // flat-bullet form → exit 0 (no-op)
		"noshall",       // missing SHALL/MUST → exit 1
		"noscenario",    // requirement with no scenario → exit 1
		"nothen",        // scenario missing THEN → exit 1
		"dup",           // duplicate headers → exit 1
		"valid/spec.md", // direct file path → exit 0
		"empty",         // dir without spec.md → exit 5
		"nonexistent",   // no such workspace or file → exit 2
	} {
		c := parityCase{
			workdir: work,
			env:     libEnv,
			goArgs:  []string{"spec-validate", arg},
		}
		t.Run("arg="+arg, func(t *testing.T) { c.assertEqual(t) })
	}

	// No-argument usage path (exit 2) — asserted separately since arg is absent.
	(parityCase{
		workdir: work,
		env:     libEnv,
		goArgs:  []string{"spec-validate"},
	}).assertEqual(t)
}

func TestSpecSkeleton(t *testing.T) {
	work := t.TempDir()

	writeSpec(t, work, "present", `# Contract

## Problem
done
## Goal
done
## Non-goals
done
## Users / actors
done
## Requirements
done
## Acceptance criteria
done
## Edge Coverage
done
## Prohibitions (must-NOT)
done
## Edge cases
done
## Measurable success
done
## Scope boundaries
done
`)
	writeSpec(t, work, "missing", `# Contract

## Problem
done
## Goal
done
## Requirements
done
`)

	for _, tc := range []struct {
		name string
		arg  string
	}{
		{"present-all", "present"},
		{"missing-some", "missing/spec.md"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			(parityCase{
				workdir: work,
				env:     libEnv,
				goArgs:  []string{"spec-skeleton", tc.arg},
			}).assertEqual(t)
		})
	}
}

// TestSpecValidateDiagnostics asserts the Go stderr carries the same finding
// substrings the legacy spec-validate-test.sh greps for.
func TestSpecValidateDiagnostics(t *testing.T) {
	work := t.TempDir()
	cases := []struct {
		name, content, wantExit1 string
	}{
		{"noshall", specNoShall, "no SHALL/MUST"},
		{"noscenario", specNoScenario, `no "#### Scenario:"`},
		{"nothen", specNoThen, "no THEN line"},
		{"dup", specDup, "duplicate Requirement header"},
	}
	for _, tc := range cases {
		writeSpec(t, work, tc.name, tc.content)
		_, stderr, code := runGo(t, work, "spec-validate", tc.name)
		if code != 1 {
			t.Errorf("%s: exit=%d, want 1", tc.name, code)
		}
		if !bytes.Contains([]byte(stderr), []byte(tc.wantExit1)) {
			t.Errorf("%s: stderr %q missing %q", tc.name, stderr, tc.wantExit1)
		}
	}
}

// ---- check-acceptance -----------------------------------------------------

func TestParityCheckAcceptance(t *testing.T) {
	work := t.TempDir()

	// id mode, all proven → exit 0
	writeFile(t, work, "id-ok/spec.md", "## Acceptance criteria\n- [ ] [AC1] a\n- [ ] [AC2] b\n- [ ] [AC3] c\n")
	writeFile(t, work, "id-ok/seal.md", "## Acceptance Criteria\n- [x] [AC1] a — evidence: t1\n- [x] [AC2] b — evidence: t2\n- [x] [AC3] c — evidence: t3\n")

	// id mode, one unproven → exit 1
	writeFile(t, work, "id-gap/spec.md", "## Acceptance criteria\n- [ ] [AC1] a\n- [ ] [AC2] b\n- [ ] [AC3] c\n")
	writeFile(t, work, "id-gap/seal.md", "## Acceptance Criteria\n- [x] [AC1] a — evidence: t1\n- [ ] [AC2] b\n- [ ] [AC3] c\n")

	// fallback (no ids), all checked → exit 0
	writeFile(t, work, "flat-ok/spec.md", "## Acceptance criteria\n- export returns CSV\n- empty dataset returns 204\n")
	writeFile(t, work, "flat-ok/seal.md", "## Acceptance Criteria\n- [x] export returns CSV\n- [x] empty dataset returns 204\n")

	// fallback, one unchecked → exit 1
	writeFile(t, work, "flat-unchecked/spec.md", "## Acceptance criteria\n- export returns CSV\n- empty dataset returns 204\n")
	writeFile(t, work, "flat-unchecked/seal.md", "## Acceptance Criteria\n- [x] export returns CSV\n- [ ] empty dataset returns 204\n")

	// fallback, seal drops a criterion → exit 1
	writeFile(t, work, "flat-dropped/spec.md", "## Acceptance criteria\n- a\n- b\n- c\n")
	writeFile(t, work, "flat-dropped/seal.md", "## Acceptance Criteria\n- [x] a\n- [x] b\n")

	// spec has no acceptance section → exit 1
	writeFile(t, work, "no-section/spec.md", "# Spec\nsome prose, no criteria heading\n")
	writeFile(t, work, "no-section/seal.md", "## Acceptance Criteria\n- [x] a\n")

	// spec has an acceptance heading but only blank lines under it → still
	// "nothing to grade" (bash `[ -z ]` after trailing-newline strip), exit 1.
	writeFile(t, work, "blank-section/spec.md", "## Acceptance criteria\n\n\n## Notes\nprose\n")
	writeFile(t, work, "blank-section/seal.md", "## Acceptance Criteria\n- [x] a\n")

	// missing seal.md → exit 5
	writeFile(t, work, "no-seal/spec.md", "## Acceptance criteria\n- [ ] [AC1] a\n")

	// empty dir (missing spec.md) → exit 5
	if err := os.MkdirAll(filepath.Join(work, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	for _, arg := range []string{
		"id-ok", "id-gap", "flat-ok", "flat-unchecked", "flat-dropped",
		"no-section", "blank-section", "no-seal", "empty", "nonexistent",
	} {
		c := parityCase{
			workdir: work,
			env:     libEnv,
			goArgs:  []string{"check-acceptance", arg},
		}
		t.Run("arg="+arg, func(t *testing.T) { c.assertEqual(t) })
	}

	// No-argument usage path (exit 2).
	(parityCase{
		workdir: work,
		env:     libEnv,
		goArgs:  []string{"check-acceptance"},
	}).assertEqual(t)
}

// ---- footprint ------------------------------------------------------------
//
// Golden tests: the expected stdout + exit values are asserted inline under
// LC_ALL=C, so the feature stands alone. The invariants a future edit must
// preserve: skip records never count, active-minutes = int((max-min)/60), the
// wright · reviewer · audit · doubt kind order with correct singular/plural
// forms, and the roster 0/1/3 exit taxonomy.

// goldenCase asserts the built binary's stdout + exit code for one invocation.
type goldenCase struct {
	name     string
	args     []string
	wantOut  string
	wantCode int
}

func (gc goldenCase) assert(t *testing.T, work string) {
	t.Helper()
	out, _, code := runGo(t, work, gc.args...)
	if out != gc.wantOut || code != gc.wantCode {
		t.Errorf("%s: got exit=%d out=%q\n want exit=%d out=%q",
			gc.name, code, out, gc.wantCode, gc.wantOut)
	}
}

func TestFootprintGolden(t *testing.T) {
	work := t.TempDir()

	// render: the deterministic fixed-epoch log from tests/footprint-test.sh.
	writeFile(t, work, ".devrites/features/rendered/footprint.log",
		"1000 wright 01-list\n1100 wright 02-detail\n"+
			"1200 reviewer devrites-code-reviewer\n1200 reviewer devrites-security-auditor\n"+
			"1300 reviewer devrites-test-analyst\n")
	// single reviewer → the singular "1 subagent (1 reviewer)" + 0-slice path.
	writeFile(t, work, ".devrites/features/one/footprint.log", "1000 reviewer spec-reviewer\n")
	// audit + doubt → those kinds' pluralization and order.
	writeFile(t, work, ".devrites/features/ad/footprint.log",
		"1000 wright a\n1000 audit x\n1000 audit y\n1000 doubt z\n")
	mkdirAllT(t, work, ".devrites/features/nolog")
	writeFile(t, work, ".devrites/features/rc-complete/footprint.log",
		"1000 reviewer spec-reviewer\n1000 reviewer code-reviewer\n1000 reviewer test-analyst\n"+
			"1000 reviewer frontend-reviewer\n1000 skip security-auditor\n"+
			"1000 skip performance-reviewer\n1000 skip devex-reviewer\n")
	writeFile(t, work, ".devrites/features/rc-unacc/footprint.log",
		"1000 reviewer spec-reviewer\n1000 reviewer code-reviewer\n")
	writeFile(t, work, ".devrites/features/rc-skip/footprint.log",
		"1000 reviewer spec-reviewer\n1000 reviewer code-reviewer\n1000 skip test-analyst\n"+
			"1000 skip frontend-reviewer\n1000 skip security-auditor\n"+
			"1000 skip performance-reviewer\n1000 skip devex-reviewer\n")
	mkdirAllT(t, work, ".devrites/features/live")

	rosterComplete := "  spec-reviewer: dispatched\n  code-reviewer: dispatched\n" +
		"  test-analyst: dispatched\n  frontend-reviewer: dispatched\n" +
		"  security-auditor: skipped\n  performance-reviewer: skipped\n" +
		"  devex-reviewer: skipped\nroster: complete (every reviewer accounted for)\n"
	rosterUnacc := "  spec-reviewer: dispatched\n  code-reviewer: dispatched\n" +
		"  test-analyst: UNACCOUNTED\n  frontend-reviewer: UNACCOUNTED\n" +
		"  security-auditor: UNACCOUNTED\n  performance-reviewer: UNACCOUNTED\n" +
		"  devex-reviewer: UNACCOUNTED\n" +
		"roster: UNACCOUNTED — test-analyst frontend-reviewer security-auditor performance-reviewer devex-reviewer\n"
	rosterSkip := "  spec-reviewer: dispatched\n  code-reviewer: dispatched\n" +
		"  test-analyst: skipped (always-on — verify carry-forward)\n" +
		"  frontend-reviewer: skipped\n  security-auditor: skipped\n" +
		"  performance-reviewer: skipped\n  devex-reviewer: skipped\n" +
		"roster: always-on skipped — test-analyst\n"

	for _, gc := range []goldenCase{
		{"render", []string{"footprint", "render", "rendered"},
			"Footprint: 5 subagents (2 wright · 3 reviewers) · 2 slices · ~5m active\n", 0},
		{"render-singular", []string{"footprint", "render", "one"},
			"Footprint: 1 subagent (1 reviewer) · 0 slices · ~0m active\n", 0},
		{"render-audit-doubt", []string{"footprint", "render", "ad"},
			"Footprint: 4 subagents (1 wright · 2 audits · 1 doubt) · 1 slice · ~0m active\n", 0},
		{"render-nolog", []string{"footprint", "render", "nolog"},
			"Footprint: n/a (no dispatch records)\n", 0},
		{"render-missing-ws", []string{"footprint", "render", "ghost"},
			"Footprint: n/a (no dispatch records)\n", 0},
		{"roster-complete", []string{"footprint", "roster", "rc-complete"}, rosterComplete, 0},
		{"roster-unaccounted", []string{"footprint", "roster", "rc-unacc"}, rosterUnacc, 3},
		{"roster-skip-alwayson", []string{"footprint", "roster", "rc-skip"}, rosterSkip, 1},
		{"roster-nolog", []string{"footprint", "roster", "nolog"},
			"roster: n/a (no dispatch records — fan-out did not run)\n", 0},
		{"log-missing-ws", []string{"footprint", "log", "ghost", "wright", "x"}, "", 0},
		{"log-usage-no-kind", []string{"footprint", "log", "live"}, "", 2},
		{"unknown-subcommand", []string{"footprint", "bogus", "live"}, "", 2},
		{"missing-slug", []string{"footprint", "render"}, "", 2},
		{"no-args", []string{"footprint"}, "", 2},
	} {
		t.Run(gc.name, func(t *testing.T) { gc.assert(t, work) })
	}

	// The footprint never emits a token/dollar/cost figure — DevRites cannot
	// truthfully source one (the same invariant footprint-test.sh guards).
	out, _, _ := runGo(t, work, "footprint", "render", "rendered")
	for _, banned := range []string{"token", "$", "cost", "usd", "USD"} {
		if bytes.Contains([]byte(out), []byte(banned)) {
			t.Errorf("render leaked a %q figure: %q", banned, out)
		}
	}
}

// TestFootprintLogAppends asserts the side-effect the parity oracle can't (it
// compares only stdout+exit): `log` appends exactly one record and a missing
// workspace is a silent no-op.
func TestFootprintLogAppends(t *testing.T) {
	work := t.TempDir()
	mkdirAllT(t, work, ".devrites/features/feat")
	logp := filepath.Join(work, ".devrites", "features", "feat", "footprint.log")

	if _, _, code := runGo(t, work, "footprint", "log", "feat", "audit", "security"); code != 0 {
		t.Fatalf("log exit=%d, want 0", code)
	}
	if n := fileLines(t, logp); n != 1 {
		t.Errorf("after one log: %d lines, want 1", n)
	}
	runGo(t, work, "footprint", "log", "feat", "wright", "01")
	if n := fileLines(t, logp); n != 2 {
		t.Errorf("after two logs: %d lines, want 2", n)
	}

	// An empty label still writes its trailing space — byte-parity with
	// footprint.sh's `printf '%s %s %s\n'`, which always emits three fields.
	runGo(t, work, "footprint", "log", "feat", "wright")
	if last := lastLine(t, logp); !bytes.HasSuffix([]byte(last), []byte(" wright ")) {
		t.Errorf("empty-label record = %q, want a trailing space (`<epoch> wright `)", last)
	}

	if _, _, code := runGo(t, work, "footprint", "log", "ghost", "wright", "x"); code != 0 {
		t.Errorf("log on a missing workspace: exit=%d, want 0 (never fail the caller)", code)
	}
	if _, err := os.Stat(filepath.Join(work, ".devrites", "features", "ghost", "footprint.log")); err == nil {
		t.Error("log on a missing workspace created a log; it must be a no-op")
	}
}

// ---- evidence-fresh -------------------------------------------------------
//
// Golden tests: expected stdout + exit are asserted inline. File mtimes are set
// explicitly so the freshness comparison is deterministic and independent of
// when the fixtures were written.

func TestEvidenceFreshGolden(t *testing.T) {
	work := t.TempDir()

	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mid := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

	const okMsg = "evidence-fresh: OK — evidence post-dates every touched file.\n"

	// Primary workspace: evidence + one touched file.
	ws := ".devrites/features/ftest"
	writeFile(t, work, ws+"/touched-files.md", "# touched\n- `src/app.js`\n")
	writeFile(t, work, ws+"/evidence.md", "# evidence\nproof\n")
	writeFile(t, work, "src/app.js", "code\n")

	// STALE: the touched file is newer than the proof → exit 3.
	chtimeT(t, work, ws+"/evidence.md", old)
	chtimeT(t, work, "src/app.js", mid)
	(goldenCase{"stale", []string{"evidence-fresh", "ftest"}, "", 3}).assert(t, work)

	// FRESH: the proof now post-dates the code → exit 0.
	chtimeT(t, work, ws+"/evidence.md", newer)
	(goldenCase{"fresh", []string{"evidence-fresh", "ftest"}, okMsg, 0}).assert(t, work)

	// proof.md is the supported alias for evidence.md. A proof-only workspace
	// must pass/fail freshness by the same mtime rule.
	writeFile(t, work, ".devrites/features/proofonly/touched-files.md", "# touched\n- `src/proofonly.js`\n")
	writeFile(t, work, ".devrites/features/proofonly/proof.md", "# proof\n")
	writeFile(t, work, "src/proofonly.js", "x\n")
	chtimeT(t, work, "src/proofonly.js", old)
	chtimeT(t, work, ".devrites/features/proofonly/proof.md", newer)
	(goldenCase{"proof-only-fresh", []string{"evidence-fresh", "proofonly"}, okMsg, 0}).assert(t, work)

	writeFile(t, work, ".devrites/features/proofstale/touched-files.md", "# touched\n- `src/proofstale.js`\n")
	writeFile(t, work, ".devrites/features/proofstale/proof.md", "# proof\n")
	writeFile(t, work, "src/proofstale.js", "x\n")
	chtimeT(t, work, ".devrites/features/proofstale/proof.md", old)
	chtimeT(t, work, "src/proofstale.js", newer)
	(goldenCase{"proof-only-stale", []string{"evidence-fresh", "proofstale"}, "", 3}).assert(t, work)

	// Default slug resolved from .devrites/ACTIVE, fresh → exit 0.
	writeFile(t, work, ".devrites/features/defslug/touched-files.md", "# touched\n- `src/other.js`\n")
	writeFile(t, work, ".devrites/features/defslug/evidence.md", "# evidence\n")
	writeFile(t, work, "src/other.js", "x\n")
	writeFile(t, work, ".devrites/ACTIVE", "defslug\n")
	chtimeT(t, work, "src/other.js", old)
	chtimeT(t, work, ".devrites/features/defslug/evidence.md", newer)
	(goldenCase{"default-slug", []string{"evidence-fresh"}, okMsg, 0}).assert(t, work)

	// An absolute touched path newer than the proof → STALE (exercises the
	// absolute-path branch, not just repo-relative resolution).
	absPath := filepath.Join(work, "absnew.js")
	writeFile(t, work, "absnew.js", "x\n")
	writeFile(t, work, ".devrites/features/abs/touched-files.md", "# touched\n- `"+absPath+"`\n")
	writeFile(t, work, ".devrites/features/abs/evidence.md", "# evidence\n")
	chtimeT(t, work, ".devrites/features/abs/evidence.md", old)
	chtimeT(t, work, "absnew.js", newer)
	(goldenCase{"absolute-path-stale", []string{"evidence-fresh", "abs"}, "", 3}).assert(t, work)

	// No workspace directory → exit 5.
	(goldenCase{"no-workspace", []string{"evidence-fresh", "ghost"}, "", 5}).assert(t, work)

	// Workspace present but no evidence → exit 5.
	mkdirAllT(t, work, ".devrites/features/noev")
	(goldenCase{"no-evidence", []string{"evidence-fresh", "noev"}, "", 5}).assert(t, work)

	// Proof present, no touched-files.md → nothing to compare, exit 0.
	writeFile(t, work, ".devrites/features/notf/evidence.md", "# evidence\n")
	(goldenCase{"no-touched-files", []string{"evidence-fresh", "notf"},
		"evidence-fresh: no touched-files.md — nothing to compare (treating as fresh).\n", 0}).assert(t, work)
}

// mkdirAllT creates an (empty) directory under workdir.
func mkdirAllT(t *testing.T, workdir, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(workdir, rel), 0o755); err != nil {
		t.Fatal(err)
	}
}

// chtimeT sets a file's mtime (and atime) so freshness comparisons are
// deterministic and identical for both the bash and Go readers.
func chtimeT(t *testing.T, workdir, rel string, mtime time.Time) {
	t.Helper()
	if err := os.Chtimes(filepath.Join(workdir, rel), mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

// fileLines counts newline-terminated lines in a file.
func fileLines(t *testing.T, path string) int {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Count(b, []byte("\n"))
}

// lastLine returns the final non-empty line of a file (its trailing newline
// stripped), so a caller can assert the exact bytes of the most recent record.
func lastLine(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimRight(b, "\n"), []byte("\n"))
	return string(lines[len(lines)-1])
}

// ---- coverage -------------------------------------------------------------
//
// Golden: the matrix, the AC id sort, the ";"-joined slices (including the awk
// substring quirk), the UNCOVERED / pending fallbacks, and the exit taxonomy
// are checked against golden snapshots.

func TestParityCoverage(t *testing.T) {
	work := t.TempDir()

	// Mixed coverage + proof, and C-locale AC ordering (AC1, AC10, AC2). The
	// "AC1" row also picks up the AC10 slice — the awk match is a substring test
	// and the port reproduces it.
	writeFeatureFile(t, work, "full", "spec.md",
		"## Acceptance criteria\n- [ ] [AC1] alpha\n- [ ] [AC2] beta\n- [ ] [AC10] dec\n")
	writeFeatureFile(t, work, "full", "tasks.md",
		"## Slice 1: list\n  Satisfies: AC1\n## Slice 2: detail\n  Satisfies: AC2, AC10\n")
	writeFeatureFile(t, work, "full", "seal.md",
		"## Acceptance Criteria\n- [x] [AC1] alpha — evidence: t1\n- [ ] [AC2] beta\n- [x] [AC10] dec — evidence: t3\n")

	// spec only → every AC UNCOVERED + pending.
	writeFeatureFile(t, work, "speconly", "spec.md",
		"## Acceptance criteria\n- [ ] [AC1] a\n- [ ] [AC2] b\n")

	// flat spec, no [ACn] ids → header + footer, zero rows.
	writeFeatureFile(t, work, "flat", "spec.md",
		"## Acceptance criteria\n- export returns CSV\n- empty dataset returns 204\n")

	// workspace dir present but no spec.md → exit 2.
	makeFeatureDir(t, work, "nospec")

	for _, arg := range []string{
		"full",        // matrix with mixed proof + the substring quirk
		"speconly",    // all UNCOVERED / pending
		"flat",        // no ids → empty matrix body
		"nospec",      // dir, no spec.md → exit 2
		"nonexistent", // no workspace dir → exit 2
	} {
		c := parityCase{
			workdir: work,
			env:     libRootEnv(work),
			goArgs:  []string{"coverage", arg},
		}
		t.Run("arg="+arg, func(t *testing.T) { c.assertEqual(t) })
	}

	// No-argument path with no active feature → "no active workspace", exit 2.
	(parityCase{
		workdir: work,
		env:     libRootEnv(work),
		goArgs:  []string{"coverage"},
	}).assertEqual(t)
}

// TestCoverageGolden asserts the exact matrix bytes with an inline expectation
// (fixtures chosen without an AC1/AC10 collision so the golden reads naturally;
// the substring quirk is exercised by TestParityCoverage's "full" fixture).
func TestCoverageGolden(t *testing.T) {
	work := t.TempDir()
	const base = ".devrites/features/"

	writeFile(t, work, base+"g/spec.md",
		"## Acceptance criteria\n- [ ] [AC1] a\n- [ ] [AC2] b\n- [ ] [AC3] c\n")
	writeFile(t, work, base+"g/tasks.md",
		"## Slice 1: list\n  Satisfies: AC1\n## Slice 2: detail\n  Satisfies: AC2, AC3\n")
	writeFile(t, work, base+"g/seal.md",
		"## Acceptance Criteria\n- [x] [AC1] a — evidence: t1\n- [x] [AC2] b — evidence: t2\n- [ ] [AC3] c\n")

	const wantG = "# Coverage matrix: g\n\n" +
		"| AC | Slice(s) | Proven in seal? |\n" +
		"|----|----------|-----------------|\n" +
		"| AC1 | Slice 1: list | yes |\n" +
		"| AC2 | Slice 2: detail | yes |\n" +
		"| AC3 | Slice 2: detail | pending |\n\n" +
		"_Generated by devrites-engine coverage from spec.md / tasks.md / seal.md. UNCOVERED rows block the analyze gate in /rite-vet (Claude) / $rite-vet (Codex)._\n"
	(goldenCase{"full-matrix", []string{"coverage", "g"}, wantG, 0}).assert(t, work)

	// spec only → UNCOVERED / pending fallbacks.
	writeFile(t, work, base+"u/spec.md", "## Acceptance criteria\n- [ ] [AC1] a\n")
	const wantU = "# Coverage matrix: u\n\n" +
		"| AC | Slice(s) | Proven in seal? |\n" +
		"|----|----------|-----------------|\n" +
		"| AC1 | — (UNCOVERED) | pending |\n\n" +
		"_Generated by devrites-engine coverage from spec.md / tasks.md / seal.md. UNCOVERED rows block the analyze gate in /rite-vet (Claude) / $rite-vet (Codex)._\n"
	(goldenCase{"uncovered", []string{"coverage", "u"}, wantU, 0}).assert(t, work)
}

// ---- doubt-coverage -------------------------------------------------------
//
// Golden: the decisions.md `doubt: MISSING` short-circuit (checked first), the
// inline-build branch, the no-log pass, the wright/doubt count line, and the
// 0/1/2/3 exit taxonomy are checked against golden snapshots.

func TestParityDoubtCoverage(t *testing.T) {
	work := t.TempDir()

	// decisions.md MISSING → exit 3, and it wins even over the footprint log.
	writeFeatureFile(t, work, "missing", "decisions.md",
		"## Decisions stood\n- use flock — doubt: MISSING\n")
	writeFeatureFile(t, work, "missing", "footprint.log", "1000 wright 01\n")

	// MISSING is checked BEFORE the inline marker → still exit 3.
	writeFeatureFile(t, work, "missing-inline", "decisions.md", "- x — doubt: MISSING\n")
	writeFeatureFile(t, work, "missing-inline", ".reconcile-inline", "")

	// inline build, no MISSING → exit 0.
	writeFeatureFile(t, work, "inline", ".reconcile-inline", "")
	writeFeatureFile(t, work, "inline", "decisions.md", "## Decisions stood\n- x — doubt: yes\n")

	// dir exists, no log / dec / marker → "no footprint log" pass.
	makeFeatureDir(t, work, "nolog")

	// wright dispatch(es), zero doubt → exit 1.
	writeFeatureFile(t, work, "nodoubt", "footprint.log",
		"1000 wright 01\n1000 reviewer code-reviewer\n1000 wright 02\n")

	// wright + doubt → exit 0.
	writeFeatureFile(t, work, "doubted", "footprint.log", "1000 wright 01\n1000 doubt d1\n")

	// zero wright (only reviewers) → exit 0, count line "0 wright · N doubt".
	writeFeatureFile(t, work, "nowright", "footprint.log",
		"1000 reviewer code-reviewer\n1000 doubt d1\n")

	for _, arg := range []string{
		"missing", "missing-inline", "inline", "nolog",
		"nodoubt", "doubted", "nowright",
		"ghost", // no workspace at all → "no footprint log" pass
	} {
		c := parityCase{
			workdir: work,
			env:     libRootEnv(work),
			goArgs:  []string{"doubt-coverage", arg},
		}
		t.Run("arg="+arg, func(t *testing.T) { c.assertEqual(t) })
	}

	// No-slug usage path → exit 2.
	(parityCase{
		workdir: work,
		env:     libRootEnv(work),
		goArgs:  []string{"doubt-coverage"},
	}).assertEqual(t)
}

// TestDoubtCoverageGolden asserts each verdict's exact stdout inline — the
// SKIPPED / inline / no-log messages and the "no doubt ran" prompt.
func TestDoubtCoverageGolden(t *testing.T) {
	work := t.TempDir()
	const base = ".devrites/features/"

	writeFile(t, work, base+"miss/decisions.md", "- x — doubt: MISSING\n")
	writeFile(t, work, base+"in/.reconcile-inline", "")
	mkdirAllT(t, work, base+"empty")
	writeFile(t, work, base+"nd/footprint.log", "1000 wright 01\n1000 wright 02\n")
	writeFile(t, work, base+"ok/footprint.log", "1000 wright 01\n1000 doubt d1\n")

	for _, gc := range []goldenCase{
		{"skipped", []string{"doubt-coverage", "miss"},
			"doubt-coverage: SKIPPED — a '## Decisions stood' entry in decisions.md has doubt: MISSING.\n" +
				"A decision was stood and recorded but never doubted. Re-dispatch doubt or escalate.\n", 3},
		{"inline", []string{"doubt-coverage", "in"},
			"doubt-coverage: inline build (.reconcile-inline) — footprint heuristic n/a; no MISSING\n" +
				"verdict recorded. Verify by hand that each stood decision carries a devrites-doubt verdict.\n", 0},
		{"no-log", []string{"doubt-coverage", "empty"},
			"doubt-coverage: no footprint log — nothing to assess (pass)\n", 0},
		{"no-doubt-ran", []string{"doubt-coverage", "nd"},
			"doubt-coverage: 2 wright · 0 doubt\n" +
				"doubt-coverage: no doubt dispatched across 2 wright run(s). Either every slice's\n" +
				"'Decisions stood' was genuinely empty, or doubt was skipped — verify against decisions.md.\n", 1},
		{"doubted", []string{"doubt-coverage", "ok"},
			"doubt-coverage: 1 wright · 1 doubt\n", 0},
	} {
		t.Run(gc.name, func(t *testing.T) { gc.assert(t, work) })
	}
}

// ---- preamble -------------------------------------------------------------
//
// Golden: the workspace-state orientation — the header, verbatim state.md (with
// and without a trailing newline), the fixed artifact-present order, the AFK/HITL
// run mode with verbatim config lines, and the questions-by-gate tally — are
// checked against golden snapshots.

func TestParityPreamble(t *testing.T) {
	// A HITL workspace (no .devrites/AFK) exercising most branches.
	hitl := t.TempDir()
	writeFeatureFile(t, hitl, "full", "state.md", "- Phase: build\n- Status: running\n- Next step: implement slice 3\n")
	writeFeatureFile(t, hitl, "full", "spec.md", "# spec\n")
	writeFeatureFile(t, hitl, "full", "plan.md", "# plan\n")
	writeFeatureFile(t, hitl, "full", "tasks.md", "# tasks\n")
	writeFeatureFile(t, hitl, "full", "questions.md",
		"## q-1\nstatus: open\ngate: blocking\n\n"+
			"## q-2\nstatus: resolved\ngate: validating\n\n"+
			"## q-3\nstatus: open\ngate: advisory\n\n"+
			"## q-4\nstatus: open\ngate: blocking\n## Notes\nprose\n")
	// questions.md ending on an OPEN q- block (no closing heading) — exercises the
	// awk END-finalize path.
	writeFeatureFile(t, hitl, "openend", "state.md", "- Phase: spec\n")
	writeFeatureFile(t, hitl, "openend", "questions.md",
		"## q-1\nstatus: resolved\ngate: blocking\n\n## q-2\nstatus: open\ngate: escalating\n")
	// state.md with NO trailing newline — `cat` must not add one before the blank line.
	writeFeatureFile(t, hitl, "nonl", "state.md", "- Phase: spec")
	// A workspace directory with no state.md / questions.md.
	makeFeatureDir(t, hitl, "bare")
	writeFile(t, hitl, ".devrites/ACTIVE", "full\n")

	for _, arg := range []string{"full", "openend", "nonl", "bare", "ghost"} {
		c := parityCase{
			workdir: hitl,
			env:     libRootEnv(hitl),
			goArgs:  []string{"preamble", arg},
		}
		t.Run("hitl/"+arg, func(t *testing.T) { c.assertEqual(t) })
	}
	// No-arg path resolves the slug from ACTIVE.
	t.Run("hitl/active", func(t *testing.T) {
		(parityCase{workdir: hitl, env: libRootEnv(hitl), goArgs: []string{"preamble"}}).assertEqual(t)
	})

	// An AFK workspace: the sentinel plus verbatim (non-comment) config lines.
	afk := t.TempDir()
	writeFeatureFile(t, afk, "a", "state.md", "- Phase: prove\n")
	writeFile(t, afk, ".devrites/AFK", "# afk config\nmax-slices: 3\n\nescalate: true\n")
	t.Run("afk/a", func(t *testing.T) {
		(parityCase{workdir: afk, env: libRootEnv(afk), goArgs: []string{"preamble", "a"}}).assertEqual(t)
	})
}

// ---- progress -------------------------------------------------------------
//
// Golden: the header rule width, the slice meter (round-half math, ✅ ALL BUILT
// vs the last-built tail), and the flow ribbon (conditional temper/vet, the
// explicit plan and terminal done cursor rules) are checked against golden snapshots.

func TestParityProgress(t *testing.T) {
	work := t.TempDir()

	sliceBlock := "\n## Slice progress\n" +
		"- [x] Slice 1: list — built\n" +
		"- [x] Slice 2: detail — built\n" +
		"- [ ] Slice 3: polish — pending\n" +
		"## Next\n- something\n"

	// build phase, 2/3 slices, temper (strategy.md) + vet (eng-review.md) present.
	writeFeatureFile(t, work, "mid", "state.md", "- Phase: build"+sliceBlock)
	writeFeatureFile(t, work, "mid", "strategy.md", "# strategy\n")
	writeFeatureFile(t, work, "mid", "eng-review.md", "# eng-review\n")

	// all built → ✅ ALL BUILT; no temper/vet artifacts → shorter ribbon.
	writeFeatureFile(t, work, "allbuilt", "state.md",
		"- Phase: build\n## Slice progress\n- [x] Slice 1: a — built\n- [x] Slice 2: b — built\n")

	// no slices at all → meter skipped entirely.
	writeFeatureFile(t, work, "noslice", "state.md", "- Phase: spec\n")

	// phase variants exercising the cursor rules.
	writeFeatureFile(t, work, "done", "state.md", "- Phase: done\n")
	writeFeatureFile(t, work, "plan", "state.md", "- Phase: plan"+sliceBlock)
	writeFeatureFile(t, work, "seal", "state.md", "- Phase: seal\n")
	// missing Phase → defaults to "spec".
	writeFeatureFile(t, work, "nophase", "state.md", "- Status: running\n")

	for _, arg := range []string{"mid", "allbuilt", "noslice", "done", "plan", "seal", "nophase", "ghost"} {
		c := parityCase{
			workdir: work,
			env:     libRootEnv(work),
			goArgs:  []string{"progress", arg},
		}
		t.Run("arg="+arg, func(t *testing.T) { c.assertEqual(t) })
	}
}

// ---- stuck ----------------------------------------------------------------
//
// Golden: `check` reads only the sha1 column, so a fixture log with controlled
// hash tokens drives every verdict (too-few / identical / ping-pong / dominant /
// varied / custom window) and the 0/3 exit split, checked against golden
// snapshots. `log` is exercised separately (its stdout is empty; its side effect
// carries a nondeterministic timestamp, so only the sha1 column is snapshotted).

func TestParityStuckCheck(t *testing.T) {
	work := t.TempDir()

	// Each record is "<epoch> <tool> <hash>"; check only reads the hash ($3).
	rec := func(h string) string { return "1700000000 Bash " + h + "\n" }
	writeFeatureFile(t, work, "few", "action.log", rec("a")+rec("a"))                         // NR < win(4)
	writeFeatureFile(t, work, "identical", "action.log", rec("a")+rec("a")+rec("a")+rec("a")) // last 4 identical
	writeFeatureFile(t, work, "pingpong", "action.log", rec("a")+rec("b")+rec("a")+rec("b")+rec("a")+rec("b"))
	writeFeatureFile(t, work, "dominant", "action.log", rec("a")+rec("a")+rec("a")+rec("a")+rec("a")+rec("a")+rec("b")+rec("c"))
	writeFeatureFile(t, work, "varied", "action.log", rec("a")+rec("b")+rec("c")+rec("d")+rec("e"))
	writeFeatureFile(t, work, "varied4", "action.log", rec("a")+rec("b")+rec("c")+rec("d")) // 4 distinct
	makeFeatureDir(t, work, "nolog")

	type cse struct {
		name   string
		goArgs []string
	}
	for _, tc := range []cse{
		{"few", []string{"check", "few"}},
		{"identical", []string{"check", "identical"}},
		{"pingpong", []string{"check", "pingpong"}},
		{"dominant", []string{"check", "dominant"}},
		{"varied", []string{"check", "varied"}},
		{"nolog", []string{"check", "nolog"}},
		{"ghost", []string{"check", "ghost"}},
		{"custom-window-2", []string{"check", "few", "2"}}, // win=2 → last 2 identical → STUCK
		// Pathological window tokens: the numeric-string vs string comparison and
		// the verbatim `win` echo — edge cases the port must reproduce.
		{"window-4x-varied4", []string{"check", "varied4", "4x"}},
		{"window-2.5-varied4", []string{"check", "varied4", "2.5"}},
		{"window-leadspace-varied4", []string{"check", "varied4", " 4"}},
		{"window-04-identical", []string{"check", "identical", "04"}},
		{"window-plus4-identical", []string{"check", "identical", "+4"}},
		{"no-cmd", nil},
		{"no-slug", []string{"check"}},
		{"bad-cmd", []string{"bogus", "few"}},
	} {
		c := parityCase{workdir: work, env: libRootEnv(work), goArgs: append([]string{"stuck"}, tc.goArgs...)}
		t.Run(tc.name, func(t *testing.T) { c.assertEqual(t) })
	}
}

// TestStuckLogHashParity proves the `log` side effect: it appends a record whose
// sha1 column is deterministic given the logged target, so that column is
// snapshotted (the epoch column is nondeterministic and excluded by reading only
// field 3). A second log appends a second record, and logging to a missing
// workspace is a silent no-op.
func TestStuckLogHashParity(t *testing.T) {
	field3 := func(t *testing.T, path string) string {
		t.Helper()
		fields := bytes.Fields(bytes.TrimRight(readFileT(t, path), "\n"))
		if len(fields) < 3 {
			t.Fatalf("log line has <3 fields: %q", string(readFileT(t, path)))
		}
		return string(fields[2])
	}

	// The new-schema features layout under DEVRITES_ROOT.
	gw := t.TempDir()
	mkdirAllT(t, gw, ".devrites/features/feat")
	goLog := filepath.Join(gw, ".devrites/features/feat/action.log")
	runArgv(t, gw, libRootEnv(gw), "", binPath, "stuck", "log", "feat", "Bash", "src/a.js")
	goHash := field3(t, goLog)

	// Snapshot the sha1 column (the epoch column is masked by reading only field 3).
	assertGoldenKey(t, t.Name(), goHash)
	if len(goHash) != 40 {
		t.Errorf("sha1 hex = %q, want 40 chars", goHash)
	}

	// A second log appends a second record.
	runArgv(t, gw, libRootEnv(gw), "", binPath, "stuck", "log", "feat", "Edit", "src/b.js")
	if n := fileLines(t, goLog); n != 2 {
		t.Errorf("after two logs: %d lines, want 2", n)
	}
	// Logging to a missing workspace is a no-op (never fails the caller).
	if _, code := runArgv(t, gw, libRootEnv(gw), "", binPath, "stuck", "log", "ghost", "Bash", "x"); code != 0 {
		t.Errorf("log on missing workspace: exit=%d, want 0", code)
	}
	if isFileT(filepath.Join(gw, ".devrites/features/ghost/action.log")) {
		t.Error("log on missing workspace created a file; must be a no-op")
	}
}

func readFileT(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func isFileT(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}
