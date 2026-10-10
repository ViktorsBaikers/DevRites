package markdowntext

import (
	"bytes"
	"errors"
	"unicode/utf8"
)

// Structural returns Markdown with fenced blocks replaced by spaces. Byte
// offsets and line breaks stay aligned with the original input.
func Structural(input []byte) ([]byte, error) {
	if !utf8.Valid(input) {
		return nil, errors.New("markdown text contains malformed UTF-8")
	}
	if bytes.IndexByte(input, 0) >= 0 {
		return nil, errors.New("markdown text contains NUL byte")
	}

	out := append([]byte(nil), input...)
	var marker byte
	fenceWidth := 0
	anchor, anchorStart := -1, 0
	afterBlank := true
	for start := 0; start < len(input); {
		lineEnd := bytes.IndexByte(input[start:], '\n')
		if lineEnd < 0 {
			lineEnd = len(input)
		} else {
			lineEnd += start
		}
		contentEnd := lineEnd
		if contentEnd > start && input[contentEnd-1] == '\r' {
			contentEnd--
		}
		line := input[start:contentEnd]

		if marker == 0 {
			if blank(line) {
				afterBlank = true
			} else if nextMarker, width, ok := openingFence(line, 3); ok {
				marker, fenceWidth = nextMarker, width
				mask(out[start:lineEnd])
				anchor, afterBlank = -1, false
			} else if nextMarker, width, ok := openingFence(line, anchor+3); ok && anchor >= 0 && indentColumns(line) >= anchor {
				anchored := anchor
				anchor, afterBlank = -1, false
				if closeEnd := listFenceEnd(input, lineEnd, nextMarker, width, indentColumns(line), anchored); closeEnd >= 0 {
					if !loneCR(input[anchorStart:closeEnd]) {
						mask(out[start:closeEnd])
					}
					if closeEnd == len(input) {
						break
					}
					start = closeEnd + 1
					continue
				}
			} else {
				column, ok := topLevelItem(line)
				if ok && (afterBlank || anchor >= 0) {
					anchor, anchorStart = column, start
				} else {
					anchor = -1
				}
				afterBlank = false
			}
		} else {
			mask(out[start:lineEnd])
			if closingFence(line, marker, fenceWidth, 3) {
				marker, fenceWidth = 0, 0
			}
		}

		if lineEnd == len(input) {
			break
		}
		start = lineEnd + 1
	}
	return out, nil
}

// loneCR reports a carriage return that is not the final byte of its line.
func loneCR(b []byte) bool {
	for i, c := range b {
		if c == '\r' && i+1 < len(b) && b[i+1] != '\n' {
			return true
		}
	}
	return false
}

// openingFence reports a fence opener indented at most limit columns.
func openingFence(line []byte, limit int) (byte, int, bool) {
	start, ok := fenceStart(line, limit)
	if !ok {
		return 0, 0, false
	}
	marker := line[start]
	end := start
	for end < len(line) && line[end] == marker {
		end++
	}
	if end-start < 3 {
		return 0, 0, false
	}
	if marker == '`' && bytes.IndexByte(line[end:], '`') >= 0 {
		return 0, 0, false
	}
	return marker, end - start, true
}

// listFenceEnd returns the end offset of the line that closes a list-indented
// fence, or -1 unless a closer with the opener's exact indentation follows and
// every line before it is blank or indented at least as far. A closer-like line
// at another indentation inside the item makes the pairing ambiguous.
func listFenceEnd(input []byte, openEnd int, marker byte, width, indent, container int) int {
	for start := openEnd + 1; start < len(input); {
		lineEnd := bytes.IndexByte(input[start:], '\n')
		if lineEnd < 0 {
			lineEnd = len(input)
		} else {
			lineEnd += start
		}
		contentEnd := lineEnd
		if contentEnd > start && input[contentEnd-1] == '\r' {
			contentEnd--
		}
		line := input[start:contentEnd]
		if !blank(line) {
			columns := indentColumns(line)
			if columns < indent {
				return -1
			}
			if closingFence(line, marker, width, container+3) {
				if columns != indent {
					return -1
				}
				return lineEnd
			}
		}
		start = lineEnd + 1
	}
	return -1
}

