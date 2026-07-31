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
