// Package acceptance implements the machine-checked acceptance ledger: the
// gates.md contract that turns each required outcome into a decidable gate.
// A runnable gate pairs a vetted CHECK command with an EXPECT oracle and passes
// only on exit 0 plus a match; recorded evidence is bound to the exact oracle
// definition so edited commands silently keep stale passes. The package owns
// parsing, non-executing status reduction, linting, and execution; it never
// judges whether an English title matches its oracle.
package acceptance

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/markdowntext"
)

// GatesFile is the canonical acceptance-ledger artifact name inside a feature
// workspace.
const GatesFile = "gates.md"

const maxGateIDLen = 64

// Gate is one parsed acceptance gate. Runnable gates carry Check and Expect;
// manual gates carry neither.
type Gate struct {
	ID       string
	Title    string
	Checked  bool
	Check    string
	Expect   string
	Cwd      string
	Evidence string
	Line     int // 1-based line of the "- [ ]" row in the original text
}

// Document is a parsed ledger. Errors is fail-closed: any entry means the
// ledger cannot produce a completion certificate.
type Document struct {
	Gates     []*Gate
	Abandoned map[string]string // gate id -> reason
	Errors    []string
	Warnings  []string
	newline   string // "\r\n" when the source used CRLF
}

var (
	gateLineRE      = regexp.MustCompile(`^\s*- \[([ xX])\] ([^\s:]+): (.*)$`)
	attrLineRE      = regexp.MustCompile(`^(\s+)(CHECK|EXPECT|CWD|EVIDENCE):(.*)$`)
	unindentedAttr  = regexp.MustCompile(`^(CHECK|EXPECT|CWD|EVIDENCE):`)
	abandonLineRE   = regexp.MustCompile(`^ABANDON: (\S+)\s*(.*)$`)
	indentedAbandon = regexp.MustCompile(`^\s+ABANDON:`)
	expectSlashRE   = regexp.MustCompile(`^/(.*)/([ims]*)$`)
)

// Expectation is a compiled EXPECT oracle: a substring or a /regex/ with the
// optional i/m/s flags mapped onto Go's inline flag syntax.
type Expectation struct {
	Kind     string // "substring" or "regex"
	Literal  string
	Regex    *regexp.Regexp
	PathLike bool // regex whose pattern reads as a literal path (lint signal)
}

// ParseLedger strictly parses a gates.md document. Fenced code blocks are
// masked per CommonMark fence rules so documentation examples never become
// gates; masked text keeps byte offsets and line numbers aligned with the
// source for evidence writeback.
func ParseLedger(text string) *Document {
	doc := &Document{Abandoned: map[string]string{}}
	newline := "\n"
	if strings.Contains(text, "\r\n") {
		newline = "\r\n"
	}
	doc.newline = newline
	structural, err := markdowntext.Structural([]byte(text))
	if err != nil {
		doc.Errors = append(doc.Errors, "ledger is not valid Markdown text: "+err.Error())
		return doc
	}
	lines := strings.Split(strings.ReplaceAll(string(structural), "\r\n", "\n"), "\n")

	var current *Gate
	seen := map[string]bool{}
	flush := func() { current = nil }

	for i, line := range lines {
		lineno := i + 1
		if match := gateLineRE.FindStringSubmatch(line); match != nil {
			id := match[2]
			if len(id) > maxGateIDLen {
				doc.Errors = append(doc.Errors, fmt.Sprintf("line %d: gate id %q exceeds %d characters", lineno, id, maxGateIDLen))
				continue
			}
			if seen[id] {
				doc.Errors = append(doc.Errors, fmt.Sprintf("line %d: duplicate gate id %q", lineno, id))
				continue
			}
			seen[id] = true
			current = &Gate{ID: id, Title: strings.TrimSpace(match[3]), Checked: match[1] != " ", Line: lineno}
			doc.Gates = append(doc.Gates, current)
			continue
		}
		if match := attrLineRE.FindStringSubmatch(line); match != nil {
			value := strings.TrimSpace(match[3])
			if current == nil {
				doc.Errors = append(doc.Errors, fmt.Sprintf("line %d: %s attribute has no gate", lineno, match[2]))
				continue
			}
			switch match[2] {
			case "CHECK":
				current.Check = unquoteCode(value)
			case "EXPECT":
				current.Expect = value
			case "CWD":
				current.Cwd = value
			case "EVIDENCE":
				current.Evidence = value
			}
			continue
		}
		if unindentedAttr.MatchString(line) {
			doc.Errors = append(doc.Errors, fmt.Sprintf("line %d: attribute must be indented beneath its gate: %q", lineno, strings.TrimSpace(line)))
			continue
		}
		if match := abandonLineRE.FindStringSubmatch(line); match != nil {
			reason := strings.TrimSpace(match[2])
			if reason == "" {
				doc.Errors = append(doc.Errors, fmt.Sprintf("line %d: ABANDON %s needs a non-blank reason", lineno, match[1]))
				continue
			}
			doc.Abandoned[match[1]] = reason
			continue
		}
		if indentedAbandon.MatchString(line) {
			doc.Errors = append(doc.Errors, fmt.Sprintf("line %d: ABANDON: must start at column 1", lineno))
			continue
		}
		// Any non-blank, non-structural line ends gate attribute scope.
		if strings.TrimSpace(line) != "" {
			flush()
		}
	}

	if len(doc.Gates) == 0 && len(doc.Errors) == 0 {
		doc.Errors = append(doc.Errors, "ledger declares no gates")
	}
	for id := range doc.Abandoned {
		if !seen[id] {
			doc.Errors = append(doc.Errors, fmt.Sprintf("ABANDON names unknown gate id %q", id))
		}
	}
	for _, gate := range doc.Gates {
		switch {
		case gate.Check == "" && gate.Expect == "":
			// Manual gate: judged by human evidence.
		case gate.Check != "" && gate.Expect != "":
			if _, err := CompileExpect(gate.Expect); err != nil {
				doc.Errors = append(doc.Errors, fmt.Sprintf("gate %s: %v", gate.ID, err))
			}
		default:
			doc.Errors = append(doc.Errors, fmt.Sprintf("gate %s is a partial runnable gate: CHECK and EXPECT must both be present or both absent", gate.ID))
		}
		if gate.Cwd != "" {
			if err := validateLedgerCwd(gate.Cwd); err != nil {
				doc.Errors = append(doc.Errors, fmt.Sprintf("gate %s: %v", gate.ID, err))
			}
		}
		if gate.Check != "" && gate.Evidence == "" {
			doc.Warnings = append(doc.Warnings, fmt.Sprintf("gate %s has no EVIDENCE line; the runner inserts one", gate.ID))
		}
	}
	return doc
}

