package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const canonicalCursor = `# State

## Cursor
| Key | Value |
| --- | --- |
| phase | temper |
| status | running |
| next_action | /rite-define |
| plan_approved | 2026-07-20 |
| afk_slices_remaining | 2 |
`

func TestCanonicalCursorReadAndWriteConsumers(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "work", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(dir, "state.md")
	if err := os.WriteFile(statePath, []byte(canonicalCursor), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("build readiness", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := BuildReadiness(root, []string{"demo"}, &stdout, &stderr); code != 0 {
			t.Fatalf("code=%d, stderr=%s", code, stderr.String())
		}
	})

	t.Run("progress", func(t *testing.T) {
		var stdout bytes.Buffer
		if code := Progress(root, []string{"demo"}, &stdout, &bytes.Buffer{}); code != 0 {
			t.Fatalf("code=%d", code)
		}
		if !strings.Contains(stdout.String(), "rite-temper") || !strings.Contains(stdout.String(), "temper ◉") {
			t.Fatalf("progress ignored table phase:\n%s", stdout.String())
		}
	})

	t.Run("tick AFK", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := TickAfk([]string{statePath}, &stdout, &stderr); code != 0 {
			t.Fatalf("code=%d, stderr=%s", code, stderr.String())
		}
		raw, err := os.ReadFile(statePath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "| afk_slices_remaining | 1 |") {
			t.Fatalf("tick AFK did not preserve/update table:\n%s", raw)
		}
	})
}

func TestClearAwaitingSupportsCanonicalCursor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.md")
	state := canonicalCursor + `
## Awaiting human
| Key | Value |
| --- | --- |
| question_id | Q-001 |
| gate | blocking |
`
	state = strings.Replace(state, "| status | running |", "| status | awaiting_human |", 1)
	if err := os.WriteFile(path, []byte(state), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := clearAwaiting(path, "Q-001"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(text, "## Awaiting human") || !strings.Contains(text, "| status | running |") {
		t.Fatalf("canonical awaiting state was not cleared:\n%s", text)
	}
	if !strings.Contains(text, "$rite-temper") || strings.Contains(text, "$rite-build") {
		t.Fatalf("canonical awaiting state resumed the wrong phase:\n%s", text)
	}
}

func TestProgressReadsCanonicalSlicesAndShowsPlanPhase(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "work", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	stateText := strings.Replace(canonicalCursor, "| phase | temper |", "| phase | plan |", 1)
	stateText = strings.Replace(stateText, "| afk_slices_remaining | 2 |", "| active_slice | SLICE-002 |", 1)
	if err := os.WriteFile(filepath.Join(dir, "state.md"), []byte(stateText), 0o644); err != nil {
		t.Fatal(err)
	}
	tasks := "## SLICE-001 First\nStatus: built\n\n## SLICE-002 Second\nStatus: pending\n"
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(tasks), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if code := Progress(root, []string{"demo"}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("code=%d", code)
	}
	text := stdout.String()
	if !strings.Contains(text, "Slice 1/2") || !strings.Contains(text, "plan ◉") || strings.Contains(text, "build ◉") {
		t.Fatalf("progress missed canonical slice/plan state:\n%s", text)
	}
}
