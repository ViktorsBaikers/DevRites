package toolpolicy

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type tokenKind uint8

const (
	wordToken tokenKind = iota
	redirectToken
	separatorToken
)

type shellToken struct {
	kind    tokenKind
	text    string
	dynamic bool
}

type shellSegment struct {
	index  int
	tokens []shellToken
}

type scannedShell struct {
	tokens   []shellToken
	segments []shellSegment
}

type scanFailure struct {
	dynamic bool
	message string
}

func (f *scanFailure) Error() string { return f.message }

// scanShell recognizes only the small POSIX command-list grammar the policy
// needs. It deliberately refuses compound commands, heredocs, and command
// substitution instead of pretending to be a general shell parser.
func scanShell(input string) (scannedShell, error) {
	var (
		tokens        []shellToken
		word          strings.Builder
		wordStarted   bool
		wordDynamic   bool
		wordProtected bool
		quote         byte
	)

	flushWord := func() {
		if !wordStarted {
			return
		}
		tokens = append(tokens, shellToken{
			kind:    wordToken,
			text:    word.String(),
			dynamic: wordDynamic,
		})
		word.Reset()
		wordStarted = false
		wordDynamic = false
		wordProtected = false
	}
	addSeparator := func(op string) {
		flushWord()
		tokens = append(tokens, shellToken{kind: separatorToken, text: op})
	}
	addRedirect := func(op string) {
		if wordStarted && !wordProtected && !wordDynamic && isDigits(word.String()) {
			op = word.String() + op
			word.Reset()
			wordStarted = false
			wordDynamic = false
			wordProtected = false
		} else {
			flushWord()
		}
		tokens = append(tokens, shellToken{kind: redirectToken, text: op})
	}

	for i := 0; i < len(input); i++ {
		c := input[i]

		switch quote {
		case '\'':
			if c == '\'' {
				quote = 0
				continue
			}
			word.WriteByte(c)
			continue
		case '"':
			switch c {
			case '"':
				quote = 0
			case '\\':
				if i+1 >= len(input) {
					return scannedShell{}, &scanFailure{message: "trailing backslash in double quote"}
				}
				next := input[i+1]
				if next == '\n' {
					i++
					continue
				}
				if strings.ContainsRune("$`\"\\", rune(next)) {
					word.WriteByte(next)
					i++
				} else {
					word.WriteByte(c)
				}
			case '`':
				return scannedShell{}, &scanFailure{dynamic: true, message: "backtick substitution is unsupported"}
			case '$':
				if i+1 < len(input) && input[i+1] == '(' {
					return scannedShell{}, &scanFailure{dynamic: true, message: "command substitution is unsupported"}
				}
				wordStarted = true
				wordDynamic = true
				word.WriteByte(c)
			default:
				word.WriteByte(c)
			}
			continue
		}

		switch c {
		case ' ', '\t', '\r':
			flushWord()
		case '\n':
			addSeparator(";")
		case '\'':
			wordStarted = true
			wordProtected = true
			quote = '\''
		case '"':
			wordStarted = true
			wordProtected = true
			quote = '"'
		case '\\':
			if i+1 >= len(input) {
				return scannedShell{}, &scanFailure{message: "trailing backslash"}
			}
			if input[i+1] == '\n' {
				i++
				continue
			}
			wordStarted = true
			wordProtected = true
			word.WriteByte(input[i+1])
			i++
		case '`':
			return scannedShell{}, &scanFailure{dynamic: true, message: "backtick substitution is unsupported"}
		case '$':
			if i+1 < len(input) && input[i+1] == '(' {
				return scannedShell{}, &scanFailure{dynamic: true, message: "command substitution is unsupported"}
			}
			wordStarted = true
			wordDynamic = true
			word.WriteByte(c)
		case '#':
			if wordStarted {
				word.WriteByte(c)
				continue
			}
			for i+1 < len(input) && input[i+1] != '\n' {
				i++
			}
		case ';':
			if i+1 < len(input) && input[i+1] == ';' {
				return scannedShell{}, &scanFailure{message: "compound command separator is unsupported"}
			}
			addSeparator(";")
		case '&':
			if i+1 < len(input) && input[i+1] == '>' {
				i++
				op := "&>"
				if i+1 < len(input) && input[i+1] == '>' {
					i++
					op = "&>>"
				}
				addRedirect(op)
				continue
			}
			if i+1 < len(input) && input[i+1] == '&' {
				i++
				addSeparator("&&")
			} else {
				addSeparator("&")
			}
		case '|':
			if i+1 < len(input) && input[i+1] == '&' {
				return scannedShell{}, &scanFailure{message: "pipeline stderr shorthand is unsupported"}
			}
			if i+1 < len(input) && input[i+1] == '|' {
				i++
				addSeparator("||")
			} else {
				addSeparator("|")
			}
		case '<', '>':
			if i+1 < len(input) && input[i+1] == '(' {
				return scannedShell{}, &scanFailure{dynamic: true, message: "process substitution is unsupported"}
			}
			op := string(c)
			if i+1 < len(input) {
				next := input[i+1]
				switch {
				case c == '<' && next == '<':
					return scannedShell{}, &scanFailure{dynamic: true, message: "heredoc and here-string input is unsupported"}
				case c == '>' && next == '>':
					i++
					op = ">>"
				case c == '>' && (next == '&' || next == '|'):
					i++
					op += string(next)
				case c == '<' && (next == '&' || next == '>'):
					i++
					op += string(next)
				}
			}
			addRedirect(op)
		case '(', ')':
			return scannedShell{}, &scanFailure{message: "compound shell syntax is unsupported"}
		default:
			wordStarted = true
			word.WriteByte(c)
		}
	}

	if quote != 0 {
		return scannedShell{}, &scanFailure{message: "unmatched quote"}
	}
	flushWord()

	segments, err := splitSegments(tokens)
	if err != nil {
		return scannedShell{}, err
	}
	return scannedShell{tokens: trimSeparators(tokens), segments: segments}, nil
}

