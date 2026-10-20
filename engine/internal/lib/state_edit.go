package lib

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/state"
)

const (
	stateSetUsage    = "usage: devrites-engine state set <slug> <key>=<value>... [--unset <key>]..."
	stateAppendUsage = `usage: devrites-engine state append <slug> <ledger.md> --heading "<text>" --body-file <path|->`
	maxAppendBytes   = 256 << 10
)

var cursorKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// appendLedgers are the append-only workspace ledgers. state.md belongs to
// `state set`; spec, plan, and tasks change only through their owning rite.
var appendLedgers = []string{"assumptions.md", "decisions.md", "drift.md", "evidence.md", "questions.md"}

// RunStateSet implements `devrites-engine state set`: it writes cursor fields in
// state.md all-or-nothing under the feature lock, so the root never splices the
// cursor with an ad-hoc script. It refuses an unknown phase, the engine-owned
// schema row, and an overwrite of an existing return cursor.
//
//	0  written    2  usage error    3  blocked; nothing written    1  I/O error
func RunStateSet(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 || strings.HasPrefix(args[0], "-") {
		return fail(stderr, stateSetUsage, 2)
	}
	slug := args[0]
	type change struct{ key, value string }
	var changes []change
	seen := map[string]bool{}
	for i := 1; i < len(args); i++ {
		var key, value string
		if args[i] == "--unset" {
			if i+1 == len(args) {
				return fail(stderr, stateSetUsage, 2)
			}
			i++
			key = args[i]
		} else {
			var ok bool
			if key, value, ok = strings.Cut(args[i], "="); !ok || value == "" {
				return fail(stderr, stateSetUsage, 2)
			}
		}
		if !cursorKeyPattern.MatchString(key) || seen[key] {
			return fail(stderr, fmt.Sprintf("state set: key %q is not a snake_case cursor key or repeats", key), 2)
		}
		seen[key] = true
		changes = append(changes, change{key, value})
	}
	sfile, code := stateEditWorkspace(root, slug, "state.md", stderr)
	if code != 0 {
		return code
	}
	for _, c := range changes {
		switch {
		case c.key == state.CursorSchema:
			return fail(stderr, "state set: BLOCKED: schema is engine-owned; run devrites-engine migrate", 3)
		case c.key == state.CursorPhase:
			if _, ok := state.PolicyFor(state.Phase(c.value)); !ok {
				return fail(stderr, fmt.Sprintf("state set: BLOCKED: unknown phase %q", c.value), 3)
			}
		case strings.ContainsAny(c.value, "|\r\n"):
			return fail(stderr, fmt.Sprintf("state set: BLOCKED: value for %s has a pipe or line break; a cursor cell holds one line", c.key), 3)
		}
	}

	code = 0
	if err := state.WithFeatureLock(root, slug, func() error {
		// #nosec G304 -- state.md inside the checked feature workspace
		raw, err := os.ReadFile(sfile)
		if err != nil {
			return err
		}
		lines := splitLinesNoTrailing(raw)
		for _, c := range changes {
			current, exists := state.CursorField(lines, c.key)
			if c.value == "" {
				lines = state.DeleteCursorField(lines, c.key)
				continue
			}
			isReturn := c.key == state.CursorReturnPhase || c.key == state.CursorReturnNextAction
			if isReturn && exists && current != "" && current != c.value {
				fmt.Fprintf(stderr, "state set: BLOCKED: %s already holds %q; restore that return cursor and --unset it before saving another\n", c.key, current)
				code = 3
				return nil
			}
			next := state.UpsertCursorField(lines, c.key, c.value)
			if got, ok := state.CursorField(next, c.key); !ok || got != c.value {
				fmt.Fprintf(stderr, "state set: BLOCKED: state.md cursor cannot hold %s; nothing written\n", c.key)
				code = 3
				return nil
			}
			lines = next
		}
		return fsutil.WriteFileAtomic(sfile, joinRecords(lines), 0o644)
	}); err != nil {
		return fail(stderr, "state set: "+err.Error(), 1)
	}
	if code == 0 {
		fmt.Fprintf(stdout, "state set: %s: %d cursor field(s) written\n", slug, len(changes))
		if slices.Contains(changes, change{state.CursorStatus, "awaiting_human"}) {
			for _, q := range afkAutoPickable(root, filepath.Join(filepath.Dir(sfile), "questions.md")) {
				fmt.Fprintf(stdout, "state set: advisory: AFK auto-picks %s (gate %s is in allow_gates); unless it is on the irreversible-risk list, resolve it with its proposed answer from questions.md via devrites-engine state resolve\n", q.id, q.gate)
			}
		}
	}
	return code
}

type openQuestion struct{ id, gate, proposed string }

