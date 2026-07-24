package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/lib"
	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/state"
)

func TestTimelineReportIgnoresItsOwnRootSelectionEvent(t *testing.T) {
	project := t.TempDir()
	initRootRoutingGitRepo(t, project)
	root := filepath.Join(project, ".devrites")
	work := filepath.Join(root, "work", "demo")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte("demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "state.md"), []byte("| phase | build |\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	event := lib.NewEventV1(lib.BoundaryAgentDispatch, "run-started", reason.RootSelected)
	event.TS = "2026-07-23T12:00:00Z"
	event.RunID = "drv-run-v1:0123456789abcdef0123456789abcdef"
	event.RootSource = lib.RootSourceExplicit
	event.Workspace = ".devrites/work/demo"
	event.PhaseBefore = state.PhaseBuild
	event.PhaseAfter = state.PhaseBuild
	event.Outcome = lib.OutcomeRecorded
	if err := lib.AppendEventV1(root, event); err != nil {
		t.Fatal(err)
	}

	withRootRoutingCWD(t, project)
	t.Setenv("DEVRITES_ROOT", root)
	t.Setenv("DEVRITES_WORKSPACE", "")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"timeline", "report"}, strings.NewReader(""), &stdout, &stderr); code != exitOK {
		t.Fatalf("timeline report code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), event.RunID) ||
		!strings.Contains(stdout.String(), "1 v1 events") {
		t.Fatalf("report selected its own root-selection event instead of the trace: %s", stdout.String())
	}
}
