package lib

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
)

// MetricsFile is the per-feature append-only event ledger. One JSON object per
// line keeps writes atomic-enough for a single appender and lets summaries
// tolerate trailing partial lines.
const MetricsFile = "metrics.jsonl"

const metricsUsage = `usage: devrites-engine metrics <record|summary> ...

  metrics record <slug> --phase <p> --event <e> [--role r] [--bytes n] [--note s]
  metrics summary [slug] [--by-slice]
`

type metricEvent struct {
	TS    string `json:"ts"`
	Phase string `json:"phase"`
	Event string `json:"event"`
	Role  string `json:"role,omitempty"`
	Bytes int64  `json:"bytes,omitempty"`
	Note  string `json:"note,omitempty"`
}

// RunMetrics records or summarizes per-feature workflow events.
//
//	metrics record <slug> --phase <p> --event <e> [--role r] [--bytes n] [--note s]
//	metrics summary <slug>
func RunMetrics(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, metricsUsage)
		return 2
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "record":
		return metricsRecord(root, rest, stdout, stderr)
	case "summary":
		return metricsSummary(root, rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "devrites: unknown metrics command %q\n", sub)
		return 2
	}
}

func metricsRecord(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, metricsUsage)
		return 2
	}
	slug := args[0]
	event := metricEvent{TS: time.Now().UTC().Format(time.RFC3339)}
	for i := 1; i < len(args); i++ {
		value := argAt(args, i+1)
		switch args[i] {
		case "--phase":
			event.Phase = value
		case "--event":
			event.Event = value
		case "--role":
			event.Role = value
		case "--bytes":
			n, err := strconv.ParseInt(value, 10, 64)
			if err != nil || n < 0 {
				fmt.Fprintf(stderr, "metrics: --bytes expects a non-negative integer, got %q\n", value)
				return 2
			}
			event.Bytes = n
		case "--note":
			event.Note = value
		default:
			fmt.Fprintf(stderr, "metrics: unknown flag %q\n", args[i])
			return 2
		}
		i++
	}
	if event.Phase == "" || event.Event == "" {
		fmt.Fprintln(stderr, "metrics: record requires --phase and --event")
		return 2
	}
	line, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintf(stderr, "metrics: %v\n", err)
		return 2
	}
	dir, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "metrics: %v\n", err)
		return 2
	}
	path := filepath.Join(dir, MetricsFile)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) // #nosec G304 -- path is a validated workspace metrics file
	if err != nil {
		fmt.Fprintf(stderr, "metrics: %v\n", err)
		return 2
	}
	_, werr := f.Write(append(line, '\n'))
	cerr := f.Close()
	if werr != nil {
		fmt.Fprintf(stderr, "metrics: %v\n", werr)
		return 2
	}
	if cerr != nil {
		fmt.Fprintf(stderr, "metrics: %v\n", cerr)
		return 2
	}
	fmt.Fprintf(stdout, "metrics: recorded %s/%s\n", event.Phase, event.Event)
	return 0
}

// recordMetric is the in-process recorder for engine-owned events (context
// bundle emission). It never fails the caller: a metrics write is observability,
// not a gate.
func recordMetric(root, slug, phase, event, role string, bytes int64) {
	appendMetric(root, slug, metricEvent{Phase: phase, Event: event, Role: role, Bytes: bytes})
}

// recordMetricNote is recordMetric for an event that carries a note, such as a
// return's reviewer outcome or an abandon reason.
func recordMetricNote(root, slug, phase, event, role, note string) {
	appendMetric(root, slug, metricEvent{Phase: phase, Event: event, Role: role, Note: note})
}

func appendMetric(root, slug string, entry metricEvent) {
	dir, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return
	}
	entry.TS = time.Now().UTC().Format(time.RFC3339)
	line, err := json.Marshal(entry)
	if err != nil {
		return
	}
	path := filepath.Join(dir, MetricsFile)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) // #nosec G304 -- path is a validated workspace metrics file
	if err != nil {
		return
	}
	_, _ = f.Write(append(line, '\n'))
	_ = f.Close()
}

// cursorPhase is the feature's recorded cursor phase, or "" when state.md is
// unreadable; engine checks label their metrics with it.
func cursorPhase(root, slug string) string {
	report, err := state.Status(root, slug)
	if err != nil {
		return ""
	}
	return string(report.Phase)
}