// afkAutoPickable lists open questions the AFK contract answers without a
// human: the gate is in the sentinel's allow_gates and a proposed answer exists.
// No sentinel, no allow_gates line, or an unreadable file yields none.
func afkAutoPickable(root, qfile string) []openQuestion {
	// #nosec G304 -- .devrites/AFK sentinel
	afk, err := os.ReadFile(filepath.Join(root, "AFK"))
	if err != nil {
		return nil
	}
	allowed := map[string]bool{}
	for _, line := range strings.Split(string(afk), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "allow_gates:"); ok {
			rest, _, _ = strings.Cut(rest, "#")
			for _, gate := range strings.Split(strings.Trim(strings.TrimSpace(rest), "[]"), ",") {
				if gate = strings.ToLower(strings.TrimSpace(gate)); gate != "" {
					allowed[gate] = true
				}
			}
		}
	}
	// #nosec G304 -- questions.md inside the checked feature workspace
	raw, err := os.ReadFile(qfile)
	if err != nil {
		return nil
	}
	lines, err := structuralLines(raw)
	if err != nil {
		return nil
	}
	var out []openQuestion
	var cur openQuestion
	var status string
	flush := func() {
		// blocking and escalating always pause, whatever allow_gates lists.
		if cur.id != "" && status == "open" && allowed[cur.gate] && cur.gate != "blocking" && cur.gate != "escalating" && cur.proposed != "" {
			out = append(out, cur)
		}
	}
	for _, line := range lines {
		if id, ok := strings.CutPrefix(line, "## "); ok && strings.HasPrefix(id, "q-") {
			flush()
			cur, status = openQuestion{id: strings.TrimSpace(id)}, ""
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || cur.id == "" {
			continue
		}
		value = strings.TrimSpace(value)
		switch key {
		case "status":
			status = strings.ToLower(value)
		case "gate":
			cur.gate = strings.ToLower(value)
		case "proposed":
			cur.proposed = value
		}
	}
	flush()
	return out
}

// RunStateAppend implements `devrites-engine state append`: it appends one
// `## <heading>` section to an append-only ledger atomically and refuses a
// heading that already exists, so a root write can never replace history.
//
//	0  appended    2  usage error    3  blocked; nothing written    1  I/O error
func RunStateAppend(root string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var positional []string
	var heading, bodyFile string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--heading", "--body-file":
			if i+1 == len(args) {
				return fail(stderr, stateAppendUsage, 2)
			}
			if args[i] == "--heading" {
				heading = args[i+1]
			} else {
				bodyFile = args[i+1]
			}
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				return fail(stderr, stateAppendUsage, 2)
			}
			positional = append(positional, args[i])
		}
	}
	heading = strings.TrimSpace(heading)
	if len(positional) != 2 || heading == "" || bodyFile == "" || strings.ContainsAny(heading, "\r\n") {
		return fail(stderr, stateAppendUsage, 2)
	}
	if strings.HasPrefix(heading, "#") || !utf8.ValidString(heading) {
		return fail(stderr, "state append: BLOCKED: pass the heading text without leading #, as valid UTF-8", 3)
	}
	slug, ledger := positional[0], positional[1]
	if !slices.Contains(appendLedgers, ledger) {
		return fail(stderr, fmt.Sprintf("state append: BLOCKED: %s is not an append-only ledger (%s)", ledger, strings.Join(appendLedgers, ", ")), 3)
	}
	body, err := readAppendBody(bodyFile, stdin)
	if err != nil {
		return fail(stderr, "state append: BLOCKED: body: "+err.Error(), 3)
	}
	bodyLines, err := structuralLines(body)
	if err != nil {
		return fail(stderr, "state append: BLOCKED: body: "+err.Error(), 3)
	}
	for _, line := range bodyLines {
		if sectionHeadingPattern.MatchString(line) {
			return fail(stderr, fmt.Sprintf("state append: BLOCKED: body line %q would open another section; use ### for sub-headings", line), 3)
		}
	}
	// Only the new entry is linted; existing questions.md history stays as written.
	var advisories []string
	if ledger == "questions.md" && strings.HasPrefix(heading, "q-") {
		var problems []string
		if problems, advisories = lintQuestionEntry(heading, bodyLines, len(body)); len(problems) > 0 {
			return fail(stderr, fmt.Sprintf("state append: BLOCKED: questions.md entry %s: %s; see afk-hitl.md questions.md schema; nothing written", heading, strings.Join(problems, "; ")), 3)
		}
	}
	path, code := stateEditWorkspace(root, slug, ledger, stderr)
	if code != 0 {
		return code
	}

	code = 0
	var lines int
	if err := state.WithFeatureLock(root, slug, func() error {
		// #nosec G304 -- append-only ledger inside the checked feature workspace
		raw, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		existing, err := structuralLines(raw)
		if err != nil {
			fmt.Fprintf(stderr, "state append: BLOCKED: %s: %v\n", ledger, err)
			code = 3
			return nil
		}
		if hasSectionHeading(existing, heading) {
			fmt.Fprintf(stderr, "state append: BLOCKED: %s already has \"## %s\" or its ID; history is append-only, pick a new ID\n", ledger, heading)
			code = 3
			return nil
		}
		var out bytes.Buffer
		out.Write(raw)
		if len(raw) > 0 {
			if !bytes.HasSuffix(raw, []byte("\n")) {
				out.WriteByte('\n')
			}
			out.WriteByte('\n')
		}
		fmt.Fprintf(&out, "## %s\n\n%s\n", heading, strings.TrimRight(string(body), "\n"))
		// An unclosed fence in the body would hide every later heading from the
		// duplicate check, so the next section must still be visible.
		probe, err := structuralLines(append(bytes.Clone(out.Bytes()), "\n## sentinel\n"...))
		if err != nil || probe[len(probe)-2] != "## sentinel" {
			fmt.Fprintln(stderr, "state append: BLOCKED: the body or ledger leaves a code fence open; close it")
			code = 3
			return nil
		}
		// Same material band the gate blocks, so a ledger cannot grow past it
		// between gates; proof ledgers stay advisory.
		if status, ok := state.ArtifactBudget(ledger, out.Bytes()); ok && state.ArtifactBudgetGateApplies(ledger) && status.Material && !status.Override {
			fmt.Fprintf(stderr, "state append: BLOCKED: %s would be %d lines/%d bytes, at or above %gx its %d-line/%d-byte budget; %s; nothing written\n",
				ledger, status.Lines, status.Bytes, state.BudgetBlockFactor, status.LineBudget, status.ByteBudget, state.BudgetRemedy(status.LineBudget))
			code = 3
			return nil
		}
		lines = state.CountLines(out.Bytes())
		return fsutil.WriteFileAtomic(path, out.Bytes(), 0o644)
	}); err != nil {
		return fail(stderr, "state append: "+err.Error(), 1)
	}
	if code != 0 {
		return code
	}
	fmt.Fprintf(stdout, "state append: %s: added \"## %s\" (%d lines)\n", ledger, heading, lines)
	if budget, ok := state.ArtifactLineBudget(ledger); ok && lines > budget {
		fmt.Fprintf(stdout, "state append: advisory: %s is %d lines, over its %d-line budget\n", ledger, lines, budget)
	}
	for _, a := range advisories {
		fmt.Fprintf(stdout, "state append: advisory: %s: %s\n", heading, a)
	}
	return 0
}

