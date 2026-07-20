package state

import (
	"strings"
	"unicode"
)

type cursorLineKind uint8

const (
	cursorLineUnknown cursorLineKind = iota
	cursorLineLegacy
	cursorLineTable
)

// CursorField reads a state.md field from either the canonical | key | value |
// cursor table or the legacy "- Key: value" form.
func CursorField(lines []string, key string) (string, bool) {
	want := normalizeCursorKey(key)
	for _, line := range lines {
		gotKey, value, _, ok := parseCursorLine(line)
		if ok && normalizeCursorKey(gotKey) == want {
			return value, true
		}
	}
	return "", false
}

// SetCursorField replaces an existing state.md field without changing whether
// the file uses the canonical table or legacy bullet form.
func SetCursorField(lines []string, key, value string) ([]string, bool) {
	want := normalizeCursorKey(key)
	out := append([]string(nil), lines...)
	for i, line := range out {
		gotKey, _, kind, ok := parseCursorLine(line)
		if !ok || normalizeCursorKey(gotKey) != want {
			continue
		}
		switch kind {
		case cursorLineTable:
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			out[i] = indent + "| " + strings.TrimSpace(gotKey) + " | " + value + " |"
		case cursorLineLegacy:
			colon := strings.IndexByte(line, ':')
			out[i] = line[:colon+1] + " " + value
		}
		return out, true
	}
	return out, false
}

func parseCursorLine(line string) (key, value string, kind cursorLineKind, ok bool) {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) >= 2 && trimmed[0] == '|' && trimmed[len(trimmed)-1] == '|' {
		cells := strings.Split(trimmed[1:len(trimmed)-1], "|")
		if len(cells) >= 2 {
			return strings.TrimSpace(cells[0]), strings.TrimSpace(cells[1]), cursorLineTable, true
		}
	}

	trimmed = strings.TrimLeft(trimmed, "-*+ \t")
	key, value, ok = strings.Cut(trimmed, ":")
	if !ok || strings.TrimSpace(key) == "" {
		return "", "", cursorLineUnknown, false
	}
	return strings.TrimSpace(key), strings.TrimSpace(value), cursorLineLegacy, true
}

func normalizeCursorKey(key string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(key) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	switch b.String() {
	case "nextstep":
		return "nextaction"
	case "qid":
		return "questionid"
	default:
		return b.String()
	}
}
