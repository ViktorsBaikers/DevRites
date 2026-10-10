// Package notes implements anchored notes: short workspace observations bound
// to a verbatim quote from one repository file. A note is exact while its
// quote is still present in the subject file; edits that move or delete the
// anchored text regrade the note moved, stale, ambiguous, or lost, so
// rationale that drifted from the code it describes surfaces at the seal gate
// instead of reading as current.
package notes

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

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
	// GradeStale: the subject file exists but the quote is gone everywhere, or
	// the subject exists and this pass declined to read it as text. Either way
	// re-anchoring is a human decision.
	GradeStale Grade = "stale"
	// GradeAmbiguous: the quote is absent from the subject and present in more
	// than one other file; re-anchoring needs human judgment.
	GradeAmbiguous Grade = "ambiguous"
	// GradeLost: the subject file is gone and the quote is found nowhere.
	GradeLost Grade = "lost"
)

// Graded pairs a note with its current grade and, for moved notes, the
// repository-relative path the quote now lives in. Detail explains a grade the
// grader could not positively determine — currently the read bound that stopped
// it — so the report never presents an unresolved read as a resolved one.
type Graded struct {
	*Note
	Grade   Grade
	MovedTo string
	Detail  string
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

// readState is the outcome of reading one file as note text. The three states
// must stay distinct all the way to the caller: a decline to read is not
// evidence of absence, and only a proven absence entitles a grader to look
// for the quote somewhere else.
type readState int

const (
	// readOK: the file was read as normalized text.
	readOK readState = iota
	// readAbsent: there is no such file.
	readAbsent
	// readDeclined: the file is present but this pass did not read it as text.
	readDeclined
)

// fileRead carries a read outcome through to its caller. Reason names the bound
// that refused the read, so a report can say why a subject was not graded
// instead of implying its quote had been deleted.
type fileRead struct {
	text   string
	state  readState
	reason string
	// found is set only by fileHasQuote: the normalized quote occurs in a
	// file that was read to its end.
	found bool
}

// readable reports whether this pass obtained the file's text.
func (f fileRead) readable() bool { return f.state == readOK }

// statText reports the file's info when it is a regular file; otherwise it
// returns the absent or declined outcome and false.
func statText(path string) (os.FileInfo, fileRead, bool) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, fileRead{state: readAbsent}, false
	}
	if err != nil {
		return nil, fileRead{state: readDeclined, reason: "stat failed: " + err.Error()}, false
	}
	if !info.Mode().IsRegular() {
		return nil, fileRead{state: readDeclined, reason: "not a regular file"}, false
	}
	return info, fileRead{}, true
}

// fileText reads a candidate file as normalized text, skipping oversized and
// binary files so quotes never match inside archives or images. Every decline
// keeps its own identity: only a missing file reports as absent. A size, mode or
// permission bound can justify refusing to read a file; none of them is evidence
// about where its quote went, so none of them may stand in for absence.
func fileText(path string) fileRead {
	info, declined, ok := statText(path)
	if !ok {
		return declined
	}
	if info.Size() > maxSearchFileBytes {
		return fileRead{state: readDeclined, reason: fmt.Sprintf("%d bytes is over the %d-byte read cap", info.Size(), maxSearchFileBytes)}
	}
	// #nosec G304 -- repository scan path under the resolved project root
	raw, err := os.ReadFile(path)
	if err != nil {
		return fileRead{state: readDeclined, reason: "read failed: " + err.Error()}
	}
	head := raw
	if len(head) > 512 {
		head = head[:512]
	}
	if strings.IndexByte(string(head), 0) >= 0 {
		return fileRead{state: readDeclined, reason: "NUL byte in the first 512 bytes"}
	}
	return fileRead{text: normalizeWS(string(raw)), state: readOK}
}

// fileHasQuote reports whether the normalized quote occurs in the file's
// normalized text, with no size limit. It streams: memory stays bounded by the
// chunk size plus the quote length, because only the last len(quote)-1
// normalized bytes carry over between chunks, so a match that straddles a
// chunk boundary is still found. The size cap exists to bound the repository-wide
// relocation scan, not to decide whether the subject itself can be anchored.
func fileHasQuote(path, quote string) fileRead {
	if _, declined, ok := statText(path); !ok {
		return declined
	}
	// #nosec G304 -- subject path under the resolved project root
	f, err := os.Open(path)
	if err != nil {
		return fileRead{state: readDeclined, reason: "read failed: " + err.Error()}
	}
	defer func() { _ = f.Close() }()
	br := bufio.NewReaderSize(f, 64<<10)
	head, err := br.Peek(512)
	if err != nil && err != io.EOF && err != bufio.ErrBufferFull {
		return fileRead{state: readDeclined, reason: "read failed: " + err.Error()}
	}
	if bytes.IndexByte(head, 0) >= 0 {
		return fileRead{state: readDeclined, reason: "NUL byte in the first 512 bytes"}
	}
	q := []byte(quote)
	var (
		chunk      = make([]byte, 32<<10)
		window     []byte // normalized tail carried between chunks, then the new chunk
		carry      []byte // incomplete trailing UTF-8 sequence
		wroteWord  bool
		pendingGap bool
	)
	for {
		n, rerr := br.Read(chunk)
		if rerr != nil && rerr != io.EOF {
			return fileRead{state: readDeclined, reason: "read failed: " + rerr.Error()}
		}
		eof := rerr == io.EOF
		raw := append(carry, chunk[:n]...)
		carry = nil
		for i := 0; i < len(raw); {
			if !eof && !utf8.FullRune(raw[i:]) {
				carry = append([]byte(nil), raw[i:]...)
				break
			}
			r, size := utf8.DecodeRune(raw[i:])
			if unicode.IsSpace(r) {
				pendingGap = true
			} else {
				if pendingGap && wroteWord {
					window = append(window, ' ')
				}
				pendingGap = false
				wroteWord = true
				window = append(window, raw[i:i+size]...)
			}
			i += size
		}
		if bytes.Contains(window, q) {
			return fileRead{state: readOK, found: true}
		}
		if keep := len(q) - 1; len(window) > keep {
			window = append(window[:0], window[len(window)-keep:]...)
		}
		if eof {
			return fileRead{state: readOK}
		}
	}
}