func closingFence(line []byte, marker byte, width, limit int) bool {
	start, ok := fenceStart(line, limit)
	if !ok || line[start] != marker {
		return false
	}
	end := start
	for end < len(line) && line[end] == marker {
		end++
	}
	if end-start < width {
		return false
	}
	for _, b := range line[end:] {
		if b != ' ' && b != '\t' {
			return false
		}
	}
	return true
}

func fenceStart(line []byte, limit int) (int, bool) {
	columns, start := leadingWidth(line)
	return start, columns <= limit && start < len(line) && (line[start] == '`' || line[start] == '~')
}

// topLevelItem returns the content column of a list item whose marker sits at
// column 0 and whose content is neither another list marker, a heading marker,
// a fence nor part of a thematic break.
func topLevelItem(line []byte) (int, bool) {
	if len(line) < 2 || isThematicBreak(line) {
		return 0, false
	}
	markerEnd := 1
	switch c := line[0]; {
	case c == '-' || c == '*' || c == '+':
	case c >= '1' && c <= '9' && line[1] == '.':
		markerEnd = 2
	default:
		return 0, false
	}
	if markerEnd >= len(line) || (line[markerEnd] != ' ' && line[markerEnd] != '\t') {
		return 0, false
	}
	column, next := markerEnd, markerEnd
	for next < len(line) && (line[next] == ' ' || line[next] == '\t') {
		if line[next] == '\t' {
			column += 4 - column%4
		} else {
			column++
		}
		next++
	}
	if next == len(line) || column-markerEnd > 4 || startsBlock(line[next:]) {
		return 0, false
	}
	return column, true
}

// startsBlock reports content that opens a list marker, heading or fence.
func startsBlock(content []byte) bool {
	i := 0
	switch c := content[0]; {
	case c == '-' || c == '*' || c == '+':
		i = 1
	case c >= '0' && c <= '9':
		for i < len(content) && i < 9 && content[i] >= '0' && content[i] <= '9' {
			i++
		}
		if i == len(content) || (content[i] != '.' && content[i] != ')') {
			return false
		}
		i++
	case c == '#':
		for i < len(content) && content[i] == '#' {
			i++
		}
		if i > 6 {
			return false
		}
	case c == '`' || c == '~':
		return len(content) >= 3 && content[1] == c && content[2] == c
	default:
		return false
	}
	return i == len(content) || content[i] == ' ' || content[i] == '\t'
}

// leadingWidth returns the column width and byte length of the leading spaces
// and tabs, with tabs advancing to the next multiple of four.
func leadingWidth(line []byte) (columns, size int) {
	for size < len(line) && (line[size] == ' ' || line[size] == '\t') {
		if line[size] == '\t' {
			columns += 4 - columns%4
		} else {
			columns++
		}
		size++
	}
	return columns, size
}

func blank(line []byte) bool {
	return len(bytes.Trim(line, " \t")) == 0
}

// indentColumns is the width of the leading whitespace with tabs advancing to
// the next multiple of four.
func indentColumns(line []byte) int {
	column := 0
	for _, b := range line {
		switch b {
		case ' ':
			column++
		case '\t':
			column += 4 - column%4
		default:
			return column
		}
	}
	return column
}

func isThematicBreak(line []byte) bool {
	trimmed := bytes.Trim(line, " \t")
	if len(trimmed) < 3 || (trimmed[0] != '-' && trimmed[0] != '*' && trimmed[0] != '_') {
		return false
	}
	count := 0
	for _, b := range trimmed {
		switch b {
		case trimmed[0]:
			count++
		case ' ', '\t':
		default:
			return false
		}
	}
	return count >= 3
}

func mask(data []byte) {
	for i, b := range data {
		if b != '\r' && b != '\n' {
			data[i] = ' '
		}
	}
}
