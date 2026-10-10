package state

import (
	"strings"
	"testing"
)

func TestCursorFieldReadsCanonicalTableAndLegacyLines(t *testing.T) {
	table := []string{
		"| Key | Value |",
		"| --- | --- |",
		"| phase | temper |",
		"| status | awaiting_human |",
		"| next_action | /rite-define |",
	}
	legacy := []string{
		"- Phase: build",
		"- Status: running",
		"- Next step: /rite-prove",
	}

	for _, tc := range []struct {
		name  string
		lines []string
		key   string
		want  string
	}{
		{"table phase", table, "phase", "temper"},
		{"table status", table, "status", "awaiting_human"},
		{"table next alias", table, "Next step", "/rite-define"},
		{"legacy phase", legacy, "phase", "build"},
		{"legacy next alias", legacy, "next_action", "/rite-prove"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := CursorField(tc.lines, tc.key)
			if !ok || got != tc.want {
				t.Fatalf("CursorField(%q) = %q, %v; want %q, true", tc.key, got, ok, tc.want)
			}
		})
	}
}

func TestSetCursorFieldPreservesCanonicalAndLegacyFormats(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []string
		key   string
		value string
		want  string
	}{
		{"table", []string{"| status | awaiting_human |"}, "status", "running", "| status | running |"},
		{"legacy", []string{"- AFK slices remaining: 2"}, "afk_slices_remaining", "1", "- AFK slices remaining: 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := SetCursorField(tc.lines, tc.key, tc.value)
			if !ok || len(got) != 1 || got[0] != tc.want {
				t.Fatalf("SetCursorField() = %v, %v; want [%q], true", got, ok, tc.want)
			}
		})
	}
}

func TestUpsertAndDeleteCursorFieldStayInsideCursorTable(t *testing.T) {
	lines := []string{
		"# State",
		"",
		"## Cursor",
		"| Key | Value |",
		"| --- | --- |",
		"| phase | build |",
		"| status | running |",
		"",
		"## Awaiting human",
		"| Key | Value |",
		"| --- | --- |",
		"| question_id | q-1 |",
	}
	lines = UpsertCursorField(lines, CursorReturnPhase, "build")
	value, ok := CursorField(lines, CursorReturnPhase)
	if !ok || value != "build" {
		t.Fatalf("return phase=(%q,%v), want build", value, ok)
	}
	returnIndex, questionIndex := -1, -1
	for i, line := range lines {
		if strings.Contains(line, CursorReturnPhase) {
			returnIndex = i
		}
		if strings.Contains(line, CursorQuestionID) {
			questionIndex = i
		}
	}
	if returnIndex < 0 || questionIndex < 0 || returnIndex > questionIndex {
		t.Fatalf("upsert inserted outside cursor table:\n%s", strings.Join(lines, "\n"))
	}
	lines = DeleteCursorField(lines, CursorReturnPhase)
	if _, ok := CursorField(lines, CursorReturnPhase); ok {
		t.Fatalf("DeleteCursorField left return phase:\n%s", strings.Join(lines, "\n"))
	}
}

func TestCursorHelpersIgnoreFencedExamples(t *testing.T) {
	lines := strings.Split(`# State

~~~md
## Cursor
| Key | Value |
| --- | --- |
| status | example |
~~~

## Cursor
| Key | Value |
| --- | --- |
| phase | build |
| status | running |`, "\n")

	if got, ok := CursorField(lines, CursorStatus); !ok || got != "running" {
		t.Fatalf("CursorField(status) = %q, %v; want running, true", got, ok)
	}
	lines, ok := SetCursorField(lines, CursorStatus, "complete")
	if !ok || lines[6] != "| status | example |" || lines[13] != "| status | complete |" {
		t.Fatalf("SetCursorField changed the wrong line:\n%s", strings.Join(lines, "\n"))
	}
	lines = UpsertCursorField(lines, CursorReturnPhase, "build")
	if lines[6] != "| status | example |" {
		t.Fatalf("UpsertCursorField changed fenced content:\n%s", strings.Join(lines, "\n"))
	}
	lines = DeleteCursorField(lines, CursorStatus)
	if lines[6] != "| status | example |" {
		t.Fatalf("DeleteCursorField changed fenced content:\n%s", strings.Join(lines, "\n"))
	}
	if _, ok := CursorField(lines, CursorStatus); ok {
		t.Fatalf("DeleteCursorField left an authoritative status:\n%s", strings.Join(lines, "\n"))
	}
}

func TestCursorHelpersRejectCorruptTextWithoutMutation(t *testing.T) {
	for _, lines := range [][]string{
		{"| status | run\x00ning |"},
		{"| status | run\xffning |"},
	} {
		original := strings.Join(lines, "\n")
		if value, ok := CursorField(lines, CursorStatus); ok || value != "" {
			t.Fatalf("CursorField(corrupt) = %q, %v", value, ok)
		}
		if got, ok := SetCursorField(lines, CursorStatus, "complete"); ok || strings.Join(got, "\n") != original {
			t.Fatalf("SetCursorField(corrupt) = %q, %v", got, ok)
		}
		if got := UpsertCursorField(lines, CursorStatus, "complete"); strings.Join(got, "\n") != original {
			t.Fatalf("UpsertCursorField(corrupt) mutated input: %q", got)
		}
		if got := DeleteCursorField(lines, CursorStatus); strings.Join(got, "\n") != original {
			t.Fatalf("DeleteCursorField(corrupt) mutated input: %q", got)
		}
	}
}

// ConvertCursorToTable and CursorForm must obtain their
// structural lines through structuralCursorLines (markdowntext.Structural), the
// same way CursorField, SetCursorField, UpsertCursorField and DeleteCursorField
// do. The cases below separate "the fence was masked" from "the call site never
// consulted the masker at all".