// stateEditWorkspace returns the path of name inside slug's current-schema
// workspace, or a nonzero exit code after reporting why it cannot be edited.
func stateEditWorkspace(root, slug, name string, stderr io.Writer) (string, int) {
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return "", fail(stderr, fmt.Sprintf("state: BLOCKED: workspace for %s: %v", slug, err), 3)
	}
	if err := state.RequireWorkspaceSchema(root, slug); err != nil {
		return "", fail(stderr, "state: BLOCKED: "+err.Error(), 3)
	}
	path := filepath.Join(workspace, name)
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return "", fail(stderr, fmt.Sprintf("state: BLOCKED: %s is not a regular file", name), 3)
	}
	return path, 0
}

func readAppendBody(name string, stdin io.Reader) ([]byte, error) {
	var body []byte
	var err error
	if name == "-" {
		body, err = io.ReadAll(io.LimitReader(stdin, maxAppendBytes+1))
		if err == nil && len(body) > maxAppendBytes {
			err = fmt.Errorf("exceeds %d bytes", maxAppendBytes)
		}
	} else {
		body, err = readBoundedRegularFile(name, maxAppendBytes)
	}
	if err != nil {
		return nil, err
	}
	body = bytes.ReplaceAll(body, []byte("\r\n"), []byte("\n"))
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, errors.New("empty")
	}
	return body, nil
}

// structuralLines masks fenced code so headings inside examples never count.
func structuralLines(raw []byte) ([]string, error) {
	masked, err := markdowntext.Structural(raw)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(masked), "\n"), nil
}

// sectionHeadingPattern matches an ATX # or ## heading that opens a section.
var sectionHeadingPattern = regexp.MustCompile(`^ {0,3}#{1,2}([ \t]|$)`)

// ledgerIDPattern matches a stable ledger ID such as DRIFT-011, EV-026, or
// q-2026-10-19-007 at the start of a heading.
var ledgerIDPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(-[A-Za-z0-9]+)*-[0-9]+$`)

func headingID(heading string) string {
	fields := strings.Fields(heading)
	if len(fields) == 0 {
		return ""
	}
	if id := strings.TrimRight(fields[0], ":.,"); ledgerIDPattern.MatchString(id) {
		return id
	}
	return ""
}

// hasSectionHeading reports a `## ` heading equal to heading, or one that
// starts with the same ledger ID: IDs are append-only and never reused.
func hasSectionHeading(lines []string, heading string) bool {
	id := headingID(heading)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !sectionHeadingPattern.MatchString(trimmed) || strings.HasPrefix(trimmed, "# ") || trimmed == "#" {
			continue
		}
		if text := strings.TrimSpace(strings.TrimPrefix(trimmed, "##")); text == heading || id != "" && headingID(text) == id {
			return true
		}
	}
	return false
}
