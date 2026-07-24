package lib

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/reason"
)

const testRunID = "drv-run-v1:0123456789abcdef0123456789abcdef"

func validTestEvent() EventV1 {
	ev := NewEventV1(BoundaryLifecycleGate, "readiness", reason.GateReadinessPassed)
	ev.TS = "2026-07-23T12:34:56Z"
	ev.RunID = testRunID
	ev.RootSource = RootSourceExplicit
	ev.Workspace = ".devrites/work/demo"
	ev.PhaseBefore = "build"
	ev.PhaseAfter = "build"
	ev.EvidencePaths = []string{".devrites/work/demo/plan.md"}
	ev.Outcome = OutcomePassed
	return ev
}

func TestValidateEventV1RejectsPrivacyAndEnumViolations(t *testing.T) {
	base := validTestEvent()
	cases := []struct {
		name   string
		mutate func(*EventV1)
	}{
		{"absolute workspace", func(ev *EventV1) { ev.Workspace = "/absolute/project/.devrites/work/demo" }},
		{"absolute evidence", func(ev *EventV1) { ev.EvidencePaths = []string{"/absolute/evidence.txt"} }},
		{"parent evidence", func(ev *EventV1) { ev.EvidencePaths = []string{"../secret.txt"} }},
		{"free text event", func(ev *EventV1) { ev.Event = "the user said yes" }},
		{"raw run id", func(ev *EventV1) { ev.RunID = "alice@example.com" }},
		{"unknown reason", func(ev *EventV1) { ev.ReasonID = "DRV-UNKNOWN" }},
		{"unknown rule", func(ev *EventV1) { ev.RuleIDs = []reason.ID{"DRV-UNKNOWN"} }},
		{"unknown mode", func(ev *EventV1) { ev.ExecutionMode = "remote" }},
		{"unknown guard", func(ev *EventV1) { ev.GuardStrength = "maybe" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ev := base
			ev.RuleIDs = append([]reason.ID(nil), base.RuleIDs...)
			ev.EvidencePaths = append([]string(nil), base.EvidencePaths...)
			tc.mutate(&ev)
			if err := ValidateEventV1(ev); err == nil {
				t.Fatalf("ValidateEventV1 accepted %+v", ev)
			}
		})
	}
}

func TestValidateEventV1AcceptsExistingMixedCaseWorkspaceSlug(t *testing.T) {
	ev := validTestEvent()
	ev.Workspace = ".devrites/features/Feature_1"
	if err := ValidateEventV1(ev); err != nil {
		t.Fatalf("existing valid slug was rejected: %v", err)
	}
}

func TestCurrentRunIDNeverPersistsArbitraryEnvironmentText(t *testing.T) {
	t.Setenv("DEVRITES_RUN_ID", "operator-private-value")
	if got := CurrentRunID(); got == "operator-private-value" || !runIDToken.MatchString(got) {
		t.Fatalf("CurrentRunID() = %q", got)
	}
	t.Setenv("DEVRITES_RUN_ID", testRunID)
	if got := CurrentRunID(); got != testRunID {
		t.Fatalf("canonical injected run id = %q", got)
	}
}

func TestAppendEventV1WritesSameRecordToExistingOwners(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	workspace := filepath.Join(root, "work", "demo")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	ev := validTestEvent()
	if err := AppendEventV1(root, ev); err != nil {
		t.Fatal(err)
	}
	var first string
	for _, file := range []string{
		filepath.Join(root, "timeline.jsonl"),
		filepath.Join(workspace, "events.jsonl"),
	} {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if first == "" {
			first = string(raw)
		} else if string(raw) != first {
			t.Fatalf("root/workspace records differ:\n%s\n%s", first, raw)
		}
		var got EventV1
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("decode %s: %v", file, err)
		}
		if got.Schema != EventSchemaV1 || got.ReasonID != reason.GateReadinessPassed {
			t.Fatalf("unexpected record: %+v", got)
		}
	}
}

func TestAppendEventV1RejectsSymlinkLogWithoutWritingTarget(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(project, "outside.jsonl")
	if err := os.WriteFile(target, []byte("sentinel\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "timeline.jsonl")); err != nil {
		t.Fatal(err)
	}
	ev := validTestEvent()
	ev.Workspace = ""
	ev.PhaseBefore = ""
	ev.PhaseAfter = ""
	ev.EvidencePaths = []string{}
	if err := AppendEventV1(root, ev); err == nil {
		t.Fatal("symlink log should be rejected")
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(raw)) != "sentinel" {
		t.Fatalf("outside target changed: %q", raw)
	}
}

func TestAppendEventV1PreflightsWorkspaceBeforeRootAppend(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	ev := validTestEvent()
	if err := AppendEventV1(root, ev); err == nil {
		t.Fatal("missing workspace should fail")
	}
	if _, err := os.Stat(filepath.Join(root, "timeline.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("root event was appended before workspace preflight: %v", err)
	}
}
