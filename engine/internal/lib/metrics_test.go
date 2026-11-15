package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// An oversized line never truncates the summary: events after it still count
// and the garbage line is reported as unparsed.
func TestMetricsSummaryReadsPastOversizedLine(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	ev := func(e string) string {
		return `{"ts":"2026-01-01T00:00:00Z","phase":"build","event":"` + e + `"}` + "\n"
	}
	ledger := ev("a") + strings.Repeat("x", 1<<20) + "\n" + ev("b") + ev("c")
	if err := os.WriteFile(filepath.Join(featureDir, MetricsFile), []byte(ledger), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunMetrics(root, []string{"summary", "feat"}, stdout, stderr)
	if code != 0 || !strings.Contains(stdout.String(), "events=3") || !strings.Contains(stdout.String(), "unparsed=1") {
		t.Fatalf("code=%d out=%q err=%q", code, stdout, stderr)
	}
}

func TestMetricsSummaryContextPartitions(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	ledger := `{"phase":"build","event":"context","role":"code-reviewer","bytes":100,"context_status":"materialized"}
{"phase":"build","event":"context","role":"code-reviewer","bytes":100,"context_status":"unchanged"}
{"phase":"build","event":"context","role":"code-reviewer","bytes":50}
{"phase":"build","event":"context","role":"code-reviewer","bytes":25,"context_status":"future"}
{"phase":"build","event":"packet","role":"code-reviewer","bytes":900,"context_status":"materialized"}
`
	writeFile(t, filepath.Join(featureDir, MetricsFile), ledger)
	var out, stderr bytes.Buffer
	if code := RunMetrics(root, []string{"summary", "feat"}, &out, &stderr); code != 0 {
		t.Fatalf("summary code=%d: %s", code, &stderr)
	}
	want := "metrics summary: feat\n" +
		fmt.Sprintf("  %-10s events=5 bytes=1175 (context=4 packet=1)\n", "build") +
		"  roles:\n" +
		fmt.Sprintf("    %-24s events=5 bytes=1175\n", "code-reviewer") +
		"  total      events=5 bytes=1175\n" +
		"  context_bytes=275 context_invocations=4\n" +
		"  context_materialized_bytes=100 context_materialized_invocations=1\n" +
		"  context_unchanged_bytes=100 context_unchanged_invocations=1\n" +
		"  context_unclassified_bytes=75 context_unclassified_invocations=2\n"
	if out.String() != want {
		t.Fatalf("summary:\n%s\nwant:\n%s", &out, want)
	}
}

func TestMetricsContextPublicationStatus(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flags []string
		phase string
		role  string
	}{
		{"phase", []string{"--phase", "build"}, "build", ""},
		{"skill", []string{"--skill", "rite-build"}, "rite-build", ""},
		{"reviewer", []string{"--phase", "build", "--role", "code-reviewer", "--slice", "SLICE-001"}, "build", "code-reviewer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, workspace, _, _ := contextIntegrityFixture(t)
			writeFile(t, filepath.Join(workspace, "tasks.md"), "## SLICE-001 Publication\nGoal: inspect publication metrics.\n")
			args := append([]string{"feat", "--skills-root", newSkillsRoot(t), "--out", "metrics.bundle.md"}, tc.flags...)
			bundlePath := filepath.Join(workspace, "metrics.bundle.md")
			ledgerPath := filepath.Join(workspace, MetricsFile)
			repeatStatus := "unchanged"
			if runtime.GOOS == "windows" {
				repeatStatus = "materialized" // Private POSIX mode cannot be proven on Windows.
			}
			for i, status := range []string{"materialized", repeatStatus, "materialized"} {
				if i == 2 {
					writeFile(t, filepath.Join(workspace, "spec.md"), "# spec\nChanged input.\n")
				}
				contextIntegrityRun(t, root, args, 0)
				body, err := os.ReadFile(bundlePath)
				if err != nil {
					t.Fatal(err)
				}
				ledger, err := os.ReadFile(ledgerPath)
				if err != nil {
					t.Fatal(err)
				}
				lines := bytes.Split(bytes.TrimSpace(ledger), []byte("\n"))
				if len(lines) != i+1 {
					t.Fatalf("invocation %d recorded %d events", i+1, len(lines))
				}
				var event metricEvent
				if err := json.Unmarshal(lines[i], &event); err != nil {
					t.Fatal(err)
				}
				if event.Event != "context" || event.Phase != tc.phase || event.Role != tc.role || event.ContextStatus != status || event.Bytes != int64(len(body)) {
					t.Fatalf("invocation %d: event=%+v want status=%s bytes=%d", i+1, event, status, len(body))
				}
			}
			before, err := os.ReadFile(ledgerPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(bundlePath + ".sha256"); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(bundlePath+".sha256", 0o700); err != nil {
				t.Fatal(err)
			}
			contextIntegrityRun(t, root, args, 2)
			after, err := os.ReadFile(ledgerPath)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("failed publication changed metrics: %v", err)
			}
		})
	}
}

