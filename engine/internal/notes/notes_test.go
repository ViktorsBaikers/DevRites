package notes

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/gate"
)

func writeFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const sampleNotes = `# Anchored notes

- NOTE-001: pool cap rationale
  SUBJECT: src/pool.go
  QUOTE: maxConns = 4
  BODY: fairness under bursty saves, not memory
- NOTE-002: retry window
  SUBJECT: src/retry.go
  QUOTE: backoff = 250 * attempts
`

func TestParseNotesWellFormed(t *testing.T) {
	doc := ParseNotes(sampleNotes)
	if len(doc.Errors) != 0 {
		t.Fatalf("errors: %v", doc.Errors)
	}
	if len(doc.Notes) != 2 {
		t.Fatalf("notes: %d", len(doc.Notes))
	}
	n := doc.Notes[0]
	if n.ID != "NOTE-001" || n.Subject != "src/pool.go" || n.Quote != "maxConns = 4" || n.Body == "" {
		t.Fatalf("note: %+v", n)
	}
}

func TestParseNotesRejectsMalformed(t *testing.T) {
	for name, text := range map[string]string{
		"duplicate id":    "- NOTE-001: a\n  SUBJECT: f.go\n  QUOTE: q\n- NOTE-001: b\n  SUBJECT: g.go\n  QUOTE: r\n",
		"missing subject": "- NOTE-001: a\n  QUOTE: q\n",
		"missing quote":   "- NOTE-001: a\n  SUBJECT: f.go\n",
		"unindented attr": "- NOTE-001: a\nSUBJECT: f.go\n  QUOTE: q\n",
		"orphan attr":     "  SUBJECT: f.go\n",
		"traversal":       "- NOTE-001: a\n  SUBJECT: ../f.go\n  QUOTE: q\n",
		"absolute":        "- NOTE-001: a\n  SUBJECT: /etc/passwd\n  QUOTE: q\n",
		"empty summary":   "- NOTE-001: \n  SUBJECT: f.go\n  QUOTE: q\n",
		"repeated quote":  "- NOTE-001: a\n  SUBJECT: f.go\n  QUOTE: q\n  QUOTE: r\n",
	} {
		t.Run(name, func(t *testing.T) {
			if doc := ParseNotes(text); len(doc.Errors) == 0 {
				t.Fatalf("want errors for %q", text)
			}
		})
	}
}

func TestParseNotesMasksFencedBlocks(t *testing.T) {
	text := "# Notes\n\n```\n- NOTE-999: fenced fake\n  SUBJECT: x.go\n  QUOTE: y\n```\n\n" + sampleNotes
	doc := ParseNotes(text)
	if len(doc.Errors) != 0 || len(doc.Notes) != 2 {
		t.Fatalf("notes=%d errors=%v", len(doc.Notes), doc.Errors)
	}
}

func TestParseNotesMasksListIndentedFences(t *testing.T) {
	text := "# Notes\n\n- how to write a note\n\n    ```\n    - NOTE-999: fenced fake\n      SUBJECT: x.go\n      QUOTE: y\n    ```\n\n" + sampleNotes
	doc := ParseNotes(text)
	if len(doc.Errors) != 0 || len(doc.Notes) != 2 {
		t.Fatalf("notes=%d errors=%v", len(doc.Notes), doc.Errors)
	}
}

