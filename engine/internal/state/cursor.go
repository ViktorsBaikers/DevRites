package state

import (
	"sort"
	"strings"
	"unicode"

	"github.com/devrites/devrites/internal/markdowntext"
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
	CursorReturnPhase        = "return_phase"
	CursorReturnNextAction   = "return_next_action"
	CursorSchema             = "schema"
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
	structural, ok := structuralCursorLines(lines)
	if !ok {
		return "", false
	}
	want := normalizeCursorKey(key)
	for _, line := range structural {
		gotKey, value, _, ok := parseCursorLine(line)
		if ok && normalizeCursorKey(gotKey) == want {
			return value, true
		}
	}
	return "", false
}

// SetCursorField replaces an existing state.md field without changing whether
// the file uses the canonical table or legacy bullet form. A value carrying a
// raw pipe is declined (false) for table rows, and a value carrying a line
// break is declined everywhere: neither is representable in a cell. A write
// the reader cannot read back is declined too.
func SetCursorField(lines []string, key, value string) ([]string, bool) {
	if strings.ContainsAny(value, "\r\n") {
		return append([]string(nil), lines...), false
	}
	structural, ok := structuralCursorLines(lines)
	if !ok {
		return append([]string(nil), lines...), false
	}
	out, ok := setCursorField(lines, structural, key, value)
	if !ok {
		return out, false
	}
	if !cursorReadable(out, key, value) {
		return append([]string(nil), lines...), false
	}
	return out, true
}

func setCursorField(lines, structural []string, key, value string) ([]string, bool) {
	want := normalizeCursorKey(key)
	out := append([]string(nil), lines...)
	for i, line := range structural {
		gotKey, _, kind, ok := parseCursorLine(line)
		if !ok || normalizeCursorKey(gotKey) != want {
			continue
		}
		original := out[i]
		switch kind {
		case cursorLineTable:
			if strings.Contains(value, "|") {
				return out, false
			}
			indent := original[:len(original)-len(strings.TrimLeft(original, " \t"))]
			out[i] = indent + "| " + strings.TrimSpace(gotKey) + " | " + value + " |"
		case cursorLineLegacy:
			colon := strings.IndexByte(original, ':')
			out[i] = original[:colon+1] + " " + value
		}
		return out, true
	}
	return out, false
}

// UpsertCursorField replaces key when present or inserts it into the canonical
// cursor table (falling back to a legacy bullet when no table exists). A value
// carrying a raw pipe keeps the legacy bullet form — a cell cannot represent
// it — and a value carrying a line break is declined with the input returned
// unchanged. Every write is verified readable before it is returned; an
// unreadable write (for example, appending after an unclosed fence) leaves the
// document unchanged.
func UpsertCursorField(lines []string, key, value string) []string {
	if strings.ContainsAny(value, "\r\n") {
		return append([]string(nil), lines...)
	}
	structural, ok := structuralCursorLines(lines)
	if !ok {
		return append([]string(nil), lines...)
	}
	var candidate []string
	if strings.Contains(value, "|") {
		candidate = append(DeleteCursorField(lines, key), "- "+key+": "+value)
	} else if out, ok := setCursorField(lines, structural, key, value); ok {
		candidate = out
	} else if lastTable := lastCursorTableRow(structural); lastTable >= 0 {
		candidate = append(append([]string(nil), lines...), "")
		copy(candidate[lastTable+2:], candidate[lastTable+1:])
		candidate[lastTable+1] = "| " + key + " | " + value + " |"
	} else {
		candidate = append(append([]string(nil), lines...), "- "+key+": "+value)
	}
	if cursorReadable(candidate, key, value) {
		return candidate
	}
	return append([]string(nil), lines...)
}

// lastCursorTableRow returns the index of the final table row inside the
// Cursor section, or -1 when no table exists.
func lastCursorTableRow(structural []string) int {
	lastTable := -1
	inCursor := false
	for i, line := range structural {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			if inCursor {
				break
			}
			inCursor = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")), "Cursor")
			continue
		}
		if len(trimmed) >= 2 && trimmed[0] == '|' && trimmed[len(trimmed)-1] == '|' {
			gotKey, _, kind, ok := parseCursorLine(line)
			normalized := normalizeCursorKey(gotKey)
			if !inCursor && ok && (normalized == normalizeCursorKey(CursorPhase) ||
				normalized == normalizeCursorKey(CursorStatus) ||
				normalized == normalizeCursorKey(CursorNextAction)) {
				inCursor = true
			}
			if inCursor && ok && kind == cursorLineTable {
				lastTable = i
			}
		}
	}
	return lastTable
}

