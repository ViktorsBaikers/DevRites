package lib

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/devritespaths"
)

// MetricsFile is the per-feature append-only event ledger. One JSON object per
// line keeps writes atomic-enough for a single appender and lets summaries
// tolerate trailing partial lines.
const MetricsFile = "metrics.jsonl"

const metricsUsage = `usage: devrites-engine metrics <record|summary> ...

  metrics record <slug> --phase <p> --event <e> [--role r] [--bytes n] [--note s]
  metrics summary [slug]
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
	dir, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return
	}
	entry := metricEvent{
		TS:    time.Now().UTC().Format(time.RFC3339),
		Phase: phase,
		Event: event,
		Role:  role,
		Bytes: bytes,
	}
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

func metricsSummary(root string, args []string, stdout, stderr io.Writer) int {
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
	f, err := os.Open(filepath.Join(dir, MetricsFile)) // #nosec G304 -- dir is a validated workspace and filename is fixed
	if err != nil {
		fmt.Fprintf(stdout, "metrics: no events recorded for %s\n", slug)
		return 0
	}
	defer func() { _ = f.Close() }()

	type phaseAgg struct {
		events int
		bytes  int64
		byKind map[string]int
	}
	type roleAgg struct {
		events int
		bytes  int64
	}
	phases := map[string]*phaseAgg{}
	roles := map[string]*roleAgg{}
	totalBytes := int64(0)
	totalEvents := 0
	bad := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
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
		if ev.Role != "" {
			ragg, ok := roles[ev.Role]
			if !ok {
				ragg = &roleAgg{}
				roles[ev.Role] = ragg
			}
			ragg.events++
			ragg.bytes += ev.Bytes
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
			fmt.Fprintf(stdout, "    %-24s events=%d bytes=%d\n", role, agg.events, agg.bytes)
		}
	}
	fmt.Fprintf(stdout, "  total      events=%d bytes=%d", totalEvents, totalBytes)
	if bad > 0 {
		fmt.Fprintf(stdout, " unparsed=%d", bad)
	}
	fmt.Fprintln(stdout)
	return 0
}
