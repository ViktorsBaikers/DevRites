package lib

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/workflow"
)

// Resolve answers or drops an open question and keeps questions.md and state.md
// consistent by updating the question block. When state.md is waiting for that
// question, it clears the "Awaiting human" block and restores the running status.
// The workspace is <root>/work/<slug>.
//
//	0  resolved         2  no active workspace      3  qid not found
//	4  qid not open      5  bad arguments
func Resolve(root string, args []string, stdout, stderr io.Writer) int {
	slug := activeSlug(root)
	if slug == "" {
		return fail(stderr, "No active workspace. Run "+workflow.ForVerb("spec").Both()+" <feature> first.", 2)
	}
	work := featureDir(root, slug)
	qfile := filepath.Join(work, "questions.md")
	sfile := filepath.Join(work, "state.md")
	if !isFile(qfile) {
		return fail(stderr, "questions.md missing at "+qfile, 2)
	}
	if !isFile(sfile) {
		return fail(stderr, "state.md missing at "+sfile, 2)
	}
	if err := state.RequireWorkspaceSchema(root, slug); err != nil {
		return fail(stderr, err.Error(), 3)
	}

	const usage = `Usage: devrites-engine state resolve [--human] <qid> "<answer>"  |  state resolve [--human] --drop <qid> ["<reason>"]  |  state resolve [--human] --batch <file>`
	human := slices.Contains(args, "--human")
	args = slices.DeleteFunc(slices.Clone(args), func(arg string) bool { return arg == "--human" })
	var mode, qid, payload string
	maxArgs := 2
	switch first := argAt(args, 0); first {
	case "--drop":
		mode = "drop"
		qid = argAt(args, 1)
		payload = argAt(args, 2)
		maxArgs = 3
		if payload == "" {
			payload = "dropped"
		}
	case "--batch":
		mode = "batch"
		payload = argAt(args, 1)
		if !isFile(payload) {
			return fail(stderr, "Batch file not found: "+payload, 5)
		}
	case "":
		return fail(stderr, usage, 5)
	default:
		mode = "answer"
		qid = first
		payload = argAt(args, 1)
		if strings.TrimSpace(payload) == "" {
			return fail(stderr, "Answer text required for "+qid, 5)
		}
	}
	// An unknown flag or a stray argument must never be recorded as a qid or
	// an answer.
	if len(args) > maxArgs || strings.HasPrefix(qid, "-") || mode != "batch" && strings.HasPrefix(payload, "--") {
		return fail(stderr, usage, 5)
	}

	code := 0
	if err := state.WithFeatureLock(root, slug, func() error {
		code = resolveMutation(slug, mode, qid, payload, qfile, sfile, human, stdout, stderr)
		return nil
	}); err != nil {
		return fail(stderr, err.Error(), 1)
	}
	return code
}

func resolveMutation(slug, mode, qid, payload, qfile, sfile string, human bool, stdout, stderr io.Writer) int {
	type resolution struct{ qid, status, text string }
	var entries []resolution
	switch mode {
	case "answer":
		entries = append(entries, resolution{qid, "answered", payload})
	case "drop":
		entries = append(entries, resolution{qid, "dropped", payload})
	case "batch":
		// #nosec G304 -- operator-passed --batch file argument
		data, err := os.ReadFile(payload)
		if err != nil {
			return fail(stderr, "Batch file not found: "+payload, 5)
		}
		for _, line := range batchLines(data) {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			status := "answered"
			if rest, ok := strings.CutPrefix(line, "--drop "); ok {
				status, line = "dropped", rest
			}
			bid, text, ok := strings.Cut(line, ":")
			bid = strings.TrimSpace(bid)
			if !ok || bid == "" || strings.HasPrefix(bid, "-") {
				return fail(stderr, "Malformed batch line (want `<qid>: <answer>` or `--drop <qid>: <reason>`): "+line, 5)
			}
			if status == "answered" && strings.TrimSpace(text) == "" {
				return fail(stderr, "Answer text required for "+bid, 5)
			}
			entries = append(entries, resolution{bid, status, strings.TrimPrefix(text, " ")})
		}
	}
	if len(entries) == 0 {
		return 0
	}
	// #nosec G304 -- questions.md path inside the feature workspace
	qdata, err := os.ReadFile(qfile)
	if err != nil {
		return fail(stderr, "questions.md missing at "+qfile, 2)
	}
	// #nosec G304 -- state.md path inside the feature workspace
	sdata, err := os.ReadFile(sfile)
	if err != nil {
		return fail(stderr, fmt.Sprintf("read state %s: %v", sfile, err), 1)
	}
	lines := splitLinesNoTrailing(qdata)
	nextState := sdata
	// Prepare both artifacts for every entry before publishing either file.
	for _, entry := range entries {
		var code int
		if lines, code = applyResolution(lines, entry.qid, entry.status, entry.text, human, stderr); code != 0 {
			return code
		}
		if nextState, err = clearAwaiting(nextState, entry.qid, slug); err != nil {
			return fail(stderr, err.Error(), 1)
		}
	}
	if err := fsutil.WriteFileAtomic(qfile, joinRecords(lines), 0o644); err != nil {
		return fail(stderr, err.Error(), 1)
	}
	if !bytes.Equal(nextState, sdata) {
		if err := fsutil.WriteFileAtomic(sfile, nextState, 0o644); err != nil {
			return fail(stderr, fmt.Sprintf("update state %s: %v", sfile, err), 1)
		}
	}
	switch mode {
	case "answer":
		fmt.Fprintf(stdout, "Resolved: %s\n", qid)
		fmt.Fprintf(stdout, "Status:   answered\n")
		fmt.Fprintf(stdout, "Workspace: questions.md + state.md updated.\n")
	case "drop":
		fmt.Fprintf(stdout, "Dropped:  %s\n", qid)
		fmt.Fprintf(stdout, "Reason:   %s\n", payload)
		fmt.Fprintf(stdout, "Workspace: questions.md + state.md updated.\n")
	case "batch":
		for _, entry := range entries {
			if entry.status == "dropped" {
				fmt.Fprintf(stdout, "Dropped:  %s: %s\n", entry.qid, entry.text)
			} else {
				fmt.Fprintf(stdout, "Resolved: %s\n", entry.qid)
			}
		}
	}
	return 0
}

