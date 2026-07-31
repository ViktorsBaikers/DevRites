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
			if nextMarker, width, ok := openingFence(line); ok {
				marker, fenceWidth = nextMarker, width
				mask(out[start:lineEnd])
			}
		} else {
			mask(out[start:lineEnd])
			if closingFence(line, marker, fenceWidth) {
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

func openingFence(line []byte) (byte, int, bool) {
	start, ok := fenceStart(line)
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

func closingFence(line []byte, marker byte, width int) bool {
	start, ok := fenceStart(line)
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

func fenceStart(line []byte) (int, bool) {
	start := 0
	for start < len(line) && line[start] == ' ' {
		start++
	}
	return start, start <= 3 && start < len(line) && (line[start] == '`' || line[start] == '~')
}

func mask(data []byte) {
	for i, b := range data {
		if b != '\r' && b != '\n' {
			data[i] = ' '
		}
	}
}
