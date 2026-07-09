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
	"github.com/devrites/devrites/internal/workflow"
)

// Resolve answers or drops an open question and keeps questions.md and state.md
// consistent: it rewrites the question's block and, when state.md is awaiting that
// question, clears the "Awaiting human" block and flips Status back to running.
// The workspace is <root>/features/<slug>.
//
//	0  resolved         2  no active workspace      3  qid not found
//	4  qid not open      5  bad arguments            6  qid collision (next-qid)
func Resolve(root string, args []string, stdout, stderr io.Writer) int {
	// next-qid works on an explicit questions.md path, without an active workspace.
	if argAt(args, 0) == "next-qid" {
		return resolveNextQID(argAt(args, 1), stdout, stderr)
	}

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
		return fail(stderr, `Usage: devrites-engine resolve <qid> "<answer>"  |  --drop <qid> ["<reason>"]  |  --batch <file>`, 5)
	default:
		mode = "answer"
		qid = first
		payload = argAt(args, 1)
		if payload == "" {
			return fail(stderr, "Answer text required for "+qid, 5)
		}
	}

	switch mode {
	case "answer":
		if code := resolveQuestion(qfile, qid, "answered", payload, stderr); code != 0 {
			return code
		}
		clearAwaiting(sfile, qid)
		fmt.Fprintf(stdout, "Resolved: %s\n", qid)
		fmt.Fprintf(stdout, "Status:   answered\n")
		fmt.Fprintf(stdout, "Workspace: questions.md + state.md updated.\n")
	case "drop":
		if code := resolveQuestion(qfile, qid, "dropped", payload, stderr); code != 0 {
			return code
		}
		clearAwaiting(sfile, qid)
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
				clearAwaiting(sfile, bid)
				fmt.Fprintf(stdout, "Dropped:  %s — %s\n", bid, reason)
			} else {
				bid, ans := splitColon(line)
				ans = strings.TrimPrefix(ans, " ")
				if code := resolveQuestion(qfile, bid, "answered", ans, stderr); code != 0 {
					return code
				}
				clearAwaiting(sfile, bid)
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

// clockNow is the single wall-clock read for the resolve command. When
// DEVRITES_NOW is set to an RFC-3339 timestamp or a bare YYYY-MM-DD date it pins
// the clock, so date-derived output (the next question id) is deterministic
// under test and stable across the day boundaries that otherwise rot golden
// snapshots. One overridable seam instead of scattered time.Now() reads. See
// ADR-0006.
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

// resolveNextQID prints the next sequential question id for today, refusing (exit
// 6) an id that already has a header — a sign questions.md was hand-edited.
func resolveNextQID(qpath string, stdout, stderr io.Writer) int {
	if qpath == "" {
		return fail(stderr, "Usage: devrites-engine resolve next-qid <questions.md path>", 5)
	}
	today := clockNow().Format("2006-01-02")
	used := 0
	var content []byte
	if isFile(qpath) {
		content, _ = os.ReadFile(qpath)
		countRe := regexp.MustCompile(`(?m)^## q-` + regexp.QuoteMeta(today) + `-`)
		used = len(countRe.FindAllIndex(content, -1))
	}
	next := used + 1
	qid := fmt.Sprintf("q-%s-%03d", today, next)
	if content != nil {
		collideRe := regexp.MustCompile(`(?m)^## ` + regexp.QuoteMeta(qid) + `([[:space:]]|$)`)
		if collideRe.Match(content) {
			return fail(stderr, "qid already present: "+qid+" (questions.md edited out of sequence)", 6)
		}
	}
	fmt.Fprintln(stdout, qid)
	return 0
}

// resolveQuestion rewrites the <qid> block in questions.md so it records the new
// status, answer, and answered-at time. It works in two passes: the first updates
// any fields already present (and detects a missing or non-open block); the second
// appends answered_at/answer if the block did not carry them. Returns 0, or 3 (qid
// not found) / 4 (qid not open), writing the message on failure.
func resolveQuestion(qfile, qid, status, answer string, stderr io.Writer) int {
	data, err := os.ReadFile(qfile)
	if err != nil {
		return fail(stderr, "questions.md missing at "+qfile, 2)
	}
	ts := nowUTC()
	target := regexp.MustCompile(`^## ` + qid + `([[:space:]]|$)`)

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

	reread, _ := os.ReadFile(qfile)
	complete := ensureAnswerFields(splitLinesNoTrailing(reread), target, answer, ts)
	_ = fsutil.WriteFileAtomic(qfile, joinRecords(complete), 0o644)
	return 0
}

var (
	qHeaderRe    = regexp.MustCompile(`^## q-`)
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
	statusSeen := false
	closeBlock := func() {
		if inQ && !statusSeen {
			out = append(out, "status: "+status, "answered_at: "+ts, "answer: "+answer)
		}
	}
	for _, line := range lines {
		switch {
		case qHeaderRe.MatchString(line):
			closeBlock()
			inQ = target.MatchString(line)
			statusSeen = false
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
			out = append(out, "answered_at: "+ts)
		case inQ && answerRe.MatchString(line):
			out = append(out, "answer: "+answer)
		default:
			out = append(out, line)
		}
	}
	closeBlock()
	return out, found, notOpen
}

// ensureAnswerFields appends answered_at/answer to the target question's block if
// the first pass did not find existing lines to rewrite.
func ensureAnswerFields(lines []string, target *regexp.Regexp, answer, ts string) []string {
	var out []string
	inQ, sawAt, sawAns := false, false, false
	for _, line := range lines {
		if qHeaderRe.MatchString(line) {
			if inQ {
				if !sawAt {
					out = append(out, "answered_at: "+ts)
				}
				if !sawAns {
					out = append(out, "answer: "+answer)
				}
			}
			inQ = target.MatchString(line)
			sawAt, sawAns = false, false
			out = append(out, line)
			continue
		}
		if inQ && answeredAtRe.MatchString(line) {
			sawAt = true
		}
		if inQ && answerRe.MatchString(line) {
			sawAns = true
		}
		out = append(out, line)
	}
	if inQ {
		if !sawAt {
			out = append(out, "answered_at: "+ts)
		}
		if !sawAns {
			out = append(out, "answer: "+answer)
		}
	}
	return out
}

var (
	awaitingRe = regexp.MustCompile(`^## Awaiting human`)
	hdrSpaceRe = regexp.MustCompile(`^##[[:space:]]`)
	statusMdRe = regexp.MustCompile(`^- Status:`)
	nextStepRe = regexp.MustCompile(`^- Next step:`)
	logHdrRe   = regexp.MustCompile(`^## Log`)
)

// clearAwaiting resumes the workflow after a question is resolved: when state.md's
// "Awaiting human" block references qid, it drops that block, flips Status back to
// running, clears the Next step, and appends a Log entry. It is a no-op when the
// workspace is not waiting on this question.
func clearAwaiting(sfile, qid string) {
	data, err := os.ReadFile(sfile)
	if err != nil {
		return
	}
	lines := splitLinesNoTrailing(data)
	qidRef := regexp.MustCompile(`qid:[[:space:]]*` + qid + `([[:space:]]|$)`)

	// First check whether the awaiting block references this question at all.
	inAw, matched := false, false
	for _, line := range lines {
		switch {
		case awaitingRe.MatchString(line):
			inAw = true
			continue
		case inAw && hdrSpaceRe.MatchString(line):
			inAw = false
		}
		if inAw && qidRef.MatchString(line) {
			matched = true
		}
	}
	if !matched {
		return
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
		case statusMdRe.MatchString(line):
			out = append(out, "- Status: running")
			continue
		case nextStepRe.MatchString(line):
			out = append(out, "- Next step: (resume — `"+workflow.ForVerb("build").Both()+"` to continue the workflow)")
			continue
		case logHdrRe.MatchString(line):
			out = append(out, line)
			inLog = true
			continue
		case inLog && hdrSpaceRe.MatchString(line):
			if !logAppended {
				out = append(out, fmt.Sprintf("- %s build: resolved %s", ts, qid))
				logAppended = true
			}
			inLog = false
			out = append(out, line)
			continue
		}
		out = append(out, line)
	}
	if inLog && !logAppended {
		out = append(out, fmt.Sprintf("- %s build: resolved %s", ts, qid))
	}
	_ = fsutil.WriteFileAtomic(sfile, joinRecords(out), 0o644)
}

// joinRecords joins lines with newlines and terminates the last one, so a rewritten
// file always ends with a trailing newline.
func joinRecords(records []string) []byte {
	if len(records) == 0 {
		return nil
	}
	return []byte(strings.Join(records, "\n") + "\n")
}
