package lib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/state"
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

func TestOfficialBulletCursorAndKeyAliasesRemainReadable(t *testing.T) {
	lines := strings.Split("- Phase: build\n- Next step: continue\n- qid: Q-001\n", "\n")
	for key, want := range map[string]string{
		state.CursorPhase:      "build",
		state.CursorNextAction: "continue",
		state.CursorQuestionID: "Q-001",
	} {
		if got, ok := state.CursorField(lines, key); !ok || got != want {
			t.Fatalf("CursorField(%q)=(%q,%v), want %q", key, got, ok, want)
		}
	}
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