func metricsSummary(root string, args []string, stdout, stderr io.Writer) int {
	bySlice := slices.Contains(args, "--by-slice")
	args = slices.DeleteFunc(slices.Clone(args), func(a string) bool { return a == "--by-slice" })
	slug, code, err := ActiveSlug(root, args)
	if err != nil {
		fmt.Fprintf(stderr, "metrics: %v\n", err)
		if code == 0 {
			code = 2
		}
		return code
	}
	dir, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "metrics: %v\n", err)
		return 2
	}
	if bySlice {
		return sliceCostSummary(root, slug, stdout, stderr)
	}
	f, err := os.Open(filepath.Join(dir, MetricsFile)) // #nosec G304 -- dir is a validated workspace and filename is fixed
	if os.IsNotExist(err) {
		fmt.Fprintf(stdout, "metrics: no events recorded for %s\n", slug)
		return 0
	}
	if err != nil {
		fmt.Fprintf(stderr, "metrics: ledger unreadable: %v\n", err)
		return 2
	}
	defer func() { _ = f.Close() }()

	type phaseAgg struct {
		events int
		bytes  int64
		byKind map[string]int
	}
	type roleAgg struct {
		events   int
		bytes    int64
		outcomes map[string]int
		returns  []storedReturn
	}
	phases := map[string]*phaseAgg{}
	roles := map[string]*roleAgg{}
	totalBytes := int64(0)
	totalEvents := 0
	bad := 0
	// Each resume's minutes to the next dispatch, in ledger (append) order:
	// "none" when another resume came first, "open" when no dispatch followed.
	var resumeGaps []string
	var pendingResume *time.Time
	// A reader, not a Scanner: one event line carries an unbounded --note. A
	// final line without a newline is handled, then the next read is ("", EOF).
	reader := bufio.NewReader(f)
	for {
		raw, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			fmt.Fprintf(stderr, "metrics: ledger unreadable after %d events: %v\n", totalEvents, readErr)
			return 2
		}
		line := strings.TrimSpace(raw)
		if line == "" {
			if readErr == io.EOF {
				break
			}
			continue
		}
		var ev metricEvent
		if json.Unmarshal([]byte(line), &ev) != nil || ev.Phase == "" || ev.Event == "" {
			bad++
			continue
		}
		agg, ok := phases[ev.Phase]
		if !ok {
			agg = &phaseAgg{byKind: map[string]int{}}
			phases[ev.Phase] = agg
		}
		agg.events++
		agg.bytes += ev.Bytes
		agg.byKind[ev.Event]++
		if ts, err := time.Parse(time.RFC3339, ev.TS); err == nil && (ev.Event == "resume" || ev.Event == "dispatch") {
			if ev.Event == "resume" {
				if pendingResume != nil {
					resumeGaps = append(resumeGaps, "none")
				}
				pendingResume = &ts
			} else if pendingResume != nil {
				resumeGaps = append(resumeGaps, strconv.FormatFloat(ts.Sub(*pendingResume).Minutes(), 'f', 0, 64))
				pendingResume = nil
			}
		}
		if ev.Role != "" {
			ragg, ok := roles[ev.Role]
			if !ok {
				ragg = &roleAgg{outcomes: map[string]int{}}
				roles[ev.Role] = ragg
			}
			ragg.events++
			ragg.bytes += ev.Bytes
			if ev.Event == "return" {
				fields := strings.Fields(ev.Note)
				if len(fields) > 0 {
					if outcome, ok := strings.CutPrefix(fields[0], "outcome="); ok {
						ragg.outcomes[outcome]++
					}
				}
				if rec, ok := parseStoredReturn(fields); ok {
					ragg.returns = append(ragg.returns, rec)
				}
			}
		}
		totalEvents++
		totalBytes += ev.Bytes
	}
	if len(phases) == 0 && bad == 0 {
		fmt.Fprintf(stdout, "metrics: no events recorded for %s\n", slug)
		return 0
	}
	ordered := make([]string, 0, len(phases))
	for phase := range phases {
		ordered = append(ordered, phase)
	}
	sort.Strings(ordered)
	fmt.Fprintf(stdout, "metrics summary: %s\n", slug)
	for _, phase := range ordered {
		agg := phases[phase]
		kinds := make([]string, 0, len(agg.byKind))
		for kind, n := range agg.byKind {
			kinds = append(kinds, fmt.Sprintf("%s=%d", kind, n))
		}
		sort.Strings(kinds)
		fmt.Fprintf(stdout, "  %-10s events=%d bytes=%d (%s)\n", phase, agg.events, agg.bytes, strings.Join(kinds, " "))
	}
	if len(roles) > 0 {
		orderedRoles := make([]string, 0, len(roles))
		for role := range roles {
			orderedRoles = append(orderedRoles, role)
		}
		sort.Strings(orderedRoles)
		fmt.Fprintln(stdout, "  roles:")
		for _, role := range orderedRoles {
			agg := roles[role]
			fmt.Fprintf(stdout, "    %-24s events=%d bytes=%d", role, agg.events, agg.bytes)
			if len(agg.outcomes) > 0 {
				tally := make([]string, 0, len(agg.outcomes))
				for outcome, n := range agg.outcomes {
					tally = append(tally, fmt.Sprintf("%s=%d", outcome, n))
				}
				sort.Strings(tally)
				fmt.Fprintf(stdout, " outcomes: %s", strings.Join(tally, " "))
			}
			if len(agg.returns) > 0 {
				resolved, reraised, open, late := findingResolution(agg.returns)
				fmt.Fprintf(stdout, " resolution: resolved=%d re-raised=%d open=%d late=%d", resolved, reraised, open, late)
			}
			fmt.Fprintln(stdout)
		}
	}
	if pendingResume != nil {
		resumeGaps = append(resumeGaps, "open")
	}
	if len(resumeGaps) > 0 {
		fmt.Fprintf(stdout, "  resumes    count=%d minutes_to_dispatch=%s\n", len(resumeGaps), strings.Join(resumeGaps, ","))
	}
	fmt.Fprintf(stdout, "  total      events=%d bytes=%d", totalEvents, totalBytes)
	if bad > 0 {
		fmt.Fprintf(stdout, " unparsed=%d", bad)
	}
	fmt.Fprintln(stdout)
	return 0
}

