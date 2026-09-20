package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
