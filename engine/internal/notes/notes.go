// Package notes implements anchored notes: short workspace observations bound
// to a verbatim quote from one repository file. A note is exact while its
// quote is still present in the subject file; edits that move or delete the
// anchored text regrade the note moved, stale, ambiguous, or lost, so
// rationale that drifted from the code it describes surfaces at the seal gate
// instead of reading as current.
package notes

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/markdowntext"
)

// NotesFile is the canonical anchored-notes artifact inside a feature
// workspace.
const NotesFile = "notes.md"

const maxNoteIDLen = 64

// Note is one parsed anchored note. Line and SubjectLine are 1-based rows in
// the source text so repair can rewrite the subject without reparsing prose.
type Note struct {
	ID          string
	Summary     string
	Subject     string
	Quote       string
	Body        string
	Line        int
	SubjectLine int
}

// Document is a parsed notes.md. Errors is fail-closed: any entry means the
// document cannot be graded reliably.
type Document struct {
	Notes   []*Note
	Errors  []string
	newline string
}

var (
	noteLineRE     = regexp.MustCompile(`^\s*- (NOTE-[^\s:]+): (.*)$`)
	attrLineRE     = regexp.MustCompile(`^(\s+)(SUBJECT|QUOTE|BODY):(.*)$`)
	unindentedAttr = regexp.MustCompile(`^(SUBJECT|QUOTE|BODY):`)
)

// ParseNotes strictly parses a notes.md document. Fenced code blocks are
// masked per CommonMark fence rules so examples never become notes.
func ParseNotes(text string) *Document {
	doc := &Document{}
	doc.newline = "\n"
	if strings.Contains(text, "\r\n") {
		doc.newline = "\r\n"
	}
	structural, err := markdowntext.Structural([]byte(text))
	if err != nil {
		doc.Errors = append(doc.Errors, "notes is not valid Markdown text: "+err.Error())
		return doc
	}
	lines := strings.Split(strings.ReplaceAll(string(structural), "\r\n", "\n"), "\n")

	var current *Note
	seen := map[string]bool{}
	flush := func() { current = nil }

	for i, line := range lines {
		lineno := i + 1
		if match := noteLineRE.FindStringSubmatch(line); match != nil {
			id := match[1]
			if len(id) > maxNoteIDLen {
				doc.Errors = append(doc.Errors, fmt.Sprintf("line %d: note id %q exceeds %d characters", lineno, id, maxNoteIDLen))
				flush()
				continue
			}
			if seen[id] {
				doc.Errors = append(doc.Errors, fmt.Sprintf("line %d: duplicate note id %q", lineno, id))
				flush()
				continue
			}
			seen[id] = true
			current = &Note{ID: id, Summary: strings.TrimSpace(match[2]), Line: lineno}
			doc.Notes = append(doc.Notes, current)
			continue
		}
		if match := attrLineRE.FindStringSubmatch(line); match != nil {
			value := strings.TrimSpace(match[3])
			if current == nil {
				doc.Errors = append(doc.Errors, fmt.Sprintf("line %d: %s attribute has no note", lineno, match[2]))
				continue
			}
			switch match[2] {
			case "SUBJECT":
				if current.Subject != "" {
					doc.Errors = append(doc.Errors, fmt.Sprintf("line %d: note %s repeats SUBJECT", lineno, current.ID))
					continue
				}
				current.Subject = value
				current.SubjectLine = lineno
			case "QUOTE":
				if current.Quote != "" {
					doc.Errors = append(doc.Errors, fmt.Sprintf("line %d: note %s repeats QUOTE", lineno, current.ID))
					continue
				}
				current.Quote = value
			case "BODY":
				if current.Body != "" {
					doc.Errors = append(doc.Errors, fmt.Sprintf("line %d: note %s repeats BODY", lineno, current.ID))
					continue
				}
				current.Body = value
			}
			continue
		}
		if unindentedAttr.MatchString(line) {
			doc.Errors = append(doc.Errors, fmt.Sprintf("line %d: attribute must be indented beneath its note: %q", lineno, strings.TrimSpace(line)))
			continue
		}
		if strings.TrimSpace(line) != "" {
			flush()
		}
	}

	for _, note := range doc.Notes {
		if note.Summary == "" {
			doc.Errors = append(doc.Errors, fmt.Sprintf("note %s has an empty summary", note.ID))
		}
		if note.Subject == "" {
			doc.Errors = append(doc.Errors, fmt.Sprintf("note %s is missing SUBJECT", note.ID))
		} else if err := ValidateSubject(note.Subject); err != nil {
			doc.Errors = append(doc.Errors, fmt.Sprintf("note %s: %v", note.ID, err))
		}
		if note.Quote == "" {
			doc.Errors = append(doc.Errors, fmt.Sprintf("note %s is missing QUOTE", note.ID))
		}
	}
	return doc
}

