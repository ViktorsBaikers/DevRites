package lib

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/state"
)

type timelineEntry struct {
	TS       string `json:"ts"`
	Event    string `json:"event"`
	Skill    string `json:"skill,omitempty"`
	Slug     string `json:"slug,omitempty"`
	Outcome  string `json:"outcome,omitempty"`
	Decision string `json:"decision,omitempty"`
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
	Note     string `json:"note,omitempty"`
}

type healthEntry struct {
	TS    string  `json:"ts"`
	Score float64 `json:"score"`
	Label string  `json:"label"`
	Note  string  `json:"note,omitempty"`
}

type reviewFingerprint struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Text     string `json:"text"`
}

// Timeline writes privacy-bounded v1 trace facts, reads a bounded local report,
// and purges only exact telemetry selections. Events never own lifecycle state.
func Timeline(root string, args []string, stdout, stderr io.Writer) int {
	switch argAt(args, 0) {
	case "log":
		entry := timelineEntry{TS: nowUTC()}
		var (
			mode     ExecutionMode
			strength GuardStrength
			reasonID reason.ID
			host     = HostEngine
			ruleIDs  []reason.ID
			evidence []string
			typed    bool
		)
		rest := args[1:]
		if len(rest) > 0 && !strings.HasPrefix(rest[0], "--") {
			entry.Event = rest[0]
			rest = rest[1:]
		}
		for i := 0; i < len(rest); i++ {
			next := func(name string) (string, bool) {
				i++
				if i >= len(rest) {
					fmt.Fprintf(stderr, "timeline: %s needs a value\n", name)
					return "", false
				}
				return rest[i], true
			}
			switch rest[i] {
			case "--event":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.Event = v
			case "--skill":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.Skill = v
			case "--slug":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.Slug = v
			case "--outcome":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.Outcome = v
			case "--decision":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.Decision = v
			case "--from":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.From = v
			case "--to":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.To = v
			case "--note":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.Note = v
			case "--execution-mode":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				mode, typed = ExecutionMode(v), true
			case "--guard-strength":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				strength, typed = GuardStrength(v), true
			case "--reason-id":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				parsed, err := reason.Parse(v)
				if err != nil {
					fmt.Fprintf(stderr, "timeline: %v\n", err)
					return 2
				}
				reasonID, typed = parsed, true
			case "--rule-id":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				parsed, err := reason.Parse(v)
				if err != nil {
					fmt.Fprintf(stderr, "timeline: %v\n", err)
					return 2
				}
				ruleIDs, typed = append(ruleIDs, parsed), true
			case "--evidence":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				evidence, typed = append(evidence, v), true
			case "--host":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				host, typed = EventHost(v), true
			default:
				fmt.Fprintf(stderr, "timeline: unknown option %s\n", rest[i])
				return 2
			}
		}
		if entry.Event == "" {
			fmt.Fprintln(stderr, timelineLogUsage)
			return 2
		}
		if !typed {
			fmt.Fprintln(stderr, "timeline: legacy free-text writes are disabled; provide typed v1 mode, guard, and reason facts")
			return 2
		}
		if mode == "" || strength == "" || reasonID == "" {
			fmt.Fprintln(stderr, "timeline: typed records require --execution-mode, --guard-strength, and --reason-id")
			return 2
		}
		if entry.Skill != "" || entry.Decision != "" || entry.Note != "" {
			fmt.Fprintln(stderr, "timeline: typed records do not retain --skill, --decision, or --note free text")
			return 2
		}
		ev := NewEventV1(BoundaryAgentDispatch, entry.Event, reasonID)
		ev.ExecutionMode = mode
		ev.GuardStrength = strength
		ev.Host = host
		ev.Outcome = OutcomeRecorded
		if entry.Outcome != "" {
			ev.Outcome = EventOutcome(entry.Outcome)
		}
		if entry.From != "" {
			ev.PhaseBefore = state.Phase(entry.From)
		}
		if entry.To != "" {
			ev.PhaseAfter = state.Phase(entry.To)
		}
		ev.RuleIDs = append(ev.RuleIDs, ruleIDs...)
		ev.EvidencePaths = append([]string(nil), evidence...)
		if err := BindEventWorkspace(root, entry.Slug, &ev); err != nil {
			fmt.Fprintf(stderr, "timeline: %v\n", err)
			return 2
		}
		if err := ValidateEventV1(ev); err != nil {
			fmt.Fprintf(stderr, "timeline: %v\n", err)
			return 2
		}
		if err := AppendEventV1(root, ev); err != nil {
			fmt.Fprintf(stderr, "timeline: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "timeline: recorded.")
		return 0
	case "list":
		limit := 20
		if argAt(args, 1) == "--limit" {
			n, err := strconv.Atoi(argAt(args, 2))
			if err != nil || n < 0 {
				fmt.Fprintln(stderr, "timeline: --limit must be a non-negative integer")
				return 2
			}
			limit = n
		}
		printTail(filepath.Join(root, "timeline.jsonl"), limit, "timeline", stdout)
		return 0
	case "report":
		return timelineReport(root, args[1:], stdout, stderr)
	case "purge":
		return timelinePurge(root, args[1:], stdout, stderr)
	default:
		fmt.Fprintln(stderr, "usage: devrites-engine timeline log|list|report|purge [...]")
		return 2
	}
}

const timelineLogUsage = "usage: devrites-engine timeline log <event> [--slug S] [--outcome O] [--from A --to B] --execution-mode named|generic|inline|none --guard-strength enforced|observed|unavailable|bypassed|n/a --reason-id DRV-* [--host engine|claude|codex] [--rule-id DRV-*] [--evidence PATH]"

const (
	telemetryTailBytes        int64 = 4 << 20
	telemetryMaxTailEvents          = 4096
	telemetryStatusTailBytes  int64 = 256 << 10
	telemetryStatusTailEvents       = 256
	telemetryMaxLineBytes           = 64 << 10
	telemetryMaxStorageBytes  int64 = 16 << 20
	telemetryMaxPurgeBytes    int64 = 16 << 20
)

const observabilityReportSchema = "devrites-observability/v1"

type telemetryReadStats struct {
	LegacyIgnored  int  `json:"legacy_ignored"`
	CorruptIgnored int  `json:"corrupt_ignored"`
	TailTruncated  bool `json:"tail_truncated"`
	ReadDegraded   bool `json:"read_degraded"`
}

type observabilityReport struct {
	Schema               string             `json:"schema"`
	RunID                string             `json:"run_id,omitempty"`
	Workspace            string             `json:"workspace,omitempty"`
	Events               int                `json:"events"`
	TraceStarted         bool               `json:"trace_started"`
	TraceFinished        bool               `json:"trace_finished"`
	StartedAt            string             `json:"started_at,omitempty"`
	FinishedAt           string             `json:"finished_at,omitempty"`
	DurationSeconds      int64              `json:"duration_seconds"`
	PhaseDurations       map[string]int64   `json:"phase_durations_seconds"`
	Retries              int                `json:"retries"`
	HumanWaits           int                `json:"human_waits"`
	Interruptions        int                `json:"interruptions"`
	Resumes              int                `json:"resumes"`
	LinkedResumes        int                `json:"linked_resumes"`
	OpenInterruptions    int                `json:"open_interruptions"`
	ExecutionMode        ExecutionMode      `json:"execution_mode,omitempty"`
	GuardStrength        GuardStrength      `json:"guard_strength,omitempty"`
	Host                 EventHost          `json:"host,omitempty"`
	LastFailedGateReason reason.ID          `json:"last_failed_gate_reason,omitempty"`
	LastFailedGatePhase  state.Phase        `json:"last_failed_gate_phase,omitempty"`
	StaleEvidence        int                `json:"stale_evidence"`
	Degradations         int                `json:"degradations"`
	FinalOutcome         string             `json:"final_outcome"`
	TelemetryRead        telemetryReadStats `json:"telemetry_read"`
}

func timelineReport(root string, args []string, stdout, stderr io.Writer) int {
	runID := ""
	wantJSON := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			wantJSON = true
		case "--run":
			i++
			if i >= len(args) || runID != "" {
				fmt.Fprintln(stderr, "timeline report: --run needs one opaque run id")
				return 2
			}
			runID = args[i]
			if !runIDToken.MatchString(runID) {
				fmt.Fprintln(stderr, "timeline report: --run must be an opaque drv-run-v1 token")
				return 2
			}
		default:
			fmt.Fprintf(stderr, "timeline report: unknown option %s\n", args[i])
			return 2
		}
	}

	events, stats := readTelemetryTail(filepath.Join(root, "timeline.jsonl"))
	report := summarizeObservability(events, stats, runID)
	if wantJSON {
		if err := json.NewEncoder(stdout).Encode(report); err != nil {
			fmt.Fprintf(stderr, "timeline report: %v\n", err)
			return 1
		}
		return 0
	}
	printObservabilityReport(report, stdout)
	return 0
}