// fail writes a message to stderr and returns the exit code.
func fail(stderr io.Writer, msg string, code int) int {
	fmt.Fprintln(stderr, msg)
	return code
}

// oneLine flattens an operator-supplied answer to a single line so it cannot
// corrupt the questions.md field structure (answer: is a one-line field).
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// batchLines returns the lines of a batch file. A final line without a
// terminating newline is kept; only the empty element after a trailing newline
// is dropped.
func batchLines(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// clockNow is the resolve command's single wall-clock read. DEVRITES_NOW accepts
// an RFC-3339 timestamp or YYYY-MM-DD date so tests can keep date-derived
// question IDs stable. See ADR-0006.
func clockNow() time.Time {
	if s := os.Getenv("DEVRITES_NOW"); s != "" {
		for _, layout := range []string{time.RFC3339, "2006-01-02"} {
			if t, err := time.Parse(layout, s); err == nil {
				return t
			}
		}
	}
	return time.Now()
}

// nowUTC returns the current time as an ISO-8601 UTC timestamp.
func nowUTC() string { return clockNow().UTC().Format("2006-01-02T15:04:05Z") }

// applyResolution rewrites the <qid> block in the questions.md lines in memory.
// Returns 0, or 3 (qid not found) / 4 (qid not open), writing the message on
// failure.
func applyResolution(lines []string, qid, status, answer string, human bool, stderr io.Writer) ([]string, int) {
	// Locate blocks on the fence-masked text, as the readiness gate does, so a
	// fenced example is never rewritten and never shadows the real question.
	var visible []string
	if len(lines) > 0 {
		masked, err := markdowntext.Structural([]byte(strings.Join(lines, "\n")))
		if err != nil {
			return nil, fail(stderr, "questions.md is not readable Markdown: "+err.Error(), 1)
		}
		visible = strings.Split(string(masked), "\n")
	}
	target := regexp.MustCompile(`^## ` + regexp.QuoteMeta(qid) + `([[:space:]]|$)`)
	updated, found, notOpen := rewriteQuestionFields(lines, visible, target, status, oneLine(answer), nowUTC(), human)
	if !found {
		return nil, fail(stderr, "qid not found: "+qid, 3)
	}
	if notOpen {
		return nil, fail(stderr, "qid not open (already answered/dropped): "+qid, 4)
	}
	return updated, 0
}

var (
	qHeaderRe    = regexp.MustCompile(`(?i)^## q-`)
	statusLineRe = regexp.MustCompile(`(?i)^status:`)
	statusStrip  = regexp.MustCompile(`(?i)^status:[[:space:]]*`)
	answeredAtRe = regexp.MustCompile(`^answered_at:`)
	answerRe     = regexp.MustCompile(`^answer:`)
	answeredByRe = regexp.MustCompile(`(?i)^[[:space:]]*answered_by:`)
)

// rewriteQuestionFields walks questions.md and, inside the target question's
// block, sets status and rewrites any answered_at/answer lines already present. It
// reports whether the block was found and whether it was already not open (an
// attempt to re-answer). A target block with no status line at all gets a full set
// of closing fields appended. Structure is matched on visible (lines with
// fenced blocks masked, same length); unmatched lines are copied from lines.
func rewriteQuestionFields(lines, visible []string, target *regexp.Regexp, status, answer, ts string, human bool) (out []string, found, notOpen bool) {
	inQ := false
	statusSeen, answeredAtSeen, answerSeen, answeredBySeen := false, false, false, false
	closeBlock := func() {
		if !inQ {
			return
		}
		if !statusSeen {
			out = append(out, "status: "+status)
		}
		if !answeredAtSeen {
			out = append(out, "answered_at: "+ts)
		}
		if !answerSeen {
			out = append(out, "answer: "+answer)
		}
		if human && !answeredBySeen {
			out = append(out, "answered_by: human")
		}
	}
	for i, original := range lines {
		line := visible[i]
		switch {
		case qHeaderRe.MatchString(line):
			closeBlock()
			inQ = target.MatchString(line)
			statusSeen, answeredAtSeen, answerSeen, answeredBySeen = false, false, false, false
			if inQ {
				found = true
			}
			out = append(out, original)
		case inQ && hdrSpaceRe.MatchString(line):
			// Any level-two heading ends the question block, as the gate reads it.
			closeBlock()
			inQ = false
			out = append(out, original)
		case inQ && statusLineRe.MatchString(line):
			statusSeen = true
			cur := statusStrip.ReplaceAllString(line, "")
			// Trim and fold case the way the readiness gate does, so a
			// question the gate counts as open is always resolvable.
			if !strings.EqualFold(strings.TrimSpace(cur), "open") {
				out = append(out, "status: "+cur)
				inQ = false
				notOpen = true
			} else {
				out = append(out, "status: "+status)
			}
		case inQ && answeredAtRe.MatchString(line):
			answeredAtSeen = true
			out = append(out, "answered_at: "+ts)
		case inQ && answerRe.MatchString(line):
			answerSeen = true
			out = append(out, "answer: "+answer)
		case inQ && answeredByRe.MatchString(line):
			if human && !answeredBySeen {
				answeredBySeen = true
				out = append(out, "answered_by: human")
			}
		default:
			out = append(out, original)
		}
	}
	closeBlock()
	return out, found, notOpen
}

var (
	awaitingRe = regexp.MustCompile(`^## Awaiting human`)
	hdrSpaceRe = regexp.MustCompile(`^##[[:space:]]`)
	logHdrRe   = regexp.MustCompile(`^## Log`)
)

// clearAwaiting removes only matching wait representations. Another question
// keeps the workspace paused; otherwise the canonical phase resumes on SOURCE.
func clearAwaiting(data []byte, qid, slug string) ([]byte, error) {
	lines := splitLinesNoTrailing(data)
	visible, err := structuralLines(data)
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}
	var out []string
	matched, otherWait := false, false
	inLog, logAt := false, -1
	for i := 0; i < len(lines); i++ {
		line := visible[i]
		if inLog && hdrSpaceRe.MatchString(line) {
			if logAt < 0 {
				logAt = len(out)
			}
			inLog = false
		}
		if logHdrRe.MatchString(line) {
			inLog = true
		}
		if awaitingRe.MatchString(line) {
			end := i + 1
			for end < len(lines) && !hdrSpaceRe.MatchString(visible[end]) {
				end++
			}
			blockMatches, blockConflicts := false, false
			for j := i + 1; j < end; j++ {
				if waitingOn, ok := state.CursorField(visible[j:j+1], state.CursorQuestionID); ok && waitingOn != "" {
					if waitingOn == qid {
						blockMatches = true
					} else {
						blockConflicts = true
					}
				}
			}
			if blockMatches && blockConflicts {
				return nil, fmt.Errorf("state has conflicting question IDs in Awaiting human")
			}
			if blockMatches {
				matched = true
			} else {
				otherWait = true
				out = append(out, lines[i:end]...)
			}
			i = end - 1
			continue
		}
		if waitingOn, ok := state.CursorField(visible[i:i+1], state.CursorQuestionID); ok && waitingOn != "" {
			if waitingOn == qid {
				matched = true
				continue
			}
			otherWait = true
		}
		out = append(out, lines[i])
	}
	if !matched {
		return data, nil
	}
	rawPhase, _ := state.CursorField(lines, state.CursorPhase)
	fields := strings.Fields(strings.ToLower(rawPhase))
	var resumePolicy state.PhasePolicy
	known := false
	if len(fields) > 0 {
		resumePolicy, known = state.PolicyFor(state.Phase(fields[0]))
	}
	if !known || resumePolicy.ResumeVerb == "" {
		return nil, fmt.Errorf("cannot resume unknown or terminal phase %q", rawPhase)
	}
	if inLog && logAt < 0 {
		logAt = len(out)
	}
	if logAt >= 0 {
		out = append(out, "")
		copy(out[logAt+1:], out[logAt:])
		out[logAt] = fmt.Sprintf("- %s %s: resolved %s", nowUTC(), resumePolicy.Target, qid)
	}
	if !otherWait {
		action := workflow.ForVerb(resumePolicy.ResumeVerb).SourceInvocation(slug)
		out = state.UpsertCursorField(out, state.CursorStatus, "running")
		out = state.UpsertCursorField(out, state.CursorNextAction, action)
		if got, _ := state.CursorField(out, state.CursorStatus); got != "running" {
			return nil, fmt.Errorf("cannot update state status")
		}
		if got, _ := state.CursorField(out, state.CursorNextAction); got != action {
			return nil, fmt.Errorf("cannot update state next_action")
		}
	}
	return joinRecords(out), nil
}

// joinRecords joins lines with newlines and terminates the last one, so a rewritten
// file always ends with a trailing newline.
func joinRecords(records []string) []byte {
	if len(records) == 0 {
		return nil
	}
	return []byte(strings.Join(records, "\n") + "\n")
}
