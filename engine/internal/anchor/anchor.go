// Package anchor matches a quoted line against file text: whitespace-normalised
// containment inside a line range, with quotes too short to identify a line
// rejected. /overhaul admit and `check anchors` share it so both admit the
// same quotes.
package anchor

import (
	"strings"
	"unicode/utf8"
)

// MinRunes is the shortest quote (non-space runes) that anchors on its own; a
// shorter one anchors only when it is the whole cited window.
const MinRunes = 8

// Result is the outcome of matching one quote.
type Result int

const (
	NotFound Result = iota
	Found
	TooShort
)

// Normalize collapses every whitespace run to one space and trims the ends.
func Normalize(s string) string { return strings.Join(strings.Fields(s), " ") }

// Lines decodes b as text (invalid UTF-8 replaced) split like Python's str.splitlines.
func Lines(b []byte) []string { return Splitlines(strings.ToValidUTF8(string(b), "�")) }

// Splitlines is Python's str.splitlines().
func Splitlines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		switch r {
		case '\n', '\r', '\v', '\f', 0x1c, 0x1d, 0x1e, 0x85, 0x2028, 0x2029:
			out = append(out, s[start:i-size])
			if r == '\r' && i < len(s) && s[i] == '\n' {
				i++
			}
			start = i
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// At matches quote inside lines start..end (1-based, inclusive). The caller
// checks 1 <= start <= end <= len(lines).
func At(lines []string, start, end int, quote string) Result {
	q := Normalize(quote)
	w := Normalize(strings.Join(lines[start-1:end], " "))
	if q == "" || !strings.Contains(w, q) {
		return NotFound
	}
	if utf8.RuneCountInString(strings.ReplaceAll(q, " ", "")) < MinRunes && q != w {
		return TooShort
	}
	return Found
}

// Find returns every 1-based line whose own text At admits quote on.
func Find(lines []string, quote string) []int {
	var hits []int
	for i := range lines {
		if At(lines, i+1, i+1, quote) == Found {
			hits = append(hits, i+1)
		}
	}
	return hits
}