func TestOpenBlockingQuestionAfterIndentedCodeStaysVisible(t *testing.T) {
	for name, text := range map[string]string{
		"top-level indented code":         "# Q\n\n    - example\n    ```\n\n## Q-001\nstatus: open\ngate: blocking\n",
		"list indented code":              "- a\n\n      - not a list\n       ```\n\n## Q-001\nstatus: open\ngate: blocking\n",
		"unclosed list fence":             "- a\n\n    ```\n\n## Q-001\nstatus: open\ngate: blocking\n",
		"unclosed ordered fence":          "1. step\n\n     ```\n## Q-001\nstatus: open\ngate: blocking\n",
		"unclosed nested fence":           "- a\n  - b\n\n      ```\n\n## Q-001\nstatus: open\ngate: blocking\n",
		"thematic break":                  "* * *\n\n    ```\n## Q-001\nstatus: open\ngate: blocking\n",
		"ordered cannot interrupt":        "text\n2. foo\n\n    ```\n## Q-001\nstatus: open\ngate: blocking\n",
		"list-indented fence ends":        "- a\n\n    ```\n    ## Q-002\n    ```\n\n## Q-001\nstatus: open\ngate: blocking\n",
		"empty dash then list fence":      "-\n\n    ```\n\n## Q-001\nstatus: open\ngate: blocking\n",
		"setext dash then list fence":     "Intro\n-\n\n    ```\n\n## Q-001\nstatus: open\ngate: blocking\n",
		"empty star then list fence":      "Some text\n*\n\n    ```\n\n## Q-001\nstatus: open\ngate: blocking\n",
		"empty dash then table":           "-\n\n    ```\n  | ID | Status | Gate |\n  |---|---|---|\n  | Q-001 | open | blocking |\n",
		"unclosed list fence to EOF":      "- a\n\n    ```\n    text\n\n## Q-001\nstatus: open\ngate: blocking\n",
		"list fence dedents before close": "- a\n\n    ```\n    text\nback\n    ```\n## Q-001\nstatus: open\ngate: blocking\n",
		"tab-indented closer then table":  "- a\n\n    ```\n\t```\n  | ID | Status | Gate |\n  |---|---|---|\n  | Q-001 | open | blocking |\n    ```\n",
	} {
		t.Run(name, func(t *testing.T) {
			if got := gate.OpenBlockingQuestionGates([]byte(text)); len(got) != 1 || got[0] != "blocking" {
				t.Fatalf("OpenBlockingQuestionGates = %v, want [blocking]", got)
			}
		})
	}
}

func TestGradeExactMovedStaleAmbiguousLost(t *testing.T) {
	project := t.TempDir()
	writeFile(t, project, "src/pool.go", "package src\n\nconst maxConns = 4\n")
	writeFile(t, project, "src/retry.go", "package src\n\nvar backoff = 250 * attempts\n")

	doc := ParseNotes(sampleNotes)
	graded := GradeAll(project, doc)
	if graded[0].Grade != GradeExact || graded[1].Grade != GradeExact {
		t.Fatalf("grades: %v %v", graded[0].Grade, graded[1].Grade)
	}

	// moved: quote relocates to exactly one other file
	writeFile(t, project, "src/pool.go", "package src\n\nconst maxConns = 8\n")
	writeFile(t, project, "src/config/limits.go", "package config\nconst maxConns = 4\n")
	graded = GradeAll(project, doc)
	if graded[0].Grade != GradeMoved || graded[0].MovedTo != "src/config/limits.go" {
		t.Fatalf("moved: %+v", graded[0])
	}

	// ambiguous: quote in two other files
	writeFile(t, project, "src/config/other.go", "package config\nconst maxConns = 4\n")
	graded = GradeAll(project, doc)
	if graded[0].Grade != GradeAmbiguous {
		t.Fatalf("ambiguous: %+v", graded[0])
	}

	// stale: subject alive, quote gone everywhere
	writeFile(t, project, "src/config/other.go", "package config\nconst other = 1\n")
	writeFile(t, project, "src/config/limits.go", "package config\nconst other = 2\n")
	graded = GradeAll(project, doc)
	if graded[0].Grade != GradeStale {
		t.Fatalf("stale: %+v", graded[0])
	}

	// lost: subject file deleted entirely
	if err := os.Remove(filepath.Join(project, "src/pool.go")); err != nil {
		t.Fatal(err)
	}
	graded = GradeAll(project, doc)
	if graded[0].Grade != GradeLost {
		t.Fatalf("lost: %+v", graded[0])
	}
}

