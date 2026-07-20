package state

import (
	"sort"
	"strings"
	"unicode"
)

// Canonical cursor keys. Callers use these constants so schema spelling changes
// remain local to the cursor package; normalizeCursorKey still accepts legacy
// presentation aliases at the text boundary.
const (
	CursorPhase              = "phase"
	CursorStatus             = "status"
	CursorNextAction         = "next_action"
	CursorQuestionID         = "question_id"
	CursorActiveSlice        = "active_slice"
	CursorAFKSlicesRemaining = "afk_slices_remaining"
)

var cursorKeyAliases = map[string]string{
	"nextstep": CursorNextAction,
	"qid":      CursorQuestionID,
}

// CursorKeyAlias is the generated-manifest view of a compatibility key.
type CursorKeyAlias struct {
	Alias     string `json:"alias"`
	Canonical string `json:"canonical"`
}

// CursorKeyAliases returns compatibility keys in deterministic order.
func CursorKeyAliases() []CursorKeyAlias {
	aliases := make([]string, 0, len(cursorKeyAliases))
	for alias := range cursorKeyAliases {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	out := make([]CursorKeyAlias, 0, len(aliases))
	for _, alias := range aliases {
		out = append(out, CursorKeyAlias{Alias: alias, Canonical: cursorKeyAliases[alias]})
	}
	return out
}

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
	normalized := b.String()
	if canonical, ok := cursorKeyAliases[normalized]; ok {
		return normalizeCursorKey(canonical)
	}
	return normalized
}