const ledgerWithFencedExample = `# State

## Recovery note

The pre-v5 ledger looked like this:

~~~md
- Phase: build
- Status: running
~~~

## Cursor

- Phase: build
- Status: running
`

// Fenced-only legacy bullets and a live table row: a ledger whose only legacy
// bullets sit inside a fence. Nothing here is structural, so nothing may change.
func TestConvertCursorToTableIsByteIdenticalOnFencedOnlyLedger(t *testing.T) {
	lines := strings.Split("# State\n\n~~~md\n- Phase: build\n~~~\n\n## Cursor\n| phase | spec |", "\n")
	original := strings.Join(lines, "\n")
	out, changed := ConvertCursorToTable(lines)
	if changed {
		t.Fatalf("ConvertCursorToTable reported a change for a fenced-only ledger:\n%s", strings.Join(out, "\n"))
	}
	if got := strings.Join(out, "\n"); got != original {
		t.Fatalf("ConvertCursorToTable mutated a fenced-only ledger:\ngot  %q\nwant %q", got, original)
	}
}

// The discriminating case: one document, the same key spelling inside and
// outside a fence. The fenced bullet must survive untouched while the live
// bullet is still converted, which neither "convert nothing" nor "convert
// everything" can satisfy.
func TestConvertCursorToTableConvertsLiveBulletsAndLeavesFencedOnes(t *testing.T) {
	lines := strings.Split(ledgerWithFencedExample, "\n")
	out, changed := ConvertCursorToTable(lines)
	if !changed {
		t.Fatalf("ConvertCursorToTable converted nothing; the live bullets must still convert:\n%s", strings.Join(out, "\n"))
	}
	if out[7] != "- Phase: build" || out[8] != "- Status: running" {
		t.Fatalf("ConvertCursorToTable rewrote the fenced example:\n%s", strings.Join(out, "\n"))
	}
	if out[13] != "| phase | build |" || out[14] != "| status | running |" {
		t.Fatalf("ConvertCursorToTable did not convert the live bullets:\n%s", strings.Join(out, "\n"))
	}
}

// A fence that is still open masks the rest of the document, which is what
// markdowntext.Structural does. The bullet after the unclosed fence is not converted.
func TestConvertCursorToTableTreatsAnUnclosedFenceAsStructuralMasking(t *testing.T) {
	lines := strings.Split("~~~md\n- Phase: build\n\n- Phase: build\n", "\n")
	out, changed := ConvertCursorToTable(lines)
	if changed {
		t.Fatalf("ConvertCursorToTable converted content after an unclosed fence:\n%s", strings.Join(out, "\n"))
	}
	if out[1] != "- Phase: build" || out[3] != "- Phase: build" {
		t.Fatalf("ConvertCursorToTable rewrote content after an unclosed fence:\n%s", strings.Join(out, "\n"))
	}
}

// CursorForm must report the presentation of the structural document only.
func TestCursorFormIgnoresFencedTableRows(t *testing.T) {
	fenced := []string{"~~~md", "| phase | x |", "~~~"}
	if got := CursorForm(fenced); got != "none" {
		t.Fatalf("CursorForm(fenced-only table) = %q, want none", got)
	}
	if got := CursorForm([]string{"| phase | x |"}); got != "table" {
		t.Fatalf("CursorForm(live table) = %q, want table", got)
	}
	if got := CursorForm([]string{"~~~md", "- phase: build", "~~~", "- Phase: build"}); got != "legacy" {
		t.Fatalf("CursorForm(fenced bullet plus live bullet) = %q, want legacy", got)
	}
}

// Proves the masker is consulted rather than the outcome being right by some
// other route. markdowntext.Structural is the only thing in this package that
// rejects NUL bytes, so a document that carries a valid, unfenced cursor bullet
// next to one NUL byte must be declined whole: ConvertCursorToTable returns
// changed=false and CursorForm reports "none". A parser that never calls the
// masker reports changed=true / "table" on exactly this input.
func TestCursorFormAndConvertConsultTheFenceMasker(t *testing.T) {
	corrupt := []string{"- Phase: build", "- Status: run\x00ning"}
	out, changed := ConvertCursorToTable(corrupt)
	if changed {
		t.Fatalf("ConvertCursorToTable converted a document the masker must decline:\n%q", strings.Join(out, "\n"))
	}
	if strings.Join(out, "\n") != strings.Join(corrupt, "\n") {
		t.Fatalf("ConvertCursorToTable mutated a document the masker must decline:\n%q", strings.Join(out, "\n"))
	}
	if got := CursorForm([]string{"| phase | build |", "- Status: run\x00ning"}); got != "none" {
		t.Fatalf("CursorForm(corrupt table) = %q, want none: the masker was not consulted", got)
	}
	if got := CursorForm([]string{"- Phase: build", "- Status: run\xffning"}); got != "none" {
		t.Fatalf("CursorForm(corrupt legacy) = %q, want none: the masker was not consulted", got)
	}
}

// Bullets in prose sections outside any fence still convert.
func TestConvertCursorToTableStillConvertsProseSections(t *testing.T) {
	lines := strings.Split("# State\n\n## Notes\n\n- Phase: build\n- owner: platform\n", "\n")
	out, changed := ConvertCursorToTable(lines)
	if !changed {
		t.Fatalf("ConvertCursorToTable converted nothing:\n%s", strings.Join(out, "\n"))
	}
	if out[4] != "| phase | build |" {
		t.Fatalf("ConvertCursorToTable did not convert the prose-section bullet:\n%s", strings.Join(out, "\n"))
	}
	if out[5] != "- owner: platform" {
		t.Fatalf("ConvertCursorToTable converted a non-cursor bullet:\n%s", strings.Join(out, "\n"))
	}
}
