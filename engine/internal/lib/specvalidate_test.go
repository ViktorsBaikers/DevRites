package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpecValidateCharacterization(t *testing.T) {
	work := t.TempDir()
	writeSpec(t, work, "valid", `### Requirement: Export
The system SHALL export reports.
#### Scenario: happy path
- **WHEN** export is requested **THEN** a file is returned
`)
	writeSpec(t, work, "flat", `# Spec
## Acceptance criteria
- [ ] export returns a CSV
`)
	writeSpec(t, work, "nothen", `### Requirement: Export
The system SHALL export reports.
#### Scenario: happy path
- **WHEN** export is requested
`)

	for _, tc := range []struct {
		name       string
		arg        string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{name: "valid", arg: filepath.Join(work, "valid"), wantCode: 0, wantStdout: "well-formed"},
		{name: "flat", arg: filepath.Join(work, "flat"), wantCode: 0, wantStdout: "simple acceptance form"},
		{name: "missing then", arg: filepath.Join(work, "nothen"), wantCode: 1, wantStderr: "has no THEN line"},
		{name: "missing arg", wantCode: 2, wantStderr: "usage: devrites-engine spec-validate"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
			code := SpecValidate(tc.arg, "", work, stdout, stderr)
			if code != tc.wantCode {
				t.Fatalf("SpecValidate code=%d, want %d; stdout=%q stderr=%q", code, tc.wantCode, stdout.String(), stderr.String())
			}
			if tc.wantStdout != "" && !strings.Contains(stdout.String(), tc.wantStdout) {
				t.Fatalf("stdout=%q, want substring %q", stdout.String(), tc.wantStdout)
			}
			if tc.wantStderr != "" && !strings.Contains(stderr.String(), tc.wantStderr) {
				t.Fatalf("stderr=%q, want substring %q", stderr.String(), tc.wantStderr)
			}
		})
	}
}

func TestSpecValidateAgainstLedger(t *testing.T) {
	// Seed a capability ledger: theming already holds "Dark mode".
	ledger := t.TempDir()
	writeSpec(t, ledger, "theming", `### Requirement: Dark mode honors the system preference
The system SHALL default to the OS colour-scheme.
#### Scenario: first load
- **WHEN** a visitor loads the app **THEN** the theme matches the OS setting
`)

	work := t.TempDir()
	// Correct: MODIFIED an existing requirement, ADD a genuinely new one.
	writeSpec(t, work, "ok", `## MODIFIED Requirements — capability: theming
### Requirement: Dark mode honors the system preference
The system SHALL default to the OS colour-scheme and persist an override.
#### Scenario: stored override
- **WHEN** a visitor has toggled the theme **THEN** the stored choice wins

## ADDED Requirements — capability: theming
### Requirement: A theme toggle is reachable from settings
The system SHALL expose a theme control in settings.
#### Scenario: toggle present
- **WHEN** settings opens **THEN** a theme control is shown
`)
	// Wrong: ADD something that already exists; MODIFY something absent.
	writeSpec(t, work, "wrong", `## ADDED Requirements — capability: theming
### Requirement: Dark mode honors the system preference
The system SHALL default to the OS colour-scheme.
#### Scenario: first load
- **WHEN** a visitor loads the app **THEN** the theme matches the OS

## MODIFIED Requirements — capability: theming
### Requirement: Nonexistent behavior
The system SHALL do a thing it never did before.
#### Scenario: x
- **WHEN** y **THEN** z
`)

	for _, tc := range []struct {
		name       string
		slug       string
		wantCode   int
		wantStderr []string
	}{
		{name: "reconciled deltas pass", slug: "ok", wantCode: 0},
		{name: "misclassified deltas fail", slug: "wrong", wantCode: 1, wantStderr: []string{
			"marked ADDED but already exists in ledger capability \"theming\"",
			"marked MODIFIED but is absent from ledger capability \"theming\"",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
			code := SpecValidate(filepath.Join(work, tc.slug), ledger, work, stdout, stderr)
			if code != tc.wantCode {
				t.Fatalf("code=%d want %d; stdout=%q stderr=%q", code, tc.wantCode, stdout.String(), stderr.String())
			}
			for _, want := range tc.wantStderr {
				if !strings.Contains(stderr.String(), want) {
					t.Fatalf("stderr=%q, want substring %q", stderr.String(), want)
				}
			}
		})
	}
}

func writeSpec(t *testing.T, work, slug, content string) {
	t.Helper()
	dir := filepath.Join(work, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