// cursorReadable reports whether key reads back as value from a candidate
// document — the write side of parseCursorLine's cell normalization.
func cursorReadable(lines []string, key, value string) bool {
	got, ok := CursorField(lines, key)
	return ok && got == strings.TrimSpace(value)
}

// DeleteCursorField removes every presentation of key from canonical or legacy
// state without disturbing unrelated prose.
func DeleteCursorField(lines []string, key string) []string {
	structural, ok := structuralCursorLines(lines)
	if !ok {
		return append([]string(nil), lines...)
	}
	want := normalizeCursorKey(key)
	out := make([]string, 0, len(lines))
	for i, line := range structural {
		gotKey, _, _, ok := parseCursorLine(line)
		if ok && normalizeCursorKey(gotKey) == want {
			continue
		}
		out = append(out, lines[i])
	}
	return out
}

// CursorForm reports the presentation of the state.md cursor: "table" when a
// canonical table row exists, "legacy" when only bullet fields exist, and
// "none" when neither is present.
func CursorForm(lines []string) string {
	form := "none"
	for _, line := range lines {
		_, _, kind, ok := parseCursorLine(line)
		if !ok {
			continue
		}
		if kind == cursorLineTable {
			return "table"
		}
		if form == "none" {
			form = "legacy"
		}
	}
	return form
}

// ConvertCursorToTable rewrites legacy bullet cursor fields into canonical
// table rows in place. A bullet whose key resolves to a canonical cursor key
// is a cursor field wherever it appears — the same rule CursorField uses to
// read — so such a bullet is converted even inside prose sections. Prose
// without a canonical cursor key is preserved, and a value carrying a raw pipe
// keeps its bullet form because a table cell cannot represent it. It reports
// whether any line changed.
func ConvertCursorToTable(lines []string) ([]string, bool) {
	changed := false
	out := append([]string(nil), lines...)
	for i, line := range out {
		key, value, kind, ok := parseCursorLine(line)
		if !ok || kind != cursorLineLegacy {
			continue
		}
		if !isMigratableCursorKey(key) {
			continue
		}
		if strings.Contains(value, "|") {
			continue
		}
		spelling, ok := canonicalCursorSpelling(key)
		if !ok {
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		out[i] = indent + "| " + spelling + " | " + value + " |"
		changed = true
	}
	return out, changed
}

func isMigratableCursorKey(key string) bool {
	_, ok := canonicalCursorSpelling(key)
	return ok
}

// canonicalCursorSpelling maps a cursor key (including aliases) to the
// canonical table-row spelling. Matching is normalization-based, so
// "Next step" resolves to the "next_action" spelling.
func canonicalCursorSpelling(key string) (string, bool) {
	normalized := normalizeCursorKey(key)
	canonicals := []string{CursorPhase, CursorStatus, CursorNextAction, CursorQuestionID,
		CursorActiveSlice, CursorAFKSlicesRemaining, CursorReturnPhase,
		CursorReturnNextAction, CursorSchema}
	for _, canonical := range canonicals {
		if normalized == normalizeCursorKey(canonical) {
			return canonical, true
		}
	}
	return "", false
}

func structuralCursorLines(lines []string) ([]string, bool) {
	if len(lines) == 0 {
		return nil, true
	}
	structural, err := markdowntext.Structural([]byte(strings.Join(lines, "\n")))
	if err != nil {
		return nil, false
	}
	masked := strings.Split(string(structural), "\n")
	return masked, len(masked) == len(lines)
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
	normalized := rawCursorKey(key)
	if canonical, ok := cursorKeyAliases[normalized]; ok {
		return normalizeCursorKey(canonical)
	}
	return normalized
}

func rawCursorKey(key string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(key) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
