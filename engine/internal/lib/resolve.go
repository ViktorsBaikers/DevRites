package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/fsutil"
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

	var mode, qid, payload string
	switch first := argAt(args, 0); first {
	case "--drop":
		mode = "drop"
		qid = argAt(args, 1)
		payload = argAt(args, 2)
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
		return fail(stderr, `Usage: devrites-engine state resolve <qid> "<answer>"  |  state resolve --drop <qid> ["<reason>"]  |  state resolve --batch <file>`, 5)
	default:
		mode = "answer"
		qid = first
		payload = argAt(args, 1)
		if payload == "" {
			return fail(stderr, "Answer text required for "+qid, 5)
		}
	}

	code := 0
	if err := state.WithFeatureLock(root, slug, func() error {
		code = resolveMutation(mode, qid, payload, qfile, sfile, stdout, stderr)
		return nil
	}); err != nil {
		return fail(stderr, err.Error(), 1)
	}
	return code
}

func resolveMutation(mode, qid, payload, qfile, sfile string, stdout, stderr io.Writer) int {
	switch mode {
	case "answer":
		if code := resolveQuestion(qfile, qid, "answered", payload, stderr); code != 0 {
			return code
		}
		if err := clearAwaiting(sfile, qid); err != nil {
			return fail(stderr, err.Error(), 1)
		}
		fmt.Fprintf(stdout, "Resolved: %s\n", qid)
		fmt.Fprintf(stdout, "Status:   answered\n")
		fmt.Fprintf(stdout, "Workspace: questions.md + state.md updated.\n")
	case "drop":
		if code := resolveQuestion(qfile, qid, "dropped", payload, stderr); code != 0 {
			return code
		}
		if err := clearAwaiting(sfile, qid); err != nil {
			return fail(stderr, err.Error(), 1)
		}
		fmt.Fprintf(stdout, "Dropped:  %s\n", qid)
		fmt.Fprintf(stdout, "Reason:   %s\n", payload)
		fmt.Fprintf(stdout, "Workspace: questions.md + state.md updated.\n")
	case "batch":
		data, err := os.ReadFile(payload)
		if err != nil {
			return fail(stderr, "Batch file not found: "+payload, 5)
		}
		for _, line := range batchLines(data) {
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "--drop") {
				rest := strings.TrimPrefix(line, "--drop ")
				bid, reason := splitColon(rest)
				reason = strings.TrimPrefix(reason, " ")
				if code := resolveQuestion(qfile, bid, "dropped", reason, stderr); code != 0 {
					return code
				}
				if err := clearAwaiting(sfile, bid); err != nil {
					return fail(stderr, err.Error(), 1)
				}
				fmt.Fprintf(stdout, "Dropped:  %s: %s\n", bid, reason)
			} else {
				bid, ans := splitColon(line)
				ans = strings.TrimPrefix(ans, " ")
				if code := resolveQuestion(qfile, bid, "answered", ans, stderr); code != 0 {
					return code
				}
				if err := clearAwaiting(sfile, bid); err != nil {
					return fail(stderr, err.Error(), 1)
				}
				fmt.Fprintf(stdout, "Resolved: %s\n", bid)
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

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// batchLines returns the newline-terminated lines of a batch file. An unterminated
// final line is dropped, matching a shell `read` loop, so a partial trailing line
// is never applied as an entry.
func batchLines(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	return lines[:len(lines)-1]
}

// splitColon splits s at the first ':', returning the parts before and after it
// (both the whole string when there is no ':').
func splitColon(s string) (before, after string) {
	before, after, ok := strings.Cut(s, ":")
	if !ok {
		return s, s
	}
	return before, after
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

// resolveQuestion rewrites the <qid> block in questions.md so it records the new
// status, answer, and answered-at time in one pass. Returns 0, or 3 (qid
// not found) / 4 (qid not open), writing the message on failure.
func resolveQuestion(qfile, qid, status, answer string, stderr io.Writer) int {
	data, err := os.ReadFile(qfile)
	if err != nil {
		return fail(stderr, "questions.md missing at "+qfile, 2)
	}
	ts := nowUTC()
	target := regexp.MustCompile(`^## ` + regexp.QuoteMeta(qid) + `([[:space:]]|$)`)

	updated, found, notOpen := rewriteQuestionFields(splitLinesNoTrailing(data), target, status, answer, ts)
	if !found {
		return fail(stderr, "qid not found: "+qid, 3)
	}
	if notOpen {
		return fail(stderr, "qid not open (already answered/dropped): "+qid, 4)
	}
	if err := fsutil.WriteFileAtomic(qfile, joinRecords(updated), 0o644); err != nil {
		return fail(stderr, err.Error(), 1)
	}

	return 0
}

var (
	qHeaderRe    = regexp.MustCompile(`(?i)^## q-`)
	statusLineRe = regexp.MustCompile(`^status:`)
	statusStrip  = regexp.MustCompile(`^status:[[:space:]]*`)
	answeredAtRe = regexp.MustCompile(`^answered_at:`)
	answerRe     = regexp.MustCompile(`^answer:`)
)

// rewriteQuestionFields walks questions.md and, inside the target question's
// block, sets status and rewrites any answered_at/answer lines already present. It
// reports whether the block was found and whether it was already not open (an
// attempt to re-answer). A target block with no status line at all gets a full set
// of closing fields appended.
func rewriteQuestionFields(lines []string, target *regexp.Regexp, status, answer, ts string) (out []string, found, notOpen bool) {
	inQ := false
	statusSeen, answeredAtSeen, answerSeen := false, false, false
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
	}
	for _, line := range lines {
		switch {
		case qHeaderRe.MatchString(line):
			closeBlock()
			inQ = target.MatchString(line)
			statusSeen, answeredAtSeen, answerSeen = false, false, false
			if inQ {
				found = true
			}
			out = append(out, line)
		case inQ && statusLineRe.MatchString(line):
			statusSeen = true
			cur := statusStrip.ReplaceAllString(line, "")
			if cur != "open" {
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
		default:
			out = append(out, line)
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

// clearAwaiting resumes the workflow after a question is resolved: when state.md's
// "Awaiting human" block references qid, it drops that block, flips Status back to
// running, clears the Next step, and appends a Log entry. It is a no-op when the
// workspace is not waiting on this question.
func clearAwaiting(sfile, qid string) error {
	data, err := os.ReadFile(sfile)
	if err != nil {
		return fmt.Errorf("read state %s: %w", sfile, err)
	}
	lines := splitLinesNoTrailing(data)
	resumePhase := state.PhaseBuild
	if rawPhase, ok := state.CursorField(lines, state.CursorPhase); ok {
		if fields := strings.Fields(strings.ToLower(rawPhase)); len(fields) > 0 {
			if phase, known := state.PhaseForName(fields[0]); known && state.ResumeVerb(phase) != "" {
				resumePhase = phase
			}
		}
	}
	resumeCommand := workflow.ForVerb(state.ResumeVerb(resumePhase))
	// First check whether the awaiting block references this question at all.
	inAw := false
	var awaitingLines []string
	for _, line := range lines {
		switch {
		case awaitingRe.MatchString(line):
			inAw = true
			continue
		case inAw && hdrSpaceRe.MatchString(line):
			inAw = false
		}
		if inAw {
			awaitingLines = append(awaitingLines, line)
		}
	}
	waitingOn, _ := state.CursorField(awaitingLines, state.CursorQuestionID)
	if waitingOn != qid {
		return nil
	}

	// Rewrite: drop the awaiting block, reset Status/Next step, log the resolution.
	ts := nowUTC()
	var out []string
	inAw, inLog, logAppended := false, false, false
	for _, line := range lines {
		if awaitingRe.MatchString(line) {
			inAw = true
			continue
		}
		if inAw && hdrSpaceRe.MatchString(line) {
			inAw = false
		}
		if inAw {
			continue
		}
		switch {
		case logHdrRe.MatchString(line):
			out = append(out, line)
			inLog = true
			continue
		case inLog && hdrSpaceRe.MatchString(line):
			if !logAppended {
				out = append(out, fmt.Sprintf("- %s %s: resolved %s", ts, resumePhase, qid))
				logAppended = true
			}
			inLog = false
			out = append(out, line)
			continue
		}
		out = append(out, line)
	}
	if inLog && !logAppended {
		out = append(out, fmt.Sprintf("- %s %s: resolved %s", ts, resumePhase, qid))
	}
	out, _ = state.SetCursorField(out, state.CursorStatus, "running")
	out, _ = state.SetCursorField(out, state.CursorNextAction, "(resume: `"+resumeCommand.Both()+"` to continue the workflow)")
	if err := fsutil.WriteFileAtomic(sfile, joinRecords(out), 0o644); err != nil {
		return fmt.Errorf("update state %s: %w", sfile, err)
	}
	return nil
}

// joinRecords joins lines with newlines and terminates the last one, so a rewritten
// file always ends with a trailing newline.
func joinRecords(records []string) []byte {
	if len(records) == 0 {
		return nil
	}
	return []byte(strings.Join(records, "\n") + "\n")
}