func readTelemetryTail(path string) ([]EventV1, telemetryReadStats) {
	return readTelemetryTailBounded(path, telemetryTailBytes, telemetryMaxTailEvents)
}

func readTelemetryTailBounded(path string, maxBytes int64, maxEvents int) ([]EventV1, telemetryReadStats) {
	var stats telemetryReadStats
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, stats
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		stats.ReadDegraded = true
		return nil, stats
	}
	f, err := os.Open(path)
	if err != nil {
		stats.ReadDegraded = true
		return nil, stats
	}
	defer f.Close()

	start := int64(0)
	if info.Size() > maxBytes {
		start = info.Size() - maxBytes
		stats.TailTruncated = true
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		stats.ReadDegraded = true
		return nil, stats
	}
	reader := bufio.NewReader(io.LimitReader(f, maxBytes))
	if start > 0 {
		for {
			_, err := reader.ReadSlice('\n')
			if err == bufio.ErrBufferFull {
				continue
			}
			if err != nil && err != io.EOF {
				stats.ReadDegraded = true
			}
			break
		}
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), telemetryMaxLineBytes)
	ring := make([]EventV1, maxEvents)
	total := 0
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var envelope struct {
			Schema string `json:"schema"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			stats.CorruptIgnored++
			continue
		}
		if envelope.Schema != EventSchemaV1 {
			stats.LegacyIgnored++
			continue
		}
		var event EventV1
		if err := json.Unmarshal(line, &event); err != nil || ValidateEventV1(event) != nil {
			stats.CorruptIgnored++
			continue
		}
		ring[total%maxEvents] = event
		total++
	}
	if scanner.Err() != nil {
		stats.ReadDegraded = true
	}
	if total > maxEvents {
		stats.TailTruncated = true
		out := make([]EventV1, maxEvents)
		cursor := total % maxEvents
		for i := range out {
			out[i] = ring[(cursor+i)%maxEvents]
		}
		return out, stats
	}
	return append([]EventV1(nil), ring[:total]...), stats
}

func summarizeObservability(events []EventV1, stats telemetryReadStats, runID string) observabilityReport {
	report := observabilityReport{
		Schema:         observabilityReportSchema,
		PhaseDurations: map[string]int64{},
		FinalOutcome:   "unfinished",
		TelemetryRead:  stats,
	}
	if runID == "" {
		runID = latestObservedRun(events)
	}
	report.RunID = runID
	if runID == "" {
		return report
	}

	selected := make([]EventV1, 0, len(events))
	for _, event := range events {
		if event.RunID == runID {
			selected = append(selected, event)
		}
	}
	sort.SliceStable(selected, func(i, j int) bool {
		left, _ := time.Parse(time.RFC3339, selected[i].TS)
		right, _ := time.Parse(time.RFC3339, selected[j].TS)
		return left.Before(right)
	})
	for i, event := range selected {
		if event.Event == "run-started" {
			selected = selected[i:]
			break
		}
	}
	report.Events = len(selected)
	if len(selected) == 0 {
		return report
	}

	firstAt, _ := time.Parse(time.RFC3339, selected[0].TS)
	lastAt := firstAt
	report.StartedAt = selected[0].TS
	for i, event := range selected {
		at, _ := time.Parse(time.RFC3339, event.TS)
		if at.After(lastAt) {
			lastAt = at
		}
		if event.Workspace != "" {
			report.Workspace = event.Workspace
		}
		if event.Event == "run-started" && !report.TraceStarted {
			report.TraceStarted = true
			report.StartedAt = event.TS
			firstAt = at
		}
		if event.Event == "run-finished" {
			report.TraceFinished = true
			report.FinishedAt = event.TS
			report.FinalOutcome = string(event.Outcome)
		}
		if event.ExecutionMode != ExecutionNone {
			report.ExecutionMode = event.ExecutionMode
		}
		if event.GuardStrength != GuardNA {
			report.GuardStrength = event.GuardStrength
		}
		if event.Host != "" {
			report.Host = event.Host
		}
		switch event.Event {
		case "retry", "recovery-retry":
			report.Retries++
		case "evidence-stale":
			report.StaleEvidence++
		}
		humanWait := event.Event == "human-wait-started" ||
			event.Event == "human-wait-resumed" ||
			event.ReasonID == reason.HookStopUnsurfacedHumanGate
		if humanWait {
			report.HumanWaits++
		}
		if event.Event == "run-interrupted" ||
			(event.ReasonID == reason.HookStopUnsurfacedHumanGate && event.Event != "human-wait-resumed") {
			report.Interruptions++
		}
		if event.Event == "run-resumed" || event.Event == "human-wait-resumed" {
			report.Resumes++
		}
		if event.Boundary == BoundaryLifecycleGate && failedEventOutcome(event.Outcome) {
			report.LastFailedGateReason = event.ReasonID
			report.LastFailedGatePhase = eventPhase(event)
		}
		if degradedEvent(event) {
			report.Degradations++
		}
		if i+1 < len(selected) {
			nextAt, _ := time.Parse(time.RFC3339, selected[i+1].TS)
			if nextAt.After(at) {
				if phase := eventPhase(event); phase != "" {
					report.PhaseDurations[string(phase)] += int64(nextAt.Sub(at) / time.Second)
				}
			}
		}
	}
	report.DurationSeconds = int64(lastAt.Sub(firstAt) / time.Second)
	report.LinkedResumes = min(report.Interruptions, report.Resumes)
	report.OpenInterruptions = report.Interruptions - report.LinkedResumes
	return report
}

func latestObservedRun(events []EventV1) string {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Event == "run-started" {
			return events[i].RunID
		}
	}
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Boundary != BoundaryRootSelection {
			return events[i].RunID
		}
	}
	return ""
}

func eventPhase(event EventV1) state.Phase {
	if event.PhaseAfter != "" {
		return event.PhaseAfter
	}
	return event.PhaseBefore
}

func failedEventOutcome(outcome EventOutcome) bool {
	return outcome == OutcomeBlocked || outcome == OutcomeDenied ||
		outcome == OutcomeUnavailable || outcome == OutcomeWarning
}

func degradedEvent(event EventV1) bool {
	return event.Event == "degraded" ||
		event.ExecutionMode == ExecutionGeneric ||
		event.ExecutionMode == ExecutionInline ||
		event.GuardStrength == GuardUnavailable ||
		event.GuardStrength == GuardBypassed ||
		event.Outcome == OutcomeUnavailable ||
		event.Outcome == OutcomeBypassed
}

func printObservabilityReport(report observabilityReport, stdout io.Writer) {
	if report.Events == 0 {
		fmt.Fprintf(stdout,
			"timeline report: no valid %s rows in the bounded tail (legacy:%d corrupt:%d degraded:%t truncated:%t).\n",
			EventSchemaV1, report.TelemetryRead.LegacyIgnored, report.TelemetryRead.CorruptIgnored,
			report.TelemetryRead.ReadDegraded, report.TelemetryRead.TailTruncated)
		return
	}
	fmt.Fprintf(stdout, "timeline report: run %s · %d v1 events · duration %s\n",
		report.RunID, report.Events, (time.Duration(report.DurationSeconds) * time.Second).String())
	if len(report.PhaseDurations) > 0 {
		phases := make([]string, 0, len(report.PhaseDurations))
		for phase := range report.PhaseDurations {
			phases = append(phases, phase)
		}
		sort.Strings(phases)
		fmt.Fprint(stdout, "phases:")
		for _, phase := range phases {
			fmt.Fprintf(stdout, " %s=%s", phase, (time.Duration(report.PhaseDurations[phase]) * time.Second).String())
		}
		fmt.Fprintln(stdout)
	}
	fmt.Fprintf(stdout, "recovery: retries:%d waits:%d interruptions:%d resumes:%d linked:%d open:%d\n",
		report.Retries, report.HumanWaits, report.Interruptions, report.Resumes,
		report.LinkedResumes, report.OpenInterruptions)
	fmt.Fprintf(stdout, "active: mode:%s guard:%s host:%s\n",
		defaultText(string(report.ExecutionMode), "none"),
		defaultText(string(report.GuardStrength), "n/a"),
		defaultText(string(report.Host), "unknown"))
	if report.LastFailedGateReason != "" {
		fmt.Fprintf(stdout, "last failed gate: %s phase:%s\n",
			report.LastFailedGateReason, defaultText(string(report.LastFailedGatePhase), "unknown"))
	} else {
		fmt.Fprintln(stdout, "last failed gate: none")
	}
	fmt.Fprintf(stdout, "evidence: stale:%d · degradation:%d · outcome:%s\n",
		report.StaleEvidence, report.Degradations, report.FinalOutcome)
	fmt.Fprintf(stdout, "compatibility: legacy:%d corrupt:%d ignored · degraded:%t truncated:%t\n",
		report.TelemetryRead.LegacyIgnored, report.TelemetryRead.CorruptIgnored,
		report.TelemetryRead.ReadDegraded, report.TelemetryRead.TailTruncated)
}

func defaultText(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// ObservabilityStatus derives compact display-only facts from the same v1 rows
// used by timeline report. It never advances or overrides lifecycle state.
func ObservabilityStatus(root, slug string) string {
	if slug == "" {
		return ""
	}
	events, stats := readTelemetryTailBounded(
		filepath.Join(root, "timeline.jsonl"),
		telemetryStatusTailBytes,
		telemetryStatusTailEvents,
	)
	work := ".devrites/work/" + slug
	legacy := ".devrites/features/" + slug
	filtered := make([]EventV1, 0, len(events))
	for _, event := range events {
		if event.Workspace == work || event.Workspace == legacy {
			filtered = append(filtered, event)
		}
	}
	report := summarizeObservability(filtered, stats, "")
	if report.Events == 0 {
		return ""
	}
	parts := []string{"obs"}
	if report.ExecutionMode != "" {
		parts = append(parts, "mode:"+string(report.ExecutionMode))
	}
	if report.HumanWaits > 0 {
		parts = append(parts, "waits:"+strconv.Itoa(report.HumanWaits))
	}
	if report.LastFailedGateReason != "" {
		parts = append(parts, "gate:"+string(report.LastFailedGateReason))
	}
	if report.StaleEvidence > 0 {
		parts = append(parts, "stale:"+strconv.Itoa(report.StaleEvidence))
	}
	if report.Degradations > 0 {
		parts = append(parts, "degraded:"+strconv.Itoa(report.Degradations))
	}
	if report.FinalOutcome != "unfinished" {
		parts = append(parts, "outcome:"+report.FinalOutcome)
	}
	if len(parts) == 1 {
		return ""
	}
	return strings.Join(parts, " ")
}

func timelinePurge(root string, args []string, stdout, stderr io.Writer) int {
	var (
		before *time.Time
		runID  string
	)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--before":
			i++
			if i >= len(args) || before != nil {
				fmt.Fprintln(stderr, "timeline purge: --before needs one RFC3339 timestamp")
				return 2
			}
			parsed, err := time.Parse(time.RFC3339, args[i])
			if err != nil {
				fmt.Fprintln(stderr, "timeline purge: --before must be RFC3339")
				return 2
			}
			before = &parsed
		case "--run":
			i++
			if i >= len(args) || runID != "" {
				fmt.Fprintln(stderr, "timeline purge: --run needs one opaque run id")
				return 2
			}
			runID = args[i]
			if !runIDToken.MatchString(runID) {
				fmt.Fprintln(stderr, "timeline purge: --run must be an opaque drv-run-v1 token")
				return 2
			}
		default:
			fmt.Fprintf(stderr, "timeline purge: unknown option %s\n", args[i])
			return 2
		}
	}
	if before == nil && runID == "" {
		fmt.Fprintln(stderr, "timeline purge: provide --before and/or --run")
		return 2
	}

	targets, err := telemetryTargets(root)
	if err != nil {
		fmt.Fprintf(stderr, "timeline purge: %v\n", err)
		return 1
	}
	removed, changed := 0, 0
	for _, target := range targets {
		n, err := purgeTelemetryFile(root, target, func(event EventV1) bool {
			if runID != "" && event.RunID != runID {
				return false
			}
			if before != nil {
				at, _ := time.Parse(time.RFC3339, event.TS)
				if !at.Before(*before) {
					return false
				}
			}
			return true
		})
		if err != nil {
			fmt.Fprintf(stderr, "timeline purge: %v\n", err)
			return 1
		}
		if n > 0 {
			changed++
			removed += n
		}
	}
	fmt.Fprintf(stdout, "timeline: purged %d v1 event(s) from %d telemetry file(s).\n", removed, changed)
	return 0
}

func telemetryTargets(root string) ([]string, error) {
	targets := []string{filepath.Join(root, "timeline.jsonl")}
	for _, owner := range []string{"work", "features"} {
		base := filepath.Join(root, owner)
		info, err := os.Lstat(base)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect telemetry owner %s: %w", owner, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, fmt.Errorf("telemetry owner %s is not a real directory", owner)
		}
		entries, err := os.ReadDir(base)
		if err != nil {
			return nil, fmt.Errorf("read telemetry owner %s: %w", owner, err)
		}
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() ||
				!slugToken.MatchString(entry.Name()) {
				continue
			}
			targets = append(targets, filepath.Join(base, entry.Name(), "events.jsonl"))
		}
	}
	return targets, nil
}

func purgeTelemetryFile(root, path string, remove func(EventV1) bool) (int, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("inspect %s: %w", filepath.Base(path), err)
	}
	if err := validateEventLineTarget(path, root); err != nil {
		return 0, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return 0, fmt.Errorf("%s is not a regular telemetry file", filepath.Base(path))
	}
	if info.Size() > telemetryMaxPurgeBytes {
		return 0, fmt.Errorf("%s exceeds the 16 MiB purge bound; delete this telemetry file manually", filepath.Base(path))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), telemetryMaxLineBytes)
	var out bytes.Buffer
	out.Grow(len(data))
	removed := 0
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		drop := false
		var envelope struct {
			Schema string `json:"schema"`
		}
		if json.Unmarshal(line, &envelope) == nil && envelope.Schema == EventSchemaV1 {
			var event EventV1
			if json.Unmarshal(line, &event) == nil && ValidateEventV1(event) == nil {
				drop = remove(event)
			}
		}
		if drop {
			removed++
			continue
		}
		out.Write(line)
		out.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("%s contains an oversized telemetry row; original preserved", filepath.Base(path))
	}
	if removed == 0 {
		return 0, nil
	}
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, current) ||
		current.Size() != info.Size() || !current.ModTime().Equal(info.ModTime()) {
		return 0, fmt.Errorf("%s changed during purge; original preserved", filepath.Base(path))
	}
	if err := fsutil.WriteFileAtomic(path, out.Bytes(), info.Mode().Perm()); err != nil {
		return 0, fmt.Errorf("rewrite %s: %w", filepath.Base(path), err)
	}
	return removed, nil
}

// Health records one compact score (legacy record/list) or runs the project code-health dashboard.
func Health(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || argAt(args, 0) == "run" || argAt(args, 0) == "check" {
		return CodeHealth(root, args, stdout, stderr)
	}
	switch argAt(args, 0) {
	case "record":
		score, ok := parseHealthScore(argAt(args, 1))
		if !ok {
			fmt.Fprintln(stderr, "usage: devrites-engine health record <score 0..10> <label> [--note N]")
			return 2
		}
		labelParts := []string{}
		note := ""
		for i := 2; i < len(args); i++ {
			if args[i] == "--note" {
				i++
				if i >= len(args) {
					fmt.Fprintln(stderr, "health: --note needs a value")
					return 2
				}
				note = args[i]
				continue
			}
			labelParts = append(labelParts, args[i])
		}
		label := strings.TrimSpace(strings.Join(labelParts, " "))
		if label == "" {
			fmt.Fprintln(stderr, "usage: devrites-engine health record <score 0..10> <label> [--note N]")
			return 2
		}
		if err := appendJSONLine(filepath.Join(root, "health-history.jsonl"), healthEntry{
			TS:    nowUTC(),
			Score: score,
			Label: label,
			Note:  note,
		}); err != nil {
			fmt.Fprintf(stderr, "health: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "health: recorded.")
		return 0
	case "list":
		limit := 20
		if argAt(args, 1) == "--limit" {
			n, err := strconv.Atoi(argAt(args, 2))
			if err != nil || n < 0 {
				fmt.Fprintln(stderr, "health: --limit must be a non-negative integer")
				return 2
			}
			limit = n
		}
		path := filepath.Join(root, "health.jsonl")
		if !isFile(path) {
			path = filepath.Join(root, "health-history.jsonl")
		}
		printTail(path, limit, "health", stdout)
		return 0
	default:
		fmt.Fprintln(stderr, "usage: devrites-engine health [run|check] | record|list [...]")
		return 2
	}
}

func ReviewFingerprints(root string, args []string, stdout, stderr io.Writer) int {
	write := false
	slug := ""
	for _, a := range args {
		if a == "--write" {
			write = true
			continue
		}
		if slug == "" {
			slug = a
		}
	}
	if slug == "" {
		slug = activeSlug(root)
	}
	if slug == "" {
		fmt.Fprintln(stderr, "usage: devrites-engine review-fingerprints [--write] <slug>")
		return 2
	}
	dir := featureDir(root, slug)
	review, ok := readFileOK(filepath.Join(dir, "review.md"))
	if !ok {
		fmt.Fprintln(stdout, "review-fingerprints: no review.md: nothing to fingerprint.")
		return 0
	}
	records := reviewFindingFingerprints(review)
	if len(records) == 0 {
		fmt.Fprintln(stdout, "review-fingerprints: no findings.")
		return 0
	}
	var jsonl strings.Builder
	for _, r := range records {
		fmt.Fprintf(stdout, "%s %s %s\n", r.ID, r.Severity, r.Text)
		if b, err := json.Marshal(r); err == nil {
			jsonl.Write(b)
			jsonl.WriteByte('\n')
		}
	}
	if write {
		if err := fsutil.WriteFileAtomic(filepath.Join(dir, "review-fingerprints.jsonl"), []byte(jsonl.String()), 0o644); err != nil {
			fmt.Fprintf(stderr, "review-fingerprints: %v\n", err)
			return 1
		}
	}
	return 0
}

func reviewFindingFingerprints(md string) []reviewFingerprint {
	var out []reviewFingerprint
	for _, line := range markdownLines(md) {
		m := findingLabel.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		text := normalizeFindingLine(line)
		sum := sha256.Sum256([]byte(strings.ToLower(m[1]) + "\n" + text))
		out = append(out, reviewFingerprint{
			ID:       hex.EncodeToString(sum[:])[:12],
			Severity: m[1],
			Text:     text,
		})
	}
	return out
}

func normalizeFindingLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "-")
	line = strings.TrimSpace(line)
	return strings.Join(strings.Fields(line), " ")
}

func markdownLines(md string) []string {
	md = strings.TrimRight(md, "\n")
	if md == "" {
		return nil
	}
	return strings.Split(md, "\n")
}

func parseHealthScore(s string) (float64, bool) {
	score, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || score < 0 || score > 10 {
		return 0, false
	}
	return score, true
}

func appendJSONLine(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}
	if telemetryLogPath(path) {
		if info, err := os.Stat(path); err == nil {
			if info.Size()+int64(len(b))+1 > telemetryMaxStorageBytes {
				return fmt.Errorf("telemetry log reached the 16 MiB local retention bound; run timeline purge")
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect telemetry log: %w", err)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("append log: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("append log: %w", err)
	}
	return nil
}

func telemetryLogPath(path string) bool {
	base := filepath.Base(path)
	return base == "timeline.jsonl" || base == "events.jsonl"
}

func printTail(path string, limit int, label string, stdout io.Writer) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stdout, "%s: no history at %s.\n", label, path)
		return
	}
	lines := splitLinesNoTrailing(data)
	if limit > 0 && len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	for _, line := range lines {
		fmt.Fprintln(stdout, line)
	}
}
