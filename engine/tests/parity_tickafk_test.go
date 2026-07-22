package main_test

// Golden snapshot for tick-afk: a MUTATING, layout-agnostic command that takes a
// state.md path as args[0]. Each fixture runs the Go binary against a fresh
// state.md and snapshots two artifacts: the command's stdout+exit (under the
// subtest name) and the resulting file bytes (under "<subtest>/state.md").
//
// The no-op / not-a-number messages embed the state.md PATH, which lives under a
// per-run temp dir, so stdout is normalized (the path -> "<STATE>") before it is
// snapshotted, keeping the golden deterministic across runs.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParityTickAfk(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{
			name:    "budget3",
			content: "# state\n\nnotes\n- AFK slices remaining: 3\nfooter\n",
		},
		{
			name:    "budget1",
			content: "- AFK slices remaining: 1\n",
		},
		{
			// Token before the first blank wins: "5 | none" -> "5" -> "4", and the
			// whole line is rewritten so the "| none" suffix is dropped.
			name:    "bulletPipe",
			content: "- AFK slices remaining: 5 | none\n",
		},
		{
			name:    "none",
			content: "- AFK slices remaining: none\n",
		},
		{
			name:    "missingField",
			content: "# state\n\nno budget here\n",
		},
		{
			name:    "nonNumeric",
			content: "- AFK slices remaining: x\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw := t.TempDir()
			state := filepath.Join(gw, "state.md")
			if err := os.WriteFile(state, []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}

			out, code := runArgv(t, gw, libEnv, "", binPath, "tick-afk", state)
			// The no-op / not-a-number message embeds the state.md path (a per-run
			// temp dir), so normalize it to a fixed token before snapshotting.
			norm := strings.ReplaceAll(out, state, "<STATE>")
			assertGolden(t, norm, code)

			// Snapshot the rewritten (or untouched) file bytes under a per-file key.
			got, err := os.ReadFile(state)
			if err != nil {
				t.Fatal(err)
			}
			assertGoldenKey(t, t.Name()+"/state.md", string(got))
		})
	}

	// Missing file: no state.md to write; the command must exit non-zero with empty
	// stdout. Only stdout+exit is snapshotted: there is no file to read.
	t.Run("missingFile", func(t *testing.T) {
		gw := t.TempDir()
		state := filepath.Join(gw, "state.md")
		out, code := runArgv(t, gw, libEnv, "", binPath, "tick-afk", state)
		assertGolden(t, strings.ReplaceAll(out, state, "<STATE>"), code)
	})
}
