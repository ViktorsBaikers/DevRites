package notes

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
)

// Exit codes match the engine CLI conventions.
const (
	ExitOK      = 0
	ExitUsage   = 2
	ExitBlocked = 3
)

const noteUsage = `usage: devrites-engine note <subcommand> <slug>

Subcommands:
  add <slug> <subject> <quote> <title> [body]
                             Anchor a workspace note to verbatim code text
  list <slug>                List notes.md entries with their current grade
  check <slug> [--repair]    Regrade every anchor; --repair rewrites moved subjects
  rm <slug> <NOTE-id>        Remove a note

Grades: exact | moved | stale | ambiguous | lost. A non-exact note means the
rationale drifted from the code it anchors to; check seal enforces it.
Exit codes: 0 ok, 2 usage, 3 blocked`

// Run is the engine entrypoint for `note …`; root is the resolved DevRites
// root from the caller's single root resolution.
func Run(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, noteUsage)
		return ExitUsage
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "-h", "-help", "--help", "help":
		fmt.Fprintln(stdout, noteUsage)
		return ExitOK
	case "add":
		return cmdAdd(root, rest, stdout, stderr)
	case "list":
		return cmdList(root, rest, stdout, stderr)
	case "check":
		return cmdCheck(root, rest, stdout, stderr)
	case "rm":
		return cmdRm(root, rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "note: unknown subcommand %q\n\n%s\n", sub, noteUsage)
		return ExitUsage
	}
}

// oneLine flattens an argument to a single note line so a pasted newline
// cannot inject rows into the artifact.
func oneLine(s string) string {
	return strings.Join(strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == '\r' }), " ")
}

func workspacePaths(root, slug string) (work, notesPath, project string, err error) {
	work, err = devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return "", "", "", fmt.Errorf("feature %q: %w", slug, err)
	}
	return work, filepath.Join(work, NotesFile), filepath.Dir(root), nil
}

func loadDocument(notesPath string) (*Document, string, error) {
	// #nosec G304 -- workspace artifact path resolved from the operator root
	raw, err := os.ReadFile(notesPath)
	if os.IsNotExist(err) {
		return &Document{}, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return ParseNotes(string(raw)), string(raw), nil
}

func cmdAdd(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) < 4 || len(args) > 5 {
		fmt.Fprintln(stderr, "usage: devrites-engine note add <slug> <subject> <quote> <title> [body]")
		return ExitUsage
	}
	slug := args[0]
	subject := filepath.ToSlash(oneLine(args[1]))
	quote := oneLine(args[2])
	title := oneLine(args[3])
	body := ""
	if len(args) == 5 {
		body = oneLine(args[4])
	}
	if err := ValidateSubject(subject); err != nil {
		fmt.Fprintf(stderr, "note add: %v\n", err)
		return ExitUsage
	}
	if quote == "" || title == "" {
		fmt.Fprintln(stderr, "note add: <quote> and <title> must be non-blank")
		return ExitUsage
	}
	_, notesPath, project, err := workspacePaths(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "note add: %v\n", err)
		return ExitBlocked
	}
	// Fail closed: a note is only written while its anchor is exact. A quote
	// that does not resolve is a wrong note, not a note to fix later.
	subjectText, ok := fileText(filepath.Join(project, filepath.FromSlash(subject)))
	if !ok {
		fmt.Fprintf(stderr, "note add: subject %q is not a readable repository file\n", subject)
		return ExitBlocked
	}
	if !strings.Contains(subjectText, normalizeWS(quote)) {
		fmt.Fprintf(stderr, "note add: quote not found in %s; anchor before writing\n", subject)
		return ExitBlocked
	}
	var assigned string
	lockErr := state.WithFeatureLock(root, slug, func() error {
		doc, text, err := loadDocument(notesPath)
		if err != nil {
			return err
		}
		if len(doc.Errors) > 0 {
			return fmt.Errorf("notes.md is malformed; repair or remove it first: %s", strings.Join(doc.Errors, "; "))
		}
		assigned = doc.NextID()
		return state.AtomicWrite(notesPath, []byte(Render(text, doc.Newline(), assigned, title, subject, quote, body)), 0o644)
	})
	if lockErr != nil {
		fmt.Fprintf(stderr, "note add: %v\n", lockErr)
		return ExitBlocked
	}
	fmt.Fprintf(stdout, "added: %s\n", assigned)
	return ExitOK
}