// unquoteCode unwraps one markdown code span: `cmd` becomes cmd. Writers copy
// commands from the backticked preflight table; approval and execution bind
// the command text, not its markdown decoration.
func unquoteCode(s string) string {
	if len(s) >= 2 && s[0] == '`' && s[len(s)-1] == '`' {
		return s[1 : len(s)-1]
	}
	return s
}

// validateLedgerCwd requires a repository-relative working directory with no
// traversal or absolute anchor.
func validateLedgerCwd(cwd string) error {
	if cwd == "" {
		return nil
	}
	if strings.HasPrefix(cwd, "/") || regexp.MustCompile(`^[A-Za-z]:[\\/]`).MatchString(cwd) || strings.HasPrefix(cwd, `\\`) {
		return fmt.Errorf("CWD must be repository-relative, not absolute: %q", cwd)
	}
	for _, seg := range strings.FieldsFunc(cwd, func(r rune) bool { return r == '/' || r == '\\' }) {
		if seg == ".." {
			return fmt.Errorf("CWD must not contain traversal: %q", cwd)
		}
	}
	return nil
}

// CompileExpect parses an EXPECT value into its oracle. A /pattern/flags form
// is a regular expression; every other value is a literal substring.
func CompileExpect(expect string) (Expectation, error) {
	if match := expectSlashRE.FindStringSubmatch(expect); match != nil {
		pattern, flags := match[1], match[2]
		prefix := ""
		for _, f := range flags {
			switch f {
			case 'i':
				prefix += "i"
			case 'm':
				prefix += "m"
			case 's':
				prefix += "s"
			default:
				return Expectation{}, fmt.Errorf("EXPECT regex has unsupported flag %q", string(f))
			}
		}
		compiled, err := regexp.Compile(prefixFlags(prefix, pattern))
		if err != nil {
			return Expectation{}, fmt.Errorf("EXPECT regex is invalid: %v", err)
		}
		return Expectation{
			Kind:     "regex",
			Literal:  expect,
			Regex:    compiled,
			PathLike: strings.Contains(pattern, "/"),
		}, nil
	}
	if expect == "" {
		return Expectation{}, fmt.Errorf("EXPECT is empty")
	}
	return Expectation{Kind: "substring", Literal: expect}, nil
}

func prefixFlags(flags, pattern string) string {
	if flags == "" {
		return pattern
	}
	return "(?" + flags + ")" + pattern
}

// Matches reports whether the canonical combined output satisfies the oracle.
func (e Expectation) Matches(output string) bool {
	if e.Kind == "regex" {
		return e.Regex.MatchString(output)
	}
	return strings.Contains(output, e.Literal)
}

// Newline returns the document's original line-ending style for writeback.
func (d *Document) Newline() string {
	if d.newline == "" {
		return "\n"
	}
	return d.newline
}
