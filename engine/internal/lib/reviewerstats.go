package lib

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

// gateStreak is how many consecutive zero-finding dispatches make a gateable
// reviewer a gate-candidate. Deterministic and engine-owned: the skill reads the
// verdict, it never re-derives the threshold.
const gateStreak = 10

// reviewerStatsFile is the project-level (cross-feature) dispatch-outcome ledger.
const reviewerStatsFile = "reviewer-stats.jsonl"

type reviewerStatEntry struct {
	TS       string `json:"ts"`
	Slug     string `json:"slug,omitempty"`
	Agent    string `json:"agent"`
	Findings int    `json:"findings"`
}

var reviewerAgentRe = regexp.MustCompile(`^devrites-[a-z][a-z-]*$`)

// alwaysOnReviewers are the roster's unconditional axes: stats are recorded for
// visibility but the verdict is always run.
var alwaysOnReviewers = map[string]bool{
	"devrites-spec-reviewer": true,
	"devrites-code-reviewer": true,
	"devrites-test-analyst":  true,
}

// insuranceReviewers are never gated regardless of streak: their value is the
// absence of findings, so a dry streak is success, not waste.
var insuranceReviewers = map[string]bool{
	"devrites-security-auditor": true,
	"devrites-doubt-reviewer":   true,
}

// ReviewerStats maintains the per-reviewer dispatch-outcome ledger the fan-out
// consults before dispatching conditional reviewers.
//
//	record <agent> <surviving-findings> [slug]  append one dispatch outcome
//	report [--json]                             per-agent verdict: run | gate-candidate
func ReviewerStats(root string, args []string, stdout, stderr io.Writer) int {
	switch argAt(args, 0) {
	case "record":
		agent := argAt(args, 1)
		if !reviewerAgentRe.MatchString(agent) {
			fmt.Fprintln(stderr, "reviewer-stats: agent must match ^devrites-[a-z][a-z-]*$")
			return 2
		}
		n, err := strconv.Atoi(argAt(args, 2))
		if err != nil || n < 0 {
			fmt.Fprintln(stderr, "reviewer-stats: <surviving-findings> must be a non-negative integer")
			return 2
		}
		slug := argAt(args, 3)
		if slug == "" {
			slug = activeSlug(root)
		}
		if err := os.MkdirAll(root, 0o755); err != nil {
			fmt.Fprintf(stderr, "reviewer-stats: %v\n", err)
			return 1
		}
		entry := reviewerStatEntry{TS: nowUTC(), Slug: slug, Agent: agent, Findings: n}
		if err := appendJSONLine(filepath.Join(root, reviewerStatsFile), entry); err != nil {
			fmt.Fprintf(stderr, "reviewer-stats: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "reviewer-stats: recorded.")
		return 0
	case "report":
		wantJSON := argAt(args, 1) == "--json"
		rows, err := reviewerStatRows(root)
		if err != nil {
			fmt.Fprintf(stderr, "reviewer-stats: %v\n", err)
			return 1
		}
		if len(rows) == 0 {
			fmt.Fprintln(stdout, "reviewer-stats: no dispatches recorded.")
			return 0
		}
		if wantJSON {
			enc := json.NewEncoder(stdout)
			return jsonEncodeRows(enc, rows, stderr)
		}
		for _, r := range rows {
			fmt.Fprintf(stdout, "%s: %s (%d finding%s / %d dispatch%s, zero-streak %d)\n",
				r.Agent, r.Verdict, r.Findings, plural(r.Findings), r.Dispatches, pluralES(r.Dispatches), r.ZeroStreak)
		}
		return 0
	default:
		fmt.Fprintln(stderr, "usage: devrites-engine reviewer-stats record <agent> <surviving-findings> [slug] | report [--json]")
		return 2
	}
}

type reviewerStatRow struct {
	Agent      string `json:"agent"`
	Dispatches int    `json:"dispatches"`
	Findings   int    `json:"findings"`
	ZeroStreak int    `json:"zero_streak"`
	Verdict    string `json:"verdict"`
}

func jsonEncodeRows(enc *json.Encoder, rows []reviewerStatRow, stderr io.Writer) int {
	for _, r := range rows {
		if err := enc.Encode(r); err != nil {
			fmt.Fprintf(stderr, "reviewer-stats: %v\n", err)
			return 1
		}
	}
	return 0
}

func reviewerStatRows(root string) ([]reviewerStatRow, error) {
	f, err := os.Open(filepath.Join(root, reviewerStatsFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	type acc struct {
		dispatches, findings, zeroStreak int
	}
	byAgent := map[string]*acc{}
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64*1024), 1024*1024)
	for s.Scan() {
		var e reviewerStatEntry
		if json.Unmarshal(s.Bytes(), &e) != nil || e.Agent == "" {
			continue // a corrupt line degrades the report, never breaks it
		}
		a := byAgent[e.Agent]
		if a == nil {
			a = &acc{}
			byAgent[e.Agent] = a
		}
		a.dispatches++
		a.findings += e.Findings
		if e.Findings == 0 {
			a.zeroStreak++
		} else {
			a.zeroStreak = 0
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	agents := make([]string, 0, len(byAgent))
	for name := range byAgent {
		agents = append(agents, name)
	}
	sort.Strings(agents)
	rows := make([]reviewerStatRow, 0, len(agents))
	for _, name := range agents {
		a := byAgent[name]
		rows = append(rows, reviewerStatRow{
			Agent:      name,
			Dispatches: a.dispatches,
			Findings:   a.findings,
			ZeroStreak: a.zeroStreak,
			Verdict:    reviewerVerdict(name, a.zeroStreak),
		})
	}
	return rows, nil
}

func reviewerVerdict(agent string, zeroStreak int) string {
	switch {
	case alwaysOnReviewers[agent]:
		return "run (always-on)"
	case insuranceReviewers[agent]:
		return "run (insurance: never gated)"
	case zeroStreak >= gateStreak:
		return "gate-candidate"
	default:
		return "run"
	}
}

func pluralES(n int) string {
	if n == 1 {
		return ""
	}
	return "es"
}
