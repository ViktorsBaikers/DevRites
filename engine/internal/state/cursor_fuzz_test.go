package state

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

var cursorFuzzKeys = []string{
	CursorPhase, CursorStatus, CursorNextAction, CursorQuestionID,
	CursorActiveSlice, CursorAFKSlicesRemaining, CursorReturnPhase,
	CursorReturnNextAction, CursorSchema,
}

func FuzzCursorRoundTrip(f *testing.F) {
	f.Add(uint8(0), "spec", []byte("# Feature\n\n## Cursor\n\n| phase | init |\n| --- | --- |\n| status | running |"))
	f.Add(uint8(1), "", []byte("- phase: spec\n- status: running"))
	f.Add(uint8(8), "5", []byte("## Cursor\n\n| phase | init |"))
	f.Add(uint8(2), "a|b", []byte("| status | running |"))
	f.Add(uint8(4), "two\nlines", []byte("| status | running |"))
	f.Add(uint8(5), "value with spaces", []byte{})
	f.Fuzz(func(t *testing.T, keyIndex uint8, value string, doc []byte) {
		// Docs and values with NUL bytes or invalid UTF-8 are outside the
		// format: the read and write paths refuse them, so round-trip says
		// nothing there.
		if !utf8.Valid(doc) || bytes.IndexByte(doc, 0) >= 0 ||
			!utf8.ValidString(value) || strings.IndexByte(value, 0) >= 0 {
			return
		}
		key := cursorFuzzKeys[int(keyIndex)%len(cursorFuzzKeys)]
		lines := strings.Split(string(doc), "\n")
		upserted := UpsertCursorField(lines, key, value)
		if strings.ContainsAny(value, "\r\n") {
			// Line breaks are representable in no form: the write must be
			// declined with the document unchanged.
			if !slices.Equal(upserted, lines) {
				t.Fatal("newline value was not declined unchanged")
			}
			return
		}
		// The upsert contract: either the document is unchanged, or the value
		// is readable from the result.
		got, ok := CursorField(upserted, key)
		if !slices.Equal(upserted, lines) && (!ok || got != strings.TrimSpace(value)) {
			t.Fatalf("upsert changed the document but %q is not readable (got %q, ok %v)", value, got, ok)
		}
		deleted := DeleteCursorField(upserted, key)
		if _, ok := CursorField(deleted, key); ok {
			t.Fatalf("deleted key %q still readable", key)
		}
		replaced, ok := SetCursorField(upserted, key, value)
		if ok {
			got, ok := CursorField(replaced, key)
			if !ok || got != strings.TrimSpace(value) {
				t.Fatalf("set %q not readable (got %q, ok %v)", value, got, ok)
			}
		}
	})
}

func FuzzCursorParsing(f *testing.F) {
	f.Add([]byte("| phase | spec |"), "phase", "x")
	f.Add([]byte("```\n| q | a |\n```"), "qid", "y")
	f.Add([]byte("- nextstep: run"), "next_action", "z")
	f.Add([]byte("## Cursor\n\n| a | b | extra |"), "", "")
	f.Fuzz(func(t *testing.T, doc []byte, key, value string) {
		if !utf8.Valid(doc) || bytes.IndexByte(doc, 0) >= 0 {
			return
		}
		lines := strings.Split(string(doc), "\n")
		_, _ = CursorField(lines, key)
		_ = CursorForm(lines)
		out, _ := SetCursorField(lines, key, value)
		out = UpsertCursorField(out, key, value)
		_ = DeleteCursorField(out, key)
		_, _ = ConvertCursorToTable(lines)
	})
}