func TestGradeSkipsDependencyAndWorkspaceDirs(t *testing.T) {
	project := t.TempDir()
	writeFile(t, project, "src/a.go", "package src\nvar other = 1\n")
	writeFile(t, project, "node_modules/dep/index.js", "var depAnchor = 1\n")
	writeFile(t, project, ".devrites/work/f/notes.md", "depAnchor = 1\n")
	doc := ParseNotes("- NOTE-001: x\n  SUBJECT: src/missing.go\n  QUOTE: depAnchor = 1\n")
	graded := GradeAll(project, doc)
	if graded[0].Grade != GradeLost {
		t.Fatalf("dependency dirs must not count as relocation: %+v", graded[0])
	}
}

func TestGradeNormalizesWhitespace(t *testing.T) {
	project := t.TempDir()
	writeFile(t, project, "src/a.go", "package src\n\n\tvar   anchor\t=\t1\n")
	doc := ParseNotes("- NOTE-001: x\n  SUBJECT: src/a.go\n  QUOTE: var anchor = 1\n")
	if g := GradeAll(project, doc)[0]; g.Grade != GradeExact {
		t.Fatalf("whitespace-normalized match: %+v", g)
	}
}

func TestNextIDFillsLowest(t *testing.T) {
	doc := ParseNotes("- NOTE-001: a\n  SUBJECT: f\n  QUOTE: q\n- NOTE-003: b\n  SUBJECT: g\n  QUOTE: r\n")
	if got := doc.NextID(); got != "NOTE-004" {
		t.Fatalf("NextID = %q", got)
	}
}

func TestRenderAndRemoveRoundTrip(t *testing.T) {
	text := Render("", "\n", "NOTE-001", "first", "a.go", "q1", "body")
	text = Render(text, "\n", "NOTE-002", "second", "b.go", "q2", "")
	doc := ParseNotes(text)
	if len(doc.Errors) != 0 || len(doc.Notes) != 2 {
		t.Fatalf("render parse: %v / %d", doc.Errors, len(doc.Notes))
	}
	removed := RemoveEntry(text, "\n", doc.Notes[0])
	doc = ParseNotes(removed)
	if len(doc.Notes) != 1 || doc.Notes[0].ID != "NOTE-002" {
		t.Fatalf("after remove: %+v", doc.Notes)
	}
	if strings.Contains(removed, "NOTE-001") {
		t.Fatal("removed text still carries NOTE-001")
	}
}

func TestRepairSubjectRewritesOnlyThatLine(t *testing.T) {
	doc := ParseNotes(sampleNotes)
	note := doc.Notes[0]
	out := RepairSubject(sampleNotes, "\n", note, "src/new/pool.go")
	if !strings.Contains(out, "  SUBJECT: src/new/pool.go") {
		t.Fatalf("repair out: %q", out)
	}
	if !strings.Contains(out, "SUBJECT: src/retry.go") {
		t.Fatal("repair touched the other note")
	}
}

// anchorQuote is the verbatim quote every subject in the cases below still
// carries, so a decline to read the file is the only reason a note is not exact.
const anchorQuote = "maxConns = 4"

// writeOversizedQuoteSubject grows path past the read cap while keeping the
// anchor quote.
func writeOversizedQuoteSubject(t *testing.T, path string) {
	t.Helper()
	filler := strings.Repeat("padding line past the read cap\n", maxSearchFileBytes/28+1)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package src\n\nconst "+anchorQuote+"\n"+filler), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() <= int64(maxSearchFileBytes) {
		t.Fatalf("oversized subject is %d bytes, not past the %d-byte read cap", info.Size(), maxSearchFileBytes)
	}
}

