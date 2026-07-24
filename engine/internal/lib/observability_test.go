package lib

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/state"
)

const observabilityOtherRunID = "drv-run-v1:fedcba9876543210fedcba9876543210"

func makeObservabilityWorkspace(t *testing.T) (string, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".devrites")
	work := filepath.Join(root, "work", "demo")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte("demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, work
}

func appendObservabilityEvent(
	t *testing.T,
	root, runID, timestamp, name string,
	boundary EventBoundary,
	reasonID reason.ID,
	outcome EventOutcome,
	phase state.Phase,
) EventV1 {
	t.Helper()
	event := NewEventV1(boundary, name, reasonID)
	event.TS = timestamp
	event.RunID = runID
	event.RootSource = RootSourceExplicit
	event.Workspace = ".devrites/work/demo"
	event.PhaseBefore = phase
	event.PhaseAfter = phase
	event.Outcome = outcome
	event.Host = HostCodex
	if err := AppendEventV1(root, event); err != nil {
		t.Fatal(err)
	}
	return event
}

func TestTimelineReportAnswersWorkflowQuestionsWithoutRawText(t *testing.T) {
	root, _ := makeObservabilityWorkspace(t)
	legacy := `{"ts":"2026-07-23T11:59:00Z","event":"auq","question":"PRIVATE_QUESTION","answer":"PRIVATE_ANSWER"}` + "\n"
	if err := os.WriteFile(
		filepath.Join(root, "timeline.jsonl"),
		[]byte(legacy+"not-json\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	start := appendObservabilityEvent(t, root, testRunID, "2026-07-23T12:00:00Z",
		"run-started", BoundaryAgentDispatch, reason.RootSelected, OutcomeRecorded, state.PhaseSpec)
	start.ExecutionMode = ExecutionNamed
	start.GuardStrength = GuardObserved
	rewriteLastObservabilityEvent(t, root, start)

	appendObservabilityEvent(t, root, testRunID, "2026-07-23T12:01:00Z",
		"retry", BoundaryAgentDispatch, reason.AgentResultMalformed, OutcomeWarning, state.PhaseBuild)
	appendObservabilityEvent(t, root, testRunID, "2026-07-23T12:02:00Z",
		"readiness", BoundaryLifecycleGate, reason.GateReadinessMissing, OutcomeBlocked, state.PhaseBuild)
	appendObservabilityEvent(t, root, testRunID, "2026-07-23T12:03:00Z",
		"run-interrupted", BoundaryHookGuard, reason.HookStopUnsurfacedHumanGate, OutcomeBlocked, state.PhaseBuild)
	appendObservabilityEvent(t, root, testRunID, "2026-07-23T12:04:00Z",
		"run-resumed", BoundaryAgentDispatch, reason.HookAllowApproved, OutcomeAllowed, state.PhaseBuild)
	appendObservabilityEvent(t, root, testRunID, "2026-07-23T12:05:00Z",
		"evidence-stale", BoundaryLifecycleGate, reason.GateReadinessMissing, OutcomeWarning, state.PhaseProve)
	degraded := appendObservabilityEvent(t, root, testRunID, "2026-07-23T12:06:00Z",
		"degraded", BoundaryAgentDispatch, reason.AgentUnavailable, OutcomeUnavailable, state.PhaseProve)
	degraded.ExecutionMode = ExecutionGeneric
	degraded.GuardStrength = GuardUnavailable
	rewriteLastObservabilityEvent(t, root, degraded)
	appendObservabilityEvent(t, root, testRunID, "2026-07-23T12:07:00Z",
		"run-finished", BoundaryLifecycleGate, reason.GateSealMissing, OutcomeBlocked, state.PhaseSeal)

	var stdout, stderr bytes.Buffer
	if code := Timeline(root, []string{"report", "--run", testRunID, "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("report code=%d stderr=%q", code, stderr.String())
	}
	var report observabilityReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Schema != observabilityReportSchema || report.Events != 8 ||
		!report.TraceStarted || !report.TraceFinished || report.DurationSeconds != 420 {
		t.Fatalf("trace summary=%+v", report)
	}
	if report.Retries != 1 || report.HumanWaits != 1 ||
		report.Interruptions != 1 || report.Resumes != 1 ||
		report.LinkedResumes != 1 || report.OpenInterruptions != 0 {
		t.Fatalf("recovery summary=%+v", report)
	}
	if report.LastFailedGateReason != reason.GateSealMissing ||
		report.LastFailedGatePhase != state.PhaseSeal ||
		report.StaleEvidence != 1 || report.Degradations != 1 ||
		report.FinalOutcome != string(OutcomeBlocked) {
		t.Fatalf("gate/outcome summary=%+v", report)
	}
	if report.PhaseDurations[string(state.PhaseSpec)] != 60 ||
		report.PhaseDurations[string(state.PhaseBuild)] != 240 ||
		report.PhaseDurations[string(state.PhaseProve)] != 120 {
		t.Fatalf("phase durations=%v", report.PhaseDurations)
	}
	if report.TelemetryRead.LegacyIgnored != 1 || report.TelemetryRead.CorruptIgnored != 1 {
		t.Fatalf("compatibility summary=%+v", report.TelemetryRead)
	}
	if strings.Contains(stdout.String(), "PRIVATE_QUESTION") ||
		strings.Contains(stdout.String(), "PRIVATE_ANSWER") {
		t.Fatalf("report exposed legacy raw text: %s", stdout.String())
	}
}

// appendObservabilityEvent writes both telemetry owners. Tests occasionally
// need to change typed fields after construction, so replace only the final
// matching row in each owner with another validated v1 row.
func rewriteLastObservabilityEvent(t *testing.T, root string, event EventV1) {
	t.Helper()
	if err := ValidateEventV1(event); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(root, "timeline.jsonl"),
		filepath.Join(root, "work", "demo", "events.jsonl"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		lines := bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n"))
		lines[len(lines)-1] = encoded
		if err := os.WriteFile(path, append(bytes.Join(lines, []byte("\n")), '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTimelineReportBoundsTailAndDegradesOnOversizedRows(t *testing.T) {
	root, _ := makeObservabilityWorkspace(t)
	event := appendObservabilityEvent(t, root, testRunID, "2026-07-23T12:00:00Z",
		"run-started", BoundaryAgentDispatch, reason.RootSelected, OutcomeRecorded, state.PhaseBuild)
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	prefix := bytes.Repeat([]byte("PRIVATE_OLD_DATA"), int(telemetryTailBytes/16)+100)
	data := append(prefix, '\n')
	data = append(data, encoded...)
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(root, "timeline.jsonl"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	events, stats := readTelemetryTail(filepath.Join(root, "timeline.jsonl"))
	if len(events) != 1 || !stats.TailTruncated || stats.ReadDegraded {
		t.Fatalf("bounded tail events=%d stats=%+v", len(events), stats)
	}

	data = append(encoded, '\n')
	data = append(data, bytes.Repeat([]byte("x"), telemetryMaxLineBytes+1)...)
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(root, "timeline.jsonl"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Timeline(root, []string{"report", "--run", testRunID, "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("oversized report code=%d stderr=%q", code, stderr.String())
	}
	var report observabilityReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Events != 1 || !report.TelemetryRead.ReadDegraded {
		t.Fatalf("oversized row did not degrade report only: %+v", report)
	}
}

func TestTelemetryAppendStopsAtLocalStorageBound(t *testing.T) {
	root, _ := makeObservabilityWorkspace(t)
	path := filepath.Join(root, "timeline.jsonl")
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), int(telemetryMaxStorageBytes)), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	event := NewEventV1(BoundaryAgentDispatch, "run-started", reason.RootSelected)
	event.TS = "2026-07-23T12:00:00Z"
	event.RunID = testRunID
	event.RootSource = RootSourceExplicit
	event.Workspace = ".devrites/work/demo"
	event.Outcome = OutcomeRecorded
	if err := AppendEventV1(root, event); err == nil ||
		!strings.Contains(err.Error(), "retention bound") {
		t.Fatalf("bounded append err=%v", err)
	}
	after, err := os.Stat(path)
	if err != nil || after.Size() != before.Size() {
		t.Fatalf("bounded append changed file: before=%d after=%d err=%v", before.Size(), after.Size(), err)
	}
}

func TestTimelinePurgeTouchesOnlySelectedV1Telemetry(t *testing.T) {
	root, work := makeObservabilityWorkspace(t)
	appendObservabilityEvent(t, root, testRunID, "2026-07-23T12:00:00Z",
		"run-started", BoundaryAgentDispatch, reason.RootSelected, OutcomeRecorded, state.PhaseBuild)
	appendObservabilityEvent(t, root, observabilityOtherRunID, "2026-07-23T13:00:00Z",
		"run-started", BoundaryAgentDispatch, reason.RootSelected, OutcomeRecorded, state.PhaseBuild)
	legacy := []byte(`{"ts":"2026-07-23T10:00:00Z","event":"auq","question":"legacy stays"}` + "\n")
	for _, path := range []string{
		filepath.Join(root, "timeline.jsonl"),
		filepath.Join(work, "events.jsonl"),
	} {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(legacy); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	correctness := map[string][]byte{
		"state.md":                []byte("state-sentinel\n"),
		"questions.md":            []byte("questions-sentinel\n"),
		"evidence.md":             []byte("evidence-sentinel\n"),
		"recovery-attempts.jsonl": []byte("recovery-sentinel\n"),
		"decisions.md":            []byte("decisions-sentinel\n"),
		".wright-allowlist":       []byte("scope-sentinel\n"),
	}
	for name, content := range correctness {
		if err := os.WriteFile(filepath.Join(work, name), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var stdout, stderr bytes.Buffer
	if code := Timeline(root, []string{"purge", "--run", testRunID}, &stdout, &stderr); code != 0 {
		t.Fatalf("purge code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "purged 2 v1 event(s) from 2 telemetry file(s)") {
		t.Fatalf("purge summary=%q", stdout.String())
	}
	for _, path := range []string{
		filepath.Join(root, "timeline.jsonl"),
		filepath.Join(work, "events.jsonl"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), testRunID) ||
			!strings.Contains(string(data), observabilityOtherRunID) ||
			!strings.Contains(string(data), "legacy stays") {
			t.Fatalf("purge scope wrong for %s: %s", path, data)
		}
	}
	for name, content := range correctness {
		got, err := os.ReadFile(filepath.Join(work, name))
		if err != nil || !bytes.Equal(got, content) {
			t.Fatalf("purge changed correctness file %s: got=%q err=%v", name, got, err)
		}
	}

	stdout.Reset()
	stderr.Reset()
	if code := Timeline(root, []string{"purge", "--before", "2026-07-23T14:00:00Z"}, &stdout, &stderr); code != 0 {
		t.Fatalf("time purge code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, path := range []string{
		filepath.Join(root, "timeline.jsonl"),
		filepath.Join(work, "events.jsonl"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), observabilityOtherRunID) ||
			!strings.Contains(string(data), "legacy stays") {
			t.Fatalf("time purge interpreted legacy or kept old v1 row in %s: %s", path, data)
		}
	}
}

func TestTimelinePurgeSelectorsAndSymlinkSafety(t *testing.T) {
	root, _ := makeObservabilityWorkspace(t)
	var stdout, stderr bytes.Buffer
	if code := Timeline(root, []string{"purge"}, &stdout, &stderr); code != 2 {
		t.Fatalf("selector-free purge code=%d", code)
	}

	outside := filepath.Join(t.TempDir(), "outside.jsonl")
	if err := os.WriteFile(outside, []byte("outside-sentinel\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "timeline.jsonl")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Timeline(root, []string{"purge", "--before", "2026-07-24T00:00:00Z"}, &stdout, &stderr); code != 1 {
		t.Fatalf("symlink purge code=%d stderr=%q", code, stderr.String())
	}
	got, err := os.ReadFile(outside)
	if err != nil || string(got) != "outside-sentinel\n" {
		t.Fatalf("purge followed telemetry symlink: got=%q err=%v", got, err)
	}
}

func TestProgressShowsV1FactsWithoutChangingLifecycle(t *testing.T) {
	root, work := makeObservabilityWorkspace(t)
	stateText := "| Key | Value |\n| --- | --- |\n| phase | build |\n| status | running |\n"
	if err := os.WriteFile(filepath.Join(work, "state.md"), []byte(stateText), 0o644); err != nil {
		t.Fatal(err)
	}
	event := appendObservabilityEvent(t, root, testRunID, time.Now().UTC().Format(time.RFC3339),
		"readiness", BoundaryLifecycleGate, reason.GateReadinessMissing, OutcomeBlocked, state.PhaseBuild)
	event.ExecutionMode = ExecutionGeneric
	rewriteLastObservabilityEvent(t, root, event)

	before, err := os.ReadFile(filepath.Join(work, "state.md"))
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if code := Progress(root, []string{"demo"}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("progress code=%d", code)
	}
	for _, want := range []string{"rite-build", "Obs   obs mode:generic", "gate:DRV-GATE-READINESS-MISSING", "degraded:1"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("progress missing %q:\n%s", want, stdout.String())
		}
	}
	after, err := os.ReadFile(filepath.Join(work, "state.md"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("observability display changed lifecycle state: err=%v", err)
	}
}
