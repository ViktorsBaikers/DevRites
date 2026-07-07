package main_test

// Parity + diagnostic tests for the ported devrites-lib commands (issue 08).
//
// Parity: each command is run through BOTH the legacy bash script and the Go
// binary against the SAME fixture, asserting identical stdout + exit code — the
// contract the parity oracle defines (stderr is diagnostic, not contract).
//
// Diagnostic: a second pass asserts the Go stderr carries the same actionable
// finding substrings the legacy tests/*.sh grep for, so the port keeps the
// script's diagnostic quality even though stderr is excluded from parity.

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// libEnv forces byte-order collation so `sort` in the bash script and
// sort.Strings in Go agree (matters for check-acceptance's id ordering).
var libEnv = []string{"LC_ALL=C", "DEVRITES_ROOT="}

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
	p := filepath.Join(workdir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
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

func libScript(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(repoRoot(t), "pack", ".claude", "skills", "devrites-lib", "scripts", name)
	if _, err := os.Stat(p); err != nil {
		t.Skipf("legacy script not present: %v", err)
	}
	return p
}

// ---- spec-validate --------------------------------------------------------

// Spec fixtures mirror tests/spec-validate-test.sh so parity is proven on the
// exact cases the legacy test already covers.
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
	requireBash(t)
	script := libScript(t, "spec-validate.sh")
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
			bash:    []string{"bash", script, arg},
			goArgs:  []string{"spec-validate", arg},
		}
		t.Run("arg="+arg, func(t *testing.T) { c.assertEqual(t) })
	}

	// No-argument usage path (exit 2) — asserted separately since arg is absent.
	(parityCase{
		workdir: work,
		env:     libEnv,
		bash:    []string{"bash", script},
		goArgs:  []string{"spec-validate"},
	}).assertEqual(t)
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
	requireBash(t)
	script := libScript(t, "check-acceptance.sh")
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
			bash:    []string{"bash", script, arg},
			goArgs:  []string{"check-acceptance", arg},
		}
		t.Run("arg="+arg, func(t *testing.T) { c.assertEqual(t) })
	}

	// No-argument usage path (exit 2).
	(parityCase{
		workdir: work,
		env:     libEnv,
		bash:    []string{"bash", script},
		goArgs:  []string{"check-acceptance"},
	}).assertEqual(t)
}