func cmdList(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(stderr, "usage: devrites-engine note list <slug>")
		return ExitUsage
	}
	_, notesPath, project, err := workspacePaths(root, args[0])
	if err != nil {
		fmt.Fprintf(stderr, "note list: %v\n", err)
		return ExitBlocked
	}
	doc, _, err := loadDocument(notesPath)
	if err != nil {
		fmt.Fprintf(stderr, "note list: %v\n", err)
		return ExitBlocked
	}
	fmt.Fprintf(stdout, "feature: %s\n", args[0])
	for _, parseErr := range doc.Errors {
		fmt.Fprintf(stdout, "error: %s\n", parseErr)
	}
	for _, graded := range GradeAll(project, doc) {
		line := fmt.Sprintf("%s: %s | %s | %s", graded.ID, graded.Grade, graded.Subject, graded.Summary)
		if graded.Grade == GradeMoved {
			line += " -> " + graded.MovedTo
		}
		fmt.Fprintln(stdout, line)
	}
	if len(doc.Errors) > 0 {
		return ExitBlocked
	}
	return ExitOK
}

func cmdCheck(root string, args []string, stdout, stderr io.Writer) int {
	var slug string
	repair := false
	for _, arg := range args {
		if arg == "--repair" {
			repair = true
			continue
		}
		if strings.HasPrefix(arg, "-") || slug != "" {
			fmt.Fprintln(stderr, "usage: devrites-engine note check <slug> [--repair]")
			return ExitUsage
		}
		slug = arg
	}
	if slug == "" {
		fmt.Fprintln(stderr, "usage: devrites-engine note check <slug> [--repair]")
		return ExitUsage
	}
	_, notesPath, project, err := workspacePaths(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "note check: %v\n", err)
		return ExitBlocked
	}
	return reportCheck(root, slug, notesPath, project, repair, stdout, stderr)
}

func reportCheck(root, slug, notesPath, project string, repair bool, stdout, stderr io.Writer) int {
	doc, _, err := loadDocument(notesPath)
	if err != nil {
		fmt.Fprintf(stderr, "note check: %v\n", err)
		return ExitBlocked
	}
	fmt.Fprintf(stdout, "feature: %s\n", slug)
	fmt.Fprintf(stdout, "notes: %d\n", len(doc.Notes))
	for _, parseErr := range doc.Errors {
		fmt.Fprintf(stdout, "error: %s\n", parseErr)
	}
	if len(doc.Errors) > 0 {
		fmt.Fprintln(stdout, "result: malformed")
		return ExitBlocked
	}
	graded := GradeAll(project, doc)
	if repair {
		// Re-read under the feature lock so a concurrent add cannot be lost.
		lockErr := state.WithFeatureLock(root, slug, func() error {
			freshDoc, freshText, err := loadDocument(notesPath)
			if err != nil {
				return err
			}
			if len(freshDoc.Errors) > 0 {
				return fmt.Errorf("notes.md changed under repair; rerun `note check`")
			}
			_, newText := repairMoved(freshText, freshDoc, GradeAll(project, freshDoc))
			if newText == freshText {
				return nil
			}
			if err := state.AtomicWrite(notesPath, []byte(newText), 0o644); err != nil {
				return err
			}
			doc = ParseNotes(newText)
			graded = GradeAll(project, doc)
			return nil
		})
		if lockErr != nil {
			fmt.Fprintf(stderr, "note check: %v\n", lockErr)
			return ExitBlocked
		}
	}
	counts := map[Grade]int{}
	for _, g := range graded {
		counts[g.Grade]++
		line := fmt.Sprintf("note: %s %s", g.ID, g.Grade)
		switch g.Grade {
		case GradeMoved:
			line += " -> " + g.MovedTo + " (repair with `note check --repair`)"
		case GradeStale, GradeAmbiguous, GradeLost:
			line += " (re-anchor or `note rm`)"
		}
		fmt.Fprintln(stdout, line)
	}
	fmt.Fprintf(stdout, "exact: %d\nmoved: %d\nstale: %d\nambiguous: %d\nlost: %d\n",
		counts[GradeExact], counts[GradeMoved], counts[GradeStale], counts[GradeAmbiguous], counts[GradeLost])
	if counts[GradeMoved]+counts[GradeStale]+counts[GradeAmbiguous]+counts[GradeLost] > 0 {
		fmt.Fprintln(stdout, "result: drifted")
		return ExitBlocked
	}
	fmt.Fprintln(stdout, "result: exact")
	return ExitOK
}