// ValidateSubject requires a repository-relative subject path with no
// traversal or absolute anchor.
func ValidateSubject(subject string) error {
	if subject == "" {
		return fmt.Errorf("SUBJECT is empty")
	}
	if strings.HasPrefix(subject, "/") || regexp.MustCompile(`^[A-Za-z]:[\\/]`).MatchString(subject) || strings.HasPrefix(subject, `\\`) {
		return fmt.Errorf("SUBJECT must be repository-relative, not absolute: %q", subject)
	}
	for _, seg := range strings.FieldsFunc(subject, func(r rune) bool { return r == '/' || r == '\\' }) {
		if seg == ".." {
			return fmt.Errorf("SUBJECT must not contain traversal: %q", subject)
		}
	}
	return nil
}

// Grade is the current anchor state of a note.
type Grade string

const (
	// GradeExact: the quote is still present in the subject file.
	GradeExact Grade = "exact"
	// GradeMoved: the quote is absent from the subject and present in exactly
	// one other repository file; the anchor is repairable.
	GradeMoved Grade = "moved"
	// GradeStale: the subject file exists but the quote is gone everywhere.
	GradeStale Grade = "stale"
	// GradeAmbiguous: the quote is absent from the subject and present in more
	// than one other file; re-anchoring needs human judgment.
	GradeAmbiguous Grade = "ambiguous"
	// GradeLost: the subject file is gone and the quote is found nowhere.
	GradeLost Grade = "lost"
)

// Graded pairs a note with its current grade and, for moved notes, the
// repository-relative path the quote now lives in.
type Graded struct {
	*Note
	Grade   Grade
	MovedTo string
}

// normalizeWS collapses whitespace so a quote matches across re-wrapping and
// indentation changes while staying verbatim in every other respect.
func normalizeWS(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// skippedDirs are never searched for relocated quotes: version control,
// dependencies, build output, and the workspace itself.
var skippedDirs = map[string]bool{
	".git":         true,
	".devrites":    true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"out":          true,
	"target":       true,
	"bin":          true,
	"coverage":     true,
}

const maxSearchFileBytes = 512 << 10

// fileText reads a candidate file as normalized text, skipping oversized and
// binary files so quotes never match inside archives or images.
func fileText(path string) (string, bool) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxSearchFileBytes {
		return "", false
	}
	// #nosec G304 -- repository scan path under the resolved project root
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	head := raw
	if len(head) > 512 {
		head = head[:512]
	}
	if strings.IndexByte(string(head), 0) >= 0 {
		return "", false
	}
	return normalizeWS(string(raw)), true
}

// quoteLocations returns every repository-relative file whose normalized text
// contains the normalized quote, excluding the subject itself.
func quoteLocations(project, subject, quote string) []string {
	var found []string
	_ = filepath.WalkDir(project, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != project && skippedDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(project, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == subject {
			return nil
		}
		if text, ok := fileText(path); ok && strings.Contains(text, quote) {
			found = append(found, rel)
			if len(found) > 1 {
				return fs.SkipAll // ambiguous already; stop early
			}
		}
		return nil
	})
	return found
}

