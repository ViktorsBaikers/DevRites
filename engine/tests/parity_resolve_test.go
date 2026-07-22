package main_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// Golden snapshot for resolve: it MUTATES questions.md + state.md. Each case
// snapshots the command's stdout + exit code and, for the mutating cases, the
// resulting file bytes. The two nondeterministic timestamp fields (answered_at
// and the Log entry) are masked out before the file content is snapshotted so the
// golden is deterministic across runs.
var (
	reAnsweredAt = regexp.MustCompile(`answered_at: \S+`)
	reLogTS      = regexp.MustCompile(`(?m)^- \S+ build: resolved`)
)

func maskResolveTS(s string) string {
	s = reAnsweredAt.ReplaceAllString(s, "answered_at: TS")
	s = reLogTS.ReplaceAllString(s, "- TS build: resolved")
	return s
}

const resolveQuestions = `## q-1 blocking
status: open
answered_at:
answer:

## q-2 advisory
status: open
answered_at:
answer:
`

const resolveState = `- Status: awaiting_human
- Next step: wait for q-1

## Awaiting human
- qid: q-1

## Log
- 2024-01-01 spec: created
`

func TestParityResolve(t *testing.T) {
	// setup writes the fixtures into a Go tree (.devrites/features/feat) and returns
	// the workspace root.
	setup := func(t *testing.T, questions, state string) (gwork string) {
		t.Helper()
		gwork = t.TempDir()
		writeFile(t, gwork, filepath.Join(".devrites", "features", "feat", "questions.md"), questions)
		writeFile(t, gwork, filepath.Join(".devrites", "features", "feat", "state.md"), state)
		writeFile(t, gwork, filepath.Join(".devrites", "ACTIVE"), "feat\n")
		return gwork
	}
	// run executes resolve in gwork and snapshots its stdout + exit code under the
	// current subtest's golden.
	run := func(t *testing.T, gwork string, args ...string) {
		t.Helper()
		out, code := runArgv(t, gwork, libRootEnv(gwork), "", binPath, append([]string{"resolve"}, args...)...)
		assertGolden(t, out, code)
	}
	// assertFileGolden snapshots the masked content of a mutated file under a
	// per-file golden key (t.Name()+"/"+name).
	assertFileGolden := func(t *testing.T, gwork, name string) {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(gwork, ".devrites", "features", "feat", name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		assertGoldenKey(t, t.Name()+"/"+name, maskResolveTS(string(b)))
	}

	t.Run("answer", func(t *testing.T) {
		gwork := setup(t, resolveQuestions, resolveState)
		run(t, gwork, "q-1", "use JWT with 15m expiry")
		assertFileGolden(t, gwork, "questions.md")
		assertFileGolden(t, gwork, "state.md")
	})

	t.Run("drop", func(t *testing.T) {
		gwork := setup(t, resolveQuestions, resolveState)
		run(t, gwork, "--drop", "q-2", "out of scope")
		assertFileGolden(t, gwork, "questions.md")
		assertFileGolden(t, gwork, "state.md")
	})

	t.Run("not-found", func(t *testing.T) {
		gwork := setup(t, resolveQuestions, resolveState)
		run(t, gwork, "q-99", "nope") // exit 3
	})

	t.Run("not-open", func(t *testing.T) {
		answered := `## q-1 blocking
status: answered
answered_at: 2024-01-01T00:00:00Z
answer: prior
`
		gwork := setup(t, answered, resolveState)
		run(t, gwork, "q-1", "again") // exit 4
	})

	t.Run("batch", func(t *testing.T) {
		gwork := setup(t, resolveQuestions, resolveState)
		writeFile(t, gwork, "batch.txt", "q-1: answer one\n--drop q-2: reason two\n")
		run(t, gwork, "--batch", "batch.txt")
		assertFileGolden(t, gwork, "questions.md")
	})

	t.Run("batch-unterminated", func(t *testing.T) {
		// No trailing newline: a final line without a terminating newline is not
		// applied (q-2 stays open), so the golden captures the truncated batch.
		gwork := setup(t, resolveQuestions, resolveState)
		writeFile(t, gwork, "batch.txt", "q-1: answer one\n--drop q-2: reason two") // no trailing \n
		run(t, gwork, "--batch", "batch.txt")
		assertFileGolden(t, gwork, "questions.md")
	})

	t.Run("missing-answer", func(t *testing.T) {
		gwork := setup(t, resolveQuestions, resolveState)
		run(t, gwork, "q-1") // no answer text -> exit 5
	})

	t.Run("no-workspace", func(t *testing.T) {
		// Bare tree: no ACTIVE, no questions.md -> exit 2.
		gwork := t.TempDir()
		run(t, gwork, "q-1", "x")
	})

	t.Run("next-qid", func(t *testing.T) {
		// Read-only: computes the next qid for an explicit questions.md path.
		// DEVRITES_NOW pins the clock so the golden is stable across day
		// boundaries: without it the qid tracks the real date and the snapshot
		// fails every day but the one it was recorded on (ADR-0006, clock seam).
		dir := t.TempDir()
		writeFile(t, dir, "questions.md", "## q-2024-01-01-001\nstatus: open\n")
		qpath := filepath.Join(dir, "questions.md")
		env := append(append([]string{}, libEnv...), "DEVRITES_NOW=2026-07-07")
		out, code := runArgv(t, dir, env, "", binPath, "resolve", "next-qid", qpath)
		assertGolden(t, out, code)
	})
}