// repairMoved rewrites each moved note's SUBJECT to the file its quote now
// lives in. Ambiguous, stale, and lost notes are left for human resolution.
func repairMoved(text string, doc *Document, graded []Graded) (int, string) {
	repaired := 0
	for _, g := range graded {
		if g.Grade != GradeMoved {
			continue
		}
		text = RepairSubject(text, doc.Newline(), g.Note, g.MovedTo)
		repaired++
	}
	return repaired, text
}

func cmdRm(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "usage: devrites-engine note rm <slug> <NOTE-id>")
		return ExitUsage
	}
	slug, id := args[0], args[1]
	_, notesPath, _, err := workspacePaths(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "note rm: %v\n", err)
		return ExitBlocked
	}
	lockErr := state.WithFeatureLock(root, slug, func() error {
		doc, text, err := loadDocument(notesPath)
		if err != nil {
			return err
		}
		var target *Note
		for _, note := range doc.Notes {
			if note.ID == id {
				target = note
			}
		}
		if target == nil {
			return fmt.Errorf("unknown note id %q", id)
		}
		return state.AtomicWrite(notesPath, []byte(RemoveEntry(text, doc.Newline(), target)), 0o644)
	})
	if lockErr != nil {
		fmt.Fprintf(stderr, "note rm: %v\n", lockErr)
		return ExitBlocked
	}
	fmt.Fprintf(stdout, "removed: %s\n", id)
	return ExitOK
}

// SealCheck regrades the workspace's anchored notes for the seal gate. An
// absent or empty notes.md passes; any non-exact note blocks with the repair
// route named. Output lines are stable and greppable.
func SealCheck(root, slug string, stdout io.Writer) bool {
	work, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return true // feature resolution is the gate's own finding, not ours
	}
	notesPath := filepath.Join(work, NotesFile)
	doc, _, err := loadDocument(notesPath)
	if err != nil {
		fmt.Fprintf(stdout, "note: unreadable notes.md: %v\n", err)
		return false
	}
	if len(doc.Notes) == 0 && len(doc.Errors) == 0 {
		return true
	}
	for _, parseErr := range doc.Errors {
		fmt.Fprintf(stdout, "note: malformed: %s\n", parseErr)
	}
	if len(doc.Errors) > 0 {
		return false
	}
	drifted := false
	for _, g := range GradeAll(filepath.Dir(root), doc) {
		if g.Grade == GradeExact {
			continue
		}
		drifted = true
		fmt.Fprintf(stdout, "note: %s %s (%s)\n", g.ID, g.Grade, g.Subject)
	}
	if drifted {
		fmt.Fprintf(stdout, "next: devrites-engine note check %s --repair  # re-anchor moved; re-anchor or rm the rest\n", slug)
	}
	return !drifted
}
