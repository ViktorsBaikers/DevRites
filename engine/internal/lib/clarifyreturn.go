package lib

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/workflow"
)

// ClarifyReturn persists and restores the later-phase cursor when an existing
// workspace is routed back through clarification.
//
//	clarify-return enter [slug]
//	clarify-return restore [slug]
func ClarifyReturn(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 || len(args) > 2 {
		fmt.Fprintln(stderr, "usage: devrites-engine clarify-return <enter|restore> [slug]")
		return 2
	}
	sub := strings.ToLower(strings.TrimSpace(args[0]))
	slug := slugOrActive(root, args[1:])
	if slug == "" {
		fmt.Fprintln(stderr, "clarify-return: no active workspace")
		return 2
	}
	path := featureFile(root, slug, "state.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "clarify-return: read state.md: %v\n", err)
		return 2
	}
	lines := splitLinesNoTrailing(raw)

	switch sub {
	case "enter":
		return changeClarifyReturn(path, lines, state.ClarifyEnter, stdout, stderr)
	case "restore":
		if err := validateDecisionCoverage(root, slug); err != nil {
			fmt.Fprintf(stderr, "clarify-return: decision coverage is not CLEAR/fresh: %v\n", err)
			return 3
		}
		return changeClarifyReturn(path, lines, state.ClarifyRestore, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "clarify-return: unknown subcommand %q\n", sub)
		return 2
	}
}

func changeClarifyReturn(path string, lines []string, intent state.ClarifyIntent, stdout, stderr io.Writer) int {
	current, err := state.ParseClarifyCursor(lines)
	if err != nil {
		fmt.Fprintf(stderr, "clarify-return: %v\n", err)
		return 2
	}
	resumePhase := current.Phase
	if intent == state.ClarifyRestore && current.HasReturn {
		resumePhase = current.ReturnPhase
	}
	next, err := state.NextClarifyCursor(intent, state.ClarifyTransitionInput{
		Cursor:                  current,
		ClarifyNextAction:       workflow.ForVerb("clarify").Both(),
		DefaultReturnNextAction: workflow.ForVerb(state.ResumeVerb(resumePhase)).Both(),
	})
	if err != nil {
		fmt.Fprintf(stderr, "clarify-return: %v\n", err)
		return 2
	}
	if next == current {
		if intent == state.ClarifyEnter {
			fmt.Fprintln(stdout, "clarify-return: already in clarify; preserved existing return cursor")
		} else {
			fmt.Fprintln(stdout, "clarify-return: no later-phase return cursor")
		}
		return 0
	}
	lines = state.ApplyClarifyCursor(lines, next)
	if err := fsutil.WriteFileAtomic(path, joinRecords(lines), 0o644); err != nil {
		fmt.Fprintf(stderr, "clarify-return: update state.md: %v\n", err)
		return 1
	}
	if intent == state.ClarifyEnter {
		fmt.Fprintf(stdout, "clarify-return: saved %s and entered clarify\n", current.Phase)
	} else {
		fmt.Fprintf(stdout, "clarify-return: restored %s\n", next.Phase)
	}
	return 0
}