// relocation is one note whose subject is gone and whose quote the shared
// repository walk is looking for.
type relocation struct {
	note        *Note
	quote       string
	subjectRead fileRead
	found       []string
	incomplete  bool
	done        bool
}

// readCandidate reads one walked file; a variable so tests can count reads.
var readCandidate = fileText

// locateQuotes walks the project once, reads each file once, and records for
// every pending note the repository-relative files (excluding its own subject)
// whose normalized text contains its quote. A note's incomplete flag is set when
// the walk or a file read failed while it was still scanning, so it may have
// missed a candidate; size and binary declines are not failures. A note stops
// scanning once a second hit makes it ambiguous.
func locateQuotes(project string, pending []*relocation) {
	active := len(pending)
	_ = filepath.WalkDir(project, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			for _, r := range pending {
				if !r.done {
					r.incomplete = true
				}
			}
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
		var read fileRead
		readDone := false
		for _, r := range pending {
			if r.done || rel == r.note.Subject {
				continue
			}
			if !readDone {
				read, readDone = readCandidate(path), true
			}
			if read.state == readDeclined && (strings.HasPrefix(read.reason, "stat failed") || strings.HasPrefix(read.reason, "read failed")) {
				r.incomplete = true
			}
			if read.readable() && strings.Contains(read.text, r.quote) {
				r.found = append(r.found, rel)
				if len(r.found) > 1 {
					r.done = true // ambiguous already; stop scanning for it
					active--
				}
			}
		}
		if active == 0 {
			return fs.SkipAll
		}
		return nil
	})
}

// gradeSubject grades one note against its subject file alone. The returned
// relocation is non-nil only when the subject is positively determined gone,
// so exact notes cost one file read. A subject this pass declines to read is
// never scanned: the scan is the only evidence that could license re-pointing
// the anchor, and a declined read is no evidence at all.
func gradeSubject(project string, note *Note) (Graded, *relocation) {
	graded := Graded{Note: note}
	subjectPath := filepath.Join(project, filepath.FromSlash(note.Subject))
	quote := normalizeWS(note.Quote)
	subjectRead := fileHasQuote(subjectPath, quote)
	if subjectRead.found {
		graded.Grade = GradeExact
		return graded, nil
	}
	if subjectRead.state == readDeclined {
		// Present but unreadable as text. Grade stale so a human re-anchors,
		// and carry the reason so the grade is not read as a deleted quote.
		graded.Grade = GradeStale
		graded.Detail = subjectRead.reason
		return graded, nil
	}
	return graded, &relocation{note: note, quote: quote, subjectRead: subjectRead}
}

// settle turns a finished scan into the note's grade.
func (r *relocation) settle(graded *Graded) {
	switch {
	case len(r.found) == 1 && r.incomplete:
		// Part of the tree went unread, so this hit may not be the only one.
		graded.Grade = GradeAmbiguous
	case len(r.found) == 1:
		graded.Grade = GradeMoved
		graded.MovedTo = r.found[0]
	case len(r.found) > 1:
		graded.Grade = GradeAmbiguous
	case r.subjectRead.readable():
		graded.Grade = GradeStale
	default:
		graded.Grade = GradeLost
	}
}

// GradeNote resolves one note's current anchor state against the project tree.
func GradeNote(project string, note *Note) Graded {
	doc := &Document{Notes: []*Note{note}}
	return GradeAll(project, doc)[0]
}

// GradeAll grades every note in document order. Subjects are graded first; the
// repository-wide relocation scan then runs once for all notes that need it.
func GradeAll(project string, doc *Document) []Graded {
	out := make([]Graded, len(doc.Notes))
	var pending []*relocation
	at := map[*relocation]int{}
	for i, note := range doc.Notes {
		g, r := gradeSubject(project, note)
		out[i] = g
		if r != nil {
			pending = append(pending, r)
			at[r] = i
		}
	}
	if len(pending) > 0 {
		locateQuotes(project, pending)
		for _, r := range pending {
			r.settle(&out[at[r]])
		}
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