// Grade resolves one note's current anchor state against the project tree.
// The repo-wide relocation scan runs only when the subject no longer carries
// the quote, so exact notes cost one file read.
func GradeNote(project string, note *Note) Graded {
	graded := Graded{Note: note}
	subjectPath := filepath.Join(project, filepath.FromSlash(note.Subject))
	subjectText, subjectOK := fileText(subjectPath)
	quote := normalizeWS(note.Quote)
	if subjectOK && strings.Contains(subjectText, quote) {
		graded.Grade = GradeExact
		return graded
	}
	found := quoteLocations(project, note.Subject, quote)
	switch {
	case len(found) == 1:
		graded.Grade = GradeMoved
		graded.MovedTo = found[0]
	case len(found) > 1:
		graded.Grade = GradeAmbiguous
	case subjectOK:
		graded.Grade = GradeStale
	default:
		graded.Grade = GradeLost
	}
	return graded
}

// GradeAll grades every note in document order.
func GradeAll(project string, doc *Document) []Graded {
	out := make([]Graded, 0, len(doc.Notes))
	for _, note := range doc.Notes {
		out = append(out, GradeNote(project, note))
	}
	return out
}

// Newline returns the document's original line-ending style for writeback.
func (d *Document) Newline() string {
	if d.newline == "" {
		return "\n"
	}
	return d.newline
}

// NextID returns the lowest unused NOTE-### id, so ids are stable and
// collision-free under the feature lock.
func (d *Document) NextID() string {
	used := map[string]bool{}
	max := 0
	for _, note := range d.Notes {
		used[note.ID] = true
		var n int
		if _, err := fmt.Sscanf(note.ID, "NOTE-%d", &n); err == nil && n > max {
			max = n
		}
	}
	for n := max + 1; ; n++ {
		id := fmt.Sprintf("NOTE-%03d", n)
		if !used[id] {
			return id
		}
	}
}

// Render appends a note entry to existing notes.md text, preserving the
// document's newline style. An absent or empty document gains its header.
func Render(text, newline, id, summary, subject, quote, body string) string {
	entry := "- " + id + ": " + summary + "\n" +
		"  SUBJECT: " + subject + "\n" +
		"  QUOTE: " + quote + "\n"
	if body != "" {
		entry += "  BODY: " + body + "\n"
	}
	if newline == "\r\n" {
		entry = strings.ReplaceAll(entry, "\n", "\r\n")
	}
	if strings.TrimSpace(text) == "" {
		text = "# Anchored notes" + newline + newline
	} else if !strings.HasSuffix(text, newline) {
		text += newline
	}
	return text + entry
}

// RemoveEntry deletes one note block (its `- NOTE-` row through its last
// contiguous attribute row) from the source text.
func RemoveEntry(text, newline string, note *Note) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	end := note.Line
	for end < len(lines) && attrLineRE.MatchString(lines[end]) {
		end++
	}
	kept := append([]string{}, lines[:note.Line-1]...)
	kept = append(kept, lines[end:]...)
	out := strings.Join(kept, "\n")
	if newline == "\r\n" {
		out = strings.ReplaceAll(out, "\n", "\r\n")
	}
	return out
}

// RepairSubject rewrites a moved note's SUBJECT line to the new
// repository-relative path, preserving indentation.
func RepairSubject(text, newline string, note *Note, movedTo string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	idx := note.SubjectLine - 1
	if idx < 0 || idx >= len(lines) {
		return text
	}
	match := attrLineRE.FindStringSubmatch(lines[idx])
	if match == nil || match[2] != "SUBJECT" {
		return text
	}
	lines[idx] = match[1] + "SUBJECT: " + movedTo
	out := strings.Join(lines, "\n")
	if newline == "\r\n" {
		out = strings.ReplaceAll(out, "\n", "\r\n")
	}
	return out
}