func splitSegments(tokens []shellToken) ([]shellSegment, error) {
	var (
		segments []shellSegment
		current  []shellToken
	)
	flush := func() error {
		if len(current) == 0 {
			return &scanFailure{message: "empty command segment"}
		}
		if _, err := commandWords(current); err != nil {
			return err
		}
		segments = append(segments, shellSegment{index: len(segments), tokens: current})
		current = nil
		return nil
	}

	for _, tok := range tokens {
		if tok.kind != separatorToken {
			current = append(current, tok)
			continue
		}
		if len(current) == 0 {
			if tok.text == ";" {
				continue
			}
			return nil, &scanFailure{message: "empty command before " + tok.text}
		}
		if err := flush(); err != nil {
			return nil, err
		}
	}
	if len(current) > 0 {
		if err := flush(); err != nil {
			return nil, err
		}
	} else if len(tokens) > 0 {
		last := tokens[len(tokens)-1]
		if last.kind == separatorToken && (last.text == "&&" || last.text == "||" || last.text == "|") {
			return nil, &scanFailure{message: "dangling command operator " + last.text}
		}
	}
	return segments, nil
}

func commandWords(tokens []shellToken) ([]shellToken, error) {
	words := make([]shellToken, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		switch tok.kind {
		case wordToken:
			words = append(words, tok)
		case redirectToken:
			if i+1 >= len(tokens) || tokens[i+1].kind != wordToken {
				return nil, &scanFailure{message: "redirection " + tok.text + " has no target"}
			}
			i++
		default:
			return nil, &scanFailure{message: "unexpected command token"}
		}
	}
	return words, nil
}

func trimSeparators(tokens []shellToken) []shellToken {
	start, end := 0, len(tokens)
	for start < end && tokens[start].kind == separatorToken && tokens[start].text == ";" {
		start++
	}
	for end > start && tokens[end-1].kind == separatorToken && tokens[end-1].text == ";" {
		end--
	}
	return tokens[start:end]
}

func normalizeTokens(tokens []shellToken) string {
	var b strings.Builder
	b.WriteString(NormalizationVersion)
	for _, tok := range tokens {
		var kind byte
		switch tok.kind {
		case wordToken:
			if tok.dynamic {
				kind = 'd'
			} else {
				kind = 'w'
			}
		case redirectToken:
			kind = 'r'
		case separatorToken:
			kind = 'o'
		}
		b.WriteByte('|')
		b.WriteByte(kind)
		b.WriteString(strconv.Itoa(len(tok.text)))
		b.WriteByte(':')
		b.WriteString(tok.text)
	}
	return b.String()
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func isAssignment(s string) bool {
	name, _, ok := strings.Cut(s, "=")
	if !ok || name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func basename(s string) string {
	if i := strings.LastIndexByte(s, '/'); i >= 0 {
		return s[i+1:]
	}
	return s
}

func requireWord(words []shellToken, at int, what string) (shellToken, error) {
	if at >= len(words) {
		return shellToken{}, fmt.Errorf("%s requires a value", what)
	}
	return words[at], nil
}
