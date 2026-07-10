package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeHookWorkspace(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".devrites")
	dir := filepath.Join(root, "work", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte("demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	state := "- Phase: build\n- Status: running\n- Next step: /rite-prove\n"
	if err := os.WriteFile(filepath.Join(dir, "state.md"), []byte(state), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_ROOT", root)
	return root
}

func TestHookEventWritesTimelineAndWorkspaceEvents(t *testing.T) {
	root := makeHookWorkspace(t)
	if code := hookEvent([]string{"subagent-stop"}, strings.NewReader(`{}`), &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("hookEvent code=%d", code)
	}
	for _, path := range []string{
		filepath.Join(root, "timeline.jsonl"),
		filepath.Join(root, "work", "demo", "events.jsonl"),
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !strings.Contains(string(raw), `"event":"subagent-stop"`) || !strings.Contains(string(raw), `"slug":"demo"`) {
			t.Fatalf("unexpected event log %s: %s", path, raw)
		}
	}
}

func TestHookAUQRecordsQuestionAndAnswer(t *testing.T) {
	root := makeHookWorkspace(t)
	payload := `{"tool_name":"AskUserQuestion","tool_input":{"questions":[{"question":"Ship now?"}]},"tool_response":{"answers":{"Ship now?":"Yes, ship"}}}`
	if code := hookAUQ(strings.NewReader(payload), &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("hookAUQ code=%d", code)
	}
	for _, path := range []string{
		filepath.Join(root, "timeline.jsonl"),
		filepath.Join(root, "work", "demo", "events.jsonl"),
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, want := range []string{`"event":"auq"`, `"question":"Ship now?"`, `"answer":"Yes, ship"`, `"slug":"demo"`} {
			if !strings.Contains(string(raw), want) {
				t.Fatalf("auq log %s missing %s: %s", path, want, raw)
			}
		}
	}
}

func TestHookAUQFailOpenOnGarbageAndNoWorkspace(t *testing.T) {
	root := makeHookWorkspace(t)
	if code := hookAUQ(strings.NewReader("not json"), &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("garbage payload must exit 0, got %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, "timeline.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("garbage payload must record nothing, stat err = %v", err)
	}
	t.Setenv("DEVRITES_ROOT", "")
	if err := os.Remove(filepath.Join(root, "ACTIVE")); err != nil {
		t.Fatal(err)
	}
	payload := `{"tool_input":{"questions":[{"question":"q"}]}}`
	if code := hookAUQ(strings.NewReader(payload), &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("no workspace must exit 0, got %d", code)
	}
}

func TestHookHandoffSnapshotAppendsResumeNote(t *testing.T) {
	root := makeHookWorkspace(t)
	stdout := &bytes.Buffer{}
	if code := hookHandoffSnapshot(strings.NewReader(`{}`), stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("hookHandoffSnapshot code=%d", code)
	}
	if !strings.Contains(stdout.String(), "compaction handoff saved") {
		t.Fatalf("handoff hook should warn on stdout, got %q", stdout.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "work", "demo", "handoff.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Handoff snapshot", "Feature: demo", "Phase: build", "Next: /rite-prove"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("handoff missing %q:\n%s", want, raw)
		}
	}
}