// storedReturn is one return recorded with dispatch return --result: its
// slice, Finding: keys, and Late: line count.
type storedReturn struct {
	slice string
	keys  []string
	late  int
}

// parseStoredReturn reads the slice=, late= and finding= tokens of a return
// note; a note without slice= was not recorded with --result.
func parseStoredReturn(fields []string) (storedReturn, bool) {
	var rec storedReturn
	found := false
	for _, field := range fields {
		if v, ok := strings.CutPrefix(field, "slice="); ok {
			rec.slice, found = v, true
		} else if v, ok := strings.CutPrefix(field, "late="); ok {
			rec.late, _ = strconv.Atoi(v)
		} else if v, ok := strings.CutPrefix(field, "finding="); ok {
			rec.keys = append(rec.keys, v)
		}
	}
	return rec, found
}

// findingResolution classifies each raised finding key of one role, in ledger
// order: re-raised when a later return on the same slice raises it again,
// resolved when a later return on that slice does not, open when no later
// return on that slice exists yet.
func findingResolution(returns []storedReturn) (resolved, reraised, open, late int) {
	for i, rec := range returns {
		late += rec.late
		for _, key := range rec.keys {
			rechecked, again := false, false
			for _, next := range returns[i+1:] {
				if next.slice != rec.slice {
					continue
				}
				rechecked = true
				if slices.Contains(next.keys, key) {
					again = true
					break
				}
			}
			switch {
			case again:
				reraised++
			case rechecked:
				resolved++
			default:
				open++
			}
		}
	}
	return resolved, reraised, open, late
}

// sliceCostSummary reports, per slice named on dispatch waves, the build wright
// rounds, slice-bound Vet rechecks, and agent-minutes (role start to return; a
// wave naming several slices splits its minutes evenly). Abandoned waves are
// excluded, and waves recorded before returns were timestamped count rounds only.
func sliceCostSummary(root, slug string, stdout, stderr io.Writer) int {
	f, _, err := loadDispatch(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "metrics: %v\n", err)
		return 2
	}
	type cost struct {
		wright, vet int
		minutes     float64
		timed       bool
	}
	per := map[string]*cost{}
	for _, w := range f.Waves {
		kind := roundKind(w)
		if kind == "" || w.State == "abandoned" {
			continue
		}
		minutes, timed := 0.0, false
		for _, r := range w.Roles {
			start, startErr := time.Parse(time.RFC3339, r.StartedAt)
			end, endErr := time.Parse(time.RFC3339, r.ReturnedAt)
			if startErr == nil && endErr == nil && !end.Before(start) {
				minutes += end.Sub(start).Minutes()
				timed = true
			}
		}
		for _, id := range w.Slices {
			c := per[id]
			if c == nil {
				c = &cost{}
				per[id] = c
			}
			if kind == "wright" {
				c.wright++
			} else {
				c.vet++
			}
			if timed {
				c.minutes += minutes / float64(len(w.Slices))
				c.timed = true
			}
		}
	}
	if len(per) == 0 {
		fmt.Fprintf(stdout, "metrics: no slice-tagged dispatch waves for %s\n", slug)
		return 0
	}
	ids := make([]string, 0, len(per))
	for id := range per {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		c := per[id]
		minutes := "n/a"
		if c.timed {
			minutes = strconv.FormatFloat(c.minutes, 'f', 0, 64)
		}
		fmt.Fprintf(stdout, "metrics: %s wright_rounds=%d vet_rechecks=%d agent_minutes=%s\n", id, c.wright, c.vet, minutes)
	}
	return 0
}
