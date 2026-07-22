package main_test

// Guard test for ADR-0006 (clock seam). The next-qid derivation must read the
// clock through the DEVRITES_NOW seam so date-derived output is deterministic.
// Without the seam this command tracked the real wall clock and its golden
// snapshot rotted at every day boundary. This test pins the class of bug shut.

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestADR0006NextQIDHonorsClockSeam(t *testing.T) {
	dir := t.TempDir()
	qpath := filepath.Join(dir, "questions.md")
	writeFile(t, dir, "questions.md", "## q-2024-01-01-001\nstatus: open\n")

	cases := []struct {
		now  string
		want string
	}{
		{"2026-03-04", "q-2026-03-04-001"},
		{"2030-12-31T23:59:59Z", "q-2030-12-31-001"},
	}
	for _, c := range cases {
		out, _, code := runDevritesIO(t, dir, "", []string{"DEVRITES_NOW=" + c.now},
			"resolve", "next-qid", qpath)
		if code != 0 {
			t.Fatalf("DEVRITES_NOW=%s: exit = %d, want 0\n%s", c.now, code, out)
		}
		if got := strings.TrimSpace(out); got != c.want {
			t.Errorf("DEVRITES_NOW=%s: qid = %q, want %q: clock seam not honored", c.now, got, c.want)
		}
	}
}
