package state

import "testing"

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
