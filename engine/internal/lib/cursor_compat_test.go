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
}