// A reviewer's Outcome line recorded at return feeds the per-role yield tally;
// an abandon is a metric too, so lost waves show up in the summary.
func TestDispatchOutcomeAndAbandonFeedSummary(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	run := func(want int, args ...string) {
		t.Helper()
		if code := RunDispatch(root, append([]string{"feat"}, args...), out, errOut); code != want {
			t.Fatalf("dispatch %v: code=%d want %d\n%s", args, code, want, errOut)
		}
	}
	run(0, "open", "--phase", "build", "--wave", "r1", "--role", "x", "--role", "y")
	run(0, "start", "--wave", "r1", "--role", "x", "--handle", "h1")
	run(0, "start", "--wave", "r1", "--role", "y", "--handle", "h2")
	run(0, "seal", "--wave", "r1")
	run(3, "return", "--wave", "r1", "--role", "x", "--outcome", "pass")
	run(0, "return", "--wave", "r1", "--role", "x", "--outcome", "findings")
	run(0, "return", "--wave", "r1", "--role", "y", "--outcome", "gap")
	run(0, "open", "--phase", "build", "--wave", "r2", "--role", "x")
	run(3, "abandon", "--wave", "r2", "--reason", "host session died")

	out.Reset()
	if code := RunMetrics(root, []string{"summary", "feat"}, out, errOut); code != 0 {
		t.Fatalf("summary code=%d\n%s", code, errOut)
	}
	got := out.String()
	for _, want := range []string{"outcomes: findings=1", "outcomes: gap=1", "abandon=1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary lacks %q:\n%s", want, got)
		}
	}
}

// Stored-return notes feed the per-role resolution tally: a key raised again
// on the same slice is re-raised, one a later same-slice return drops is
// resolved, and one with no later same-slice return is still open.
func TestMetricsSummaryFindingResolution(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	ret := func(role, note string) string {
		return `{"ts":"2026-01-01T00:00:00Z","phase":"vet","event":"return","role":"` + role + `","note":"` + note + `"}` + "\n"
	}
	a, b := "finding=code-reviewer|SLICE-001|a.go|AC-001|important", "finding=code-reviewer|SLICE-001|b.go|AC-002|critical"
	ledger := ret("code-reviewer", "outcome=findings counts=critical=1,important=1 slice=SLICE-001 late=1 "+a+" "+b) +
		ret("code-reviewer", "outcome=findings counts=important=1 slice=SLICE-001 late=0 "+a) +
		ret("code-reviewer", "outcome=findings counts=nit=1 slice=SLICE-002 late=0 finding=code-reviewer|SLICE-002|c.go|-|nit") +
		ret("test-analyst", "outcome=no-findings")
	if err := os.WriteFile(filepath.Join(featureDir, MetricsFile), []byte(ledger), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunMetrics(root, []string{"summary", "feat"}, stdout, stderr); code != 0 {
		t.Fatalf("code=%d err=%q", code, stderr)
	}
	got := stdout.String()
	for _, want := range []string{"outcomes: findings=3 resolution: resolved=1 re-raised=1 open=2 late=1", "outcomes: no-findings=1\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary lacks %q:\n%s", want, got)
		}
	}
}

// Each resume is timed to the next dispatch in ledger order; a resume followed
// by another resume reads "none", a trailing one "open". A ledger without
// resume events keeps the old summary.
func TestMetricsSummaryResumeToDispatch(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	ev := func(ts, e string) string {
		return `{"ts":"2026-01-01T` + ts + `Z","phase":"build","event":"` + e + `","role":"root"}` + "\n"
	}
	path := filepath.Join(featureDir, MetricsFile)
	if err := os.WriteFile(path, []byte(ev("00:00:00", "dispatch")+ev("01:00:00", "resume")+ev("01:12:00", "dispatch")+ev("01:13:00", "dispatch")+ev("02:00:00", "resume")+ev("03:00:00", "resume")), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunMetrics(root, []string{"summary", "feat"}, stdout, stderr); code != 0 {
		t.Fatalf("code=%d err=%q", code, stderr)
	}
	if want := "  resumes    count=3 minutes_to_dispatch=12,none,open\n"; !strings.Contains(stdout.String(), want) {
		t.Fatalf("summary lacks %q:\n%s", want, stdout)
	}

	if err := os.WriteFile(path, []byte(ev("00:00:00", "dispatch")), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if code := RunMetrics(root, []string{"summary", "feat"}, stdout, stderr); code != 0 || strings.Contains(stdout.String(), "resumes") {
		t.Fatalf("no-resume summary changed: code=%d\n%s", code, stdout)
	}
}