// unreadableSubjects are the ways a subject this pass will not read as text can
// still contain the quote. Each degrades an existing, quote-bearing subject.
var unreadableSubjects = map[string]func(t *testing.T, path string){
	"permission denied": func(t *testing.T, path string) {
		if err := os.Chmod(path, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
		if _, err := os.ReadFile(path); err == nil {
			t.Skip("chmod 000 does not deny reads for this process; the permission decline is not exercised here")
		}
	},
	"nul prefixed": func(t *testing.T, path string) {
		if err := os.WriteFile(path, []byte("\x00package src\n\nconst "+anchorQuote+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	},
}

// subjectProject builds a project whose quote-bearing subject this pass will
// decline, next to one unrelated file carrying the same quote. That second
// file is the trap: only a wrongly-run relocation scan would name it.
func subjectProject(t *testing.T, degrade func(t *testing.T, path string)) string {
	t.Helper()
	project := t.TempDir()
	writeFile(t, project, "src/pool.go", "package src\n\nconst "+anchorQuote+"\n")
	writeFile(t, project, "other/unrelated.txt", "unrelated prose repeating "+anchorQuote+" verbatim\n")
	degrade(t, filepath.Join(project, "src", "pool.go"))
	return project
}

func anchorNotes() string {
	return Render("", "\n", "NOTE-001", "pool cap rationale", "src/pool.go", anchorQuote, "")
}

// A subject this pass declines to read is not evidence that the quote moved.
// The assertion is the negative, not the silence: no moved grade, no
// relocation target, and repairMoved returns the artifact byte-identical.
func TestUnreadableSubjectIsNotRelocatedOrRewritten(t *testing.T) {
	for name, degrade := range unreadableSubjects {
		t.Run(name, func(t *testing.T) {
			project := subjectProject(t, degrade)

			notesText := anchorNotes()
			doc := ParseNotes(notesText)
			if len(doc.Errors) != 0 {
				t.Fatalf("parse: %v", doc.Errors)
			}
			graded := GradeAll(project, doc)
			if graded[0].Grade == GradeExact {
				t.Fatalf("declined subject graded exact without being read: %+v", graded[0])
			}
			if graded[0].Grade == GradeMoved {
				t.Fatalf("declined subject graded moved -> %q", graded[0].MovedTo)
			}
			if graded[0].MovedTo != "" {
				t.Fatalf("declined subject carries a relocation target: %q", graded[0].MovedTo)
			}

			repaired, repairedText := repairMoved(notesText, doc, graded)
			if repaired != 0 {
				t.Fatalf("repairMoved rewrote %d note(s) it did not positively determine moved", repaired)
			}
			if repairedText != notesText {
				t.Fatalf("repairMoved changed the artifact:\n got %q\nwant %q", repairedText, notesText)
			}
		})
	}
}

// cmdAdd re-verifies the quote against the subject before writing. It must
// refuse a subject the pass will not read as text, so a note can never be
// created against one.
func TestAddRefusesSubjectItCannotReadAsText(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	work := filepath.Join(root, "work", "feat")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, project, "src/pool.go", "\x00package src\n\nconst "+anchorQuote+"\n")

	t.Setenv("DEVRITES_WORKSPACE", "")
	var stdout, stderr bytes.Buffer
	if code := cmdAdd(root, []string{"feat", "src/pool.go", anchorQuote, "pool cap rationale"},
		&stdout, &stderr); code != ExitBlocked {
		t.Fatalf("note add exit = %d, want %d for an unreadable subject:\n%s%s",
			code, ExitBlocked, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(work, NotesFile)); err == nil {
		t.Fatal("note add wrote notes.md for a subject it could not read")
	}
}

// A subject past the read cap that still carries the quote anchors exactly. The
// cap bounds relocation scanning, not whether a note can be anchored at all.
func TestOversizedSubjectWithQuoteGradesExactAndAnchors(t *testing.T) {
	project := t.TempDir()
	writeFile(t, project, "other/unrelated.txt", "unrelated prose repeating "+anchorQuote+" verbatim\n")
	writeOversizedQuoteSubject(t, filepath.Join(project, "src", "pool.go"))

	doc := ParseNotes(anchorNotes())
	graded := GradeAll(project, doc)
	if graded[0].Grade != GradeExact || graded[0].MovedTo != "" || graded[0].Detail != "" {
		t.Fatalf("oversized quote-bearing subject = %+v, want exact", graded[0])
	}

	root := filepath.Join(project, ".devrites")
	work := filepath.Join(root, "work", "feat")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_WORKSPACE", "")
	var stdout, stderr bytes.Buffer
	if code := cmdAdd(root, []string{"feat", "src/pool.go", anchorQuote, "pool cap rationale"},
		&stdout, &stderr); code != ExitOK {
		t.Fatalf("note add exit = %d, want %d for an oversized subject carrying the quote:\n%s%s",
			code, ExitOK, stdout.String(), stderr.String())
	}
	stdout.Reset()
	if code := cmdCheck(root, []string{"feat"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("note check exit = %d, want %d:\n%s%s", code, ExitOK, stdout.String(), stderr.String())
	}
}

// An oversized subject that has lost the quote is read to the end and graded on
// what it holds, so the relocation scan may then name the single other home.
func TestOversizedSubjectWithoutQuoteIsRegradedFromItsContent(t *testing.T) {
	project := t.TempDir()
	writeFile(t, project, "other/unrelated.txt", "unrelated prose repeating "+anchorQuote+" verbatim\n")
	subject := filepath.Join(project, "src", "pool.go")
	writeOversizedQuoteSubject(t, subject)
	if err := os.WriteFile(subject, bytes.ReplaceAll(mustRead(t, subject), []byte(anchorQuote), []byte("maxConns = 8")), 0o644); err != nil {
		t.Fatal(err)
	}
	graded := GradeAll(project, ParseNotes(anchorNotes()))
	if graded[0].Grade != GradeMoved || graded[0].MovedTo != "other/unrelated.txt" {
		t.Fatalf("oversized subject without the quote = %+v, want moved -> other/unrelated.txt", graded[0])
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The quote is matched after whitespace normalization, so it must be found when
// the raw whitespace around and inside it straddles any internal read boundary.
func TestSubjectQuoteFoundAcrossReadBoundaries(t *testing.T) {
	for _, pad := range []int{0, 1, 4095, 4096, 4097, 32764, 32766, 32767, 32768, 32769, 32770, 65520, 65530, 65533, 65534, 65535, 65536, 65537, 131070, 131072} {
		for _, sp := range []string{" ", "\n\t ", "\u00a0\u2003"} {
			project := t.TempDir()
			body := strings.Repeat("\n", pad) + "const maxConns" + sp + "=" + sp + "4" + sp + "tail\n"
			writeFile(t, project, "src/pool.go", body)
			if g := GradeNote(project, &Note{Subject: "src/pool.go", Quote: anchorQuote}); g.Grade != GradeExact {
				t.Fatalf("pad %d spacing %q graded %+v, want exact", pad, sp, g)
			}
			writeFile(t, project, "src/pool.go", strings.Repeat("\n", pad)+"const maxConns"+sp+"=")
			if g := GradeNote(project, &Note{Subject: "src/pool.go", Quote: anchorQuote}); g.Grade == GradeExact {
				t.Fatalf("pad %d spacing %q graded exact for a truncated quote", pad, sp)
			}
		}
	}
}

// The rewrite is the harm, so the CLI oracle is a byte comparison of notes.md
// across `note check --repair`, not a line-count or a "no false positive".
func TestCheckRepairLeavesNotesByteIdenticalWhenSubjectUnreadable(t *testing.T) {
	for name, degrade := range unreadableSubjects {
		t.Run(name, func(t *testing.T) {
			project := t.TempDir()
			subjectPath := filepath.Join(project, "src", "pool.go")
			writeFile(t, project, "src/pool.go", "package src\n\nconst "+anchorQuote+"\n")
			writeFile(t, project, "other/unrelated.txt", "unrelated prose repeating "+anchorQuote+" verbatim\n")
			root := filepath.Join(project, ".devrites")
			work := filepath.Join(root, "work", "feat")
			if err := os.MkdirAll(work, 0o755); err != nil {
				t.Fatal(err)
			}
			notesPath := filepath.Join(work, NotesFile)
			if err := os.WriteFile(notesPath, []byte(anchorNotes()), 0o644); err != nil {
				t.Fatal(err)
			}
			degrade(t, subjectPath)

			before, err := os.ReadFile(notesPath)
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("DEVRITES_WORKSPACE", "")
			var stdout, stderr bytes.Buffer
			code := cmdCheck(root, []string{"feat", "--repair"}, &stdout, &stderr)

			// Check the rewrite first: re-pointing the anchor, even to a
			// correct location, is itself the harm for a subject that was not read.
			after, err := os.ReadFile(notesPath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatalf("notes.md was rewritten in place:\nbefore %q\nafter  %q", before, after)
			}
			if code == ExitOK {
				t.Fatalf("note check --repair exited 0 for a subject it never read:\n%s%s", stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), "result: drifted") {
				t.Fatalf("want a drifted result, got:\n%s", stdout.String())
			}
			for _, line := range strings.Split(stdout.String(), "\n") {
				if strings.HasPrefix(line, "note: NOTE-001 moved") {
					t.Fatalf("reported a relocation the scan was never entitled to run: %q\n%s", line, stdout.String())
				}
			}
		})
	}
}

// A relocation scan that could not enter part of the tree has not seen every
// candidate, so a single hit among the readable files is not a unique location.
func TestPartialScanDoesNotGradeSingleHitMoved(t *testing.T) {
	project := t.TempDir()
	writeFile(t, project, "visible/a.txt", "prose repeating "+anchorQuote+" verbatim\n")
	writeFile(t, project, "hidden/b.txt", "prose repeating "+anchorQuote+" verbatim\n")
	hidden := filepath.Join(project, "hidden")
	if err := os.Chmod(hidden, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(hidden, 0o755) })
	if _, err := os.ReadDir(hidden); err == nil {
		t.Skip("chmod 000 does not deny directory reads for this process")
	}
	root := filepath.Join(project, ".devrites")
	work := filepath.Join(root, "work", "feat")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	notesPath := filepath.Join(work, NotesFile)
	if err := os.WriteFile(notesPath, []byte(anchorNotes()), 0o644); err != nil {
		t.Fatal(err)
	}

	doc := ParseNotes(anchorNotes())
	graded := GradeAll(project, doc)
	if graded[0].Grade != GradeAmbiguous {
		t.Fatalf("grade = %s (moved to %q), want %s for a partial scan", graded[0].Grade, graded[0].MovedTo, GradeAmbiguous)
	}

	before, err := os.ReadFile(notesPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_WORKSPACE", "")
	var stdout, stderr bytes.Buffer
	cmdCheck(root, []string{"feat", "--repair"}, &stdout, &stderr)
	after, err := os.ReadFile(notesPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("notes.md was rewritten from a partial scan:\nbefore %q\nafter  %q", before, after)
	}
}

// Several notes whose subjects are gone share one repository walk: each
// candidate file is read once however many notes are looking for a quote, and
// every note keeps the grade and location it would get on its own.
func TestGradeAllReadsEachCandidateOnce(t *testing.T) {
	project := t.TempDir()
	writeFile(t, project, "a.txt", "alpha quote one lives here\n")
	writeFile(t, project, "b.txt", "beta quote two lives here\n")
	writeFile(t, project, "c.txt", "beta quote two lives here too\n")
	writeFile(t, project, "d.txt", "nothing relevant\n")
	doc := ParseNotes(`- NOTE-001: one
  SUBJECT: gone/one.txt
  QUOTE: alpha quote one
- NOTE-002: two
  SUBJECT: gone/two.txt
  QUOTE: beta quote two
- NOTE-003: three
  SUBJECT: gone/three.txt
  QUOTE: absent from everywhere
`)
	reads := map[string]int{}
	orig := readCandidate
	readCandidate = func(path string) fileRead {
		reads[filepath.Base(path)]++
		return orig(path)
	}
	t.Cleanup(func() { readCandidate = orig })

	graded := GradeAll(project, doc)
	for name, n := range reads {
		if n > 1 {
			t.Errorf("%s read %d times, want at most once", name, n)
		}
	}
	if graded[0].Grade != GradeMoved || graded[0].MovedTo != "a.txt" {
		t.Errorf("note 1: %+v", graded[0])
	}
	if graded[1].Grade != GradeAmbiguous {
		t.Errorf("note 2: %+v", graded[1])
	}
	if graded[2].Grade != GradeLost {
		t.Errorf("note 3: %+v", graded[2])
	}
}
